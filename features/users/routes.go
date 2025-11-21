package users

import (
	"github.com/go-chi/chi/v5"
)

func Routes(handler *Handler) (string, func(r chi.Router)) {
	return "/users", func(r chi.Router) {
		r.Post("/", handler.CreateUser)       // POST /users
		r.Get("/", handler.ListUsers)         // GET /users
		r.Get("/{id}", handler.GetUser)       // GET /users/{id}
		r.Put("/{id}", handler.UpdateUser)    // PUT /users/{id}
		r.Delete("/{id}", handler.DeleteUser) // DELETE /users/{id}
	}
}
