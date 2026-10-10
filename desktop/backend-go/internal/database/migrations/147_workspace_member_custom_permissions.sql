-- Add per-member custom permission overrides expected by workspace services.
-- Some local databases were created before this column existed, which caused
-- workspace member/profile endpoints to fail when selecting a workspace.

ALTER TABLE workspace_members
  ADD COLUMN IF NOT EXISTS custom_permissions JSONB NOT NULL DEFAULT '{}'::jsonb;
