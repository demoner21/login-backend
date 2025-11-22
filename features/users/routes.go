package users

import (
	"loginbackend/internal/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func Routes(handler *Handler, jwtSecret string, redisClient *redis.Client) (string, func(r chi.Router)) {

	return "/users", func(r chi.Router) {
		// 1. Rota Pública (Registro)
		r.Post("/", handler.CreateUser)

		// 2. Rotas Protegidas (Requerem Login)
		r.Group(func(r chi.Router) {
			// Middleware de Autenticação
			r.Use(middleware.AuthMiddleware(jwtSecret, redisClient))

			// A) LISTAGEM GERAL: Apenas Admins (opcional, ou aberto a todos dependendo da regra)
			// Vamos manter restrito a Admin para evitar vazamento de base
			r.With(middleware.RequireRole(1)).Get("/", handler.ListUsers)

			// B) LEITURA DE PERFIL: Aberto para qualquer usuário logado ver o perfil de outros
			// (Se quiser que seja privado, mova para o grupo de baixo)
			r.Get("/{id}", handler.GetUser)

			// C) ZONA DE PERIGO (Owner Policy): Apenas o Dono ou Admin toca aqui
			r.Group(func(r chi.Router) {
				r.Use(middleware.OwnerOrAdmin)

				r.Put("/{id}", handler.UpdateUser)
				r.Delete("/{id}", handler.DeleteUser)

				r.Post("/{id}/change-password", handler.ChangePassword)
				r.Post("/{id}/avatar", handler.UploadAvatar)
			})
		})
	}
}
