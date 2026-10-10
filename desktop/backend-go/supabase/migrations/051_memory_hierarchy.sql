-- Migration 030: Memory Hierarchy System
-- Description: Implements isolation between workspace (shared) and user (private) memories

-- Extend workspace_memories table with visibility and access control
ALTER TABLE workspace_memories
ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'workspace', -- workspace, private, shared
ADD COLUMN IF NOT EXISTS owner_user_id TEXT, -- NULL for workspace-level, user_id for private
ADD COLUMN IF NOT EXISTS shared_with TEXT[], -- Array of user_ids for shared private memories
ADD COLUMN IF NOT EXISTS memory_type TEXT NOT NULL DEFAULT 'general', -- general, decision, pattern, context
ADD COLUMN IF NOT EXISTS tags TEXT[], -- Searchable tags
ADD COLUMN IF NOT EXISTS metadata JSONB, -- Flexible metadata storage
ADD COLUMN IF NOT EXISTS access_count INT NOT NULL DEFAULT 0, -- Track usage
ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ;

-- Add constraint for visibility values (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'check_memory_visibility') THEN
        ALTER TABLE workspace_memories
        ADD CONSTRAINT check_memory_visibility
        CHECK (visibility IN ('workspace', 'private', 'shared'));
    END IF;
END $$;

-- Add constraint for memory_type values (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'check_memory_type') THEN
        ALTER TABLE workspace_memories
        ADD CONSTRAINT check_memory_type
        CHECK (memory_type IN ('general', 'decision', 'pattern', 'context', 'learning', 'preference'));
    END IF;
END $$;

-- Private memories must have an owner (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'check_private_owner') THEN
        ALTER TABLE workspace_memories
        ADD CONSTRAINT check_private_owner
        CHECK (
            (visibility = 'workspace' AND owner_user_id IS NULL) OR
            (visibility IN ('private', 'shared') AND owner_user_id IS NOT NULL)
        );
    END IF;
END $$;

