-- BusinessOS-owned MIOSA tenant workspace mapping.
-- The raw MIOSA tenant key stays in credential_vault under provider_id
-- miosa_platform. This table stores only non-secret workspace attribution.

CREATE TABLE IF NOT EXISTS miosa_workspace_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    miosa_workspace_id TEXT NOT NULL,
    external_workspace_id TEXT NOT NULL,
    external_user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'linked',
    created_by_user_id TEXT REFERENCES "user"(id) ON DELETE SET NULL,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id),
    UNIQUE (miosa_workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_miosa_workspace_links_external_workspace
    ON miosa_workspace_links(external_workspace_id);

CREATE INDEX IF NOT EXISTS idx_miosa_workspace_links_external_user
    ON miosa_workspace_links(external_user_id);
