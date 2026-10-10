-- Migration 030: Memory Hierarchy System (v2 - adjusted for existing schema)
-- Description: Implements isolation between workspace (shared) and user (private) memories

-- Add missing columns to workspace_memories (most already exist)
ALTER TABLE workspace_memories
ADD COLUMN IF NOT EXISTS owner_user_id TEXT, -- NULL for workspace-level, user_id for private
ADD COLUMN IF NOT EXISTS shared_with TEXT[]; -- Array of user_ids for shared private memories

-- Update visibility constraint (if not already set)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'check_memory_visibility_v2'
    ) THEN
        ALTER TABLE workspace_memories
        ADD CONSTRAINT check_memory_visibility_v2
        CHECK (visibility IN ('workspace', 'private', 'shared'));
    END IF;
END $$;

-- Private memories must have an owner
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'check_private_owner'
    ) THEN
        ALTER TABLE workspace_memories
        ADD CONSTRAINT check_private_owner
        CHECK (
            (visibility = 'workspace' AND owner_user_id IS NULL) OR
            (visibility IN ('private', 'shared') AND owner_user_id IS NOT NULL)
        );
    END IF;
END $$;

-- Create additional indexes for memory hierarchy queries
CREATE INDEX IF NOT EXISTS idx_workspace_memories_owner ON workspace_memories(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace_visibility ON workspace_memories(workspace_id, visibility);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace_owner ON workspace_memories(workspace_id, owner_user_id);

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

    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Workspace-level memory: check if user is workspace member
    IF v_memory.visibility = 'workspace' OR v_memory.visibility IS NULL THEN
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
            OR p_user_id = ANY(COALESCE(v_memory.shared_with, ARRAY[]::TEXT[]));
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
    title TEXT,
    content TEXT,
    memory_type TEXT,
    importance NUMERIC,
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
        wm.title,
        wm.content,
        wm.memory_type,
        wm.importance_score as importance,
        wm.tags,
        wm.metadata,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND (wm.visibility = 'workspace' OR wm.visibility IS NULL)
    AND wm.is_active = true
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC NULLS LAST, wm.created_at DESC
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
    title TEXT,
    content TEXT,
    memory_type TEXT,
    importance NUMERIC,
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
        wm.title,
        wm.content,
        wm.memory_type,
        wm.importance_score as importance,
        wm.tags,
        wm.metadata,
        wm.visibility,
        wm.shared_with,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND wm.is_active = true
    AND (
        (wm.visibility = 'private' AND wm.owner_user_id = p_user_id)
        OR
        (wm.visibility = 'shared' AND (wm.owner_user_id = p_user_id OR p_user_id = ANY(COALESCE(wm.shared_with, ARRAY[]::TEXT[]))))
    )
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC NULLS LAST, wm.created_at DESC
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
    title TEXT,
    content TEXT,
    memory_type TEXT,
    visibility TEXT,
    importance NUMERIC,
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
        wm.title,
        wm.content,
        wm.memory_type,
        wm.visibility,
        wm.importance_score as importance,
        wm.tags,
        wm.metadata,
        (wm.owner_user_id = p_user_id OR wm.owner_user_id IS NULL) as is_owner,
        wm.access_count,
        wm.created_at
    FROM workspace_memories wm
    WHERE wm.workspace_id = p_workspace_id
    AND wm.is_active = true
    AND (
        wm.visibility = 'workspace' OR wm.visibility IS NULL
        OR
        (wm.visibility = 'private' AND wm.owner_user_id = p_user_id)
        OR
        (wm.visibility = 'shared' AND (wm.owner_user_id = p_user_id OR p_user_id = ANY(COALESCE(wm.shared_with, ARRAY[]::TEXT[]))))
    )
    AND (p_memory_type IS NULL OR wm.memory_type = p_memory_type)
    ORDER BY wm.importance_score DESC NULLS LAST, wm.created_at DESC
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
    SELECT visibility, owner_user_id
    INTO v_current_visibility, v_current_owner
    FROM workspace_memories
    WHERE id = p_memory_id;

    IF v_current_owner != p_owner_id THEN
        RAISE EXCEPTION 'Only the owner can share this memory';
    END IF;

    IF v_current_visibility != 'private' THEN
        RAISE EXCEPTION 'Only private memories can be shared';
    END IF;

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
        access_count = COALESCE(access_count, 0) + 1,
        last_accessed_at = NOW()
    WHERE id = p_memory_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON COLUMN workspace_memories.visibility IS 'Memory visibility: workspace (all members), private (owner only), shared (owner + specific users)';
COMMENT ON COLUMN workspace_memories.owner_user_id IS 'Owner of private/shared memories, NULL for workspace-level';
COMMENT ON COLUMN workspace_memories.shared_with IS 'User IDs who have access to shared memories';
COMMENT ON FUNCTION can_access_memory IS 'Check if user can access a specific memory based on visibility rules';
COMMENT ON FUNCTION get_workspace_memories IS 'Get workspace-level memories (accessible to all workspace members)';
COMMENT ON FUNCTION get_user_memories IS 'Get user''s private and shared memories';
COMMENT ON FUNCTION get_accessible_memories IS 'Get all memories accessible to user (workspace + private + shared)';
COMMENT ON FUNCTION share_memory IS 'Share a private memory with specific users';
COMMENT ON FUNCTION unshare_memory IS 'Make a shared memory private again';
COMMENT ON FUNCTION track_memory_access IS 'Increment access counter when memory is retrieved';
