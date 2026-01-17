package tasks

import (
	"encoding/json"
	"errors"
	"loginbackend/pkg/utils"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTask(userID string, req CreateTaskRequest) (*Task, error) {
	// Validações básicas (pode usar validator library aqui)

	// Gera Snowflake ID
	taskID := utils.GenerateSnowflakeID()

	// Relógio vetorial inicial: { "user_id": 1 }
	initialClock := map[string]int64{userID: 1}
	clockJSON, _ := json.Marshal(initialClock)

	task := Task{
		ID:          taskID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      "Pending",
		DueDate:     req.DueDate,
		OwnerID:     userID,
		Version:     1,
		VectorClock: clockJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(task); err != nil {
		return nil, err
	}

	return &task, nil
}

// ListTasks chama o repositório para listar
func (s *Service) ListTasks(userID string) ([]Task, error) {
	tasks, err := s.repo.List(userID)
	if err != nil {
		return nil, err
	}

	// Retornar slice vazio ao invés de nil se não houver tarefas (melhor para JSON)
	if tasks == nil {
		return []Task{}, nil
	}

	return tasks, nil
}

// DeleteTask chama o repositório para deletar
func (s *Service) DeleteTask(taskID, userID string) error {
	// O repositório já valida se a task pertence ao usuário na cláusula WHERE
	return s.repo.Delete(taskID, userID)
}

func (s *Service) UpdateTask(taskID, userID string, req UpdateTaskRequest) (*Task, error) {
	// 1. Buscar estado atual
	task, err := s.repo.FindByID(taskID, userID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("tarefa não encontrada")
	}

	// 2. Aplicar mudanças
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	// 3. Lógica de Sistema Distribuído
	task.Version++

	// CORREÇÃO DO PANIC: Inicializar o mapa explicitamente antes de usar
	clockMap := make(map[string]int64)

	if len(task.VectorClock) > 0 {
		// Ignoramos erro de unmarshal, pois se falhar, usamos o map vazio criado acima
		_ = json.Unmarshal(task.VectorClock, &clockMap)
	}

	clockMap[userID]++ // Agora é seguro, pois o map nunca é nil

	newClock, _ := json.Marshal(clockMap)
	task.VectorClock = newClock

	// 4. Salvar
	if err := s.repo.Update(*task); err != nil {
		return nil, err
	}

	return task, nil
}
