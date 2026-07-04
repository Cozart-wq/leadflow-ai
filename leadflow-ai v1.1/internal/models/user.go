package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserRoleMember = "member"
	UserRoleAdmin  = "admin"
)

// User — учётная запись владельца задач и лидов. PasswordHash никогда не
// сериализуется в JSON (json:"-"), чтобы исключить случайную утечку хэша
// через API-ответы, даже если модель по невнимательности будет возвращена
// напрямую из обработчика вместо DTO.
type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name" json:"name"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
