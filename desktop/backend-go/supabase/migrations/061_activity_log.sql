-- Migration 037: Activity Log System
-- Universal audit trail for all entity changes across BusinessOS
-- Tracks who did what, when, and to what entity

-- ============================================================================
-- ACTIVITY LOG TABLE
-- ============================================================================

-- Activity types
DO $$ BEGIN
    CREATE TYPE activity_action AS ENUM (
        -- CRUD operations
        'created',
        'updated',
        'deleted',
        'restored',
        'archived',

        -- Status changes
        'status_changed',
        'priority_changed',
        'assigned',
        'unassigned',

        -- Relationships
        'linked',
        'unlinked',
        'moved',

        -- Content
        'commented',
        'mentioned',
        'attached',
        'detached',

        -- Sharing
        'shared',
        'unshared',

        -- Integration events
        'synced',
        'imported',
        'exported',

        -- Custom actions
        'custom'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What entity was affected
    entity_type VARCHAR(100) NOT NULL,  -- 'task', 'project', 'client', 'custom_record', etc.
    entity_id UUID NOT NULL,            -- ID of the affected entity
    entity_name VARCHAR(500),           -- Cached name for display (denormalized)

    -- What happened
    action activity_action NOT NULL,
    action_detail VARCHAR(255),         -- Additional context: field name, etc.

    -- Who did it
    actor_id VARCHAR(255),              -- NULL for system actions
    actor_name VARCHAR(255),            -- Cached name for display

    -- Changes (for updates)
    changes JSONB,
    -- Format: {"field_name": {"old": "...", "new": "..."}}

    -- Related entities (for links, assignments, etc.)
    related_entity_type VARCHAR(100),
    related_entity_id UUID,
    related_entity_name VARCHAR(500),

    -- Context
    metadata JSONB DEFAULT '{}',
    -- ip_address, user_agent, request_id, etc.

    -- When
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- INDEXES FOR COMMON QUERIES
-- ============================================================================

-- User's activity feed
CREATE INDEX IF NOT EXISTS idx_activity_log_user
    ON activity_log(user_id, created_at DESC);

-- Entity history
CREATE INDEX IF NOT EXISTS idx_activity_log_entity
    ON activity_log(entity_type, entity_id, created_at DESC);

-- Actor's actions
CREATE INDEX IF NOT EXISTS idx_activity_log_actor
    ON activity_log(actor_id, created_at DESC)
    WHERE actor_id IS NOT NULL;

-- By action type
CREATE INDEX IF NOT EXISTS idx_activity_log_action
    ON activity_log(user_id, action, created_at DESC);

-- Recent activity (for dashboard)
CREATE INDEX IF NOT EXISTS idx_activity_log_recent
    ON activity_log(created_at DESC);

-- Related entity lookups
CREATE INDEX IF NOT EXISTS idx_activity_log_related
    ON activity_log(related_entity_type, related_entity_id)
    WHERE related_entity_id IS NOT NULL;

-- ============================================================================
-- PARTITIONING PREPARATION (for scale)
-- ============================================================================

-- Note: For high-volume production, consider partitioning by time:
-- CREATE TABLE activity_log_y2026m01 PARTITION OF activity_log
--     FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- ============================================================================
-- RETENTION POLICY FUNCTION
-- ============================================================================

-- Function to clean old activity logs (run periodically)
CREATE OR REPLACE FUNCTION cleanup_old_activity_logs(retention_days INT DEFAULT 365)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM activity_log
    WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- HELPER FUNCTION TO LOG ACTIVITY
-- ============================================================================

CREATE OR REPLACE FUNCTION log_activity(
    p_user_id VARCHAR(255),
    p_entity_type VARCHAR(100),
    p_entity_id UUID,
    p_entity_name VARCHAR(500),
    p_action activity_action,
    p_actor_id VARCHAR(255) DEFAULT NULL,
    p_actor_name VARCHAR(255) DEFAULT NULL,
    p_action_detail VARCHAR(255) DEFAULT NULL,
    p_changes JSONB DEFAULT NULL,
    p_related_entity_type VARCHAR(100) DEFAULT NULL,
    p_related_entity_id UUID DEFAULT NULL,
    p_related_entity_name VARCHAR(500) DEFAULT NULL,
    p_metadata JSONB DEFAULT '{}'
)
RETURNS UUID AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO activity_log (
        user_id, entity_type, entity_id, entity_name,
        action, action_detail, actor_id, actor_name,
        changes, related_entity_type, related_entity_id, related_entity_name,
        metadata
    ) VALUES (
        p_user_id, p_entity_type, p_entity_id, p_entity_name,
        p_action, p_action_detail, p_actor_id, p_actor_name,
        p_changes, p_related_entity_type, p_related_entity_id, p_related_entity_name,
        p_metadata
    )
    RETURNING id INTO v_id;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE activity_log IS 'Universal audit trail for all entity changes';
COMMENT ON COLUMN activity_log.entity_type IS 'Type of entity: task, project, client, custom_record, etc.';
COMMENT ON COLUMN activity_log.changes IS 'JSON object of changed fields with old/new values';
COMMENT ON FUNCTION cleanup_old_activity_logs IS 'Removes activity logs older than retention period';
COMMENT ON FUNCTION log_activity IS 'Helper function to insert activity log entries';
