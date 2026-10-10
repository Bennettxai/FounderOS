BEGIN;

-- =============================================================================
-- Migration 100: Rename osa_generated_apps → osa_module_instances
--                        osa_generated_files → osa_module_files
-- =============================================================================

-- 1. Rename the two primary tables
ALTER TABLE osa_generated_apps RENAME TO osa_module_instances;
ALTER TABLE osa_generated_files RENAME TO osa_module_files;

-- 2. Rename app_id → module_instance_id in osa_execution_history
ALTER TABLE osa_execution_history RENAME COLUMN app_id TO module_instance_id;

-- 3. Rename app_id → module_instance_id in osa_build_events
ALTER TABLE osa_build_events RENAME COLUMN app_id TO module_instance_id;

-- 4. Rename app_id → module_instance_id in osa_webhooks
ALTER TABLE osa_webhooks RENAME COLUMN app_id TO module_instance_id;

-- 5. Rename app_id → module_instance_id in osa_workflows
ALTER TABLE osa_workflows RENAME COLUMN app_id TO module_instance_id;

-- 6. Rename app_id → module_instance_id in osa_module_files (was osa_generated_files)
ALTER TABLE osa_module_files RENAME COLUMN app_id TO module_instance_id;

-- 7. Rename app_id → module_instance_id in osa_template_usage_log
ALTER TABLE osa_template_usage_log RENAME COLUMN app_id TO module_instance_id;

-- 8. Rename app_id → module_instance_id in sandbox_events
ALTER TABLE sandbox_events RENAME COLUMN app_id TO module_instance_id;

-- 9. Rename osa_app_id → module_instance_id in user_generated_apps
ALTER TABLE user_generated_apps RENAME COLUMN osa_app_id TO module_instance_id;

COMMIT;
