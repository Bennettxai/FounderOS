-- ================================================================
-- Migration: 087_app_versions
-- Description: Add versioning system for user-generated apps
-- Author: Database Migration Expert
-- Date: 2026-01-21
-- ================================================================

-- +migrate Up
-- Description: Create app_versions table for snapshotting and restoring app states

-- ================================================================
-- TABLE: app_versions
-- Description: Stores version snapshots of user-generated apps
-- ================================================================
CREATE TABLE IF NOT EXISTS app_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID NOT NULL REFERENCES user_generated_apps(id) ON DELETE CASCADE,
    version_number VARCHAR(20) NOT NULL,

    -- Snapshot data (complete app state)
    snapshot_data JSONB NOT NULL,
    snapshot_metadata JSONB DEFAULT '{}'::jsonb,

    -- Version metadata
    change_summary TEXT,
    created_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE(app_id, version_number)
);

-- ================================================================
-- INDEXES
-- ================================================================

-- Primary lookup: get versions for a specific app
CREATE INDEX idx_app_versions_app_id ON app_versions(app_id);

-- Time-based queries: get recent versions
CREATE INDEX idx_app_versions_created_at ON app_versions(created_at DESC);

-- User tracking: find versions created by specific user
CREATE INDEX idx_app_versions_created_by ON app_versions(created_by) WHERE created_by IS NOT NULL;

-- Composite: app + time (most common query pattern)
CREATE INDEX idx_app_versions_app_time ON app_versions(app_id, created_at DESC);

-- GIN index for snapshot_data JSONB queries (if needed for filtering)
CREATE INDEX idx_app_versions_snapshot_data ON app_versions USING GIN(snapshot_data);

-- ================================================================
-- COMMENTS
-- ================================================================

COMMENT ON TABLE app_versions IS 'Version snapshots of user-generated apps for rollback/restore functionality';
COMMENT ON COLUMN app_versions.version_number IS 'Semantic version number (e.g., 0.0.1, 0.1.0, 1.0.0)';
COMMENT ON COLUMN app_versions.snapshot_data IS 'Complete app state including config, settings, custom code';
COMMENT ON COLUMN app_versions.snapshot_metadata IS 'Additional metadata about the snapshot (file sizes, checksums, etc.)';
COMMENT ON COLUMN app_versions.change_summary IS 'Human-readable description of changes in this version';

-- +migrate Down
-- Description: Remove app versioning system

DROP INDEX IF EXISTS idx_app_versions_snapshot_data;
DROP INDEX IF EXISTS idx_app_versions_app_time;
DROP INDEX IF EXISTS idx_app_versions_created_by;
DROP INDEX IF EXISTS idx_app_versions_created_at;
DROP INDEX IF EXISTS idx_app_versions_app_id;

DROP TABLE IF EXISTS app_versions CASCADE;
