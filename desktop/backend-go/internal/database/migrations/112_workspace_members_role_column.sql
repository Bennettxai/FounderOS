-- 112_workspace_members_role_column.sql
-- The Go code consistently uses a `role` column on workspace_members (signup
-- provisioning, GetUserRole, invites, admin role changes - ~13 queries), but
-- some DBs were built from a migration set that only has `role_name`/`role_id`.
-- The missing column made `provisionDefaultWorkspace` fail silently (background
-- goroutine), so NO workspace was ever created on signup, and `GetUserRole`
-- ("SELECT role ...") errored - breaking every workspace-scoped role check.
--
-- Add `role` and keep it in sync with the existing role_name so both work.

ALTER TABLE workspace_members ADD COLUMN IF NOT EXISTS role VARCHAR(100) DEFAULT 'member';

-- Backfill from role_name where present so existing rows keep their role.
UPDATE workspace_members
SET role = COALESCE(NULLIF(role_name, ''), role, 'member')
WHERE role IS NULL OR role = 'member';
