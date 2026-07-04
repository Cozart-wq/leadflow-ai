DROP INDEX IF EXISTS idx_leads_task_id;

ALTER TABLE leads
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS ai_message,
    DROP COLUMN IF EXISTS ai_recommendation,
    DROP COLUMN IF EXISTS quality_score,
    DROP COLUMN IF EXISTS socials,
    DROP COLUMN IF EXISTS task_id;

DROP TABLE IF EXISTS tasks;
DROP TYPE IF EXISTS task_status;
