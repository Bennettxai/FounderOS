-- 131_workspace_app_logos.sql
-- Optional custom logo URL for workspace apps. If empty, the UI falls back to
-- the app domain's favicon.

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '';
