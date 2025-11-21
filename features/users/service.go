package users

import (
	"errors"
	"fmt"
	"loginbackend/features/shared/models"
	"loginbackend/pkg/utils"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create - Cria usuário (sem lógica de login/sessão)
func (s *Service) Create(req CreateUserRequest) (*models.User, error) {
	// Validações básicas
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("nome, email e senha são obrigatórios")
	}

	if len(req.Password) < 6 {
		return nil, errors.New("a senha deve ter pelo menos 6 caracteres")
	}

	// Verificar se email já existe
	exists, err := s.repo.EmailExists(req.Email)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar email: %w", err)
	}
	if exists {
		return nil, errors.New("email já cadastrado")
	}

	// Hash da senha
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	// Definir role padrão se não especificada
	roleID := req.RoleID
	if roleID == 0 {
		roleID = 2 // USER
	}

	// Gerar Snowflake ID
	snowflakeID := utils.GenerateSnowflakeID()

	// Criar usuário
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

	// Buscar usuário criado para retornar com ID
	createdUser, err := s.repo.FindByEmail(user.Email)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário criado: %w", err)
	}

	// Retornar usuário sem password hash
	createdUser.PasswordHash = ""
	return createdUser, nil
}

// GetByID - Busca usuário por ID
// ✅ CORREÇÃO: Recebe string diretamente
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

	// Remover password hash de todos os usuários
	for i := range users {
		users[i].PasswordHash = ""
	}

	return users, nil
}

// Update - Atualiza usuário
// ✅ CORREÇÃO: Recebe string diretamente
func (s *Service) Update(userID string, req UpdateUserRequest) (*models.User, error) {
	// Buscar usuário existente
	existing, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if existing == nil {
		return nil, errors.New("usuário não encontrado")
	}

	// Aplicar updates
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Email != nil {
		existing.Email = *req.Email
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
// ✅ CORREÇÃO: Recebe string diretamente
func (s *Service) Delete(userID string) error {
	return s.repo.Delete(userID)
}
