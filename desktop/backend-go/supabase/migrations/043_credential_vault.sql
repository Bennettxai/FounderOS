-- Migration: 027_credential_vault.sql
-- Unified credential storage with encryption for all OAuth tokens and API keys
-- This replaces the separate google_oauth_tokens, slack_oauth_tokens, notion_oauth_tokens tables
-- with a single encrypted vault

-- ============================================================================
-- CREDENTIAL VAULT
-- ============================================================================
-- Stores all credentials (OAuth tokens, API keys) encrypted with AES-256-GCM
-- The encryption key is stored in environment (TOKEN_ENCRYPTION_KEY)

CREATE TABLE IF NOT EXISTS credential_vault (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL,

    -- Credential type
    credential_type VARCHAR(20) NOT NULL DEFAULT 'oauth', -- oauth, api_key, custom

    -- Encrypted credential data (AES-256-GCM)
    -- Contains JSON: {"access_token": "...", "refresh_token": "...", "token_type": "..."}
    encrypted_data BYTEA NOT NULL,

    -- Encryption metadata
    encryption_version INT DEFAULT 1, -- For future key rotation

    -- Token expiry (for OAuth)
    expires_at TIMESTAMPTZ,

    -- External account info (NOT encrypted - for display purposes)
    external_account_id VARCHAR(255),
    external_account_email VARCHAR(255),
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),

    -- Scopes granted
    scopes TEXT[] DEFAULT '{}',

    -- Metadata (NOT encrypted - for app logic)
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    last_rotated_at TIMESTAMPTZ,

    -- One credential per user per provider
    UNIQUE(user_id, provider_id)
);

-- ============================================================================
-- INDEXES
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_credential_vault_user ON credential_vault(user_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_provider ON credential_vault(provider_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_type ON credential_vault(credential_type);
CREATE INDEX IF NOT EXISTS idx_credential_vault_expires ON credential_vault(expires_at)
    WHERE expires_at IS NOT NULL;

-- ============================================================================
-- WEBHOOK REGISTRATIONS
-- ============================================================================
-- Track webhooks registered with external providers for real-time sync

CREATE TABLE IF NOT EXISTS integration_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL,

    -- Webhook configuration
    webhook_url TEXT NOT NULL,
    webhook_secret_encrypted BYTEA, -- For signature verification

    -- Events this webhook listens for
    events TEXT[] NOT NULL DEFAULT '{}',

    -- Status
    status VARCHAR(20) DEFAULT 'active', -- active, paused, failed, deleted

    -- Tracking
    last_triggered_at TIMESTAMPTZ,
    failure_count INT DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, provider_id, webhook_url)
);

CREATE INDEX IF NOT EXISTS idx_webhooks_user ON integration_webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_provider ON integration_webhooks(provider_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON integration_webhooks(status);

-- ============================================================================
-- DATA SYNC MAPPINGS
-- ============================================================================
-- Configure how data from external providers maps to BusinessOS modules

CREATE TABLE IF NOT EXISTS data_sync_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Source (external)
    source_provider VARCHAR(50) NOT NULL, -- hubspot, slack, etc.
    source_entity VARCHAR(100) NOT NULL,  -- contacts, deals, messages, etc.

    -- Target (BusinessOS)
    target_module VARCHAR(50) NOT NULL,   -- clients, tasks, contexts, etc.
    target_entity VARCHAR(100),           -- Optional sub-entity

    -- Field mappings: {"source_field": "target_field", ...}
    field_mappings JSONB NOT NULL DEFAULT '{}',

    -- Transform rules: {"field": {"type": "date", "format": "..."}, ...}
    transform_rules JSONB DEFAULT '{}',

    -- Sync configuration
    enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'import', -- import, export, bidirectional
    sync_frequency VARCHAR(20) DEFAULT 'manual', -- realtime, hourly, daily, manual

    -- Tracking
    last_synced_at TIMESTAMPTZ,
    records_synced INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_provider, source_entity, target_module)
);

CREATE INDEX IF NOT EXISTS idx_sync_mappings_user ON data_sync_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_mappings_provider ON data_sync_mappings(source_provider);
CREATE INDEX IF NOT EXISTS idx_sync_mappings_enabled ON data_sync_mappings(enabled) WHERE enabled = true;

-- ============================================================================
-- MIGRATION HELPER: Copy existing tokens to vault
-- ============================================================================
-- This function helps migrate existing OAuth tokens to the new vault
-- Run manually after deploying the migration

-- Note: Actual migration of data should be done in application code
-- because we need access to the encryption key to re-encrypt tokens

-- Create a view to help identify unmigrated tokens
CREATE OR REPLACE VIEW unmigrated_oauth_tokens AS
SELECT
    'google' as provider,
    user_id,
    created_at
FROM google_oauth_tokens
WHERE NOT EXISTS (
    SELECT 1 FROM credential_vault cv
    WHERE cv.user_id = google_oauth_tokens.user_id
    AND cv.provider_id = 'google'
)
UNION ALL
SELECT
    'slack' as provider,
    user_id,
    created_at
FROM slack_oauth_tokens
WHERE NOT EXISTS (
    SELECT 1 FROM credential_vault cv
    WHERE cv.user_id = slack_oauth_tokens.user_id
    AND cv.provider_id = 'slack'
)
UNION ALL
SELECT
    'notion' as provider,
    user_id,
    created_at
FROM notion_oauth_tokens
WHERE NOT EXISTS (
    SELECT 1 FROM credential_vault cv
    WHERE cv.user_id = notion_oauth_tokens.user_id
    AND cv.provider_id = 'notion'
);

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE credential_vault IS 'Encrypted storage for OAuth tokens and API keys';
COMMENT ON COLUMN credential_vault.encrypted_data IS 'AES-256-GCM encrypted JSON containing tokens';
COMMENT ON COLUMN credential_vault.encryption_version IS 'Version of encryption key used, for key rotation';
COMMENT ON TABLE integration_webhooks IS 'Webhook registrations for real-time data sync';
COMMENT ON TABLE data_sync_mappings IS 'Configuration for mapping external data to BusinessOS';
