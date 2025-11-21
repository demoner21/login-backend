package middleware

import (
	"context"
	"loginbackend/pkg/utils"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// AuthMiddleware valida o token JWT e adiciona os claims ao context
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Obter token do header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Verificar formato "Bearer {token}"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validar token
			claims, err := utils.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Adicionar claims ao context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext obtém os claims do usuário do context
func GetUserFromContext(ctx context.Context) *utils.TokenClaims {
	if claims, ok := ctx.Value(UserContextKey).(*utils.TokenClaims); ok {
		return claims
	}
	return nil
}
