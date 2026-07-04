package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)

type Task struct {
	ID        uuid.UUID     `db:"id" json:"id"`
	UserID    uuid.NullUUID `db:"user_id" json:"user_id,omitempty"`
	Query     string        `db:"query" json:"query"`
	Status    string        `db:"status" json:"status"`
	Error     *string       `db:"error" json:"error,omitempty"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt time.Time     `db:"updated_at" json:"updated_at"`
}
