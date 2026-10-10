-- Custom Modules System - User-created modules and marketplace
-- Enables users to create, share, and install custom modules

-- Main modules table
CREATE TABLE IF NOT EXISTS custom_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by TEXT NOT NULL,
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '0.0.1',
    manifest JSONB NOT NULL,
    config JSONB DEFAULT '{}',
    icon VARCHAR(255),
    tags TEXT[] DEFAULT '{}',
    keywords TEXT[] DEFAULT '{}',
    is_public BOOLEAN DEFAULT FALSE,
    is_published BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    install_count INTEGER DEFAULT 0,
    star_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    UNIQUE(workspace_id, slug)
);

-- Module versions table (for version history)
CREATE TABLE IF NOT EXISTS custom_module_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    changelog TEXT,
    manifest_snapshot JSONB NOT NULL,
    config_snapshot JSONB DEFAULT '{}',
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_stable BOOLEAN DEFAULT FALSE,
    is_breaking BOOLEAN DEFAULT FALSE,
    UNIQUE(module_id, version)
);

-- Module installations table
CREATE TABLE IF NOT EXISTS custom_module_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL,
    installed_by UUID NOT NULL,
    installed_version VARCHAR(50) NOT NULL,
    config_override JSONB DEFAULT '{}',
    is_enabled BOOLEAN DEFAULT TRUE,
    is_auto_update BOOLEAN DEFAULT FALSE,
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    UNIQUE(module_id, workspace_id)
);

-- Module shares table (for sharing with specific users/workspaces)
CREATE TABLE IF NOT EXISTS custom_module_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    shared_with_user_id UUID,
    shared_with_workspace_id UUID,
    shared_with_email VARCHAR(255),
    can_view BOOLEAN DEFAULT TRUE,
    can_install BOOLEAN DEFAULT TRUE,
    can_modify BOOLEAN DEFAULT FALSE,
    can_reshare BOOLEAN DEFAULT FALSE,
    shared_by UUID NOT NULL,
    shared_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    CHECK (
        (shared_with_user_id IS NOT NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NOT NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NOT NULL)
    )
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_custom_modules_workspace ON custom_modules(workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_modules_created_by ON custom_modules(created_by);
CREATE INDEX IF NOT EXISTS idx_custom_modules_slug ON custom_modules(slug);
CREATE INDEX IF NOT EXISTS idx_custom_modules_category ON custom_modules(category);
CREATE INDEX IF NOT EXISTS idx_custom_modules_public_published ON custom_modules(is_public, is_published);
CREATE INDEX IF NOT EXISTS idx_custom_modules_tags ON custom_modules USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_custom_modules_keywords ON custom_modules USING GIN(keywords);

CREATE INDEX IF NOT EXISTS idx_custom_module_versions_module ON custom_module_versions(module_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_versions_version ON custom_module_versions(module_id, version);

CREATE INDEX IF NOT EXISTS idx_custom_module_installations_workspace ON custom_module_installations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_installations_module ON custom_module_installations(module_id);

CREATE INDEX IF NOT EXISTS idx_custom_module_shares_module ON custom_module_shares(module_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_user ON custom_module_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_workspace ON custom_module_shares(shared_with_workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_email ON custom_module_shares(shared_with_email);
