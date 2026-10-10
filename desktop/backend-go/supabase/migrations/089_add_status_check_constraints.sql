-- Add CHECK Constraints for Status/Health Enums
-- Created: 2026-02-06
-- Description: Adds database-level CHECK constraints to osa_generated_apps
--              to enforce valid values for status, sandbox_status, and health_status
-- IDEMPOTENT: Safe to run multiple times (checks for constraint existence)

-- Status constraint: enforces valid app lifecycle states
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_app_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
      ADD CONSTRAINT check_app_status
      CHECK (status IN ('generating', 'generated', 'deploying', 'deployed', 'running', 'stopped', 'failed', 'archived'));
  END IF;
END$$;

-- Sandbox status constraint: enforces valid sandbox lifecycle states
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_sandbox_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
      ADD CONSTRAINT check_sandbox_status
      CHECK (sandbox_status IN ('none', 'pending', 'deploying', 'running', 'stopped', 'failed', 'removing'));
  END IF;
END$$;

-- Health status constraint: enforces valid health check states
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'check_health_status'
      AND conrelid = 'osa_generated_apps'::regclass
  ) THEN
    ALTER TABLE osa_generated_apps
      ADD CONSTRAINT check_health_status
      CHECK (health_status IN ('unknown', 'healthy', 'unhealthy', 'degraded'));
  END IF;
END$$;
