-- Add Instagram as a first-class ContentOS connector.
INSERT INTO integration_providers (
	id,
	name,
	description,
	category,
	icon_url,
	modules,
	skills,
	status,
	created_at,
	updated_at
) VALUES (
	'instagram',
	'Instagram',
	'Connect posts, Reels, profile metrics, publishing status, and ContentOS performance audits.',
	'social',
	'/logos/integrations/instagram.svg',
	ARRAY['content'],
	ARRAY['instagram.audit_profile', 'instagram.sync_reels', 'instagram.publish_media'],
	'available',
	NOW(),
	NOW()
) ON CONFLICT (id) DO UPDATE SET
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	category = EXCLUDED.category,
	icon_url = EXCLUDED.icon_url,
	modules = EXCLUDED.modules,
	skills = EXCLUDED.skills,
	status = EXCLUDED.status,
	updated_at = NOW();
