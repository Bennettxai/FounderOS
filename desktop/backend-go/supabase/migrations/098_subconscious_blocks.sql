-- ================================================
-- Migration 098: Subconscious Memory Blocks
-- Description: Persistent memory blocks for the Subconscious Observer.
--              Stores cross-session observations (preferences, patterns,
--              guidance, self-improvement notes) that get injected into
--              agent prompts. Modeled after Letta's memory block pattern.
-- Author: Roberto
-- Date: 2026-03-05
-- ================================================

CREATE TABLE IF NOT EXISTS subconscious_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    block_type TEXT NOT NULL,       -- core_directives, guidance, session_patterns, user_preferences, tool_guidelines, self_improvement, project_context, pending_items
    content TEXT NOT NULL DEFAULT '',
    weight FLOAT NOT NULL DEFAULT 0.5,   -- importance weight [0.0, 1.0] for injection ordering
    version INT NOT NULL DEFAULT 1,      -- monotonic version for optimistic concurrency
    expires_at TIMESTAMPTZ,              -- optional TTL (NULL = permanent)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Each user has at most one block per type
CREATE UNIQUE INDEX IF NOT EXISTS idx_subconscious_blocks_user_type
    ON subconscious_blocks(user_id, block_type);

-- Injection reads: fetch active blocks ordered by weight
CREATE INDEX IF NOT EXISTS idx_subconscious_blocks_weight
    ON subconscious_blocks(user_id, weight DESC)
    WHERE expires_at IS NULL OR expires_at > NOW();

-- Expiry cleanup
CREATE INDEX IF NOT EXISTS idx_subconscious_blocks_expiry
    ON subconscious_blocks(expires_at)
    WHERE expires_at IS NOT NULL;

COMMENT ON TABLE subconscious_blocks IS 'Persistent memory blocks for Subconscious Observer — cross-session pattern accumulation and prompt injection';
COMMENT ON COLUMN subconscious_blocks.block_type IS 'Block category: core_directives, guidance, session_patterns, user_preferences, tool_guidelines, self_improvement, project_context, pending_items';
COMMENT ON COLUMN subconscious_blocks.weight IS 'Importance weight [0.0, 1.0] — higher weight blocks are injected first';
COMMENT ON COLUMN subconscious_blocks.version IS 'Monotonic version for optimistic concurrency control';
