-- 136_fix_app_catalog_logos.sql
-- Replace bad or missing catalog logo paths with real assets.

UPDATE app_catalog
SET logo_url = '/app-logos/gohighlevel.ico',
    updated_at = NOW()
WHERE slug = 'gohighlevel';

UPDATE app_catalog
SET logo_url = '/app-logos/google-calendar.svg',
    updated_at = NOW()
WHERE slug = 'google-calendar';

UPDATE app_catalog
SET logo_url = '/app-logos/clickup.svg',
    updated_at = NOW()
WHERE slug = 'clickup';
