package users

import (
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db}
}

func (r *Repository) Create(u User) error {
	query := `
        INSERT INTO users 
        (id, email, name, password_hash, role_id) 
        VALUES (?, ?, ?, ?, ?)
    `
	_, err := r.db.Exec(query, u.ID, u.Email, u.Name, u.PasswordHash, u.RoleID)
	return err
}

func (r *Repository) FindByEmail(email string) (*User, error) {
	row := r.db.QueryRow(`
        SELECT id, email, name, password_hash, role_id, is_active, 
               created_at, updated_at, last_password_update, is_email_verified
        FROM users WHERE email = ?
    `, email)

	var u User
	if err := row.Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.RoleID,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.LastPasswordUpdate,
		&u.IsEmailVerified,
	); err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *Repository) EmailExists(email string) bool {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ?`, email)

	var count int
	row.Scan(&count)

	return count > 0
}

func (r *Repository) List() ([]User, error) {
	rows, err := r.db.Query(`
        SELECT id, email, name, role_id, is_active, created_at, 
               updated_at, last_password_update, is_email_verified
        FROM users ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User

	for rows.Next() {
		var u User
		err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.Name,
			&u.RoleID,
			&u.IsActive,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.LastPasswordUpdate,
			&u.IsEmailVerified,
		)
		if err != nil {
			return nil, err
		}

		list = append(list, u)
	}

	return list, nil
}

func (r *Repository) UpdateLastLogin(id string) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = current_timestamp WHERE id = ?`, id)
	return err
}
