package tasks

import (
	"encoding/json"
	"time"
)

// Task mapeia a tabela 'tasks'
type Task struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Priority    string          `json:"priority"`
	Status      string          `json:"status"`
	OwnerID     string          `json:"owner_id"`
	DueDate     *time.Time      `json:"due_date,omitempty"` // NOVO CAMPO
	Version     int64           `json:"version"`
	VectorClock json.RawMessage `json:"vector_clock"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateTaskRequest é o payload esperado via Swagger
type CreateTaskRequest struct {
	Title       string     `json:"title" validate:"required,min=3"`
	Description string     `json:"description"`
	Priority    string     `json:"priority" validate:"oneof=Low Medium High"`
	DueDate     *time.Time `json:"due_date" example:"2026-01-20T15:00:00Z"` // NOVO CAMPO
}

// TaskEvent mapeia a tabela 'task_events'
type TaskEvent struct {
	ID             int64           `json:"id"`
	TaskID         string          `json:"task_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Version        int64           `json:"version"`
	UserID         string          `json:"user_id"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
}

// UpdateTaskRequest para alterações parciais (PATCH/PUT)
type UpdateTaskRequest struct {
	Title       *string    `json:"title" validate:"omitempty,min=3"`
	Description *string    `json:"description"`
	Priority    *string    `json:"priority" validate:"omitempty,oneof=Low Medium High"`
	Status      *string    `json:"status" validate:"omitempty,oneof=Pending InProgress Done Canceled"`
	DueDate     *time.Time `json:"due_date"`
}

// SyncRequest é o payload que o cliente envia quando recupera conexão
type SyncRequest struct {
	LastPulledVersion int64        `json:"last_pulled_version"` // Última versão que o cliente conhece
	Changes           []SyncChange `json:"changes"`             // Lista de mudanças locais
}

type SyncChange struct {
	Type    string          `json:"type" validate:"required,oneof=CREATE UPDATE DELETE"`
	TaskID  string          `json:"task_id" validate:"required"`
	Payload json.RawMessage `json:"payload"` // O conteúdo da Task (completo ou parcial)
}

// SyncResponse devolve o que foi processado e o que há de novo no servidor
type SyncResponse struct {
	SyncedCount int    `json:"synced_count"`
	NewTasks    []Task `json:"new_tasks"` // Tarefas que o cliente não tinha
}
