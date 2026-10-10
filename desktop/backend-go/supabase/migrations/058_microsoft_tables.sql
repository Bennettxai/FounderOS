-- Migration 035: Microsoft 365 Integration Tables
-- This adds storage for Microsoft 365 data (Outlook, OneDrive, To Do, etc.)

-- ============================================================================
-- MICROSOFT OAUTH TOKENS
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expiry TIMESTAMPTZ,
    scopes TEXT[],
    microsoft_id VARCHAR(255),
    microsoft_email VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_microsoft_tokens_user ON microsoft_oauth_tokens(user_id);

-- ============================================================================
-- MICROSOFT OUTLOOK MAIL
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_mail_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,

    -- Conversation threading
    conversation_id VARCHAR(255),

    -- Message details
    subject TEXT,
    body_preview TEXT,
    body_content TEXT,
    body_content_type VARCHAR(50), -- text, html
    importance VARCHAR(50), -- low, normal, high

    -- Sender
    from_email VARCHAR(255),
    from_name VARCHAR(255),

    -- Recipients
    to_recipients JSONB DEFAULT '[]',
    cc_recipients JSONB DEFAULT '[]',
    bcc_recipients JSONB DEFAULT '[]',
    reply_to JSONB DEFAULT '[]',

    -- Flags
    is_read BOOLEAN DEFAULT FALSE,
    is_draft BOOLEAN DEFAULT FALSE,
    has_attachments BOOLEAN DEFAULT FALSE,

    -- Folder
    folder_id VARCHAR(255),
    folder_name VARCHAR(255),

    -- Categories/Labels
    categories JSONB DEFAULT '[]',
    flag_status VARCHAR(50), -- notFlagged, flagged, complete

    -- Attachments metadata
    attachments JSONB DEFAULT '[]',

    -- Timestamps
    received_datetime TIMESTAMPTZ,
    sent_datetime TIMESTAMPTZ,
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_mail_user ON microsoft_mail_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_conversation ON microsoft_mail_messages(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_folder ON microsoft_mail_messages(user_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_ms_mail_received ON microsoft_mail_messages(user_id, received_datetime DESC);
CREATE INDEX IF NOT EXISTS idx_ms_mail_from ON microsoft_mail_messages(user_id, from_email);

-- ============================================================================
-- MICROSOFT OUTLOOK CALENDAR
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    event_id VARCHAR(255) NOT NULL,

    -- Calendar
    calendar_id VARCHAR(255),
    calendar_name VARCHAR(255),

    -- Event details
    subject VARCHAR(500),
    body_preview TEXT,
    body_content TEXT,
    body_content_type VARCHAR(50),

    -- Location
    location_display_name VARCHAR(500),
    location_address JSONB,
    location_coordinates JSONB, -- lat, long

    -- Time
    start_datetime TIMESTAMPTZ,
    start_timezone VARCHAR(100),
    end_datetime TIMESTAMPTZ,
    end_timezone VARCHAR(100),
    is_all_day BOOLEAN DEFAULT FALSE,

    -- Recurrence
    recurrence JSONB, -- pattern, range
    series_master_id VARCHAR(255),
    type VARCHAR(50), -- singleInstance, occurrence, exception, seriesMaster

    -- Attendees
    attendees JSONB DEFAULT '[]',
    organizer_email VARCHAR(255),
    organizer_name VARCHAR(255),

    -- Online meeting
    is_online_meeting BOOLEAN DEFAULT FALSE,
    online_meeting_provider VARCHAR(100), -- teamsForBusiness, skypeForBusiness, etc.
    online_meeting_url TEXT,
    online_meeting_join_url TEXT,

    -- Response
    response_status VARCHAR(50), -- none, organizer, accepted, tentative, declined
    response_time TIMESTAMPTZ,

    -- Flags
    importance VARCHAR(50),
    sensitivity VARCHAR(50), -- normal, personal, private, confidential
    show_as VARCHAR(50), -- free, tentative, busy, oof, workingElsewhere
    is_cancelled BOOLEAN DEFAULT FALSE,
    is_reminder_on BOOLEAN DEFAULT TRUE,
    reminder_minutes_before_start INT DEFAULT 15,

    -- Categories
    categories JSONB DEFAULT '[]',

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_calendar_user ON microsoft_calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_calendar ON microsoft_calendar_events(user_id, calendar_id);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_start ON microsoft_calendar_events(user_id, start_datetime);
CREATE INDEX IF NOT EXISTS idx_ms_calendar_series ON microsoft_calendar_events(user_id, series_master_id);

-- ============================================================================
-- MICROSOFT CONTACTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    contact_id VARCHAR(255) NOT NULL,

    -- Name
    display_name VARCHAR(255),
    given_name VARCHAR(255),
    surname VARCHAR(255),
    middle_name VARCHAR(255),
    nickname VARCHAR(255),
    title VARCHAR(50), -- Mr., Ms., Dr., etc.

    -- Contact info
    email_addresses JSONB DEFAULT '[]',
    phone_numbers JSONB DEFAULT '[]',
    addresses JSONB DEFAULT '[]',
    im_addresses JSONB DEFAULT '[]',
    websites JSONB DEFAULT '[]',

    -- Organization
    company_name VARCHAR(255),
    department VARCHAR(255),
    job_title VARCHAR(255),
    office_location VARCHAR(255),
    profession VARCHAR(255),
    manager VARCHAR(255),
    assistant_name VARCHAR(255),

    -- Personal
    birthday DATE,
    spouse_name VARCHAR(255),
    personal_notes TEXT,

    -- Photo
    photo_url TEXT,

    -- Category
    categories JSONB DEFAULT '[]',

    -- Parent folder
    parent_folder_id VARCHAR(255),

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, contact_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_contacts_user ON microsoft_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_contacts_name ON microsoft_contacts(user_id, display_name);
CREATE INDEX IF NOT EXISTS idx_ms_contacts_company ON microsoft_contacts(user_id, company_name);

-- ============================================================================
-- MICROSOFT ONEDRIVE
-- ============================================================================

CREATE TABLE IF NOT EXISTS microsoft_onedrive_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    item_id VARCHAR(255) NOT NULL,

    -- File info
    name VARCHAR(500) NOT NULL,
    description TEXT,
    mime_type VARCHAR(255),
    size_bytes BIGINT,

    -- Path info
    parent_reference_id VARCHAR(255),
    parent_reference_path TEXT,
    web_url TEXT,

    -- Type
    is_folder BOOLEAN DEFAULT FALSE,
    folder_child_count INT,

    -- File specific
    file_hash VARCHAR(255), -- quickXorHash or sha1Hash

    -- Sharing
    shared BOOLEAN DEFAULT FALSE,
    shared_scope VARCHAR(50), -- anonymous, organization, users
    shared_link JSONB,

    -- Permissions
    permissions JSONB DEFAULT '[]',

    -- Created/Modified by
    created_by_user_email VARCHAR(255),
    created_by_user_name VARCHAR(255),
    last_modified_by_user_email VARCHAR(255),
    last_modified_by_user_name VARCHAR(255),

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Download
    download_url TEXT,

    -- Thumbnails
    thumbnails JSONB DEFAULT '[]',

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_onedrive_user ON microsoft_onedrive_files(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_parent ON microsoft_onedrive_files(user_id, parent_reference_id);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_folder ON microsoft_onedrive_files(user_id, is_folder);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_name ON microsoft_onedrive_files(user_id, name);
CREATE INDEX IF NOT EXISTS idx_ms_onedrive_modified ON microsoft_onedrive_files(user_id, last_modified_datetime DESC);

-- ============================================================================
-- MICROSOFT TO DO
-- ============================================================================

-- Task Lists
CREATE TABLE IF NOT EXISTS microsoft_todo_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(255) NOT NULL,

    -- List info
    display_name VARCHAR(255) NOT NULL,
    is_owner BOOLEAN DEFAULT TRUE,
    is_shared BOOLEAN DEFAULT FALSE,
    wellknown_list_name VARCHAR(50), -- none, defaultList, flaggedEmails, unknownFutureValue

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, list_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_todo_lists_user ON microsoft_todo_lists(user_id);

-- Tasks
CREATE TABLE IF NOT EXISTS microsoft_todo_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(255) NOT NULL,

    -- Task details
    title VARCHAR(500) NOT NULL,
    body_content TEXT,
    body_content_type VARCHAR(50),
    importance VARCHAR(50), -- low, normal, high
    status VARCHAR(50), -- notStarted, inProgress, completed, waitingOnOthers, deferred

    -- Dates
    due_datetime TIMESTAMPTZ,
    due_timezone VARCHAR(100),
    start_datetime TIMESTAMPTZ,
    start_timezone VARCHAR(100),
    completed_datetime TIMESTAMPTZ,
    completed_timezone VARCHAR(100),

    -- Recurrence
    recurrence JSONB,
    is_reminder_on BOOLEAN DEFAULT FALSE,
    reminder_datetime TIMESTAMPTZ,

    -- Categories
    categories JSONB DEFAULT '[]',

    -- Linked resources
    linked_resources JSONB DEFAULT '[]',

    -- Checklist items
    checklist_items JSONB DEFAULT '[]',

    -- Timestamps
    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,

    -- Sync tracking
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_user ON microsoft_todo_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_list ON microsoft_todo_tasks(user_id, list_id);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_status ON microsoft_todo_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_ms_todo_tasks_due ON microsoft_todo_tasks(user_id, due_datetime);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'microsoft_oauth_tokens', 'microsoft_mail_messages', 'microsoft_calendar_events',
        'microsoft_contacts', 'microsoft_onedrive_files', 'microsoft_todo_lists', 'microsoft_todo_tasks'
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
COMMENT ON TABLE microsoft_oauth_tokens IS 'Microsoft 365 OAuth tokens for users';
COMMENT ON TABLE microsoft_mail_messages IS 'Synced Outlook email messages';
COMMENT ON TABLE microsoft_calendar_events IS 'Synced Outlook calendar events';
COMMENT ON TABLE microsoft_contacts IS 'Synced Outlook contacts';
COMMENT ON TABLE microsoft_onedrive_files IS 'Synced OneDrive files and folders';
COMMENT ON TABLE microsoft_todo_lists IS 'Microsoft To Do task lists';
COMMENT ON TABLE microsoft_todo_tasks IS 'Microsoft To Do tasks';
