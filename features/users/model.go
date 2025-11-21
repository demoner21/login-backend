package users

import "time"

// DTOs específicos para gestão de usuários
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleID   int    `json:"role_id,omitempty"`
}

type UpdateUserRequest struct {
	Name   *string `json:"name,omitempty"`
	Email  *string `json:"email,omitempty"`
	RoleID *int    `json:"role_id,omitempty"`
}

type UserResponse struct {
	ID          string     `json:"id"` // Agora é string (Snowflake ID)
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	RoleID      int        `json:"role_id"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}
