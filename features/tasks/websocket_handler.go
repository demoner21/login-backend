package tasks

import (
	"encoding/json"
	"loginbackend/internal/http/middleware"
	"net/http"
	"strconv"
	"time"

	// Alias IMPORTANTE:
	ws "loginbackend/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	gws "github.com/gorilla/websocket"
)

// Upgrader do Gorilla (usando alias gws)
var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Ajustar para produção
	},
}

type Handler struct {
	service  *Service
	hub      *ws.Hub
	validate *validator.Validate
}

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewHandler(service *Service, hub *ws.Hub) *Handler {
	return &Handler{
		service:  service,
		hub:      hub,
		validate: validator.New(),
	}
}

// CreateTask
// @Summary Create a new task
// @Description Cria uma nova tarefa e inicializa o vector clock
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body CreateTaskRequest true "Dados da tarefa"
// @Success 201 {object} Response{data=Task}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /tasks [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Pegar UserID do token (middleware Auth)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Chama Service
	task, err := h.service.CreateTask(claims.UserID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	// Broadcast via WebSocket (usando o Hub exportado)
	msg := &ws.Message{
		Type:      "task_created",
		TaskID:    task.ID,
		Payload:   json.RawMessage(`{"title":"` + task.Title + `"}`),
		UserID:    claims.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	h.hub.Broadcast <- msg

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: "Tarefa criada com sucesso",
		Data:    task,
	})
}

// HandleWebSocket gerencia a conexão WS para tarefas
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade usando Gorilla
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Cria Client usando struct do pacote interno 'ws'
	client := &ws.Client{
		ID:     "ws-" + claims.UserID + "-" + time.Now().String(),
		UserID: claims.UserID,
		Conn:   conn,
		Hub:    h.hub,
		Send:   make(chan []byte, 256),
		Rooms:  make(map[string]bool),
	}

	h.hub.Register <- client

	go h.writePump(client)
	go h.readPump(client)
}

// readPump (Lê do cliente, ex: join_room)
func (h *Handler) readPump(client *ws.Client) {
	defer func() {
		h.hub.Unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg ws.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "join_room":
			h.hub.JoinRoom(client, msg.TaskID)
		case "leave_room":
			h.hub.LeaveRoom(client, msg.TaskID)
		}
	}
}

// writePump (Escreve para o cliente)
func (h *Handler) writePump(client *ws.Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(gws.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(gws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(gws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ListTasks lista as tarefas do usuário logado
// @Summary List user tasks
// @Description Retorna todas as tarefas ativas do usuário
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{data=[]Task}
// @Failure 500 {object} Response
// @Router /tasks [get]
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tasks, err := h.service.ListTasks(claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: tasks})
}

// DeleteTask remove uma tarefa (soft delete)
// @Summary Delete a task
// @Description Marca uma tarefa como deletada
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /tasks/{id} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Pega o ID da URL usando Chi
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "ID da tarefa é obrigatório"})
		return
	}

	// Chama o serviço
	err := h.service.DeleteTask(taskID, claims.UserID)
	if err != nil {
		// Se não encontrou ou não é dono, tratamos como erro genérico ou 404
		// Para simplificar, retornamos 404/403 mascarado ou 500
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // Ou NotFound dependendo da lógica exata do erro
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	// Opcional: Notificar via WebSocket que a tarefa foi removida
	msg := &ws.Message{
		Type:      "task_deleted",
		TaskID:    taskID,
		UserID:    claims.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	h.hub.Broadcast <- msg

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Tarefa removida com sucesso"})
}

// UpdateTask atualiza uma tarefa existente
// @Summary Update a task
// @Description Atualiza campos de uma tarefa (título, status, etc)
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Param request body UpdateTaskRequest true "Campos para atualizar"
// @Success 200 {object} Response{data=Task}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /tasks/{id} [put]
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "ID obrigatório"})
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "JSON inválido"})
		return
	}

	// Validação
	if err := h.validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	updatedTask, err := h.service.UpdateTask(taskID, claims.UserID, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	// Broadcast Websocket
	msg := &ws.Message{
		Type:      "task_updated",
		TaskID:    updatedTask.ID,
		Payload:   json.RawMessage(`{"status":"` + updatedTask.Status + `", "version":` + strconv.FormatInt(updatedTask.Version, 10) + `}`),
		UserID:    claims.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	h.hub.Broadcast <- msg

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Message: "Tarefa atualizada",
		Data:    updatedTask,
	})
}
