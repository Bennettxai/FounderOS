-- Migration: 078_osa_app_registry.sql
-- Description: Flexible OSA App Registry with JSONB-based extensibility
-- Created: 2026-01-25
-- Purpose: Replace rigid schema with flexible JSONB columns for future-proof app registry
-- Design Principles:
--   - Use JSONB for all variable/extensible data (not hard-coded columns)
--   - Template types stored as TEXT (not ENUM) to allow new types without migration
--   - Settings/config in JSONB (not fixed columns)
--   - Future-proof for new app types/metadata without schema changes

-- =============================================================================
-- OSA_APPS - Main App Registry (Flexible Design)
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_apps (
    -- Core Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Template Type (TEXT, not ENUM - add new types without migration)
    -- Examples: 'business_os', 'agency_os', 'content_os', 'sales_os', 'custom', etc.
    template_type TEXT NOT NULL DEFAULT 'custom',

    -- Status (TEXT, not ENUM)
    -- Values: 'generating', 'active', 'failed', 'archived', 'paused'
    status TEXT NOT NULL DEFAULT 'generating',

    -- JSONB: Generation Context (FLEXIBLE - stores prompt, preferences, any metadata)
    -- Schema: { "prompt": "...", "user_preferences": {...}, "ai_model": "...", "generation_params": {...} }
    generation_context JSONB DEFAULT '{}',

    -- JSONB: Deployment Configuration (FLEXIBLE - stores repo, URLs, any deploy settings)
    -- Schema: { "repo_url": "...", "deployment_url": "...", "commit_sha": "...", "branch": "...", "deploy_platform": "..." }
    deployment_config JSONB DEFAULT '{}',

    -- JSONB: App Metadata (FLEXIBLE - stores any app-specific data)
    -- Schema: { "files_created": 123, "tests_passed": true, "build_status": "...", "tech_stack": [...], "features": [...] }
    app_metadata JSONB DEFAULT '{}',

    -- Ownership & Timestamps
    created_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint
    CONSTRAINT osa_apps_name_workspace_unique UNIQUE(name, workspace_id)
);

-- Indexes for performance
CREATE INDEX idx_osa_apps_workspace ON osa_apps(workspace_id);
CREATE INDEX idx_osa_apps_created_by ON osa_apps(created_by);
CREATE INDEX idx_osa_apps_status ON osa_apps(status);
CREATE INDEX idx_osa_apps_template_type ON osa_apps(template_type);

-- GIN indexes for JSONB queries
CREATE INDEX idx_osa_apps_generation_context ON osa_apps USING GIN(generation_context);
CREATE INDEX idx_osa_apps_deployment_config ON osa_apps USING GIN(deployment_config);
CREATE INDEX idx_osa_apps_app_metadata ON osa_apps USING GIN(app_metadata);

-- =============================================================================
-- OSA_WORKSPACES - Enhanced Workspace Tracking (Flexible Design)
-- =============================================================================
-- NOTE: Table already exists from 065_osa_integration.sql
-- This migration ONLY adds missing flexibility if needed

-- Add index on settings JSONB if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE indexname = 'idx_osa_workspaces_settings'
    ) THEN
        CREATE INDEX idx_osa_workspaces_settings ON osa_workspaces USING GIN(settings);
    END IF;
END $$;

-- Add description column if missing
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'osa_workspaces' AND column_name = 'description'
    ) THEN
        ALTER TABLE osa_workspaces ADD COLUMN description TEXT;
    END IF;
END $$;

-- Add metadata JSONB column if missing
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'osa_workspaces' AND column_name = 'metadata'
    ) THEN
        ALTER TABLE osa_workspaces ADD COLUMN metadata JSONB DEFAULT '{}';
        CREATE INDEX idx_osa_workspaces_metadata ON osa_workspaces USING GIN(metadata);
    END IF;
END $$;

