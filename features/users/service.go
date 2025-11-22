package users

import (
	"errors"
	"fmt"
	"loginbackend/features/shared/models"
	"loginbackend/pkg/utils"

	"github.com/go-playground/validator/v10"
)

type Service struct {
	repo     *Repository
	validate *validator.Validate
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:     repo,
		validate: validator.New(),
	}
}

// Create - Cria usuário
func (s *Service) Create(req CreateUserRequest) (*models.User, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("erro de validação: %w", err)
	}

	exists, err := s.repo.EmailExists(req.Email)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar email: %w", err)
	}
	if exists {
		return nil, errors.New("email já cadastrado")
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	roleID := req.RoleID
	if roleID == 0 {
		roleID = 2 // USER
	}

	snowflakeID := utils.GenerateSnowflakeID()

	user := models.User{
		ID:           snowflakeID,
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		RoleID:       roleID,
		IsActive:     true,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("erro ao criar usuário: %w", err)
	}

	createdUser, err := s.repo.FindByEmail(user.Email)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário criado: %w", err)
	}

	createdUser.PasswordHash = ""
	return createdUser, nil
}

// GetByID - Busca usuário por ID
func (s *Service) GetByID(userID string) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if user == nil {
		return nil, errors.New("usuário não encontrado")
	}

	user.PasswordHash = ""
	return user, nil
}

// List - Lista todos os usuários
func (s *Service) List() ([]models.User, error) {
	users, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("erro ao listar usuários: %w", err)
	}

	for i := range users {
		users[i].PasswordHash = ""
	}

	return users, nil
}

// Update - Atualiza usuário
func (s *Service) Update(userID string, req UpdateUserRequest) (*models.User, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("erro de validação: %w", err)
	}

	existing, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if existing == nil {
		return nil, errors.New("usuário não encontrado")
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Email != nil {
		existing.Email = *req.Email
	}
	if req.Phone != nil {
		existing.Phone = req.Phone
	}
	if req.JobTitle != nil {
		existing.JobTitle = req.JobTitle
	}
	if req.Location != nil {
		existing.Location = req.Location
	}
	if req.Country != nil {
		existing.Country = req.Country
	}
	if req.City != nil {
		existing.City = req.City
	}
	if req.State != nil {
		existing.State = req.State
	}
	if req.PostalCode != nil {
		existing.PostalCode = req.PostalCode
	}
	if req.TaxID != nil {
		existing.TaxID = req.TaxID
	}
	if req.RoleID != nil {
		existing.RoleID = *req.RoleID
	}

	if err := s.repo.Update(*existing); err != nil {
		return nil, fmt.Errorf("erro ao atualizar usuário: %w", err)
	}

	existing.PasswordHash = ""
	return existing, nil
}

// Delete - Desativa usuário
func (s *Service) Delete(userID string) error {
	return s.repo.Delete(userID)
}

func (s *Service) ChangePassword(userID string, req ChangePasswordRequest) error {

	if err := s.validate.Struct(req); err != nil {
		return fmt.Errorf("dados inválidos: %w", err)
	}

	user, err := s.repo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if user == nil {
		return errors.New("usuário não encontrado")
	}

	if !utils.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return errors.New("senha atual incorreta")
	}

	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("erro ao gerar hash: %w", err)
	}

	return s.repo.UpdatePassword(userID, newHash)
}

func (s *Service) UpdateAvatar(userID, avatarURL string) (*models.User, error) {
	// Verificar se user existe
	user, err := s.repo.FindByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("usuário não encontrado")
	}

	// Atualizar
	if err := s.repo.UpdateAvatar(userID, avatarURL); err != nil {
		return nil, err
	}

	// Retornar usuário atualizado
	user.AvatarURL = &avatarURL
	return user, nil
}
