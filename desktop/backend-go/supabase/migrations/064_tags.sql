-- Migration 039: Universal Tags System
-- Tagging system that works across all entities in BusinessOS
-- Supports hierarchical tags, colors, and tag groups

-- ============================================================================
-- TAGS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Tag definition
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,  -- URL-safe version
    description TEXT,

    -- Appearance
    color VARCHAR(50),
    icon VARCHAR(50),

    -- Hierarchy (optional)
    parent_id UUID REFERENCES tags(id) ON DELETE SET NULL,

    -- Grouping (optional)
    group_name VARCHAR(100),  -- e.g., "Priority", "Status", "Category"

    -- Scope: which entity types can use this tag
    allowed_entity_types TEXT[] DEFAULT '{}',  -- Empty = all types
    -- Example: ARRAY['task', 'project'] means only tasks and projects

    -- Usage tracking
    usage_count INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, slug)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- User's tags
CREATE INDEX IF NOT EXISTS idx_tags_user
    ON tags(user_id);

-- By name for autocomplete
CREATE INDEX IF NOT EXISTS idx_tags_name
    ON tags(user_id, name);

-- By group
CREATE INDEX IF NOT EXISTS idx_tags_group
    ON tags(user_id, group_name)
    WHERE group_name IS NOT NULL;

-- Hierarchy
CREATE INDEX IF NOT EXISTS idx_tags_parent
    ON tags(parent_id)
    WHERE parent_id IS NOT NULL;

-- Most used tags
CREATE INDEX IF NOT EXISTS idx_tags_usage
    ON tags(user_id, usage_count DESC);

-- ============================================================================
-- TAG ASSIGNMENTS (Junction table)
-- ============================================================================

CREATE TABLE IF NOT EXISTS tag_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,

    -- What entity is tagged
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,

    -- Who assigned it
    assigned_by VARCHAR(255),

    -- When
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Prevent duplicate assignments
    UNIQUE(tag_id, entity_type, entity_id)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Find tags for an entity
CREATE INDEX IF NOT EXISTS idx_tag_assignments_entity
    ON tag_assignments(entity_type, entity_id);

-- Find entities with a specific tag
CREATE INDEX IF NOT EXISTS idx_tag_assignments_tag
    ON tag_assignments(tag_id);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update tag usage count
CREATE OR REPLACE FUNCTION update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET usage_count = usage_count - 1 WHERE id = OLD.tag_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_tag_usage ON tag_assignments;
CREATE TRIGGER update_tag_usage
    AFTER INSERT OR DELETE ON tag_assignments
    FOR EACH ROW
    EXECUTE FUNCTION update_tag_usage_count();

-- Auto-update timestamps
DROP TRIGGER IF EXISTS tags_updated_at ON tags;
CREATE TRIGGER tags_updated_at
    BEFORE UPDATE ON tags
    FOR EACH ROW
    EXECUTE FUNCTION update_custom_updated_at();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Generate slug from name
CREATE OR REPLACE FUNCTION generate_tag_slug(p_name VARCHAR(100))
RETURNS VARCHAR(100) AS $$
BEGIN
    RETURN LOWER(REGEXP_REPLACE(
        REGEXP_REPLACE(p_name, '[^a-zA-Z0-9\s-]', '', 'g'),
        '[\s]+', '-', 'g'
    ));
END;
$$ LANGUAGE plpgsql;

-- Get or create a tag
CREATE OR REPLACE FUNCTION get_or_create_tag(
    p_user_id VARCHAR(255),
    p_name VARCHAR(100),
    p_color VARCHAR(50) DEFAULT NULL,
    p_group_name VARCHAR(100) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_slug VARCHAR(100);
    v_id UUID;
BEGIN
    v_slug := generate_tag_slug(p_name);

    -- Try to find existing
    SELECT id INTO v_id
    FROM tags
    WHERE user_id = p_user_id AND slug = v_slug;

    IF v_id IS NOT NULL THEN
        RETURN v_id;
    END IF;

    -- Create new
    INSERT INTO tags (user_id, name, slug, color, group_name)
    VALUES (p_user_id, p_name, v_slug, p_color, p_group_name)
    RETURNING id INTO v_id;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- Get all tags for an entity
CREATE OR REPLACE FUNCTION get_entity_tags(
    p_entity_type VARCHAR(100),
    p_entity_id UUID
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(100),
    color VARCHAR(50),
    group_name VARCHAR(100)
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.name, t.color, t.group_name
    FROM tags t
    JOIN tag_assignments ta ON t.id = ta.tag_id
    WHERE ta.entity_type = p_entity_type
      AND ta.entity_id = p_entity_id
    ORDER BY t.group_name NULLS LAST, t.name;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE tags IS 'Universal tags that can be applied to any entity';
COMMENT ON TABLE tag_assignments IS 'Junction table linking tags to entities';
COMMENT ON COLUMN tags.slug IS 'URL-safe lowercase version of name';
COMMENT ON COLUMN tags.allowed_entity_types IS 'Restrict tag to specific entity types';
COMMENT ON COLUMN tags.usage_count IS 'Auto-updated count of assignments';
