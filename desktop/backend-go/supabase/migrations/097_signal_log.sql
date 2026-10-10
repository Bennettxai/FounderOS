-- ================================================
-- Migration 097: Signal Log Table
-- Description: Async signal logging for Signal Theory metrics and feedback loop data.
--              Stores metadata about each chat interaction (genre, mode, latency)
--              to feed the homeostatic/autopoietic feedback systems.
-- Author: Roberto
-- Date: 2026-03-03
-- ================================================

CREATE TABLE IF NOT EXISTS signal_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    conversation_id UUID,
    mode TEXT NOT NULL DEFAULT 'ASSIST',
    genre TEXT NOT NULL DEFAULT 'INFORM',
    signal_type TEXT NOT NULL DEFAULT 'chat',
    format TEXT NOT NULL DEFAULT 'MARKDOWN',
    weight FLOAT NOT NULL DEFAULT 0.5,
    message_preview TEXT,
    response_length INT,
    latency_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for per-user time-series queries (feedback loop reads)
CREATE INDEX IF NOT EXISTS idx_signal_log_user_time ON signal_log(user_id, created_at DESC);

-- Index for genre distribution analysis
CREATE INDEX IF NOT EXISTS idx_signal_log_genre ON signal_log(genre, created_at DESC);

COMMENT ON TABLE signal_log IS 'Async signal metadata log for Signal Theory feedback loop (homeostatic, autopoietic, Q-learning)';
COMMENT ON COLUMN signal_log.genre IS 'Detected genre: DIRECT, INFORM, COMMIT, DECIDE, EXPRESS';
COMMENT ON COLUMN signal_log.mode IS 'Focus mode at time of signal: ASSIST, BUILD, RESEARCH, etc.';
COMMENT ON COLUMN signal_log.weight IS 'Signal importance weight (0.0-1.0) for feedback prioritization';
