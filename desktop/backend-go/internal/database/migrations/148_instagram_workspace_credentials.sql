-- Instagram credentials belong to a BusinessOS workspace, not an individual user.
-- Existing Instagram vault rows contain plaintext access tokens and cannot be
-- migrated safely in SQL because the application encryption key is unavailable
-- to migrations. Remove them and require affected workspaces to reconnect.
DELETE FROM credential_vault WHERE provider_id = 'instagram';
DELETE FROM user_integrations WHERE provider_id = 'instagram';

CREATE TABLE IF NOT EXISTS instagram_workspace_credentials (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    connected_by_user_id VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    encrypted_data BYTEA NOT NULL,
    encryption_version INTEGER NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ,
    external_account_id VARCHAR(255),
    external_account_name VARCHAR(255),
    scopes TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_instagram_workspace_credentials_connected_by
    ON instagram_workspace_credentials(connected_by_user_id);

CREATE TABLE IF NOT EXISTS instagram_oauth_states (
    state_hash BYTEA PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_instagram_oauth_states_expires
    ON instagram_oauth_states(expires_at);
