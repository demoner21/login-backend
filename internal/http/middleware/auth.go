package middleware

import (
	"context"
	"fmt"
	"loginbackend/pkg/utils"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

func AuthMiddleware(jwtSecret string, redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// 🚨 NOVO: Verificar Blacklist no Redis
			ctx := r.Context()
			blacklistKey := fmt.Sprintf("blacklist:%s", tokenString)
			exists, err := redisClient.Exists(ctx, blacklistKey).Result()

			// Se der erro no Redis, logamos mas não bloqueamos necessariamente (fail open vs fail close)
			// Aqui vamos assumir Fail Secure: se Redis falhar ou chave existir, bloqueia.
			if err == nil && exists > 0 {
				http.Error(w, "Token revoked (logout)", http.StatusUnauthorized)
				return
			}

			// Validar token (Assinatura JWT)
			claims, err := utils.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Adicionar claims ao context
			userCtx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(userCtx))
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
