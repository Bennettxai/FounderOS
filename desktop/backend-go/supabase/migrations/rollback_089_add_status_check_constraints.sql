-- Rollback: Remove CHECK constraints from osa_generated_apps
ALTER TABLE osa_generated_apps DROP CONSTRAINT IF EXISTS check_app_status;
ALTER TABLE osa_generated_apps DROP CONSTRAINT IF EXISTS check_sandbox_status;
ALTER TABLE osa_generated_apps DROP CONSTRAINT IF EXISTS check_health_status;
