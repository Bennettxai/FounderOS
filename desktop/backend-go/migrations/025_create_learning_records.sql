-- Migration: 025_create_learning_records.sql
-- Description: Create learning_records table for OSA agent learning capture
-- Date: 2026-01-30

-- +goose Up
CREATE TABLE IF NOT EXISTS learning_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    agent_type VARCHAR(50) NOT NULL,
    task_id VARCHAR(100) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    duration_ms BIGINT NOT NULL,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    patterns JSONB NOT NULL DEFAULT '[]'::jsonb,
    classification JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_learning_records_agent_type ON learning_records(agent_type);
CREATE INDEX IF NOT EXISTS idx_learning_records_timestamp ON learning_records(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_learning_records_success ON learning_records(success);
CREATE INDEX IF NOT EXISTS idx_learning_records_task_id ON learning_records(task_id);

-- GIN index for JSONB pattern searching
CREATE INDEX IF NOT EXISTS idx_learning_records_patterns ON learning_records USING GIN(patterns);
CREATE INDEX IF NOT EXISTS idx_learning_records_classification ON learning_records USING GIN(classification);

-- Composite index for pattern retrieval queries
CREATE INDEX IF NOT EXISTS idx_learning_records_agent_success ON learning_records(agent_type, success, timestamp DESC);

-- Comments for documentation
COMMENT ON TABLE learning_records IS 'OSA agent learning capture - stores patterns and metrics from agent executions';
COMMENT ON COLUMN learning_records.agent_type IS 'Type of agent (frontend, backend, database, test, orchestrator)';
COMMENT ON COLUMN learning_records.task_id IS 'Unique identifier for the task execution';
COMMENT ON COLUMN learning_records.success IS 'Whether the agent execution succeeded';
COMMENT ON COLUMN learning_records.duration_ms IS 'Execution duration in milliseconds';
COMMENT ON COLUMN learning_records.tokens_used IS 'Number of tokens consumed by Claude API';
COMMENT ON COLUMN learning_records.patterns IS 'JSONB array of extracted patterns from successful operations';
COMMENT ON COLUMN learning_records.classification IS 'JSONB object with domain, complexity, and value classifications';
COMMENT ON COLUMN learning_records.metadata IS 'Additional context data (workspace, priority, files created, etc)';

-- +goose Down
DROP INDEX IF EXISTS idx_learning_records_agent_success;
DROP INDEX IF EXISTS idx_learning_records_classification;
DROP INDEX IF EXISTS idx_learning_records_patterns;
DROP INDEX IF EXISTS idx_learning_records_task_id;
DROP INDEX IF EXISTS idx_learning_records_success;
DROP INDEX IF EXISTS idx_learning_records_timestamp;
DROP INDEX IF EXISTS idx_learning_records_agent_type;
DROP TABLE IF EXISTS learning_records;
