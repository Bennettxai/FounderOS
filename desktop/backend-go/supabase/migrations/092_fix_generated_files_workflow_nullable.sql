-- Migration: 092_fix_generated_files_workflow_nullable.sql
-- Description: Fix two FK violations that prevent generated files from being persisted:
--
--   BUG 1 — osa_generated_files.workflow_id NOT NULL
--     saveFileToDatabase() passes workflow_id=NULL (no workflow row exists in the queue path)
--     causing every file INSERT to fail the NOT NULL constraint.
--     The create_initial_file_version trigger then propagates the NULL to osa_file_versions
--     which also has workflow_id NOT NULL, creating a second failure point.
--
--   BUG 2 — osa_generated_apps.workspace_id REFERENCES osa_workspaces(id)
--     app_generation_queue.workspace_id REFERENCES workspaces(id) (the main workspace table).
--     The orchestrator uses the queue's workspace_id to insert into osa_generated_apps, but
--     osa_generated_apps.workspace_id references osa_workspaces(id) — a completely different
--     table. The UUID from workspaces will never match a row in osa_workspaces, causing an FK
--     violation on the osa_generated_apps INSERT that must happen before any file saves.
--
-- Fix summary:
--   1. Make osa_generated_files.workflow_id nullable.
--   2. Make osa_file_versions.workflow_id nullable (trigger carry-over).
--   3. Fix UNIQUE constraint on osa_generated_files (workflow_id-based is broken for NULLs;
--      replace with app_id-based partial index for the queue-driven path).
--   4. Change osa_generated_apps.workspace_id FK from osa_workspaces(id) to workspaces(id),
--      matching the FK target used by app_generation_queue.
--
-- Created: 2026-02-24

-- =========================================================================
-- BUG 1 FIX: osa_generated_files — make workflow_id nullable
-- =========================================================================

ALTER TABLE osa_generated_files ALTER COLUMN workflow_id DROP NOT NULL;

-- Drop old table-level unique constraint (semantically broken when workflow_id is NULL,
-- because NULL != NULL in SQL so no uniqueness is enforced for queue-driven files).
ALTER TABLE osa_generated_files DROP CONSTRAINT IF EXISTS osa_files_workflow_path_unique;

-- Correct uniqueness constraint for queue-driven files: deduplicate by (app_id, file_path).
-- Partial index so it only fires when app_id is set (queue path).
CREATE UNIQUE INDEX IF NOT EXISTS osa_files_app_path_unique
    ON osa_generated_files (app_id, file_path)
    WHERE app_id IS NOT NULL;

-- Restore workflow-scoped uniqueness for the legacy OSA-5 workflow path.
-- Partial index so it only fires when workflow_id is set.
CREATE UNIQUE INDEX IF NOT EXISTS osa_files_workflow_path_unique
    ON osa_generated_files (workflow_id, file_path)
    WHERE workflow_id IS NOT NULL;

-- =========================================================================
-- BUG 1 FIX (continued): osa_file_versions — make workflow_id nullable
-- The create_initial_file_version trigger propagates NEW.workflow_id into
-- osa_file_versions on every osa_generated_files INSERT. When workflow_id
-- is NULL the trigger INSERT would also fail without this change.
-- =========================================================================

ALTER TABLE osa_file_versions ALTER COLUMN workflow_id DROP NOT NULL;

-- =========================================================================
-- BUG 2 FIX: osa_generated_apps — re-wire workspace_id FK to workspaces(id)
--
-- The app_generation_queue.workspace_id references workspaces(id).
-- The orchestrator uses that same UUID when inserting into osa_generated_apps.
-- But osa_generated_apps.workspace_id previously referenced osa_workspaces(id),
-- a completely separate table. The UUIDs will never match across tables.
--
-- Fix: drop the old FK and add a new one pointing at workspaces(id).
-- =========================================================================

-- Drop the old FK constraint (name from 068_osa_integration.sql)
ALTER TABLE osa_generated_apps
    DROP CONSTRAINT IF EXISTS osa_generated_apps_workspace_id_fkey;

-- Also try the common PostgreSQL auto-generated constraint name pattern
ALTER TABLE osa_generated_apps
    DROP CONSTRAINT IF EXISTS osa_generated_apps_workspace_id_fkey1;

-- Add the corrected FK pointing at the main workspaces table
ALTER TABLE osa_generated_apps
    ADD CONSTRAINT osa_generated_apps_workspace_id_fkey
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces(id)
    ON DELETE CASCADE;

-- =========================================================================
-- Documentation
-- =========================================================================

COMMENT ON COLUMN osa_generated_files.workflow_id IS
    'Optional reference to osa_workflows. NULL when file is saved from the app generation queue '
    '(which does not create workflow rows). Set when file is saved through the full OSA-5 workflow path.';

COMMENT ON COLUMN osa_file_versions.workflow_id IS
    'Optional reference to osa_workflows. Mirrors osa_generated_files.workflow_id. NULL when '
    'the parent file was created via the queue-driven path (no workflow row exists).';

COMMENT ON COLUMN osa_generated_apps.workspace_id IS
    'References workspaces(id) — the main workspace table used by app_generation_queue. '
    'Changed from osa_workspaces(id) in migration 092 to align with queue FK target.';
