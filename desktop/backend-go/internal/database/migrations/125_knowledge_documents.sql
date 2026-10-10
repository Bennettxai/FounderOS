-- 125_knowledge_documents.sql
-- Cloud-stored copy of a workspace's knowledge, synced from a user's local
-- Optimal Engine / workspace files. Local-first stays the source of truth; this
-- is the shared copy so teammates WITHOUT their own engine can view the
-- knowledge in the Knowledge module (read-only), and teammates WITH an engine
-- can pull it into their local engine.
--
-- Additive only. Does not touch the local file-tree path (KNOWLEDGE_WORKSPACES_ROOT);
-- the knowledge handlers fall back to this table only when local files are absent
-- (i.e. on the cloud backend, for teammates without their own engine).
--
-- Keyed by the workspace FOLDER SLUG (matches ?workspace= in /api/knowledge/tree|file),
-- with workspace_id retained for future role-scoped gating.
DROP TABLE IF EXISTS knowledge_documents;
CREATE TABLE knowledge_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_slug TEXT NOT NULL,                 -- workspace folder slug
    workspace_id   UUID,                          -- BusinessOS workspace (for role gating), nullable
    path           TEXT NOT NULL,                 -- relative path within the workspace (e.g. inbox/signals/x.md)
    title          TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL DEFAULT '',      -- raw markdown (incl. frontmatter)
    section        TEXT NOT NULL DEFAULT 'docs',  -- top-level folder, drives the module tabs
    synced_by      VARCHAR(255),
    synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_slug, path)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_documents_slug ON knowledge_documents(workspace_slug);
