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

-- Add missing columns to integration_sync_log if they don't exist (table might have been created by 039)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'integration_sync_log' AND column_name = 'user_id') THEN
        ALTER TABLE integration_sync_log ADD COLUMN user_id VARCHAR(255);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'integration_sync_log' AND column_name = 'provider_id') THEN
        ALTER TABLE integration_sync_log ADD COLUMN provider_id VARCHAR(100);
    END IF;
END $$;

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
