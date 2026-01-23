package acl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	pkgacl "loginbackend/pkg/acl"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GrantACL concede ou atualiza permissões
func (r *Repository) GrantACL(acl ACL) error {
	var metadataJSON []byte
	var err error

	if acl.Metadata != nil {
		metadataJSON, err = json.Marshal(acl.Metadata)
		if err != nil {
			return fmt.Errorf("erro ao converter metadata para json: %w", err)
		}
	} else {
		// Se for nulo, salvamos como um objeto JSON vazio '{}' ou NULL
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO acls 
		(resource_id, resource_type, grantee_type, grantee_id, permissions, granted_by, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (resource_id, resource_type, grantee_type, grantee_id)
		DO UPDATE SET 
			permissions = EXCLUDED.permissions,
			expires_at = EXCLUDED.expires_at,
			metadata = EXCLUDED.metadata
	`

	_, err = r.db.Exec(query,
		acl.ResourceID,
		acl.ResourceType,
		acl.GranteeType,
		acl.GranteeID,
		int(acl.Permissions),
		acl.GrantedBy,
		acl.ExpiresAt,
		metadataJSON,
	)

	return err
}

// GetACL busca ACLs de um recurso
func (r *Repository) GetACL(resourceID string, resourceType pkgacl.ResourceType) ([]ACL, error) {
	query := `
		SELECT id, resource_id, resource_type, grantee_type, grantee_id, 
		       permissions, granted_by, granted_at, expires_at
		FROM acls
		WHERE resource_id = $1 AND resource_type = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY granted_at DESC
	`

	rows, err := r.db.Query(query, resourceID, resourceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acls []ACL
	for rows.Next() {
		var a ACL
		var granteeID sql.NullString
		var expiresAt sql.NullTime
		var permissions int

		err := rows.Scan(
			&a.ID, &a.ResourceID, &a.ResourceType, &a.GranteeType, &granteeID,
			&permissions, &a.GrantedBy, &a.GrantedAt, &expiresAt,
		)
		if err != nil {
			return nil, err
		}

		if granteeID.Valid {
			a.GranteeID = &granteeID.String
		}
		if expiresAt.Valid {
			a.ExpiresAt = &expiresAt.Time
		}
		a.Permissions = pkgacl.Permission(permissions)

		acls = append(acls, a)
	}

	return acls, nil
}

// RevokeACL remove permissões
func (r *Repository) RevokeACL(resourceID string, resourceType pkgacl.ResourceType, granteeID *string, granteeType pkgacl.GranteeType) error {
	// 1. SEGURANÇA: Validação de Tipo
	// Tentamos converter a string para int64. Se falhar, nem chamamos o banco.
	resourceIDInt, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return fmt.Errorf("resource_id inválido (deve ser numérico): %w", err)
	}

	// Opcional: Se granteeID não for nulo, validar ele também
	var granteeIDInt *int64
	if granteeID != nil {
		val, err := strconv.ParseInt(*granteeID, 10, 64)
		if err != nil {
			return fmt.Errorf("grantee_id inválido: %w", err)
		}
		granteeIDInt = &val
	}

	query := `
		DELETE FROM acls
		WHERE resource_id = $1 
		  AND resource_type = $2
		  AND grantee_type = $3
		  AND (grantee_id = $4 OR ($4 IS NULL AND grantee_id IS NULL))
	`

	// 2. ATOMICIDADE: O comando DELETE é atômico por natureza no Postgres.
	// Passamos o resourceIDInt (int64) em vez da string.
	result, err := r.db.Exec(query, resourceIDInt, resourceType, granteeType, granteeIDInt)

	if err != nil {
		return fmt.Errorf("erro ao revogar permissão: %w", err)
	}

	// 3. Verificação (Opcional, mas boa prática)
	// Se quiser saber se alguém realmente perdeu acesso:
	rows, _ := result.RowsAffected()
	if rows == 0 {
		// Não é erro, mas pode ser útil para logar que nada mudou
		// return fmt.Errorf("nenhuma permissão encontrada para revogar")
	}

	return nil
}

// CheckPermission verifica se usuário tem permissão
func (r *Repository) CheckPermission(userID, resourceID string, resourceType pkgacl.ResourceType, requiredPerm pkgacl.Permission) (bool, error) {
	// 1. Tentar cache primeiro
	cached, err := r.getCachedPermissions(userID, resourceID, resourceType)
	if err == nil && cached != nil {
		return cached.Has(requiredPerm), nil
	}

	// 2. Calcular via função SQL
	var effectivePerm int
	err = r.db.QueryRow(`
		SELECT calculate_effective_permissions($1, $2, $3)
	`, userID, resourceID, resourceType).Scan(&effectivePerm)

	if err != nil {
		return false, err
	}

	perm := pkgacl.Permission(effectivePerm)

	// 3. Salvar no cache
	r.cachePermissions(userID, resourceID, resourceType, perm)

	return perm.Has(requiredPerm), nil
}

// getCachedPermissions busca permissões do cache
func (r *Repository) getCachedPermissions(userID, resourceID string, resourceType pkgacl.ResourceType) (*pkgacl.Permission, error) {
	var perm int
	err := r.db.QueryRow(`
		SELECT effective_permissions 
		FROM resource_permissions_cache
		WHERE user_id = $1 
		  AND resource_id = $2 
		  AND resource_type = $3
		  AND expires_at > NOW()
	`, userID, resourceID, resourceType).Scan(&perm)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p := pkgacl.Permission(perm)
	return &p, nil
}

// cachePermissions salva permissões no cache
func (r *Repository) cachePermissions(userID, resourceID string, resourceType pkgacl.ResourceType, perm pkgacl.Permission) {
	r.db.Exec(`
		INSERT INTO resource_permissions_cache 
		(user_id, resource_id, resource_type, effective_permissions)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, resource_id, resource_type)
		DO UPDATE SET 
			effective_permissions = EXCLUDED.effective_permissions,
			cached_at = NOW(),
			expires_at = NOW() + INTERVAL '1 hour'
	`, userID, resourceID, resourceType, int(perm))
}

// ListSharedWithMe lista recursos compartilhados com o usuário
func (r *Repository) ListSharedWithMe(userID string, resourceType *pkgacl.ResourceType) ([]SharedResource, error) {
	query := `
		SELECT DISTINCT
			a.resource_id,
			a.resource_type,
			calculate_effective_permissions($1, a.resource_id, a.resource_type) AS permissions,
			a.granted_by,
			MIN(a.granted_at) AS shared_at
		FROM acls a
		LEFT JOIN team_members tm ON a.grantee_type = 'TEAM' AND tm.team_id = a.grantee_id
		WHERE (
			(a.grantee_type = 'USER' AND a.grantee_id = $1) OR
			(a.grantee_type = 'TEAM' AND tm.user_id = $1) OR
			a.grantee_type = 'PUBLIC'
		)
		AND (a.expires_at IS NULL OR a.expires_at > NOW())
	`

	args := []interface{}{userID}

	if resourceType != nil {
		query += ` AND a.resource_type = $2`
		args = append(args, *resourceType)
	}

	query += ` GROUP BY a.resource_id, a.resource_type, a.granted_by ORDER BY shared_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []SharedResource
	for rows.Next() {
		var sr SharedResource
		var permissions int

		err := rows.Scan(
			&sr.ResourceID,
			&sr.ResourceType,
			&permissions,
			&sr.SharedBy,
			&sr.SharedAt,
		)
		if err != nil {
			return nil, err
		}

		sr.Permissions = pkgacl.Permission(permissions)
		resources = append(resources, sr)
	}

	return resources, nil
}

// ListSharedByMe lista recursos compartilhados pelo usuário
func (r *Repository) ListSharedByMe(userID string, resourceType *pkgacl.ResourceType) ([]SharedResource, error) {
	query := `
		SELECT DISTINCT
			resource_id,
			resource_type,
			permissions,
			granted_by,
			granted_at
		FROM acls
		WHERE granted_by = $1
		AND (expires_at IS NULL OR expires_at > NOW())
	`

	args := []interface{}{userID}

	if resourceType != nil {
		query += ` AND resource_type = $2`
		args = append(args, *resourceType)
	}

	query += ` ORDER BY granted_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []SharedResource
	for rows.Next() {
		var sr SharedResource
		var permissions int

		err := rows.Scan(
			&sr.ResourceID,
			&sr.ResourceType,
			&permissions,
			&sr.SharedBy,
			&sr.SharedAt,
		)
		if err != nil {
			return nil, err
		}

		sr.Permissions = pkgacl.Permission(permissions)
		resources = append(resources, sr)
	}

	return resources, nil
}

// IsOwner verifica se usuário é o dono do recurso
func (r *Repository) IsOwner(userID, resourceID string, resourceType pkgacl.ResourceType) (bool, error) {
	var ownerID string
	var query string

	switch resourceType {
	case pkgacl.ResourceTask:
		query = `SELECT owner_id FROM tasks WHERE id = $1 AND deleted_at IS NULL`
	case pkgacl.ResourceFarmArea:
		query = `SELECT owner_id FROM farm_areas WHERE id = $1`
	case pkgacl.ResourceTeam:
		query = `
			SELECT user_id FROM team_members 
			WHERE team_id = $1 AND user_id = $2 AND role = 'Admin'
		`
		err := r.db.QueryRow(query, resourceID, userID).Scan(&ownerID)
		return err == nil, nil
	default:
		return false, fmt.Errorf("unsupported resource type")
	}

	err := r.db.QueryRow(query, resourceID).Scan(&ownerID)
	if err != nil {
		return false, err
	}

	return ownerID == userID, nil
}
