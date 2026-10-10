-- Migration 096: Mode transition tracking for OSA orchestrator
-- Tracks mode changes per conversation for pattern detection and session health

CREATE TABLE IF NOT EXISTS mode_transitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    from_mode       TEXT NOT NULL DEFAULT '',
    to_mode         TEXT NOT NULL,
    confidence      DOUBLE PRECISION NOT NULL DEFAULT 0,
    duration_ms     BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mode_transitions_conversation ON mode_transitions(conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mode_transitions_recent ON mode_transitions(created_at DESC);
