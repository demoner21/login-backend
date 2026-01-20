package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(task Task) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ATUALIZAÇÃO DA QUERY: Adicionado due_date ($11)
	queryTask := `
		INSERT INTO tasks (
			id, title, description, priority, status, owner_id, 
			version, vector_clock, created_at, updated_at, due_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = tx.Exec(queryTask,
		task.ID,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		task.OwnerID,
		task.Version,
		[]byte(task.VectorClock),
		task.CreatedAt,
		task.UpdatedAt,
		task.DueDate, // Parâmetro $11
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir task: %w", err)
	}

	// 2. Inserir Evento Inicial (TaskCreated)
	eventPayload := fmt.Sprintf(`{"title": "%s", "description": "%s"}`, task.Title, task.Description)

	// Gera ID do evento (Snowflake também é útil aqui, ou serial)
	// Assumindo BIGSERIAL no banco, passamos o resto. Se for Snowflake, gere aqui.
	queryEvent := `
		INSERT INTO task_events (task_id, event_type, payload, version, vector_clock, user_id, sequence_number)
		VALUES ($1, $2, $3, $4, $5, $6, (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM task_events))
	`
	_, err = tx.Exec(queryEvent,
		task.ID, "TaskCreated", eventPayload, task.Version, []byte(task.VectorClock), task.OwnerID,
	)
	if err != nil {
		return fmt.Errorf("erro ao inserir evento: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) List(userID string) ([]Task, error) {
	// Busca tarefas onde o usuário é dono OU (opcional) faz parte do time
	query := `
        SELECT id, title, description, priority, status, owner_id, 
               version, vector_clock, created_at, updated_at, due_date
        FROM tasks 
        WHERE owner_id = $1 AND deleted_at IS NULL
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var vectorClockBytes []byte // Para ler o JSONB do banco

		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Priority, &t.Status, &t.OwnerID,
			&t.Version, &vectorClockBytes, &t.CreatedAt, &t.UpdatedAt, &t.DueDate,
		)
		if err != nil {
			return nil, err
		}

		// Converter bytes de volta para JSON RawMessage
		t.VectorClock = json.RawMessage(vectorClockBytes)
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (r *Repository) ListTasks(userID string) ([]Task, error) {
	query := `
		SELECT id, title, description, priority, status, owner_id, 
			   version, vector_clock, created_at, updated_at, due_date
		FROM tasks 
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query de listagem: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var vectorClockBytes []byte
		var dueDate sql.NullTime // Usamos NullTime para garantir scan seguro de nulos

		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Priority, &t.Status, &t.OwnerID,
			&t.Version, &vectorClockBytes, &t.CreatedAt, &t.UpdatedAt, &dueDate,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear task: %w", err)
		}

		// Converte NullTime para *time.Time
		if dueDate.Valid {
			t.DueDate = &dueDate.Time
		}

		// Converte bytes do banco para json.RawMessage
		t.VectorClock = json.RawMessage(vectorClockBytes)

		tasks = append(tasks, t)
	}

	return tasks, nil
}

// Delete realiza um Soft Delete (marca deleted_at)
// Exige ownerID para garantir que ninguém delete a tarefa dos outros
func (r *Repository) Delete(taskID, userID string) error {
	query := `
		UPDATE tasks 
		SET deleted_at = CURRENT_TIMESTAMP 
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, taskID, userID)
	if err != nil {
		return fmt.Errorf("erro ao deletar task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("tarefa não encontrada ou permissão negada")
	}

	return nil
}

// FindByID busca uma tarefa específica (para validar antes do update)
func (r *Repository) FindByID(taskID, userID string) (*Task, error) {
	query := `
		SELECT id, title, description, priority, status, owner_id, 
			   version, vector_clock, created_at, updated_at, due_date
		FROM tasks 
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`
	var t Task
	var vectorClockBytes []byte
	var dueDate sql.NullTime

	err := r.db.QueryRow(query, taskID, userID).Scan(
		&t.ID, &t.Title, &t.Description, &t.Priority, &t.Status, &t.OwnerID,
		&t.Version, &vectorClockBytes, &t.CreatedAt, &t.UpdatedAt, &dueDate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Não encontrado
		}
		return nil, err
	}

	if dueDate.Valid {
		t.DueDate = &dueDate.Time
	}
	t.VectorClock = json.RawMessage(vectorClockBytes)

	return &t, nil
}

// Update atualiza a tarefa e insere o evento de mudança
func (r *Repository) Update(task Task) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Atualizar Tabela Tasks
	query := `
		UPDATE tasks 
		SET title = $1, description = $2, priority = $3, status = $4, 
			due_date = $5, version = $6, vector_clock = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND owner_id = $9
	`

	_, err = tx.Exec(query,
		task.Title, task.Description, task.Priority, task.Status,
		task.DueDate, task.Version, []byte(task.VectorClock),
		task.ID, task.OwnerID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar task: %w", err)
	}

	// 2. Registrar Evento (TaskUpdated)
	// Em um sistema real, o payload conteria apenas o "diff" (o que mudou)
	eventPayload := fmt.Sprintf(`{"status": "%s", "version": %d}`, task.Status, task.Version)

	eventQuery := `
		INSERT INTO task_events (task_id, event_type, payload, version, vector_clock, user_id, sequence_number)
		VALUES ($1, 'TaskUpdated', $2, $3, $4, $5, (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM task_events))
	`
	_, err = tx.Exec(eventQuery,
		task.ID, eventPayload, task.Version, []byte(task.VectorClock), task.OwnerID,
	)
	if err != nil {
		return fmt.Errorf("erro ao registrar evento de update: %w", err)
	}

	return tx.Commit()
}

// ListChanges busca tarefas que foram modificadas APÓS uma certa versão
// Isso inclui tarefas novas ou atualizadas por outros dispositivos
func (r *Repository) ListChanges(userID string, minVersion int64) ([]Task, error) {
	// Nota: Aqui assumimos que 'version' é global ou usamos updated_at
	// Para simplificar, vamos usar updated_at se version for complexo,
	// mas como temos version na tabela, usamos ela.

	query := `
		SELECT id, title, description, priority, status, owner_id, 
			   version, vector_clock, created_at, updated_at, due_date
		FROM tasks 
		WHERE owner_id = $1 AND version > $2
		ORDER BY version ASC
	`

	rows, err := r.db.Query(query, userID, minVersion)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar mudanças: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var vectorClockBytes []byte
		var dueDate sql.NullTime

		rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Priority, &t.Status, &t.OwnerID,
			&t.Version, &vectorClockBytes, &t.CreatedAt, &t.UpdatedAt, &dueDate,
		)
		if dueDate.Valid {
			t.DueDate = &dueDate.Time
		}
		t.VectorClock = json.RawMessage(vectorClockBytes)

		tasks = append(tasks, t)
	}
	return tasks, nil
}
