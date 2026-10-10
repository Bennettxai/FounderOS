-- Migration 033: Fathom Analytics and Google Docs Integration Tables
-- This adds storage for Fathom analytics data and Google Docs/Drive content

-- ============================================================================
-- FATHOM ANALYTICS TABLES
-- ============================================================================

-- Fathom sites (website properties)
CREATE TABLE IF NOT EXISTS fathom_sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    name VARCHAR(255),
    sharing_url TEXT,
    share_config VARCHAR(50),
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id)
);

CREATE INDEX IF NOT EXISTS idx_fathom_sites_user ON fathom_sites(user_id);

-- Fathom aggregations (daily analytics data)
CREATE TABLE IF NOT EXISTS fathom_aggregations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    pageviews INT DEFAULT 0,
    avg_duration DECIMAL(10,2) DEFAULT 0,
    bounce_rate DECIMAL(5,2) DEFAULT 0,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, date)
);

CREATE INDEX IF NOT EXISTS idx_fathom_agg_user_site ON fathom_aggregations(user_id, site_id);
CREATE INDEX IF NOT EXISTS idx_fathom_agg_date ON fathom_aggregations(user_id, site_id, date DESC);

-- Fathom page-level analytics (grouped by pathname)
CREATE TABLE IF NOT EXISTS fathom_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    pathname VARCHAR(500) NOT NULL,
    hostname VARCHAR(255),
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    pageviews INT DEFAULT 0,
    avg_duration DECIMAL(10,2) DEFAULT 0,
    bounce_rate DECIMAL(5,2) DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, pathname, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_pages_user_site ON fathom_pages(user_id, site_id);
CREATE INDEX IF NOT EXISTS idx_fathom_pages_pathname ON fathom_pages(user_id, pathname);

-- Fathom referrers analytics
CREATE TABLE IF NOT EXISTS fathom_referrers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    referrer VARCHAR(500) NOT NULL,
    visits INT DEFAULT 0,
    uniques INT DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, referrer, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_referrers_user_site ON fathom_referrers(user_id, site_id);

-- Fathom custom events
CREATE TABLE IF NOT EXISTS fathom_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    event_id VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    count INT DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, site_id, event_id, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_fathom_events_user_site ON fathom_events(user_id, site_id);

-- ============================================================================
-- GOOGLE DRIVE/DOCS TABLES
-- ============================================================================

-- Google Drive files
CREATE TABLE IF NOT EXISTS google_drive_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    file_id VARCHAR(255) NOT NULL,

    -- File metadata
    name VARCHAR(500) NOT NULL,
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),
    size_bytes BIGINT,

    -- Folder/hierarchy info
    parent_folder_id VARCHAR(255),
    parent_folder_name VARCHAR(500),
    path TEXT,

    -- Permissions/sharing
    shared BOOLEAN DEFAULT FALSE,
    sharing_user VARCHAR(255),
    permissions JSONB DEFAULT '[]',

    -- Content info
    web_view_link TEXT,
    web_content_link TEXT,
    thumbnail_link TEXT,
    icon_link TEXT,

    -- Timestamps from Google
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
    viewed_by_me_time TIMESTAMPTZ,

    -- Owners and modifiers
    owners JSONB DEFAULT '[]',
    last_modifying_user JSONB,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_drive_files_user ON google_drive_files(user_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_parent ON google_drive_files(user_id, parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_mime ON google_drive_files(user_id, mime_type);
CREATE INDEX IF NOT EXISTS idx_drive_files_modified ON google_drive_files(user_id, modified_time DESC);

-- Google Docs content (extracted from Drive for document-specific data)
CREATE TABLE IF NOT EXISTS google_docs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    document_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Document metadata
    title VARCHAR(500) NOT NULL,

    -- Content (plain text extraction for search)
    body_text TEXT,
    word_count INT DEFAULT 0,

    -- Document structure
    headers JSONB DEFAULT '[]',

    -- Document info
    locale VARCHAR(20),

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_docs_user ON google_docs(user_id);
CREATE INDEX IF NOT EXISTS idx_docs_title ON google_docs(user_id, title);
CREATE INDEX IF NOT EXISTS idx_docs_modified ON google_docs(user_id, modified_time DESC);
CREATE INDEX IF NOT EXISTS idx_docs_search ON google_docs USING GIN(to_tsvector('english', body_text));

-- Google Sheets
CREATE TABLE IF NOT EXISTS google_sheets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    spreadsheet_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Spreadsheet metadata
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),
    time_zone VARCHAR(100),

    -- Sheet info
    sheet_count INT DEFAULT 0,
    sheets JSONB DEFAULT '[]', -- Array of sheet names and properties

    -- Named ranges
    named_ranges JSONB DEFAULT '[]',

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, spreadsheet_id)
);

CREATE INDEX IF NOT EXISTS idx_sheets_user ON google_sheets(user_id);
CREATE INDEX IF NOT EXISTS idx_sheets_title ON google_sheets(user_id, title);

-- Google Slides presentations
CREATE TABLE IF NOT EXISTS google_slides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    presentation_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,

    -- Presentation metadata
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),

    -- Slide info
    slide_count INT DEFAULT 0,
    slides JSONB DEFAULT '[]', -- Array of slide info

    -- Page size
    page_width DECIMAL(10,2),
    page_height DECIMAL(10,2),

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, presentation_id)
);

CREATE INDEX IF NOT EXISTS idx_slides_user ON google_slides(user_id);
CREATE INDEX IF NOT EXISTS idx_slides_title ON google_slides(user_id, title);

-- ============================================================================
-- GOOGLE CONTACTS TABLES
-- ============================================================================

