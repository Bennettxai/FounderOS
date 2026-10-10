-- Migration: 044_osa_workflows_files.sql
-- Description: OSA-5 Workflows, Generated Files, and File Versioning
-- Created: 2026-01-09
-- Phase: 3 (Data Sync & File Management)

-- =============================================================================
-- OSA WORKFLOWS
-- Tracks OSA-5 code generation workflows
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Ownership & context
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    app_id UUID REFERENCES osa_generated_apps(id) ON DELETE SET NULL,

    -- Workflow identification
    osa_workflow_id VARCHAR(255), -- External OSA-5 workflow ID
    workflow_type VARCHAR(50) NOT NULL, -- 'app_generation', 'feature_addition', 'bug_fix', 'refactor', 'test_generation'

    -- Description & context
    title VARCHAR(500) NOT NULL,
    description TEXT,
    user_prompt TEXT NOT NULL, -- Original user request

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'planning', 'generating', 'testing', 'completed', 'failed', 'cancelled'

    -- Progress tracking
    progress_percent INTEGER DEFAULT 0 CHECK (progress_percent >= 0 AND progress_percent <= 100),
    current_phase VARCHAR(50), -- 'analyzing', 'planning', 'generating', 'testing', 'deploying'

    -- Results
    files_generated INTEGER DEFAULT 0,
    tests_passed INTEGER DEFAULT 0,
    tests_failed INTEGER DEFAULT 0,
    build_successful BOOLEAN,

    -- Error tracking
    error_message TEXT,
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,

    -- OSA-5 context
    osa_sandbox_id VARCHAR(255), -- E2B sandbox ID
    osa_metadata JSONB DEFAULT '{}', -- Full OSA-5 response data

    -- Execution metrics
    duration_seconds INTEGER,
    tokens_used INTEGER,
    estimated_cost_usd DECIMAL(10, 4),

    -- Tags for organization
    tags TEXT[],

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ
);

CREATE INDEX idx_osa_workflows_user ON osa_workflows(user_id);
CREATE INDEX idx_osa_workflows_workspace ON osa_workflows(workspace_id);
CREATE INDEX idx_osa_workflows_app ON osa_workflows(app_id);
CREATE INDEX idx_osa_workflows_status ON osa_workflows(status);
CREATE INDEX idx_osa_workflows_type ON osa_workflows(workflow_type);
CREATE INDEX idx_osa_workflows_osa_id ON osa_workflows(osa_workflow_id) WHERE osa_workflow_id IS NOT NULL;
CREATE INDEX idx_osa_workflows_created ON osa_workflows(created_at DESC);
CREATE INDEX idx_osa_workflows_tags ON osa_workflows USING GIN(tags);

-- =============================================================================
-- OSA GENERATED FILES
-- Individual files generated during workflows
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_generated_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Workflow association
    workflow_id UUID NOT NULL REFERENCES osa_workflows(id) ON DELETE CASCADE,
    app_id UUID REFERENCES osa_generated_apps(id) ON DELETE CASCADE,

    -- File identification
    file_path TEXT NOT NULL, -- Relative path from project root
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL, -- 'code', 'config', 'test', 'documentation', 'asset', 'migration'
    language VARCHAR(50), -- 'typescript', 'svelte', 'python', 'go', 'sql', 'json', 'yaml', etc.

    -- File content
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL, -- SHA-256 hash for deduplication and change detection

    -- File metadata
    file_size_bytes INTEGER NOT NULL,
    line_count INTEGER,
    encoding VARCHAR(20) DEFAULT 'utf-8',

    -- Installation tracking
    installation_status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'installed', 'skipped', 'failed', 'conflict'
    installation_path TEXT, -- Actual installed path in BusinessOS
    installed_at TIMESTAMPTZ,

    -- Conflict resolution
    conflict_reason TEXT,
    conflict_resolved_by VARCHAR(50), -- 'user', 'auto', 'merge'
    conflict_resolution JSONB, -- Details of how conflict was resolved

    -- File purpose & context
    purpose TEXT, -- Why this file was generated
    dependencies TEXT[], -- List of files this depends on
    dependents TEXT[], -- List of files that depend on this

    -- Version info (references osa_file_versions)
    current_version INTEGER DEFAULT 1,
    is_latest BOOLEAN DEFAULT true,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure unique paths per workflow
    CONSTRAINT osa_files_workflow_path_unique UNIQUE(workflow_id, file_path)
);

