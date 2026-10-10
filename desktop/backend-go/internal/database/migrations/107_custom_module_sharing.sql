-- Add share_scope to custom_modules for organization-wide module sharing.
-- scope 'workspace' (default) = visible only in its own workspace.
-- scope 'organization' = visible to all workspaces in the same organization.

ALTER TABLE custom_modules
    ADD COLUMN IF NOT EXISTS share_scope VARCHAR(20) NOT NULL DEFAULT 'workspace'
        CHECK (share_scope IN ('workspace', 'organization'));

CREATE INDEX IF NOT EXISTS idx_custom_modules_share_scope
    ON custom_modules (share_scope)
    WHERE share_scope = 'organization';
