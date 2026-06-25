package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	pkgacl "loginbackend/pkg/acl"
	"loginbackend/pkg/utils"
	"time"
)

// UserResolver é o necessário de 'users' para resolver email -> ID.
// Mantido como interface mínima para não acoplar tasks a users.Service inteiro.
type UserResolver interface {
	FindIDByEmail(email string) (userID string, found bool, err error)
}

// ACLGranter é o necessário de 'acl' para conceder acesso e descobrir
// quem tem acesso a um recurso — usado tanto no compartilhamento
// quanto na notificação em tempo real de mudanças.
type ACLGranter interface {
	GrantTaskAccess(grantedBy, resourceID, granteeUserID string, permissions pkgacl.Permission) error
	ListCollaboratorIDs(resourceID string, resourceType pkgacl.ResourceType) ([]string, error)
}

type Service struct {
	repo         *Repository
	userResolver UserResolver
	aclGranter   ACLGranter
}

func NewService(repo *Repository, userResolver UserResolver, aclGranter ACLGranter) *Service {
	return &Service{repo: repo, userResolver: userResolver, aclGranter: aclGranter}
}

func (s *Service) CreateTask(userID string, req CreateTaskRequest) (*CreateTaskResult, error) {
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

	result := &CreateTaskResult{Task: task}

	for _, email := range req.SharedWith {
		granteeID, found, err := s.userResolver.FindIDByEmail(email)
		if err != nil {
			result.ShareWarnings = append(result.ShareWarnings, fmt.Sprintf("%s: erro ao buscar usuário", email))
			continue
		}
		if !found {
			result.ShareWarnings = append(result.ShareWarnings, fmt.Sprintf("%s: usuário não encontrado", email))
			continue
		}
		if granteeID == userID {
			result.ShareWarnings = append(result.ShareWarnings, fmt.Sprintf("%s: você não pode compartilhar consigo mesmo", email))
			continue
		}

		// Padrão ao compartilhar na criação: somente leitura.
		// Para conceder edição, o owner usa o modal de compartilhar
		// (ShareTaskModal) já com a tarefa criada.
		if err := s.aclGranter.GrantTaskAccess(userID, taskID, granteeID, pkgacl.PermissionRead); err != nil {
			result.ShareWarnings = append(result.ShareWarnings, fmt.Sprintf("%s: erro ao compartilhar", email))
			continue
		}

		result.SharedUserIDs = append(result.SharedUserIDs, granteeID)
	}

	return result, nil
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

// GetTaskOwner retorna o owner_id de uma task. Usado pelo handler para
// montar a lista de notificação antes de operações destrutivas (delete),
// já que após o soft delete a busca normal não encontra mais a task.
func (s *Service) GetTaskOwner(taskID string) (string, error) {
	task, err := s.repo.FindByID(taskID)
	if err != nil {
		return "", err
	}
	if task == nil {
		return "", errors.New("tarefa não encontrada")
	}
	return task.OwnerID, nil
}

// ListRecipients monta a lista de quem deve ser notificado sobre uma
// mudança nesta task: o owner + todos os colaboradores com ACL ativa,
// excluindo o próprio autor da ação (que já sabe o que fez).
//
// Resolvido aqui, uma vez, no momento da mutação — não dentro do Hub,
// que deve permanecer um roteador em memória sem acesso a banco.
func (s *Service) ListRecipients(ownerID, actorID, taskID string) ([]string, error) {
	collaborators, err := s.aclGranter.ListCollaboratorIDs(taskID, pkgacl.ResourceTask)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{actorID: true}
	var recipients []string

	if !seen[ownerID] {
		recipients = append(recipients, ownerID)
		seen[ownerID] = true
	}
	for _, id := range collaborators {
		if !seen[id] {
			recipients = append(recipients, id)
			seen[id] = true
		}
	}
	return recipients, nil
}
