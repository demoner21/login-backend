package acl

import (
	"errors"
	"fmt"

	pkgacl "loginbackend/pkg/acl"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GrantAccess concede permissões (validação + lógica de negócio)
func (s *Service) GrantAccess(userID string, req GrantACLRequest) error {
	// 1. Validar se usuário tem permissão de SHARE no recurso
	canShare, err := s.repo.CheckPermission(userID, req.ResourceID, req.ResourceType, pkgacl.PermissionShare)
	if err != nil {
		return err
	}

	// Exceção: Se for owner, sempre pode compartilhar (mesmo sem ACL explícita)
	isOwner, _ := s.repo.IsOwner(userID, req.ResourceID, req.ResourceType)

	if !canShare && !isOwner {
		return errors.New("você não tem permissão para compartilhar este recurso")
	}

	// 2. Parse de permissões
	permissions, err := pkgacl.ParsePermissions(req.Permissions)
	if err != nil {
		return err
	}

	// 3. Validar GranteeID conforme tipo
	if req.GranteeType == pkgacl.GranteePublic && req.GranteeID != nil {
		return errors.New("PUBLIC não pode ter grantee_id")
	}
	if req.GranteeType != pkgacl.GranteePublic && req.GranteeID == nil {
		return errors.New("USER e TEAM requerem grantee_id")
	}

	// 4. Criar ACL
	acl := ACL{
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		GranteeType:  req.GranteeType,
		GranteeID:    req.GranteeID,
		Permissions:  permissions,
		GrantedBy:    userID,
		ExpiresAt:    req.ExpiresAt,
		Metadata:     req.Metadata,
	}

	return s.repo.GrantACL(acl)
}

// Share - Método simplificado para compartilhar com múltiplos alvos
func (s *Service) Share(userID string, req ShareRequest) error {
	// Verificar permissão de compartilhamento
	canShare, err := s.repo.CheckPermission(userID, req.ResourceID, req.ResourceType, pkgacl.PermissionShare)
	if err != nil {
		return err
	}

	isOwner, _ := s.repo.IsOwner(userID, req.ResourceID, req.ResourceType)
	if !canShare && !isOwner {
		return errors.New("permissão negada para compartilhar")
	}

	// Processar cada alvo
	for _, target := range req.ShareWith {
		permissions, err := pkgacl.ParsePermissions(target.Role)
		if err != nil {
			return fmt.Errorf("role inválida '%s': %w", target.Role, err)
		}

		acl := ACL{
			ResourceID:   req.ResourceID,
			ResourceType: req.ResourceType,
			GranteeType:  target.Type,
			GranteeID:    target.ID,
			Permissions:  permissions,
			GrantedBy:    userID,
			ExpiresAt:    target.ExpiresAt,
		}

		if err := s.repo.GrantACL(acl); err != nil {
			return err
		}
	}

	return nil
}

// RevokeAccess remove permissões
func (s *Service) RevokeAccess(userID string, resourceID string, resourceType pkgacl.ResourceType, granteeID *string, granteeType pkgacl.GranteeType) error {
	// Verificar se tem permissão de ADMIN ou SHARE
	canRevoke, err := s.repo.CheckPermission(userID, resourceID, resourceType, pkgacl.PermissionAdmin)
	if err != nil {
		return err
	}

	if !canRevoke {
		canRevoke, _ = s.repo.CheckPermission(userID, resourceID, resourceType, pkgacl.PermissionShare)
	}

	isOwner, _ := s.repo.IsOwner(userID, resourceID, resourceType)

	if !canRevoke && !isOwner {
		return errors.New("permissão negada para revogar acesso")
	}

	return s.repo.RevokeACL(resourceID, resourceType, granteeID, granteeType)
}

// GetResourceACL lista todas as ACLs de um recurso
func (s *Service) GetResourceACL(userID string, resourceID string, resourceType pkgacl.ResourceType) ([]ACL, error) {
	// Verificar se tem permissão de ler o recurso OU é owner
	canRead, err := s.repo.CheckPermission(userID, resourceID, resourceType, pkgacl.PermissionRead)
	if err != nil {
		return nil, err
	}

	isOwner, _ := s.repo.IsOwner(userID, resourceID, resourceType)

	if !canRead && !isOwner {
		return nil, errors.New("permissão negada")
	}

	return s.repo.GetACL(resourceID, resourceType)
}

// CheckPermission wrapper para uso externo (middleware)
func (s *Service) CheckPermission(userID, resourceID string, resourceType pkgacl.ResourceType, requiredPerm pkgacl.Permission) (bool, error) {
	// Owner sempre tem todas as permissões
	isOwner, err := s.repo.IsOwner(userID, resourceID, resourceType)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// Verifica ACL
	return s.repo.CheckPermission(userID, resourceID, resourceType, requiredPerm)
}

// ListSharedWithMe lista recursos compartilhados com o usuário
func (s *Service) ListSharedWithMe(userID string, resourceType *pkgacl.ResourceType) ([]SharedResource, error) {
	return s.repo.ListSharedWithMe(userID, resourceType)
}

// ListSharedByMe lista recursos compartilhados pelo usuário
func (s *Service) ListSharedByMe(userID string, resourceType *pkgacl.ResourceType) ([]SharedResource, error) {
	return s.repo.ListSharedByMe(userID, resourceType)
}

// UpdatePermissions atualiza permissões existentes (helper)
func (s *Service) UpdatePermissions(userID string, resourceID string, resourceType pkgacl.ResourceType, granteeID *string, granteeType pkgacl.GranteeType, newPermissions pkgacl.Permission) error {
	// Validar se pode alterar
	canShare, err := s.repo.CheckPermission(userID, resourceID, resourceType, pkgacl.PermissionShare)
	if err != nil {
		return err
	}

	isOwner, _ := s.repo.IsOwner(userID, resourceID, resourceType)

	if !canShare && !isOwner {
		return errors.New("permissão negada")
	}

	// Atualizar (usa UPSERT do GrantACL)
	acl := ACL{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		GranteeType:  granteeType,
		GranteeID:    granteeID,
		Permissions:  newPermissions,
		GrantedBy:    userID,
	}

	return s.repo.GrantACL(acl)
}
