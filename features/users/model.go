package users

import "time"

type Role struct {
	ID          int       `json:"id"`
	RoleName    string    `json:"role_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	PasswordHash       string     `json:"-"`
	LastPasswordUpdate time.Time  `json:"last_password_update"`
	RefreshToken       *string    `json:"refresh_token,omitempty"`
	IsEmailVerified    bool       `json:"is_email_verified"`
	TotpSecret         *string    `json:"totp_secret,omitempty"`
	ProfileImageUrl    *string    `json:"profile_image_url,omitempty"`
	RoleID             int        `json:"role_id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	IsActive           bool       `json:"is_active"`
}
