-- +migrate Up
-- Workspace Versioning System
-- Allows capturing and restoring workspace snapshots

CREATE TABLE IF NOT EXISTS workspace_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version_number VARCHAR(20) NOT NULL,
    snapshot_data JSONB NOT NULL,
    snapshot_metadata JSONB DEFAULT '{}'::jsonb,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, version_number)
);

CREATE INDEX idx_workspace_versions_workspace ON workspace_versions(workspace_id);
CREATE INDEX idx_workspace_versions_created_at ON workspace_versions(workspace_id, created_at DESC);

COMMENT ON TABLE workspace_versions IS 'Stores workspace snapshots for versioning and restore';
COMMENT ON COLUMN workspace_versions.snapshot_data IS 'Complete workspace state including apps, members, settings, etc.';
COMMENT ON COLUMN workspace_versions.snapshot_metadata IS 'Metadata about snapshot: app count, member count, storage size, etc.';

-- +migrate Down
DROP TABLE IF EXISTS workspace_versions CASCADE;
