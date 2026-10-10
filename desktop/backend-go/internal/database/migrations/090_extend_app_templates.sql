-- +migrate Up
-- Migration 090: Extend app_templates for YAML template metadata
-- Adds columns to link DB templates to their YAML sources

-- Add YAML template metadata columns
ALTER TABLE app_templates
ADD COLUMN IF NOT EXISTS yaml_template_name VARCHAR(100),
ADD COLUMN IF NOT EXISTS yaml_version VARCHAR(20),
ADD COLUMN IF NOT EXISTS template_variables JSONB;

-- Index for fast YAML template lookup
CREATE INDEX IF NOT EXISTS idx_app_templates_yaml_name
ON app_templates(yaml_template_name);

-- Comment on new columns
COMMENT ON COLUMN app_templates.yaml_template_name IS 'Reference to YAML template file name (source of truth)';
COMMENT ON COLUMN app_templates.yaml_version IS 'Version from YAML template file';
COMMENT ON COLUMN app_templates.template_variables IS 'JSONB storing template variable definitions from YAML';

-- Update existing templates with YAML references (if they match known templates)
-- This helps with initial sync detection
UPDATE app_templates SET yaml_template_name = 'crm-app-generation' WHERE template_name = 'crm_module' AND yaml_template_name IS NULL;
UPDATE app_templates SET yaml_template_name = 'dashboard-creation' WHERE template_name = 'saas_dashboard' AND yaml_template_name IS NULL;

-- +migrate Down
-- Rollback: Remove YAML metadata columns

DROP INDEX IF EXISTS idx_app_templates_yaml_name;

ALTER TABLE app_templates
DROP COLUMN IF EXISTS template_variables,
DROP COLUMN IF EXISTS yaml_version,
DROP COLUMN IF EXISTS yaml_template_name;
