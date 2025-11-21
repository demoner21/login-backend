package auth

import "time"

// DTOs para requests/responses
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UserResponse struct {
	ID        string    `json:"id"` // Agora é string (Snowflake ID)
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	RoleID    int       `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}
