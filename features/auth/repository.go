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
			   last_login_at, profile_image_url
		FROM users WHERE email = ?
	`, email)

	var user models.User
	var lastLoginAt, profileImageUrl sql.NullString
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

	return &user, nil
}

func (r *Repository) UpdateLastLogin(userID int) error {
	_, err := r.db.Exec(
		`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`,
		userID,
	)
	return err
}

func (r *Repository) SaveRefreshToken(userID int, refreshToken string) error {
	_, err := r.db.Exec(
		`UPDATE users SET refresh_token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		refreshToken, userID,
	)
	return err
}

func (r *Repository) FindUserByRefreshToken(refreshToken string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, name, role_id, is_active, created_at
		FROM users WHERE refresh_token = ? AND is_active = true
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

func (r *Repository) ClearRefreshToken(userID int) error {
	_, err := r.db.Exec(
		`UPDATE users SET refresh_token = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		userID,
	)
	return err
}
