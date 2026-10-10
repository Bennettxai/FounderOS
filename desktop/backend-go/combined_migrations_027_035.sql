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
-- Migration: 028_data_imports.sql
-- Data import infrastructure for external data sources
-- Handles ChatGPT/Claude exports, CRM imports, analytics data, etc.

-- ============================================================================
-- IMPORT JOBS
-- ============================================================================
-- Tracks all data import operations with status and progress

CREATE TYPE import_status AS ENUM (
    'pending',      -- Waiting to start
    'validating',   -- Validating file/data format
    'mapping',      -- User configuring field mappings
    'processing',   -- Import in progress
    'completed',    -- Successfully completed
    'failed',       -- Failed with error
    'cancelled'     -- Cancelled by user
);

CREATE TYPE import_source_type AS ENUM (
    -- AI Chat Exports
    'chatgpt_export',       -- OpenAI ChatGPT JSON export
    'claude_export',        -- Anthropic Claude JSON export
    'custom_chat_export',   -- Other chat format

    -- CRM Data
    'hubspot_contacts',     -- HubSpot CRM contacts
    'hubspot_deals',        -- HubSpot deals/opportunities
    'hubspot_companies',    -- HubSpot companies
    'salesforce_contacts',  -- Salesforce contacts
    'salesforce_accounts',  -- Salesforce accounts

    -- Task/Project Tools
    'linear_issues',        -- Linear issues
    'notion_database',      -- Notion database export
    'asana_tasks',          -- Asana tasks
    'jira_issues',          -- Jira issues

    -- Calendar/Meetings
    'google_calendar',      -- Google Calendar events
    'outlook_calendar',     -- Outlook Calendar events

    -- Analytics
    'fathom_analytics',     -- Fathom Analytics data
    'plausible_analytics',  -- Plausible Analytics data

    -- Files/Documents
    'google_drive',         -- Google Drive files
    'dropbox',              -- Dropbox files
    'notion_pages',         -- Notion pages

    -- Other
    'csv_generic',          -- Generic CSV import
    'json_generic',         -- Generic JSON import
    'custom'                -- Custom import type
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Import type and source
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),  -- Link to integration_providers if applicable

    -- File/data info
    original_filename VARCHAR(500),
    file_size_bytes BIGINT,
    content_type VARCHAR(100),

    -- Status tracking
    status import_status DEFAULT 'pending',
    progress_percent INT DEFAULT 0,

    -- Counts
    total_records INT DEFAULT 0,
    processed_records INT DEFAULT 0,
    imported_records INT DEFAULT 0,
    skipped_records INT DEFAULT 0,
    failed_records INT DEFAULT 0,

    -- Mapping configuration (from data_sync_mappings or inline)
    field_mapping JSONB DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    import_options JSONB DEFAULT '{}',  -- dedup strategy, conflict handling, etc.

    -- Target module in BusinessOS
    target_module VARCHAR(50) NOT NULL,  -- clients, conversations, tasks, etc.
    target_entity VARCHAR(100),          -- Optional sub-entity

    -- Results
    result_summary JSONB DEFAULT '{}',   -- Summary of what was imported
    error_log JSONB DEFAULT '[]',        -- Array of errors with line numbers

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Error info
    error_message TEXT,
    error_details JSONB
);

CREATE INDEX idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_jobs_source ON import_jobs(source_type);
CREATE INDEX idx_import_jobs_created ON import_jobs(created_at DESC);

-- ============================================================================
-- IMPORTED RECORDS TRACKING
-- ============================================================================
-- Links external record IDs to BusinessOS record IDs for deduplication

CREATE TABLE IF NOT EXISTS imported_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,

    -- Source identification
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),
    external_id VARCHAR(500) NOT NULL,  -- ID from source system

    -- Target identification
    target_module VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100),
    target_record_id UUID NOT NULL,     -- ID in BusinessOS

    -- Metadata
    external_data_hash VARCHAR(64),     -- SHA256 of source data for change detection
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_type, external_id)
);

CREATE INDEX idx_imported_records_user ON imported_records(user_id);
CREATE INDEX idx_imported_records_source ON imported_records(source_type, external_id);
CREATE INDEX idx_imported_records_target ON imported_records(target_module, target_record_id);

-- ============================================================================
-- FIELD MAPPING TEMPLATES
-- ============================================================================
-- Pre-defined field mapping templates for common import types

CREATE TABLE IF NOT EXISTS import_mapping_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Template identification
    source_type import_source_type NOT NULL,
    target_module VARCHAR(50) NOT NULL,
    template_name VARCHAR(100) NOT NULL,

    -- Mapping definition
    field_mappings JSONB NOT NULL DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    default_values JSONB DEFAULT '{}',

    -- Template metadata
    description TEXT,
    is_system_template BOOLEAN DEFAULT FALSE,  -- Built-in vs user-created
    created_by VARCHAR(255),                   -- NULL for system templates

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(source_type, target_module, template_name)
);

-- ============================================================================
-- CONVERSATION IMPORTS TABLE
-- ============================================================================
-- Stores imported conversations from ChatGPT, Claude, etc.

