-- 124_module_fields.sql
-- Additive fields for common marketing and operations modules.
-- All ADD COLUMN IF NOT EXISTS so this is safe to re-run.

-- Offers: tiered offer ladder.
--   tier    = where this sits in the ladder (audit | phase-1 | phase-2 | lane)
--   promise = the outcome one-liner the buyer is paying for
--   cta     = the call-to-action that sells it (default Solutions Call)
ALTER TABLE offers ADD COLUMN IF NOT EXISTS tier    VARCHAR(40)  DEFAULT 'audit';
ALTER TABLE offers ADD COLUMN IF NOT EXISTS promise TEXT         DEFAULT '';
ALTER TABLE offers ADD COLUMN IF NOT EXISTS cta     VARCHAR(120) DEFAULT 'Book a Solutions Call';

-- Campaigns: channel pushes carry the message bank hook + CTA.
--   hook = the buyer hook this push leads with
--   cta  = the call-to-action it drives to
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS hook TEXT         DEFAULT '';
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS cta  VARCHAR(120) DEFAULT 'Book a Solutions Call';

-- Personas: ICP segments are explicitly best-fit vs poor-fit in the offer doc.
--   fit = best | poor (drives who we chase vs disqualify)
ALTER TABLE personas ADD COLUMN IF NOT EXISTS fit VARCHAR(20) DEFAULT 'best';

-- Rhythm: the operating cadence has an explicit owner.
ALTER TABLE rhythm_entries ADD COLUMN IF NOT EXISTS owner VARCHAR(120) DEFAULT '';

-- Sites: web properties are funnels/pages, not just generic sites.
--   kind = funnel | page | form | site | app
--   cta  = the primary call-to-action on the property
ALTER TABLE sites ADD COLUMN IF NOT EXISTS kind VARCHAR(40)  DEFAULT 'page';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS cta  VARCHAR(120) DEFAULT 'Book a Solutions Call';
