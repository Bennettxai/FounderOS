-- 164: the retired CRM archive on /os/funnel (FounderOS v1 funnel_archive,
-- docs/funnel-crm-archive.md): Attio + GoHighLevel journeys restored
-- 2026-09-26, each stored whole as the FunnelJourney JSON. ETL target of
-- FounderOS v1's SQLite funnel_archive (id, venture, journey TEXT); the
-- workspace is the journey's venture workspace. Read-only for the page.
CREATE TABLE IF NOT EXISTS founderos_funnel_archive (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    venture TEXT NOT NULL CHECK (venture IN ('vantage', 'launchpad-cohort')),
    journey JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_archive_workspace ON founderos_funnel_archive(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_archive_venture ON founderos_funnel_archive(venture);
