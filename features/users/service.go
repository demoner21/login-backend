package users

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo}
}

func (s *Service) Register(name, email, password string) error {
	// Validações básicas
	if name == "" || email == "" || password == "" {
		return errors.New("nome, email e senha são obrigatórios")
	}

	if len(password) < 6 {
		return errors.New("a senha deve ter pelo menos 6 caracteres")
	}

	if s.repo.EmailExists(email) {
		return errors.New("email já cadastrado")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	user := User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       2, // USER
		IsActive:     true,
	}

	return s.repo.Create(user)
}

func (s *Service) Login(email, password string) (*User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	if !user.IsActive {
		return nil, errors.New("usuário inativo")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("senha incorreta")
	}

	s.repo.UpdateLastLogin(user.ID)

	user.PasswordHash = ""
	return user, nil
}

func (s *Service) ListUsers() ([]User, error) {
	return s.repo.List()
}
