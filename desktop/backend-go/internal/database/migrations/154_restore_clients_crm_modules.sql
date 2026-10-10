-- Restore the Clients and CRM surfaces that were accidentally removed by the
-- June 25 desktop OAuth commit while their schema, handlers, and stores remained.
-- Restore the generated client projections locally before hydrating CRM records.
ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS pipeline_stage TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'pipeline_stage') STORED;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS agency_type TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'agency_type') STORED;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS outreach_status TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'outreach_status') STORED;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS offer_fit TEXT
  GENERATED ALWAYS AS (custom_fields ->> 'offer_fit') STORED;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS fit_score INTEGER
  GENERATED ALWAYS AS (
    CASE
      WHEN (custom_fields ->> 'fit_score') ~ '^[0-9]+$'
        THEN (custom_fields ->> 'fit_score')::INTEGER
      ELSE NULL
    END
  ) STORED;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS next_step_date TEXT
  GENERATED ALWAYS AS (NULLIF(custom_fields ->> 'next_step_date', '')) STORED;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS estimated_hours NUMERIC(10, 2);

CREATE INDEX IF NOT EXISTS idx_clients_workspace_pipeline_stage
  ON clients (workspace_id, pipeline_stage);
CREATE INDEX IF NOT EXISTS idx_clients_workspace_outreach_status
  ON clients (workspace_id, outreach_status);
CREATE INDEX IF NOT EXISTS idx_clients_workspace_fit_score
  ON clients (workspace_id, fit_score);
