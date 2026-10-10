-- BusinessOS SQLite Schema for Electron Desktop App
-- This mirrors the PostgreSQL schema with SQLite-compatible syntax
-- Includes sync columns for offline-first functionality

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- ============================================
-- SYNC METADATA TABLE
-- Tracks sync state for the entire database
-- ============================================
CREATE TABLE IF NOT EXISTS sync_metadata (
    id INTEGER PRIMARY KEY,
    last_sync_at TEXT,
    sync_token TEXT,
    server_url TEXT,
    user_id TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- ============================================
-- USER DATA (cached from server auth)
-- ============================================
CREATE TABLE IF NOT EXISTS local_user (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT,
    avatar_url TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- ============================================
-- CONTEXTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS contexts (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT DEFAULT 'CUSTOM' CHECK(type IN ('PERSON', 'BUSINESS', 'PROJECT', 'CUSTOM', 'document', 'DOCUMENT')),
    content TEXT,
    structured_data TEXT, -- JSON
    system_prompt_template TEXT,
    blocks TEXT DEFAULT '[]', -- JSON
    cover_image TEXT,
    icon TEXT,
    parent_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    is_template INTEGER DEFAULT 0,
    is_archived INTEGER DEFAULT 0,
    last_edited_at TEXT,
    word_count INTEGER DEFAULT 0,
    is_public INTEGER DEFAULT 0,
    share_id TEXT UNIQUE,
    property_schema TEXT DEFAULT '[]', -- JSON
    properties TEXT DEFAULT '{}', -- JSON
    client_id TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced' CHECK(sync_status IN ('synced', 'pending_create', 'pending_update', 'pending_delete', 'conflict')),
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_contexts_user_id ON contexts(user_id);
CREATE INDEX IF NOT EXISTS idx_contexts_parent_id ON contexts(parent_id);
CREATE INDEX IF NOT EXISTS idx_contexts_sync_status ON contexts(sync_status);

-- ============================================
-- CONVERSATIONS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    title TEXT DEFAULT 'New Conversation',
    context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced' CHECK(sync_status IN ('synced', 'pending_create', 'pending_update', 'pending_delete', 'conflict')),
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_sync_status ON conversations(sync_status);

-- ============================================
-- MESSAGES TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('USER', 'ASSISTANT', 'SYSTEM', 'user', 'assistant', 'system')),
    content TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    message_metadata TEXT, -- JSON
    -- Sync columns
    sync_status TEXT DEFAULT 'synced' CHECK(sync_status IN ('synced', 'pending_create', 'pending_update', 'pending_delete', 'conflict')),
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_sync_status ON messages(sync_status);

-- ============================================
-- CONVERSATION TAGS
-- ============================================
CREATE TABLE IF NOT EXISTS conversation_tags (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- PROJECTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'ACTIVE' CHECK(status IN ('ACTIVE', 'PAUSED', 'COMPLETED', 'ARCHIVED')),
    priority TEXT DEFAULT 'MEDIUM' CHECK(priority IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW')),
    client_name TEXT,
    project_type TEXT DEFAULT 'internal',
    project_metadata TEXT, -- JSON
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_sync_status ON projects(sync_status);

-- ============================================
-- PROJECT NOTES
-- ============================================
CREATE TABLE IF NOT EXISTS project_notes (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- PROJECT CONVERSATIONS (junction table)
-- ============================================
CREATE TABLE IF NOT EXISTS project_conversations (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    linked_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    last_synced_at TEXT,
    PRIMARY KEY (project_id, conversation_id)
);

-- ============================================
-- ARTIFACTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    conversation_id TEXT REFERENCES conversations(id) ON DELETE SET NULL,
    message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('CODE', 'DOCUMENT', 'MARKDOWN', 'REACT', 'HTML', 'SVG')),
    language TEXT,
    content TEXT NOT NULL,
    summary TEXT,
    version INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_artifacts_user_id ON artifacts(user_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_sync_status ON artifacts(sync_status);

-- ============================================
-- ARTIFACT VERSIONS
-- ============================================
CREATE TABLE IF NOT EXISTS artifact_versions (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    artifact_id TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- NODES TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    parent_id TEXT REFERENCES nodes(id) ON DELETE SET NULL,
    context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('BUSINESS', 'PROJECT', 'LEARNING', 'OPERATIONAL')),
    health TEXT DEFAULT 'NOT_STARTED' CHECK(health IN ('HEALTHY', 'NEEDS_ATTENTION', 'CRITICAL', 'NOT_STARTED')),
    purpose TEXT,
    current_status TEXT,
    this_week_focus TEXT, -- JSON
    decision_queue TEXT, -- JSON
    delegation_ready TEXT, -- JSON
    is_active INTEGER DEFAULT 0,
    is_archived INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_nodes_user_id ON nodes(user_id);
CREATE INDEX IF NOT EXISTS idx_nodes_sync_status ON nodes(sync_status);

-- ============================================
-- NODE METRICS
-- ============================================
CREATE TABLE IF NOT EXISTS node_metrics (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    metric_value TEXT NOT NULL,
    recorded_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- TEAM MEMBERS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS team_members (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    avatar_url TEXT,
    status TEXT DEFAULT 'AVAILABLE' CHECK(status IN ('AVAILABLE', 'BUSY', 'OVERLOADED', 'OOO')),
    capacity INTEGER DEFAULT 0,
    manager_id TEXT REFERENCES team_members(id) ON DELETE SET NULL,
    skills TEXT, -- JSON
    hourly_rate REAL,
    share_calendar INTEGER DEFAULT 0,
    calendar_user_id TEXT,
    joined_at TEXT DEFAULT (datetime('now')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);
CREATE INDEX IF NOT EXISTS idx_team_members_sync_status ON team_members(sync_status);

-- ============================================
-- TEAM MEMBER ACTIVITIES
-- ============================================
CREATE TABLE IF NOT EXISTS team_member_activities (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    member_id TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    activity_type TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- TASKS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'todo' CHECK(status IN ('todo', 'in_progress', 'done', 'cancelled')),
    priority TEXT DEFAULT 'medium' CHECK(priority IN ('critical', 'high', 'medium', 'low')),
    due_date TEXT,
    completed_at TEXT,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    assignee_id TEXT REFERENCES team_members(id) ON DELETE SET NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_sync_status ON tasks(sync_status);

-- ============================================
-- FOCUS ITEMS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS focus_items (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    text TEXT NOT NULL,
    completed INTEGER DEFAULT 0,
    focus_date TEXT DEFAULT (datetime('now')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_focus_items_user_id ON focus_items(user_id);
CREATE INDEX IF NOT EXISTS idx_focus_items_sync_status ON focus_items(sync_status);

-- ============================================
-- DAILY LOGS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS daily_logs (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    date TEXT NOT NULL,
    content TEXT NOT NULL,
    transcription_source TEXT,
    extracted_actions TEXT, -- JSON
    extracted_patterns TEXT, -- JSON
    energy_level INTEGER,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT,
    UNIQUE(user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_logs_user_id ON daily_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_logs_sync_status ON daily_logs(sync_status);

-- ============================================
-- USER SETTINGS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS user_settings (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT UNIQUE NOT NULL,
    default_model TEXT,
    email_notifications INTEGER DEFAULT 1,
    daily_summary INTEGER DEFAULT 0,
    theme TEXT DEFAULT 'light',
    sidebar_collapsed INTEGER DEFAULT 0,
    share_analytics INTEGER DEFAULT 1,
    custom_settings TEXT, -- JSON
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- ============================================
-- CLIENTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS clients (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT DEFAULT 'company' CHECK(type IN ('company', 'individual')),
    email TEXT,
    phone TEXT,
    website TEXT,
    industry TEXT,
    company_size TEXT,
    address TEXT,
    city TEXT,
    state TEXT,
    zip_code TEXT,
    country TEXT,
    status TEXT DEFAULT 'lead' CHECK(status IN ('lead', 'prospect', 'active', 'inactive', 'churned')),
    source TEXT,
    assigned_to TEXT,
    lifetime_value REAL,
    tags TEXT DEFAULT '[]', -- JSON
    custom_fields TEXT DEFAULT '{}', -- JSON
    notes TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    last_contacted_at TEXT,
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_clients_user_id ON clients(user_id);
CREATE INDEX IF NOT EXISTS idx_clients_sync_status ON clients(sync_status);

-- ============================================
-- CLIENT CONTACTS
-- ============================================
CREATE TABLE IF NOT EXISTS client_contacts (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    role TEXT,
    is_primary INTEGER DEFAULT 0,
    notes TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- CLIENT INTERACTIONS
-- ============================================
CREATE TABLE IF NOT EXISTS client_interactions (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    contact_id TEXT REFERENCES client_contacts(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK(type IN ('call', 'email', 'meeting', 'note')),
    subject TEXT NOT NULL,
    description TEXT,
    outcome TEXT,
    occurred_at TEXT DEFAULT (datetime('now')),
    created_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- CLIENT DEALS
-- ============================================
CREATE TABLE IF NOT EXISTS client_deals (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value REAL DEFAULT 0,
    stage TEXT DEFAULT 'qualification' CHECK(stage IN ('qualification', 'proposal', 'negotiation', 'closed_won', 'closed_lost')),
    probability INTEGER DEFAULT 0,
    expected_close_date TEXT,
    notes TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    closed_at TEXT,
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT
);

-- ============================================
-- CALENDAR EVENTS TABLE (Local cache + manual events)
-- ============================================
CREATE TABLE IF NOT EXISTS calendar_events (
    id TEXT PRIMARY KEY,
    server_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    google_event_id TEXT,
    calendar_id TEXT DEFAULT 'primary',
    title TEXT,
    description TEXT,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    all_day INTEGER DEFAULT 0,
    location TEXT,
    attendees TEXT DEFAULT '[]', -- JSON
    status TEXT DEFAULT 'confirmed',
    visibility TEXT DEFAULT 'default',
    html_link TEXT,
    source TEXT DEFAULT 'businessos' CHECK(source IN ('google', 'businessos', 'manual')),
    meeting_type TEXT DEFAULT 'other' CHECK(meeting_type IN (
        'team', 'sales', 'onboarding', 'kickoff', 'implementation',
        'standup', 'retrospective', 'planning', 'review', 'one_on_one',
        'client', 'internal', 'external', 'other'
    )),
    context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    client_id TEXT REFERENCES clients(id) ON DELETE SET NULL,
    recording_url TEXT,
    meeting_link TEXT,
    external_links TEXT DEFAULT '[]', -- JSON
    meeting_notes TEXT,
    action_items TEXT DEFAULT '[]', -- JSON
    synced_at TEXT DEFAULT (datetime('now')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    -- Sync columns
    sync_status TEXT DEFAULT 'synced',
    sync_version INTEGER DEFAULT 1,
    last_synced_at TEXT,
    UNIQUE(user_id, google_event_id)
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_user_id ON calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_time ON calendar_events(user_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_sync_status ON calendar_events(sync_status);

-- ============================================
-- SYNC QUEUE TABLE
-- Tracks pending sync operations
-- ============================================
CREATE TABLE IF NOT EXISTS sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    operation TEXT NOT NULL CHECK(operation IN ('create', 'update', 'delete')),
    payload TEXT, -- JSON of the record data
    attempts INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    processed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_queue_pending ON sync_queue(processed_at) WHERE processed_at IS NULL;

-- ============================================
-- OFFLINE CHANGES LOG
-- For conflict resolution
-- ============================================
CREATE TABLE IF NOT EXISTS offline_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_offline_changes_record ON offline_changes(table_name, record_id);

-- Initialize sync metadata
INSERT OR IGNORE INTO sync_metadata (id, created_at) VALUES (1, datetime('now'));
