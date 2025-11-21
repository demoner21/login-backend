package auth

import (
	"github.com/go-chi/chi/v5"
)

func Routes(handler *Handler) (string, func(r chi.Router)) {
	return "/auth", func(r chi.Router) {
		r.Post("/login", handler.Login)
		r.Post("/refresh", handler.Refresh)
		r.Post("/logout", handler.Logout)
	}
}
