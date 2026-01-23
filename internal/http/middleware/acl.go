package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	pkgacl "loginbackend/pkg/acl"
)

// ACLService interface para evitar import circular
type ACLService interface {
	CheckPermission(userID, resourceID string, resourceType pkgacl.ResourceType, requiredPerm pkgacl.Permission) (bool, error)
}

// RequirePermission verifica se o usuário tem permissão para acessar o recurso
func RequirePermission(aclService ACLService, resourceType pkgacl.ResourceType, requiredPerm pkgacl.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			resourceID := chi.URLParam(r, "id")
			if resourceID == "" {
				http.Error(w, "resource ID missing", http.StatusBadRequest)
				return
			}

			hasPermission, err := aclService.CheckPermission(
				claims.UserID,
				resourceID,
				resourceType,
				requiredPerm,
			)

			if err != nil {
				http.Error(w, "error checking permissions", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOwnerOrShared verifica se usuário é owner OU tem permissão via ACL
func RequireOwnerOrShared(aclService ACLService, resourceType pkgacl.ResourceType, requiredPerm pkgacl.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			resourceID := chi.URLParam(r, "id")
			if resourceID == "" {
				http.Error(w, "resource ID missing", http.StatusBadRequest)
				return
			}

			hasPermission, err := aclService.CheckPermission(
				claims.UserID,
				resourceID,
				resourceType,
				requiredPerm,
			)

			if err != nil {
				http.Error(w, "error checking permissions", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, "Forbidden: you don't have access to this resource", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireCanShare - Middleware específico
func RequireCanShare(aclService ACLService, resourceType pkgacl.ResourceType) func(http.Handler) http.Handler {
	return RequirePermission(aclService, resourceType, pkgacl.PermissionShare)
}

// RequireCanDelete - Middleware específico
func RequireCanDelete(aclService ACLService, resourceType pkgacl.ResourceType) func(http.Handler) http.Handler {
	return RequirePermission(aclService, resourceType, pkgacl.PermissionDelete)
}

// RequireCanWrite - Middleware específico
func RequireCanWrite(aclService ACLService, resourceType pkgacl.ResourceType) func(http.Handler) http.Handler {
	return RequirePermission(aclService, resourceType, pkgacl.PermissionWrite)
}
