-- Migration 040: Entity Links System
-- Universal relationship system for linking any entities together
-- Supports various link types: related, blocked_by, parent_of, etc.

-- ============================================================================
-- ENTITY LINKS TABLE
-- ============================================================================

-- Link types
DO $$ BEGIN
    CREATE TYPE entity_link_type AS ENUM (
        -- General relationships
        'related',          -- Generic relationship
        'mentions',         -- One entity mentions another

        -- Hierarchy
        'parent_of',        -- Parent-child relationship
        'child_of',         -- Child of parent

        -- Dependencies
        'blocks',           -- Source blocks target
        'blocked_by',       -- Source is blocked by target
        'depends_on',       -- Source depends on target

        -- Duplicates
        'duplicate_of',     -- Source is duplicate of target
        'original_of',      -- Source is original, target is duplicate

        -- Derivation
        'derived_from',     -- Created from another entity
        'spawned',          -- Created another entity

        -- Cross-module links
        'task_for',         -- Task for a project/client
        'project_for',      -- Project for a client
        'note_about',       -- Note about an entity
        'meeting_about',    -- Meeting about an entity

        -- Custom
        'custom'            -- User-defined relationship
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Source entity
    source_type VARCHAR(100) NOT NULL,
    source_id UUID NOT NULL,
    source_name VARCHAR(500),  -- Cached for display

    -- Target entity
    target_type VARCHAR(100) NOT NULL,
    target_id UUID NOT NULL,
    target_name VARCHAR(500),  -- Cached for display

    -- Relationship
    link_type entity_link_type NOT NULL DEFAULT 'related',
    custom_link_type VARCHAR(100),  -- If link_type = 'custom'

    -- Bidirectional?
    is_bidirectional BOOLEAN DEFAULT FALSE,

    -- Metadata
    description TEXT,
    metadata JSONB DEFAULT '{}',

    -- Who created it
    created_by VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Prevent exact duplicates
    UNIQUE(source_type, source_id, target_type, target_id, link_type)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Find links from a source
CREATE INDEX IF NOT EXISTS idx_entity_links_source
    ON entity_links(source_type, source_id);

-- Find links to a target
CREATE INDEX IF NOT EXISTS idx_entity_links_target
    ON entity_links(target_type, target_id);

-- User's links
CREATE INDEX IF NOT EXISTS idx_entity_links_user
    ON entity_links(user_id);

-- By link type
CREATE INDEX IF NOT EXISTS idx_entity_links_type
    ON entity_links(link_type);

-- Bidirectional links (for symmetric lookups)
CREATE INDEX IF NOT EXISTS idx_entity_links_bidirectional
    ON entity_links(is_bidirectional)
    WHERE is_bidirectional = TRUE;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update timestamps
DROP TRIGGER IF EXISTS entity_links_updated_at ON entity_links;
CREATE TRIGGER entity_links_updated_at
    BEFORE UPDATE ON entity_links
    FOR EACH ROW
    EXECUTE FUNCTION update_custom_updated_at();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Create a bidirectional link (creates both directions)
CREATE OR REPLACE FUNCTION create_bidirectional_link(
    p_user_id VARCHAR(255),
    p_source_type VARCHAR(100),
    p_source_id UUID,
    p_source_name VARCHAR(500),
    p_target_type VARCHAR(100),
    p_target_id UUID,
    p_target_name VARCHAR(500),
    p_link_type entity_link_type DEFAULT 'related',
    p_created_by VARCHAR(255) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_id UUID;
BEGIN
    -- Create primary link
    INSERT INTO entity_links (
        user_id, source_type, source_id, source_name,
        target_type, target_id, target_name,
        link_type, is_bidirectional, created_by
    ) VALUES (
        p_user_id, p_source_type, p_source_id, p_source_name,
        p_target_type, p_target_id, p_target_name,
        p_link_type, TRUE, p_created_by
    )
    ON CONFLICT (source_type, source_id, target_type, target_id, link_type)
    DO UPDATE SET updated_at = NOW()
    RETURNING id INTO v_id;

    -- Create reverse link (if different entities)
    IF NOT (p_source_type = p_target_type AND p_source_id = p_target_id) THEN
        INSERT INTO entity_links (
            user_id, source_type, source_id, source_name,
            target_type, target_id, target_name,
            link_type, is_bidirectional, created_by
        ) VALUES (
            p_user_id, p_target_type, p_target_id, p_target_name,
            p_source_type, p_source_id, p_source_name,
            p_link_type, TRUE, p_created_by
        )
        ON CONFLICT (source_type, source_id, target_type, target_id, link_type)
        DO UPDATE SET updated_at = NOW();
    END IF;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- Get all linked entities for a given entity
CREATE OR REPLACE FUNCTION get_linked_entities(
    p_entity_type VARCHAR(100),
    p_entity_id UUID,
    p_link_type entity_link_type DEFAULT NULL
)
RETURNS TABLE (
    link_id UUID,
    linked_type VARCHAR(100),
    linked_id UUID,
    linked_name VARCHAR(500),
    link_type entity_link_type,
    direction VARCHAR(10),
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    -- Links where entity is source
    SELECT
        el.id,
        el.target_type,
        el.target_id,
        el.target_name,
        el.link_type,
        'outgoing'::VARCHAR(10),
        el.created_at
    FROM entity_links el
    WHERE el.source_type = p_entity_type
      AND el.source_id = p_entity_id
      AND (p_link_type IS NULL OR el.link_type = p_link_type)

    UNION ALL

    -- Links where entity is target (for non-bidirectional)
    SELECT
        el.id,
        el.source_type,
        el.source_id,
        el.source_name,
        el.link_type,
        'incoming'::VARCHAR(10),
        el.created_at
    FROM entity_links el
    WHERE el.target_type = p_entity_type
      AND el.target_id = p_entity_id
      AND el.is_bidirectional = FALSE
      AND (p_link_type IS NULL OR el.link_type = p_link_type)

    ORDER BY created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Check if entity has blocking dependencies
CREATE OR REPLACE FUNCTION has_blockers(
    p_entity_type VARCHAR(100),
    p_entity_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM entity_links
        WHERE target_type = p_entity_type
          AND target_id = p_entity_id
          AND link_type = 'blocks'
    );
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE entity_links IS 'Universal relationship system linking any entities';
COMMENT ON COLUMN entity_links.is_bidirectional IS 'If true, link works both directions';
COMMENT ON COLUMN entity_links.custom_link_type IS 'User-defined relationship name when link_type=custom';
COMMENT ON FUNCTION create_bidirectional_link IS 'Creates a two-way relationship between entities';
COMMENT ON FUNCTION get_linked_entities IS 'Returns all entities linked to a given entity';
COMMENT ON FUNCTION has_blockers IS 'Checks if an entity is blocked by other entities';
