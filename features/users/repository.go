package users

import (
	"database/sql"
	"fmt"
	"loginbackend/features/shared/models"
	"loginbackend/pkg/utils"
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
		(id, email, name, password_hash, role_id, is_active) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.RoleID,
		user.IsActive,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar usuário no banco de dados: %w", err)
	}

	return nil
}

func (r *Repository) FindByID(id string) (*models.User, error) {
	// ⚠️ Atualizado para buscar TODAS as colunas novas
	row := r.db.QueryRow(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url, refresh_token,
			   phone, job_title, location, avatar_url,
			   country, city, state, postal_code, tax_id
		FROM users WHERE id = $1
	`, id)

	return r.scanUser(row)
}

func (r *Repository) FindByEmail(email string) (*models.User, error) {
	// ⚠️ Atualizado para buscar TODAS as colunas novas
	row := r.db.QueryRow(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url, refresh_token,
			   phone, job_title, location, avatar_url,
			   country, city, state, postal_code, tax_id
		FROM users WHERE email = $1
	`, email)

	return r.scanUser(row)
}

func (r *Repository) EmailExists(email string) (bool, error) {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = $1`, email)

	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("erro ao verificar existência do email: %w", err)
	}

	return count > 0, nil
}

func (r *Repository) List() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT id, email, name, password_hash, role_id, is_active, 
			   created_at, updated_at, last_password_update, is_email_verified,
			   last_login_at, profile_image_url, refresh_token,
			   phone, job_title, location, avatar_url,
			   country, city, state, postal_code, tax_id
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query de listagem: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear usuário: %w", err)
		}
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos resultados: %w", err)
	}

	return users, nil
}

func (r *Repository) Update(user models.User) error {
	query := `
		UPDATE users SET 
			email = $1, name = $2, role_id = $3, is_active = $4,
			phone = $5, job_title = $6, location = $7,
			country = $8, city = $9, state = $10, postal_code = $11, tax_id = $12,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $13
	`
	result, err := r.db.Exec(query,
		user.Email, user.Name, user.RoleID, user.IsActive,
		user.Phone, user.JobTitle, user.Location,
		user.Country, user.City, user.State, user.PostalCode, user.TaxID,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("erro ao executar update do usuário: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("usuário não encontrado para update")
	}

	return nil
}

func (r *Repository) Delete(id string) error {
	result, err := r.db.Exec(
		`UPDATE users SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("erro ao desativar usuário: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("usuário não encontrado para desativação")
	}

	return nil
}

func (r *Repository) UpdateLastLogin(userID string) error {
	result, err := r.db.Exec(
		`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar último login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("usuário não encontrado para atualizar último login")
	}

	return nil
}

func (r *Repository) SaveRefreshToken(userID string, refreshToken string) error {
	result, err := r.db.Exec(
		`UPDATE users SET refresh_token = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		refreshToken, userID,
	)
	if err != nil {
		return fmt.Errorf("erro ao salvar refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("usuário não encontrado para salvar refresh token")
	}

	return nil
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
		return nil, fmt.Errorf("erro ao buscar usuário por refresh token: %w", err)
	}

	return &user, nil
}

func (r *Repository) ClearRefreshToken(userID string) error {
	result, err := r.db.Exec(
		`UPDATE users SET refresh_token = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("erro ao limpar refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("usuário não encontrado para limpar refresh token")
	}

	return nil
}

// scanUser - Helper para scan do sql.Row
func (r *Repository) scanUser(row *sql.Row) (*models.User, error) {
	var user models.User
	// Declaração correta de TODAS as variáveis nullable
	var lastLoginAt, refreshToken, profileImageUrl sql.NullString
	var phone, jobTitle, location, avatarUrl, country, city, state, postalCode, taxId sql.NullString
	var lastPasswordUpdate sql.NullTime

	err := row.Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.RoleID, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &lastPasswordUpdate, &user.IsEmailVerified,
		&lastLoginAt, &profileImageUrl, &refreshToken,
		// Novos campos
		&phone, &jobTitle, &location, &avatarUrl,
		&country, &city, &state, &postalCode, &taxId,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao scanear usuário: %w", err)
	}

	return r.mapScannedValues(&user, lastPasswordUpdate, profileImageUrl, refreshToken, lastLoginAt, phone, jobTitle, location, avatarUrl, country, city, state, postalCode, taxId)
}

// scanUserFromRows - Helper para scan do sql.Rows
func (r *Repository) scanUserFromRows(rows *sql.Rows) (*models.User, error) {
	var user models.User
	var lastLoginAt, refreshToken, profileImageUrl sql.NullString
	var phone, jobTitle, location, avatarUrl, country, city, state, postalCode, taxId sql.NullString
	var lastPasswordUpdate sql.NullTime

	err := rows.Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.RoleID, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &lastPasswordUpdate, &user.IsEmailVerified,
		&lastLoginAt, &profileImageUrl, &refreshToken,
		// Novos campos
		&phone, &jobTitle, &location, &avatarUrl,
		&country, &city, &state, &postalCode, &taxId,
	)

	if err != nil {
		return nil, fmt.Errorf("erro ao scanear usuário do rows: %w", err)
	}

	return r.mapScannedValues(&user, lastPasswordUpdate, profileImageUrl, refreshToken, lastLoginAt, phone, jobTitle, location, avatarUrl, country, city, state, postalCode, taxId)
}

// mapScannedValues - Helper privado para evitar duplicação de código de mapeamento
func (r *Repository) mapScannedValues(
	user *models.User,
	lastPasswordUpdate sql.NullTime,
	profileImageUrl, refreshToken, lastLoginAt sql.NullString,
	phone, jobTitle, location, avatarUrl, country, city, state, postalCode, taxId sql.NullString,
) (*models.User, error) {

	if lastPasswordUpdate.Valid {
		user.LastPasswordUpdate = lastPasswordUpdate.Time
	} else {
		user.LastPasswordUpdate = user.CreatedAt
	}

	if profileImageUrl.Valid {
		user.AvatarURL = &profileImageUrl.String
	}

	if refreshToken.Valid {
		user.RefreshToken = &refreshToken.String
	}

	if lastLoginAt.Valid {
		lastLoginTime, err := utils.ParseTime(lastLoginAt.String)
		if err == nil {
			user.LastLoginAt = &lastLoginTime
		}
	}

	if phone.Valid {
		user.Phone = &phone.String
	}
	if jobTitle.Valid {
		user.JobTitle = &jobTitle.String
	}
	if location.Valid {
		user.Location = &location.String
	}
	if avatarUrl.Valid {
		user.AvatarURL = &avatarUrl.String
	}
	if country.Valid {
		user.Country = &country.String
	}
	if city.Valid {
		user.City = &city.String
	}
	if state.Valid {
		user.State = &state.String
	}
	if postalCode.Valid {
		user.PostalCode = &postalCode.String
	}
	if taxId.Valid {
		user.TaxID = &taxId.String
	}

	return user, nil
}

func (r *Repository) UpdatePassword(userID, newHash string) error {
	_, err := r.db.Exec(`
		UPDATE users 
		SET password_hash = $1, last_password_update = CURRENT_TIMESTAMP 
		WHERE id = $2
	`, newHash, userID)
	return err
}
