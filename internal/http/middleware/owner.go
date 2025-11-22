package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// OwnerOrAdmin verifica se o usuário logado é o DONO do recurso ou um ADMIN
func OwnerOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Pegar o ID do usuário logado (vem do token JWT via AuthMiddleware)
		claims := GetUserFromContext(r.Context())
		if claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Pegar o ID do recurso alvo na URL (ex: /users/{id})
		targetID := chi.URLParam(r, "id")

		// 3. REGRA DE OURO:
		// Permite se:
		// A) O usuário for SUPER_ADMIN (Role 1)
		// OU
		// B) O ID do usuário logado for IGUAL ao ID alvo da URL

		isOwner := claims.UserID == targetID
		isAdmin := claims.RoleID == 1

		if isOwner || isAdmin {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden: você não tem permissão para alterar este perfil", http.StatusForbidden)
			return
		}
	})
}