-- Create indexes for memory hierarchy queries
CREATE INDEX IF NOT EXISTS idx_workspace_memories_visibility ON workspace_memories(visibility);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_owner ON workspace_memories(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_type ON workspace_memories(memory_type);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace_visibility ON workspace_memories(workspace_id, visibility);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace_owner ON workspace_memories(workspace_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_tags ON workspace_memories USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_metadata ON workspace_memories USING gin(metadata);

-- Drop existing functions from migration 050 before recreating with new signatures
DROP FUNCTION IF EXISTS can_access_memory(TEXT, UUID);
DROP FUNCTION IF EXISTS get_workspace_memories(UUID, TEXT, TEXT, INT);
DROP FUNCTION IF EXISTS get_user_memories(UUID, TEXT, TEXT, INT);
DROP FUNCTION IF EXISTS get_accessible_memories(UUID, TEXT, TEXT, INT);
DROP FUNCTION IF EXISTS share_memory(UUID, TEXT, TEXT[]);
DROP FUNCTION IF EXISTS unshare_memory(UUID, TEXT);
DROP FUNCTION IF EXISTS track_memory_access(UUID);

-- Function to check if user can access a memory
CREATE OR REPLACE FUNCTION can_access_memory(
    p_user_id TEXT,
    p_memory_id UUID
)
RETURNS BOOLEAN AS $$
DECLARE
    v_memory RECORD;
BEGIN
    SELECT visibility, owner_user_id, shared_with, workspace_id
    INTO v_memory
    FROM workspace_memories
    WHERE id = p_memory_id;

    -- Memory doesn't exist
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Workspace-level memory: check if user is workspace member
    IF v_memory.visibility = 'workspace' THEN
        RETURN EXISTS (
            SELECT 1 FROM workspace_members
            WHERE workspace_id = v_memory.workspace_id
            AND user_id = p_user_id
            AND status = 'active'
        );
    END IF;

    -- Private memory: only owner can access
    IF v_memory.visibility = 'private' THEN
        RETURN v_memory.owner_user_id = p_user_id;
    END IF;

    -- Shared memory: owner or shared_with users
    IF v_memory.visibility = 'shared' THEN
        RETURN v_memory.owner_user_id = p_user_id
            OR p_user_id = ANY(v_memory.shared_with);
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- Function to get workspace-level memories (accessible to all workspace members)
CREATE OR REPLACE FUNCTION get_workspace_memories(
    p_workspace_id UUID,
    p_user_id TEXT,
    p_memory_type TEXT DEFAULT NULL,
    p_limit INT DEFAULT 50
)
RETURNS TABLE (
    id UUID,
    content TEXT,
    memory_type TEXT,
    importance_score FLOAT,
    tags TEXT[],
    metadata JSONB,
    access_count INT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    -- Verify user is workspace member
    IF NOT EXISTS (
        SELECT 1 FROM workspace_members
        WHERE workspace_id = p_workspace_id
        AND user_id = p_user_id
        AND status = 'active'
    ) THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT
        wm.id,
        wm.content,
        wm.memory_type,
        wm.importance_score,
        wm.tags,
        wm.metadata,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND wm.visibility = 'workspace'
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC, wm.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's private memories
CREATE OR REPLACE FUNCTION get_user_memories(
    p_workspace_id UUID,
    p_user_id TEXT,
    p_memory_type TEXT DEFAULT NULL,
    p_limit INT DEFAULT 50
)
RETURNS TABLE (
    id UUID,
    content TEXT,
    memory_type TEXT,
    importance_score FLOAT,
    tags TEXT[],
    metadata JSONB,
    visibility TEXT,
    shared_with TEXT[],
    access_count INT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        wm.id,
        wm.content,
        wm.memory_type,
        wm.importance_score,
        wm.tags,
        wm.metadata,
        wm.visibility,
        wm.shared_with,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND (
        -- User's own private memories
        (wm.visibility = 'private' AND wm.owner_user_id = p_user_id)
        OR
        -- Memories shared with this user
        (wm.visibility = 'shared' AND (wm.owner_user_id = p_user_id OR p_user_id = ANY(wm.shared_with)))
    )
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC, wm.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to get all accessible memories for user (workspace + private + shared)
CREATE OR REPLACE FUNCTION get_accessible_memories(
    p_workspace_id UUID,
    p_user_id TEXT,
    p_memory_type TEXT DEFAULT NULL,
    p_limit INT DEFAULT 100
)
RETURNS TABLE (
    id UUID,
    content TEXT,
    memory_type TEXT,
    visibility TEXT,
    importance_score FLOAT,
    tags TEXT[],
    metadata JSONB,
    is_owner BOOLEAN,
    access_count INT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    -- Verify user is workspace member
    IF NOT EXISTS (
        SELECT 1 FROM workspace_members
        WHERE workspace_id = p_workspace_id
        AND user_id = p_user_id
        AND status = 'active'
    ) THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT
        wm.id,
        wm.content,
        wm.memory_type,
        wm.visibility,
        wm.importance_score,
        wm.tags,
        wm.metadata,
        (wm.owner_user_id = p_user_id OR wm.owner_user_id IS NULL) as is_owner,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND (
        -- Workspace-level memories (accessible to all)
        wm.visibility = 'workspace'
        OR
        -- User's private memories
        (wm.visibility = 'private' AND wm.owner_user_id = p_user_id)
        OR
        -- Memories shared with this user
        (wm.visibility = 'shared' AND (wm.owner_user_id = p_user_id OR p_user_id = ANY(wm.shared_with)))
    )
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC, wm.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to share a private memory with specific users
CREATE OR REPLACE FUNCTION share_memory(
    p_memory_id UUID,
    p_owner_id TEXT,
    p_share_with_user_ids TEXT[]
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_visibility TEXT;
    v_current_owner TEXT;
BEGIN
    -- Get current memory details
    SELECT visibility, owner_user_id
    INTO v_current_visibility, v_current_owner
    FROM workspace_memories
    WHERE id = p_memory_id;

    -- Verify ownership
    IF v_current_owner != p_owner_id THEN
        RAISE EXCEPTION 'Only the owner can share this memory';
    END IF;

    -- Can only share private memories
    IF v_current_visibility != 'private' THEN
        RAISE EXCEPTION 'Only private memories can be shared';
    END IF;

    -- Update visibility and shared_with
    UPDATE workspace_memories
    SET
        visibility = 'shared',
        shared_with = p_share_with_user_ids,
        updated_at = NOW()
    WHERE id = p_memory_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to unshare (make private again)
CREATE OR REPLACE FUNCTION unshare_memory(
    p_memory_id UUID,
    p_owner_id TEXT
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE workspace_memories
    SET
        visibility = 'private',
        shared_with = NULL,
        updated_at = NOW()
    WHERE id = p_memory_id
    AND owner_user_id = p_owner_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to track memory access
CREATE OR REPLACE FUNCTION track_memory_access(
    p_memory_id UUID
)
RETURNS VOID AS $$
BEGIN
    UPDATE workspace_memories
    SET
        access_count = access_count + 1,
        last_accessed_at = NOW()
    WHERE id = p_memory_id;
END;
$$ LANGUAGE plpgsql;

-- View for easy querying of memory hierarchy
CREATE OR REPLACE VIEW memory_hierarchy_view AS
SELECT
    wm.id,
    wm.workspace_id,
    wm.content,
    wm.memory_type,
    wm.visibility,
    wm.owner_user_id,
    wm.shared_with,
    wm.importance_score,
    wm.tags,
    wm.metadata,
    wm.access_count,
    wm.last_accessed_at,
    wm.created_at,
    wm.updated_at,
    CASE
        WHEN wm.visibility = 'workspace' THEN 'Workspace (Shared)'
        WHEN wm.visibility = 'private' THEN 'Private'
        WHEN wm.visibility = 'shared' THEN 'Shared with specific users'
    END as visibility_label,
    CASE
        WHEN wm.visibility = 'workspace' THEN 'all'
        WHEN wm.visibility = 'private' THEN 'owner_only'
        WHEN wm.visibility = 'shared' THEN 'owner_and_shared'
    END as access_scope
FROM workspace_memories wm;

COMMENT ON TABLE workspace_memories IS 'Hierarchical memory storage: workspace (shared) and user (private) memories';
COMMENT ON COLUMN workspace_memories.visibility IS 'Memory visibility: workspace (all members), private (owner only), shared (owner + specific users)';
COMMENT ON COLUMN workspace_memories.owner_user_id IS 'Owner of private/shared memories, NULL for workspace-level';
COMMENT ON COLUMN workspace_memories.shared_with IS 'User IDs who have access to shared memories';
COMMENT ON COLUMN workspace_memories.memory_type IS 'Type of memory: general, decision, pattern, context, learning, preference';
COMMENT ON COLUMN workspace_memories.tags IS 'Searchable tags for categorization';
COMMENT ON COLUMN workspace_memories.metadata IS 'Flexible JSONB metadata for custom fields';
COMMENT ON FUNCTION can_access_memory IS 'Check if user can access a specific memory based on visibility rules';
COMMENT ON FUNCTION get_workspace_memories IS 'Get workspace-level memories (accessible to all workspace members)';
COMMENT ON FUNCTION get_user_memories IS 'Get user''s private and shared memories';
COMMENT ON FUNCTION get_accessible_memories IS 'Get all memories accessible to user (workspace + private + shared)';
COMMENT ON FUNCTION share_memory IS 'Share a private memory with specific users';
COMMENT ON FUNCTION unshare_memory IS 'Make a shared memory private again';
COMMENT ON FUNCTION track_memory_access IS 'Increment access counter when memory is retrieved';