-- =============================================================================
-- OSA_DEPLOYMENTS - Deployment Version History (Flexible Design)
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_deployments (
    -- Core Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES osa_apps(id) ON DELETE CASCADE,

    -- Version Tracking
    version VARCHAR(50) NOT NULL,
    commit_sha VARCHAR(255),

    -- JSONB: Deployment Configuration (FLEXIBLE - stores URLs, paths, any deployment data)
    -- Schema: { "deployment_url": "...", "repo_url": "...", "branch": "...", "build_logs": "...", "deploy_platform": "..." }
    deployment_config JSONB DEFAULT '{}',

    -- JSONB: Deployment Metadata (FLEXIBLE - stores build info, test results, etc.)
    -- Schema: { "build_status": "...", "tests_passed": true, "files_changed": [...], "deploy_duration_ms": 1234 }
    metadata JSONB DEFAULT '{}',

    -- Ownership & Timestamps
    deployed_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint
    CONSTRAINT osa_deployments_app_version_unique UNIQUE(app_id, version)
);

-- Indexes for performance
CREATE INDEX idx_osa_deployments_app ON osa_deployments(app_id);
CREATE INDEX idx_osa_deployments_deployed_by ON osa_deployments(deployed_by);
CREATE INDEX idx_osa_deployments_deployed_at ON osa_deployments(deployed_at DESC);

-- GIN indexes for JSONB queries
CREATE INDEX idx_osa_deployments_config ON osa_deployments USING GIN(deployment_config);
CREATE INDEX idx_osa_deployments_metadata ON osa_deployments USING GIN(metadata);

-- =============================================================================
-- UPDATE TRIGGERS - Auto-update updated_at timestamps
-- =============================================================================

-- osa_apps updated_at trigger
CREATE OR REPLACE FUNCTION update_osa_apps_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_osa_apps_updated_at
    BEFORE UPDATE ON osa_apps
    FOR EACH ROW
    EXECUTE FUNCTION update_osa_apps_updated_at();

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE osa_apps IS 'Flexible OSA App Registry - uses JSONB for extensibility without schema changes';
COMMENT ON COLUMN osa_apps.template_type IS 'TEXT (not ENUM) - allows new template types without migration';
COMMENT ON COLUMN osa_apps.status IS 'TEXT (not ENUM) - flexible status values';
COMMENT ON COLUMN osa_apps.generation_context IS 'JSONB - stores prompt, preferences, any generation metadata';
COMMENT ON COLUMN osa_apps.deployment_config IS 'JSONB - stores repo URL, deployment URL, any deploy settings';
COMMENT ON COLUMN osa_apps.app_metadata IS 'JSONB - extensible storage for app-specific data';

COMMENT ON TABLE osa_deployments IS 'OSA App Deployment Version History - flexible JSONB-based storage';
COMMENT ON COLUMN osa_deployments.deployment_config IS 'JSONB - stores URLs, paths, any deployment configuration';
COMMENT ON COLUMN osa_deployments.metadata IS 'JSONB - stores build info, test results, any deployment metadata';

-- =============================================================================
-- EXAMPLE JSONB USAGE (commented out - for documentation)
-- =============================================================================

-- Example: Insert app with JSONB data
/*
INSERT INTO osa_apps (workspace_id, name, description, template_type, status, generation_context, deployment_config, app_metadata, created_by)
VALUES (
    'workspace-uuid-here',
    'My CRM App',
    'Customer relationship management system',
    'business_os',
    'active',
    '{"prompt": "Build a CRM with contacts and deals", "user_preferences": {"theme": "dark", "language": "en"}, "ai_model": "claude-sonnet-4"}'::jsonb,
    '{"repo_url": "https://github.com/user/my-crm", "deployment_url": "https://my-crm.vercel.app", "commit_sha": "abc123", "branch": "main"}'::jsonb,
    '{"files_created": 42, "tests_passed": true, "build_status": "success", "tech_stack": ["SvelteKit", "Go", "PostgreSQL"]}'::jsonb,
    'user-id-here'
);
*/

-- Example: Query JSONB fields
/*
-- Find apps with specific template type
SELECT * FROM osa_apps WHERE template_type = 'business_os';

-- Find apps by deployment URL
SELECT * FROM osa_apps WHERE deployment_config->>'deployment_url' = 'https://my-app.com';

-- Find apps with tests passed
SELECT * FROM osa_apps WHERE (app_metadata->>'tests_passed')::boolean = true;

-- Find apps created with specific AI model
SELECT * FROM osa_apps WHERE generation_context->>'ai_model' = 'claude-sonnet-4';

-- Update JSONB fields (merge, don't replace)
UPDATE osa_apps
SET app_metadata = app_metadata || '{"last_build": "2026-01-25T10:00:00Z"}'::jsonb
WHERE id = 'app-uuid-here';
*/
