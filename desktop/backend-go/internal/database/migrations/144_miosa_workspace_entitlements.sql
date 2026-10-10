-- Workspace-level MIOSA capacity entitlement.
-- BusinessOS admins use this to decide which customer workspaces are allowed
-- to spend the BusinessOS-owned MIOSA tenant credits.

CREATE TABLE IF NOT EXISTS miosa_workspace_entitlements (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    sandbox_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    computer_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    desktop_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    enabled_by_user_id TEXT,
    enabled_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_miosa_workspace_entitlements_sandbox_enabled
    ON miosa_workspace_entitlements(sandbox_enabled);
