-- Migration: 043_sync_conflicts.sql
-- Description: Sync conflicts table for bidirectional sync conflict detection and resolution
-- Created: 2026-01-09
-- Phase: 3 (Data Sync)

-- =============================================================================
-- SYNC CONFLICTS TABLE
-- Tracks conflicts detected during bidirectional sync operations
-- =============================================================================
CREATE TABLE IF NOT EXISTS sync_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Conflict target
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,

    -- Conflict data
    local_data JSONB NOT NULL,
    remote_data JSONB NOT NULL,
    local_updated_at TIMESTAMPTZ NOT NULL,
    remote_updated_at TIMESTAMPTZ NOT NULL,
    conflict_fields TEXT[] NOT NULL,

    -- Resolution
    resolution_strategy VARCHAR(50),
    resolved_data JSONB,
    resolved_by VARCHAR(255),  -- User ID who resolved, NULL = automatic
    resolved_at TIMESTAMPTZ,
    reasoning TEXT,

    -- Metadata
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT valid_resolution_strategy CHECK (
        resolution_strategy IN ('timestamp_based', 'field_level_merge', 'manual_review', 'local_wins', 'remote_wins')
    )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_unresolved ON sync_conflicts(resolved_at)
    WHERE resolved_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sync_conflicts_entity ON sync_conflicts(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_detected ON sync_conflicts(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_strategy ON sync_conflicts(resolution_strategy);

-- Update trigger
CREATE OR REPLACE FUNCTION update_sync_conflicts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_conflicts_updated_at
    BEFORE UPDATE ON sync_conflicts
    FOR EACH ROW
    EXECUTE FUNCTION update_sync_conflicts_updated_at();

-- Comments
COMMENT ON TABLE sync_conflicts IS 'Tracks conflicts detected during bidirectional sync operations';
COMMENT ON COLUMN sync_conflicts.entity_type IS 'Type of entity that has conflicts (workspace, app, module)';
COMMENT ON COLUMN sync_conflicts.conflict_fields IS 'Array of field names that are in conflict';
COMMENT ON COLUMN sync_conflicts.resolution_strategy IS 'Strategy used to resolve the conflict';
COMMENT ON COLUMN sync_conflicts.resolved_by IS 'User who resolved manually, NULL for automatic resolution';
COMMENT ON COLUMN sync_conflicts.reasoning IS 'Human-readable explanation of how conflict was resolved';
