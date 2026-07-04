CREATE TYPE task_status AS ENUM ('pending', 'running', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    status task_status NOT NULL DEFAULT 'pending',
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE leads
    ADD COLUMN IF NOT EXISTS task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS socials JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS quality_score INTEGER,
    ADD COLUMN IF NOT EXISTS ai_recommendation TEXT,
    ADD COLUMN IF NOT EXISTS ai_message TEXT,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'new';

CREATE INDEX IF NOT EXISTS idx_leads_task_id ON leads (task_id);
