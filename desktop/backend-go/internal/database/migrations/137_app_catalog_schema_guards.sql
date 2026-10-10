-- 137_app_catalog_schema_guards.sql
-- Keep older local databases aligned with the app catalog handler schema.
-- This is intentionally idempotent and does not seed workspace app data.

ALTER TABLE IF EXISTS workspace_apps
ADD COLUMN IF NOT EXISTS catalog_app_id UUID,
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS launch_mode VARCHAR(40) DEFAULT 'iframe',
ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS icon VARCHAR(80) DEFAULT 'layout-grid',
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS color VARCHAR(40) DEFAULT '#111827',
ADD COLUMN IF NOT EXISTS category VARCHAR(80) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS show_on_desktop BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS show_in_dock BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS position_index INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
ADD COLUMN IF NOT EXISTS created_by VARCHAR(255),
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

ALTER TABLE IF EXISTS app_catalog
ADD COLUMN IF NOT EXISTS slug VARCHAR(120),
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS launch_mode VARCHAR(40) DEFAULT 'iframe',
ADD COLUMN IF NOT EXISTS icon VARCHAR(80) DEFAULT 'layout-grid',
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS color VARCHAR(40) DEFAULT '#111827',
ADD COLUMN IF NOT EXISTS category VARCHAR(80) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS position_index INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_workspace_apps_workspace ON workspace_apps(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_status ON workspace_apps(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_desktop ON workspace_apps(workspace_id, show_on_desktop);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_position ON workspace_apps(workspace_id, position_index, name);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_catalog ON workspace_apps(workspace_id, catalog_app_id);
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_index i
        JOIN pg_attribute a
          ON a.attrelid = i.indrelid
         AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = 'app_catalog'::regclass
          AND i.indisunique
          AND a.attname = 'slug'
    ) THEN
        CREATE UNIQUE INDEX idx_app_catalog_slug_unique ON app_catalog(slug);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_app_catalog_status ON app_catalog(status, position_index, name);
CREATE INDEX IF NOT EXISTS idx_app_catalog_category ON app_catalog(category, position_index, name);

UPDATE app_catalog
SET logo_url = '/app-logos/google-calendar.svg',
    updated_at = NOW()
WHERE slug = 'google-calendar'
  AND logo_url = '/app-logos/google-calendar.ico';

UPDATE app_catalog
SET logo_url = '/app-logos/clickup.svg',
    updated_at = NOW()
WHERE slug = 'clickup'
  AND logo_url = '/app-logos/clickup.ico';
