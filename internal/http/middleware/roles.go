package middleware

import (
	"net/http"
)

// RequireRole verifica se o usuário tem o RoleID necessário
func RequireRole(requiredRoleID int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Recuperar dados do usuário do contexto
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// 2. Verificar se o RoleID é suficiente (1 = SUPER_ADMIN)
			if claims.RoleID != requiredRoleID {
				http.Error(w, "Forbidden: access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
