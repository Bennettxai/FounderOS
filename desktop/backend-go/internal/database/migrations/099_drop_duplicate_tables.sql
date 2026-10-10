-- Migration 099: Drop duplicate tables and unify client_deals into CRM deals
-- Two parts:
--   Part 1: Drop osa_apps + osa_deployments (superseded by osa_generated_apps)
--   Part 2: Migrate client_deals data into the CRM deals table, then drop client_deals

BEGIN;

-- =============================================================================
-- PART 1: Drop duplicate OSA tables
-- osa_apps and osa_deployments are superseded by osa_generated_apps
-- =============================================================================

DROP TABLE IF EXISTS osa_deployments CASCADE;
DROP TABLE IF EXISTS osa_apps CASCADE;

-- =============================================================================
-- PART 2: Unify client_deals into CRM deals
-- =============================================================================

-- -------------------------------------------------------------------------
-- Step 1: Create a "Legacy Pipeline" for each user that has client_deals
-- -------------------------------------------------------------------------
INSERT INTO pipelines (user_id, name, description, is_default)
SELECT DISTINCT c.user_id,
       'Legacy Pipeline',
       'Auto-created from client deals migration',
       false
FROM client_deals cd
JOIN clients c ON cd.client_id = c.id
ON CONFLICT DO NOTHING;

-- -------------------------------------------------------------------------
-- Step 2: Create pipeline_stages for each dealstage enum value
-- Maps to the Legacy Pipeline for each user.
-- The actual dealstage enum values are:
--   qualification, proposal, negotiation, closed_won, closed_lost
-- -------------------------------------------------------------------------
INSERT INTO pipeline_stages (pipeline_id, name, position, probability, stage_type)
SELECT p.id, stage.name, stage.position, stage.probability, stage.stage_type
FROM pipelines p
CROSS JOIN (
    VALUES
        ('Qualification',  1,   10, 'open'),
        ('Proposal',       2,   30, 'open'),
        ('Negotiation',    3,   60, 'open'),
        ('Closed Won',     4,  100, 'won'),
        ('Closed Lost',    5,    0, 'lost')
) AS stage(name, position, probability, stage_type)
WHERE p.name = 'Legacy Pipeline'
  AND p.description = 'Auto-created from client deals migration'
ON CONFLICT (pipeline_id, name) DO NOTHING;

-- -------------------------------------------------------------------------
-- Step 3: Add client_id column to deals table (nullable FK to clients)
-- -------------------------------------------------------------------------
ALTER TABLE deals ADD COLUMN IF NOT EXISTS client_id UUID REFERENCES clients(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_deals_client_id ON deals(client_id);

-- -------------------------------------------------------------------------
-- Step 4: Migrate data from client_deals → deals
-- Maps dealstage enum values to the corresponding pipeline_stage rows
-- in the user's Legacy Pipeline.
-- -------------------------------------------------------------------------
INSERT INTO deals (
    user_id,
    pipeline_id,
    stage_id,
    name,
    description,
    amount,
    probability,
    expected_close_date,
    actual_close_date,
    status,
    client_id,
    created_at,
    updated_at
)
SELECT
    c.user_id,
    p.id AS pipeline_id,
    ps.id AS stage_id,
    cd.name,
    cd.notes,
    cd.value,
    cd.probability,
    cd.expected_close_date,
    cd.closed_at::date AS actual_close_date,
    CASE
        WHEN cd.stage = 'closed_won'  THEN 'won'
        WHEN cd.stage = 'closed_lost' THEN 'lost'
        ELSE 'open'
    END AS status,
    cd.client_id,
    cd.created_at,
    cd.updated_at
FROM client_deals cd
JOIN clients c ON cd.client_id = c.id
JOIN pipelines p ON p.user_id = c.user_id
    AND p.name = 'Legacy Pipeline'
    AND p.description = 'Auto-created from client deals migration'
JOIN pipeline_stages ps ON ps.pipeline_id = p.id
    AND ps.name = CASE cd.stage
        WHEN 'qualification' THEN 'Qualification'
        WHEN 'proposal'      THEN 'Proposal'
        WHEN 'negotiation'   THEN 'Negotiation'
        WHEN 'closed_won'    THEN 'Closed Won'
        WHEN 'closed_lost'   THEN 'Closed Lost'
    END;

-- -------------------------------------------------------------------------
-- Step 5: Drop the now-empty client_deals table
-- -------------------------------------------------------------------------
DROP TABLE IF EXISTS client_deals CASCADE;

-- -------------------------------------------------------------------------
-- Step 6: Drop the dealstage enum type
-- Only client_deals used it; it is no longer referenced.
-- -------------------------------------------------------------------------
DROP TYPE IF EXISTS dealstage CASCADE;

COMMIT;
