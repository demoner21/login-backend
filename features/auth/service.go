package auth

import (
	"errors"
	"fmt"
	"loginbackend/pkg/utils"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo          *Repository
	jwtSecret     string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type Config struct {
	JWTSecret     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func NewService(repo *Repository, cfg Config) *Service {
	return &Service{
		repo:          repo,
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
		UserID: user.ID, // Agora é int, compatível com o model
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
		ID:        user.ID, // Agora é int, compatível com o model
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
		UserID: user.ID, // Agora é int, compatível com o model
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
		ID:        user.ID, // Agora é int, compatível com o model
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

func (s *Service) Logout(userID string) error {
	return s.repo.ClearRefreshToken(userID)
}

// ValidateToken valida um token JWT e retorna os claims
func (s *Service) ValidateToken(tokenString string) (*utils.TokenClaims, error) {
	return utils.ValidateJWT(tokenString, s.jwtSecret)
}
