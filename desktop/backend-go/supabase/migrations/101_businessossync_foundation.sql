-- ═══════════════════════════════════════════════════════════════════════════════
-- BusinessOSSync Foundation Migration
-- ═══════════════════════════════════════════════════════════════════════════════
-- Created: 2026-06-24
-- Description: Adds workspace scoping to shareable content (starting with contexts)
--              and a per-module sync policy table (canonical = sync to workspace,
--              one-off = local only). This is the data-model foundation for
--              BusinessOSSync (local-first + selective cloud sync).
--
-- Safety: fully idempotent (IF NOT EXISTS / ON CONFLICT DO NOTHING). Safe to run
--         against the existing populated cloud DB. NULL workspace_id = local-only,
--         so nothing leaks across the team until a row is deliberately stamped.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- 1. Workspace scoping on contexts (knowledge docs).
--    Nullable on purpose: NULL = private/local-only. A row only becomes
--    team-visible when stamped with a workspace_id by a workspace-mode module.
--    ON DELETE SET NULL: deleting a workspace demotes its docs to local, never
--    destroys content.
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE contexts
    ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_contexts_workspace_updated
    ON contexts(workspace_id, updated_at);

-- ───────────────────────────────────────────────────────────────────────────────
-- 2. Per-module sync policy.
--    One row per (workspace, module). sync_mode='workspace' => rows of that module
--    sync to the shared cloud DB for the team; sync_mode='local' => stay on device.
--    is_published gates whether the workspace-mode data is currently shared.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sync_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module VARCHAR(100) NOT NULL,
    sync_mode VARCHAR(20) NOT NULL DEFAULT 'local'
        CHECK (sync_mode IN ('workspace', 'local')),
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, module)
);

CREATE INDEX IF NOT EXISTS idx_sync_policies_workspace ON sync_policies(workspace_id);

-- ───────────────────────────────────────────────────────────────────────────────
-- 3. Backfill existing contexts -> the owner's workspace, ONLY when unambiguous
--    (the user belongs to exactly one active workspace). Multi-workspace users are
--    left NULL (local-only) to avoid guessing. user_id is VARCHAR on both sides.
-- ───────────────────────────────────────────────────────────────────────────────
UPDATE contexts c
SET workspace_id = m.workspace_id
FROM (
    -- exactly one active workspace per user here, so (array_agg)[1] is unambiguous
    -- (uuid has no MIN() aggregate, hence array_agg instead)
    SELECT user_id, (array_agg(workspace_id))[1] AS workspace_id
    FROM workspace_members
    WHERE status = 'active'
    GROUP BY user_id
    HAVING COUNT(*) = 1
) m
WHERE c.user_id = m.user_id
  AND c.workspace_id IS NULL;

-- ───────────────────────────────────────────────────────────────────────────────
-- 4. Seed canonical module policies for every existing workspace so shared modules
--    sync out of the box. One-off / mini-app modules are intentionally NOT seeded
--    (absence => treated as local by the application).
-- ───────────────────────────────────────────────────────────────────────────────
INSERT INTO sync_policies (workspace_id, module, sync_mode, is_published)
SELECT w.id, m.module, 'workspace', TRUE
FROM workspaces w
CROSS JOIN (VALUES ('knowledge'), ('contexts'), ('projects'), ('tasks')) AS m(module)
ON CONFLICT (workspace_id, module) DO NOTHING;
