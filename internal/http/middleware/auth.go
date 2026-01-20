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
			tokenString := ""

			// 1. Tenta pegar do Header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}

			// 2. Se não achou no Header, tenta pegar da Query String (Para WebSocket)
			// Isso é necessário porque o JS no browser não envia header no handshake
			if tokenString == "" {
				tokenString = r.URL.Query().Get("token")
			}

			// 3. Se ainda estiver vazio, erro
			if tokenString == "" {
				http.Error(w, "Authorization header or token query param required", http.StatusUnauthorized)
				return
			}

			// 🚨 Verificar Blacklist no Redis (Código existente mantido)
			ctx := r.Context()
			blacklistKey := fmt.Sprintf("blacklist:%s", tokenString)
			exists, err := redisClient.Exists(ctx, blacklistKey).Result()
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
