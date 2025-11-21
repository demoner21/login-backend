package auth

import (
	"loginbackend/internal/http/ratelimit"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/redis/go-redis/v9"
)

func Routes(handler *Handler, redisClient *redis.Client) (string, func(r chi.Router)) {
	return "/auth", func(r chi.Router) {

		// Rate Limit específico para LOGIN (Anti-Brute Force)
		loginLimiter := httprate.Limit(
			5,
			1*time.Minute,
			httprate.WithKeyFuncs(httprate.KeyByIP),
			// ✅ Usa o alias 'hrredis' aqui
			httprate.WithLimitCounter(ratelimit.NewRedisLimitCounter(redisClient, "global-rate-limit:")),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Muitas tentativas de login. Aguarde 1 minuto.", 429)
			}),
		)

		r.With(loginLimiter).Post("/login", handler.Login)
		r.Post("/refresh", handler.Refresh)
		r.Post("/logout", handler.Logout)
	}
}
