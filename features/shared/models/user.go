package models

import "time"

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`

	// Novos Campos de Perfil
	Phone     *string `json:"phone,omitempty"`
	JobTitle  *string `json:"job_title,omitempty"`
	Location  *string `json:"location,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"` // Mudou de ProfileImageUrl para AvatarURL

	// Campos de Endereço
	Country    *string `json:"country,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	TaxID      *string `json:"tax_id,omitempty"`

	// Campos de Sistema
	RoleID             int        `json:"role_id"`
	IsActive           bool       `json:"is_active"`
	IsEmailVerified    bool       `json:"is_email_verified"`
	LastPasswordUpdate time.Time  `json:"last_password_update"`
	RefreshToken       *string    `json:"-"`
	TotpSecret         *string    `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
}
