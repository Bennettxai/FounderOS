-- 126_boards.sql
-- Boards: the composition layer.
-- A board is NOT a new data module. It composes *views* of existing modules
-- (tasks, projects, relationships/clients, team, deals...) onto one surface,
-- all sharing one context filter (typically a client). Modules keep owning
-- their data; a board only describes which slices to show and how.
--
--   layout = ordered array of view definitions:
--     [{ "view": "tasks", "filters": {...}, "group_by": "status" }, ...]
--
-- kind:
--   'client' - a client board; subject_type/subject_id point at the client
--              whose context every view is filtered to.
--   'custom' - a user-composed board; context (if any) lives in the layout
--              view filters themselves.
--
-- is_pinned + position drive the sidebar "Boards" section (mirrors how
-- module_installations.is_enabled pins custom modules).
--
-- Additive only. No existing table is modified.
CREATE TABLE IF NOT EXISTS boards (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by   VARCHAR(255) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    kind         VARCHAR(50)  NOT NULL DEFAULT 'custom' CHECK (kind IN ('client', 'custom')),
    subject_type VARCHAR(50),                        -- e.g. 'client' (what the board is about)
    subject_id   UUID,                               -- id of the subject entity (no FK: subject_type varies)
    layout       JSONB NOT NULL DEFAULT '[]',        -- ordered [{view, filters, group_by}]
    is_pinned    BOOLEAN DEFAULT FALSE,
    position     INT DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_boards_workspace ON boards(workspace_id);
CREATE INDEX IF NOT EXISTS idx_boards_workspace_pinned ON boards(workspace_id, is_pinned) WHERE is_pinned;
