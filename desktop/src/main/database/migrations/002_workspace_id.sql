-- BusinessOSSync: workspace scoping for shareable modules on the local SQLite mirror.
-- Mirrors cloud migration 101_businessossync_foundation.sql, which added a nullable
-- workspace_id to contexts/projects/tasks. NULL = private/local-only; a row becomes
-- team-visible only when the server stamps it with a workspace_id (it comes back down
-- on the next workspace-scoped pull). The desktop never needs to stamp it itself.
--
-- SQLite has no "ADD COLUMN IF NOT EXISTS", and this file only runs once (its name is
-- recorded in the migrations table after it applies), so a plain ADD COLUMN is correct
-- for every install that already has 001_initial.sql. Fresh installs run both in order.

ALTER TABLE contexts ADD COLUMN workspace_id TEXT;
ALTER TABLE projects ADD COLUMN workspace_id TEXT;
ALTER TABLE tasks    ADD COLUMN workspace_id TEXT;

-- Workspace-scoped pull merges by updated_at and filters/joins on workspace_id, so
-- index the same (workspace_id, updated_at) shape the cloud uses.
CREATE INDEX IF NOT EXISTS idx_contexts_workspace ON contexts(workspace_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_projects_workspace ON projects(workspace_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_tasks_workspace    ON tasks(workspace_id, updated_at);
