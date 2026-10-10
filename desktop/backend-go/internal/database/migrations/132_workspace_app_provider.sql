-- 132_workspace_app_provider.sql
-- Tracks where an app URL came from: custom, MIOSA, Vercel, Netlify, etc.

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom';
