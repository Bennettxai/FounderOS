-- 097_sorx_schedules.sql
-- Proactive SORX scheduler: stores per-user cron schedules for skills.
-- All schedules default to enabled = false; users explicitly opt in.

CREATE TABLE IF NOT EXISTS sorx_schedules (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id     TEXT        NOT NULL,
    user_id      TEXT        NOT NULL,
    workspace_id TEXT,                          -- NULL means personal (non-workspace)
    cron_expr    TEXT        NOT NULL,
    params       JSONB       NOT NULL DEFAULT '{}',
    enabled      BOOLEAN     NOT NULL DEFAULT false,
    last_run_at  TIMESTAMPTZ,
    last_status  TEXT,                          -- 'complete' | 'failed'
    last_error   TEXT,
    next_run_at  TIMESTAMPTZ,
    run_count    INTEGER     NOT NULL DEFAULT 0,
    fail_count   INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookups by user.
CREATE INDEX IF NOT EXISTS idx_sorx_schedules_user
    ON sorx_schedules (user_id);

-- Scheduler startup only needs to load enabled rows.
CREATE INDEX IF NOT EXISTS idx_sorx_schedules_enabled
    ON sorx_schedules (enabled)
    WHERE enabled = true;

-- One schedule per skill per user (personal, no workspace).
CREATE UNIQUE INDEX IF NOT EXISTS idx_sorx_schedules_unique
    ON sorx_schedules (skill_id, user_id)
    WHERE workspace_id IS NULL;

-- One schedule per skill per user per workspace.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sorx_schedules_unique_ws
    ON sorx_schedules (skill_id, user_id, workspace_id)
    WHERE workspace_id IS NOT NULL;

-- Keep updated_at current automatically.
CREATE OR REPLACE FUNCTION update_sorx_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sorx_schedules_updated_at_trigger ON sorx_schedules;
CREATE TRIGGER sorx_schedules_updated_at_trigger
    BEFORE UPDATE ON sorx_schedules
    FOR EACH ROW EXECUTE FUNCTION update_sorx_schedules_updated_at();
