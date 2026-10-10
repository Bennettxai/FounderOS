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
