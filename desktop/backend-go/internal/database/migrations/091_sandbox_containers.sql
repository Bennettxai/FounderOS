-- +migrate Up
-- Migration 091: Add sandbox container support to osa_generated_apps
-- Adds columns for Docker container tracking and creates sandbox_events table

-- =============================================================================
-- PART 1: Add sandbox columns to osa_generated_apps
-- =============================================================================

-- Container identification and status
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS container_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS sandbox_port INTEGER,
ADD COLUMN IF NOT EXISTS sandbox_url TEXT,
ADD COLUMN IF NOT EXISTS sandbox_status VARCHAR(50) DEFAULT 'none',
ADD COLUMN IF NOT EXISTS container_image VARCHAR(255),
ADD COLUMN IF NOT EXISTS app_type VARCHAR(50);

-- Health monitoring columns
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS last_health_check TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS health_status VARCHAR(50) DEFAULT 'unknown';

-- Comments explaining column purposes
COMMENT ON COLUMN osa_generated_apps.container_id IS 'Docker container ID when app is deployed to sandbox';
COMMENT ON COLUMN osa_generated_apps.sandbox_port IS 'Allocated host port for sandbox container (9000-9999 range)';
COMMENT ON COLUMN osa_generated_apps.sandbox_url IS 'Full URL to access the sandbox app (e.g., http://localhost:9001)';
COMMENT ON COLUMN osa_generated_apps.sandbox_status IS 'Container status: none, pending, deploying, running, stopped, failed, removing';
COMMENT ON COLUMN osa_generated_apps.container_image IS 'Docker image used (e.g., node:20-alpine, python:3.11-alpine)';
COMMENT ON COLUMN osa_generated_apps.app_type IS 'Detected app type: nodejs, python, go, static';
COMMENT ON COLUMN osa_generated_apps.last_health_check IS 'Timestamp of last health check';
COMMENT ON COLUMN osa_generated_apps.health_status IS 'Health status: unknown, healthy, unhealthy, degraded';

-- =============================================================================
-- PART 2: Create indexes for sandbox queries
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_osa_apps_container_id
ON osa_generated_apps(container_id)
WHERE container_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_osa_apps_sandbox_status
ON osa_generated_apps(sandbox_status)
WHERE sandbox_status != 'none';

CREATE INDEX IF NOT EXISTS idx_osa_apps_sandbox_port
ON osa_generated_apps(sandbox_port)
WHERE sandbox_port IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_osa_apps_health_status
ON osa_generated_apps(health_status)
WHERE health_status != 'unknown';

-- =============================================================================
-- PART 3: Create sandbox_events table for lifecycle logging
-- =============================================================================

CREATE TABLE IF NOT EXISTS sandbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES osa_generated_apps(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    container_id VARCHAR(255),
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for querying events by app
CREATE INDEX IF NOT EXISTS idx_sandbox_events_app_id
ON sandbox_events(app_id);

-- Index for querying events by type
CREATE INDEX IF NOT EXISTS idx_sandbox_events_type
ON sandbox_events(event_type);

-- Index for chronological queries
CREATE INDEX IF NOT EXISTS idx_sandbox_events_created
ON sandbox_events(created_at DESC);

-- Comments
COMMENT ON TABLE sandbox_events IS 'Lifecycle events for sandbox containers (deploy, start, stop, health checks, errors)';
COMMENT ON COLUMN sandbox_events.event_type IS 'Event type: deploying, deployed, started, stopped, healthy, unhealthy, failed, removed, restarted';
COMMENT ON COLUMN sandbox_events.details IS 'Additional event data as JSON (error messages, metrics, etc.)';

-- +migrate Down
-- Rollback: Remove sandbox columns and events table

-- Drop indexes first
DROP INDEX IF EXISTS idx_sandbox_events_created;
DROP INDEX IF EXISTS idx_sandbox_events_type;
DROP INDEX IF EXISTS idx_sandbox_events_app_id;
DROP INDEX IF EXISTS idx_osa_apps_health_status;
DROP INDEX IF EXISTS idx_osa_apps_sandbox_port;
DROP INDEX IF EXISTS idx_osa_apps_sandbox_status;
DROP INDEX IF EXISTS idx_osa_apps_container_id;

-- Drop events table
DROP TABLE IF EXISTS sandbox_events;

-- Remove columns from osa_generated_apps (reverse order)
ALTER TABLE osa_generated_apps
DROP COLUMN IF EXISTS health_status,
DROP COLUMN IF EXISTS last_health_check,
DROP COLUMN IF EXISTS app_type,
DROP COLUMN IF EXISTS container_image,
DROP COLUMN IF EXISTS sandbox_status,
DROP COLUMN IF EXISTS sandbox_url,
DROP COLUMN IF EXISTS sandbox_port,
DROP COLUMN IF EXISTS container_id;
