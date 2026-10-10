-- 130_workspace_apps.sql
-- Workspace-owned URL apps that BusinessOS can launch inside its desktop.
-- These are not native OS apps and not canonical BusinessOS modules.

CREATE TABLE IF NOT EXISTS workspace_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    catalog_app_id UUID,
    name VARCHAR(160) NOT NULL,
    app_type VARCHAR(40) DEFAULT 'web_app',
    provider VARCHAR(60) DEFAULT 'custom',
    url TEXT NOT NULL DEFAULT '',
    launch_mode VARCHAR(40) DEFAULT 'iframe',
    status VARCHAR(40) DEFAULT 'active',
    icon VARCHAR(80) DEFAULT 'layout-grid',
    logo_url TEXT DEFAULT '',
    color VARCHAR(40) DEFAULT '#111827',
    category VARCHAR(80) DEFAULT 'general',
    notes TEXT DEFAULT '',
    show_on_desktop BOOLEAN DEFAULT TRUE,
    show_in_dock BOOLEAN DEFAULT TRUE,
    position_index INTEGER DEFAULT 0,
    url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_apps_workspace ON workspace_apps(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_status ON workspace_apps(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_desktop ON workspace_apps(workspace_id, show_on_desktop);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_position ON workspace_apps(workspace_id, position_index, name);
