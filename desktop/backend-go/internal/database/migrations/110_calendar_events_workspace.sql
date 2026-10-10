-- 110_calendar_events_workspace.sql
-- Make calendar events a workspace-scopeable primitive. Events are owned by a
-- user (user_id) and optionally shared with a workspace (workspace_id). NULL
-- workspace_id = personal event. Mirrors the contexts/entity_links model so the
-- calendar works standalone (native events) and can be shared with the team,
-- with Google/Outlook as sync sources into the same table.

ALTER TABLE calendar_events ADD COLUMN IF NOT EXISTS workspace_id UUID;

CREATE INDEX IF NOT EXISTS idx_calendar_events_workspace
    ON calendar_events(workspace_id, start_time, end_time);
