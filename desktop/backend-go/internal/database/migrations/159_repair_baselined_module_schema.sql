-- Repair module schema that was falsely recorded as applied by the legacy
-- existing-database baseline path. Every statement is additive and idempotent.

CREATE TABLE IF NOT EXISTS communication_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL
        CHECK (provider IN ('gmail', 'outlook', 'slack', 'teams', 'whatsapp')),
    scope VARCHAR(24) NOT NULL
        CHECK (scope IN ('account', 'conversation')),
    external_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider, scope, external_id)
);

CREATE INDEX IF NOT EXISTS idx_communication_routes_resolve
    ON communication_routes(user_id, provider, scope, external_id)
    WHERE enabled;
CREATE INDEX IF NOT EXISTS idx_communication_routes_workspace
    ON communication_routes(workspace_id);

CREATE TABLE IF NOT EXISTS microsoft_teams_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    team_name VARCHAR(255),
    display_name VARCHAR(255),
    description TEXT,
    email VARCHAR(255),
    web_url TEXT,
    membership_type VARCHAR(50),
    is_archived BOOLEAN DEFAULT FALSE,
    is_favorite_by_default BOOLEAN DEFAULT FALSE,
    member_count INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_datetime TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, team_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_user
    ON microsoft_teams_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_team
    ON microsoft_teams_channels(user_id, team_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_activity
    ON microsoft_teams_channels(user_id, last_message_at DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS microsoft_teams_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    reply_to_id VARCHAR(255),
    message_type VARCHAR(50),
    importance VARCHAR(50),
    subject TEXT,
    summary TEXT,
    body_content TEXT,
    body_content_type VARCHAR(50),
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    sender_type VARCHAR(50),
    attachments JSONB DEFAULT '[]',
    mentions JSONB DEFAULT '[]',
    reactions JSONB DEFAULT '[]',
    web_url TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_edited BOOLEAN DEFAULT FALSE,
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,
    last_edited_datetime TIMESTAMPTZ,
    deleted_datetime TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, team_id, channel_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_user
    ON microsoft_teams_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_channel
    ON microsoft_teams_messages(user_id, team_id, channel_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_recent
    ON microsoft_teams_messages(user_id, team_id, channel_id, created_datetime DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_thread
    ON microsoft_teams_messages(user_id, reply_to_id) WHERE reply_to_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_sender
    ON microsoft_teams_messages(user_id, sender_id);

CREATE TABLE IF NOT EXISTS microsoft_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    subscription_id VARCHAR(255) NOT NULL,
    resource VARCHAR(500) NOT NULL,
    resource_kind VARCHAR(50) NOT NULL,
    change_type VARCHAR(100),
    notification_url TEXT,
    client_state VARCHAR(255),
    expires_at TIMESTAMPTZ,
    last_renewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, subscription_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_subs_user ON microsoft_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_subs_expiring
    ON microsoft_subscriptions(expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ms_subs_subscription
    ON microsoft_subscriptions(subscription_id);

CREATE TABLE IF NOT EXISTS google_gmail_watches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    topic_name VARCHAR(500) NOT NULL,
    history_id BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_renewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmail_watches_expiring
    ON google_gmail_watches(expires_at);

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

CREATE INDEX IF NOT EXISTS idx_custom_module_versions_module
    ON custom_module_versions(module_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_versions_version
    ON custom_module_versions(module_id, version);
CREATE INDEX IF NOT EXISTS idx_custom_module_installations_workspace
    ON custom_module_installations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_installations_module
    ON custom_module_installations(module_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_module
    ON custom_module_shares(module_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_user
    ON custom_module_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_workspace
    ON custom_module_shares(shared_with_workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_module_shares_email
    ON custom_module_shares(shared_with_email);

ALTER TABLE content_items
    ADD COLUMN IF NOT EXISTS theme VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS views INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reach INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS likes INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS comments INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS saves INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS shares INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reposts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS follows INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS profile_activity INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS accounts_engaged INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS avg_watch_time_seconds NUMERIC(10,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS retention_rate NUMERIC(6,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS analytics_notes TEXT NOT NULL DEFAULT '';
