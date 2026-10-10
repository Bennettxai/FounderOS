-- 138_workspace_desktop_spaces.sql
-- Workspace-owned desktop spaces for shared BusinessOS desktops.

CREATE TABLE IF NOT EXISTS workspace_desktop_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    kind VARCHAR(40) NOT NULL DEFAULT 'personal' CHECK (kind IN ('personal', 'team', 'workspace')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_desktop_spaces_workspace
    ON workspace_desktop_spaces(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_workspace_desktop_spaces_kind
    ON workspace_desktop_spaces(workspace_id, kind, name);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_workspace_id_fkey'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_workspace_id_fkey
            FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_kind_check'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_kind_check
            CHECK (kind IN ('personal', 'team', 'workspace'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_config_check'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_config_check
            CHECK (jsonb_typeof(config) = 'object');
    END IF;
END $$;
