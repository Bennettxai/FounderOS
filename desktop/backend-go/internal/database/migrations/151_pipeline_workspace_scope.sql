-- Pipelines and their default selection are workspace-owned.
-- Legacy rows are backfilled only when the owner belongs to exactly one
-- workspace, because guessing across multiple workspaces would leak data.

ALTER TABLE pipelines
    ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;

WITH unambiguous_owner_workspace AS (
    -- PostgreSQL has no MIN(uuid). The HAVING clause guarantees one distinct
    -- workspace per owner, so selecting the lexical minimum UUID text is safe.
    SELECT wm.user_id, MIN(wm.workspace_id::text)::uuid AS workspace_id
    FROM workspace_members wm
    WHERE wm.status = 'active'
    GROUP BY wm.user_id
    HAVING COUNT(DISTINCT wm.workspace_id) = 1
)
UPDATE pipelines p
SET workspace_id = u.workspace_id
FROM unambiguous_owner_workspace u
WHERE p.workspace_id IS NULL
  AND p.user_id = u.user_id;

CREATE INDEX IF NOT EXISTS idx_pipelines_workspace_user
    ON pipelines(workspace_id, user_id);
