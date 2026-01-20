package tasks

import (
	"loginbackend/internal/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func Routes(handler *Handler, jwtSecret string, redisClient *redis.Client) (string, func(r chi.Router)) {
	return "/tasks", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, redisClient))

		r.Post("/", handler.CreateTask)
		r.Get("/", handler.ListTasks)
		r.Put("/{id}", handler.UpdateTask)
		r.Delete("/{id}", handler.DeleteTask)
		r.Get("/ws", handler.HandleWebSocket)
		r.Post("/sync", handler.SyncTasks)
	}
}
