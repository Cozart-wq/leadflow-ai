package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/leadflow-ai/leadflow-ai/internal/models"
)

// ErrEmailTaken возвращается при попытке зарегистрировать пользователя
// с уже занятым email. Отдельная sentinel-ошибка позволяет вызывающему
// коду (auth_service) отличить конфликт email от прочих ошибок БД и
// вернуть по нему осмысленный HTTP-статус (409), не разбирая текст
// драйверной ошибки.
var ErrEmailTaken = errors.New("user_repository: email already registered")

// postgresUniqueViolation — SQLSTATE код нарушения уникального
// ограничения в PostgreSQL.
const postgresUniqueViolation = "23505"

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
		VALUES (:id, :email, :password_hash, :name, :role, now(), now())
		RETURNING created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, query, u)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == postgresUniqueViolation {
			return ErrEmailTaken
		}
		return fmt.Errorf("user_repository: create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&u.CreatedAt, &u.UpdatedAt); err != nil {
			return fmt.Errorf("user_repository: create scan: %w", err)
		}
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, fmt.Errorf("user_repository: get by email: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("user_repository: get by id: %w", err)
	}
	return &u, nil
}