CREATE INDEX idx_osa_files_workflow ON osa_generated_files(workflow_id);
CREATE INDEX idx_osa_files_app ON osa_generated_files(app_id);
CREATE INDEX idx_osa_files_type ON osa_generated_files(file_type);
CREATE INDEX idx_osa_files_language ON osa_generated_files(language);
CREATE INDEX idx_osa_files_status ON osa_generated_files(installation_status);
CREATE INDEX idx_osa_files_hash ON osa_generated_files(content_hash);
CREATE INDEX idx_osa_files_path ON osa_generated_files(file_path);
CREATE INDEX idx_osa_files_name ON osa_generated_files(file_name);
CREATE INDEX idx_osa_files_latest ON osa_generated_files(is_latest) WHERE is_latest = true;

-- =============================================================================
-- OSA FILE VERSIONS
-- Version history for generated files
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- File reference
    file_id UUID NOT NULL REFERENCES osa_generated_files(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES osa_workflows(id) ON DELETE CASCADE,

    -- Version info
    version_number INTEGER NOT NULL,

    -- Content snapshot
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    file_size_bytes INTEGER NOT NULL,

    -- Change tracking
    change_type VARCHAR(50) NOT NULL, -- 'created', 'modified', 'renamed', 'moved', 'restored'
    change_summary TEXT, -- Brief description of what changed
    diff_from_previous TEXT, -- Unified diff format
    lines_added INTEGER DEFAULT 0,
    lines_removed INTEGER DEFAULT 0,

    -- Author tracking
    created_by_workflow_type VARCHAR(50), -- What type of workflow created this version
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique version numbers per file
    CONSTRAINT osa_file_versions_unique UNIQUE(file_id, version_number)
);

CREATE INDEX idx_osa_file_versions_file ON osa_file_versions(file_id, version_number DESC);
CREATE INDEX idx_osa_file_versions_workflow ON osa_file_versions(workflow_id);
CREATE INDEX idx_osa_file_versions_hash ON osa_file_versions(content_hash);
CREATE INDEX idx_osa_file_versions_created ON osa_file_versions(created_at DESC);