CREATE TABLE IF NOT EXISTS imported_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,

    -- Source info
    source_type import_source_type NOT NULL,
    external_conversation_id VARCHAR(255),

    -- Conversation data
    title VARCHAR(500),
    model VARCHAR(100),                       -- gpt-4, claude-3-opus, etc.

    -- Messages are stored as JSONB array
    -- Format: [{"role": "user/assistant/system", "content": "...", "timestamp": "..."}]
    messages JSONB NOT NULL DEFAULT '[]',
    message_count INT DEFAULT 0,

    -- Metadata
    original_created_at TIMESTAMPTZ,          -- When created in source system
    original_updated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',              -- Source-specific metadata

    -- BusinessOS integration
    linked_context_id UUID,                   -- Link to contexts table
    linked_project_id UUID,                   -- Link to projects
    tags TEXT[] DEFAULT '{}',

    -- Full-text search
    search_content TEXT,                      -- Concatenated text for search

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_imported_conversations_user ON imported_conversations(user_id);
CREATE INDEX idx_imported_conversations_source ON imported_conversations(source_type);
CREATE INDEX idx_imported_conversations_job ON imported_conversations(import_job_id);
CREATE INDEX idx_imported_conversations_search ON imported_conversations USING GIN(to_tsvector('english', search_content));

-- ============================================================================
-- INSERT SYSTEM MAPPING TEMPLATES
-- ============================================================================

-- ChatGPT to Conversations template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'chatgpt_export',
    'conversations',
    'default',
    '{
        "title": "title",
        "create_time": "original_created_at",
        "update_time": "original_updated_at",
        "mapping.messages": "messages"
    }',
    '{
        "create_time": {"type": "timestamp", "format": "unix"},
        "update_time": {"type": "timestamp", "format": "unix"},
        "messages": {"type": "chatgpt_messages"}
    }',
    'Default mapping for ChatGPT conversation exports',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- Claude to Conversations template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'claude_export',
    'conversations',
    'default',
    '{
        "name": "title",
        "created_at": "original_created_at",
        "updated_at": "original_updated_at",
        "chat_messages": "messages"
    }',
    '{
        "created_at": {"type": "timestamp", "format": "iso8601"},
        "updated_at": {"type": "timestamp", "format": "iso8601"},
        "chat_messages": {"type": "claude_messages"}
    }',
    'Default mapping for Claude conversation exports',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- HubSpot Contacts to Clients template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'hubspot_contacts',
    'clients',
    'default',
    '{
        "properties.firstname": "first_name",
        "properties.lastname": "last_name",
        "properties.email": "email",
        "properties.phone": "phone",
        "properties.company": "company_name",
        "properties.jobtitle": "job_title",
        "properties.hs_object_id": "external_id",
        "properties.createdate": "original_created_at",
        "properties.lastmodifieddate": "original_updated_at"
    }',
    '{
        "createdate": {"type": "timestamp", "format": "unix_ms"},
        "lastmodifieddate": {"type": "timestamp", "format": "unix_ms"}
    }',
    'Default mapping for HubSpot contacts to BusinessOS clients',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE import_jobs IS 'Tracks all data import operations with status and progress';
COMMENT ON TABLE imported_records IS 'Links external record IDs to BusinessOS records for deduplication';
COMMENT ON TABLE import_mapping_templates IS 'Pre-defined field mapping templates for common import types';
COMMENT ON TABLE imported_conversations IS 'Stores imported conversations from ChatGPT, Claude, etc.';
-- Migration 029: Add unique constraint to calendar_events for Google event deduplication
-- Required for ON CONFLICT upsert to work properly during calendar sync

-- Add unique constraint for (user_id, google_event_id) if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'calendar_events_user_id_google_event_id_key'
    ) THEN
        ALTER TABLE calendar_events
        ADD CONSTRAINT calendar_events_user_id_google_event_id_key
        UNIQUE (user_id, google_event_id);
    END IF;
END $$;
-- Migration 030: Emails and Channels for Communication Hub
-- This adds support for Gmail sync and Slack/Discord channels

-- Emails table for Gmail integration
CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'gmail',
    external_id VARCHAR(255) NOT NULL,
    thread_id VARCHAR(255),

    -- Email metadata
    subject TEXT,
    snippet TEXT,
    from_email VARCHAR(255),
    from_name VARCHAR(255),
    to_emails JSONB DEFAULT '[]',
    cc_emails JSONB DEFAULT '[]',
    bcc_emails JSONB DEFAULT '[]',
    reply_to VARCHAR(255),

    -- Content
    body_text TEXT,
    body_html TEXT,
    attachments JSONB DEFAULT '[]',

    -- Status flags
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_important BOOLEAN DEFAULT FALSE,
    is_draft BOOLEAN DEFAULT FALSE,
    is_sent BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_trash BOOLEAN DEFAULT FALSE,
    labels JSONB DEFAULT '[]',

    -- Dates
    date TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, provider, external_id)
);

-- Indexes for email queries
CREATE INDEX IF NOT EXISTS idx_emails_user_thread ON emails(user_id, thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_user_date ON emails(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_user_unread ON emails(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_emails_user_starred ON emails(user_id, is_starred) WHERE is_starred = TRUE;
CREATE INDEX IF NOT EXISTS idx_emails_user_provider ON emails(user_id, provider);

-- Channels table for Slack/Discord/Teams
CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- 'slack', 'discord', 'teams'
    external_id VARCHAR(255) NOT NULL,
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),

    -- Channel metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    topic TEXT,
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_dm BOOLEAN DEFAULT FALSE,
    member_count INT DEFAULT 0,
    unread_count INT DEFAULT 0,

    -- Dates
    last_message_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, provider, external_id)
);

-- Channel messages table
CREATE TABLE IF NOT EXISTS channel_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL,

    -- Sender info
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    sender_avatar VARCHAR(500),

    -- Message content
    content TEXT,
    content_html TEXT,
    attachments JSONB DEFAULT '[]',
    reactions JSONB DEFAULT '[]',
    mentions JSONB DEFAULT '[]',

    -- Thread info
    thread_ts VARCHAR(50),
    parent_message_id UUID REFERENCES channel_messages(id),
    reply_count INT DEFAULT 0,
    is_thread_root BOOLEAN DEFAULT FALSE,

    -- Status
    is_edited BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,

    -- Dates
    sent_at TIMESTAMP WITH TIME ZONE,
    edited_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(channel_id, external_id)
);

