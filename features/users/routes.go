package users

import (
	"loginbackend/internal/http/middleware"

	"github.com/go-chi/chi/v5"
)

func Routes(handler *Handler, jwtSecret string) (string, func(r chi.Router)) {

	return "/users", func(r chi.Router) {
		// Rota pública (registro)
		r.Post("/", handler.CreateUser)

		// Rotas protegidas (requerem autenticação)
		r.Group(func(r chi.Router) {
			// Passamos o jwtSecret recebido para o middleware
			r.Use(middleware.AuthMiddleware(jwtSecret))

			r.Get("/", handler.ListUsers)
			r.Get("/{id}", handler.GetUser)
			r.Put("/{id}", handler.UpdateUser)
			r.Delete("/{id}", handler.DeleteUser)
		})
	}
}
