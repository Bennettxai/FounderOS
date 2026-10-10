-- Migration 085: Workspace Version Snapshots
-- Description: Add version control system for workspaces
-- Allows users to save and restore workspace snapshots

CREATE TABLE IF NOT EXISTS workspace_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version_number VARCHAR(20) NOT NULL, -- "0.0.1", "0.0.2", etc.
    snapshot_data JSONB NOT NULL, -- Complete workspace state
    snapshot_metadata JSONB, -- Additional metadata (file counts, app counts, etc.)
    created_by VARCHAR(255) REFERENCES "user"(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure unique version per workspace
    UNIQUE(workspace_id, version_number)
);

-- Index for fetching workspace versions
CREATE INDEX idx_workspace_versions_workspace ON workspace_versions(workspace_id, created_at DESC);

-- Index for version lookup
CREATE INDEX idx_workspace_versions_version ON workspace_versions(workspace_id, version_number);

-- Comments
COMMENT ON TABLE workspace_versions IS 'Version control snapshots for workspaces';
COMMENT ON COLUMN workspace_versions.snapshot_data IS 'JSONB snapshot of workspace state: apps, files, settings';
COMMENT ON COLUMN workspace_versions.version_number IS 'Semantic version: 0.0.1, 0.0.2, etc.';
