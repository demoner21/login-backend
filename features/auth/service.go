package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"loginbackend/pkg/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo          *Repository
	redis         *redis.Client
	jwtSecret     string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type Config struct {
	JWTSecret     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func NewService(repo *Repository, redisClient *redis.Client, cfg Config) *Service {
	return &Service{
		repo:          repo,
		redis:         redisClient,
		jwtSecret:     cfg.JWTSecret,
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
	}
}

func (s *Service) Login(email, password string) (*LoginResponse, error) {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, errors.New("erro ao buscar usuário")
	}

	if user == nil {
		return nil, errors.New("credenciais inválidas")
	}

	if !user.IsActive {
		return nil, errors.New("usuário inativo")
	}

	// Verificar senha
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("senha incorreta")
	}

	// Atualizar último login
	if err := s.repo.UpdateLastLogin(user.ID); err != nil {
		return nil, fmt.Errorf("erro ao atualizar último login: %w", err)
	}

	// Gerar tokens
	accessToken, err := utils.GenerateJWT(utils.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		RoleID: user.RoleID,
	}, s.jwtSecret, s.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar refresh token: %w", err)
	}

	if err := s.repo.SaveRefreshToken(user.ID, refreshToken); err != nil {
		return nil, fmt.Errorf("erro ao salvar refresh token: %w", err)
	}

	userResponse := &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		RoleID:    user.RoleID,
		CreatedAt: user.CreatedAt,
	}

	return &LoginResponse{
		User:         userResponse,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

func (s *Service) RefreshToken(refreshToken string) (*LoginResponse, error) {
	user, err := s.repo.FindUserByRefreshToken(refreshToken)
	if err != nil || user == nil {
		return nil, errors.New("refresh token inválido")
	}

	accessToken, err := utils.GenerateJWT(utils.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		RoleID: user.RoleID,
	}, s.jwtSecret, s.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar access token: %w", err)
	}

	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar refresh token: %w", err)
	}

	if err := s.repo.SaveRefreshToken(user.ID, newRefreshToken); err != nil {
		return nil, fmt.Errorf("erro ao salvar refresh token: %w", err)
	}

	userResponse := &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		RoleID:    user.RoleID,
		CreatedAt: user.CreatedAt,
	}

	return &LoginResponse{
		User:         userResponse,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

func (s *Service) Logout(tokenString, userID string) error {
	// 1. Limpar Refresh Token no Banco (Postgres)
	if err := s.repo.ClearRefreshToken(userID); err != nil {
		return err
	}

	// 2. Blacklist do Access Token no Redis
	return s.addToBlacklist(tokenString)
}

// Helper para adicionar à Blacklist
func (s *Service) addToBlacklist(tokenString string) error {
	// Parse do token (sem validar assinatura, apenas para pegar o 'exp')
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("erro ao ler token para blacklist: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("claims inválidos")
	}

	// Calcular tempo restante de vida do token
	var exp int64
	switch v := claims["exp"].(type) {
	case float64:
		exp = int64(v)
	case json.Number:
		exp, _ = v.Int64()
	}

	if exp == 0 {
		return nil // Sem expiração, não faz nada (ou força um default)
	}

	expiresAt := time.Unix(exp, 0)
	timeRemaining := time.Until(expiresAt)

	// Se já expirou, não precisa salvar no Redis
	if timeRemaining <= 0 {
		return nil
	}

	// Salvar no Redis: Chave = "blacklist:{token}", Valor = "revoked", TTL = timeRemaining
	ctx := context.Background()
	key := fmt.Sprintf("blacklist:%s", tokenString)

	if err := s.redis.Set(ctx, key, "revoked", timeRemaining).Err(); err != nil {
		return fmt.Errorf("erro ao salvar na blacklist: %w", err)
	}

	return nil
}

// ValidateToken valida um token JWT e retorna os claims
func (s *Service) ValidateToken(tokenString string) (*utils.TokenClaims, error) {
	return utils.ValidateJWT(tokenString, s.jwtSecret)
}
