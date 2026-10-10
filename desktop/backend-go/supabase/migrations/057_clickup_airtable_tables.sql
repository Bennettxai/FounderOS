-- Migration 034: ClickUp and Airtable Integration Tables
-- This adds storage for ClickUp project management and Airtable database data

-- ============================================================================
-- CLICKUP TABLES
-- ============================================================================

-- ClickUp workspaces (called Teams in API v2)
CREATE TABLE IF NOT EXISTS clickup_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    avatar TEXT,
    member_count INT DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_workspaces_user ON clickup_workspaces(user_id);

-- ClickUp spaces
CREATE TABLE IF NOT EXISTS clickup_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    space_id VARCHAR(100) NOT NULL,
    workspace_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    private BOOLEAN DEFAULT FALSE,
    archived BOOLEAN DEFAULT FALSE,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, space_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_spaces_user ON clickup_spaces(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_spaces_workspace ON clickup_spaces(user_id, workspace_id);

-- ClickUp folders
CREATE TABLE IF NOT EXISTS clickup_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    folder_id VARCHAR(100) NOT NULL,
    space_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    hidden BOOLEAN DEFAULT FALSE,
    archived BOOLEAN DEFAULT FALSE,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, folder_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_folders_user ON clickup_folders(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_folders_space ON clickup_folders(user_id, space_id);

-- ClickUp lists
CREATE TABLE IF NOT EXISTS clickup_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(100) NOT NULL,
    folder_id VARCHAR(100),
    space_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    archived BOOLEAN DEFAULT FALSE,
    task_count INT DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, list_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_lists_user ON clickup_lists(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_lists_folder ON clickup_lists(user_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_clickup_lists_space ON clickup_lists(user_id, space_id);

-- ClickUp tasks
CREATE TABLE IF NOT EXISTS clickup_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(100) NOT NULL,
    custom_id VARCHAR(100),
    list_id VARCHAR(100) NOT NULL,
    folder_id VARCHAR(100),
    space_id VARCHAR(100) NOT NULL,

    -- Task details
    name VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(100),
    status_color VARCHAR(50),
    priority VARCHAR(50),
    priority_color VARCHAR(50),

    -- Dates
    due_date TIMESTAMPTZ,
    start_date TIMESTAMPTZ,
    date_created TIMESTAMPTZ,
    date_updated TIMESTAMPTZ,
    date_closed TIMESTAMPTZ,

    -- Time tracking
    time_spent BIGINT DEFAULT 0,
    time_estimate BIGINT,

    -- Hierarchy
    parent_task_id VARCHAR(100),

    -- Assignments
    assignees JSONB DEFAULT '[]',
    creator JSONB,
    tags JSONB DEFAULT '[]',

    -- URLs
    url TEXT,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_clickup_tasks_user ON clickup_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_list ON clickup_tasks(user_id, list_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_space ON clickup_tasks(user_id, space_id);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_status ON clickup_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_clickup_tasks_due ON clickup_tasks(user_id, due_date);

-- ============================================================================
-- AIRTABLE TABLES
-- ============================================================================

-- Airtable bases
CREATE TABLE IF NOT EXISTS airtable_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permission_level VARCHAR(50), -- none, read, comment, edit, create
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, base_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_bases_user ON airtable_bases(user_id);

-- Airtable tables (within bases)
CREATE TABLE IF NOT EXISTS airtable_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    table_id VARCHAR(100) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    primary_field_id VARCHAR(100),
    fields JSONB DEFAULT '[]',
    views JSONB DEFAULT '[]',
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, table_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_tables_user ON airtable_tables(user_id);
CREATE INDEX IF NOT EXISTS idx_airtable_tables_base ON airtable_tables(user_id, base_id);

-- Airtable records
CREATE TABLE IF NOT EXISTS airtable_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    record_id VARCHAR(100) NOT NULL,
    table_id VARCHAR(100) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    fields JSONB DEFAULT '{}',
    created_time_airtable TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, record_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_records_user ON airtable_records(user_id);
CREATE INDEX IF NOT EXISTS idx_airtable_records_table ON airtable_records(user_id, table_id);
CREATE INDEX IF NOT EXISTS idx_airtable_records_base ON airtable_records(user_id, base_id);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Apply update triggers to new tables
DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'clickup_workspaces', 'clickup_spaces', 'clickup_folders', 'clickup_lists', 'clickup_tasks',
        'airtable_bases', 'airtable_tables', 'airtable_records'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_integration_updated_at()', t, t);
    END LOOP;
END $$;

-- Add comments
COMMENT ON TABLE clickup_workspaces IS 'ClickUp workspaces (Teams in API v2)';
COMMENT ON TABLE clickup_spaces IS 'ClickUp spaces within workspaces';
COMMENT ON TABLE clickup_folders IS 'ClickUp folders within spaces';
COMMENT ON TABLE clickup_lists IS 'ClickUp lists (task containers)';
COMMENT ON TABLE clickup_tasks IS 'ClickUp tasks with full metadata';
COMMENT ON TABLE airtable_bases IS 'Airtable bases (workspaces)';
COMMENT ON TABLE airtable_tables IS 'Airtable tables within bases';
COMMENT ON TABLE airtable_records IS 'Airtable records (rows) with field data';
