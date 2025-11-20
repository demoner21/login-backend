package users

import (
	"errors"

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
	if s.repo.EmailExists(email) {
		return errors.New("email já cadastrado")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       2, // USER
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

	return user, nil
}

func (s *Service) ListUsers() ([]User, error) {
	return s.repo.List()
}
