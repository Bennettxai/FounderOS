-- 133_app_catalog.sql
-- Platform catalog for apps that can be installed into a workspace desktop.

CREATE TABLE IF NOT EXISTS app_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(120) UNIQUE NOT NULL,
    name VARCHAR(160) NOT NULL,
    provider VARCHAR(60) DEFAULT 'custom',
    app_type VARCHAR(40) DEFAULT 'web_app',
    url TEXT NOT NULL DEFAULT '',
    launch_mode VARCHAR(40) DEFAULT 'iframe',
    icon VARCHAR(80) DEFAULT 'layout-grid',
    logo_url TEXT DEFAULT '',
    color VARCHAR(40) DEFAULT '#111827',
    category VARCHAR(80) DEFAULT 'general',
    notes TEXT DEFAULT '',
    url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
    status VARCHAR(40) DEFAULT 'active',
    is_featured BOOLEAN DEFAULT FALSE,
    position_index INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE app_catalog
ADD COLUMN IF NOT EXISTS slug VARCHAR(120),
ADD COLUMN IF NOT EXISTS name VARCHAR(160),
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '',
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

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS catalog_app_id UUID REFERENCES app_catalog(id) ON DELETE SET NULL;

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
CREATE INDEX IF NOT EXISTS idx_workspace_apps_catalog ON workspace_apps(workspace_id, catalog_app_id);
