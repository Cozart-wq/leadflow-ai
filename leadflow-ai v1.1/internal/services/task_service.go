package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/leadflow-ai/leadflow-ai/internal/models"
	"github.com/leadflow-ai/leadflow-ai/internal/repositories"
)

type TaskService struct {
	tasks *repositories.TaskRepository
}

func NewTaskService(tasks *repositories.TaskRepository) *TaskService {
	return &TaskService{tasks: tasks}
}

func (s *TaskService) Create(ctx context.Context, userID uuid.UUID, query string) (*models.Task, error) {
	if query == "" {
		return nil, fmt.Errorf("task_service: query must not be empty")
	}
	t := &models.Task{
		ID:     uuid.New(),
		UserID: uuid.NullUUID{UUID: userID, Valid: true},
		Query:  query,
		Status: models.TaskStatusPending,
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("task_service: create: %w", err)
	}
	return t, nil
}

func (s *TaskService) Get(ctx context.Context, userID, id uuid.UUID) (*models.Task, error) {
	return s.tasks.GetByID(ctx, userID, id)
}

func (s *TaskService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Task, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.tasks.List(ctx, userID, limit, offset)
}

func (s *TaskService) UpdateStatus(ctx context.Context, id uuid.UUID, status string, taskErr *string) error {
	return s.tasks.UpdateStatus(ctx, id, status, taskErr)
}
