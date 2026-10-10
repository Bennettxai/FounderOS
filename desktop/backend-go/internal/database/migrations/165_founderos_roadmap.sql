-- 165: /os/roadmap and /os/reference (FounderOS v1 app/roadmap, app/reference).
-- ETL targets of FounderOS v1's SQLite phases, roadmap_items and domains,
-- which the ETL used to drop as retired. All three live in the FounderOS HQ
-- workspace; source keys are kept verbatim. The CHECKs mirror v1's Zod
-- (lib/schemas.ts RoadmapStatusSchema and the 2026-Q2 quarter shape).
-- phase_id and department_id are soft references, as in v1. Idempotent.
CREATE TABLE IF NOT EXISTS founderos_phases (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    items JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_phases_workspace ON founderos_phases(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_roadmap_items (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    quarter TEXT NOT NULL CHECK (quarter ~ '^[0-9]{4}-Q[1-4]$'),
    status TEXT NOT NULL CHECK (status IN ('done', 'now', 'next', 'later')),
    department_id TEXT,
    description TEXT NOT NULL DEFAULT '',
    phase_id TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_roadmap_items_workspace ON founderos_roadmap_items(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_domains (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    color TEXT NOT NULL,
    items JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_domains_workspace ON founderos_domains(workspace_id);