-- Google contacts
CREATE TABLE IF NOT EXISTS google_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL, -- people/c12345

    -- Name
    display_name VARCHAR(255),
    given_name VARCHAR(255),
    family_name VARCHAR(255),
    middle_name VARCHAR(255),

    -- Contact info
    emails JSONB DEFAULT '[]',
    phone_numbers JSONB DEFAULT '[]',
    addresses JSONB DEFAULT '[]',

    -- Organization
    organization VARCHAR(255),
    job_title VARCHAR(255),
    department VARCHAR(255),

    -- Photo
    photo_url TEXT,

    -- Grouping
    contact_groups JSONB DEFAULT '[]',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, resource_name)
);

CREATE INDEX IF NOT EXISTS idx_contacts_user ON google_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON google_contacts(user_id, display_name);
CREATE INDEX IF NOT EXISTS idx_contacts_org ON google_contacts(user_id, organization);

-- ============================================================================
-- GOOGLE TASKS TABLES
-- ============================================================================

-- Google task lists
CREATE TABLE IF NOT EXISTS google_task_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,

    -- List info
    title VARCHAR(255) NOT NULL,
    kind VARCHAR(100),

    -- Timestamps
    updated TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_list_id)
);

CREATE INDEX IF NOT EXISTS idx_task_lists_user ON google_task_lists(user_id);

-- Google tasks
CREATE TABLE IF NOT EXISTS google_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,

    -- Task info
    title VARCHAR(500) NOT NULL,
    notes TEXT,
    status VARCHAR(50) DEFAULT 'needsAction', -- needsAction, completed

    -- Due date
    due TIMESTAMPTZ,

    -- Completion
    completed TIMESTAMPTZ,
    deleted BOOLEAN DEFAULT FALSE,
    hidden BOOLEAN DEFAULT FALSE,

    -- Hierarchy
    parent_task_id VARCHAR(255),
    position VARCHAR(100),

    -- Links
    links JSONB DEFAULT '[]',

    -- Timestamps
    updated TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_google_tasks_user ON google_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_google_tasks_list ON google_tasks(user_id, task_list_id);
CREATE INDEX IF NOT EXISTS idx_google_tasks_status ON google_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_google_tasks_due ON google_tasks(user_id, due);

-- ============================================================================
-- HUBSPOT CRM TABLES (for synced HubSpot data)
-- ============================================================================

-- HubSpot contacts
CREATE TABLE IF NOT EXISTS hubspot_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Contact info
    email VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    phone VARCHAR(100),

    -- Company
    company VARCHAR(255),
    job_title VARCHAR(255),

    -- Lead info
    lifecycle_stage VARCHAR(100),
    lead_status VARCHAR(100),

    -- Owner
    owner_id VARCHAR(100),

    -- Properties (all other HubSpot properties)
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_user ON hubspot_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_email ON hubspot_contacts(user_id, email);

-- HubSpot companies
CREATE TABLE IF NOT EXISTS hubspot_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Company info
    name VARCHAR(500),
    domain VARCHAR(255),
    industry VARCHAR(255),

    -- Company size
    number_of_employees INT,
    annual_revenue DECIMAL(15,2),

    -- Location
    city VARCHAR(255),
    state VARCHAR(255),
    country VARCHAR(255),

    -- Owner
    owner_id VARCHAR(100),

    -- Properties
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_companies_user ON hubspot_companies(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_companies_name ON hubspot_companies(user_id, name);

-- HubSpot deals
CREATE TABLE IF NOT EXISTS hubspot_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,

    -- Deal info
    deal_name VARCHAR(500),
    amount DECIMAL(15,2),
    pipeline VARCHAR(255),
    deal_stage VARCHAR(255),

    -- Close date
    close_date DATE,

    -- Owner
    owner_id VARCHAR(100),

    -- Associated objects
    associated_company_ids JSONB DEFAULT '[]',
    associated_contact_ids JSONB DEFAULT '[]',

    -- Properties
    properties JSONB DEFAULT '{}',

    -- Timestamps
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_deals_user ON hubspot_deals(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_stage ON hubspot_deals(user_id, deal_stage);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_close ON hubspot_deals(user_id, close_date);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update timestamp triggers
CREATE OR REPLACE FUNCTION update_integration_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers to new tables
DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'fathom_sites', 'fathom_pages', 'fathom_referrers', 'fathom_events',
        'google_drive_files', 'google_docs', 'google_sheets', 'google_slides',
        'google_contacts', 'google_task_lists', 'google_tasks',
        'hubspot_contacts', 'hubspot_companies', 'hubspot_deals'
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
COMMENT ON TABLE fathom_sites IS 'Fathom Analytics website properties';
COMMENT ON TABLE fathom_aggregations IS 'Daily aggregated analytics data from Fathom';
COMMENT ON TABLE google_drive_files IS 'Synced Google Drive files metadata';
COMMENT ON TABLE google_docs IS 'Synced Google Docs with text extraction';
COMMENT ON TABLE google_sheets IS 'Synced Google Sheets metadata';
COMMENT ON TABLE google_slides IS 'Synced Google Slides presentations';
COMMENT ON TABLE google_contacts IS 'Synced Google Contacts';
COMMENT ON TABLE google_tasks IS 'Synced Google Tasks';
COMMENT ON TABLE hubspot_contacts IS 'Synced HubSpot CRM contacts';
COMMENT ON TABLE hubspot_companies IS 'Synced HubSpot CRM companies';
COMMENT ON TABLE hubspot_deals IS 'Synced HubSpot CRM deals';
