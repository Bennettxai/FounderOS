-- 121_crm_field_pack.sql
-- Agency Lead CRM Field Pack (additive).
-- Source signal: workspaces/agency-miosa/inbox/signals/2026-06-29-agency-lead-crm-field-pack.md
--
-- The clients table is the CRM primitive. Agency-specific lead fields are written
-- by the frontend into clients.custom_fields (JSONB), which the Go handler already
-- round-trips end to end. To keep those fields queryable (filtering, reporting,
-- engine sync) WITHOUT changing the Go write path, we surface the high-traffic
-- fields as GENERATED STORED columns derived from custom_fields, plus indexes.
--
-- Fully additive: ADD COLUMN IF NOT EXISTS / CREATE INDEX IF NOT EXISTS. Safe to
-- re-run. No backfill needed (generated columns populate automatically).

-- Pipeline stage (14 stages live in the field pack; default to "lead_identified").
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS pipeline_stage TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'pipeline_stage') STORED;

-- Agency type (taxonomy, e.g. marketing / gohighlevel / med_spa / ...).
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS agency_type TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'agency_type') STORED;

-- Outreach status (not_contacted / visited / called / ... / follow_up_later).
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS outreach_status TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'outreach_status') STORED;

-- Offer fit (growth_systems_audit / phase_1_rebuild / ... / not_fit / unknown).
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS offer_fit TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'offer_fit') STORED;

-- Fit score 1..5 (cast text -> int; invalid/empty becomes NULL).
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS fit_score INTEGER
  GENERATED ALWAYS AS (
    NULLIF(custom_fields ->> 'fit_score', '')::INTEGER
  ) STORED;

-- Next step date (ISO yyyy-mm-dd string). Kept as TEXT because casting text -> date
-- is not immutable (depends on DateStyle) and so cannot back a generated column.
-- ISO yyyy-mm-dd sorts lexically the same as chronologically, so TEXT is sufficient
-- for the follow-up queue ordering.
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS next_step_date TEXT
  GENERATED ALWAYS AS (NULLIF(custom_fields ->> 'next_step_date', '')) STORED;

-- Indexes for board grouping, filtering and follow-up queue queries.
CREATE INDEX IF NOT EXISTS idx_clients_pipeline_stage ON clients (user_id, pipeline_stage);
CREATE INDEX IF NOT EXISTS idx_clients_agency_type ON clients (user_id, agency_type);
CREATE INDEX IF NOT EXISTS idx_clients_outreach_status ON clients (user_id, outreach_status);
CREATE INDEX IF NOT EXISTS idx_clients_fit_score ON clients (user_id, fit_score);
CREATE INDEX IF NOT EXISTS idx_clients_next_step_date ON clients (user_id, next_step_date);

-- GIN index over custom_fields so secondary field-pack keys (pain_category,
-- physical_office_status, tool_stack, etc.) stay searchable.
CREATE INDEX IF NOT EXISTS idx_clients_custom_fields_gin ON clients USING GIN (custom_fields);
