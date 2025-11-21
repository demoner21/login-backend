package models

import "time"

type User struct {
	ID                 int        `json:"id"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	PasswordHash       string     `json:"-"`
	LastPasswordUpdate time.Time  `json:"last_password_update"`
	RefreshToken       *string    `json:"-"`
	IsEmailVerified    bool       `json:"is_email_verified"`
	TotpSecret         *string    `json:"-"`
	ProfileImageUrl    *string    `json:"profile_image_url,omitempty"`
	RoleID             int        `json:"role_id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	IsActive           bool       `json:"is_active"`
}
