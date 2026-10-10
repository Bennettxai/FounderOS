-- 128_team_members_workspace.sql
-- Team members were purely user_id-scoped, so the same roster appeared in
-- EVERY workspace the user opened. Same fix as clients (migration 127):
-- a team member belongs to one business context. NULL = unassigned/legacy.
ALTER TABLE team_members ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_team_members_workspace ON team_members(workspace_id);
