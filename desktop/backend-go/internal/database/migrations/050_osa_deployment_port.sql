-- Migration: 045_osa_deployment_port.sql
-- Description: Add deployment_port column to osa_generated_apps
-- Created: 2026-01-09

ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS deployment_port INTEGER;

CREATE INDEX IF NOT EXISTS idx_osa_apps_deployment_port ON osa_generated_apps(deployment_port);
