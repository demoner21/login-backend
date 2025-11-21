package http

import (
	"loginbackend/config"
	"loginbackend/internal/http/ratelimit"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg *config.Config, redisClient *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	// Rate Limit GLOBAL (DDoS Protection)
	r.Use(httprate.Limit(
		100,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitCounter(ratelimit.NewRedisLimitCounter(redisClient, "global-rate")),
	))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	return r
}
