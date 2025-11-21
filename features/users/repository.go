package users

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

func (r *Repository) Create(user models.User) error {
	query := `
		INSERT INTO users 
		(email, name, password_hash, role_id, is_active) 
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.RoleID,
		user.IsActive,
	)
	if err != nil {
		return err
	}

	// Obter o ID gerado
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)

	return nil
}

func (r *Repository) FindByID(id int) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url
		FROM users WHERE id = ?
	`, id)

	return r.scanUser(row)
}

func (r *Repository) FindByEmail(email string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url
		FROM users WHERE email = ?
	`, email)

	return r.scanUser(row)
}

func (r *Repository) EmailExists(email string) (bool, error) {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ?`, email)

	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repository) List() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT id, email, name, role_id, is_active, created_at, 
			   updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	return users, nil
}

func (r *Repository) Update(user models.User) error {
	query := `
		UPDATE users SET 
			email = ?, name = ?, role_id = ?, is_active = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		user.Email,
		user.Name,
		user.RoleID,
		user.IsActive,
		user.ID,
	)
	return err
}

func (r *Repository) Delete(id int) error {
	_, err := r.db.Exec(
		`UPDATE users SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		id,
	)
	return err
}

// scanUser - Helper para scan do sql.Row
func (r *Repository) scanUser(row *sql.Row) (*models.User, error) {
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

// scanUserFromRows - Helper para scan do sql.Rows
func (r *Repository) scanUserFromRows(rows *sql.Rows) (*models.User, error) {
	var user models.User
	var lastLoginAt, profileImageUrl sql.NullString
	var lastPasswordUpdate sql.NullTime

	err := rows.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
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
