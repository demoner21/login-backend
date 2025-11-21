package auth

import (
	"database/sql"
	"loginbackend/features/shared/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByEmail(email string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url, refresh_token
		FROM users WHERE email = $1
	`, email)

	var user models.User
	var lastLoginAt, profileImageUrl, refreshToken sql.NullString
	var lastPasswordUpdate sql.NullTime

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.RoleID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastPasswordUpdate,
		&user.IsEmailVerified,
		&lastLoginAt,
		&profileImageUrl,
		&refreshToken,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if lastPasswordUpdate.Valid {
		user.LastPasswordUpdate = lastPasswordUpdate.Time
	}
	if profileImageUrl.Valid {
		user.ProfileImageUrl = &profileImageUrl.String
	}
	if refreshToken.Valid {
		user.RefreshToken = &refreshToken.String
	}

	return &user, nil
}

func (r *Repository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1`,
		userID,
	)
	return err
}

func (r *Repository) SaveRefreshToken(userID string, refreshToken string) error {
	_, err := r.db.Exec(
		`UPDATE users SET refresh_token = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		refreshToken, userID,
	)
	return err
}

func (r *Repository) FindUserByRefreshToken(refreshToken string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, name, role_id, is_active, created_at
		FROM users WHERE refresh_token = $1 AND is_active = true
	`, refreshToken)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.RoleID,
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) ClearRefreshToken(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET refresh_token = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		userID,
	)
	return err
}
