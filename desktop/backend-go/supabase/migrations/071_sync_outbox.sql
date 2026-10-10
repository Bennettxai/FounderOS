-- Migration: 043_sync_outbox.sql
-- Description: Transactional outbox pattern for sync events
-- Created: 2026-01-09
-- Phase: 3 (Data Sync)

-- =============================================================================
-- SYNC OUTBOX TABLE
-- Transactional outbox for reliable event publishing
-- =============================================================================
CREATE TABLE IF NOT EXISTS sync_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Aggregate identification
    aggregate_type VARCHAR(50) NOT NULL, -- 'user', 'workspace', 'app', 'project', 'task'
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL, -- 'created', 'updated', 'deleted', 'restored'

    -- Event data
    payload JSONB NOT NULL,
    vector_clock JSONB NOT NULL, -- Vector clock for conflict detection

    -- Processing status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,

    -- Error tracking
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    scheduled_for TIMESTAMPTZ -- For retry backoff scheduling
);

-- Index for processor queries (FOR UPDATE SKIP LOCKED)
-- Note: Cannot use NOW() in WHERE clause as it's not IMMUTABLE
CREATE INDEX IF NOT EXISTS idx_sync_outbox_pending ON sync_outbox (created_at)
WHERE status = 'pending';

-- Index for status tracking
CREATE INDEX IF NOT EXISTS idx_sync_outbox_status ON sync_outbox(status);

-- Index for aggregate lookup
CREATE INDEX IF NOT EXISTS idx_sync_outbox_aggregate ON sync_outbox(aggregate_type, aggregate_id);

-- Index for scheduled retry
CREATE INDEX IF NOT EXISTS idx_sync_outbox_scheduled ON sync_outbox(scheduled_for)
WHERE scheduled_for IS NOT NULL AND status = 'failed';

-- =============================================================================
-- SYNC DEAD LETTER QUEUE (DLQ)
-- Messages that exceeded max retries
-- =============================================================================
CREATE TABLE IF NOT EXISTS sync_dlq (
    id UUID PRIMARY KEY,

    -- Original outbox data
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    vector_clock JSONB NOT NULL,

    -- Failure information
    attempts INT NOT NULL,
    last_error TEXT NOT NULL,
    failure_reason VARCHAR(255),

    -- Timestamps
    original_created_at TIMESTAMPTZ NOT NULL,
    moved_to_dlq_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Resolution tracking
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_dlq_aggregate ON sync_dlq(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_sync_dlq_resolved ON sync_dlq(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_dlq_moved_at ON sync_dlq(moved_to_dlq_at DESC);

-- =============================================================================
-- UPDATE TRIGGER
-- Auto-update updated_at timestamp
-- =============================================================================
CREATE OR REPLACE FUNCTION update_sync_outbox_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_outbox_updated_at
    BEFORE UPDATE ON sync_outbox
    FOR EACH ROW
    EXECUTE FUNCTION update_sync_outbox_updated_at();

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================
COMMENT ON TABLE sync_outbox IS 'Transactional outbox for reliable event publishing with retry logic';
COMMENT ON TABLE sync_dlq IS 'Dead letter queue for events that exceeded max retry attempts';

COMMENT ON COLUMN sync_outbox.status IS 'pending: not yet processed, processing: currently being processed, completed: successfully published, failed: retry scheduled';
COMMENT ON COLUMN sync_outbox.vector_clock IS 'Logical timestamp for distributed conflict detection (map[string]int)';
COMMENT ON COLUMN sync_outbox.scheduled_for IS 'Timestamp for next retry attempt (NULL for immediate processing)';
COMMENT ON COLUMN sync_outbox.max_attempts IS 'Maximum number of processing attempts before moving to DLQ';

-- =============================================================================
-- CLEANUP FUNCTION (Optional - run periodically)
-- Removes old completed events to prevent unbounded growth
-- =============================================================================
CREATE OR REPLACE FUNCTION cleanup_old_sync_outbox()
RETURNS void AS $$
BEGIN
    -- Delete completed events older than 7 days
    DELETE FROM sync_outbox
    WHERE status = 'completed'
    AND processed_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_old_sync_outbox IS 'Cleanup function to remove old completed events (run daily via cron job)';
