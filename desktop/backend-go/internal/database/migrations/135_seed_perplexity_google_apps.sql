-- 135_seed_perplexity_google_apps.sql
-- Add Perplexity and Google Search to the workspace app catalog.

INSERT INTO app_catalog (
    slug, name, provider, app_type, url, launch_mode, icon, logo_url, color,
    category, notes, url_class, status, is_featured, position_index
) VALUES
    ('perplexity', 'Perplexity', 'perplexity', 'web_app', 'https://www.perplexity.ai', 'iframe', 'search', '/app-logos/perplexity.svg', '#111827', 'ai', 'AI answer engine and research app.', 'custom_domain', 'active', TRUE, 30),
    ('google', 'Google', 'google', 'web_app', 'https://www.google.com', 'iframe', 'search', '/app-logos/google.svg', '#4285F4', 'search', 'Google Search.', 'custom_domain', 'active', TRUE, 40)
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
