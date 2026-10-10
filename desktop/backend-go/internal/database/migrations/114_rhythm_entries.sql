-- 114_rhythm_entries.sql
-- The Rhythm module: a per-workspace operating cadence — daily/weekly/monthly
-- entries (focuses, blockers, priorities, notes) so a team's rhythm is captured
-- and decodable by humans AND AI agents. Workspace-scoped.

CREATE TABLE IF NOT EXISTS rhythm_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    period VARCHAR(20) NOT NULL DEFAULT 'daily',   -- daily | weekly | monthly
    kind VARCHAR(20) NOT NULL DEFAULT 'focus',     -- focus | blocker | priority | note
    content TEXT NOT NULL DEFAULT '',
    entry_date DATE,
    position INT DEFAULT 0,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rhythm_entries_workspace ON rhythm_entries(workspace_id, period);
