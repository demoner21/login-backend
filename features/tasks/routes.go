package tasks

import (
	"loginbackend/internal/http/middleware"
	pkgacl "loginbackend/pkg/acl"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func Routes(
	handler *Handler,
	jwtSecret string,
	redisClient *redis.Client,
	aclService middleware.ACLService, // INTERFACE, não tipo concreto
) (string, func(r chi.Router)) {
	return "/tasks", func(r chi.Router) {
		// Middleware global de autenticação
		r.Use(middleware.AuthMiddleware(jwtSecret, redisClient))

		// ============================================
		// ROTAS PÚBLICAS (SEM ACL - Apenas Auth)
		// ============================================

		// POST /tasks - Criar tarefa (qualquer usuário autenticado)
		r.Post("/", handler.CreateTask)

		// GET /tasks - Listar minhas tarefas (retorna apenas owner_id do usuário)
		r.Get("/", handler.ListTasks)

		// WebSocket - Conectar ao hub
		r.Get("/ws", handler.HandleWebSocket)

		// POST /tasks/sync - Sincronizar offline
		r.Post("/sync", handler.SyncTasks)

		// ============================================
		// ROTAS PROTEGIDAS POR ACL (Owner ou Compartilhado)
		// ============================================

		r.Group(func(r chi.Router) {
			// GET /tasks/{id} - Ver detalhes (requer READ)
			// NOTA: Se você tiver um handler GetTask separado, adicione aqui
			// r.With(
			//     middleware.RequireOwnerOrShared(aclService, acl.ResourceTask, acl.PermissionRead),
			// ).Get("/{id}", handler.GetTask)

			// PUT /tasks/{id} - Atualizar (requer WRITE)
			r.With(
				middleware.RequireOwnerOrShared(aclService, pkgacl.ResourceTask, pkgacl.PermissionWrite),
			).Put("/{id}", handler.UpdateTask)

			// DELETE /tasks/{id} - Deletar (requer DELETE)
			r.With(
				middleware.RequireOwnerOrShared(aclService, pkgacl.ResourceTask, pkgacl.PermissionDelete),
			).Delete("/{id}", handler.DeleteTask)
		})
	}
}
