-- 122_content_fields.sql
-- Deepens the Content module from a bare title/body pipeline into the structure the
-- VSL / ad message bank actually implies: every piece of content has a hook (the
-- opening line that earns attention), a call to action, and a channel it ships on.
-- The body column already exists (it holds the script / copy / outline).
-- Additive and idempotent. Models hook -> body -> CTA per channel without
-- hardcoding workspace data.

ALTER TABLE content_items ADD COLUMN IF NOT EXISTS hook    TEXT DEFAULT '';  -- opening attention grab
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS cta     TEXT DEFAULT '';  -- call to action / next step
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS channel VARCHAR(60) DEFAULT '';  -- where it ships (instagram|youtube|tiktok|linkedin|newsletter|x|...)
