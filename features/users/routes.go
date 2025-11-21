package users

import (
	"loginbackend/internal/http/middleware"
	"os"

	"github.com/go-chi/chi/v5"
)

func Routes(handler *Handler) (string, func(r chi.Router)) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "fallback-secret-change-in-production"
	}

	return "/users", func(r chi.Router) {
		// Rota pública (registro)
		r.Post("/", handler.CreateUser) // POST /users (criar conta)

		// Rotas protegidas (requerem autenticação)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtSecret))

			r.Get("/", handler.ListUsers)         // GET /users
			r.Get("/{id}", handler.GetUser)       // GET /users/{id}
			r.Put("/{id}", handler.UpdateUser)    // PUT /users/{id}
			r.Delete("/{id}", handler.DeleteUser) // DELETE /users/{id}
		})
	}
}