-- =============================================================================
-- OSA INSTALLATION LOG
-- Detailed log of file installations into BusinessOS
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_installation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Context
    workflow_id UUID NOT NULL REFERENCES osa_workflows(id) ON DELETE CASCADE,
    file_id UUID REFERENCES osa_generated_files(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Installation details
    action VARCHAR(50) NOT NULL, -- 'install', 'skip', 'overwrite', 'merge', 'rollback'
    source_path TEXT NOT NULL,
    destination_path TEXT NOT NULL,

    -- Result
    status VARCHAR(50) NOT NULL, -- 'success', 'failed', 'partial', 'rolled_back'

    -- Backup info (for rollback)
    backup_path TEXT,
    backup_content TEXT,
    backup_hash VARCHAR(64),

    -- Error tracking
    error_message TEXT,
    error_details JSONB,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_osa_install_workflow ON osa_installation_log(workflow_id);
CREATE INDEX idx_osa_install_file ON osa_installation_log(file_id);
CREATE INDEX idx_osa_install_user ON osa_installation_log(user_id);
CREATE INDEX idx_osa_install_status ON osa_installation_log(status);
CREATE INDEX idx_osa_install_action ON osa_installation_log(action);
CREATE INDEX idx_osa_install_created ON osa_installation_log(created_at DESC);

-- =============================================================================
-- OSA SYNC LOG (Enhanced from 042_osa_integration.sql)
-- Comprehensive audit trail for all sync operations
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Sync context
    sync_session_id UUID NOT NULL, -- Groups related sync operations
    entity_type VARCHAR(50) NOT NULL, -- 'user', 'workspace', 'app', 'module', 'workflow', 'file'
    entity_id UUID NOT NULL,

    -- Operation details
    operation VARCHAR(50) NOT NULL, -- 'create', 'update', 'delete', 'restore', 'conflict'
    direction VARCHAR(50) NOT NULL, -- 'to_osa', 'from_osa', 'bidirectional'

    -- Data snapshots
    before_snapshot JSONB,
    after_snapshot JSONB,
    changes JSONB, -- Structured diff of what changed

    -- Status
    status VARCHAR(50) NOT NULL, -- 'success', 'failed', 'conflict', 'skipped'

    -- Conflict details
    conflict_type VARCHAR(50), -- 'concurrent_update', 'deleted_modified', 'schema_mismatch'
    conflict_resolution VARCHAR(50), -- 'local_wins', 'remote_wins', 'merged', 'manual_required'

    -- Vector clock for distributed ordering
    vector_clock JSONB,

    -- Performance metrics
    duration_ms INTEGER,
    payload_size_bytes INTEGER,

    -- Error tracking
    error_message TEXT,
    error_stack TEXT,
    retry_count INTEGER DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_osa_sync_log_session ON osa_sync_log(sync_session_id);
CREATE INDEX idx_osa_sync_log_entity ON osa_sync_log(entity_type, entity_id);
CREATE INDEX idx_osa_sync_log_operation ON osa_sync_log(operation);
CREATE INDEX idx_osa_sync_log_status ON osa_sync_log(status);
CREATE INDEX idx_osa_sync_log_created ON osa_sync_log(created_at DESC);
CREATE INDEX idx_osa_sync_log_conflicts ON osa_sync_log(conflict_type) WHERE conflict_type IS NOT NULL;

-- =============================================================================
-- OSA MODULE DEPENDENCIES
-- Track dependencies between modules
-- =============================================================================
CREATE TABLE IF NOT EXISTS osa_module_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Dependency relationship
    module_id UUID NOT NULL REFERENCES osa_modules(id) ON DELETE CASCADE,
    depends_on_module_id UUID NOT NULL REFERENCES osa_modules(id) ON DELETE CASCADE,

    -- Dependency metadata
    dependency_type VARCHAR(50) NOT NULL, -- 'required', 'optional', 'peer', 'dev'
    version_constraint VARCHAR(100), -- Semantic version constraint (e.g., "^1.0.0", ">=2.0.0")

    -- Installation order (for topological sort)
    priority INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent circular dependencies and duplicates
    CONSTRAINT osa_module_deps_unique UNIQUE(module_id, depends_on_module_id),
    CONSTRAINT osa_module_deps_no_self CHECK(module_id != depends_on_module_id)
);

CREATE INDEX idx_osa_module_deps_module ON osa_module_dependencies(module_id);
CREATE INDEX idx_osa_module_deps_depends ON osa_module_dependencies(depends_on_module_id);
CREATE INDEX idx_osa_module_deps_type ON osa_module_dependencies(dependency_type);

-- =============================================================================
-- UPDATE TRIGGERS
-- Auto-update updated_at timestamps
-- =============================================================================

-- osa_workflows
CREATE OR REPLACE FUNCTION update_osa_workflows_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_osa_workflows_updated_at
    BEFORE UPDATE ON osa_workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_osa_workflows_updated_at();

-- osa_generated_files
CREATE OR REPLACE FUNCTION update_osa_generated_files_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_osa_generated_files_updated_at
    BEFORE UPDATE ON osa_generated_files
    FOR EACH ROW
    EXECUTE FUNCTION update_osa_generated_files_updated_at();