-- Indexes for channel queries
CREATE INDEX IF NOT EXISTS idx_channels_user_provider ON channels(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_channel_messages_channel ON channel_messages(channel_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_channel_messages_thread ON channel_messages(channel_id, thread_ts);

-- Integration sync log for tracking sync history
CREATE TABLE IF NOT EXISTS integration_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    sync_type VARCHAR(50) NOT NULL, -- 'calendar', 'email', 'channels', 'messages'
    status VARCHAR(50) NOT NULL, -- 'started', 'completed', 'failed', 'partial'

    -- Sync details
    records_synced INT DEFAULT 0,
    records_failed INT DEFAULT 0,
    error_message TEXT,

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INT,

    -- Metadata
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_sync_log_user_provider ON integration_sync_log(user_id, provider_id, started_at DESC);

-- Add sync stats columns to user_integrations if they don't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_integrations' AND column_name = 'total_records_synced') THEN
        ALTER TABLE user_integrations ADD COLUMN total_records_synced INT DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_integrations' AND column_name = 'last_sync_status') THEN
        ALTER TABLE user_integrations ADD COLUMN last_sync_status VARCHAR(50);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_integrations' AND column_name = 'last_sync_duration_ms') THEN
        ALTER TABLE user_integrations ADD COLUMN last_sync_duration_ms INT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_integrations' AND column_name = 'sync_stats') THEN
        ALTER TABLE user_integrations ADD COLUMN sync_stats JSONB DEFAULT '{}';
    END IF;
END $$;
-- Migration 031: Notion Integration Tables
-- Tables for storing synced Notion databases and pages

-- Notion databases table
CREATE TABLE IF NOT EXISTS notion_databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    notion_id VARCHAR(255) NOT NULL,

    -- Database metadata
    title VARCHAR(500),
    description TEXT,
    icon VARCHAR(500),
    cover VARCHAR(500),
    url VARCHAR(500),

    -- Properties schema (stored as JSONB)
    properties JSONB DEFAULT '{}',

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, notion_id)
);

-- Indexes for notion databases
CREATE INDEX IF NOT EXISTS idx_notion_databases_user ON notion_databases(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_databases_title ON notion_databases(user_id, title);

-- Notion pages table (entries in databases or standalone pages)
CREATE TABLE IF NOT EXISTS notion_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    notion_id VARCHAR(255) NOT NULL,
    database_id UUID REFERENCES notion_databases(id) ON DELETE SET NULL,

    -- Page metadata
    title VARCHAR(500),
    icon VARCHAR(500),
    cover VARCHAR(500),
    url VARCHAR(500),
    archived BOOLEAN DEFAULT FALSE,

    -- Properties (from database schema)
    properties JSONB DEFAULT '{}',

    -- Content (optional - for full page sync)
    content JSONB DEFAULT '[]',  -- Array of blocks

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, notion_id)
);

-- Indexes for notion pages
CREATE INDEX IF NOT EXISTS idx_notion_pages_user ON notion_pages(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_pages_database ON notion_pages(database_id);
CREATE INDEX IF NOT EXISTS idx_notion_pages_title ON notion_pages(user_id, title);
CREATE INDEX IF NOT EXISTS idx_notion_pages_archived ON notion_pages(user_id, archived);

-- Slack channels table (if not using the generic channels table)
-- This provides Slack-specific fields
CREATE TABLE IF NOT EXISTS slack_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    slack_id VARCHAR(255) NOT NULL,

    -- Channel metadata
    name VARCHAR(255) NOT NULL,
    is_private BOOLEAN DEFAULT FALSE,
    is_dm BOOLEAN DEFAULT FALSE,
    is_mpim BOOLEAN DEFAULT FALSE,  -- Multi-person IM
    member_count INT DEFAULT 0,
    topic TEXT,
    purpose TEXT,
    unread_count INT DEFAULT 0,

    -- Activity tracking
    last_activity TIMESTAMP WITH TIME ZONE,

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, slack_id)
);

-- Indexes for slack channels
CREATE INDEX IF NOT EXISTS idx_slack_channels_user ON slack_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_slack_channels_name ON slack_channels(user_id, name);
CREATE INDEX IF NOT EXISTS idx_slack_channels_activity ON slack_channels(user_id, last_activity DESC);

-- Slack messages table
CREATE TABLE IF NOT EXISTS slack_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    channel_id UUID NOT NULL REFERENCES slack_channels(id) ON DELETE CASCADE,
    slack_ts VARCHAR(50) NOT NULL,  -- Slack timestamp (unique ID)

    -- Sender info
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),

    -- Message content
    content TEXT,
    thread_ts VARCHAR(50),  -- Parent thread timestamp
    reply_count INT DEFAULT 0,
    is_edited BOOLEAN DEFAULT FALSE,

    -- Timestamps
    sent_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, channel_id, slack_ts)
);

