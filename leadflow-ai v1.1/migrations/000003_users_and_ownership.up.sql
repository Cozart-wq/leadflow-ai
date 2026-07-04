-- 000003_users_and_ownership.up.sql
--
-- Вводит пользователей и владение задачами. users.id используется как
-- владелец tasks — это минимальный шаг к multi-tenant модели, необходимый
-- перед тем, как строить Company/Project management (v1.2+), которые
-- будут принадлежать пользователю так же, как сейчас задачи.

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- user_id остаётся nullable на уровне схемы: это единственный способ
-- добавить колонку с FK на существующую таблицу без бэкфилла или простоя.
-- Прикладной код (task_service) всегда проставляет user_id для новых
-- задач — NOT NULL как бизнес-инвариант обеспечивается на уровне сервиса,
-- а не БД, до тех пор пока не будет проведён бэкфилл старых данных.
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks (user_id);
