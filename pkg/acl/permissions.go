package acl

import (
	"fmt"
	"strconv"
	"strings"
)

// Permission representa uma permissão individual usando bitmask
type Permission int

const (
	PermissionNone   Permission = 0
	PermissionRead   Permission = 1 << 0 // 1
	PermissionWrite  Permission = 1 << 1 // 2
	PermissionDelete Permission = 1 << 2 // 4
	PermissionShare  Permission = 1 << 3 // 8
	PermissionAdmin  Permission = 1 << 4 // 16
)

// Roles pré-definidos (combinações comuns)
const (
	RoleViewer     Permission = PermissionRead
	RoleEditor     Permission = PermissionRead | PermissionWrite
	RoleOwner      Permission = PermissionRead | PermissionWrite | PermissionDelete | PermissionShare
	RoleFullAccess Permission = PermissionRead | PermissionWrite | PermissionDelete | PermissionShare | PermissionAdmin
)

// ResourceType define os tipos de recursos que podem ter ACL
type ResourceType string

const (
	ResourceTask     ResourceType = "TASK"
	ResourceFarmArea ResourceType = "FARM_AREA"
	ResourceTeam     ResourceType = "TEAM"
	ResourceDocument ResourceType = "DOCUMENT"
)

// GranteeType define quem pode receber permissões
type GranteeType string

const (
	GranteeUser   GranteeType = "USER"
	GranteeTeam   GranteeType = "TEAM"
	GranteePublic GranteeType = "PUBLIC"
)

// Has verifica se uma permissão específica está presente no bitmask
func (p Permission) Has(perm Permission) bool {
	return p&perm == perm
}

// Add adiciona uma permissão ao bitmask
func (p Permission) Add(perm Permission) Permission {
	return p | perm
}

// Remove remove uma permissão do bitmask
func (p Permission) Remove(perm Permission) Permission {
	return p &^ perm
}

// String retorna representação legível das permissões
func (p Permission) String() string {
	perms := []string{}
	if p.Has(PermissionRead) {
		perms = append(perms, "READ")
	}
	if p.Has(PermissionWrite) {
		perms = append(perms, "WRITE")
	}
	if p.Has(PermissionDelete) {
		perms = append(perms, "DELETE")
	}
	if p.Has(PermissionShare) {
		perms = append(perms, "SHARE")
	}
	if p.Has(PermissionAdmin) {
		perms = append(perms, "ADMIN")
	}
	if len(perms) == 0 {
		return "NONE"
	}
	return strings.Join(perms, "|")
}

// ParsePermissions converte string ou int para Permission
func ParsePermissions(input interface{}) (Permission, error) {
	switch v := input.(type) {
	case int:
		return Permission(v), nil
	case string:
		// Aceita tanto nomes quanto números
		switch strings.ToUpper(v) {
		case "VIEWER":
			return RoleViewer, nil
		case "EDITOR":
			return RoleEditor, nil
		case "OWNER":
			return RoleOwner, nil
		case "ADMIN":
			return RoleFullAccess, nil
		default:
			// Tenta converter para int
			val, err := strconv.Atoi(v)
			if err != nil {
				return PermissionNone, fmt.Errorf("invalid permission: %s", v)
			}
			return Permission(val), nil
		}
	default:
		return PermissionNone, fmt.Errorf("unsupported permission type")
	}
}
