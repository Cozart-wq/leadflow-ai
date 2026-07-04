// Package database отвечает за установку соединения с PostgreSQL.
//
// Почему sqlx, а не database/sql напрямую и не полноценная ORM (gorm):
//   - database/sql напрямую требует много шаблонного кода для сканирования
//     строк в структуры (rows.Scan(&a, &b, &c, ...)), что многословно
//     и хрупко при изменении схемы;
//   - sqlx добавляет удобные StructScan/Get/Select поверх стандартного
//     database/sql, не пряча SQL за абстракцией — репозитории по-прежнему
//     пишут явный SQL, что важно для контроля над запросами и производительностью;
//   - полноценная ORM (gorm) генерирует SQL за разработчика, что усложняет
//     отладку сложных запросов и противоречит принципу "явное лучше неявного",
//     которого придерживается этот проект.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // pq регистрирует драйвер "postgres" для database/sql

	"github.com/leadflow-ai/leadflow-ai/internal/config"
)

// New открывает пул соединений с PostgreSQL и проверяет его доступность.
//
// Возвращается *sqlx.DB, а не обёрнутый интерфейс — на этом этапе проекта
// дополнительная абстракция не нужна: репозитории и так изолируют
// доступ к БД от остального кода (см. internal/repositories). Добавлять
// интерфейс поверх *sqlx.DB стоит только тогда, когда появится реальная
// потребность (например, подмена БД в юнит-тестах через мок).
func New(ctx context.Context, cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: failed to open connection: %w", err)
	}

	// Настройки пула соединений выставляются явно, а не оставляются
	// на усмотрение стандартных значений Go (безлимитные соединения
	// могут исчерпать лимиты подключений PostgreSQL под нагрузкой).
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database: failed to ping database: %w", err)
	}

	return db, nil
}
