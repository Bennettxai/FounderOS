-- Migration: 046_osa_app_metadata.sql
-- Description: Add app metadata columns to osa_generated_apps for UI display
-- Created: 2026-01-11
-- Phase: OSA-5 Integration Enhancement

-- =============================================================================
-- ADD APP METADATA COLUMNS
-- =============================================================================

-- Add app_name column (distinct from 'name' which may be technical)
-- COMMENT: Human-friendly app name for UI display (e.g., "My CRM System")
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS app_name VARCHAR(255);

-- Add category column for filtering and organization
-- COMMENT: App category for grouping (e.g., "CRM", "Analytics", "Finance", "Custom")
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS category VARCHAR(100);

-- Add icon_type column for visual identification
-- COMMENT: Icon identifier for UI rendering (e.g., "building", "chart-bar", "users")
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS icon_type VARCHAR(50);

-- Add app_description column (distinct from 'description' for detailed metadata)
-- COMMENT: User-facing description of app functionality
ALTER TABLE osa_generated_apps
ADD COLUMN IF NOT EXISTS app_description TEXT;

-- =============================================================================
-- SET DEFAULT VALUES FOR EXISTING ROWS
-- =============================================================================

-- Set app_name to display_name for existing rows
UPDATE osa_generated_apps
SET app_name = display_name
WHERE app_name IS NULL;

-- Set default category for existing rows
UPDATE osa_generated_apps
SET category = 'Custom'
WHERE category IS NULL;

-- Set default icon for existing rows
UPDATE osa_generated_apps
SET icon_type = 'cube'
WHERE icon_type IS NULL;

-- Set app_description from description for existing rows
UPDATE osa_generated_apps
SET app_description = description
WHERE app_description IS NULL AND description IS NOT NULL;

-- =============================================================================
-- CREATE INDEX FOR CATEGORY FILTERING
-- =============================================================================

-- Index on category for efficient filtering in workspace views
CREATE INDEX IF NOT EXISTS idx_osa_apps_category
ON osa_generated_apps(category);

-- Composite index for workspace + category filtering
CREATE INDEX IF NOT EXISTS idx_osa_apps_workspace_category
ON osa_generated_apps(workspace_id, category);

-- =============================================================================
-- ADD COLUMN COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON COLUMN osa_generated_apps.app_name IS
'Human-friendly application name for UI display';

COMMENT ON COLUMN osa_generated_apps.category IS
'Application category for filtering and grouping (e.g., CRM, Analytics, Finance, Custom)';

COMMENT ON COLUMN osa_generated_apps.icon_type IS
'Icon identifier for UI rendering using icon library (e.g., building, chart-bar, users, cube)';

COMMENT ON COLUMN osa_generated_apps.app_description IS
'User-facing description of application functionality and purpose';
