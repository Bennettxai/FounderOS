-- Migration 094: Add missing sandbox/app columns to osa_generated_apps
-- The sqlc queries reference these columns but they don't exist in the DB table.

ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS container_id TEXT;
ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS sandbox_port INTEGER;
ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS sandbox_url TEXT;
ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS container_image TEXT;
ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS app_type TEXT DEFAULT 'web';
ALTER TABLE osa_generated_apps ADD COLUMN IF NOT EXISTS last_health_check TIMESTAMPTZ;

-- Create sandbox_events table (referenced by sandbox.sql queries)
CREATE TABLE IF NOT EXISTS sandbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id uuid NOT NULL REFERENCES osa_generated_apps(id) ON DELETE CASCADE,
    event_type varchar(100) NOT NULL,
    event_data jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sandbox_events_app_id ON sandbox_events(app_id);
CREATE INDEX IF NOT EXISTS idx_sandbox_events_type ON sandbox_events(event_type);
CREATE INDEX IF NOT EXISTS idx_sandbox_events_created ON sandbox_events(created_at DESC);
