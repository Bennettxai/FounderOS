-- Migration: 026_oauth_tokens_tables.sql
-- Creates OAuth token storage tables for Google, Slack, and Notion integrations
-- Run this migration to ensure all OAuth tables exist with correct schema

-- ============================================
-- Google OAuth Tokens (fix existing table)
-- ============================================
-- Add missing columns to existing google_oauth_tokens table
ALTER TABLE google_oauth_tokens
  ADD COLUMN IF NOT EXISTS google_email VARCHAR(255),
  ADD COLUMN IF NOT EXISTS scopes TEXT[];

-- Migrate old scope data if exists (commented out - scope column doesn't exist in fresh installs)
-- UPDATE google_oauth_tokens SET scopes = ARRAY[scope] WHERE scope IS NOT NULL AND scopes IS NULL;

-- ============================================
-- Slack OAuth Tokens
-- ============================================
CREATE TABLE IF NOT EXISTS slack_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id VARCHAR(255) NOT NULL,
    workspace_name VARCHAR(255),
    bot_token TEXT NOT NULL,
    user_token TEXT,
    bot_user_id VARCHAR(255),
    authed_user_id VARCHAR(255),
    bot_scopes TEXT[],
    user_scopes TEXT[],
    incoming_webhook_url TEXT,
    incoming_webhook_channel VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slack_oauth_user ON slack_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_slack_oauth_workspace ON slack_oauth_tokens(workspace_id);

-- ============================================
-- Notion OAuth Tokens
-- ============================================
CREATE TABLE IF NOT EXISTS notion_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id VARCHAR(255) NOT NULL,
    workspace_name VARCHAR(255),
    workspace_icon TEXT,
    access_token TEXT NOT NULL,
    bot_id VARCHAR(255),
    owner_type VARCHAR(50),
    owner_user_id VARCHAR(255),
    owner_user_name VARCHAR(255),
    owner_user_email VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notion_oauth_user ON notion_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_oauth_workspace ON notion_oauth_tokens(workspace_id);
