package acl

import (
	"time"

	"loginbackend/pkg/acl"
)

// ============================================
// MODELS
// ============================================

type ACL struct {
	ID           int64            `json:"id"`
	ResourceID   string           `json:"resource_id"`
	ResourceType acl.ResourceType `json:"resource_type"`
	GranteeType  acl.GranteeType  `json:"grantee_type"`
	GranteeID    *string          `json:"grantee_id,omitempty"`
	Permissions  acl.Permission   `json:"permissions"`
	GrantedBy    string           `json:"granted_by"`
	GrantedAt    time.Time        `json:"granted_at"`
	ExpiresAt    *time.Time       `json:"expires_at,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// GrantACLRequest - Request para criar/atualizar ACL
type GrantACLRequest struct {
	ResourceID   string           `json:"resource_id" validate:"required"`
	ResourceType acl.ResourceType `json:"resource_type" validate:"required"`
	GranteeType  acl.GranteeType  `json:"grantee_type" validate:"required"`
	GranteeID    *string          `json:"grantee_id"`
	Permissions  interface{}      `json:"permissions" validate:"required"`
	ExpiresAt    *time.Time       `json:"expires_at"`
	Metadata     map[string]any   `json:"metadata"`
}

// ShareRequest - Request simplificado para compartilhar
type ShareRequest struct {
	ResourceID   string           `json:"resource_id" validate:"required"`
	ResourceType acl.ResourceType `json:"resource_type" validate:"required"`
	ShareWith    []ShareTarget    `json:"share_with" validate:"required,min=1"`
}

type ShareTarget struct {
	Type      acl.GranteeType `json:"type" validate:"required"`
	ID        *string         `json:"id"`
	Role      string          `json:"role" validate:"required"`
	ExpiresAt *time.Time      `json:"expires_at"`
}

// SharedResource - Resposta para listagem
type SharedResource struct {
	ResourceID   string           `json:"resource_id"`
	ResourceType acl.ResourceType `json:"resource_type"`
	ResourceData interface{}      `json:"resource_data"`
	Permissions  acl.Permission   `json:"permissions"`
	SharedBy     string           `json:"shared_by"`
	SharedAt     time.Time        `json:"shared_at"`
}