-- Indexes for slack messages
CREATE INDEX IF NOT EXISTS idx_slack_messages_channel ON slack_messages(channel_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_slack_messages_thread ON slack_messages(channel_id, thread_ts);
CREATE INDEX IF NOT EXISTS idx_slack_messages_sender ON slack_messages(sender_id);

-- Comments
COMMENT ON TABLE notion_databases IS 'Synced Notion databases with their property schemas';
COMMENT ON TABLE notion_pages IS 'Synced Notion pages/database entries';
COMMENT ON TABLE slack_channels IS 'Synced Slack channels, DMs, and group messages';
COMMENT ON TABLE slack_messages IS 'Synced Slack messages from channels';
-- Linear Integration Tables
-- Migration: 032_linear_tables.sql

-- Linear issues synced from Linear API
CREATE TABLE IF NOT EXISTS linear_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    identifier TEXT NOT NULL, -- e.g., "ENG-123"
    title TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL DEFAULT 'backlog',
    priority INTEGER DEFAULT 0,
    assignee TEXT,
    project TEXT,
    team TEXT NOT NULL,
    due_date DATE,
    external_created_at TIMESTAMPTZ,
    external_updated_at TIMESTAMPTZ,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

-- Linear projects synced from Linear API
CREATE TABLE IF NOT EXISTS linear_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL DEFAULT 'planned',
    progress DECIMAL(5,2) DEFAULT 0,
    start_date DATE,
    target_date DATE,
    team TEXT,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

-- Linear teams synced from Linear API
CREATE TABLE IF NOT EXISTS linear_teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    key TEXT NOT NULL, -- e.g., "ENG"
    name TEXT NOT NULL,
    description TEXT,
    issue_count INTEGER DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_linear_issues_user ON linear_issues(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_issues_identifier ON linear_issues(identifier);
CREATE INDEX IF NOT EXISTS idx_linear_issues_state ON linear_issues(user_id, state);
CREATE INDEX IF NOT EXISTS idx_linear_issues_team ON linear_issues(user_id, team);
CREATE INDEX IF NOT EXISTS idx_linear_issues_updated ON linear_issues(external_updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_linear_projects_user ON linear_projects(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_projects_state ON linear_projects(user_id, state);

CREATE INDEX IF NOT EXISTS idx_linear_teams_user ON linear_teams(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_teams_key ON linear_teams(user_id, key);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_linear_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS linear_issues_updated_at ON linear_issues;
CREATE TRIGGER linear_issues_updated_at
    BEFORE UPDATE ON linear_issues
    FOR EACH ROW EXECUTE FUNCTION update_linear_updated_at();

DROP TRIGGER IF EXISTS linear_projects_updated_at ON linear_projects;
CREATE TRIGGER linear_projects_updated_at
    BEFORE UPDATE ON linear_projects
    FOR EACH ROW EXECUTE FUNCTION update_linear_updated_at();

DROP TRIGGER IF EXISTS linear_teams_updated_at ON linear_teams;
CREATE TRIGGER linear_teams_updated_at
    BEFORE UPDATE ON linear_teams
    FOR EACH ROW EXECUTE FUNCTION update_linear_updated_at();
-- Migration 033: Fathom Analytics and Google Docs Integration Tables
-- This adds storage for Fathom analytics data and Google Docs/Drive content

-- ============================================================================
-- FATHOM ANALYTICS TABLES
-- ============================================================================

-- Fathom sites (website properties)
CREATE TABLE IF NOT EXISTS fathom_sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    name VARCHAR(255),
    sharing_url TEXT,
    share_config VARCHAR(50),
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id)
);

CREATE INDEX IF NOT EXISTS idx_fathom_sites_user ON fathom_sites(user_id);

-- Fathom aggregations (daily analytics data)
CREATE TABLE IF NOT EXISTS fathom_aggregations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    pageviews INT DEFAULT 0,
    avg_duration DECIMAL(10,2) DEFAULT 0,
    bounce_rate DECIMAL(5,2) DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, date)
);

CREATE INDEX IF NOT EXISTS idx_fathom_agg_user_site ON fathom_aggregations(user_id, site_id);
CREATE INDEX IF NOT EXISTS idx_fathom_agg_date ON fathom_aggregations(user_id, site_id, date DESC);

-- Fathom page-level analytics (grouped by pathname)
CREATE TABLE IF NOT EXISTS fathom_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    pathname VARCHAR(500) NOT NULL,
    hostname VARCHAR(255),
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    pageviews INT DEFAULT 0,
    avg_duration DECIMAL(10,2) DEFAULT 0,
    bounce_rate DECIMAL(5,2) DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, pathname, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_pages_user_site ON fathom_pages(user_id, site_id);
CREATE INDEX IF NOT EXISTS idx_fathom_pages_pathname ON fathom_pages(user_id, pathname);

-- Fathom referrers analytics
CREATE TABLE IF NOT EXISTS fathom_referrers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    referrer VARCHAR(500) NOT NULL,
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, referrer, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_referrers_user_site ON fathom_referrers(user_id, site_id);

-- Fathom custom events
CREATE TABLE IF NOT EXISTS fathom_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    event_id VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    count INT DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, event_id, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_events_user_site ON fathom_events(user_id, site_id);

-- ============================================================================
-- GOOGLE DRIVE/DOCS TABLES
-- ============================================================================

-- Google Drive files
CREATE TABLE IF NOT EXISTS google_drive_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    file_id VARCHAR(255) NOT NULL,

    -- File metadata
    name VARCHAR(500) NOT NULL,
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),
    size_bytes BIGINT,

    -- Folder/hierarchy info
    parent_folder_id VARCHAR(255),
    parent_folder_name VARCHAR(500),
    path TEXT,

    -- Permissions/sharing
    shared BOOLEAN DEFAULT FALSE,
    sharing_user VARCHAR(255),
    permissions JSONB DEFAULT '[]',

    -- Content info
    web_view_link TEXT,
    web_content_link TEXT,
    thumbnail_link TEXT,
    icon_link TEXT,

    -- Timestamps from Google
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
    viewed_by_me_time TIMESTAMPTZ,

    -- Owners and modifiers
    owners JSONB DEFAULT '[]',
    last_modifying_user JSONB,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_drive_files_user ON google_drive_files(user_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_parent ON google_drive_files(user_id, parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_mime ON google_drive_files(user_id, mime_type);
CREATE INDEX IF NOT EXISTS idx_drive_files_modified ON google_drive_files(user_id, modified_time DESC);

-- Google Docs content (extracted from Drive for document-specific data)
CREATE TABLE IF NOT EXISTS google_docs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    document_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Document metadata
    title VARCHAR(500) NOT NULL,

    -- Content (plain text extraction for search)
    body_text TEXT,
    word_count INT DEFAULT 0,

    -- Document structure
    headers JSONB DEFAULT '[]',

    -- Document info
    locale VARCHAR(20),

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_docs_user ON google_docs(user_id);
CREATE INDEX IF NOT EXISTS idx_docs_title ON google_docs(user_id, title);
CREATE INDEX IF NOT EXISTS idx_docs_modified ON google_docs(user_id, modified_time DESC);
CREATE INDEX IF NOT EXISTS idx_docs_search ON google_docs USING GIN(to_tsvector('english', body_text));

-- Google Sheets
CREATE TABLE IF NOT EXISTS google_sheets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    spreadsheet_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Spreadsheet metadata
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),
    time_zone VARCHAR(100),

    -- Sheet info
    sheet_count INT DEFAULT 0,
    sheets JSONB DEFAULT '[]', -- Array of sheet names and properties

    -- Named ranges
    named_ranges JSONB DEFAULT '[]',

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, spreadsheet_id)
);

