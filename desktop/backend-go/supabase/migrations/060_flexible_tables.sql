-- Migration 036: Flexible Tables System
-- User-created custom tables similar to Airtable/NocoDB
-- Each user can create their own tables, fields, records, and views
-- This is SEPARATE from synced integration data (airtable_*, notion_*, etc.)

-- ============================================================================
-- CUSTOM TABLES (User-created databases)
-- ============================================================================

-- Custom tables created by users (like Airtable bases but user-owned)
CREATE TABLE IF NOT EXISTS custom_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Table metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(50),

    -- Organization
    workspace_id UUID,  -- Optional: group tables into workspaces
    folder_id UUID,     -- Optional: organize into folders

    -- Settings
    settings JSONB DEFAULT '{
        "defaultView": "grid",
        "allowDuplicates": true,
        "trackHistory": true
    }',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_tables_user ON custom_tables(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_tables_workspace ON custom_tables(workspace_id) WHERE workspace_id IS NOT NULL;

-- ============================================================================
-- CUSTOM FIELDS (Columns/Properties)
-- ============================================================================

-- Field types enum
DO $$ BEGIN
    CREATE TYPE custom_field_type AS ENUM (
        'text',           -- Single line text
        'long_text',      -- Multi-line text / rich text
        'number',         -- Numeric value
        'currency',       -- Money with currency
        'percent',        -- Percentage
        'date',           -- Date only
        'datetime',       -- Date and time
        'checkbox',       -- Boolean
        'select',         -- Single select from options
        'multi_select',   -- Multiple select from options
        'user',           -- Reference to team member
        'email',          -- Email address
        'phone',          -- Phone number
        'url',            -- Web URL
        'attachment',     -- File attachments
        'relation',       -- Link to another table
        'lookup',         -- Pull data from related records
        'formula',        -- Calculated field
        'rollup',         -- Aggregate related records
        'count',          -- Count related records
        'created_time',   -- Auto: when record was created
        'modified_time',  -- Auto: when record was last modified
        'created_by',     -- Auto: who created record
        'modified_by',    -- Auto: who last modified record
        'autonumber',     -- Auto-incrementing number
        'rating',         -- Star rating (1-5)
        'duration',       -- Time duration
        'json'            -- Raw JSON data
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Custom fields (columns) for tables
CREATE TABLE IF NOT EXISTS custom_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,

    -- Field definition
    name VARCHAR(255) NOT NULL,
    field_type custom_field_type NOT NULL DEFAULT 'text',
    description TEXT,

    -- Ordering
    position INT NOT NULL DEFAULT 0,

    -- Field configuration (varies by type)
    config JSONB DEFAULT '{}',
    -- Examples:
    -- select/multi_select: {"options": [{"id": "...", "name": "...", "color": "..."}]}
    -- relation: {"linkedTableId": "...", "symmetric": true}
    -- formula: {"expression": "..."}
    -- number: {"precision": 2, "format": "decimal"}
    -- currency: {"symbol": "$", "precision": 2}

    -- Validation
    required BOOLEAN DEFAULT FALSE,
    unique_values BOOLEAN DEFAULT FALSE,

    -- Visibility
    hidden BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(table_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_fields_table ON custom_fields(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_fields_position ON custom_fields(table_id, position);

-- ============================================================================
-- CUSTOM FIELD OPTIONS (For select/multi_select fields)
-- ============================================================================

CREATE TABLE IF NOT EXISTS custom_field_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_id UUID NOT NULL REFERENCES custom_fields(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    position INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(field_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_field_options_field ON custom_field_options(field_id);

-- ============================================================================
-- CUSTOM RECORDS (Rows)
-- ============================================================================

CREATE TABLE IF NOT EXISTS custom_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,

    -- All field values stored as JSONB
    -- Format: {"field_id": value, "field_id": value}
    data JSONB DEFAULT '{}',

    -- Row ordering within views
    position INT,

    -- Audit
    created_by VARCHAR(255),
    modified_by VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_records_table ON custom_records(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_records_created ON custom_records(table_id, created_at DESC);

-- GIN index for searching within JSONB data
CREATE INDEX IF NOT EXISTS idx_custom_records_data ON custom_records USING GIN (data);

-- ============================================================================
-- CUSTOM VIEWS (Saved view configurations)
-- ============================================================================

-- View types
DO $$ BEGIN
    CREATE TYPE custom_view_type AS ENUM (
        'grid',       -- Spreadsheet view
        'kanban',     -- Kanban board
        'calendar',   -- Calendar view
        'gallery',    -- Card gallery
        'timeline',   -- Gantt/timeline
        'list',       -- Simple list
        'form'        -- Data entry form
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS custom_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,

    -- View metadata
    name VARCHAR(255) NOT NULL,
    view_type custom_view_type NOT NULL DEFAULT 'grid',
    description TEXT,

    -- Is this the default view for the table?
    is_default BOOLEAN DEFAULT FALSE,

    -- View configuration
    config JSONB DEFAULT '{
        "visibleFields": [],
        "fieldWidths": {},
        "rowHeight": "medium"
    }',

    -- Filters
    filters JSONB DEFAULT '[]',
    -- Format: [{"fieldId": "...", "operator": "eq", "value": "..."}]

    -- Sorting
    sorts JSONB DEFAULT '[]',
    -- Format: [{"fieldId": "...", "direction": "asc"}]

    -- Grouping (for kanban, etc.)
    group_by UUID,  -- Field ID to group by

    -- View-specific settings
    -- Kanban: {"stackField": "...", "coverField": "..."}
    -- Calendar: {"dateField": "...", "titleField": "..."}
    -- Gallery: {"coverField": "...", "titleField": "..."}
    view_settings JSONB DEFAULT '{}',

    -- Ordering
    position INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_views_table ON custom_views(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_views_default ON custom_views(table_id, is_default) WHERE is_default = TRUE;

-- ============================================================================
-- RECORD HISTORY (For tracking changes)
-- ============================================================================

CREATE TABLE IF NOT EXISTS custom_record_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id UUID NOT NULL REFERENCES custom_records(id) ON DELETE CASCADE,

    -- What changed
    field_id UUID,  -- NULL for record-level actions (create, delete)
    action VARCHAR(50) NOT NULL,  -- 'create', 'update', 'delete', 'restore'

    -- Values
    old_value JSONB,
    new_value JSONB,

    -- Who changed it
    changed_by VARCHAR(255),

    -- When
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_record_history_record ON custom_record_history(record_id);
CREATE INDEX IF NOT EXISTS idx_custom_record_history_time ON custom_record_history(changed_at DESC);

-- ============================================================================
-- WORKSPACES (Optional organization layer)
-- ============================================================================

CREATE TABLE IF NOT EXISTS custom_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(50),

    -- Sharing settings
    visibility VARCHAR(20) DEFAULT 'private',  -- private, team, public

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_workspaces_user ON custom_workspaces(user_id);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update timestamps
CREATE OR REPLACE FUNCTION update_custom_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'custom_tables', 'custom_fields', 'custom_records',
        'custom_views', 'custom_workspaces'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_custom_updated_at()', t, t);
    END LOOP;
END $$;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE custom_tables IS 'User-created database tables (like Airtable bases)';
COMMENT ON TABLE custom_fields IS 'Field definitions for custom tables';
COMMENT ON TABLE custom_field_options IS 'Options for select/multi_select fields';
COMMENT ON TABLE custom_records IS 'Rows of data in custom tables';
COMMENT ON TABLE custom_views IS 'Saved view configurations for custom tables';
COMMENT ON TABLE custom_record_history IS 'Change history for custom records';
COMMENT ON TABLE custom_workspaces IS 'Workspaces to organize custom tables';
