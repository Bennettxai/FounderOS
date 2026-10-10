-- Migration: 101_mcp_servers.sql
-- Description: MCP (Model Context Protocol) server connections for dynamic tool integration
-- Created: 2026-03-08
-- Purpose: Allow users to connect external MCP servers so agents can use external tools

CREATE TABLE IF NOT EXISTS mcp_servers (
    -- Core identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) NOT NULL,

    -- Server config
    name            VARCHAR(100) NOT NULL,
    description     TEXT DEFAULT '',
    server_url      TEXT NOT NULL,
    transport       VARCHAR(20) NOT NULL DEFAULT 'sse',   -- 'sse' | 'streamable_http'
    auth_type       VARCHAR(20) NOT NULL DEFAULT 'none',  -- 'none' | 'api_key' | 'bearer'
    auth_token_enc  TEXT,                                   -- AES-256-GCM encrypted token
    custom_headers  JSONB NOT NULL DEFAULT '{}',

    -- State
    enabled         BOOLEAN NOT NULL DEFAULT true,
    tools_cache     JSONB NOT NULL DEFAULT '[]',           -- Discovered tools from server
    status          VARCHAR(20) NOT NULL DEFAULT 'disconnected', -- 'connected' | 'disconnected' | 'error'
    last_error      TEXT,
    last_connected  TIMESTAMPTZ,

    -- Metadata
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    UNIQUE(user_id, name)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_mcp_servers_user ON mcp_servers(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_enabled ON mcp_servers(user_id, enabled);
