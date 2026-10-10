-- Project templates are reusable, workspace-scoped delivery blueprints.
-- The public edition provides the primitive and starts with no seeded templates.

CREATE TABLE IF NOT EXISTS delivery_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID,
    key           VARCHAR(120) NOT NULL,
    name          VARCHAR(300) NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    phases        JSONB NOT NULL DEFAULT '[]'::jsonb,
    deliverables  JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_builtin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_by    VARCHAR(255),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_templates_global_key
    ON delivery_templates(key) WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_templates_ws_key
    ON delivery_templates(workspace_id, key) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_delivery_templates_workspace
    ON delivery_templates(workspace_id);
