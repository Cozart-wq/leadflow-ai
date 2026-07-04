package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/leadflow-ai/leadflow-ai/internal/models"
)

type LeadRepository struct {
	db *sqlx.DB
}

func NewLeadRepository(db *sqlx.DB) *LeadRepository {
	return &LeadRepository{db: db}
}

func (r *LeadRepository) Create(ctx context.Context, l *models.Lead) error {
	query := `
		INSERT INTO leads (id, task_id, company_name, website, email, phone, socials, quality_score, ai_recommendation, ai_message, status, created_at, updated_at)
		VALUES (:id, :task_id, :company_name, :website, :email, :phone, :socials, :quality_score, :ai_recommendation, :ai_message, :status, now(), now())
		RETURNING created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, query, l)
	if err != nil {
		return fmt.Errorf("lead_repository: create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&l.CreatedAt, &l.UpdatedAt); err != nil {
			return fmt.Errorf("lead_repository: create scan: %w", err)
		}
	}
	return nil
}

func (r *LeadRepository) Update(ctx context.Context, l *models.Lead) error {
	query := `
		UPDATE leads SET
			company_name = :company_name,
			website = :website,
			email = :email,
			phone = :phone,
			socials = :socials,
			quality_score = :quality_score,
			ai_recommendation = :ai_recommendation,
			ai_message = :ai_message,
			status = :status,
			updated_at = now()
		WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, l)
	if err != nil {
		return fmt.Errorf("lead_repository: update: %w", err)
	}
	return nil
}

// Владение лидом не хранится напрямую в таблице leads — оно наследуется
// от задачи через task_id (лид принадлежит тому, кому принадлежит задача,
// в рамках которой он был найден). Поэтому все выборки, видимые
// пользователю, идут через JOIN с tasks и фильтр по tasks.user_id, а не
// по отдельной колонке leads.user_id.

func (r *LeadRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*models.Lead, error) {
	var l models.Lead
	err := r.db.GetContext(ctx, &l, `
		SELECT leads.* FROM leads
		JOIN tasks ON tasks.id = leads.task_id
		WHERE leads.id = $1 AND tasks.user_id = $2`, id, userID)
	if err != nil {
		return nil, fmt.Errorf("lead_repository: get by id: %w", err)
	}
	return &l, nil
}

func (r *LeadRepository) List(ctx context.Context, userID uuid.UUID, taskID *uuid.UUID, limit, offset int) ([]models.Lead, error) {
	var leads []models.Lead
	var err error
	if taskID != nil {
		err = r.db.SelectContext(ctx, &leads, `
			SELECT leads.* FROM leads
			JOIN tasks ON tasks.id = leads.task_id
			WHERE tasks.user_id = $1 AND leads.task_id = $2
			ORDER BY leads.created_at DESC LIMIT $3 OFFSET $4`,
			userID, *taskID, limit, offset)
	} else {
		err = r.db.SelectContext(ctx, &leads, `
			SELECT leads.* FROM leads
			JOIN tasks ON tasks.id = leads.task_id
			WHERE tasks.user_id = $1
			ORDER BY leads.created_at DESC LIMIT $2 OFFSET $3`,
			userID, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("lead_repository: list: %w", err)
	}
	return leads, nil
}

func (r *LeadRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM leads
		USING tasks
		WHERE leads.id = $1 AND leads.task_id = tasks.id AND tasks.user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("lead_repository: delete: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("lead_repository: delete rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("lead_repository: delete: lead not found")
	}
	return nil
}

func (r *LeadRepository) ExistsByWebsite(ctx context.Context, website string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM leads WHERE website = $1)`, website)
	if err != nil {
		return false, fmt.Errorf("lead_repository: exists by website: %w", err)
	}
	return exists, nil
}
