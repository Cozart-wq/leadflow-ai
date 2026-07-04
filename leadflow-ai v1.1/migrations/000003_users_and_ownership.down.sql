DROP INDEX IF EXISTS idx_tasks_user_id;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS users;
