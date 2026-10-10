-- 161: the operator memory writes land in Optimal Engine (GBrain retired).
-- Plaud ingests may now record via='oe', and Fathom call archiving gets its
-- own idempotency ledger. Idempotent: safe to re-apply (check-schema).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'founderos_plaud_ingests_via_check') THEN
        ALTER TABLE founderos_plaud_ingests DROP CONSTRAINT founderos_plaud_ingests_via_check;
    END IF;
    ALTER TABLE founderos_plaud_ingests
        ADD CONSTRAINT founderos_plaud_ingests_via_check CHECK (via IN ('gbrain', 'store', 'oe'));
END $$;

CREATE TABLE IF NOT EXISTS founderos_call_archive (
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    signal_id TEXT,
    entry JSONB NOT NULL DEFAULT '{}',
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source, external_id)
);
CREATE INDEX IF NOT EXISTS idx_founderos_call_archive_workspace ON founderos_call_archive(workspace_id);
