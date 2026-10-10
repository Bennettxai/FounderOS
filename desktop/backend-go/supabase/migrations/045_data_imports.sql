-- Migration: 028_data_imports.sql
-- Data import infrastructure for external data sources
-- Handles ChatGPT/Claude exports, CRM imports, analytics data, etc.

-- ============================================================================
-- IMPORT JOBS
-- ============================================================================
-- Tracks all data import operations with status and progress

CREATE TYPE import_status AS ENUM (
    'pending',      -- Waiting to start
    'validating',   -- Validating file/data format
    'mapping',      -- User configuring field mappings
    'processing',   -- Import in progress
    'completed',    -- Successfully completed
    'failed',       -- Failed with error
    'cancelled'     -- Cancelled by user
);

CREATE TYPE import_source_type AS ENUM (
    -- AI Chat Exports
    'chatgpt_export',       -- OpenAI ChatGPT JSON export
    'claude_export',        -- Anthropic Claude JSON export
    'custom_chat_export',   -- Other chat format

    -- CRM Data
    'hubspot_contacts',     -- HubSpot CRM contacts
    'hubspot_deals',        -- HubSpot deals/opportunities
    'hubspot_companies',    -- HubSpot companies
    'salesforce_contacts',  -- Salesforce contacts
    'salesforce_accounts',  -- Salesforce accounts

    -- Task/Project Tools
    'linear_issues',        -- Linear issues
    'notion_database',      -- Notion database export
    'asana_tasks',          -- Asana tasks
    'jira_issues',          -- Jira issues

    -- Calendar/Meetings
    'google_calendar',      -- Google Calendar events
    'outlook_calendar',     -- Outlook Calendar events

    -- Analytics
    'fathom_analytics',     -- Fathom Analytics data
    'plausible_analytics',  -- Plausible Analytics data

    -- Files/Documents
    'google_drive',         -- Google Drive files
    'dropbox',              -- Dropbox files
    'notion_pages',         -- Notion pages

    -- Other
    'csv_generic',          -- Generic CSV import
    'json_generic',         -- Generic JSON import
    'custom'                -- Custom import type
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Import type and source
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),  -- Link to integration_providers if applicable

    -- File/data info
    original_filename VARCHAR(500),
    file_size_bytes BIGINT,
    content_type VARCHAR(100),

    -- Status tracking
    status import_status DEFAULT 'pending',
    progress_percent INT DEFAULT 0,

    -- Counts
    total_records INT DEFAULT 0,
    processed_records INT DEFAULT 0,
    imported_records INT DEFAULT 0,
    skipped_records INT DEFAULT 0,
    failed_records INT DEFAULT 0,

    -- Mapping configuration (from data_sync_mappings or inline)
    field_mapping JSONB DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    import_options JSONB DEFAULT '{}',  -- dedup strategy, conflict handling, etc.

    -- Target module in BusinessOS
    target_module VARCHAR(50) NOT NULL,  -- clients, conversations, tasks, etc.
    target_entity VARCHAR(100),          -- Optional sub-entity

    -- Results
    result_summary JSONB DEFAULT '{}',   -- Summary of what was imported
    error_log JSONB DEFAULT '[]',        -- Array of errors with line numbers

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Error info
    error_message TEXT,
    error_details JSONB
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_source ON import_jobs(source_type);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created ON import_jobs(created_at DESC);

-- ============================================================================
-- IMPORTED RECORDS TRACKING
-- ============================================================================
-- Links external record IDs to BusinessOS record IDs for deduplication

CREATE TABLE IF NOT EXISTS imported_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,

    -- Source identification
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),
    external_id VARCHAR(500) NOT NULL,  -- ID from source system

    -- Target identification
    target_module VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100),
    target_record_id UUID NOT NULL,     -- ID in BusinessOS

    -- Metadata
    external_data_hash VARCHAR(64),     -- SHA256 of source data for change detection
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_type, external_id)
);

CREATE INDEX IF NOT EXISTS idx_imported_records_user ON imported_records(user_id);
CREATE INDEX IF NOT EXISTS idx_imported_records_source ON imported_records(source_type, external_id);
CREATE INDEX IF NOT EXISTS idx_imported_records_target ON imported_records(target_module, target_record_id);

-- ============================================================================
-- FIELD MAPPING TEMPLATES
-- ============================================================================
-- Pre-defined field mapping templates for common import types

