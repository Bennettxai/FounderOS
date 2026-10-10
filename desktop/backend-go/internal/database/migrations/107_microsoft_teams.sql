-- Migration 107: Microsoft Teams channels and messages.
--
-- Adds two parallel tables for Teams data, mirroring the
-- microsoft_mail_messages shape (schema.sql:1908) instead of folding into
-- the generic `channels` table. Per Leon's spec decision (see
-- docs/design/comms-channels-spec.md §1.3) and Axis's Wave 2 brief —
-- channel-style messaging stays provider-specific so each surface can
-- evolve its own metadata (Slack reactions vs Teams adaptive cards) without
-- a continuous schema migration tax on the unified `channels` table.
--
-- The unified channels endpoint (GET /api/comms/channels) reads from both
-- slack_channels and microsoft_teams_channels and normalizes server-side.

BEGIN;

-- ============================================================================
-- 1. microsoft_teams_channels
-- ============================================================================
--
-- One row per channel per (user, team). team_id + channel_id are Microsoft
-- Graph identifiers. team_name is denormalized for sidebar rendering so
-- the UI doesn't need a second lookup per channel.

CREATE TABLE IF NOT EXISTS microsoft_teams_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,

    -- Team metadata (denormalized for display)
    team_name VARCHAR(255),

    -- Channel metadata
    display_name VARCHAR(255),
    description TEXT,
    email VARCHAR(255),
    web_url TEXT,
    membership_type VARCHAR(50),  -- 'standard' | 'private' | 'shared'
    is_archived BOOLEAN DEFAULT FALSE,
    is_favorite_by_default BOOLEAN DEFAULT FALSE,

    -- Activity / membership (best-effort; not always available without extra calls)
    member_count INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,

    -- Graph timestamps
    created_datetime TIMESTAMPTZ,

    -- Sync tracking
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

-- ============================================================================
-- 2. microsoft_teams_messages
-- ============================================================================
--
-- One row per Teams chatMessage. message_id is the Graph chatMessage id,
-- channel_id and team_id are the Graph identifiers (NOT BO uuids — keeping
-- both is cheaper than a join on every message read). reply_to_id links
-- replies to the thread root.

CREATE TABLE IF NOT EXISTS microsoft_teams_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,

    -- Threading
    reply_to_id VARCHAR(255),  -- root message id when this is a reply

    -- Message metadata
    message_type VARCHAR(50),  -- 'message' | 'chatEvent' | 'systemEventMessage' | etc
    importance VARCHAR(50),    -- 'normal' | 'high' | 'urgent'
    subject TEXT,
    summary TEXT,

    -- Body
    body_content TEXT,
    body_content_type VARCHAR(50),  -- 'text' | 'html'

    -- Sender
    sender_id VARCHAR(255),    -- aadObjectId
    sender_name VARCHAR(255),  -- displayName at time of send
    sender_type VARCHAR(50),   -- 'user' | 'application' | 'unknownFutureValue'

    -- Rich payloads
    attachments JSONB DEFAULT '[]',
    mentions JSONB DEFAULT '[]',
    reactions JSONB DEFAULT '[]',

    -- Status
    web_url TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_edited BOOLEAN DEFAULT FALSE,

    -- Graph timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,
    last_edited_datetime TIMESTAMPTZ,
    deleted_datetime TIMESTAMPTZ,

    -- Sync tracking
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

COMMIT;
