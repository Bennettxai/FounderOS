-- Migration 106: CRM tables (companies, activities, contact-company links,
-- deal stage history) + missing columns on existing pipelines/pipeline_stages/deals.
--
-- Backstory: commit bb57b8a added crm.sql queries and sqlc-generated code for
-- 7 CRM tables, but never wrote the actual CREATE TABLE migrations. Some
-- (clients, deals, pipelines, pipeline_stages) ended up in production via
-- ad-hoc means with a subset of columns; others (companies, crm_activities,
-- contact_company_relations, deal_stage_history) were never created at all.
-- Migration 099 already references deals/pipelines/pipeline_stages and
-- assumes they exist; this migration backfills everything sqlc expects.

BEGIN;

-- ============================================================================
-- 1. crm_activity_type enum (used by crm_activities)
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'crm_activity_type') THEN
        CREATE TYPE crm_activity_type AS ENUM (
            'call', 'email', 'meeting', 'note', 'task',
            'sms', 'demo', 'proposal', 'contract', 'other'
        );
    END IF;
END$$;

-- ============================================================================
-- 2. companies
-- ============================================================================

CREATE TABLE IF NOT EXISTS companies (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    legal_name        VARCHAR(255),
    industry          VARCHAR(100),
    company_size      VARCHAR(50),
    website           VARCHAR(255),
    email             VARCHAR(255),
    phone             VARCHAR(50),
    address_line1     VARCHAR(255),
    address_line2     VARCHAR(255),
    city              VARCHAR(100),
    state             VARCHAR(100),
    postal_code       VARCHAR(20),
    country           VARCHAR(100),
    annual_revenue    NUMERIC(14,2),
    currency          VARCHAR(10),
    fiscal_year_end   VARCHAR(20),
    tax_id            VARCHAR(100),
    linkedin_url      VARCHAR(255),
    twitter_handle    VARCHAR(100),
    owner_id          VARCHAR(255),
    lifecycle_stage   VARCHAR(50),
    lead_source       VARCHAR(100),
    health_score      INTEGER,
    engagement_score  INTEGER,
    logo_url          VARCHAR(500),
    custom_fields     JSONB DEFAULT '{}'::jsonb,
    metadata          JSONB DEFAULT '{}'::jsonb,
    external_id       VARCHAR(255),
    external_source   VARCHAR(100),
    last_synced_at    TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_companies_user_id        ON companies(user_id);
CREATE INDEX IF NOT EXISTS idx_companies_industry       ON companies(industry) WHERE industry IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_companies_lifecycle      ON companies(lifecycle_stage) WHERE lifecycle_stage IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_companies_external       ON companies(external_source, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_companies_name_trgm      ON companies USING GIN (name gin_trgm_ops);

-- ============================================================================
-- 3. contact_company_relations  (links clients table → companies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS contact_company_relations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id  UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    company_id  UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    job_title   VARCHAR(255),
    department  VARCHAR(100),
    role_type   VARCHAR(50),
    is_primary  BOOLEAN DEFAULT FALSE,
    is_active   BOOLEAN DEFAULT TRUE,
    started_at  DATE,
    ended_at    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (contact_id, company_id)
);

CREATE INDEX IF NOT EXISTS idx_ccr_contact ON contact_company_relations(contact_id);
CREATE INDEX IF NOT EXISTS idx_ccr_company ON contact_company_relations(company_id);

-- ============================================================================
-- 4. crm_activities
-- ============================================================================

CREATE TABLE IF NOT EXISTS crm_activities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             VARCHAR(255) NOT NULL,
    activity_type       crm_activity_type NOT NULL,
    subject             VARCHAR(500) NOT NULL,
    description         TEXT,
    outcome             TEXT,
    deal_id             UUID REFERENCES deals(id) ON DELETE SET NULL,
    company_id          UUID REFERENCES companies(id) ON DELETE SET NULL,
    contact_id          UUID REFERENCES clients(id) ON DELETE SET NULL,
    participants        JSONB DEFAULT '[]'::jsonb,
    activity_date       TIMESTAMPTZ NOT NULL,
    duration_minutes    INTEGER,
    call_direction      VARCHAR(20),
    call_disposition    VARCHAR(50),
    call_recording_url  VARCHAR(500),
    email_direction     VARCHAR(20),
    email_message_id    VARCHAR(255),
    meeting_location    VARCHAR(255),
    meeting_url         VARCHAR(500),
    owner_id            VARCHAR(255),
    completed_by        VARCHAR(255),
    is_completed        BOOLEAN DEFAULT FALSE,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_activities_user      ON crm_activities(user_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_deal      ON crm_activities(deal_id)    WHERE deal_id    IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_crm_activities_company   ON crm_activities(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_crm_activities_contact   ON crm_activities(contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_crm_activities_pending   ON crm_activities(user_id, activity_date) WHERE is_completed = FALSE;

-- ============================================================================
-- 5. deal_stage_history
-- ============================================================================

CREATE TABLE IF NOT EXISTS deal_stage_history (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id           UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    from_stage_id     UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    to_stage_id       UUID NOT NULL REFERENCES pipeline_stages(id) ON DELETE CASCADE,
    changed_by        VARCHAR(255),
    changed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_seconds  INTEGER,
    deal_amount       NUMERIC(14,2)
);

CREATE INDEX IF NOT EXISTS idx_dsh_deal ON deal_stage_history(deal_id);
CREATE INDEX IF NOT EXISTS idx_dsh_to_stage ON deal_stage_history(to_stage_id);

-- ============================================================================
-- 6. Backfill missing columns on pipelines / pipeline_stages / deals
-- ============================================================================

ALTER TABLE pipelines
    ADD COLUMN IF NOT EXISTS description    TEXT,
    ADD COLUMN IF NOT EXISTS pipeline_type  VARCHAR(50),
    ADD COLUMN IF NOT EXISTS currency       VARCHAR(10),
    ADD COLUMN IF NOT EXISTS is_active      BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS color          VARCHAR(20),
    ADD COLUMN IF NOT EXISTS icon           VARCHAR(50);

-- pipeline name should be unique within a pipeline_id (used by ON CONFLICT in 099)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'pipeline_stages_pipeline_id_name_key'
    ) THEN
        ALTER TABLE pipeline_stages
            ADD CONSTRAINT pipeline_stages_pipeline_id_name_key UNIQUE (pipeline_id, name);
    END IF;
END$$;

ALTER TABLE pipeline_stages
    ADD COLUMN IF NOT EXISTS description   TEXT,
    ADD COLUMN IF NOT EXISTS stage_type    VARCHAR(20),
    ADD COLUMN IF NOT EXISTS rotting_days  INTEGER,
    ADD COLUMN IF NOT EXISTS color         VARCHAR(20),
    ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE deals
    ADD COLUMN IF NOT EXISTS currency           VARCHAR(10),
    ADD COLUMN IF NOT EXISTS owner_id           VARCHAR(255),
    ADD COLUMN IF NOT EXISTS company_id         UUID REFERENCES companies(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS primary_contact_id UUID REFERENCES clients(id)   ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS lost_reason        TEXT,
    ADD COLUMN IF NOT EXISTS priority           VARCHAR(20),
    ADD COLUMN IF NOT EXISTS lead_source        VARCHAR(100),
    ADD COLUMN IF NOT EXISTS deal_score         INTEGER,
    ADD COLUMN IF NOT EXISTS custom_fields      JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS stage_entered_at   TIMESTAMPTZ DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_deals_company_id ON deals(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_deals_owner      ON deals(owner_id)   WHERE owner_id   IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_deals_status     ON deals(user_id, status);

COMMIT;
