package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/leadflow-ai/leadflow-ai/internal/models"
)

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *models.Task) error {
	query := `
		INSERT INTO tasks (id, user_id, query, status, error, created_at, updated_at)
		VALUES (:id, :user_id, :query, :status, :error, now(), now())
		RETURNING created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, query, t)
	if err != nil {
		return fmt.Errorf("task_repository: create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&t.CreatedAt, &t.UpdatedAt); err != nil {
			return fmt.Errorf("task_repository: create scan: %w", err)
		}
	}
	return nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, taskErr *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = $1, error = $2, updated_at = now() WHERE id = $3`,
		status, taskErr, id)
	if err != nil {
		return fmt.Errorf("task_repository: update status: %w", err)
	}
	return nil
}

// GetByID возвращает задачу, только если она принадлежит userID. Задача,
// принадлежащая другому пользователю, неотличима от несуществующей — это
// намеренно: сообщать "задача существует, но не ваша" раскрывает факт
// существования чужого ресурса по угадываемому id.
func (r *TaskRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*models.Task, error) {
	var t models.Task
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return nil, fmt.Errorf("task_repository: get by id: %w", err)
	}
	return &t, nil
}

func (r *TaskRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.SelectContext(ctx, &tasks,
		`SELECT * FROM tasks WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("task_repository: list: %w", err)
	}
	return tasks, nil
}
