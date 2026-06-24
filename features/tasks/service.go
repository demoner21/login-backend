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
	// userID fica na assinatura para uso futuro em auditoria
	// (ex: coluna deleted_by). A permissão já foi validada
	// pelo middleware RequireOwnerOrShared.
	return s.repo.Delete(taskID)
}

func (s *Service) UpdateTask(taskID, userID string, req UpdateTaskRequest) (*Task, error) {
	task, err := s.repo.FindByID(taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("tarefa não encontrada")
	}

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

	task.Version++

	clockMap := make(map[string]int64)
	if len(task.VectorClock) > 0 {
		_ = json.Unmarshal(task.VectorClock, &clockMap)
	}
	clockMap[userID]++ // userID = quem está editando agora (owner ou colaborador via ACL)

	newClock, _ := json.Marshal(clockMap)
	task.VectorClock = newClock

	if err := s.repo.Update(*task, userID); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Service) ProcessSync(userID string, req SyncRequest) (*SyncResponse, error) {
	// 1. Processar as mudanças vindas do Cliente (Push)
	count := 0

	// Nota: Em produção, isso deve ser feito dentro de uma transação única no Repository
	// Para simplificar aqui, vamos chamar os métodos existentes, mas o ideal é Transactional.
	for _, change := range req.Changes {
		var err error

		switch change.Type {
		case "CREATE":
			var createReq CreateTaskRequest
			if json.Unmarshal(change.Payload, &createReq); err == nil {
				// Força o ID que veio do cliente (importante para offline)
				// Precisaríamos refatorar o CreateTask para aceitar ID externo ou
				// lidar com UUIDs temporários. Vamos assumir criação simples por enquanto.
				_, err = s.CreateTask(userID, createReq)
			}

		case "UPDATE":
			var updateReq UpdateTaskRequest
			if json.Unmarshal(change.Payload, &updateReq); err == nil {
				_, err = s.UpdateTask(change.TaskID, userID, updateReq)
			}

		case "DELETE":
			err = s.DeleteTask(change.TaskID, userID)
		}

		if err == nil {
			count++
		}
		// Se der erro (ex: conflito), por enquanto ignoramos ou logamos.
		// Numa implementação avançada, retornamos uma lista de erros.
	}

	// 2. Buscar novidades para o Cliente (Pull)
	// Vamos precisar de um método no Repo que busque tasks com Version > LastPulledVersion
	newTasks, err := s.repo.ListChanges(userID, req.LastPulledVersion)
	if err != nil {
		return nil, err
	}

	if newTasks == nil {
		newTasks = []Task{}
	}

	return &SyncResponse{
		SyncedCount: count,
		NewTasks:    newTasks,
	}, nil
}
