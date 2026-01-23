package acl

import (
	"loginbackend/internal/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// ============================================
// ROUTES
// ============================================

func Routes(handler *Handler, jwtSecret string, redisClient *redis.Client) (string, func(r chi.Router)) {
	return "/api", func(r chi.Router) {
		// Middleware global de autenticação
		r.Use(middleware.AuthMiddleware(jwtSecret, redisClient))

		// ACL Management
		r.Post("/acl", handler.GrantACL)
		r.Get("/acl/{resource_id}", handler.GetACL)
		r.Delete("/acl/{resource_id}", handler.RevokeACL)

		// Sharing
		r.Post("/share", handler.ShareResource)
		r.Get("/shared-with-me", handler.ListSharedWithMe)
		r.Get("/shared-by-me", handler.ListSharedByMe)
	}
}