-- =============================================================================
-- FILE VERSION TRIGGER
-- Auto-create version entry when file content changes
-- =============================================================================
CREATE OR REPLACE FUNCTION create_file_version_on_update()
RETURNS TRIGGER AS $$
BEGIN
    -- Only create version if content actually changed
    IF OLD.content != NEW.content OR OLD.file_path != NEW.file_path THEN
        -- Increment version number
        NEW.current_version = OLD.current_version + 1;

        -- Create version history entry
        INSERT INTO osa_file_versions (
            file_id,
            workflow_id,
            version_number,
            content,
            content_hash,
            file_size_bytes,
            change_type,
            created_by_user_id,
            metadata
        ) VALUES (
            NEW.id,
            NEW.workflow_id,
            NEW.current_version,
            NEW.content,
            NEW.content_hash,
            NEW.file_size_bytes,
            CASE
                WHEN OLD.file_path != NEW.file_path THEN 'moved'
                ELSE 'modified'
            END,
            NULL, -- Can be enhanced to track actual user
            jsonb_build_object(
                'previous_version', OLD.current_version,
                'previous_hash', OLD.content_hash,
                'size_delta', NEW.file_size_bytes - OLD.file_size_bytes
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_file_version
    BEFORE UPDATE ON osa_generated_files
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content OR OLD.file_path IS DISTINCT FROM NEW.file_path)
    EXECUTE FUNCTION create_file_version_on_update();

-- =============================================================================
-- FILE INSERT VERSION TRIGGER
-- Create initial version when file is first created
-- =============================================================================
CREATE OR REPLACE FUNCTION create_initial_file_version()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO osa_file_versions (
        file_id,
        workflow_id,
        version_number,
        content,
        content_hash,
        file_size_bytes,
        change_type,
        created_by_user_id,
        metadata
    ) VALUES (
        NEW.id,
        NEW.workflow_id,
        1,
        NEW.content,
        NEW.content_hash,
        NEW.file_size_bytes,
        'created',
        NULL,
        jsonb_build_object('initial_version', true)
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_initial_version
    AFTER INSERT ON osa_generated_files
    FOR EACH ROW
    EXECUTE FUNCTION create_initial_file_version();

-- =============================================================================
-- WORKFLOW STATUS UPDATE TRIGGER
-- Auto-update timestamps based on status changes
-- =============================================================================
CREATE OR REPLACE FUNCTION update_workflow_status_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    -- Set started_at when workflow begins execution
    IF OLD.status IN ('pending', 'planning') AND NEW.status = 'generating' AND NEW.started_at IS NULL THEN
        NEW.started_at = NOW();
    END IF;

    -- Set completed_at when workflow finishes
    IF OLD.status NOT IN ('completed', 'failed', 'cancelled')
       AND NEW.status IN ('completed', 'failed')
       AND NEW.completed_at IS NULL THEN
        NEW.completed_at = NOW();

        -- Calculate duration
        IF NEW.started_at IS NOT NULL THEN
            NEW.duration_seconds = EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at))::INTEGER;
        END IF;
    END IF;

    -- Set cancelled_at when workflow is cancelled
    IF OLD.status != 'cancelled' AND NEW.status = 'cancelled' AND NEW.cancelled_at IS NULL THEN
        NEW.cancelled_at = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_workflow_status_timestamps
    BEFORE UPDATE ON osa_workflows
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION update_workflow_status_timestamps();

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE osa_workflows IS 'OSA-5 code generation workflows with status tracking and metrics';
COMMENT ON TABLE osa_generated_files IS 'Individual files generated during OSA workflows with installation tracking';
COMMENT ON TABLE osa_file_versions IS 'Complete version history for generated files with diffs';
COMMENT ON TABLE osa_installation_log IS 'Audit trail of file installations into BusinessOS';
COMMENT ON TABLE osa_sync_log IS 'Comprehensive sync operation audit trail with conflict tracking';
COMMENT ON TABLE osa_module_dependencies IS 'Module dependency graph for installation ordering';

COMMENT ON COLUMN osa_workflows.status IS 'pending: queued, planning: analyzing requirements, generating: creating code, testing: running tests, completed: finished successfully, failed: error occurred, cancelled: user cancelled';
COMMENT ON COLUMN osa_workflows.workflow_type IS 'Type of code generation workflow';
COMMENT ON COLUMN osa_workflows.progress_percent IS 'Completion percentage (0-100)';

COMMENT ON COLUMN osa_generated_files.file_type IS 'Categorization: code, config, test, documentation, asset, migration';
COMMENT ON COLUMN osa_generated_files.installation_status IS 'pending: not installed, installed: successfully installed, skipped: user skipped, failed: installation error, conflict: merge conflict detected';
COMMENT ON COLUMN osa_generated_files.content_hash IS 'SHA-256 hash for deduplication and change detection';
COMMENT ON COLUMN osa_generated_files.is_latest IS 'Whether this is the current version (for quick queries)';