CREATE INDEX IF NOT EXISTS idx_sheets_user ON google_sheets(user_id);
CREATE INDEX IF NOT EXISTS idx_sheets_title ON google_sheets(user_id, title);

-- Google Slides presentations
CREATE TABLE IF NOT EXISTS google_slides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    presentation_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Presentation metadata
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),

    -- Slide info
    slide_count INT DEFAULT 0,
    slides JSONB DEFAULT '[]', -- Array of slide info

    -- Page size
    page_width DECIMAL(10,2),
    page_height DECIMAL(10,2),

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, presentation_id)
);

CREATE INDEX IF NOT EXISTS idx_slides_user ON google_slides(user_id);
CREATE INDEX IF NOT EXISTS idx_slides_title ON google_slides(user_id, title);

-- ============================================================================
-- GOOGLE CONTACTS TABLES
-- ============================================================================

-- Google contacts
CREATE TABLE IF NOT EXISTS google_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL, -- people/c12345

    -- Name
    display_name VARCHAR(255),
    given_name VARCHAR(255),
    family_name VARCHAR(255),
    middle_name VARCHAR(255),

    -- Contact info
    emails JSONB DEFAULT '[]',
    phone_numbers JSONB DEFAULT '[]',
    addresses JSONB DEFAULT '[]',

    -- Organization
    organization VARCHAR(255),
    job_title VARCHAR(255),
    department VARCHAR(255),

    -- Photo
    photo_url TEXT,

    -- Grouping
    contact_groups JSONB DEFAULT '[]',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, resource_name)
);

CREATE INDEX IF NOT EXISTS idx_contacts_user ON google_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON google_contacts(user_id, display_name);
CREATE INDEX IF NOT EXISTS idx_contacts_org ON google_contacts(user_id, organization);

-- ============================================================================
-- GOOGLE TASKS TABLES
-- ============================================================================

-- Google task lists
CREATE TABLE IF NOT EXISTS google_task_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,

    -- List info
    title VARCHAR(255) NOT NULL,
    kind VARCHAR(100),

    -- Timestamps
    updated TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_list_id)
);

CREATE INDEX IF NOT EXISTS idx_task_lists_user ON google_task_lists(user_id);

-- Google tasks
CREATE TABLE IF NOT EXISTS google_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,

    -- Task info
    title VARCHAR(500) NOT NULL,
    notes TEXT,
    status VARCHAR(50) DEFAULT 'needsAction', -- needsAction, completed

    -- Due date
    due TIMESTAMPTZ,

    -- Completion
    completed TIMESTAMPTZ,
    deleted BOOLEAN DEFAULT FALSE,
    hidden BOOLEAN DEFAULT FALSE,

    -- Hierarchy
    parent_task_id VARCHAR(255),
    position VARCHAR(100),

    -- Links
    links JSONB DEFAULT '[]',

    -- Timestamps
    updated TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_google_tasks_user ON google_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_google_tasks_list ON google_tasks(user_id, task_list_id);
CREATE INDEX IF NOT EXISTS idx_google_tasks_status ON google_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_google_tasks_due ON google_tasks(user_id, due);

-- ============================================================================
-- HUBSPOT CRM TABLES (for synced HubSpot data)
-- ============================================================================

-- HubSpot contacts
CREATE TABLE IF NOT EXISTS hubspot_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Contact info
    email VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    phone VARCHAR(100),

    -- Company
    company VARCHAR(255),
    job_title VARCHAR(255),

    -- Lead info
    lifecycle_stage VARCHAR(100),
    lead_status VARCHAR(100),

    -- Owner
    owner_id VARCHAR(100),

    -- Properties (all other HubSpot properties)
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_user ON hubspot_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_email ON hubspot_contacts(user_id, email);

