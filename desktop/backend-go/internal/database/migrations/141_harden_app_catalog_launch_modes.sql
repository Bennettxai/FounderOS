-- 141_harden_app_catalog_launch_modes.sql
-- Keep store entries explicit: third-party SaaS opens in the browser, while
-- MIOSA/deployed apps remain eligible for embedded iframe launches.

INSERT INTO app_catalog (
    slug, name, provider, app_type, url, launch_mode, icon, logo_url, color,
    category, notes, url_class, status, is_featured, position_index
) VALUES
    ('perplexity', 'Perplexity', 'perplexity', 'web_app', 'https://www.perplexity.ai', 'browser', 'search', '/app-logos/perplexity.svg', '#111827', 'ai', 'AI answer engine and research app.', 'custom_domain', 'active', TRUE, 30),
    ('google', 'Google', 'google', 'web_app', 'https://www.google.com', 'browser', 'search', '/app-logos/google.svg', '#4285F4', 'search', 'Google Search.', 'custom_domain', 'active', TRUE, 40),
    ('gohighlevel', 'GoHighLevel', 'gohighlevel', 'web_app', 'https://app.gohighlevel.com', 'browser', 'workflow', '/app-logos/gohighlevel.ico', '#111827', 'crm', 'CRM, marketing automation, and agency client ops.', 'custom_domain', 'active', TRUE, 110)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    provider = EXCLUDED.provider,
    app_type = EXCLUDED.app_type,
    url = EXCLUDED.url,
    launch_mode = EXCLUDED.launch_mode,
    icon = EXCLUDED.icon,
    logo_url = EXCLUDED.logo_url,
    color = EXCLUDED.color,
    category = EXCLUDED.category,
    notes = EXCLUDED.notes,
    url_class = EXCLUDED.url_class,
    status = EXCLUDED.status,
    is_featured = EXCLUDED.is_featured,
    position_index = EXCLUDED.position_index,
    updated_at = NOW();

UPDATE app_catalog
SET launch_mode = 'browser',
    updated_at = NOW()
WHERE status = 'active'
  AND url_class = 'custom_domain'
  AND provider <> 'miosa';

UPDATE app_catalog
SET launch_mode = 'iframe',
    updated_at = NOW()
WHERE status = 'active'
  AND (
    provider = 'miosa'
    OR url_class IN ('temporary_preview', 'always_on_preview', 'stable_sandbox_embed', 'durable_deployment')
  );