COMMENT ON COLUMN osa_file_versions.change_type IS 'Type of change: created, modified, renamed, moved, restored';
COMMENT ON COLUMN osa_file_versions.diff_from_previous IS 'Unified diff showing changes from previous version';

COMMENT ON COLUMN osa_installation_log.action IS 'install: new file, skip: user skipped, overwrite: replaced existing, merge: merged with existing, rollback: reverted installation';
COMMENT ON COLUMN osa_installation_log.backup_content IS 'Original file content before overwrite (for rollback)';

COMMENT ON COLUMN osa_sync_log.sync_session_id IS 'Groups related sync operations in a single batch';
COMMENT ON COLUMN osa_sync_log.vector_clock IS 'Logical timestamp for distributed conflict detection';
COMMENT ON COLUMN osa_sync_log.conflict_type IS 'concurrent_update: both modified, deleted_modified: deleted locally but modified remotely, schema_mismatch: incompatible schemas';

-- =============================================================================
-- HELPER FUNCTIONS
-- =============================================================================

-- Get file version history
CREATE OR REPLACE FUNCTION get_file_version_history(p_file_id UUID)
RETURNS TABLE (
    version_number INTEGER,
    content_hash VARCHAR(64),
    change_type VARCHAR(50),
    change_summary TEXT,
    lines_added INTEGER,
    lines_removed INTEGER,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        v.version_number,
        v.content_hash,
        v.change_type,
        v.change_summary,
        v.lines_added,
        v.lines_removed,
        v.created_at
    FROM osa_file_versions v
    WHERE v.file_id = p_file_id
    ORDER BY v.version_number DESC;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_file_version_history IS 'Returns version history for a file ordered by version (newest first)';

-- Get workflow statistics
CREATE OR REPLACE FUNCTION get_workflow_stats(p_workflow_id UUID)
RETURNS TABLE (
    total_files INTEGER,
    files_by_type JSONB,
    files_by_status JSONB,
    total_lines INTEGER,
    total_size_bytes BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::INTEGER as total_files,
        jsonb_object_agg(file_type, type_count) as files_by_type,
        jsonb_object_agg(installation_status, status_count) as files_by_status,
        SUM(line_count)::INTEGER as total_lines,
        SUM(file_size_bytes)::BIGINT as total_size_bytes
    FROM (
        SELECT
            file_type,
            installation_status,
            line_count,
            file_size_bytes,
            COUNT(*) OVER (PARTITION BY file_type) as type_count,
            COUNT(*) OVER (PARTITION BY installation_status) as status_count
        FROM osa_generated_files
        WHERE workflow_id = p_workflow_id
    ) stats;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_workflow_stats IS 'Returns aggregate statistics for files in a workflow';

-- =============================================================================
-- CLEANUP FUNCTION
-- Remove old workflow data to prevent unbounded growth
-- =============================================================================
CREATE OR REPLACE FUNCTION cleanup_old_osa_workflows()
RETURNS void AS $$
BEGIN
    -- Archive completed workflows older than 90 days
    -- Delete failed workflows older than 30 days
    -- Keep cancelled workflows for 7 days

    DELETE FROM osa_workflows
    WHERE
        (status = 'completed' AND completed_at < NOW() - INTERVAL '90 days')
        OR (status = 'failed' AND completed_at < NOW() - INTERVAL '30 days')
        OR (status = 'cancelled' AND cancelled_at < NOW() - INTERVAL '7 days');

    -- Cleanup old sync logs (keep 30 days)
    DELETE FROM osa_sync_log
    WHERE created_at < NOW() - INTERVAL '30 days';

    -- Cleanup old installation logs (keep 60 days)
    DELETE FROM osa_installation_log
    WHERE created_at < NOW() - INTERVAL '60 days';
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_old_osa_workflows IS 'Cleanup function to remove old workflow data (run weekly via cron job)';
