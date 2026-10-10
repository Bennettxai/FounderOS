-- ═══════════════════════════════════════════════════════════════════════════════
-- Custom Modules System Migration
-- ═══════════════════════════════════════════════════════════════════════════════
-- Created: 2026-01-25
-- Description: Adds tables and indexes for custom module creation, export/import,
--              installation, and sharing functionality
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: custom_modules
-- Stores user-created modules with metadata and configuration
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS custom_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Ownership
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Module Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL, -- URL-friendly name
    description TEXT,
    category VARCHAR(100), -- e.g., 'productivity', 'communication', 'finance'

    -- Versioning
    version VARCHAR(50) DEFAULT '0.0.1', -- Semantic versioning

    -- Module Content (JSONB for flexibility)
    manifest JSONB NOT NULL DEFAULT '{}', -- Module configuration
    /*
     * Manifest structure:
     * {
     *   "dependencies": ["module-slug-1", "module-slug-2"],
     *   "actions": [
     *     {
     *       "name": "my_custom_action",
     *       "type": "function|api|workflow",
     *       "handler": "...",
     *       "params": {...},
     *       "returns": {...}
     *     }
     *   ],
     *   "files": {
     *     "functions/handler.js": "base64_encoded_content",
     *     "config/settings.json": "base64_encoded_content"
     *   },
     *   "config_schema": {...},  -- JSON Schema for module configuration
     *   "permissions": ["read:contacts", "write:tasks"]
     * }
     */

    -- Configuration
    config JSONB DEFAULT '{}', -- User-specific configuration for this module

    -- Metadata
    icon VARCHAR(100), -- Lucide icon name or URL
    tags TEXT[], -- Searchable tags
    keywords TEXT[], -- Search keywords

    -- Publication Status
    is_public BOOLEAN DEFAULT FALSE, -- Can be shared/discovered
    is_published BOOLEAN DEFAULT FALSE, -- Published to module registry
    is_template BOOLEAN DEFAULT FALSE, -- Can be cloned

    -- Stats
    install_count INTEGER DEFAULT 0,
    star_count INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(workspace_id, slug),
    CONSTRAINT valid_slug CHECK (slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT valid_version CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

-- Indexes for custom_modules
CREATE INDEX idx_custom_modules_workspace ON custom_modules(workspace_id);
CREATE INDEX idx_custom_modules_created_by ON custom_modules(created_by);
CREATE INDEX idx_custom_modules_slug ON custom_modules(slug);
CREATE INDEX idx_custom_modules_category ON custom_modules(category);
CREATE INDEX idx_custom_modules_public ON custom_modules(is_public) WHERE is_public = TRUE;
CREATE INDEX idx_custom_modules_published ON custom_modules(is_published) WHERE is_published = TRUE;
CREATE INDEX idx_custom_modules_tags ON custom_modules USING GIN(tags);
CREATE INDEX idx_custom_modules_keywords ON custom_modules USING GIN(keywords);
CREATE INDEX idx_custom_modules_manifest ON custom_modules USING GIN(manifest);

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: custom_module_versions
-- Tracks version history for custom modules
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS custom_module_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Reference
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,

    -- Version Info
    version VARCHAR(50) NOT NULL, -- Semantic versioning
    changelog TEXT, -- What changed in this version

    -- Snapshot
    manifest_snapshot JSONB NOT NULL, -- Full manifest at this version
    config_snapshot JSONB, -- Config schema at this version

    -- Metadata
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Release Info
    is_stable BOOLEAN DEFAULT TRUE,
    is_breaking BOOLEAN DEFAULT FALSE, -- Breaking changes from previous version

    -- Constraints
    UNIQUE(module_id, version),
    CONSTRAINT valid_version CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

-- Indexes for custom_module_versions
CREATE INDEX idx_module_versions_module ON custom_module_versions(module_id);
CREATE INDEX idx_module_versions_created_at ON custom_module_versions(created_at DESC);
CREATE INDEX idx_module_versions_stable ON custom_module_versions(module_id, is_stable)
    WHERE is_stable = TRUE;

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: module_installations
-- Tracks which modules are installed in which workspaces
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS module_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Reference
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    installed_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,

    -- Installation Details
    installed_version VARCHAR(50) NOT NULL,
    config_override JSONB DEFAULT '{}', -- Workspace-specific configuration

    -- State
    is_enabled BOOLEAN DEFAULT TRUE,
    is_auto_update BOOLEAN DEFAULT TRUE, -- Auto-update to new versions

    -- Timestamps
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(module_id, workspace_id),
    CONSTRAINT valid_version CHECK (installed_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

-- Indexes for module_installations
CREATE INDEX idx_module_installations_module ON module_installations(module_id);
CREATE INDEX idx_module_installations_workspace ON module_installations(workspace_id);
CREATE INDEX idx_module_installations_enabled ON module_installations(workspace_id, is_enabled)
    WHERE is_enabled = TRUE;

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: module_shares
-- Manages sharing permissions for modules
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS module_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Reference
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,

    -- Sharing Target (one of these will be set)
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with_workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    shared_with_email VARCHAR(255), -- Share via email (pending invitation)

    -- Permissions
    can_view BOOLEAN DEFAULT TRUE,
    can_install BOOLEAN DEFAULT TRUE,
    can_modify BOOLEAN DEFAULT FALSE, -- Can edit the module
    can_reshare BOOLEAN DEFAULT FALSE, -- Can share with others

    -- Metadata
    shared_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE, -- Optional expiration

    -- Constraints
    CONSTRAINT one_target_only CHECK (
        (shared_with_user_id IS NOT NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NOT NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NOT NULL)
    )
);

-- Indexes for module_shares
CREATE INDEX idx_module_shares_module ON module_shares(module_id);
CREATE INDEX idx_module_shares_user ON module_shares(shared_with_user_id)
    WHERE shared_with_user_id IS NOT NULL;
CREATE INDEX idx_module_shares_workspace ON module_shares(shared_with_workspace_id)
    WHERE shared_with_workspace_id IS NOT NULL;
CREATE INDEX idx_module_shares_email ON module_shares(shared_with_email)
    WHERE shared_with_email IS NOT NULL;

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: Update custom_modules.updated_at on changes
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_custom_modules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_custom_modules_updated_at
    BEFORE UPDATE ON custom_modules
    FOR EACH ROW
    EXECUTE FUNCTION update_custom_modules_updated_at();

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: Update module_installations.updated_at on changes
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_module_installations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_module_installations_updated_at
    BEFORE UPDATE ON module_installations
    FOR EACH ROW
    EXECUTE FUNCTION update_module_installations_updated_at();

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: Create version snapshot when module is updated
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION create_module_version_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    -- Only create snapshot if version changed
    IF OLD.version <> NEW.version THEN
        INSERT INTO custom_module_versions (
            module_id,
            version,
            manifest_snapshot,
            config_snapshot,
            created_by
        ) VALUES (
            NEW.id,
            NEW.version,
            NEW.manifest,
            NEW.config,
            NEW.created_by
        )
        ON CONFLICT (module_id, version) DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_module_version_snapshot
    AFTER UPDATE ON custom_modules
    FOR EACH ROW
    WHEN (OLD.version <> NEW.version)
    EXECUTE FUNCTION create_module_version_snapshot();

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: Increment install_count when module is installed
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION increment_module_install_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE custom_modules
    SET install_count = install_count + 1
    WHERE id = NEW.module_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_increment_module_install_count
    AFTER INSERT ON module_installations
    FOR EACH ROW
    EXECUTE FUNCTION increment_module_install_count();

-- ═══════════════════════════════════════════════════════════════════════════════
-- COMMENTS FOR DOCUMENTATION
-- ═══════════════════════════════════════════════════════════════════════════════

COMMENT ON TABLE custom_modules IS 'User-created modules with reusable functionality, actions, and configurations';
COMMENT ON TABLE custom_module_versions IS 'Version history and snapshots for custom modules';
COMMENT ON TABLE module_installations IS 'Tracks which modules are installed in which workspaces';
COMMENT ON TABLE module_shares IS 'Sharing permissions for custom modules';

COMMENT ON COLUMN custom_modules.manifest IS 'JSONB manifest containing module definition: dependencies, actions, files, config schema, permissions';
COMMENT ON COLUMN custom_modules.config IS 'User-specific configuration for this module instance';
COMMENT ON COLUMN custom_modules.is_public IS 'Can be discovered and shared with others';
COMMENT ON COLUMN custom_modules.is_published IS 'Published to BusinessOS module registry';
COMMENT ON COLUMN custom_modules.is_template IS 'Can be cloned/forked by other users';

-- ═══════════════════════════════════════════════════════════════════════════════
-- END OF MIGRATION
-- ═══════════════════════════════════════════════════════════════════════════════
