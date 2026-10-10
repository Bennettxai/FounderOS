-- Module Records: workspace-scoped record store for "kind: records" custom modules.
-- Each row belongs to a module and stores arbitrary field data as jsonb.

CREATE TABLE IF NOT EXISTS module_records (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id   UUID        NOT NULL,
    workspace_id UUID       NOT NULL,
    created_by  VARCHAR,
    data        JSONB       NOT NULL DEFAULT '{}',
    position    INT         DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- List records for a module in display order
CREATE INDEX IF NOT EXISTS idx_module_records_module_position
    ON module_records (module_id, position, created_at);

-- Workspace-scoped queries (e.g. bulk deletes, access checks)
CREATE INDEX IF NOT EXISTS idx_module_records_workspace
    ON module_records (workspace_id);
