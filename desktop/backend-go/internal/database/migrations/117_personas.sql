-- 117_personas.sql
-- The Personas module: per-workspace buyer/customer personas (ideal customer
-- profiles) so humans AND AI agents target messaging and offers consistently.
-- Workspace-scoped; one row per persona.

CREATE TABLE IF NOT EXISTS personas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    segment VARCHAR(200) DEFAULT '',
    pains TEXT DEFAULT '',
    objections TEXT DEFAULT '',
    language TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_personas_workspace ON personas(workspace_id);
