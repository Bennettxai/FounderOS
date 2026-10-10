-- Migration 095: Context Windows table for per-conversation token budget tracking
-- Used by ContextTrackerService to persist context state across restarts

CREATE TABLE IF NOT EXISTS context_windows (
    conversation_id UUID PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_tokens    INTEGER NOT NULL DEFAULT 0,
    max_tokens      INTEGER NOT NULL DEFAULT 128000,
    reserve_tokens  INTEGER NOT NULL DEFAULT 25600,
    blocks          JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_context_windows_user_id ON context_windows(user_id);
CREATE INDEX IF NOT EXISTS idx_context_windows_last_accessed ON context_windows(last_accessed_at);
