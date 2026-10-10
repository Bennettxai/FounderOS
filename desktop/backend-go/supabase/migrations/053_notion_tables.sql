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