-- HubSpot companies
CREATE TABLE IF NOT EXISTS hubspot_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Company info
    name VARCHAR(500),
    domain VARCHAR(255),
    industry VARCHAR(255),

    -- Company size
    number_of_employees INT,
    annual_revenue DECIMAL(15,2),

    -- Location
    city VARCHAR(255),
    state VARCHAR(255),
    country VARCHAR(255),

    -- Owner
    owner_id VARCHAR(100),

    -- Properties
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_companies_user ON hubspot_companies(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_companies_name ON hubspot_companies(user_id, name);

-- HubSpot deals
CREATE TABLE IF NOT EXISTS hubspot_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Deal info
    deal_name VARCHAR(500),
    amount DECIMAL(15,2),
    pipeline VARCHAR(255),
    deal_stage VARCHAR(255),

    -- Close date
    close_date DATE,

    -- Owner
    owner_id VARCHAR(100),

    -- Associated objects
    associated_company_ids JSONB DEFAULT '[]',
    associated_contact_ids JSONB DEFAULT '[]',

    -- Properties
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_deals_user ON hubspot_deals(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_stage ON hubspot_deals(user_id, deal_stage);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_close ON hubspot_deals(user_id, close_date);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update timestamp triggers
CREATE OR REPLACE FUNCTION update_integration_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers to new tables
DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'fathom_sites', 'fathom_pages', 'fathom_referrers', 'fathom_events',
        'google_drive_files', 'google_docs', 'google_sheets', 'google_slides',
        'google_contacts', 'google_task_lists', 'google_tasks',
        'hubspot_contacts', 'hubspot_companies', 'hubspot_deals'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_integration_updated_at()', t, t);
    END LOOP;
END $$;

-- Add comments
COMMENT ON TABLE fathom_sites IS 'Fathom Analytics website properties';
COMMENT ON TABLE fathom_aggregations IS 'Daily aggregated analytics data from Fathom';
COMMENT ON TABLE google_drive_files IS 'Synced Google Drive files metadata';
COMMENT ON TABLE google_docs IS 'Synced Google Docs with text extraction';
COMMENT ON TABLE google_sheets IS 'Synced Google Sheets metadata';
COMMENT ON TABLE google_slides IS 'Synced Google Slides presentations';
COMMENT ON TABLE google_contacts IS 'Synced Google Contacts';
COMMENT ON TABLE google_tasks IS 'Synced Google Tasks';
COMMENT ON TABLE hubspot_contacts IS 'Synced HubSpot CRM contacts';
COMMENT ON TABLE hubspot_companies IS 'Synced HubSpot CRM companies';
COMMENT ON TABLE hubspot_deals IS 'Synced HubSpot CRM deals';
-- Migration 034: ClickUp and Airtable Integration Tables
-- This adds storage for ClickUp project management and Airtable database data

-- ============================================================================
-- CLICKUP TABLES
-- ============================================================================

-- ClickUp workspaces (called Teams in API v2)
CREATE TABLE IF NOT EXISTS clickup_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    avatar TEXT,
    member_count INT DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_workspaces_user ON clickup_workspaces(user_id);

-- ClickUp spaces
CREATE TABLE IF NOT EXISTS clickup_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    space_id VARCHAR(100) NOT NULL,
    workspace_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    private BOOLEAN DEFAULT FALSE,
    archived BOOLEAN DEFAULT FALSE,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, space_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_spaces_user ON clickup_spaces(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_spaces_workspace ON clickup_spaces(user_id, workspace_id);

-- ClickUp folders
CREATE TABLE IF NOT EXISTS clickup_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    folder_id VARCHAR(100) NOT NULL,
    space_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    hidden BOOLEAN DEFAULT FALSE,
    archived BOOLEAN DEFAULT FALSE,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, folder_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_folders_user ON clickup_folders(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_folders_space ON clickup_folders(user_id, space_id);

-- ClickUp lists
CREATE TABLE IF NOT EXISTS clickup_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(100) NOT NULL,
    folder_id VARCHAR(100),
    space_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    archived BOOLEAN DEFAULT FALSE,
    task_count INT DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, list_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_lists_user ON clickup_lists(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_lists_folder ON clickup_lists(user_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_clickup_lists_space ON clickup_lists(user_id, space_id);

-- ClickUp tasks
CREATE TABLE IF NOT EXISTS clickup_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(100) NOT NULL,
    custom_id VARCHAR(100),
    list_id VARCHAR(100) NOT NULL,
    folder_id VARCHAR(100),
    space_id VARCHAR(100) NOT NULL,

    -- Task details
    name VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(100),
    status_color VARCHAR(50),
    priority VARCHAR(50),
    priority_color VARCHAR(50),

    -- Dates
    due_date TIMESTAMPTZ,
    start_date TIMESTAMPTZ,
    date_created TIMESTAMPTZ,
    date_updated TIMESTAMPTZ,
    date_closed TIMESTAMPTZ,

    -- Time tracking
    time_spent BIGINT DEFAULT 0,
    time_estimate BIGINT,

    -- Hierarchy
    parent_task_id VARCHAR(100),

    -- Assignments
    assignees JSONB DEFAULT '[]',
    creator JSONB,
    tags JSONB DEFAULT '[]',

    -- URLs
    url TEXT,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_tasks_user ON clickup_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_list ON clickup_tasks(user_id, list_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_space ON clickup_tasks(user_id, space_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_status ON clickup_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_due ON clickup_tasks(user_id, due_date);

-- ============================================================================
-- AIRTABLE TABLES
-- ============================================================================

-- Airtable bases
CREATE TABLE IF NOT EXISTS airtable_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permission_level VARCHAR(50), -- none, read, comment, edit, create
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, base_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_bases_user ON airtable_bases(user_id);

-- Airtable tables (within bases)
CREATE TABLE IF NOT EXISTS airtable_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    table_id VARCHAR(100) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    primary_field_id VARCHAR(100),
    fields JSONB DEFAULT '[]',
    views JSONB DEFAULT '[]',
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, table_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_tables_user ON airtable_tables(user_id);
CREATE INDEX IF NOT EXISTS idx_airtable_tables_base ON airtable_tables(user_id, base_id);

-- Airtable records
CREATE TABLE IF NOT EXISTS airtable_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    record_id VARCHAR(100) NOT NULL,
    table_id VARCHAR(100) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    fields JSONB DEFAULT '{}',
    created_time_airtable TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, record_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_records_user ON airtable_records(user_id);
CREATE INDEX IF NOT EXISTS idx_airtable_records_table ON airtable_records(user_id, table_id);
CREATE INDEX IF NOT EXISTS idx_airtable_records_base ON airtable_records(user_id, base_id);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Apply update triggers to new tables
DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'clickup_workspaces', 'clickup_spaces', 'clickup_folders', 'clickup_lists', 'clickup_tasks',
        'airtable_bases', 'airtable_tables', 'airtable_records'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_integration_updated_at()', t, t);
    END LOOP;
END $$;

-- Add comments
COMMENT ON TABLE clickup_workspaces IS 'ClickUp workspaces (Teams in API v2)';
COMMENT ON TABLE clickup_spaces IS 'ClickUp spaces within workspaces';
COMMENT ON TABLE clickup_folders IS 'ClickUp folders within spaces';
COMMENT ON TABLE clickup_lists IS 'ClickUp lists (task containers)';
COMMENT ON TABLE clickup_tasks IS 'ClickUp tasks with full metadata';
COMMENT ON TABLE airtable_bases IS 'Airtable bases (workspaces)';
COMMENT ON TABLE airtable_tables IS 'Airtable tables within bases';
COMMENT ON TABLE airtable_records IS 'Airtable records (rows) with field data';
-- Migration 035: Microsoft 365 Integration Tables
-- This adds storage for Microsoft 365 data (Outlook, OneDrive, To Do, etc.)

-- ============================================================================
-- MICROSOFT OAUTH TOKENS
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expiry TIMESTAMPTZ,
    scopes TEXT[],
    microsoft_id VARCHAR(255),
    microsoft_email VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_microsoft_tokens_user ON microsoft_oauth_tokens(user_id);

-- ============================================================================
-- MICROSOFT OUTLOOK MAIL
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_mail_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,

    -- Conversation threading
    conversation_id VARCHAR(255),

    -- Message details
    subject TEXT,
    body_preview TEXT,
    body_content TEXT,
    body_content_type VARCHAR(50), -- text, html
    importance VARCHAR(50), -- low, normal, high

    -- Sender
    from_email VARCHAR(255),
    from_name VARCHAR(255),

    -- Recipients
    to_recipients JSONB DEFAULT '[]',
    cc_recipients JSONB DEFAULT '[]',
    bcc_recipients JSONB DEFAULT '[]',
    reply_to JSONB DEFAULT '[]',

    -- Flags
    is_read BOOLEAN DEFAULT FALSE,
    is_draft BOOLEAN DEFAULT FALSE,
    has_attachments BOOLEAN DEFAULT FALSE,

    -- Folder
    folder_id VARCHAR(255),
    folder_name VARCHAR(255),

    -- Categories/Labels
    categories JSONB DEFAULT '[]',
    flag_status VARCHAR(50), -- notFlagged, flagged, complete

    -- Attachments metadata
    attachments JSONB DEFAULT '[]',

    -- Timestamps
    received_datetime TIMESTAMPTZ,
    sent_datetime TIMESTAMPTZ,
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_mail_user ON microsoft_mail_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_conversation ON microsoft_mail_messages(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_folder ON microsoft_mail_messages(user_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_received ON microsoft_mail_messages(user_id, received_datetime DESC);
CREATE INDEX IF NOT EXISTS idx_ms_mail_from ON microsoft_mail_messages(user_id, from_email);

-- ============================================================================
-- MICROSOFT OUTLOOK CALENDAR
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    event_id VARCHAR(255) NOT NULL,

    -- Calendar
    calendar_id VARCHAR(255),
    calendar_name VARCHAR(255),

    -- Event details
    subject VARCHAR(500),
    body_preview TEXT,
    body_content TEXT,
    body_content_type VARCHAR(50),

    -- Location
    location_display_name VARCHAR(500),
    location_address JSONB,
    location_coordinates JSONB, -- lat, long

    -- Time
    start_datetime TIMESTAMPTZ,
    start_timezone VARCHAR(100),
    end_datetime TIMESTAMPTZ,
    end_timezone VARCHAR(100),
    is_all_day BOOLEAN DEFAULT FALSE,

    -- Recurrence
    recurrence JSONB, -- pattern, range
    series_master_id VARCHAR(255),
    type VARCHAR(50), -- singleInstance, occurrence, exception, seriesMaster

    -- Attendees
    attendees JSONB DEFAULT '[]',
    organizer_email VARCHAR(255),
    organizer_name VARCHAR(255),

    -- Online meeting
    is_online_meeting BOOLEAN DEFAULT FALSE,
    online_meeting_provider VARCHAR(100), -- teamsForBusiness, skypeForBusiness, etc.
    online_meeting_url TEXT,
    online_meeting_join_url TEXT,

    -- Response
    response_status VARCHAR(50), -- none, organizer, accepted, tentative, declined
    response_time TIMESTAMPTZ,

    -- Flags
    importance VARCHAR(50),
    sensitivity VARCHAR(50), -- normal, personal, private, confidential
    show_as VARCHAR(50), -- free, tentative, busy, oof, workingElsewhere
    is_cancelled BOOLEAN DEFAULT FALSE,
    is_reminder_on BOOLEAN DEFAULT TRUE,
    reminder_minutes_before_start INT DEFAULT 15,

    -- Categories
    categories JSONB DEFAULT '[]',

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_calendar_user ON microsoft_calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_calendar ON microsoft_calendar_events(user_id, calendar_id);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_start ON microsoft_calendar_events(user_id, start_datetime);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_series ON microsoft_calendar_events(user_id, series_master_id);

-- ============================================================================
-- MICROSOFT CONTACTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    contact_id VARCHAR(255) NOT NULL,

    -- Name
    display_name VARCHAR(255),
    given_name VARCHAR(255),
    surname VARCHAR(255),
    middle_name VARCHAR(255),
    nickname VARCHAR(255),
    title VARCHAR(50), -- Mr., Ms., Dr., etc.

    -- Contact info
    email_addresses JSONB DEFAULT '[]',
    phone_numbers JSONB DEFAULT '[]',
    addresses JSONB DEFAULT '[]',
    im_addresses JSONB DEFAULT '[]',
    websites JSONB DEFAULT '[]',

    -- Organization
    company_name VARCHAR(255),
    department VARCHAR(255),
    job_title VARCHAR(255),
    office_location VARCHAR(255),
    profession VARCHAR(255),
    manager VARCHAR(255),
    assistant_name VARCHAR(255),

    -- Personal
    birthday DATE,
    spouse_name VARCHAR(255),
    personal_notes TEXT,

    -- Photo
    photo_url TEXT,

    -- Category
    categories JSONB DEFAULT '[]',

    -- Parent folder
    parent_folder_id VARCHAR(255),

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, contact_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_contacts_user ON microsoft_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_contacts_name ON microsoft_contacts(user_id, display_name);
CREATE INDEX IF NOT EXISTS idx_ms_contacts_company ON microsoft_contacts(user_id, company_name);

-- ============================================================================
-- MICROSOFT ONEDRIVE
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_onedrive_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    item_id VARCHAR(255) NOT NULL,

    -- File info
    name VARCHAR(500) NOT NULL,
    description TEXT,
    mime_type VARCHAR(255),
    size_bytes BIGINT,

    -- Path info
    parent_reference_id VARCHAR(255),
    parent_reference_path TEXT,
    web_url TEXT,

    -- Type
    is_folder BOOLEAN DEFAULT FALSE,
    folder_child_count INT,

    -- File specific
    file_hash VARCHAR(255), -- quickXorHash or sha1Hash

    -- Sharing
    shared BOOLEAN DEFAULT FALSE,
    shared_scope VARCHAR(50), -- anonymous, organization, users
    shared_link JSONB,

    -- Permissions
    permissions JSONB DEFAULT '[]',

    -- Created/Modified by
    created_by_user_email VARCHAR(255),
    created_by_user_name VARCHAR(255),
    last_modified_by_user_email VARCHAR(255),
    last_modified_by_user_name VARCHAR(255),

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Download
    download_url TEXT,

    -- Thumbnails
    thumbnails JSONB DEFAULT '[]',

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_onedrive_user ON microsoft_onedrive_files(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_parent ON microsoft_onedrive_files(user_id, parent_reference_id);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_folder ON microsoft_onedrive_files(user_id, is_folder);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_name ON microsoft_onedrive_files(user_id, name);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_modified ON microsoft_onedrive_files(user_id, last_modified_datetime DESC);

-- ============================================================================
-- MICROSOFT TO DO
-- ============================================================================

-- Task Lists
CREATE TABLE IF NOT EXISTS microsoft_todo_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(255) NOT NULL,

    -- List info
    display_name VARCHAR(255) NOT NULL,
    is_owner BOOLEAN DEFAULT TRUE,
    is_shared BOOLEAN DEFAULT FALSE,
    wellknown_list_name VARCHAR(50), -- none, defaultList, flaggedEmails, unknownFutureValue

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, list_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_todo_lists_user ON microsoft_todo_lists(user_id);

-- Tasks
CREATE TABLE IF NOT EXISTS microsoft_todo_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(255) NOT NULL,

    -- Task details
    title VARCHAR(500) NOT NULL,
    body_content TEXT,
    body_content_type VARCHAR(50),
    importance VARCHAR(50), -- low, normal, high
    status VARCHAR(50), -- notStarted, inProgress, completed, waitingOnOthers, deferred

    -- Dates
    due_datetime TIMESTAMPTZ,
    due_timezone VARCHAR(100),
    start_datetime TIMESTAMPTZ,
    start_timezone VARCHAR(100),
    completed_datetime TIMESTAMPTZ,
    completed_timezone VARCHAR(100),

    -- Recurrence
    recurrence JSONB,
    is_reminder_on BOOLEAN DEFAULT FALSE,
    reminder_datetime TIMESTAMPTZ,

    -- Categories
    categories JSONB DEFAULT '[]',

    -- Linked resources
    linked_resources JSONB DEFAULT '[]',

    -- Checklist items
    checklist_items JSONB DEFAULT '[]',

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_user ON microsoft_todo_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_list ON microsoft_todo_tasks(user_id, list_id);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_status ON microsoft_todo_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_due ON microsoft_todo_tasks(user_id, due_datetime);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'microsoft_oauth_tokens', 'microsoft_mail_messages', 'microsoft_calendar_events',
        'microsoft_contacts', 'microsoft_onedrive_files', 'microsoft_todo_lists', 'microsoft_todo_tasks'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_integration_updated_at()', t, t);
    END LOOP;
END $$;

-- Add comments
COMMENT ON TABLE microsoft_oauth_tokens IS 'Microsoft 365 OAuth tokens for users';
COMMENT ON TABLE microsoft_mail_messages IS 'Synced Outlook email messages';
COMMENT ON TABLE microsoft_calendar_events IS 'Synced Outlook calendar events';
COMMENT ON TABLE microsoft_contacts IS 'Synced Outlook contacts';
COMMENT ON TABLE microsoft_onedrive_files IS 'Synced OneDrive files and folders';
COMMENT ON TABLE microsoft_todo_lists IS 'Microsoft To Do task lists';
COMMENT ON TABLE microsoft_todo_tasks IS 'Microsoft To Do tasks';
