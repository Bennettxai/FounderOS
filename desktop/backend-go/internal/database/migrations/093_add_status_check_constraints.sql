-- +migrate Up
-- Migration 093: Add CHECK constraints for status enums in osa_generated_apps
-- Ensures data integrity by enforcing valid status values at database level
-- IDEMPOTENT: Safe to run multiple times (checks for constraint existence)

-- =============================================================================
-- PART 1: Add CHECK constraint for app status
-- =============================================================================

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_app_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
    ADD CONSTRAINT check_app_status
    CHECK (status IN (
        'generating',  -- App generation in progress
        'generated',   -- App generation completed (default)
        'deploying',   -- Deployment process started
        'deployed',    -- Deployment completed
        'running',     -- Application is running
        'stopped',     -- Application has been stopped
        'failed',      -- Generation or deployment failed
        'archived'     -- App archived/no longer active
    ));
  END IF;
END$$;

COMMENT ON CONSTRAINT check_app_status ON osa_generated_apps IS
'Enforces valid app status values. See migration 093 for full enum list.';

-- =============================================================================
-- PART 2: Add CHECK constraint for sandbox status
-- =============================================================================

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_sandbox_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
    ADD CONSTRAINT check_sandbox_status
    CHECK (sandbox_status IN (
        'none',        -- No sandbox deployed (default)
        'pending',     -- Sandbox deployment queued
        'deploying',   -- Deploying to sandbox container
        'running',     -- Container is running
        'stopped',     -- Container has been stopped
        'failed',      -- Container deployment/startup failed
        'removing'     -- Container is being removed/cleaned up
    ));
  END IF;
END$$;

COMMENT ON CONSTRAINT check_sandbox_status ON osa_generated_apps IS
'Enforces valid sandbox container status values. Default is "none" for no sandbox.';

-- =============================================================================
-- PART 3: Add CHECK constraint for health status
-- =============================================================================

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_health_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
    ADD CONSTRAINT check_health_status
    CHECK (health_status IN (
        'unknown',     -- No health check performed (default)
        'healthy',     -- Container passing health checks
        'unhealthy',   -- Container failed health checks
        'degraded'     -- Container partially functional
    ));
  END IF;
END$$;

COMMENT ON CONSTRAINT check_health_status ON osa_generated_apps IS
'Enforces valid health check status values. Default is "unknown" before first check.';

-- +migrate Down
-- Rollback: Remove all CHECK constraints

ALTER TABLE osa_generated_apps
DROP CONSTRAINT IF EXISTS check_health_status;

ALTER TABLE osa_generated_apps
DROP CONSTRAINT IF EXISTS check_sandbox_status;

ALTER TABLE osa_generated_apps
DROP CONSTRAINT IF EXISTS check_app_status;
