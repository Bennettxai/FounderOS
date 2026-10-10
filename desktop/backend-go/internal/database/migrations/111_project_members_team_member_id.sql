-- 111_project_members_team_member_id.sql
-- Align project_members with the canonical schema. The ListTeamMembers query
-- (team_members.sql) counts active projects via
--   JOIN project_members pm ... WHERE pm.team_member_id = tm.id
-- but some DBs were built from a migration set where project_members only had
-- user_id, so the column was missing and the Team module 500'd
-- (ERROR: column pm.team_member_id does not exist).

ALTER TABLE project_members ADD COLUMN IF NOT EXISTS team_member_id UUID;
CREATE INDEX IF NOT EXISTS idx_project_members_team_member ON project_members(team_member_id);