CREATE TABLE IF NOT EXISTS import_mapping_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Template identification
    source_type import_source_type NOT NULL,
    target_module VARCHAR(50) NOT NULL,
    template_name VARCHAR(100) NOT NULL,

    -- Mapping definition
    field_mappings JSONB NOT NULL DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    default_values JSONB DEFAULT '{}',

    -- Template metadata
    description TEXT,
    is_system_template BOOLEAN DEFAULT FALSE,  -- Built-in vs user-created
    created_by VARCHAR(255),                   -- NULL for system templates

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(source_type, target_module, template_name)
);

-- ============================================================================
-- CONVERSATION IMPORTS TABLE
-- ============================================================================
-- Stores imported conversations from ChatGPT, Claude, etc.

CREATE TABLE IF NOT EXISTS imported_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,

    -- Source info
    source_type import_source_type NOT NULL,
    external_conversation_id VARCHAR(255),

    -- Conversation data
    title VARCHAR(500),
    model VARCHAR(100),                       -- gpt-4, claude-3-opus, etc.

    -- Messages are stored as JSONB array
    -- Format: [{"role": "user/assistant/system", "content": "...", "timestamp": "..."}]
    messages JSONB NOT NULL DEFAULT '[]',
    message_count INT DEFAULT 0,

    -- Metadata
    original_created_at TIMESTAMPTZ,          -- When created in source system
    original_updated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',              -- Source-specific metadata

    -- BusinessOS integration
    linked_context_id UUID,                   -- Link to contexts table
    linked_project_id UUID,                   -- Link to projects
    tags TEXT[] DEFAULT '{}',

    -- Full-text search
    search_content TEXT,                      -- Concatenated text for search

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imported_conversations_user ON imported_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_source ON imported_conversations(source_type);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_job ON imported_conversations(import_job_id);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_search ON imported_conversations USING GIN(to_tsvector('english', search_content));

-- ============================================================================
-- INSERT SYSTEM MAPPING TEMPLATES
-- ============================================================================

-- ChatGPT to Conversations template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'chatgpt_export',
    'conversations',
    'default',
    '{
        "title": "title",
        "create_time": "original_created_at",
        "update_time": "original_updated_at",
        "mapping.messages": "messages"
    }',
    '{
        "create_time": {"type": "timestamp", "format": "unix"},
        "update_time": {"type": "timestamp", "format": "unix"},
        "messages": {"type": "chatgpt_messages"}
    }',
    'Default mapping for ChatGPT conversation exports',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- Claude to Conversations template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'claude_export',
    'conversations',
    'default',
    '{
        "name": "title",
        "created_at": "original_created_at",
        "updated_at": "original_updated_at",
        "chat_messages": "messages"
    }',
    '{
        "created_at": {"type": "timestamp", "format": "iso8601"},
        "updated_at": {"type": "timestamp", "format": "iso8601"},
        "chat_messages": {"type": "claude_messages"}
    }',
    'Default mapping for Claude conversation exports',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- HubSpot Contacts to Clients template
INSERT INTO import_mapping_templates (source_type, target_module, template_name, field_mappings, transform_rules, description, is_system_template)
VALUES (
    'hubspot_contacts',
    'clients',
    'default',
    '{
        "properties.firstname": "first_name",
        "properties.lastname": "last_name",
        "properties.email": "email",
        "properties.phone": "phone",
        "properties.company": "company_name",
        "properties.jobtitle": "job_title",
        "properties.hs_object_id": "external_id",
        "properties.createdate": "original_created_at",
        "properties.lastmodifieddate": "original_updated_at"
    }',
    '{
        "createdate": {"type": "timestamp", "format": "unix_ms"},
        "lastmodifieddate": {"type": "timestamp", "format": "unix_ms"}
    }',
    'Default mapping for HubSpot contacts to BusinessOS clients',
    TRUE
) ON CONFLICT (source_type, target_module, template_name) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE import_jobs IS 'Tracks all data import operations with status and progress';
COMMENT ON TABLE imported_records IS 'Links external record IDs to BusinessOS records for deduplication';
COMMENT ON TABLE import_mapping_templates IS 'Pre-defined field mapping templates for common import types';
COMMENT ON TABLE imported_conversations IS 'Stores imported conversations from ChatGPT, Claude, etc.';
