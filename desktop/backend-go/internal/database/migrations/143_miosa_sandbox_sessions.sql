-- BusinessOS MIOSA sandbox session audit.
-- This tracks non-secret sandbox/session metadata created through the
-- BusinessOS-owned MIOSA tenant.

CREATE TABLE IF NOT EXISTS miosa_sandbox_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    miosa_sandbox_id TEXT NOT NULL,
    miosa_workspace_id TEXT,
    external_workspace_id TEXT NOT NULL,
    external_user_id TEXT NOT NULL,
    terminal_session_id TEXT,
    status TEXT NOT NULL DEFAULT 'created',
    preview_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_miosa_sandbox_sessions_workspace
    ON miosa_sandbox_sessions(workspace_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_miosa_sandbox_sessions_user
    ON miosa_sandbox_sessions(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_miosa_sandbox_sessions_sandbox
    ON miosa_sandbox_sessions(miosa_sandbox_id);
