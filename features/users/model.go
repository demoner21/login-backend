package users

import "time"

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	RoleID   int    `json:"role_id,omitempty"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,max=20"`
	JobTitle *string `json:"job_title,omitempty" validate:"omitempty,max=100"`
	Location *string `json:"location,omitempty" validate:"omitempty,max=150"`

	// Endereço
	Country    *string `json:"country,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	TaxID      *string `json:"tax_id,omitempty"`

	// Admin fields
	RoleID *int `json:"role_id,omitempty"`
}

// Request específica para troca de senha
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

type UserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	RoleID      int        `json:"role_id"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// UserSearchResult é a versão pública/segura do usuário, usada em
// buscas para compartilhamento. Nunca expõe email completo de terceiros
// para evitar enumeração — só confirma o que o buscador já sabe.
type UserSearchResult struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
