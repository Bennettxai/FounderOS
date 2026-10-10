-- 119_sites.sql
-- The Sites module: a per-workspace registry of the business's web properties
-- (landing pages, marketing sites, apps), so humans AND AI agents have one
-- canonical list of where the business lives online. Workspace-scoped.

CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    url TEXT DEFAULT '',
    status VARCHAR(40) DEFAULT 'live',   -- live | draft | building | archived
    notes TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sites_workspace ON sites(workspace_id);
