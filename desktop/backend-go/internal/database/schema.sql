-- BusinessOS Database Schema for sqlc
-- Note: Better Auth manages the "user" and "session" tables externally

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Enum types (matching actual database - some use UPPERCASE values)
DO $$ BEGIN
CREATE TYPE messagerole AS ENUM ('USER', 'ASSISTANT', 'SYSTEM', 'user', 'assistant', 'system');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE artifacttype AS ENUM ('CODE', 'DOCUMENT', 'MARKDOWN', 'REACT', 'HTML', 'SVG');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE contexttype AS ENUM ('PERSON', 'BUSINESS', 'PROJECT', 'CUSTOM', 'document', 'DOCUMENT');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE projectstatus AS ENUM ('ACTIVE', 'PAUSED', 'COMPLETED', 'ARCHIVED');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE projectpriority AS ENUM ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE nodetype AS ENUM ('BUSINESS', 'PROJECT', 'LEARNING', 'OPERATIONAL');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE nodehealth AS ENUM ('HEALTHY', 'NEEDS_ATTENTION', 'CRITICAL', 'NOT_STARTED');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE taskstatus AS ENUM ('todo', 'in_progress', 'done', 'cancelled');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE taskpriority AS ENUM ('critical', 'high', 'medium', 'low');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE memberstatus AS ENUM ('AVAILABLE', 'BUSY', 'OVERLOADED', 'OOO');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE clienttype AS ENUM ('company', 'individual');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE clientstatus AS ENUM ('lead', 'prospect', 'active', 'inactive', 'churned');
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
CREATE TYPE interactiontype AS ENUM ('call', 'email', 'meeting', 'note');
EXCEPTION WHEN duplicate_object THEN null; END $$;
-- Dropped in migration 099 (client_deals → deals unification)
-- CREATE TYPE dealstage AS ENUM ('qualification', 'proposal', 'negotiation', 'closed_won', 'closed_lost');

-- Better Auth managed user table (defined here for SQLC JOINs)
CREATE TABLE IF NOT EXISTS "user" (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    "emailVerified" BOOLEAN DEFAULT FALSE,
    image TEXT,
    "createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    platform_role VARCHAR(20) DEFAULT 'user'
        CHECK (platform_role IN ('superadmin', 'admin', 'user'))
);

-- Contexts table (for documents, profiles)
CREATE TABLE IF NOT EXISTS contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type contexttype DEFAULT 'CUSTOM',
    content TEXT,
    structured_data JSONB,
    system_prompt_template TEXT,
    blocks JSONB DEFAULT '[]',
    cover_image VARCHAR(500),
    icon VARCHAR(50),
    parent_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    is_template BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    last_edited_at TIMESTAMP,
    word_count INTEGER DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    share_id VARCHAR(32) UNIQUE,
    property_schema JSONB DEFAULT '[]',
    properties JSONB DEFAULT '{}',
    client_id UUID,
    workspace_id UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contexts_user_id ON contexts(user_id);
CREATE INDEX IF NOT EXISTS idx_contexts_parent_id ON contexts(parent_id);
CREATE INDEX IF NOT EXISTS idx_contexts_is_archived ON contexts(is_archived);
CREATE INDEX IF NOT EXISTS idx_contexts_share_id ON contexts(share_id);

-- Conversations table
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) DEFAULT 'New Conversation',
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role messagerole NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    message_metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);

-- Conversation tags
CREATE TABLE IF NOT EXISTS conversation_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tag VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Clients table
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type clienttype DEFAULT 'company',
    email VARCHAR(255),
    phone VARCHAR(50),
    website VARCHAR(255),
    industry VARCHAR(100),
    company_size VARCHAR(50),
    address VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    zip_code VARCHAR(20),
    country VARCHAR(100),
    status clientstatus DEFAULT 'lead',
    source VARCHAR(100),
    assigned_to VARCHAR(255),
    lifetime_value NUMERIC(12, 2),
    tags JSONB DEFAULT '[]',
    custom_fields JSONB DEFAULT '{}',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_contacted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_clients_user_id ON clients(user_id);
CREATE INDEX IF NOT EXISTS ix_clients_user_status ON clients(user_id, status);
CREATE INDEX IF NOT EXISTS ix_clients_user_type ON clients(user_id, type);

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status projectstatus DEFAULT 'ACTIVE',
    priority projectpriority DEFAULT 'MEDIUM',
    client_name VARCHAR(255),
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    project_type VARCHAR(100) DEFAULT 'internal',
    project_metadata JSONB,
    -- Date tracking
    start_date DATE,
    due_date DATE,
    completed_at TIMESTAMP WITH TIME ZONE,
    -- Visibility/sharing
    visibility VARCHAR(20) DEFAULT 'private',
    owner_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_client ON projects(client_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_due_date ON projects(due_date);

-- Project notes
CREATE TABLE IF NOT EXISTS project_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Project conversations (many-to-many)
CREATE TABLE IF NOT EXISTS project_conversations (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    linked_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (project_id, conversation_id)
);

-- Artifacts table
CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    type artifacttype NOT NULL,
    language VARCHAR(50),
    content TEXT NOT NULL,
    summary VARCHAR(500),
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_artifacts_user_id ON artifacts(user_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_conversation_id ON artifacts(conversation_id);

-- Artifact versions
CREATE TABLE IF NOT EXISTS artifact_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Nodes table
CREATE TABLE IF NOT EXISTS nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES nodes(id) ON DELETE SET NULL,
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    type nodetype NOT NULL,
    health nodehealth DEFAULT 'NOT_STARTED',
    purpose TEXT,
    current_status TEXT,
    this_week_focus JSONB,
    decision_queue JSONB,
    delegation_ready JSONB,
    is_active BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_user_id ON nodes(user_id);

-- Node metrics
CREATE TABLE IF NOT EXISTS node_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    metric_name VARCHAR(255) NOT NULL,
    metric_value VARCHAR(500) NOT NULL,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- Node to Project links (many-to-many)
CREATE TABLE IF NOT EXISTS node_projects (
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    linked_by VARCHAR(255),
    PRIMARY KEY (node_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_node_projects_node ON node_projects(node_id);
CREATE INDEX IF NOT EXISTS idx_node_projects_project ON node_projects(project_id);

-- Node to Context links (many-to-many)
CREATE TABLE IF NOT EXISTS node_contexts (
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    context_id UUID NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    linked_by VARCHAR(255),
    PRIMARY KEY (node_id, context_id)
);

CREATE INDEX IF NOT EXISTS idx_node_contexts_node ON node_contexts(node_id);
CREATE INDEX IF NOT EXISTS idx_node_contexts_context ON node_contexts(context_id);

-- Node to Conversation links (many-to-many)
CREATE TABLE IF NOT EXISTS node_conversations (
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    linked_by VARCHAR(255),
    PRIMARY KEY (node_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_node_conversations_node ON node_conversations(node_id);
CREATE INDEX IF NOT EXISTS idx_node_conversations_conversation ON node_conversations(conversation_id);

-- Team members table
CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    status memberstatus DEFAULT 'AVAILABLE',
    capacity INTEGER DEFAULT 0,
    manager_id UUID REFERENCES team_members(id) ON DELETE SET NULL,
    skills JSONB,
    hourly_rate NUMERIC(10, 2),
    share_calendar BOOLEAN DEFAULT FALSE,
    calendar_user_id VARCHAR(255),
    joined_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);

-- Team member activities
CREATE TABLE IF NOT EXISTS team_member_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    activity_type VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status taskstatus DEFAULT 'todo',
    priority taskpriority DEFAULT 'medium',
    due_date TIMESTAMP,
    start_date TIMESTAMP,
    completed_at TIMESTAMP,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    assignee_id UUID REFERENCES team_members(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    custom_status_id UUID,
    position INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(user_id, position);

-- Project custom statuses
CREATE TABLE IF NOT EXISTS project_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) DEFAULT '#6B7280',
    position INT DEFAULT 0,
    is_done_state BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_project_statuses_project ON project_statuses(project_id);

-- Add FK from tasks to project_statuses
DO $$ BEGIN
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_custom_status FOREIGN KEY (custom_status_id) REFERENCES project_statuses(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Task assignees (many-to-many)
CREATE TABLE IF NOT EXISTS task_assignees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'assignee',
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by VARCHAR(255),
    UNIQUE(task_id, team_member_id)
);

CREATE INDEX IF NOT EXISTS idx_task_assignees_task ON task_assignees(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_member ON task_assignees(team_member_id);

-- Task dependencies
DO $$ BEGIN
CREATE TYPE dependencytype AS ENUM ('finish_to_start', 'start_to_start', 'finish_to_finish', 'start_to_finish');
EXCEPTION WHEN duplicate_object THEN null; END $$;

CREATE TABLE IF NOT EXISTS task_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    predecessor_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    successor_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type dependencytype DEFAULT 'finish_to_start',
    lag_days INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(predecessor_id, successor_id)
);

CREATE INDEX IF NOT EXISTS idx_task_deps_predecessor ON task_dependencies(predecessor_id);
CREATE INDEX IF NOT EXISTS idx_task_deps_successor ON task_dependencies(successor_id);

-- Focus items table
CREATE TABLE IF NOT EXISTS focus_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    text VARCHAR(500) NOT NULL,
    completed BOOLEAN DEFAULT FALSE,
    focus_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_focus_items_user_id ON focus_items(user_id);

-- Daily logs table
CREATE TABLE IF NOT EXISTS daily_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    content TEXT NOT NULL,
    transcription_source VARCHAR(50),
    extracted_actions JSONB,
    extracted_patterns JSONB,
    energy_level INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_logs_user_id ON daily_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_logs_date ON daily_logs(date);

-- User settings table
CREATE TABLE IF NOT EXISTS user_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE NOT NULL,
    default_model VARCHAR(100),
    email_notifications BOOLEAN DEFAULT TRUE,
    daily_summary BOOLEAN DEFAULT FALSE,
    theme VARCHAR(20) DEFAULT 'light',
    sidebar_collapsed BOOLEAN DEFAULT FALSE,
    share_analytics BOOLEAN DEFAULT TRUE,
    custom_settings JSONB,
    -- Thinking/COT settings
    thinking_enabled BOOLEAN DEFAULT false,
    thinking_show_in_ui BOOLEAN DEFAULT true,
    thinking_save_traces BOOLEAN DEFAULT true,
    thinking_default_template_id UUID,
    thinking_max_tokens INT DEFAULT 4096,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- Add FK from contexts to clients
DO $$ BEGIN
ALTER TABLE contexts ADD CONSTRAINT fk_contexts_client_id FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN null; END $$;
CREATE INDEX IF NOT EXISTS idx_contexts_client_id ON contexts(client_id);

-- Client contacts
CREATE TABLE IF NOT EXISTS client_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    role VARCHAR(100),
    is_primary BOOLEAN DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_client_contacts_client ON client_contacts(client_id);

-- Client interactions
CREATE TABLE IF NOT EXISTS client_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES client_contacts(id) ON DELETE SET NULL,
    type interactiontype NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT,
    outcome VARCHAR(255),
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_client_interactions_client ON client_interactions(client_id);
CREATE INDEX IF NOT EXISTS ix_client_interactions_occurred ON client_interactions(occurred_at);

-- DROPPED in migration 099: client_deals
-- CREATE TABLE client_deals (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
--     name VARCHAR(255) NOT NULL,
--     value NUMERIC(12, 2) DEFAULT 0,
--     stage dealstage DEFAULT 'qualification',
--     probability INTEGER DEFAULT 0,
--     expected_close_date DATE,
--     notes TEXT,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     closed_at TIMESTAMP WITH TIME ZONE
-- );
--
-- CREATE INDEX ix_client_deals_client ON client_deals(client_id);
-- CREATE INDEX ix_client_deals_stage ON client_deals(stage);

-- Google OAuth tokens for calendar integration
CREATE TABLE IF NOT EXISTS google_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    expiry TIMESTAMP WITH TIME ZONE NOT NULL,
    scopes TEXT[],
    google_email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_google_oauth_user_id ON google_oauth_tokens(user_id);

-- Slack OAuth tokens for workspace integration
CREATE TABLE IF NOT EXISTS slack_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE NOT NULL,
    -- Workspace info
    workspace_id VARCHAR(255) NOT NULL,
    workspace_name VARCHAR(255),
    -- Tokens - Slack provides both bot and user tokens
    bot_token TEXT NOT NULL,
    user_token TEXT,
    -- Token metadata
    bot_user_id VARCHAR(255),
    authed_user_id VARCHAR(255),
    -- Scopes granted
    bot_scopes TEXT[],
    user_scopes TEXT[],
    -- Webhook URL (if configured)
    incoming_webhook_url TEXT,
    incoming_webhook_channel VARCHAR(255),
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slack_oauth_user_id ON slack_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_slack_oauth_workspace ON slack_oauth_tokens(workspace_id);

-- Notion OAuth tokens for workspace integration
CREATE TABLE IF NOT EXISTS notion_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE NOT NULL,
    -- Workspace info
    workspace_id VARCHAR(255) NOT NULL,
    workspace_name VARCHAR(255),
    workspace_icon TEXT,
    -- Token - Notion provides a single access token (no refresh needed)
    access_token TEXT NOT NULL,
    bot_id VARCHAR(255),
    -- Owner info
    owner_type VARCHAR(50), -- 'user' or 'workspace'
    owner_user_id VARCHAR(255),
    owner_user_name VARCHAR(255),
    owner_user_email VARCHAR(255),
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notion_oauth_user_id ON notion_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_oauth_workspace ON notion_oauth_tokens(workspace_id);

-- Meeting types enum
DO $$ BEGIN
CREATE TYPE meetingtype AS ENUM (
    'team', 'sales', 'onboarding', 'kickoff', 'implementation',
    'standup', 'retrospective', 'planning', 'review', 'one_on_one',
    'client', 'internal', 'external', 'other'
);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Calendar events cache with meeting management
CREATE TABLE IF NOT EXISTS calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    google_event_id VARCHAR(255),
    calendar_id VARCHAR(255) DEFAULT 'primary',
    title VARCHAR(500),
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,
    location VARCHAR(500),
    attendees JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'confirmed',
    visibility VARCHAR(50) DEFAULT 'default',
    html_link TEXT,
    source VARCHAR(50) DEFAULT 'google',
    workspace_id UUID,

    -- Meeting management fields
    meeting_type meetingtype DEFAULT 'other',
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,

    -- Recording and external links
    recording_url TEXT,
    meeting_link TEXT,
    external_links JSONB DEFAULT '[]',

    -- Meeting notes and follow-ups
    meeting_notes TEXT,
    action_items JSONB DEFAULT '[]',

    -- Metadata
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, google_event_id)
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_user_id ON calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_time ON calendar_events(user_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_source ON calendar_events(source);
CREATE INDEX IF NOT EXISTS idx_calendar_events_type ON calendar_events(meeting_type);
CREATE INDEX IF NOT EXISTS idx_calendar_events_context ON calendar_events(context_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_project ON calendar_events(project_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_client ON calendar_events(client_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_workspace ON calendar_events(workspace_id, start_time, end_time);

CREATE TABLE IF NOT EXISTS calendar_event_colors (
    event_id UUID PRIMARY KEY REFERENCES calendar_events(id) ON DELETE CASCADE,
    color_id VARCHAR(50),
    color_hex VARCHAR(7)
        CHECK (color_hex IS NULL OR color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===== USAGE ANALYTICS TABLES =====

-- AI usage tracking (per request)
CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,

    -- Provider and model info
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,

    -- Token usage
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    thinking_tokens INTEGER DEFAULT 0,  -- COT reasoning tokens (tracked separately)

    -- Agent tracking
    agent_name VARCHAR(100),
    delegated_to VARCHAR(100),
    parent_request_id UUID REFERENCES ai_usage_logs(id) ON DELETE SET NULL,

    -- Request context
    request_type VARCHAR(50),  -- 'chat', 'completion', 'extract', 'analyze'
    context_ids UUID[],
    node_id UUID REFERENCES nodes(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,

    -- Timing
    duration_ms INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Cost tracking (optional)
    estimated_cost NUMERIC(10, 6),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_usage_user_id ON ai_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_conversation ON ai_usage_logs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_provider ON ai_usage_logs(provider);
CREATE INDEX IF NOT EXISTS idx_ai_usage_model ON ai_usage_logs(model);
CREATE INDEX IF NOT EXISTS idx_ai_usage_agent ON ai_usage_logs(agent_name);
CREATE INDEX IF NOT EXISTS idx_ai_usage_date ON ai_usage_logs(started_at);

-- MCP tool usage tracking
CREATE TABLE IF NOT EXISTS mcp_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Tool info
    tool_name VARCHAR(255) NOT NULL,
    server_name VARCHAR(255),

    -- Request details
    input_params JSONB,
    output_result JSONB,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,

    -- Timing
    duration_ms INTEGER DEFAULT 0,

    -- Context
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    ai_request_id UUID REFERENCES ai_usage_logs(id) ON DELETE SET NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_usage_user_id ON mcp_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_usage_tool ON mcp_usage_logs(tool_name);
CREATE INDEX IF NOT EXISTS idx_mcp_usage_date ON mcp_usage_logs(created_at);

-- Daily usage summary (aggregated)
CREATE TABLE IF NOT EXISTS usage_daily_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    date DATE NOT NULL,

    -- AI usage totals
    ai_requests INTEGER DEFAULT 0,
    ai_input_tokens INTEGER DEFAULT 0,
    ai_output_tokens INTEGER DEFAULT 0,
    ai_total_tokens INTEGER DEFAULT 0,
    ai_thinking_tokens BIGINT DEFAULT 0,  -- COT reasoning tokens
    ai_estimated_cost NUMERIC(10, 4) DEFAULT 0,

    -- Breakdown by provider
    provider_breakdown JSONB DEFAULT '{}',

    -- Breakdown by model
    model_breakdown JSONB DEFAULT '{}',

    -- Breakdown by agent
    agent_breakdown JSONB DEFAULT '{}',

    -- MCP usage totals
    mcp_requests INTEGER DEFAULT 0,
    mcp_tool_breakdown JSONB DEFAULT '{}',

    -- System usage
    conversations_created INTEGER DEFAULT 0,
    messages_sent INTEGER DEFAULT 0,
    artifacts_created INTEGER DEFAULT 0,
    documents_created INTEGER DEFAULT 0,

    -- Context usage
    contexts_accessed UUID[],
    nodes_accessed UUID[],
    projects_accessed UUID[],

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_usage_summary_user_id ON usage_daily_summary(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_summary_date ON usage_daily_summary(date);

-- System event logs (general activity tracking)
CREATE TABLE IF NOT EXISTS system_event_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Event details
    event_type VARCHAR(100) NOT NULL,  -- 'page_view', 'action', 'api_call'
    event_name VARCHAR(255) NOT NULL,
    event_data JSONB,

    -- Context
    module VARCHAR(100),  -- 'chat', 'calendar', 'clients', 'nodes', etc.
    resource_type VARCHAR(100),
    resource_id UUID,

    -- Session info
    session_id VARCHAR(255),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_events_user_id ON system_event_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_system_events_type ON system_event_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_system_events_module ON system_event_logs(module);
CREATE INDEX IF NOT EXISTS idx_system_events_date ON system_event_logs(created_at);

-- ===== CUSTOM SLASH COMMANDS =====

-- User custom commands for AI chat
CREATE TABLE IF NOT EXISTS user_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,           -- e.g., "weekly-report" (the slash command name)
    display_name VARCHAR(100) NOT NULL,  -- e.g., "Weekly Report" (shown in UI)
    description TEXT,                    -- Description of what the command does
    icon VARCHAR(10),                    -- emoji icon
    system_prompt TEXT NOT NULL,         -- Custom prompt template
    context_sources TEXT[] DEFAULT '{}', -- What context to load: documents, projects, clients, tasks, artifacts
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_user_commands_user_id ON user_commands(user_id);
CREATE INDEX IF NOT EXISTS idx_user_commands_name ON user_commands(user_id, name);

-- ===== CUSTOM AGENTS =====

-- User-defined custom agents with custom system prompts and configurations
CREATE TABLE IF NOT EXISTS custom_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Agent Identity
    name VARCHAR(50) NOT NULL,              -- e.g., "code-reviewer" (internal name, lowercase)
    display_name VARCHAR(100) NOT NULL,     -- e.g., "Code Reviewer" (shown in UI)
    description TEXT,                       -- What the agent does
    avatar VARCHAR(50),                     -- emoji or icon identifier

    -- Agent Configuration
    system_prompt TEXT NOT NULL,            -- Base system prompt for the agent
    model_preference VARCHAR(100),          -- Preferred model (e.g., "claude-3-opus")
    temperature DECIMAL(3,2) DEFAULT 0.7,   -- Default temperature
    max_tokens INTEGER DEFAULT 4096,        -- Default max tokens

    -- Capabilities
    capabilities TEXT[] DEFAULT '{}',       -- e.g., ["code_review", "analysis", "writing"]
    tools_enabled TEXT[] DEFAULT '{}',      -- Which tools the agent can use
    context_sources TEXT[] DEFAULT '{}',    -- What context to auto-load: documents, projects, etc.

    -- Behavior Settings
    thinking_enabled BOOLEAN DEFAULT FALSE,  -- Enable COT for this agent
    streaming_enabled BOOLEAN DEFAULT TRUE,  -- Enable streaming responses
    apply_personalization BOOLEAN DEFAULT FALSE,  -- Use prompt personalizations from learning system
    welcome_message TEXT,                   -- Welcome message shown when starting conversation
    suggested_prompts TEXT[] DEFAULT '{}',  -- Array of suggested prompts for users

    -- Agent Type/Category
    category VARCHAR(50) DEFAULT 'general', -- general, coding, writing, analysis, business, custom
    is_public BOOLEAN DEFAULT FALSE,        -- Whether to share with team (future)
    is_featured BOOLEAN DEFAULT FALSE,      -- Show prominently in featured list

    -- Usage & Status
    is_active BOOLEAN DEFAULT TRUE,
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_agents_user_id ON custom_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_agents_name ON custom_agents(user_id, name);
CREATE INDEX IF NOT EXISTS idx_custom_agents_category ON custom_agents(category);

-- Agent presets (built-in templates users can copy)
CREATE TABLE IF NOT EXISTS agent_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar VARCHAR(50),
    system_prompt TEXT NOT NULL,
    model_preference VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4096,
    capabilities TEXT[] DEFAULT '{}',
    tools_enabled TEXT[] DEFAULT '{}',
    context_sources TEXT[] DEFAULT '{}',
    thinking_enabled BOOLEAN DEFAULT FALSE,
    category VARCHAR(50) DEFAULT 'general',
    times_copied INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ===== VOICE NOTES =====

-- Voice transcription history with stats
CREATE TABLE IF NOT EXISTS voice_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    transcript TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    word_count INTEGER NOT NULL,
    words_per_minute NUMERIC(10, 2),
    language VARCHAR(10) DEFAULT 'en',
    audio_file_path VARCHAR(500),
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voice_notes_user_id ON voice_notes(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_notes_date ON voice_notes(created_at);
CREATE INDEX IF NOT EXISTS idx_voice_notes_context ON voice_notes(context_id);
CREATE INDEX IF NOT EXISTS idx_voice_notes_project ON voice_notes(project_id);

-- ===== PROJECT MANAGEMENT ENHANCEMENTS =====

-- Project role type for team assignment
DO $$ BEGIN
CREATE TYPE projectrole AS ENUM ('owner', 'admin', 'member', 'viewer');
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Project members (team assignment)
CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    team_member_id UUID REFERENCES team_members(id) ON DELETE CASCADE,
    role projectrole DEFAULT 'member',
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by VARCHAR(255),
    UNIQUE(project_id, user_id),
    UNIQUE(project_id, team_member_id)
);

CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_team_member ON project_members(team_member_id);

-- Project tags (user-defined labels)
CREATE TABLE IF NOT EXISTS project_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(7) DEFAULT '#6366f1',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_project_tags_user ON project_tags(user_id);

-- Project tag assignments (many-to-many)
CREATE TABLE IF NOT EXISTS project_tag_assignments (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES project_tags(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (project_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_tag_assignments_project ON project_tag_assignments(project_id);
CREATE INDEX IF NOT EXISTS idx_tag_assignments_tag ON project_tag_assignments(tag_id);

-- Project documents (linking projects to contexts/documents)
CREATE TABLE IF NOT EXISTS project_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    linked_by VARCHAR(255),
    UNIQUE(project_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_project_docs_project ON project_documents(project_id);
CREATE INDEX IF NOT EXISTS idx_project_docs_document ON project_documents(document_id);

-- Project templates
CREATE TABLE IF NOT EXISTS project_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    default_status projectstatus DEFAULT 'ACTIVE',
    default_priority projectpriority DEFAULT 'MEDIUM',
    template_data JSONB DEFAULT '{}',
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_templates_user ON project_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_templates_public ON project_templates(is_public) WHERE is_public = TRUE;

-- ===== CHAIN OF THOUGHT (COT) THINKING SYSTEM =====

-- Thinking type enum
DO $$ BEGIN
CREATE TYPE thinkingtype AS ENUM ('analysis', 'planning', 'reflection', 'tool_use', 'reasoning', 'evaluation');
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Thinking/reasoning tracking
CREATE TABLE IF NOT EXISTS thinking_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,

    -- Thinking content
    thinking_content TEXT NOT NULL,
    thinking_type thinkingtype DEFAULT 'reasoning',
    step_number INT DEFAULT 1,

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INT,

    -- Token tracking
    thinking_tokens INT DEFAULT 0,

    -- Metadata
    model_used VARCHAR(100),
    reasoning_template_id UUID,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_thinking_traces_user ON thinking_traces(user_id);
CREATE INDEX IF NOT EXISTS idx_thinking_traces_conversation ON thinking_traces(conversation_id);
CREATE INDEX IF NOT EXISTS idx_thinking_traces_message ON thinking_traces(message_id);
CREATE INDEX IF NOT EXISTS idx_thinking_traces_template ON thinking_traces(reasoning_template_id);

-- Custom reasoning templates/systems
CREATE TABLE IF NOT EXISTS reasoning_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Template configuration
    system_prompt TEXT,
    thinking_instruction TEXT,
    output_format VARCHAR(50) DEFAULT 'streaming',

    -- Options
    show_thinking BOOLEAN DEFAULT true,
    save_thinking BOOLEAN DEFAULT true,
    max_thinking_tokens INT DEFAULT 4096,

    -- Usage tracking
    times_used INT DEFAULT 0,

    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reasoning_templates_user ON reasoning_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_reasoning_templates_default ON reasoning_templates(user_id, is_default) WHERE is_default = true;

-- ===== INTEGRATIONS MODULE (Migration 025) =====

-- Integration Providers (system-defined catalog)
CREATE TABLE IF NOT EXISTS integration_providers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    icon_url TEXT,
    oauth_config JSONB NOT NULL DEFAULT '{}',
    modules TEXT[] NOT NULL DEFAULT '{}',
    skills TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'available',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Baseline providers required by first-party OAuth handlers. The migration
-- ledger can be baselined on an existing local database, so these must live in
-- the canonical schema as well as historical migrations.
INSERT INTO integration_providers (id, name, description, category, modules, skills, status)
VALUES (
    'google_calendar',
    'Google Calendar',
    'Google calendar integration',
    'calendar',
    ARRAY['calendar', 'daily_log', 'projects'],
    ARRAY['google_calendar.sync_daily_log', 'google_calendar.create_event', 'google_calendar.get_events'],
    'available'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    modules = EXCLUDED.modules,
    skills = EXCLUDED.skills,
    status = EXCLUDED.status,
    updated_at = NOW();

-- User Integration Connections
CREATE TABLE IF NOT EXISTS user_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL REFERENCES integration_providers(id),
    status VARCHAR(20) DEFAULT 'connected',
    connected_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    access_token_encrypted BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    scopes TEXT[],
    external_account_id VARCHAR(255),
    external_account_name VARCHAR(255),
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{"enabledSkills": [], "notifications": true}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_user_integrations_user ON user_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_provider ON user_integrations(provider_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_status ON user_integrations(status);

-- Module Integration Settings
CREATE TABLE IF NOT EXISTS module_integration_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    module_id VARCHAR(50) NOT NULL,
    provider_id VARCHAR(50) NOT NULL REFERENCES integration_providers(id),
    enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'bidirectional',
    sync_frequency VARCHAR(20) DEFAULT 'realtime',
    custom_settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, module_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_module_integration_settings_user ON module_integration_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_module_integration_settings_module ON module_integration_settings(module_id);

-- User Model Preferences
CREATE TABLE IF NOT EXISTS user_model_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE UNIQUE,
    tier_2_model JSONB DEFAULT '{"model_id": "claude-3-5-haiku", "provider": "anthropic"}',
    tier_3_model JSONB DEFAULT '{"model_id": "claude-sonnet-4", "provider": "anthropic"}',
    tier_4_model JSONB DEFAULT '{"model_id": "claude-opus-4", "provider": "anthropic"}',
    tier_2_fallbacks JSONB DEFAULT '[]',
    tier_3_fallbacks JSONB DEFAULT '[]',
    tier_4_fallbacks JSONB DEFAULT '[]',
    skill_overrides JSONB DEFAULT '{}',
    allow_model_upgrade_on_failure BOOLEAN DEFAULT true,
    max_latency_ms INTEGER DEFAULT 30000,
    prefer_local BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pending Decisions (human-in-the-loop)
CREATE TABLE IF NOT EXISTS pending_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id VARCHAR(255) NOT NULL,
    skill_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    description TEXT,
    options TEXT[],
    input_fields JSONB,
    context JSONB,
    priority VARCHAR(20) DEFAULT 'medium',
    status VARCHAR(20) DEFAULT 'pending',
    decision TEXT,
    decision_inputs JSONB,
    decided_by VARCHAR(255) REFERENCES "user"(id),
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_decisions_user_status ON pending_decisions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_pending_decisions_execution ON pending_decisions(execution_id);
CREATE INDEX IF NOT EXISTS idx_pending_decisions_expires ON pending_decisions(expires_at) WHERE status = 'pending';

-- Integration Sync Log
CREATE TABLE IF NOT EXISTS integration_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id UUID NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    module_id VARCHAR(50),
    sync_type VARCHAR(50) NOT NULL,
    direction VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    records_processed INT DEFAULT 0,
    records_created INT DEFAULT 0,
    records_updated INT DEFAULT 0,
    records_failed INT DEFAULT 0,
    error_message TEXT,
    error_details JSONB,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_integration_sync_log_integration ON integration_sync_log(user_integration_id);
CREATE INDEX IF NOT EXISTS idx_integration_sync_log_started ON integration_sync_log(started_at DESC);

-- Skill Executions
CREATE TABLE IF NOT EXISTS skill_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    current_step INT DEFAULT 0,
    params JSONB DEFAULT '{}',
    result JSONB,
    error TEXT,
    context JSONB DEFAULT '{}',
    step_results JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_skill_executions_user ON skill_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_skill_executions_status ON skill_executions(status);

-- ===== CREDENTIAL VAULT (Migration 027) =====

-- Unified credential storage with encryption
CREATE TABLE IF NOT EXISTS credential_vault (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL,
    credential_type VARCHAR(20) NOT NULL DEFAULT 'oauth',
    encrypted_data BYTEA NOT NULL,
    encryption_version INT DEFAULT 1,
    expires_at TIMESTAMPTZ,
    external_account_id VARCHAR(255),
    external_account_email VARCHAR(255),
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),
    scopes TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    last_rotated_at TIMESTAMPTZ,
    UNIQUE(user_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_credential_vault_user ON credential_vault(user_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_provider ON credential_vault(provider_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_type ON credential_vault(credential_type);
CREATE INDEX IF NOT EXISTS idx_credential_vault_expires ON credential_vault(expires_at) WHERE expires_at IS NOT NULL;

-- Webhook registrations
CREATE TABLE IF NOT EXISTS integration_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL,
    webhook_url TEXT NOT NULL,
    webhook_secret_encrypted BYTEA,
    events TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ,
    failure_count INT DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, provider_id, webhook_url)
);

CREATE INDEX IF NOT EXISTS idx_webhooks_user ON integration_webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_provider ON integration_webhooks(provider_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON integration_webhooks(status);

-- Data sync mappings
CREATE TABLE IF NOT EXISTS data_sync_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    source_provider VARCHAR(50) NOT NULL,
    source_entity VARCHAR(100) NOT NULL,
    target_module VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100),
    field_mappings JSONB NOT NULL DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'import',
    sync_frequency VARCHAR(20) DEFAULT 'manual',
    last_synced_at TIMESTAMPTZ,
    records_synced INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, source_provider, source_entity, target_module)
);

CREATE INDEX IF NOT EXISTS idx_sync_mappings_user ON data_sync_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_mappings_provider ON data_sync_mappings(source_provider);
CREATE INDEX IF NOT EXISTS idx_sync_mappings_enabled ON data_sync_mappings(enabled) WHERE enabled = true;

-- ===== DATA IMPORTS (Migration 028) =====

-- Import status enum
DO $$ BEGIN
CREATE TYPE import_status AS ENUM (
    'pending', 'validating', 'mapping', 'processing', 'completed', 'failed', 'cancelled'
);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Import source type enum
DO $$ BEGIN
CREATE TYPE import_source_type AS ENUM (
    'chatgpt_export', 'claude_export', 'custom_chat_export',
    'hubspot_contacts', 'hubspot_deals', 'hubspot_companies',
    'salesforce_contacts', 'salesforce_accounts',
    'linear_issues', 'notion_database', 'asana_tasks', 'jira_issues',
    'google_calendar', 'outlook_calendar',
    'fathom_analytics', 'plausible_analytics',
    'google_drive', 'dropbox', 'notion_pages',
    'csv_generic', 'json_generic', 'custom'
);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- Import jobs table
CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),
    original_filename VARCHAR(500),
    file_size_bytes BIGINT,
    content_type VARCHAR(100),
    status import_status DEFAULT 'pending',
    progress_percent INT DEFAULT 0,
    total_records INT DEFAULT 0,
    processed_records INT DEFAULT 0,
    imported_records INT DEFAULT 0,
    skipped_records INT DEFAULT 0,
    failed_records INT DEFAULT 0,
    field_mapping JSONB DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    import_options JSONB DEFAULT '{}',
    target_module VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100),
    result_summary JSONB DEFAULT '{}',
    error_log JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    error_details JSONB
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_source ON import_jobs(source_type);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created ON import_jobs(created_at DESC);

-- Imported records tracking (deduplication)
CREATE TABLE IF NOT EXISTS imported_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,
    source_type import_source_type NOT NULL,
    source_provider VARCHAR(50),
    external_id VARCHAR(500) NOT NULL,
    target_module VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100),
    target_record_id UUID NOT NULL,
    external_data_hash VARCHAR(64),
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, source_type, external_id)
);

CREATE INDEX IF NOT EXISTS idx_imported_records_user ON imported_records(user_id);
CREATE INDEX IF NOT EXISTS idx_imported_records_source ON imported_records(source_type, external_id);
CREATE INDEX IF NOT EXISTS idx_imported_records_target ON imported_records(target_module, target_record_id);

-- Import mapping templates
CREATE TABLE IF NOT EXISTS import_mapping_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type import_source_type NOT NULL,
    target_module VARCHAR(50) NOT NULL,
    template_name VARCHAR(100) NOT NULL,
    field_mappings JSONB NOT NULL DEFAULT '{}',
    transform_rules JSONB DEFAULT '{}',
    default_values JSONB DEFAULT '{}',
    description TEXT,
    is_system_template BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(source_type, target_module, template_name)
);

-- Imported conversations (ChatGPT, Claude, etc.)
CREATE TABLE IF NOT EXISTS imported_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,
    source_type import_source_type NOT NULL,
    external_conversation_id VARCHAR(255),
    title VARCHAR(500),
    model VARCHAR(100),
    messages JSONB NOT NULL DEFAULT '[]',
    message_count INT DEFAULT 0,
    original_created_at TIMESTAMPTZ,
    original_updated_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    linked_context_id UUID,
    linked_project_id UUID,
    tags TEXT[] DEFAULT '{}',
    search_content TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imported_conversations_user ON imported_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_source ON imported_conversations(source_type);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_job ON imported_conversations(import_job_id);
CREATE INDEX IF NOT EXISTS idx_imported_conversations_search ON imported_conversations USING GIN(to_tsvector('english', search_content));

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
    updated_at TIMESTAMPTZ DEFAULT NOW(),
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
    updated_at TIMESTAMPTZ DEFAULT NOW(),
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
    updated_at TIMESTAMPTZ DEFAULT NOW(),
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
    name VARCHAR(500) NOT NULL,
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),
    size_bytes BIGINT,
    parent_folder_id VARCHAR(255),
    parent_folder_name VARCHAR(500),
    path TEXT,
    shared BOOLEAN DEFAULT FALSE,
    sharing_user VARCHAR(255),
    permissions JSONB DEFAULT '[]',
    web_view_link TEXT,
    web_content_link TEXT,
    thumbnail_link TEXT,
    icon_link TEXT,
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
    viewed_by_me_time TIMESTAMPTZ,
    owners JSONB DEFAULT '[]',
    last_modifying_user JSONB,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_drive_files_user ON google_drive_files(user_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_parent ON google_drive_files(user_id, parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_mime ON google_drive_files(user_id, mime_type);
CREATE INDEX IF NOT EXISTS idx_drive_files_modified ON google_drive_files(user_id, modified_time DESC);

-- Google Docs content
CREATE TABLE IF NOT EXISTS google_docs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    document_id VARCHAR(255) NOT NULL,
    drive_file_id UUID REFERENCES google_drive_files(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    body_text TEXT,
    word_count INT DEFAULT 0,
    headers JSONB DEFAULT '[]',
    locale VARCHAR(20),
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
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
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),
    time_zone VARCHAR(100),
    sheet_count INT DEFAULT 0,
    sheets JSONB DEFAULT '[]',
    named_ranges JSONB DEFAULT '[]',
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
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
    title VARCHAR(500) NOT NULL,
    locale VARCHAR(20),
    slide_count INT DEFAULT 0,
    slides JSONB DEFAULT '[]',
    page_width DECIMAL(10,2),
    page_height DECIMAL(10,2),
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
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

CREATE TABLE IF NOT EXISTS google_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    given_name VARCHAR(255),
    family_name VARCHAR(255),
    middle_name VARCHAR(255),
    emails JSONB DEFAULT '[]',
    phone_numbers JSONB DEFAULT '[]',
    addresses JSONB DEFAULT '[]',
    organization VARCHAR(255),
    job_title VARCHAR(255),
    department VARCHAR(255),
    photo_url TEXT,
    contact_groups JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_time TIMESTAMPTZ,
    modified_time TIMESTAMPTZ,
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

CREATE TABLE IF NOT EXISTS google_task_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    kind VARCHAR(100),
    updated TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, task_list_id)
);

CREATE INDEX IF NOT EXISTS idx_task_lists_user ON google_task_lists(user_id);

CREATE TABLE IF NOT EXISTS google_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    task_list_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    notes TEXT,
    status VARCHAR(50) DEFAULT 'needsAction',
    due TIMESTAMPTZ,
    completed TIMESTAMPTZ,
    deleted BOOLEAN DEFAULT FALSE,
    hidden BOOLEAN DEFAULT FALSE,
    parent_task_id VARCHAR(255),
    position VARCHAR(100),
    links JSONB DEFAULT '[]',
    updated TIMESTAMPTZ,
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
-- HUBSPOT CRM TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS hubspot_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    phone VARCHAR(100),
    company VARCHAR(255),
    job_title VARCHAR(255),
    lifecycle_stage VARCHAR(100),
    lead_status VARCHAR(100),
    owner_id VARCHAR(100),
    properties JSONB DEFAULT '{}',
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_user ON hubspot_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_contacts_email ON hubspot_contacts(user_id, email);

CREATE TABLE IF NOT EXISTS hubspot_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,
    name VARCHAR(500),
    domain VARCHAR(255),
    industry VARCHAR(255),
    number_of_employees INT,
    annual_revenue DECIMAL(15,2),
    city VARCHAR(255),
    state VARCHAR(255),
    country VARCHAR(255),
    owner_id VARCHAR(100),
    properties JSONB DEFAULT '{}',
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_companies_user ON hubspot_companies(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_companies_name ON hubspot_companies(user_id, name);

CREATE TABLE IF NOT EXISTS hubspot_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    hubspot_id VARCHAR(100) NOT NULL,
    deal_name VARCHAR(500),
    amount DECIMAL(15,2),
    pipeline VARCHAR(255),
    deal_stage VARCHAR(255),
    close_date DATE,
    owner_id VARCHAR(100),
    associated_company_ids JSONB DEFAULT '[]',
    associated_contact_ids JSONB DEFAULT '[]',
    properties JSONB DEFAULT '{}',
    created_at_hubspot TIMESTAMPTZ,
    updated_at_hubspot TIMESTAMPTZ,
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, hubspot_id)
);

CREATE INDEX IF NOT EXISTS idx_hubspot_deals_user ON hubspot_deals(user_id);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_stage ON hubspot_deals(user_id, deal_stage);
CREATE INDEX IF NOT EXISTS idx_hubspot_deals_close ON hubspot_deals(user_id, close_date);

-- ============================================================================
-- CLICKUP TABLES
-- ============================================================================

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

CREATE TABLE IF NOT EXISTS clickup_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(100) NOT NULL,
    custom_id VARCHAR(100),
    list_id VARCHAR(100) NOT NULL,
    folder_id VARCHAR(100),
    space_id VARCHAR(100) NOT NULL,
    name VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(100),
    status_color VARCHAR(50),
    priority VARCHAR(50),
    priority_color VARCHAR(50),
    due_date TIMESTAMPTZ,
    start_date TIMESTAMPTZ,
    date_created TIMESTAMPTZ,
    date_updated TIMESTAMPTZ,
    date_closed TIMESTAMPTZ,
    time_spent BIGINT DEFAULT 0,
    time_estimate BIGINT,
    parent_task_id VARCHAR(100),
    assignees JSONB DEFAULT '[]',
    creator JSONB,
    tags JSONB DEFAULT '[]',
    url TEXT,
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

CREATE TABLE IF NOT EXISTS airtable_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    base_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permission_level VARCHAR(50),
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, base_id)
);

CREATE INDEX IF NOT EXISTS idx_airtable_bases_user ON airtable_bases(user_id);

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

CREATE OR REPLACE FUNCTION update_integration_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

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

-- ============================================================================
-- EMAILS (Gmail integration)
-- ============================================================================

CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'gmail',
    external_id VARCHAR(255) NOT NULL,
    thread_id VARCHAR(255),

    -- Email metadata
    subject TEXT,
    snippet TEXT,
    from_email VARCHAR(255),
    from_name VARCHAR(255),
    to_emails JSONB DEFAULT '[]',
    cc_emails JSONB DEFAULT '[]',
    bcc_emails JSONB DEFAULT '[]',
    reply_to VARCHAR(255),

    -- Content
    body_text TEXT,
    body_html TEXT,
    attachments JSONB DEFAULT '[]',

    -- Status flags
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_important BOOLEAN DEFAULT FALSE,
    is_draft BOOLEAN DEFAULT FALSE,
    is_sent BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_trash BOOLEAN DEFAULT FALSE,
    labels JSONB DEFAULT '[]',

    -- Dates
    date TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_emails_user_thread ON emails(user_id, thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_user_date ON emails(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_user_unread ON emails(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_emails_user_starred ON emails(user_id, is_starred) WHERE is_starred = TRUE;
CREATE INDEX IF NOT EXISTS idx_emails_user_provider ON emails(user_id, provider);

-- ============================================================================
-- CHANNELS (Slack/Discord/Teams)
-- ============================================================================

CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),

    -- Channel metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    topic TEXT,
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_dm BOOLEAN DEFAULT FALSE,
    member_count INT DEFAULT 0,
    unread_count INT DEFAULT 0,

    -- Dates
    last_message_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_channels_user_provider ON channels(user_id, provider);

-- ============================================================================
-- CHANNEL MESSAGES
-- ============================================================================

CREATE TABLE IF NOT EXISTS channel_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL,

    -- Sender info
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    sender_avatar VARCHAR(500),

    -- Message content
    content TEXT,
    content_html TEXT,
    attachments JSONB DEFAULT '[]',
    reactions JSONB DEFAULT '[]',
    mentions JSONB DEFAULT '[]',

    -- Thread info
    thread_ts VARCHAR(50),
    parent_message_id UUID REFERENCES channel_messages(id),
    reply_count INT DEFAULT 0,
    is_thread_root BOOLEAN DEFAULT FALSE,

    -- Status
    is_edited BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,

    -- Dates
    sent_at TIMESTAMP WITH TIME ZONE,
    edited_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(channel_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_messages_channel ON channel_messages(channel_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_channel_messages_thread ON channel_messages(channel_id, thread_ts);

-- NOTE: integration_sync_log is defined above (line ~1117)

-- ============================================================================
-- NOTION DATABASES
-- ============================================================================

CREATE TABLE IF NOT EXISTS notion_databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    notion_id VARCHAR(255) NOT NULL,

    -- Database metadata
    title VARCHAR(500),
    description TEXT,
    icon VARCHAR(500),
    cover VARCHAR(500),
    url VARCHAR(500),

    -- Properties schema (stored as JSONB)
    properties JSONB DEFAULT '{}',

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, notion_id)
);

CREATE INDEX IF NOT EXISTS idx_notion_databases_user ON notion_databases(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_databases_title ON notion_databases(user_id, title);

-- ============================================================================
-- NOTION PAGES
-- ============================================================================

CREATE TABLE IF NOT EXISTS notion_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    notion_id VARCHAR(255) NOT NULL,
    database_id UUID REFERENCES notion_databases(id) ON DELETE SET NULL,

    -- Page metadata
    title VARCHAR(500),
    icon VARCHAR(500),
    cover VARCHAR(500),
    url VARCHAR(500),
    archived BOOLEAN DEFAULT FALSE,

    -- Properties (from database schema)
    properties JSONB DEFAULT '{}',

    -- Content (optional - for full page sync)
    content JSONB DEFAULT '[]',

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, notion_id)
);

CREATE INDEX IF NOT EXISTS idx_notion_pages_user ON notion_pages(user_id);
CREATE INDEX IF NOT EXISTS idx_notion_pages_database ON notion_pages(database_id);
CREATE INDEX IF NOT EXISTS idx_notion_pages_title ON notion_pages(user_id, title);
CREATE INDEX IF NOT EXISTS idx_notion_pages_archived ON notion_pages(user_id, archived);

-- ============================================================================
-- SLACK CHANNELS (Slack-specific)
-- ============================================================================

CREATE TABLE IF NOT EXISTS slack_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    slack_id VARCHAR(255) NOT NULL,

    -- Channel metadata
    name VARCHAR(255) NOT NULL,
    is_private BOOLEAN DEFAULT FALSE,
    is_dm BOOLEAN DEFAULT FALSE,
    is_mpim BOOLEAN DEFAULT FALSE,
    member_count INT DEFAULT 0,
    topic TEXT,
    purpose TEXT,
    unread_count INT DEFAULT 0,

    -- Activity tracking
    last_activity TIMESTAMP WITH TIME ZONE,

    -- Sync tracking
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, slack_id)
);

CREATE INDEX IF NOT EXISTS idx_slack_channels_user ON slack_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_slack_channels_name ON slack_channels(user_id, name);
CREATE INDEX IF NOT EXISTS idx_slack_channels_activity ON slack_channels(user_id, last_activity DESC);

-- ============================================================================
-- SLACK MESSAGES
-- ============================================================================

CREATE TABLE IF NOT EXISTS slack_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    channel_id UUID NOT NULL REFERENCES slack_channels(id) ON DELETE CASCADE,
    slack_ts VARCHAR(50) NOT NULL,

    -- Sender info
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),

    -- Message content
    content TEXT,
    thread_ts VARCHAR(50),
    reply_count INT DEFAULT 0,
    is_edited BOOLEAN DEFAULT FALSE,

    -- Timestamps
    sent_at TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, channel_id, slack_ts)
);

CREATE INDEX IF NOT EXISTS idx_slack_messages_channel ON slack_messages(channel_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_slack_messages_thread ON slack_messages(channel_id, thread_ts);
CREATE INDEX IF NOT EXISTS idx_slack_messages_sender ON slack_messages(sender_id);

-- ============================================================================
-- LINEAR ISSUES
-- ============================================================================

CREATE TABLE IF NOT EXISTS linear_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    external_id TEXT NOT NULL,
    identifier TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL DEFAULT 'backlog',
    priority INTEGER DEFAULT 0,
    assignee TEXT,
    project TEXT,
    team TEXT NOT NULL,
    due_date DATE,
    external_created_at TIMESTAMPTZ,
    external_updated_at TIMESTAMPTZ,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_linear_issues_user ON linear_issues(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_issues_identifier ON linear_issues(identifier);
CREATE INDEX IF NOT EXISTS idx_linear_issues_state ON linear_issues(user_id, state);
CREATE INDEX IF NOT EXISTS idx_linear_issues_team ON linear_issues(user_id, team);
CREATE INDEX IF NOT EXISTS idx_linear_issues_updated ON linear_issues(external_updated_at DESC);

-- ============================================================================
-- LINEAR PROJECTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS linear_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL DEFAULT 'planned',
    progress DECIMAL(5,2) DEFAULT 0,
    start_date DATE,
    target_date DATE,
    team TEXT,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_linear_projects_user ON linear_projects(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_projects_state ON linear_projects(user_id, state);

-- ============================================================================
-- LINEAR TEAMS
-- ============================================================================

CREATE TABLE IF NOT EXISTS linear_teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    external_id TEXT NOT NULL,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    issue_count INTEGER DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_linear_teams_user ON linear_teams(user_id);
CREATE INDEX IF NOT EXISTS idx_linear_teams_key ON linear_teams(user_id, key);

-- Update timestamp triggers for new tables
CREATE OR REPLACE FUNCTION update_linear_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'emails', 'channels', 'channel_messages', 'integration_sync_log',
        'notion_databases', 'notion_pages', 'slack_channels', 'slack_messages',
        'linear_issues', 'linear_projects', 'linear_teams'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        BEGIN
            EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_integration_updated_at()', t, t);
        EXCEPTION WHEN undefined_function THEN
            EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_linear_updated_at()', t, t);
        END;
    END LOOP;
END $$;

-- Comments for new tables
COMMENT ON TABLE emails IS 'Synced emails from Gmail and other providers';
COMMENT ON TABLE channels IS 'Synced communication channels from Slack/Discord/Teams';
COMMENT ON TABLE channel_messages IS 'Messages within communication channels';
COMMENT ON TABLE integration_sync_log IS 'Log of all integration sync operations';
COMMENT ON TABLE notion_databases IS 'Synced Notion databases with their property schemas';
COMMENT ON TABLE notion_pages IS 'Synced Notion pages/database entries';
COMMENT ON TABLE slack_channels IS 'Synced Slack channels, DMs, and group messages';
COMMENT ON TABLE slack_messages IS 'Synced Slack messages from channels';
COMMENT ON TABLE linear_issues IS 'Synced Linear issues';
COMMENT ON TABLE linear_projects IS 'Synced Linear projects';
COMMENT ON TABLE linear_teams IS 'Synced Linear teams';

-- ============================================================================
-- FLEXIBLE TABLES SYSTEM (036)
-- ============================================================================

-- Field types enum
DO $$ BEGIN
    CREATE TYPE custom_field_type AS ENUM (
        'text', 'long_text', 'number', 'currency', 'percent',
        'date', 'datetime', 'checkbox', 'select', 'multi_select',
        'user', 'email', 'phone', 'url', 'attachment',
        'relation', 'lookup', 'formula', 'rollup', 'count',
        'created_time', 'modified_time', 'created_by', 'modified_by',
        'autonumber', 'rating', 'duration', 'json'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- View types
DO $$ BEGIN
    CREATE TYPE custom_view_type AS ENUM (
        'grid', 'kanban', 'calendar', 'gallery', 'timeline', 'list', 'form'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS custom_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(50),
    visibility VARCHAR(20) DEFAULT 'private',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS custom_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(50),
    workspace_id UUID REFERENCES custom_workspaces(id) ON DELETE SET NULL,
    folder_id UUID,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS custom_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    field_type custom_field_type NOT NULL DEFAULT 'text',
    description TEXT,
    position INT NOT NULL DEFAULT 0,
    config JSONB DEFAULT '{}',
    required BOOLEAN DEFAULT FALSE,
    unique_values BOOLEAN DEFAULT FALSE,
    hidden BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(table_id, name)
);

CREATE TABLE IF NOT EXISTS custom_field_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_id UUID NOT NULL REFERENCES custom_fields(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(field_id, name)
);

CREATE TABLE IF NOT EXISTS custom_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,
    data JSONB DEFAULT '{}',
    position INT,
    created_by VARCHAR(255),
    modified_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS custom_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES custom_tables(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    view_type custom_view_type NOT NULL DEFAULT 'grid',
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    filters JSONB DEFAULT '[]',
    sorts JSONB DEFAULT '[]',
    group_by UUID,
    view_settings JSONB DEFAULT '{}',
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS custom_record_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id UUID NOT NULL REFERENCES custom_records(id) ON DELETE CASCADE,
    field_id UUID,
    action VARCHAR(50) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    changed_by VARCHAR(255),
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_workspaces_user ON custom_workspaces(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_tables_user ON custom_tables(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_tables_workspace ON custom_tables(workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_fields_table ON custom_fields(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_field_options_field ON custom_field_options(field_id);
CREATE INDEX IF NOT EXISTS idx_custom_records_table ON custom_records(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_records_data ON custom_records USING GIN (data);
CREATE INDEX IF NOT EXISTS idx_custom_views_table ON custom_views(table_id);
CREATE INDEX IF NOT EXISTS idx_custom_record_history_record ON custom_record_history(record_id);

-- ============================================================================
-- ACTIVITY LOG SYSTEM (037)
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE activity_action AS ENUM (
        'created', 'updated', 'deleted', 'restored', 'archived',
        'status_changed', 'priority_changed', 'assigned', 'unassigned',
        'linked', 'unlinked', 'moved',
        'commented', 'mentioned', 'attached', 'detached',
        'shared', 'unshared',
        'synced', 'imported', 'exported',
        'custom'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    entity_name VARCHAR(500),
    action activity_action NOT NULL,
    action_detail VARCHAR(255),
    actor_id VARCHAR(255),
    actor_name VARCHAR(255),
    changes JSONB,
    related_entity_type VARCHAR(100),
    related_entity_id UUID,
    related_entity_name VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_log_user ON activity_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_log_entity ON activity_log(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_log_actor ON activity_log(actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_log_recent ON activity_log(created_at DESC);

-- ============================================================================
-- ATTACHMENTS SYSTEM (038)
-- ============================================================================

CREATE TABLE IF NOT EXISTS attachment_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES attachment_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50),
    entity_type VARCHAR(100),
    entity_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    file_name VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'local',
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),
    thumbnail_url TEXT,
    preview_url TEXT,
    width INT,
    height INT,
    page_count INT,
    duration_seconds INT,
    processing_status VARCHAR(50) DEFAULT 'ready',
    processing_error TEXT,
    metadata JSONB DEFAULT '{}',
    uploaded_by VARCHAR(255),
    folder_id UUID REFERENCES attachment_folders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS attachment_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attachment_id UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    version_label VARCHAR(100),
    file_size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(attachment_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_attachment_folders_user ON attachment_folders(user_id);
CREATE INDEX IF NOT EXISTS idx_attachments_entity ON attachments(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_attachments_user ON attachments(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_attachments_folder ON attachments(folder_id);
CREATE INDEX IF NOT EXISTS idx_attachment_versions_attachment ON attachment_versions(attachment_id);

-- ============================================================================
-- TAGS SYSTEM (039)
-- ============================================================================

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(50),
    icon VARCHAR(50),
    parent_id UUID REFERENCES tags(id) ON DELETE SET NULL,
    group_name VARCHAR(100),
    allowed_entity_types TEXT[] DEFAULT '{}',
    usage_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, slug)
);

CREATE TABLE IF NOT EXISTS tag_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    assigned_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tag_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_tags_user ON tags(user_id);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(user_id, name);
CREATE INDEX IF NOT EXISTS idx_tags_group ON tags(user_id, group_name);
CREATE INDEX IF NOT EXISTS idx_tag_assignments_entity ON tag_assignments(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_tag_assignments_tag ON tag_assignments(tag_id);

-- ============================================================================
-- ENTITY LINKS SYSTEM (040)
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE entity_link_type AS ENUM (
        'related', 'mentions',
        'parent_of', 'child_of',
        'blocks', 'blocked_by', 'depends_on',
        'duplicate_of', 'original_of',
        'derived_from', 'spawned',
        'task_for', 'project_for', 'note_about', 'meeting_about',
        'custom'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID,
    source_type VARCHAR(100) NOT NULL,
    source_id UUID NOT NULL,
    source_name VARCHAR(500),
    target_type VARCHAR(100) NOT NULL,
    target_id UUID NOT NULL,
    target_name VARCHAR(500),
    link_type entity_link_type NOT NULL DEFAULT 'related',
    custom_link_type VARCHAR(100),
    is_bidirectional BOOLEAN DEFAULT FALSE,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(source_type, source_id, target_type, target_id, link_type)
);

CREATE INDEX IF NOT EXISTS idx_entity_links_source ON entity_links(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_target ON entity_links(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_user ON entity_links(user_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_type ON entity_links(link_type);
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_source ON entity_links(workspace_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_target ON entity_links(workspace_id, target_type, target_id);

-- ============================================================================
-- CRM SYSTEM (041)
-- ============================================================================

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    industry VARCHAR(100),
    company_size VARCHAR(50),
    website VARCHAR(500),
    email VARCHAR(255),
    phone VARCHAR(50),
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100),
    annual_revenue DECIMAL(15,2),
    currency VARCHAR(10) DEFAULT 'USD',
    fiscal_year_end VARCHAR(20),
    tax_id VARCHAR(100),
    linkedin_url VARCHAR(500),
    twitter_handle VARCHAR(100),
    owner_id VARCHAR(255),
    lifecycle_stage VARCHAR(50),
    lead_source VARCHAR(100),
    health_score INT,
    engagement_score INT,
    logo_url TEXT,
    custom_fields JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    external_id VARCHAR(255),
    external_source VARCHAR(50),
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contact_company_relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    job_title VARCHAR(255),
    department VARCHAR(100),
    role_type VARCHAR(50),
    is_primary BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    started_at DATE,
    ended_at DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(contact_id, company_id)
);

CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    -- The workspaces table is defined later in this canonical schema.
    -- Add the foreign key after that table exists so fresh bootstraps work.
    workspace_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pipeline_type VARCHAR(50) DEFAULT 'sales',
    currency VARCHAR(10) DEFAULT 'USD',
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    color VARCHAR(50),
    icon VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipelines_workspace_user ON pipelines(workspace_id, user_id);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    position INT NOT NULL DEFAULT 0,
    probability INT DEFAULT 0,
    stage_type VARCHAR(50) DEFAULT 'open',
    rotting_days INT,
    color VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(pipeline_id, name)
);

CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    pipeline_id UUID NOT NULL REFERENCES pipelines(id),
    stage_id UUID NOT NULL REFERENCES pipeline_stages(id),
    name VARCHAR(500) NOT NULL,
    description TEXT,
    amount DECIMAL(15,2),
    currency VARCHAR(10) DEFAULT 'USD',
    probability INT,
    expected_close_date DATE,
    actual_close_date DATE,
    owner_id VARCHAR(255),
    company_id UUID REFERENCES companies(id),
    primary_contact_id UUID,
    status VARCHAR(50) DEFAULT 'open',
    lost_reason VARCHAR(255),
    priority VARCHAR(20) DEFAULT 'medium',
    lead_source VARCHAR(100),
    deal_score INT,
    custom_fields JSONB DEFAULT '{}',
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    stage_entered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deals_client_id ON deals(client_id);

DO $$ BEGIN
    CREATE TYPE crm_activity_type AS ENUM (
        'call', 'email', 'meeting', 'note', 'task', 'demo',
        'proposal_sent', 'contract_sent', 'follow_up', 'linkedin_message', 'other'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS crm_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    activity_type crm_activity_type NOT NULL,
    subject VARCHAR(500) NOT NULL,
    description TEXT,
    outcome TEXT,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    contact_id UUID,
    participants JSONB DEFAULT '[]',
    activity_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_minutes INT,
    call_direction VARCHAR(20),
    call_disposition VARCHAR(50),
    call_recording_url TEXT,
    email_direction VARCHAR(20),
    email_message_id VARCHAR(255),
    meeting_location VARCHAR(255),
    meeting_url TEXT,
    owner_id VARCHAR(255),
    completed_by VARCHAR(255),
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deal_stage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    from_stage_id UUID REFERENCES pipeline_stages(id),
    to_stage_id UUID NOT NULL REFERENCES pipeline_stages(id),
    changed_by VARCHAR(255),
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    duration_seconds INT,
    deal_amount DECIMAL(15,2)
);

CREATE INDEX IF NOT EXISTS idx_companies_user ON companies(user_id);
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(user_id, name);
CREATE INDEX IF NOT EXISTS idx_companies_industry ON companies(user_id, industry);
CREATE INDEX IF NOT EXISTS idx_contact_company_contact ON contact_company_relations(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_company_company ON contact_company_relations(company_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_user ON pipelines(user_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_deals_user ON deals(user_id);
CREATE INDEX IF NOT EXISTS idx_deals_pipeline ON deals(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage_id);
CREATE INDEX IF NOT EXISTS idx_deals_company ON deals(company_id);
CREATE INDEX IF NOT EXISTS idx_deals_status ON deals(user_id, status);
CREATE INDEX IF NOT EXISTS idx_crm_activities_user ON crm_activities(user_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_deal ON crm_activities(deal_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_company ON crm_activities(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_date ON crm_activities(activity_date DESC);
CREATE INDEX IF NOT EXISTS idx_deal_stage_history_deal ON deal_stage_history(deal_id);

-- ============================================================================
-- UPDATE TRIGGERS FOR NEW TABLES
-- ============================================================================

CREATE OR REPLACE FUNCTION update_custom_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- MEMORIES SYSTEM (Migration 016)
-- ============================================================================

CREATE TABLE IF NOT EXISTS memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    content TEXT NOT NULL,
    memory_type VARCHAR(50) NOT NULL,
    category VARCHAR(100),
    source_type VARCHAR(50) NOT NULL,
    source_id UUID,
    source_context TEXT,
    project_id UUID,
    node_id UUID,
    importance_score DECIMAL(3,2) DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    embedding_model VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(memory_type);
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);
CREATE INDEX IF NOT EXISTS idx_memories_node ON memories(node_id);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance_score DESC);
CREATE INDEX IF NOT EXISTS idx_memories_active ON memories(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at DESC);

CREATE TABLE IF NOT EXISTS memory_associations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    relevance_score DECIMAL(3,2) DEFAULT 0.5,
    association_type VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_assoc_memory ON memory_associations(memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_assoc_entity ON memory_associations(entity_type, entity_id);

CREATE TABLE IF NOT EXISTS memory_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    access_type VARCHAR(50) NOT NULL,
    accessing_agent VARCHAR(100),
    conversation_id UUID,
    trigger_query TEXT,
    relevance_score DECIMAL(3,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_access_memory ON memory_access_log(memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_access_time ON memory_access_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_access_user ON memory_access_log(user_id);

CREATE TABLE IF NOT EXISTS user_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    fact_key VARCHAR(255) NOT NULL,
    fact_value TEXT NOT NULL,
    fact_type VARCHAR(50) NOT NULL,
    source_memory_id UUID REFERENCES memories(id) ON DELETE SET NULL,
    confidence_score DECIMAL(3,2) DEFAULT 1.0,
    is_active BOOLEAN DEFAULT TRUE,
    last_confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, fact_key)
);

CREATE INDEX IF NOT EXISTS idx_user_facts_user ON user_facts(user_id);
CREATE INDEX IF NOT EXISTS idx_user_facts_type ON user_facts(fact_type);
CREATE INDEX IF NOT EXISTS idx_user_facts_active ON user_facts(user_id, is_active);

-- ============================================================================
-- LEARNING SYSTEM (Migration 021)
-- ============================================================================

CREATE TABLE IF NOT EXISTS learning_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    learning_type VARCHAR(50) NOT NULL,
    learning_content TEXT NOT NULL,
    learning_summary VARCHAR(500),
    source_type VARCHAR(50) NOT NULL,
    source_id UUID,
    source_context TEXT,
    confidence_score DECIMAL(3,2) DEFAULT 0.5,
    times_applied INTEGER DEFAULT 0,
    last_applied_at TIMESTAMPTZ,
    successful_applications INTEGER DEFAULT 0,
    created_memory_id UUID,
    created_fact_key VARCHAR(255),
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    was_validated BOOLEAN DEFAULT FALSE,
    validated_at TIMESTAMPTZ,
    validation_result VARCHAR(50),
    validation_notes TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    superseded_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_learning_events_user ON learning_events(user_id);
CREATE INDEX IF NOT EXISTS idx_learning_events_type ON learning_events(learning_type);
CREATE INDEX IF NOT EXISTS idx_learning_events_source ON learning_events(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_learning_events_active ON learning_events(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_learning_events_confidence ON learning_events(user_id, confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_learning_events_created ON learning_events(created_at DESC);

CREATE TABLE IF NOT EXISTS user_behavior_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(100) NOT NULL,
    pattern_key VARCHAR(255) NOT NULL,
    pattern_value TEXT NOT NULL,
    pattern_description TEXT,
    observation_count INTEGER DEFAULT 1,
    first_observed_at TIMESTAMPTZ DEFAULT NOW(),
    last_observed_at TIMESTAMPTZ DEFAULT NOW(),
    evidence_ids UUID[] DEFAULT '{}',
    confidence_score DECIMAL(3,2) DEFAULT 0.5,
    min_observations_for_confidence INTEGER DEFAULT 3,
    is_applied BOOLEAN DEFAULT FALSE,
    applied_in_prompt BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    deactivated_at TIMESTAMPTZ,
    deactivation_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, pattern_type, pattern_key)
);

CREATE INDEX IF NOT EXISTS idx_behavior_patterns_user ON user_behavior_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_type ON user_behavior_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_active ON user_behavior_patterns(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_confidence ON user_behavior_patterns(user_id, confidence_score DESC);

CREATE TABLE IF NOT EXISTS feedback_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    feedback_type VARCHAR(50) NOT NULL,
    feedback_value TEXT,
    rating INTEGER,
    conversation_id UUID,
    agent_type VARCHAR(100),
    focus_mode VARCHAR(50),
    original_content TEXT,
    expected_content TEXT,
    was_processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    resulting_learning_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_log_user ON feedback_log(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_log_target ON feedback_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_feedback_log_type ON feedback_log(feedback_type);
CREATE INDEX IF NOT EXISTS idx_feedback_log_unprocessed ON feedback_log(was_processed) WHERE was_processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_feedback_log_created ON feedback_log(created_at DESC);

CREATE TABLE IF NOT EXISTS personalization_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    preferred_tone VARCHAR(50) DEFAULT 'professional',
    preferred_verbosity VARCHAR(50) DEFAULT 'balanced',
    preferred_format VARCHAR(50) DEFAULT 'structured',
    prefers_examples BOOLEAN DEFAULT TRUE,
    prefers_analogies BOOLEAN DEFAULT FALSE,
    prefers_code_samples BOOLEAN DEFAULT FALSE,
    prefers_visual_aids BOOLEAN DEFAULT FALSE,
    expertise_areas TEXT[] DEFAULT '{}',
    learning_areas TEXT[] DEFAULT '{}',
    common_topics TEXT[] DEFAULT '{}',
    timezone VARCHAR(100),
    preferred_working_hours JSONB DEFAULT '{}',
    most_active_hours INTEGER[] DEFAULT '{}',
    total_conversations INTEGER DEFAULT 0,
    total_feedback_given INTEGER DEFAULT 0,
    positive_feedback_ratio DECIMAL(3,2) DEFAULT 0.5,
    profile_completeness DECIMAL(3,2) DEFAULT 0.0,
    last_profile_update TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_personalization_user ON personalization_profiles(user_id);

-- ============================================================================
-- CONTEXT SYSTEM (Migration 017)
-- ============================================================================

CREATE TABLE IF NOT EXISTS context_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    context_tree JSONB NOT NULL DEFAULT '{}',
    summary TEXT,
    key_facts TEXT[] DEFAULT '{}',
    document_types TEXT[] DEFAULT '{}',
    total_documents INTEGER DEFAULT 0,
    total_file_size_bytes BIGINT DEFAULT 0,
    total_contexts INTEGER DEFAULT 0,
    total_memories INTEGER DEFAULT 0,
    total_artifacts INTEGER DEFAULT 0,
    total_tasks INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_context_profiles_user ON context_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_context_profiles_entity ON context_profiles(entity_type, entity_id);

CREATE TABLE IF NOT EXISTS context_loading_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(50) NOT NULL,
    trigger_value VARCHAR(255),
    load_memories BOOLEAN DEFAULT TRUE,
    memory_types TEXT[] DEFAULT '{}',
    memory_limit INTEGER DEFAULT 10,
    load_contexts BOOLEAN DEFAULT TRUE,
    context_categories TEXT[] DEFAULT '{}',
    context_limit INTEGER DEFAULT 5,
    load_artifacts BOOLEAN DEFAULT FALSE,
    artifact_types TEXT[] DEFAULT '{}',
    artifact_limit INTEGER DEFAULT 3,
    load_recent_conversations BOOLEAN DEFAULT TRUE,
    conversation_limit INTEGER DEFAULT 3,
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_context_rules_user ON context_loading_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_context_rules_trigger ON context_loading_rules(trigger_type, trigger_value);
CREATE INDEX IF NOT EXISTS idx_context_rules_active ON context_loading_rules(user_id, is_active);

CREATE TABLE IF NOT EXISTS context_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID NOT NULL,
    agent_type VARCHAR(100) NOT NULL,
    agent_id UUID,
    max_context_tokens INTEGER NOT NULL,
    used_context_tokens INTEGER DEFAULT 0,
    available_tokens INTEGER,
    loaded_memories UUID[] DEFAULT '{}',
    loaded_contexts UUID[] DEFAULT '{}',
    loaded_artifacts UUID[] DEFAULT '{}',
    loaded_documents UUID[] DEFAULT '{}',
    base_system_prompt TEXT,
    injected_context TEXT,
    total_system_prompt_tokens INTEGER,
    project_id UUID,
    node_id UUID,
    focus_mode VARCHAR(50),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_context_sessions_user ON context_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_context_sessions_conversation ON context_sessions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_context_sessions_active ON context_sessions(user_id, ended_at) WHERE ended_at IS NULL;

CREATE TABLE IF NOT EXISTS context_retrieval_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES context_sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    retrieval_type VARCHAR(50) NOT NULL,
    item_id UUID NOT NULL,
    item_title VARCHAR(255),
    retrieval_method VARCHAR(50) NOT NULL,
    query_used TEXT,
    relevance_score DECIMAL(3,2),
    token_count INTEGER,
    was_truncated BOOLEAN DEFAULT FALSE,
    was_used_in_response BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_context_retrieval_session ON context_retrieval_log(session_id);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_item ON context_retrieval_log(item_id);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_user ON context_retrieval_log(user_id);

-- ============================================================================
-- DOCUMENT PROCESSING (Migration 019)
-- ============================================================================

CREATE TABLE IF NOT EXISTS uploaded_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    filename VARCHAR(500) NOT NULL,
    original_filename VARCHAR(500) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    file_type VARCHAR(50) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    storage_path VARCHAR(1000) NOT NULL,
    storage_provider VARCHAR(50) DEFAULT 'local',
    extracted_text TEXT,
    page_count INTEGER,
    word_count INTEGER,
    context_profile_id UUID,
    project_id UUID,
    node_id UUID,
    document_type VARCHAR(100),
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    processing_status VARCHAR(50) DEFAULT 'pending',
    processing_error TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uploaded_docs_user ON uploaded_documents(user_id);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_profile ON uploaded_documents(context_profile_id) WHERE context_profile_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_project ON uploaded_documents(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_node ON uploaded_documents(node_id) WHERE node_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_type ON uploaded_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_status ON uploaded_documents(processing_status);

CREATE TABLE IF NOT EXISTS document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER,
    page_number INTEGER,
    start_char INTEGER,
    end_char INTEGER,
    section_title VARCHAR(255),
    chunk_type VARCHAR(50) DEFAULT 'text',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_chunks_document ON document_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_index ON document_chunks(document_id, chunk_index);

CREATE TABLE IF NOT EXISTS document_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    reference_type VARCHAR(50) DEFAULT 'related',
    context TEXT,
    relevance_score DECIMAL(3,2) DEFAULT 0.5,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_refs_document ON document_references(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_refs_entity ON document_references(entity_type, entity_id);

-- ============================================================================
-- CONVERSATION SUMMARIES (Migration 020)
-- ============================================================================

CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    key_points TEXT[] DEFAULT '{}',
    decisions_made TEXT[] DEFAULT '{}',
    action_items TEXT[] DEFAULT '{}',
    topics TEXT[] DEFAULT '{}',
    mentioned_entities JSONB DEFAULT '{}',
    message_count INTEGER,
    time_range_start TIMESTAMPTZ,
    time_range_end TIMESTAMPTZ,
    summarized_at TIMESTAMPTZ DEFAULT NOW(),
    summary_version INTEGER DEFAULT 1,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conv_summaries_conv ON conversation_summaries(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_user ON conversation_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_time ON conversation_summaries(summarized_at DESC);

CREATE TABLE IF NOT EXISTS context_profile_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_profile_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    item_id UUID NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    token_estimate INTEGER,
    last_accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_auto_added BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(context_profile_id, item_type, item_id)
);

CREATE INDEX IF NOT EXISTS idx_profile_items_profile ON context_profile_items(context_profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_type ON context_profile_items(item_type, item_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_user ON context_profile_items(user_id);

-- ============================================================================
-- APPLICATION PROFILES (Migration 022)
-- ============================================================================

CREATE TABLE IF NOT EXISTS application_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    app_type VARCHAR(100),
    version VARCHAR(50),
    tech_stack JSONB DEFAULT '{}',
    languages TEXT[] DEFAULT '{}',
    frameworks TEXT[] DEFAULT '{}',
    structure_tree JSONB NOT NULL DEFAULT '{}',
    root_path VARCHAR(1000),
    components JSONB DEFAULT '[]',
    total_components INTEGER DEFAULT 0,
    modules JSONB DEFAULT '[]',
    total_modules INTEGER DEFAULT 0,
    icons JSONB DEFAULT '[]',
    assets JSONB DEFAULT '[]',
    api_endpoints JSONB DEFAULT '[]',
    total_endpoints INTEGER DEFAULT 0,
    database_schema JSONB DEFAULT '{}',
    total_tables INTEGER DEFAULT 0,
    conventions JSONB DEFAULT '{}',
    coding_standards TEXT,
    integration_points JSONB DEFAULT '[]',
    readme_summary TEXT,
    documentation_urls TEXT[] DEFAULT '{}',
    last_synced_at TIMESTAMPTZ,
    sync_source VARCHAR(255),
    sync_branch VARCHAR(100),
    sync_commit VARCHAR(100),
    auto_sync_enabled BOOLEAN DEFAULT FALSE,
    last_analyzed_at TIMESTAMPTZ,
    analysis_version INTEGER DEFAULT 1,
    lines_of_code INTEGER,
    file_count INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_profiles_user ON application_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_app_profiles_type ON application_profiles(app_type);
CREATE INDEX IF NOT EXISTS idx_app_profiles_name ON application_profiles(user_id, name);

CREATE TABLE IF NOT EXISTS application_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    component_type VARCHAR(100),
    description TEXT,
    props JSONB DEFAULT '[]',
    events JSONB DEFAULT '[]',
    slots JSONB DEFAULT '[]',
    imports TEXT[] DEFAULT '{}',
    exported_as VARCHAR(255),
    usage_examples JSONB DEFAULT '[]',
    used_in TEXT[] DEFAULT '{}',
    lines_of_code INTEGER,
    last_modified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_app_components_profile ON application_components(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_components_type ON application_components(component_type);
CREATE INDEX IF NOT EXISTS idx_app_components_name ON application_components(app_profile_id, name);

CREATE TABLE IF NOT EXISTS application_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    handler_path VARCHAR(1000),
    description TEXT,
    summary VARCHAR(255),
    path_params JSONB DEFAULT '[]',
    query_params JSONB DEFAULT '[]',
    body_schema JSONB DEFAULT '{}',
    response_schema JSONB DEFAULT '{}',
    auth_required BOOLEAN DEFAULT FALSE,
    required_permissions TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    deprecated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, method, path)
);

CREATE INDEX IF NOT EXISTS idx_app_endpoints_profile ON application_api_endpoints(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_method ON application_api_endpoints(method);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_path ON application_api_endpoints(app_profile_id, path);

-- ============================================================================
-- DASHBOARD & ANALYTICS ENHANCEMENTS (Migration 023)
-- ============================================================================

-- Analytics Snapshots - Historical metrics tracking for trends
CREATE TABLE IF NOT EXISTS analytics_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID,
    snapshot_date DATE NOT NULL,
    metrics JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_analytics_snapshots_user_date 
    ON analytics_snapshots(user_id, snapshot_date DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_snapshots_workspace 
    ON analytics_snapshots(workspace_id) WHERE workspace_id IS NOT NULL;

-- Dashboard Views - Dashboard usage tracking
CREATE TABLE IF NOT EXISTS dashboard_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    session_id VARCHAR(100),
    duration_seconds INTEGER,
    widget_interactions JSONB DEFAULT '[]',
    source VARCHAR(50),
    device_type VARCHAR(20)
);

CREATE INDEX IF NOT EXISTS idx_dashboard_views_dashboard 
    ON dashboard_views(dashboard_id, viewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_dashboard_views_user 
    ON dashboard_views(user_id, viewed_at DESC);

-- Dashboard Shares - Granular sharing permissions
CREATE TABLE IF NOT EXISTS dashboard_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL,
    shared_with_user_id VARCHAR(255),
    shared_with_role VARCHAR(100),
    shared_with_workspace_id UUID,
    permission VARCHAR(20) DEFAULT 'view',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    UNIQUE(dashboard_id, shared_with_user_id),
    UNIQUE(dashboard_id, shared_with_role)
);

CREATE INDEX IF NOT EXISTS idx_dashboard_shares_dashboard 
    ON dashboard_shares(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_user 
    ON dashboard_shares(shared_with_user_id) WHERE shared_with_user_id IS NOT NULL;

-- Widget Data Cache - Performance optimization for expensive queries
CREATE TABLE IF NOT EXISTS widget_data_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    widget_type VARCHAR(100) NOT NULL,
    cache_key VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    hit_count INTEGER DEFAULT 0,
    last_hit_at TIMESTAMPTZ,
    UNIQUE(user_id, widget_type, cache_key)
);

CREATE INDEX IF NOT EXISTS idx_widget_cache_lookup 
    ON widget_data_cache(user_id, widget_type, cache_key);
CREATE INDEX IF NOT EXISTS idx_widget_cache_expiry 
    ON widget_data_cache(expires_at);

-- ============================================================================
-- WORKSPACES (Multi-tenant support)
-- ============================================================================

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    logo_url VARCHAR(500),
    plan_type VARCHAR(50) DEFAULT 'free',
    max_members INTEGER DEFAULT 5,
    max_projects INTEGER DEFAULT 10,
    max_storage_gb INTEGER DEFAULT 5,
    owner_id VARCHAR(255) NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'pipelines_workspace_id_fkey'
          AND conrelid = 'pipelines'::regclass
    ) THEN
        ALTER TABLE pipelines
            ADD CONSTRAINT pipelines_workspace_id_fkey
            FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Provider records remain account-owned. These routes attach communication
-- sources to governed workspaces without moving or duplicating source data.
CREATE TABLE IF NOT EXISTS communication_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL CHECK (provider IN ('gmail', 'outlook', 'slack', 'teams', 'whatsapp')),
    scope VARCHAR(24) NOT NULL CHECK (scope IN ('account', 'conversation')),
    external_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider, scope, external_id)
);
CREATE INDEX IF NOT EXISTS idx_communication_routes_resolve
    ON communication_routes(user_id, provider, scope, external_id) WHERE enabled;
CREATE INDEX IF NOT EXISTS idx_communication_routes_workspace ON communication_routes(workspace_id);

-- Workspace Roles
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    description TEXT,
    color VARCHAR(20),
    icon VARCHAR(50),
    hierarchy_level INTEGER DEFAULT 0,
    is_system BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    permissions JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);

-- Workspace Members
CREATE TABLE IF NOT EXISTS workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role_id UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role_name VARCHAR(100) DEFAULT 'member',
    role VARCHAR(100) DEFAULT 'member',
    status VARCHAR(20) DEFAULT 'active',
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    invited_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);

-- Workspace Invites
CREATE TABLE IF NOT EXISTS workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role_id UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role VARCHAR(100) DEFAULT 'member',
    token VARCHAR(255) UNIQUE,
    status VARCHAR(20) DEFAULT 'pending',
    expires_at TIMESTAMPTZ,
    invited_by VARCHAR(255) NOT NULL,
    invited_by_id VARCHAR(255),
    invited_by_name VARCHAR(255),
    invited_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    accepted_by VARCHAR(255),
    accepted_by_user_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace ON workspace_invites(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_email ON workspace_invites(email);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_token ON workspace_invites(token);

-- Workspace Versions
CREATE TABLE IF NOT EXISTS workspace_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    version_number VARCHAR(20) NOT NULL,
    snapshot_data JSONB NOT NULL,
    snapshot_metadata JSONB DEFAULT '{}'::jsonb,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_workspace_versions_workspace ON workspace_versions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_versions_created_at ON workspace_versions(workspace_id, created_at DESC);

-- Workspace Memories
CREATE TABLE IF NOT EXISTS workspace_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    title VARCHAR(255),
    summary TEXT,
    content TEXT NOT NULL,
    memory_type VARCHAR(50) DEFAULT 'general',
    category VARCHAR(100) DEFAULT 'general',
    scope_type VARCHAR(50),
    scope_id UUID,
    visibility VARCHAR(20) DEFAULT 'workspace',
    created_by VARCHAR(255),
    importance_score FLOAT DEFAULT 0.5,
    tags TEXT[],
    source VARCHAR(100),
    embedding vector(1536),
    metadata JSONB DEFAULT '{}',
    is_pinned BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    is_archived BOOLEAN DEFAULT FALSE,
    access_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace ON workspace_memories(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_category ON workspace_memories(category);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_type ON workspace_memories(memory_type);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_scope ON workspace_memories(scope_type, scope_id);

-- User Workspace Profiles
CREATE TABLE IF NOT EXISTS user_workspace_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    display_name VARCHAR(255),
    title VARCHAR(255),
    department VARCHAR(255),
    avatar_url VARCHAR(500),
    work_email VARCHAR(255),
    phone VARCHAR(50),
    timezone VARCHAR(100),
    working_hours JSONB DEFAULT '{}',
    notification_preferences JSONB DEFAULT '{}',
    expertise_areas TEXT[],
    bio TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_user ON user_workspace_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_workspace ON user_workspace_profiles(workspace_id);

-- Workspace Project Members
CREATE TABLE IF NOT EXISTS workspace_project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    project_role VARCHAR(50) DEFAULT 'member',
    assigned_by VARCHAR(255),
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    notification_level VARCHAR(20) DEFAULT 'all',
    permissions JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_project_members_project ON workspace_project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_user ON workspace_project_members(user_id);

-- ============================================================================
-- USER DASHBOARDS
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    layout JSONB DEFAULT '[]',
    visibility VARCHAR(20) DEFAULT 'private',
    share_token VARCHAR(100) UNIQUE,
    is_enforced BOOLEAN DEFAULT FALSE,
    enforced_for_roles TEXT[],
    created_via VARCHAR(50) DEFAULT 'manual',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_dashboards_user ON user_dashboards(user_id);
CREATE INDEX IF NOT EXISTS idx_user_dashboards_workspace ON user_dashboards(workspace_id);

-- Dashboard Widgets (Widget Type Registry)
CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    widget_type VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) DEFAULT 'general',
    icon VARCHAR(50),
    default_config JSONB DEFAULT '{}',
    config_schema JSONB DEFAULT '{}',
    default_size JSONB DEFAULT '{"width": 2, "height": 2}',
    min_size JSONB DEFAULT '{"width": 1, "height": 1}',
    sse_events TEXT[],
    supported_sizes TEXT[],
    min_width INTEGER DEFAULT 1,
    min_height INTEGER DEFAULT 1,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_premium BOOLEAN DEFAULT FALSE,
    required_permissions TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_category ON dashboard_widgets(category);
CREATE INDEX IF NOT EXISTS idx_dashboard_widgets_enabled ON dashboard_widgets(is_enabled);

-- Dashboard Templates
CREATE TABLE IF NOT EXISTS dashboard_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) DEFAULT 'general',
    thumbnail_url VARCHAR(500),
    preview_image_url VARCHAR(500),
    layout JSONB DEFAULT '[]',
    is_default BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    target_roles TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboard_templates_default ON dashboard_templates(is_default);

-- ============================================================================
-- NOTIFICATIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    entity_type VARCHAR(50),
    entity_id UUID,
    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    sender_avatar_url VARCHAR(500),
    batch_id UUID,
    batch_count INTEGER DEFAULT 1,
    priority VARCHAR(20) DEFAULT 'normal',
    metadata JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    channels_sent TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_entity ON notifications(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC);

-- Notification Preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    email_enabled BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN DEFAULT TRUE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    type_settings JSONB DEFAULT '{}',
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_timezone VARCHAR(100),
    email_digest_enabled BOOLEAN DEFAULT FALSE,
    email_digest_time TIME,
    email_digest_timezone VARCHAR(100),
    muted_types TEXT[],
    custom_settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_notification_preferences_user ON notification_preferences(user_id);

-- Notification Batches
CREATE TABLE IF NOT EXISTS notification_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    batch_key VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    pending_ids UUID[] DEFAULT '{}',
    pending_count INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    dispatch_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_batches_user ON notification_batches(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_batches_pending ON notification_batches(status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notification_batches_dispatch ON notification_batches(dispatch_at);

-- Push Subscriptions
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions(user_id);

-- Push Devices (for mobile)
CREATE TABLE IF NOT EXISTS push_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    push_token TEXT NOT NULL,
    app_version VARCHAR(50),
    os_version VARCHAR(50),
    device_model VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_push_devices_user ON push_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_push_devices_active ON push_devices(is_active) WHERE is_active = TRUE;

-- ============================================================================
-- COMMENTS & MENTIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    content TEXT NOT NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    is_edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMPTZ,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_entity ON comments(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

-- Entity Mentions
CREATE TABLE IF NOT EXISTS entity_mentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type VARCHAR(50) NOT NULL,
    source_id UUID NOT NULL,
    mentioned_user_id VARCHAR(255) NOT NULL,
    mention_text VARCHAR(255),
    position_in_text INTEGER,
    entity_type VARCHAR(50),
    entity_id UUID,
    mentioned_by VARCHAR(255) NOT NULL,
    notified BOOLEAN DEFAULT FALSE,
    notified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_mentions_source ON entity_mentions(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_entity_mentions_user ON entity_mentions(mentioned_user_id);
CREATE INDEX IF NOT EXISTS idx_entity_mentions_unnotified ON entity_mentions(notified) WHERE notified = FALSE;

-- Comment Reactions
CREATE TABLE IF NOT EXISTS comment_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    emoji VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(comment_id, user_id, emoji)
);

CREATE INDEX IF NOT EXISTS idx_comment_reactions_comment ON comment_reactions(comment_id);
CREATE INDEX IF NOT EXISTS idx_comment_reactions_user ON comment_reactions(user_id);

-- ============================================================================
-- SYNC CONFLICTS (Migration 043)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sync_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Conflict target
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,

    -- Conflict data
    local_data JSONB NOT NULL,
    remote_data JSONB NOT NULL,
    local_updated_at TIMESTAMPTZ NOT NULL,
    remote_updated_at TIMESTAMPTZ NOT NULL,
    conflict_fields TEXT[] NOT NULL,

    -- Resolution
    resolution_strategy VARCHAR(50),
    resolved_data JSONB,
    resolved_by VARCHAR(255),  -- NULL = automatic
    resolved_at TIMESTAMPTZ,
    reasoning TEXT,

    -- Metadata
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT valid_resolution_strategy CHECK (
        resolution_strategy IN ('timestamp_based', 'field_level_merge', 'manual_review', 'local_wins', 'remote_wins')
    )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_unresolved ON sync_conflicts(resolved_at)
    WHERE resolved_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sync_conflicts_entity ON sync_conflicts(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_detected ON sync_conflicts(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_strategy ON sync_conflicts(resolution_strategy);

COMMENT ON TABLE sync_conflicts IS 'Tracks conflicts detected during bidirectional sync operations';
COMMENT ON COLUMN sync_conflicts.entity_type IS 'Type of entity that has conflicts (workspace, app, module)';
COMMENT ON COLUMN sync_conflicts.conflict_fields IS 'Array of field names that are in conflict';
COMMENT ON COLUMN sync_conflicts.resolution_strategy IS 'Strategy used to resolve the conflict';
COMMENT ON COLUMN sync_conflicts.resolved_by IS 'User who resolved manually, NULL for automatic resolution';
COMMENT ON COLUMN sync_conflicts.reasoning IS 'Human-readable explanation of how conflict was resolved';

-- =============================================================================
-- OSA INTEGRATION TABLES (from 042_osa_integration.sql)
-- =============================================================================

CREATE TABLE IF NOT EXISTS osa_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    module_type VARCHAR(50) NOT NULL,
    schema_definition JSONB,
    api_definition JSONB,
    ui_definition JSONB,
    created_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    workspace_id UUID,
    status VARCHAR(50) DEFAULT 'draft',
    version VARCHAR(50) DEFAULT '1.0.0',
    metadata JSONB DEFAULT '{}',
    tags TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deployed_at TIMESTAMPTZ,
    CONSTRAINT osa_modules_name_workspace_unique UNIQUE(name, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_osa_modules_workspace ON osa_modules(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_modules_created_by ON osa_modules(created_by);
CREATE INDEX IF NOT EXISTS idx_osa_modules_status ON osa_modules(status);
CREATE INDEX IF NOT EXISTS idx_osa_modules_type ON osa_modules(module_type);
CREATE INDEX IF NOT EXISTS idx_osa_modules_tags ON osa_modules USING GIN(tags);

CREATE TABLE IF NOT EXISTS osa_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(20) DEFAULT '2d',
    layout JSONB DEFAULT '{}',
    active_modules UUID[] DEFAULT '{}',
    template_type VARCHAR(50) DEFAULT 'business_os',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT osa_workspaces_user_name_unique UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_osa_workspaces_user ON osa_workspaces(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_workspaces_template ON osa_workspaces(template_type);
CREATE INDEX IF NOT EXISTS idx_osa_workspaces_mode ON osa_workspaces(mode);

DO $$ BEGIN
ALTER TABLE osa_modules
ADD CONSTRAINT fk_osa_modules_workspace
FOREIGN KEY (workspace_id)
REFERENCES osa_workspaces(id)
ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN null; END $$;

CREATE TABLE IF NOT EXISTS osa_module_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_id UUID REFERENCES osa_modules(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    osa_workflow_id VARCHAR(255),
    osa_sandbox_id VARCHAR(255),
    code_repository TEXT,
    deployment_url TEXT,
    status VARCHAR(50) DEFAULT 'generated',
    files_created INTEGER DEFAULT 0,
    tests_passed BOOLEAN DEFAULT false,
    build_status VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    error_stack TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    generated_at TIMESTAMPTZ,
    deployed_at TIMESTAMPTZ,
    last_build_at TIMESTAMPTZ,
    deployment_port INTEGER,
    app_name VARCHAR(255),
    category VARCHAR(100),
    icon_type VARCHAR(50),
    app_description TEXT,
    -- Sandbox container columns (Migration 091)
    container_id VARCHAR(255),
    sandbox_port INTEGER,
    sandbox_url TEXT,
    sandbox_status VARCHAR(50) DEFAULT 'none',
    container_image VARCHAR(255),
    app_type VARCHAR(50),
    last_health_check TIMESTAMPTZ,
    health_status VARCHAR(50) DEFAULT 'unknown'
);

CREATE INDEX IF NOT EXISTS idx_osa_apps_workspace ON osa_module_instances(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_apps_module ON osa_module_instances(module_id);
CREATE INDEX IF NOT EXISTS idx_osa_apps_status ON osa_module_instances(status);
CREATE INDEX IF NOT EXISTS idx_osa_apps_workflow ON osa_module_instances(osa_workflow_id);
CREATE INDEX IF NOT EXISTS idx_osa_apps_deployment_port ON osa_module_instances(deployment_port);
CREATE INDEX IF NOT EXISTS idx_osa_apps_category ON osa_module_instances(category);
CREATE INDEX IF NOT EXISTS idx_osa_apps_workspace_category ON osa_module_instances(workspace_id, category);
-- Sandbox indexes (Migration 091)
CREATE INDEX IF NOT EXISTS idx_osa_apps_container_id ON osa_module_instances(container_id) WHERE container_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_apps_sandbox_status ON osa_module_instances(sandbox_status) WHERE sandbox_status != 'none';
CREATE INDEX IF NOT EXISTS idx_osa_apps_sandbox_port ON osa_module_instances(sandbox_port) WHERE sandbox_port IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_apps_health_status ON osa_module_instances(health_status) WHERE health_status != 'unknown';

-- =============================================================================
-- SANDBOX EVENTS TABLE (Migration 091)
-- Lifecycle events for sandbox containers
-- =============================================================================

CREATE TABLE IF NOT EXISTS sandbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_instance_id UUID NOT NULL REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    container_id VARCHAR(255),
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sandbox_events_module_instance ON sandbox_events(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_sandbox_events_type ON sandbox_events(event_type);
CREATE INDEX IF NOT EXISTS idx_sandbox_events_created ON sandbox_events(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_execution_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    command TEXT NOT NULL,
    working_directory TEXT,
    environment_vars JSONB DEFAULT '{}',
    output TEXT,
    error_output TEXT,
    exit_code INTEGER,
    duration_ms INTEGER,
    triggered_by VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_exec_user ON osa_execution_history(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_app ON osa_execution_history(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_workspace ON osa_execution_history(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_created ON osa_execution_history(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_sync_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    osa_entity_id VARCHAR(255),
    osa_entity_type VARCHAR(50),
    sync_status VARCHAR(50) DEFAULT 'pending',
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    sync_direction VARCHAR(50) DEFAULT 'bidirectional',
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    local_snapshot JSONB,
    remote_snapshot JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT osa_sync_entity_unique UNIQUE(entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_osa_sync_entity ON osa_sync_status(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_osa_sync_status ON osa_sync_status(sync_status);
CREATE INDEX IF NOT EXISTS idx_osa_sync_next ON osa_sync_status(next_sync_at);

CREATE TABLE IF NOT EXISTS osa_build_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_instance_id UUID NOT NULL REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    build_id VARCHAR(255),
    phase VARCHAR(50),
    progress_percent INTEGER DEFAULT 0,
    status_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_build_app ON osa_build_events(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_build_workspace ON osa_build_events(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_build_type ON osa_build_events(event_type);
CREATE INDEX IF NOT EXISTS idx_osa_build_created ON osa_build_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_osa_build_build_id ON osa_build_events(build_id);

CREATE TABLE IF NOT EXISTS osa_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    webhook_url TEXT NOT NULL,
    secret_key VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_webhooks_workspace ON osa_webhooks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_app ON osa_webhooks(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_event ON osa_webhooks(event_type);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_enabled ON osa_webhooks(enabled);

-- =============================================================================
-- OSA WORKFLOWS & FILES (from 049_osa_workflows_files.sql)
-- =============================================================================

CREATE TABLE IF NOT EXISTS osa_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE SET NULL,
    osa_workflow_id VARCHAR(255),
    workflow_type VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    user_prompt TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress_percent INTEGER DEFAULT 0 CHECK (progress_percent >= 0 AND progress_percent <= 100),
    current_phase VARCHAR(50),
    files_generated INTEGER DEFAULT 0,
    tests_passed INTEGER DEFAULT 0,
    tests_failed INTEGER DEFAULT 0,
    build_successful BOOLEAN,
    error_message TEXT,
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,
    osa_sandbox_id VARCHAR(255),
    osa_metadata JSONB DEFAULT '{}',
    duration_seconds INTEGER,
    tokens_used INTEGER,
    estimated_cost_usd DECIMAL(10, 4),
    tags TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_osa_workflows_user ON osa_workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_workspace ON osa_workflows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_app ON osa_workflows(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_status ON osa_workflows(status);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_type ON osa_workflows(workflow_type);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_osa_id ON osa_workflows(osa_workflow_id) WHERE osa_workflow_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_workflows_created ON osa_workflows(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_tags ON osa_workflows USING GIN(tags);

CREATE TABLE IF NOT EXISTS osa_module_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES osa_workflows(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    language VARCHAR(50),
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    line_count INTEGER,
    encoding VARCHAR(20) DEFAULT 'utf-8',
    installation_status VARCHAR(50) DEFAULT 'pending',
    installation_path TEXT,
    installed_at TIMESTAMPTZ,
    conflict_reason TEXT,
    conflict_resolved_by VARCHAR(50),
    conflict_resolution JSONB,
    purpose TEXT,
    dependencies TEXT[],
    dependents TEXT[],
    current_version INTEGER DEFAULT 1,
    is_latest BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_files_workflow ON osa_module_files(workflow_id);
CREATE INDEX IF NOT EXISTS idx_osa_files_app ON osa_module_files(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_files_type ON osa_module_files(file_type);
CREATE INDEX IF NOT EXISTS idx_osa_files_language ON osa_module_files(language);
CREATE INDEX IF NOT EXISTS idx_osa_files_status ON osa_module_files(installation_status);
CREATE INDEX IF NOT EXISTS idx_osa_files_hash ON osa_module_files(content_hash);
CREATE INDEX IF NOT EXISTS idx_osa_files_path ON osa_module_files(file_path);
CREATE INDEX IF NOT EXISTS idx_osa_files_name ON osa_module_files(file_name);
CREATE INDEX IF NOT EXISTS idx_osa_files_latest ON osa_module_files(is_latest) WHERE is_latest = true;
CREATE UNIQUE INDEX IF NOT EXISTS osa_files_app_path_unique
    ON osa_module_files (module_instance_id, file_path)
    WHERE module_instance_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS osa_files_workflow_path_unique
    ON osa_module_files (workflow_id, file_path)
    WHERE workflow_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS osa_file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES osa_module_files(id) ON DELETE CASCADE,
    workflow_id UUID REFERENCES osa_workflows(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    change_type VARCHAR(50) NOT NULL,
    change_summary TEXT,
    diff_from_previous TEXT,
    lines_added INTEGER DEFAULT 0,
    lines_removed INTEGER DEFAULT 0,
    created_by_workflow_type VARCHAR(50),
    created_by_user_id VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT osa_file_versions_unique UNIQUE(file_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_osa_file_versions_file ON osa_file_versions(file_id, version_number DESC);
CREATE INDEX IF NOT EXISTS idx_osa_file_versions_workflow ON osa_file_versions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_osa_file_versions_hash ON osa_file_versions(content_hash);
CREATE INDEX IF NOT EXISTS idx_osa_file_versions_created ON osa_file_versions(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_installation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES osa_workflows(id) ON DELETE CASCADE,
    file_id UUID REFERENCES osa_module_files(id) ON DELETE SET NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    source_path TEXT NOT NULL,
    destination_path TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    backup_path TEXT,
    backup_content TEXT,
    backup_hash VARCHAR(64),
    error_message TEXT,
    error_details JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_install_workflow ON osa_installation_log(workflow_id);
CREATE INDEX IF NOT EXISTS idx_osa_install_file ON osa_installation_log(file_id);
CREATE INDEX IF NOT EXISTS idx_osa_install_user ON osa_installation_log(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_install_status ON osa_installation_log(status);
CREATE INDEX IF NOT EXISTS idx_osa_install_action ON osa_installation_log(action);
CREATE INDEX IF NOT EXISTS idx_osa_install_created ON osa_installation_log(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sync_session_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    operation VARCHAR(50) NOT NULL,
    direction VARCHAR(50) NOT NULL,
    before_snapshot JSONB,
    after_snapshot JSONB,
    changes JSONB,
    status VARCHAR(50) NOT NULL,
    conflict_type VARCHAR(50),
    conflict_resolution VARCHAR(50),
    vector_clock JSONB,
    duration_ms INTEGER,
    payload_size_bytes INTEGER,
    error_message TEXT,
    error_stack TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_sync_log_session ON osa_sync_log(sync_session_id);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_entity ON osa_sync_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_operation ON osa_sync_log(operation);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_status ON osa_sync_log(status);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_created ON osa_sync_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_conflicts ON osa_sync_log(conflict_type) WHERE conflict_type IS NOT NULL;

-- =============================================================================
-- SYNC OUTBOX TABLES (from 043_sync_outbox.sql)
-- =============================================================================

CREATE TABLE IF NOT EXISTS sync_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    vector_clock JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    scheduled_for TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sync_outbox_pending ON sync_outbox (status, scheduled_for, created_at)
WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_sync_outbox_status ON sync_outbox(status);
CREATE INDEX IF NOT EXISTS idx_sync_outbox_aggregate ON sync_outbox(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_sync_outbox_scheduled ON sync_outbox(scheduled_for)
WHERE scheduled_for IS NOT NULL AND status = 'failed';

CREATE TABLE IF NOT EXISTS sync_dlq (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    vector_clock JSONB NOT NULL,
    attempts INT NOT NULL,
    last_error TEXT NOT NULL,
    failure_reason VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    moved_to_dlq_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_dlq_aggregate ON sync_dlq(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_sync_dlq_resolved ON sync_dlq(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_dlq_moved_at ON sync_dlq(moved_to_dlq_at DESC);

-- =============================================================================
-- SIGNAL LOG
-- =============================================================================

CREATE TABLE IF NOT EXISTS signal_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    conversation_id UUID,
    mode TEXT NOT NULL DEFAULT 'ASSIST',
    genre TEXT NOT NULL DEFAULT 'INFORM',
    signal_type TEXT NOT NULL DEFAULT 'chat',
    format TEXT NOT NULL DEFAULT 'MARKDOWN',
    weight FLOAT NOT NULL DEFAULT 0.5,
    message_preview TEXT,
    response_length INT,
    latency_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signal_log_user_time ON signal_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_signal_log_genre ON signal_log(genre, created_at DESC);

COMMENT ON TABLE signal_log IS 'Async signal metadata log for Signal Theory feedback loop (homeostatic, autopoietic, Q-learning)';
COMMENT ON COLUMN signal_log.genre IS 'Detected genre: DIRECT, INFORM, COMMIT, DECIDE, EXPRESS';
COMMENT ON COLUMN signal_log.mode IS 'Focus mode at time of signal: ASSIST, BUILD, RESEARCH, etc.';
COMMENT ON COLUMN signal_log.weight IS 'Signal importance weight (0.0-1.0) for feedback prioritization';

-- =============================================================================
-- BACKGROUND JOBS
-- =============================================================================

CREATE TABLE IF NOT EXISTS background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    worker_id VARCHAR(100),
    locked_until TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    result JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_background_jobs_available
    ON background_jobs (priority DESC, scheduled_at ASC)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status);
CREATE INDEX IF NOT EXISTS idx_background_jobs_type ON background_jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_background_jobs_created ON background_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_background_jobs_locked ON background_jobs(locked_until) WHERE locked_until IS NOT NULL;

CREATE OR REPLACE FUNCTION calculate_retry_time(attempt_count INTEGER)
RETURNS TIMESTAMPTZ AS $$
BEGIN
    RETURN NOW() + (LEAST(POWER(2, GREATEST(attempt_count, 1)), 300)::TEXT || ' seconds')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION release_stuck_jobs()
RETURNS INTEGER AS $$
DECLARE
    released_count INTEGER;
BEGIN
    UPDATE background_jobs
    SET status = 'pending',
        locked_until = NULL,
        worker_id = NULL
    WHERE status = 'running'
      AND locked_until IS NOT NULL
      AND locked_until < NOW();

    GET DIAGNOSTICS released_count = ROW_COUNT;
    RETURN released_count;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    cron_expression VARCHAR(120) NOT NULL,
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    name VARCHAR(255),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_active_due
    ON scheduled_jobs(next_run_at)
    WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_type ON scheduled_jobs(job_type);

CREATE TABLE IF NOT EXISTS sorx_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    workspace_id TEXT,
    cron_expr TEXT NOT NULL,
    params JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT false,
    last_run_at TIMESTAMPTZ,
    last_status TEXT,
    last_error TEXT,
    next_run_at TIMESTAMPTZ,
    run_count INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sorx_schedules_user ON sorx_schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_sorx_schedules_enabled ON sorx_schedules(enabled) WHERE enabled = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sorx_schedules_unique
    ON sorx_schedules(skill_id, user_id)
    WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sorx_schedules_unique_ws
    ON sorx_schedules(skill_id, user_id, workspace_id)
    WHERE workspace_id IS NOT NULL;

CREATE OR REPLACE FUNCTION update_sorx_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sorx_schedules_updated_at_trigger ON sorx_schedules;
DO $$ BEGIN
CREATE TRIGGER sorx_schedules_updated_at_trigger
    BEFORE UPDATE ON sorx_schedules
    FOR EACH ROW EXECUTE FUNCTION update_sorx_schedules_updated_at();
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- =============================================================================
-- APP TEMPLATES SYSTEM (from 081_app_templates_system.sql)
-- =============================================================================

CREATE TABLE IF NOT EXISTS app_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    icon_type VARCHAR(50),
    target_business_types TEXT[],
    target_challenges TEXT[],
    target_team_sizes TEXT[],
    priority_score INT DEFAULT 50,
    template_config JSONB DEFAULT '{}'::jsonb,
    required_modules TEXT[] DEFAULT '{}',
    optional_features TEXT[] DEFAULT '{}',
    generation_prompt TEXT,
    scaffold_type VARCHAR(50) DEFAULT 'svelte',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_templates_business_types ON app_templates USING GIN(target_business_types);
CREATE INDEX IF NOT EXISTS idx_app_templates_challenges ON app_templates USING GIN(target_challenges);
CREATE INDEX IF NOT EXISTS idx_app_templates_team_sizes ON app_templates USING GIN(target_team_sizes);
CREATE INDEX IF NOT EXISTS idx_app_templates_category ON app_templates(category);
CREATE INDEX IF NOT EXISTS idx_app_templates_priority ON app_templates(priority_score DESC);

CREATE TABLE IF NOT EXISTS user_generated_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id UUID REFERENCES app_templates(id) ON DELETE SET NULL,
    app_name VARCHAR(200) NOT NULL,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE SET NULL,
    is_visible BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_favorite BOOLEAN DEFAULT FALSE,
    position_index INT,
    custom_config JSONB DEFAULT '{}'::jsonb,
    custom_icon VARCHAR(50),
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    access_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, app_name)
);

CREATE INDEX IF NOT EXISTS idx_user_generated_apps_workspace ON user_generated_apps(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_generated_apps_template ON user_generated_apps(template_id);
CREATE INDEX IF NOT EXISTS idx_user_generated_apps_module_instance ON user_generated_apps(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_user_generated_apps_visibility ON user_generated_apps(workspace_id, is_visible) WHERE is_visible = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_generated_apps_position ON user_generated_apps(workspace_id, position_index);

CREATE TABLE IF NOT EXISTS app_generation_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id UUID REFERENCES app_templates(id) ON DELETE SET NULL, -- OPTIONAL for AI generative mode
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    priority INT DEFAULT 8,
    generation_context JSONB DEFAULT '{}'::jsonb,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_app_generation_queue_status ON app_generation_queue(status, priority DESC);
CREATE INDEX IF NOT EXISTS idx_app_generation_queue_workspace ON app_generation_queue(workspace_id);
CREATE INDEX IF NOT EXISTS idx_app_generation_queue_pending ON app_generation_queue(workspace_id, status) WHERE status = 'pending';

-- App Versions (Migration 087)
CREATE TABLE IF NOT EXISTS app_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES user_generated_apps(id) ON DELETE CASCADE,
    version_number VARCHAR(20) NOT NULL,
    snapshot_data JSONB NOT NULL,
    snapshot_metadata JSONB DEFAULT '{}'::jsonb,
    change_summary TEXT,
    created_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_app_versions_app_id ON app_versions(app_id);
CREATE INDEX IF NOT EXISTS idx_app_versions_created_at ON app_versions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_versions_created_by ON app_versions(created_by) WHERE created_by IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_app_versions_app_time ON app_versions(app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_versions_snapshot_data ON app_versions USING GIN(snapshot_data);

-- ===== ONBOARDING SYSTEM =====

-- Onboarding sessions table
CREATE TABLE IF NOT EXISTS onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress', -- in_progress, completed, abandoned, expired
    current_step VARCHAR(100) NOT NULL DEFAULT 'company_name',
    steps_completed TEXT[] DEFAULT '{}',
    extracted_data JSONB DEFAULT '{}',
    low_confidence_count INT DEFAULT 0,
    fallback_triggered BOOLEAN DEFAULT FALSE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    analysis_completed BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_user_id ON onboarding_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_status ON onboarding_sessions(status);
CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_expires_at ON onboarding_sessions(expires_at);

-- Onboarding conversation history
CREATE TABLE IF NOT EXISTS onboarding_conversation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- user, agent, system
    content TEXT NOT NULL,
    confidence_score DECIMAL(3,2),
    extracted_fields JSONB,
    question_type VARCHAR(100),
    sequence_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_conversation_session_id ON onboarding_conversation_history(session_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_conversation_sequence ON onboarding_conversation_history(session_id, sequence_number);

-- Onboarding email metadata (per-email extracted data)
CREATE TABLE IF NOT EXISTS onboarding_email_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    email_id VARCHAR(255) NOT NULL, -- Gmail message ID
    sender_domain VARCHAR(255),
    subject_keywords TEXT[], -- Top keywords from subject
    body_keywords TEXT[], -- Top keywords from body
    detected_tools JSONB DEFAULT '{}', -- {"Slack": 5, "Notion": 3}
    topics JSONB DEFAULT '{}', -- {"collaboration": 10, "development": 5}
    sentiment VARCHAR(50), -- positive, negative, neutral
    importance_score DECIMAL(3,2), -- 0.00 - 1.00
    category VARCHAR(100), -- work, personal, marketing, etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, email_id)
);

CREATE INDEX IF NOT EXISTS idx_onboarding_email_metadata_session_id ON onboarding_email_metadata(session_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_email_metadata_email_id ON onboarding_email_metadata(email_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_email_metadata_sender_domain ON onboarding_email_metadata(sender_domain);

-- ═══════════════════════════════════════════════════════════════════════════════
-- Custom Modules System (Migration 088)
-- ═══════════════════════════════════════════════════════════════════════════════

-- Custom Modules
CREATE TABLE IF NOT EXISTS custom_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    version VARCHAR(50) DEFAULT '0.0.1',
    manifest JSONB NOT NULL DEFAULT '{}',
    config JSONB DEFAULT '{}',
    icon VARCHAR(100),
    tags TEXT[],
    keywords TEXT[],
    is_public BOOLEAN DEFAULT FALSE,
    is_published BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    install_count INTEGER DEFAULT 0,
    star_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(workspace_id, slug),
    CONSTRAINT valid_slug CHECK (slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT valid_version CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

CREATE INDEX IF NOT EXISTS idx_custom_modules_workspace ON custom_modules(workspace_id);
CREATE INDEX IF NOT EXISTS idx_custom_modules_created_by ON custom_modules(created_by);
CREATE INDEX IF NOT EXISTS idx_custom_modules_slug ON custom_modules(slug);
CREATE INDEX IF NOT EXISTS idx_custom_modules_category ON custom_modules(category);
CREATE INDEX IF NOT EXISTS idx_custom_modules_public ON custom_modules(is_public) WHERE is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_custom_modules_published ON custom_modules(is_published) WHERE is_published = TRUE;
CREATE INDEX IF NOT EXISTS idx_custom_modules_tags ON custom_modules USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_custom_modules_keywords ON custom_modules USING GIN(keywords);
CREATE INDEX IF NOT EXISTS idx_custom_modules_manifest ON custom_modules USING GIN(manifest);

-- Custom Module Versions
CREATE TABLE IF NOT EXISTS custom_module_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    changelog TEXT,
    manifest_snapshot JSONB NOT NULL,
    config_snapshot JSONB,
    created_by VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_stable BOOLEAN DEFAULT TRUE,
    is_breaking BOOLEAN DEFAULT FALSE,
    UNIQUE(module_id, version),
    CONSTRAINT valid_version CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

CREATE INDEX IF NOT EXISTS idx_module_versions_module ON custom_module_versions(module_id);
CREATE INDEX IF NOT EXISTS idx_module_versions_created_at ON custom_module_versions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_module_versions_stable ON custom_module_versions(module_id, is_stable) WHERE is_stable = TRUE;

-- Module Installations
CREATE TABLE IF NOT EXISTS module_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    installed_by VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE SET NULL,
    installed_version VARCHAR(50) NOT NULL,
    config_override JSONB DEFAULT '{}',
    is_enabled BOOLEAN DEFAULT TRUE,
    is_auto_update BOOLEAN DEFAULT TRUE,
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(module_id, workspace_id),
    CONSTRAINT valid_version CHECK (installed_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

CREATE INDEX IF NOT EXISTS idx_module_installations_module ON module_installations(module_id);
CREATE INDEX IF NOT EXISTS idx_module_installations_workspace ON module_installations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_module_installations_enabled ON module_installations(workspace_id, is_enabled) WHERE is_enabled = TRUE;

-- Module Shares
CREATE TABLE IF NOT EXISTS module_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES custom_modules(id) ON DELETE CASCADE,
    shared_with_user_id VARCHAR(255) REFERENCES "user"(id) ON DELETE CASCADE,
    shared_with_workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    shared_with_email VARCHAR(255),
    can_view BOOLEAN DEFAULT TRUE,
    can_install BOOLEAN DEFAULT TRUE,
    can_modify BOOLEAN DEFAULT FALSE,
    can_reshare BOOLEAN DEFAULT FALSE,
    shared_by VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    shared_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT one_target_only CHECK (
        (shared_with_user_id IS NOT NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NOT NULL AND shared_with_email IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_workspace_id IS NULL AND shared_with_email IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_module_shares_module ON module_shares(module_id);
CREATE INDEX IF NOT EXISTS idx_module_shares_user ON module_shares(shared_with_user_id) WHERE shared_with_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_module_shares_workspace ON module_shares(shared_with_workspace_id) WHERE shared_with_workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_module_shares_email ON module_shares(shared_with_email) WHERE shared_with_email IS NOT NULL;

-- =============================================================================
-- DROPPED in migration 099: osa_apps
-- =============================================================================

-- CREATE TABLE IF NOT EXISTS osa_apps (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     workspace_id UUID NOT NULL REFERENCES osa_workspaces(id) ON DELETE CASCADE,
--     name VARCHAR(255) NOT NULL,
--     description TEXT,
--     template_type TEXT NOT NULL DEFAULT 'custom',
--     status TEXT NOT NULL DEFAULT 'generating',
--     generation_context JSONB DEFAULT '{}',
--     deployment_config JSONB DEFAULT '{}',
--     app_metadata JSONB DEFAULT '{}',
--     created_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
--     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     CONSTRAINT osa_apps_name_workspace_unique UNIQUE(name, workspace_id)
-- );
--
-- CREATE INDEX idx_osa_apps_workspace ON osa_apps(workspace_id);
-- CREATE INDEX idx_osa_apps_created_by ON osa_apps(created_by);
-- CREATE INDEX idx_osa_apps_status ON osa_apps(status);
-- CREATE INDEX idx_osa_apps_template_type ON osa_apps(template_type);
-- CREATE INDEX idx_osa_apps_generation_context ON osa_apps USING GIN(generation_context);
-- CREATE INDEX idx_osa_apps_deployment_config ON osa_apps USING GIN(deployment_config);
-- CREATE INDEX idx_osa_apps_app_metadata ON osa_apps USING GIN(app_metadata);

-- =============================================================================
-- DROPPED in migration 099: osa_deployments
-- =============================================================================

-- CREATE TABLE IF NOT EXISTS osa_deployments (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     app_id UUID NOT NULL REFERENCES osa_apps(id) ON DELETE CASCADE,
--     version VARCHAR(50) NOT NULL,
--     commit_sha VARCHAR(255),
--     deployment_config JSONB DEFAULT '{}',
--     metadata JSONB DEFAULT '{}',
--     deployed_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
--     deployed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     CONSTRAINT osa_deployments_app_version_unique UNIQUE(app_id, version)
-- );
--
-- CREATE INDEX idx_osa_deployments_app ON osa_deployments(app_id);
-- CREATE INDEX idx_osa_deployments_deployed_by ON osa_deployments(deployed_by);
-- CREATE INDEX idx_osa_deployments_deployed_at ON osa_deployments(deployed_at DESC);
-- CREATE INDEX idx_osa_deployments_config ON osa_deployments USING GIN(deployment_config);
-- CREATE INDEX idx_osa_deployments_metadata ON osa_deployments USING GIN(metadata);

-- =============================================================================
-- OSA PROMPT TEMPLATES (from migration 078)
-- =============================================================================

CREATE TABLE IF NOT EXISTS osa_prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('system', 'workspace', 'user')),
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) REFERENCES "user"(id) ON DELETE CASCADE,
    template_content TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '{"variables": [], "required": []}',
    category VARCHAR(100),
    tags TEXT[],
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    is_active BOOLEAN DEFAULT true,
    parent_template_id UUID REFERENCES osa_prompt_templates(id) ON DELETE SET NULL,
    usage_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE
            WHEN usage_count = 0 THEN NULL
            ELSE ROUND((success_count::DECIMAL / usage_count::DECIMAL) * 100, 2)
        END
    ) STORED,
    avg_render_time_ms INTEGER,
    avg_generation_time_sec INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    CONSTRAINT osa_prompt_scope_check CHECK (
        (scope = 'system' AND workspace_id IS NULL AND user_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_osa_prompt_name_scope_unique ON osa_prompt_templates(
    name,
    scope,
    COALESCE(workspace_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(user_id, 'system')
);
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_name ON osa_prompt_templates(name);
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_scope ON osa_prompt_templates(scope);
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_workspace ON osa_prompt_templates(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_user ON osa_prompt_templates(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_category ON osa_prompt_templates(category) WHERE category IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_active ON osa_prompt_templates(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_tags ON osa_prompt_templates USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_parent ON osa_prompt_templates(parent_template_id) WHERE parent_template_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_osa_prompt_templates_last_used ON osa_prompt_templates(last_used_at DESC) WHERE last_used_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS osa_template_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES osa_prompt_templates(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES osa_workflows(id) ON DELETE SET NULL,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE SET NULL,
    variables_used JSONB NOT NULL,
    render_time_ms INTEGER,
    generation_time_sec INTEGER,
    tokens_used INTEGER,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    error_details JSONB,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_feedback TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_osa_template_usage_template ON osa_template_usage_log(template_id);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_user ON osa_template_usage_log(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_workspace ON osa_template_usage_log(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_workflow ON osa_template_usage_log(workflow_id);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_status ON osa_template_usage_log(status);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_created ON osa_template_usage_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_osa_template_usage_rating ON osa_template_usage_log(user_rating) WHERE user_rating IS NOT NULL;

-- =============================================================================
-- MCP SERVERS - External Model Context Protocol server connections
-- =============================================================================
CREATE TABLE IF NOT EXISTS mcp_servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    description     TEXT DEFAULT '',
    server_url      TEXT NOT NULL,
    transport       VARCHAR(20) NOT NULL DEFAULT 'sse',
    auth_type       VARCHAR(20) NOT NULL DEFAULT 'none',
    auth_token_enc  TEXT,
    custom_headers  JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    tools_cache     JSONB NOT NULL DEFAULT '[]',
    status          VARCHAR(20) NOT NULL DEFAULT 'disconnected',
    last_error      TEXT,
    last_connected  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_user ON mcp_servers(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_enabled ON mcp_servers(user_id, enabled);

-- =============================================================================
-- OPTIMAL ENGINE — Signal classification, knowledge graph, search index,
--                  learning loop (migration 105)
-- All tables scoped by workspace_id for multi-tenant isolation.
-- Cloud users: PostgreSQL. Local users: SQLite + filesystem.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- OPTIMAL SIGNALS — Classified markdown files with S=(M,G,T,F,W)
CREATE TABLE IF NOT EXISTS optimal_signals (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    node_id         UUID         REFERENCES nodes(id) ON DELETE SET NULL,
    created_by      VARCHAR(255) NOT NULL REFERENCES "user"(id),

    title           VARCHAR(500) NOT NULL DEFAULT '',
    slug            VARCHAR(255) NOT NULL DEFAULT '',
    uri             TEXT         NOT NULL DEFAULT '',
    path            TEXT         NOT NULL DEFAULT '',
    content         TEXT         NOT NULL DEFAULT '',

    mode            VARCHAR(50)  NOT NULL DEFAULT 'linguistic',
    genre           VARCHAR(100) NOT NULL DEFAULT 'note',
    signal_type     VARCHAR(50)  NOT NULL DEFAULT 'inform',
    format          VARCHAR(50)  NOT NULL DEFAULT 'markdown',
    structure       VARCHAR(200) NOT NULL DEFAULT '',

    sn_ratio        REAL         NOT NULL DEFAULT 0.5,

    l0_abstract     TEXT         NOT NULL DEFAULT '',
    l1_overview     TEXT         NOT NULL DEFAULT '',

    primary_node    VARCHAR(255) NOT NULL DEFAULT '',
    cross_ref_nodes JSONB        NOT NULL DEFAULT '[]',
    entities        JSONB        NOT NULL DEFAULT '[]',

    valid_from      TIMESTAMPTZ  DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,
    supersedes      UUID         REFERENCES optimal_signals(id) ON DELETE SET NULL,

    metadata        JSONB        NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    indexed_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE optimal_signals
    ADD COLUMN IF NOT EXISTS search_vector tsvector
        GENERATED ALWAYS AS (
            setweight(to_tsvector('english', COALESCE(title,       '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(l0_abstract, '')), 'B') ||
            setweight(to_tsvector('english', COALESCE(l1_overview, '')), 'C') ||
            setweight(to_tsvector('english', COALESCE(content,     '')), 'D')
        ) STORED;

CREATE INDEX IF NOT EXISTS idx_optimal_signals_workspace   ON optimal_signals(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_node        ON optimal_signals(node_id);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_genre       ON optimal_signals(genre);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_created     ON optimal_signals(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_mode        ON optimal_signals(mode);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_slug        ON optimal_signals(workspace_id, slug);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_valid       ON optimal_signals(workspace_id, valid_from DESC) WHERE valid_until IS NULL;
CREATE INDEX IF NOT EXISTS idx_optimal_signals_fts         ON optimal_signals USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_title_trgm  ON optimal_signals USING GIN(title gin_trgm_ops);

-- OPTIMAL ENTITIES — People, orgs, tools, concepts extracted from signals
CREATE TABLE IF NOT EXISTS optimal_entities (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    name            VARCHAR(500) NOT NULL,
    canonical_name  VARCHAR(500) NOT NULL DEFAULT '',
    type            VARCHAR(100) NOT NULL DEFAULT 'person',

    properties      JSONB        NOT NULL DEFAULT '{}',
    mention_count   INTEGER      NOT NULL DEFAULT 1,

    first_seen      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id, canonical_name)
);

CREATE INDEX IF NOT EXISTS idx_optimal_entities_workspace  ON optimal_entities(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_entities_name       ON optimal_entities(name);
CREATE INDEX IF NOT EXISTS idx_optimal_entities_type       ON optimal_entities(workspace_id, type);

-- OPTIMAL EDGES — Knowledge graph relationships
CREATE TABLE IF NOT EXISTS optimal_edges (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    source_id       UUID         NOT NULL,
    source_type     VARCHAR(50)  NOT NULL DEFAULT 'entity',
    target_id       UUID         NOT NULL,
    target_type     VARCHAR(50)  NOT NULL DEFAULT 'entity',

    relation        VARCHAR(100) NOT NULL DEFAULT 'related',
    weight          REAL         NOT NULL DEFAULT 1.0,

    valid_from      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,

    reason          TEXT,
    metadata        JSONB        NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id, source_id, target_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_optimal_edges_workspace  ON optimal_edges(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_source     ON optimal_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_target     ON optimal_edges(target_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_relation   ON optimal_edges(workspace_id, relation);

-- OPTIMAL OBSERVATIONS — Learning loop (SICA friction capture)
CREATE TABLE IF NOT EXISTS optimal_observations (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by      VARCHAR(255) NOT NULL REFERENCES "user"(id),

    category        VARCHAR(100) NOT NULL DEFAULT 'pattern',
    content         TEXT         NOT NULL,
    confidence      REAL         NOT NULL DEFAULT 0.5,
    source          VARCHAR(100) NOT NULL DEFAULT 'explicit',

    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_observations_workspace  ON optimal_observations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_observations_category   ON optimal_observations(workspace_id, category);
CREATE INDEX IF NOT EXISTS idx_optimal_observations_created    ON optimal_observations(created_at DESC);

-- OPTIMAL TOPOLOGY — Per-workspace routing config (one row per workspace)
CREATE TABLE IF NOT EXISTS optimal_topology (
    id               UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID  NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    config           JSONB NOT NULL DEFAULT '{}',
    genre_skeletons  JSONB NOT NULL DEFAULT '{}',
    genre_half_lives JSONB NOT NULL DEFAULT '{}',

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_optimal_topology_workspace ON optimal_topology(workspace_id);

-- OPTIMAL SESSIONS — Conversation sessions with signal extraction
CREATE TABLE IF NOT EXISTS optimal_sessions (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id         UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by           VARCHAR(255) NOT NULL REFERENCES "user"(id),

    status               VARCHAR(50)  NOT NULL DEFAULT 'active',
    summary              TEXT         NOT NULL DEFAULT '',
    extracted_signal_ids JSONB        NOT NULL DEFAULT '[]',
    message_count        INTEGER      NOT NULL DEFAULT 0,

    started_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    committed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_sessions_workspace  ON optimal_sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_sessions_status     ON optimal_sessions(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_optimal_sessions_started    ON optimal_sessions(started_at DESC);

-- OPTIMAL DECISION LOG — Append-only audit trail (never UPDATE or DELETE)
CREATE TABLE IF NOT EXISTS optimal_decision_log (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    actor           VARCHAR(255) NOT NULL,
    action          VARCHAR(100) NOT NULL,
    target_type     VARCHAR(100) NOT NULL,
    target_id       UUID         NOT NULL,

    what_changed    TEXT         NOT NULL,
    why             TEXT,
    alternatives_considered JSONB,
    context         JSONB,

    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_workspace  ON optimal_decision_log(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_target     ON optimal_decision_log(target_id);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_created    ON optimal_decision_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_actor      ON optimal_decision_log(workspace_id, actor);

-- ============================================================================
-- MICROSOFT TEAMS (Wave 2 — see migration 107_microsoft_teams.sql)
-- ============================================================================
-- Provider-specific tables for Teams data, mirroring microsoft_mail_messages
-- shape (line 1908) per Leon's spec decision (docs/design/comms-channels-spec.md
-- §1.3). The unified channels endpoint reads slack_channels +
-- microsoft_teams_channels and normalizes server-side.

CREATE TABLE IF NOT EXISTS microsoft_teams_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,

    team_name VARCHAR(255),

    display_name VARCHAR(255),
    description TEXT,
    email VARCHAR(255),
    web_url TEXT,
    membership_type VARCHAR(50),
    is_archived BOOLEAN DEFAULT FALSE,
    is_favorite_by_default BOOLEAN DEFAULT FALSE,

    member_count INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,

    created_datetime TIMESTAMPTZ,

    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, team_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_user
    ON microsoft_teams_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_team
    ON microsoft_teams_channels(user_id, team_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_channels_activity
    ON microsoft_teams_channels(user_id, last_message_at DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS microsoft_teams_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,

    reply_to_id VARCHAR(255),

    message_type VARCHAR(50),
    importance VARCHAR(50),
    subject TEXT,
    summary TEXT,

    body_content TEXT,
    body_content_type VARCHAR(50),

    sender_id VARCHAR(255),
    sender_name VARCHAR(255),
    sender_type VARCHAR(50),

    attachments JSONB DEFAULT '[]',
    mentions JSONB DEFAULT '[]',
    reactions JSONB DEFAULT '[]',

    web_url TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_edited BOOLEAN DEFAULT FALSE,

    created_datetime TIMESTAMPTZ,
    last_modified_datetime TIMESTAMPTZ,
    last_edited_datetime TIMESTAMPTZ,
    deleted_datetime TIMESTAMPTZ,

    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, team_id, channel_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_user
    ON microsoft_teams_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_channel
    ON microsoft_teams_messages(user_id, team_id, channel_id);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_recent
    ON microsoft_teams_messages(user_id, team_id, channel_id, created_datetime DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_thread
    ON microsoft_teams_messages(user_id, reply_to_id) WHERE reply_to_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ms_teams_messages_sender
    ON microsoft_teams_messages(user_id, sender_id);

-- ============================================================================
-- MICROSOFT GRAPH SUBSCRIPTIONS (Wave 3 — see migration 108_microsoft_subscriptions.sql)
-- ============================================================================
-- Active Graph subscriptions per user, used by the renewal scheduler to
-- refresh subscriptions before they expire (Graph caps message subs at
-- ~71h). Webhook receiver verifies clientState against this table.

CREATE TABLE IF NOT EXISTS microsoft_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    subscription_id VARCHAR(255) NOT NULL,
    resource VARCHAR(500) NOT NULL,
    resource_kind VARCHAR(50) NOT NULL,
    change_type VARCHAR(100),

    notification_url TEXT,
    client_state VARCHAR(255),

    expires_at TIMESTAMPTZ,
    last_renewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, subscription_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_subs_user ON microsoft_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_subs_expiring
    ON microsoft_subscriptions(expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ms_subs_subscription
    ON microsoft_subscriptions(subscription_id);

-- ============================================================================
-- GOOGLE GMAIL WATCHES (Wave 3 — see migration 109_gmail_watches.sql)
-- ============================================================================
-- Per-user Gmail Pub/Sub watch state. history_id pins the baseline for
-- incremental fetches; renewal scheduler refreshes watches every 12h
-- before Google's 7-day expiry.

CREATE TABLE IF NOT EXISTS google_gmail_watches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,

    topic_name VARCHAR(500) NOT NULL,

    history_id BIGINT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    last_renewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmail_watches_expiring
    ON google_gmail_watches(expires_at);

-- ============================================================================
-- Workspace RBAC: role_permissions + seed_default_workspace_roles()
-- (restored — required by workspace_service.CreateWorkspace; was missing from
--  the consolidated schema, breaking workspace creation. Source: 041_workspaces_and_roles.sql)
-- ============================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    permission VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, role, resource, permission)
);

CREATE OR REPLACE FUNCTION seed_default_workspace_roles(p_workspace_id UUID)
RETURNS void AS $$
BEGIN
    -- 1. OWNER (Full access)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'owner',
        'Owner',
        'Full workspace access including billing and deletion',
        '#8B5CF6',
        'crown',
        true,
        false,
        1,
        '{"projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": true, "share": true}, "workspace": {"invite_members": true, "manage_roles": true, "manage_billing": true, "delete_workspace": true}, "agents": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
    );

    -- 2. ADMIN (Almost full access except ownership transfer)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'admin',
        'Admin',
        'Full access except billing and workspace deletion',
        '#3B82F6',
        'shield',
        true,
        false,
        2,
        '{"projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": true, "share": true}, "workspace": {"invite_members": true, "manage_roles": true, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
    );

    -- 3. MANAGER (Can manage projects and team)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'manager',
        'Manager',
        'Can manage projects, tasks, and team members',
        '#10B981',
        'users',
        true,
        false,
        3,
        '{"projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": false, "share": true}, "workspace": {"invite_members": true, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
    );

    -- 4. MEMBER (Standard access - create and edit own work)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'member',
        'Member',
        'Standard team member - can create and manage own work',
        '#6366F1',
        'user',
        true,
        true,  -- Default role for new members
        4,
        '{"projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": false}, "tasks": {"create": true, "read": true, "update": true, "delete": false, "assign": false}, "contexts": {"create": true, "read": true, "update": true, "delete": false, "share": true}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}}'::jsonb
    );

    -- 5. VIEWER (Read-only access)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'viewer',
        'Viewer',
        'Read-only access to workspace resources',
        '#64748B',
        'eye',
        true,
        false,
        5,
        '{"projects": {"create": false, "read": true, "update": false, "delete": false, "manage_members": false}, "tasks": {"create": false, "read": true, "update": false, "delete": false, "assign": false}, "contexts": {"create": false, "read": true, "update": false, "delete": false, "share": false}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}}'::jsonb
    );

    -- 6. GUEST (Limited read access to specific projects only)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'guest',
        'Guest',
        'Limited access to specific projects only',
        '#94A3B8',
        'user-plus',
        true,
        false,
        6,
        '{"projects": {"create": false, "read": true, "update": false, "delete": false, "manage_members": false}, "tasks": {"create": false, "read": true, "update": false, "delete": false, "assign": false}, "contexts": {"create": false, "read": true, "update": false, "delete": false, "share": false}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": false, "create_custom_agents": false, "access_workspace_memory": false, "modify_workspace_memory": false}}'::jsonb
    );

    -- Populate role_permissions table from the JSONB permissions
    -- This is for compatibility with role_context.go
    INSERT INTO role_permissions (workspace_id, role, resource, permission, metadata)
    SELECT
        workspace_id,
        name as role,
        perm.key as resource,
        perm_detail.key as permission,
        jsonb_build_object('allowed', perm_detail.value) as metadata
    FROM workspace_roles wr,
         jsonb_each(wr.permissions) as perm,
         jsonb_each(perm.value) as perm_detail
    WHERE wr.workspace_id = p_workspace_id
      AND perm_detail.value::text = 'true';

END;
$$ LANGUAGE plpgsql;

-- === BusinessOS schema-drift reconciliation (schema.sql was missing these vs Go code) ===
-- workspace_members: members/profile endpoints select wm.custom_permissions
ALTER TABLE workspace_members ADD COLUMN IF NOT EXISTS custom_permissions JSONB;
-- team_members: team endpoint filters/updates tm.workspace_id
ALTER TABLE team_members ADD COLUMN IF NOT EXISTS workspace_id UUID;
CREATE INDEX IF NOT EXISTS idx_team_members_workspace ON team_members(workspace_id);
-- custom_modules: modules list selects share_scope (from 107_custom_module_sharing.sql)
ALTER TABLE custom_modules ADD COLUMN IF NOT EXISTS share_scope VARCHAR(20) NOT NULL DEFAULT 'workspace';
-- boards: composition layer, sidebar Boards section (from 126_boards.sql)
CREATE TABLE IF NOT EXISTS boards (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by   VARCHAR(255) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    kind         VARCHAR(50)  NOT NULL DEFAULT 'custom' CHECK (kind IN ('client', 'custom')),
    subject_type VARCHAR(50),
    subject_id   UUID,
    layout       JSONB NOT NULL DEFAULT '[]',
    is_pinned    BOOLEAN DEFAULT FALSE,
    position     INT DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_boards_workspace ON boards(workspace_id);
CREATE INDEX IF NOT EXISTS idx_boards_workspace_pinned ON boards(workspace_id, is_pinned) WHERE is_pinned;

-- === schema-drift reconciliation part 2: rhythm_entries (114) + workspace_preferences (120) ===
-- 114_rhythm_entries.sql
-- The Rhythm module: a per-workspace operating cadence — daily/weekly/monthly
-- entries (focuses, blockers, priorities, notes) so a team's rhythm is captured
-- and decodable by humans AND AI agents. Workspace-scoped.

CREATE TABLE IF NOT EXISTS rhythm_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    period VARCHAR(20) NOT NULL DEFAULT 'daily',   -- daily | weekly | monthly
    kind VARCHAR(20) NOT NULL DEFAULT 'focus',     -- focus | blocker | priority | note
    content TEXT NOT NULL DEFAULT '',
    entry_date DATE,
    position INT DEFAULT 0,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmail_watches_expiring
    ON google_gmail_watches(expires_at);
CREATE INDEX IF NOT EXISTS idx_rhythm_entries_workspace ON rhythm_entries(workspace_id, period);

-- Workspace preferences: the business-level settings primitive.
-- These are fundamental settings every module reads (time zone, date/time
-- format, week start, working hours), NOT calendar-specific. Calendar-specific
-- config (default view, notifications, member colors) lives in the `calendar`
-- JSONB so the calendar module can configure a feature without a new table.
CREATE TABLE IF NOT EXISTS workspace_preferences (
    workspace_id          UUID PRIMARY KEY,
    timezone              TEXT     NOT NULL DEFAULT 'America/New_York',
    date_format           TEXT     NOT NULL DEFAULT 'MM/DD/YYYY',
    time_format           TEXT     NOT NULL DEFAULT '12h',   -- 12h | 24h
    week_start            SMALLINT NOT NULL DEFAULT 0,        -- 0=Sunday, 1=Monday
    working_hours_start   SMALLINT NOT NULL DEFAULT 9,
    working_hours_end     SMALLINT NOT NULL DEFAULT 17,
    default_event_minutes SMALLINT NOT NULL DEFAULT 60,
    language              TEXT     NOT NULL DEFAULT 'en-US',
    calendar              JSONB    NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- NOTE: the "module fields (124)" block that ALTERs offers/campaigns/personas/
-- rhythm_entries/sites was moved to the END of this file. Those tables are
-- CREATEd by the canonical-migrations block below, so their ADD COLUMNs must
-- run after it — otherwise a fresh build errors ("relation ... does not exist")
-- and the columns are silently dropped.

-- ============================================================================
-- CANONICAL MIGRATIONS 109-140 (schema.sql had fallen behind; these add the
-- module tables: glossary, rhythm, campaigns, offers, personas, content,
-- sites, boards, knowledge_documents, workspace_apps/app_catalog, storage, etc.
-- All additive / IF NOT EXISTS / ON CONFLICT. Reconciled 2026-07-10.)
-- ============================================================================

-- ---- 109_entity_links_relations.sql ----
-- 109_entity_links_relations.sql
-- Make the generic relations layer (entity_links) workspace-aware so any module
-- can relate its core primitive to any other module's primitive by ID, scoped to
-- a workspace and shareable with the team. NULL workspace_id = personal/local link.
--
-- This is the one primitive that lets modules with DIFFERENT core primitives
-- (client / project / event / deal ...) compose by reference instead of by
-- duplicating data. Each module still owns its own table; this only stores pointers.

ALTER TABLE entity_links ADD COLUMN IF NOT EXISTS workspace_id UUID;

-- Lookups: "all links in this workspace touching entity X" (source or target).
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_source
    ON entity_links(workspace_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_target
    ON entity_links(workspace_id, target_type, target_id);

-- ---- 110_calendar_events_workspace.sql ----
-- 110_calendar_events_workspace.sql
-- Make calendar events a workspace-scopeable primitive. Events are owned by a
-- user (user_id) and optionally shared with a workspace (workspace_id). NULL
-- workspace_id = personal event. Mirrors the contexts/entity_links model so the
-- calendar works standalone (native events) and can be shared with the team,
-- with Google/Outlook as sync sources into the same table.

ALTER TABLE calendar_events ADD COLUMN IF NOT EXISTS workspace_id UUID;

CREATE INDEX IF NOT EXISTS idx_calendar_events_workspace
    ON calendar_events(workspace_id, start_time, end_time);

-- ---- 111_project_members_team_member_id.sql ----
-- 111_project_members_team_member_id.sql
-- Align project_members with the canonical schema. The ListTeamMembers query
-- (team_members.sql) counts active projects via
--   JOIN project_members pm ... WHERE pm.team_member_id = tm.id
-- but some DBs were built from a migration set where project_members only had
-- user_id, so the column was missing and the Team module 500'd
-- (ERROR: column pm.team_member_id does not exist).

ALTER TABLE project_members ADD COLUMN IF NOT EXISTS team_member_id UUID;
CREATE INDEX IF NOT EXISTS idx_project_members_team_member ON project_members(team_member_id);

-- ---- 112_workspace_members_role_column.sql ----
-- 112_workspace_members_role_column.sql
-- The Go code consistently uses a `role` column on workspace_members (signup
-- provisioning, GetUserRole, invites, admin role changes - ~13 queries), but
-- some DBs were built from a migration set that only has `role_name`/`role_id`.
-- The missing column made `provisionDefaultWorkspace` fail silently (background
-- goroutine), so NO workspace was ever created on signup, and `GetUserRole`
-- ("SELECT role ...") errored - breaking every workspace-scoped role check.
--
-- Add `role` and keep it in sync with the existing role_name so both work.

ALTER TABLE workspace_members ADD COLUMN IF NOT EXISTS role VARCHAR(100) DEFAULT 'member';

-- Backfill from role_name where present so existing rows keep their role.
UPDATE workspace_members
SET role = COALESCE(NULLIF(role_name, ''), role, 'member')
WHERE role IS NULL OR role = 'member';

-- ---- 113_glossary_terms.sql ----
-- 113_glossary_terms.sql
-- The Glossary module: a per-workspace dictionary of the business's own terms
-- (e.g. what "proposal" means for us), so humans AND AI agents decode language
-- consistently. Workspace-scoped; one canonical term per workspace.

CREATE TABLE IF NOT EXISTS glossary_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    term VARCHAR(300) NOT NULL,
    definition TEXT NOT NULL DEFAULT '',
    category VARCHAR(120),
    aliases TEXT,                       -- comma-separated alt names / variants
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, term)
);

CREATE INDEX IF NOT EXISTS idx_glossary_terms_workspace ON glossary_terms(workspace_id);
CREATE INDEX IF NOT EXISTS idx_glossary_terms_term ON glossary_terms(workspace_id, term);

-- ---- 115_campaigns.sql ----
-- 115_campaigns.sql
-- The Campaigns module: a per-workspace registry of marketing/outreach campaigns
-- (email, ads, sms, organic) with lifecycle status. Workspace-scoped.

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    channel VARCHAR(40) DEFAULT 'email',     -- email|ads|sms|organic|other
    status VARCHAR(40) DEFAULT 'draft',      -- draft|active|paused|done
    description TEXT DEFAULT '',
    start_date DATE,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_workspace ON campaigns(workspace_id);

-- ---- 116_offers.sql ----
-- 116_offers.sql
-- The Offers module: a per-workspace catalog of the business's productized offers
-- (name, price, status, what's included), so the team and AI agents share one
-- source of truth for what we sell. Workspace-scoped.

CREATE TABLE IF NOT EXISTS offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    price VARCHAR(120) DEFAULT '',
    status VARCHAR(40) DEFAULT 'active',   -- active | draft | archived
    description TEXT DEFAULT '',
    includes TEXT DEFAULT '',              -- what's included (free text / list)
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_offers_workspace ON offers(workspace_id);

-- ---- 117_personas.sql ----
-- 117_personas.sql
-- The Personas module: per-workspace buyer/customer personas (ideal customer
-- profiles) so humans AND AI agents target messaging and offers consistently.
-- Workspace-scoped; one row per persona.

CREATE TABLE IF NOT EXISTS personas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    segment VARCHAR(200) DEFAULT '',
    pains TEXT DEFAULT '',
    objections TEXT DEFAULT '',
    language TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_personas_workspace ON personas(workspace_id);

-- ---- 118_content_items.sql ----
-- 118_content_items.sql
-- The Content module: a per-workspace pipeline of content the business is making
-- (posts, reels, newsletters, podcasts, threads, articles) tracked through its
-- lifecycle (idea -> draft -> scheduled -> published). Workspace-scoped.

CREATE TABLE IF NOT EXISTS content_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    title VARCHAR(300) NOT NULL,
    content_type VARCHAR(40) DEFAULT 'post',   -- post|reel|newsletter|podcast|thread|article|other
    status VARCHAR(40) DEFAULT 'idea',         -- idea|draft|scheduled|published
    link TEXT DEFAULT '',
    body TEXT DEFAULT '',
    hook TEXT DEFAULT '',
    caption TEXT DEFAULT '',
    cta TEXT DEFAULT '',
    channel VARCHAR(60) DEFAULT '',
    category VARCHAR(100) DEFAULT '',
    client VARCHAR(255) DEFAULT '',
    campaign VARCHAR(255) DEFAULT '',
    owner VARCHAR(255) DEFAULT '',
    editor VARCHAR(255) DEFAULT '',
    priority VARCHAR(40) DEFAULT 'normal',
    due_date TEXT DEFAULT '',
    publish_date TEXT DEFAULT '',
    asset_link TEXT DEFAULT '',
    review_link TEXT DEFAULT '',
    revision_notes TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_items_workspace ON content_items(workspace_id);

-- ---- 119_sites.sql ----
-- 119_sites.sql
-- The Sites module: a per-workspace registry of the business's web properties
-- (landing pages, marketing sites, apps), so humans AND AI agents have one
-- canonical list of where the business lives online. Workspace-scoped.

CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    url TEXT DEFAULT '',
    status VARCHAR(40) DEFAULT 'live',   -- live | draft | building | archived
    notes TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sites_workspace ON sites(workspace_id);

-- ---- 122_content_fields.sql ----
-- 122_content_fields.sql
-- Deepens the Content module from a bare title/body pipeline into the structure the
-- VSL / ad message bank actually implies: every piece of content has a hook (the
-- opening line that earns attention), a call to action, and a channel it ships on.
-- The body column already exists (it holds the script / copy / outline).
-- Additive and idempotent. Mirrors the message-bank model from
-- agency-miosa/inbox/signals/2026-06-29-current-vsl-and-ad-message-bank.md
-- (hook -> body -> CTA, per channel) without hardcoding any of Roberto's data.

ALTER TABLE content_items ADD COLUMN IF NOT EXISTS hook    TEXT DEFAULT '';  -- opening attention grab
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS cta     TEXT DEFAULT '';  -- call to action / next step
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS channel VARCHAR(60) DEFAULT '';  -- where it ships (instagram|youtube|tiktok|linkedin|newsletter|x|...)

-- ---- 123_project_template.sql ----
-- 123_project_template.sql
-- Project Templates: reusable, selectable delivery blueprints that turn a known
-- engagement (e.g. the Agency MIOSA "Growth Systems Audit") into a configured
-- project with its phases and deliverables already filled in.
--
-- Primitive:  a project = a bounded initiative with milestones/tasks/owners/status.
-- Feature:    "create a project from a template" — the template carries the phases
--             (milestones) and deliverables so a new client engagement is one click.
--
-- Workspace model (mirrors glossary_terms): rows are workspace-scoped via
-- workspace_id, but workspace_id IS NULL marks a GLOBAL built-in template that
-- every workspace can pick from. The Growth Systems Audit ships as a global row.
-- Additive only: no existing table/column is modified. The projects table already
-- has project_type (template key) and project_metadata JSONB (snapshot of phases),
-- so creating-from-template needs no schema change on projects itself.

CREATE TABLE IF NOT EXISTS delivery_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID,                       -- NULL = global built-in template
    key           VARCHAR(120) NOT NULL,      -- stable slug, written to projects.project_type
    name          VARCHAR(300) NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    -- phases: ordered milestones, each with deliverables. Shape:
    -- [ { "name": "Phase 0: Setup", "tasks": [...], "deliverables": [...] }, ... ]
    phases        JSONB NOT NULL DEFAULT '[]'::jsonb,
    -- deliverables: the engagement's standard outputs (flat list).
    deliverables  JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_builtin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_by    VARCHAR(255),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- One canonical key per scope. Two partial unique indexes because a single
-- UNIQUE(workspace_id, key) treats every NULL workspace_id as distinct.
CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_templates_global_key
    ON delivery_templates(key) WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_templates_ws_key
    ON delivery_templates(workspace_id, key) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_delivery_templates_workspace
    ON delivery_templates(workspace_id);

-- Seed the Growth Systems Audit as a global built-in template.
-- Source: OptimalOS agency-miosa signal "growth systems audit project template".
-- Phases mirror Phase 0 -> Phase 7; deliverables mirror "Standard Deliverables".
INSERT INTO delivery_templates (workspace_id, key, name, description, is_builtin, phases, deliverables)
VALUES (
    NULL,
    'growth_systems_audit',
    'Growth Systems Audit',
    'Repeatable Agency MIOSA delivery template. Audits a client''s growth system across the five pillars (Marketing, Acquisition, Onboarding, Delivery, Retention), finds leaks, and produces a prioritized rebuild plan and a Phase 1 proposal.',
    TRUE,
    $json$[
      {
        "name": "Phase 0: Setup",
        "tasks": [
          "Create client account record",
          "Create project",
          "Assign owner",
          "Collect access list",
          "Schedule kickoff",
          "Collect existing docs",
          "Collect tool list",
          "Collect current funnel links",
          "Collect CRM access",
          "Collect calendar / booking links",
          "Collect reporting examples"
        ],
        "deliverables": ["Client account profile", "Access checklist", "Project workspace", "Audit timeline"]
      },
      {
        "name": "Phase 1: Business Model Intake",
        "tasks": [
          "What does the company sell?",
          "Who buys it?",
          "What is the main promise?",
          "What does the client give you?",
          "What does the team do?",
          "What does the client receive?",
          "What are the main service lines?",
          "What is internal versus client-facing?"
        ],
        "deliverables": ["Business model summary", "Input / process / output map", "Internal / external map"]
      },
      {
        "name": "Phase 2: Five-Pillar Audit",
        "tasks": [
          "Marketing: tools, owner, workflow, documents, bottlenecks, risks, quick wins, rebuild needs",
          "Acquisition: tools, owner, workflow, documents, bottlenecks, risks, quick wins, rebuild needs",
          "Onboarding: tools, owner, workflow, documents, bottlenecks, risks, quick wins, rebuild needs",
          "Delivery: tools, owner, workflow, documents, bottlenecks, risks, quick wins, rebuild needs",
          "Retention: tools, owner, workflow, documents, bottlenecks, risks, quick wins, rebuild needs"
        ],
        "deliverables": ["Five-pillar map"]
      },
      {
        "name": "Phase 3: Tool And Workflow Review",
        "tasks": [
          "Review CRM, calendar, email, SMS, automation, project management, file storage, reporting, content tools, ad platforms, analytics",
          "Classify each tool: keep, clean up, replace, connect, pause, unknown"
        ],
        "deliverables": ["Tool stack map", "CRM / pipeline notes"]
      },
      {
        "name": "Phase 4: Gap Analysis",
        "tasks": [
          "Identify missing owner, process, field, handoff, document, client visibility, follow-up, reporting, proof capture, duplicate system, unnecessary automation",
          "Score each gap: impact, urgency, difficulty, revenue risk, client experience risk"
        ],
        "deliverables": ["Five-pillar gap analysis"]
      },
      {
        "name": "Phase 5: Rebuild Plan",
        "tasks": [
          "What to fix first",
          "What to ignore for now",
          "What must be rebuilt",
          "What can be improved quickly",
          "What needs custom software",
          "What needs training",
          "What needs a new SOP",
          "What needs CRM changes"
        ],
        "deliverables": ["Prioritized rebuild plan"]
      },
      {
        "name": "Phase 6: Client Readout",
        "tasks": [
          "Deliver audit summary, five-pillar map, top leaks, recommended Phase 1 scope, timeline, budget, dependencies, next step"
        ],
        "deliverables": ["Growth Systems Audit report"]
      },
      {
        "name": "Phase 7: Phase 1 Proposal",
        "tasks": [
          "If fit is strong: define Phase 1 implementation scope, milestones, deliverables, owner map, price, payment terms, start date"
        ],
        "deliverables": ["Phase 1 recommendation"]
      }
    ]$json$::jsonb,
    $json$[
      "Growth Systems Audit report",
      "Input / process / output map",
      "Internal / external asset map",
      "Tool stack map",
      "CRM / pipeline notes",
      "Five-pillar gap analysis",
      "Prioritized rebuild plan",
      "Phase 1 recommendation"
    ]$json$::jsonb
)
ON CONFLICT DO NOTHING;

-- ---- 125_knowledge_documents.sql ----
-- 125_knowledge_documents.sql
-- Cloud-stored copy of a workspace's knowledge, synced from a user's local
-- Optimal Engine / workspace files. Local-first stays the source of truth; this
-- is the shared copy so teammates WITHOUT their own engine can view the
-- knowledge in the Knowledge module (read-only), and teammates WITH an engine
-- can pull it into their local engine.
--
-- Additive only. Does not touch the local file-tree path (KNOWLEDGE_WORKSPACES_ROOT);
-- the knowledge handlers fall back to this table only when local files are absent
-- (i.e. on the cloud backend, for teammates without their own engine).
--
-- Keyed by the workspace FOLDER SLUG (matches ?workspace= in /api/knowledge/tree|file),
-- with workspace_id retained for future role-scoped gating.
DROP TABLE IF EXISTS knowledge_documents;
CREATE TABLE IF NOT EXISTS knowledge_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_slug TEXT NOT NULL,                 -- folder slug, e.g. agency-miosa
    workspace_id   UUID,                          -- BusinessOS workspace (for role gating), nullable
    path           TEXT NOT NULL,                 -- relative path within the workspace (e.g. inbox/signals/x.md)
    title          TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL DEFAULT '',      -- raw markdown (incl. frontmatter)
    section        TEXT NOT NULL DEFAULT 'docs',  -- top-level folder, drives the module tabs
    synced_by      VARCHAR(255),
    synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_slug, path)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_documents_slug ON knowledge_documents(workspace_slug);

-- ---- 127_clients_workspace.sql ----
-- 127_clients_workspace.sql
-- Clients (the Relationships module's company/account records) were purely
-- user_id-scoped, so every client a user owns appeared in EVERY workspace.
-- Wrong model: a client belongs to a business context - BetterStem is an
-- Agency MIOSA account, Paykit is a MIOSA-platform deal, ProTrade/FasterWay
-- are historical Lunivate work. Add workspace scoping.
--
-- NULL workspace_id = unassigned/personal (kept for backfill safety; the list
-- endpoint shows only the active workspace's clients when X-Workspace-ID is
-- present, so unassigned clients no longer leak into every workspace).
ALTER TABLE clients ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_clients_workspace ON clients(workspace_id);

-- ---- 128_team_members_workspace.sql ----
-- 128_team_members_workspace.sql
-- Team members were purely user_id-scoped, so the same roster appeared in
-- EVERY workspace the user opened. Same fix as clients (migration 127):
-- a team member belongs to one business context. NULL = unassigned/legacy.
ALTER TABLE team_members ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_team_members_workspace ON team_members(workspace_id);

-- ---- 129_deals_client.sql ----
-- 129_deals_client.sql
-- Link deals to their client. The deals table predates the client-nested deal
-- endpoints; CreateClientDeal now stamps client_id (clients_deals.go) so a
-- client's deals show on its list and board.
ALTER TABLE deals ADD COLUMN IF NOT EXISTS client_id UUID REFERENCES clients(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_deals_client ON deals(client_id);

-- ---- 130_workspace_apps.sql ----
-- 130_workspace_apps.sql
-- Workspace-owned URL apps that BusinessOS can launch inside its desktop.
-- These are not native OS apps and not canonical BusinessOS modules.

CREATE TABLE IF NOT EXISTS workspace_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    catalog_app_id UUID,
    name VARCHAR(160) NOT NULL,
    app_type VARCHAR(40) DEFAULT 'web_app',
    provider VARCHAR(60) DEFAULT 'custom',
    url TEXT NOT NULL DEFAULT '',
    launch_mode VARCHAR(40) DEFAULT 'iframe',
    status VARCHAR(40) DEFAULT 'active',
    icon VARCHAR(80) DEFAULT 'layout-grid',
    logo_url TEXT DEFAULT '',
    color VARCHAR(40) DEFAULT '#111827',
    category VARCHAR(80) DEFAULT 'general',
    notes TEXT DEFAULT '',
    show_on_desktop BOOLEAN DEFAULT TRUE,
    show_in_dock BOOLEAN DEFAULT TRUE,
    position_index INTEGER DEFAULT 0,
    url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_apps_workspace ON workspace_apps(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_status ON workspace_apps(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_desktop ON workspace_apps(workspace_id, show_on_desktop);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_position ON workspace_apps(workspace_id, position_index, name);

-- ---- 131_workspace_app_logos.sql ----
-- 131_workspace_app_logos.sql
-- Optional custom logo URL for workspace apps. If empty, the UI falls back to
-- the app domain's favicon.

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '';

-- ---- 132_workspace_app_provider.sql ----
-- 132_workspace_app_provider.sql
-- Tracks where an app URL came from: custom, MIOSA, Vercel, Netlify, etc.

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom';

-- ---- 133_app_catalog.sql ----
-- 133_app_catalog.sql
-- Platform catalog for apps that can be installed into a workspace desktop.

CREATE TABLE IF NOT EXISTS app_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(120) UNIQUE NOT NULL,
    name VARCHAR(160) NOT NULL,
    provider VARCHAR(60) DEFAULT 'custom',
    app_type VARCHAR(40) DEFAULT 'web_app',
    url TEXT NOT NULL DEFAULT '',
    launch_mode VARCHAR(40) DEFAULT 'iframe',
    icon VARCHAR(80) DEFAULT 'layout-grid',
    logo_url TEXT DEFAULT '',
    color VARCHAR(40) DEFAULT '#111827',
    category VARCHAR(80) DEFAULT 'general',
    notes TEXT DEFAULT '',
    url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
    status VARCHAR(40) DEFAULT 'active',
    is_featured BOOLEAN DEFAULT FALSE,
    position_index INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE app_catalog
ADD COLUMN IF NOT EXISTS slug VARCHAR(120),
ADD COLUMN IF NOT EXISTS name VARCHAR(160),
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS launch_mode VARCHAR(40) DEFAULT 'iframe',
ADD COLUMN IF NOT EXISTS icon VARCHAR(80) DEFAULT 'layout-grid',
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS color VARCHAR(40) DEFAULT '#111827',
ADD COLUMN IF NOT EXISTS category VARCHAR(80) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS position_index INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

ALTER TABLE workspace_apps
ADD COLUMN IF NOT EXISTS catalog_app_id UUID REFERENCES app_catalog(id) ON DELETE SET NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_index i
        JOIN pg_attribute a
          ON a.attrelid = i.indrelid
         AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = 'app_catalog'::regclass
          AND i.indisunique
          AND a.attname = 'slug'
    ) THEN
        CREATE UNIQUE INDEX idx_app_catalog_slug_unique ON app_catalog(slug);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_app_catalog_status ON app_catalog(status, position_index, name);
CREATE INDEX IF NOT EXISTS idx_app_catalog_category ON app_catalog(category, position_index, name);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_catalog ON workspace_apps(workspace_id, catalog_app_id);

-- ---- 134_seed_core_app_catalog.sql ----
-- 134_seed_core_app_catalog.sql
-- Real default catalog entries for commonly used web apps.

INSERT INTO app_catalog (
    slug, name, provider, app_type, url, launch_mode, icon, logo_url, color,
    category, notes, url_class, status, is_featured, position_index
) VALUES
    ('chatgpt', 'ChatGPT', 'openai', 'web_app', 'https://chatgpt.com', 'iframe', 'message-square', '/app-logos/chatgpt.svg', '#111827', 'ai', 'OpenAI ChatGPT web app.', 'custom_domain', 'active', TRUE, 10),
    ('claude', 'Claude', 'anthropic', 'web_app', 'https://claude.ai', 'iframe', 'message-square', '/app-logos/claude.svg', '#111827', 'ai', 'Anthropic Claude web app.', 'custom_domain', 'active', TRUE, 20),
    ('perplexity', 'Perplexity', 'perplexity', 'web_app', 'https://www.perplexity.ai', 'iframe', 'search', '/app-logos/perplexity.svg', '#111827', 'ai', 'AI answer engine and research app.', 'custom_domain', 'active', TRUE, 30),
    ('google', 'Google', 'google', 'web_app', 'https://www.google.com', 'iframe', 'search', '/app-logos/google.svg', '#4285F4', 'search', 'Google Search.', 'custom_domain', 'active', TRUE, 40),
    ('openai-platform', 'OpenAI Platform', 'openai', 'web_app', 'https://platform.openai.com', 'iframe', 'terminal', '/app-logos/openai.svg', '#111827', 'ai', 'OpenAI developer platform.', 'custom_domain', 'active', TRUE, 50),
    ('higgsfield', 'Higgsfield', 'higgsfield', 'web_app', 'https://higgsfield.ai', 'iframe', 'sparkles', '/app-logos/higgsfield.ico', '#111827', 'creative', 'AI-native creative suite for image, video, and voice workflows.', 'custom_domain', 'active', TRUE, 60),
    ('slack', 'Slack', 'slack', 'web_app', 'https://app.slack.com', 'iframe', 'messages-square', '/app-logos/slack.ico', '#4A154B', 'communication', 'Team messaging and channels.', 'custom_domain', 'active', TRUE, 70),
    ('discord', 'Discord', 'discord', 'web_app', 'https://discord.com/app', 'iframe', 'messages-square', '/app-logos/discord.svg', '#5865F2', 'communication', 'Community and team voice/chat workspace.', 'custom_domain', 'active', TRUE, 80),
    ('notion', 'Notion', 'notion', 'web_app', 'https://www.notion.so', 'iframe', 'file-text', '/app-logos/notion.svg', '#111827', 'docs', 'Docs, wiki, and databases.', 'custom_domain', 'active', TRUE, 90),
    ('clickup', 'ClickUp', 'clickup', 'web_app', 'https://app.clickup.com', 'iframe', 'check-square', '/app-logos/clickup.svg', '#7B68EE', 'project-management', 'Tasks, docs, goals, and operations.', 'custom_domain', 'active', TRUE, 100),
    ('gohighlevel', 'GoHighLevel', 'gohighlevel', 'web_app', 'https://app.gohighlevel.com', 'iframe', 'workflow', '/app-logos/gohighlevel.ico', '#111827', 'crm', 'CRM, marketing automation, and agency client ops.', 'custom_domain', 'active', TRUE, 110),
    ('hubspot', 'HubSpot', 'hubspot', 'web_app', 'https://app.hubspot.com', 'iframe', 'users', '/app-logos/hubspot.ico', '#FF7A59', 'crm', 'CRM, sales, marketing, and support platform.', 'custom_domain', 'active', TRUE, 120),
    ('gmail', 'Gmail', 'google', 'web_app', 'https://mail.google.com', 'iframe', 'mail', '/app-logos/gmail.ico', '#EA4335', 'communication', 'Google email inbox.', 'custom_domain', 'active', FALSE, 130),
    ('google-calendar', 'Google Calendar', 'google', 'web_app', 'https://calendar.google.com', 'iframe', 'calendar', '/app-logos/google-calendar.svg', '#4285F4', 'calendar', 'Google Calendar.', 'custom_domain', 'active', FALSE, 140),
    ('google-drive', 'Google Drive', 'google', 'web_app', 'https://drive.google.com', 'iframe', 'folder', '/app-logos/google-drive.ico', '#0F9D58', 'files', 'Google file storage and docs.', 'custom_domain', 'active', FALSE, 130),
    ('google-meet', 'Google Meet', 'google', 'web_app', 'https://meet.google.com', 'iframe', 'video', '/app-logos/google-meet.ico', '#00897B', 'meetings', 'Google video meetings.', 'custom_domain', 'active', FALSE, 140),
    ('zoom', 'Zoom', 'zoom', 'web_app', 'https://app.zoom.us/wc', 'iframe', 'video', '/app-logos/zoom.ico', '#0B5CFF', 'meetings', 'Zoom web client.', 'custom_domain', 'active', FALSE, 150),
    ('fathom', 'Fathom', 'fathom', 'web_app', 'https://fathom.video', 'iframe', 'mic', '/app-logos/fathom.ico', '#111827', 'meetings', 'Meeting recorder and summaries.', 'custom_domain', 'active', FALSE, 160),
    ('fireflies', 'Fireflies', 'fireflies', 'web_app', 'https://app.fireflies.ai', 'iframe', 'mic', '/app-logos/fireflies.ico', '#6D5DFB', 'meetings', 'Meeting notes and conversation intelligence.', 'custom_domain', 'active', FALSE, 170),
    ('figma', 'Figma', 'figma', 'web_app', 'https://www.figma.com/files', 'iframe', 'pen-tool', '/app-logos/figma.svg', '#111827', 'design', 'Collaborative design files.', 'custom_domain', 'active', FALSE, 180),
    ('airtable', 'Airtable', 'airtable', 'web_app', 'https://airtable.com', 'iframe', 'table', '/app-logos/airtable.webp', '#FCB400', 'database', 'No-code databases and operations tables.', 'custom_domain', 'active', FALSE, 190),
    ('trello', 'Trello', 'trello', 'web_app', 'https://trello.com', 'iframe', 'kanban', '/app-logos/trello.ico', '#0079BF', 'project-management', 'Kanban boards and task tracking.', 'custom_domain', 'active', FALSE, 200),
    ('asana', 'Asana', 'asana', 'web_app', 'https://app.asana.com', 'iframe', 'list-checks', '/app-logos/asana.ico', '#F06A6A', 'project-management', 'Team projects and task management.', 'custom_domain', 'active', FALSE, 210),
    ('linear', 'Linear', 'linear', 'web_app', 'https://linear.app', 'iframe', 'list-checks', '/app-logos/linear.ico', '#5E6AD2', 'project-management', 'Issues, product work, and engineering planning.', 'custom_domain', 'active', FALSE, 220),
    ('github', 'GitHub', 'github', 'web_app', 'https://github.com', 'iframe', 'github', '/app-logos/github.ico', '#111827', 'development', 'Code repositories and collaboration.', 'custom_domain', 'active', FALSE, 230),
    ('vercel', 'Vercel', 'vercel', 'web_app', 'https://vercel.com/dashboard', 'iframe', 'triangle', '/app-logos/vercel.ico', '#111827', 'deployment', 'Frontend deployments and hosting.', 'custom_domain', 'active', FALSE, 240),
    ('netlify', 'Netlify', 'netlify', 'web_app', 'https://app.netlify.com', 'iframe', 'cloud', '/app-logos/netlify.ico', '#00AD9F', 'deployment', 'Web deployments and hosting.', 'custom_domain', 'active', FALSE, 250),
    ('render', 'Render', 'render', 'web_app', 'https://dashboard.render.com', 'iframe', 'cloud', '/app-logos/render.svg', '#46E3B7', 'deployment', 'App hosting, workers, and databases.', 'custom_domain', 'active', FALSE, 260),
    ('replit', 'Replit', 'replit', 'web_app', 'https://replit.com', 'iframe', 'code-2', '/app-logos/replit.svg', '#F26207', 'development', 'Browser IDE and deployments.', 'custom_domain', 'active', FALSE, 270),
    ('supabase', 'Supabase', 'supabase', 'web_app', 'https://supabase.com/dashboard', 'iframe', 'database', '/app-logos/supabase.svg', '#3ECF8E', 'database', 'Postgres, auth, storage, and backend services.', 'custom_domain', 'active', FALSE, 280),
    ('stripe', 'Stripe', 'stripe', 'web_app', 'https://dashboard.stripe.com', 'iframe', 'credit-card', '/app-logos/stripe.ico', '#635BFF', 'payments', 'Payments, billing, and financial operations.', 'custom_domain', 'active', FALSE, 290),
    ('shopify', 'Shopify', 'shopify', 'web_app', 'https://admin.shopify.com', 'iframe', 'shopping-bag', '/app-logos/shopify.ico', '#95BF47', 'commerce', 'Commerce admin and store operations.', 'custom_domain', 'active', FALSE, 300),
    ('webflow', 'Webflow', 'webflow', 'web_app', 'https://webflow.com/dashboard', 'iframe', 'layout-template', '/app-logos/webflow.ico', '#146EF5', 'sites', 'Visual website builder and CMS.', 'custom_domain', 'active', FALSE, 310),
    ('canva', 'Canva', 'canva', 'web_app', 'https://www.canva.com', 'iframe', 'palette', '/app-logos/canva.ico', '#00C4CC', 'creative', 'Design and brand creative production.', 'custom_domain', 'active', FALSE, 320),
    ('capcut', 'CapCut', 'capcut', 'web_app', 'https://www.capcut.com', 'iframe', 'scissors', '/app-logos/capcut.ico', '#111827', 'creative', 'Video editing and creative production.', 'custom_domain', 'active', FALSE, 330),
    ('midjourney', 'Midjourney', 'midjourney', 'web_app', 'https://www.midjourney.com', 'iframe', 'image', '/app-logos/midjourney.png', '#111827', 'creative', 'AI image generation.', 'custom_domain', 'active', FALSE, 340),
    ('runway', 'Runway', 'runway', 'web_app', 'https://runwayml.com', 'iframe', 'film', '/app-logos/runway.png', '#111827', 'creative', 'AI video generation and editing.', 'custom_domain', 'active', FALSE, 350),
    ('elevenlabs', 'ElevenLabs', 'elevenlabs', 'web_app', 'https://elevenlabs.io/app', 'iframe', 'audio-lines', '/app-logos/elevenlabs.ico', '#111827', 'creative', 'AI voice and audio generation.', 'custom_domain', 'active', FALSE, 360),
    ('descript', 'Descript', 'descript', 'web_app', 'https://web.descript.com', 'iframe', 'file-audio', '/app-logos/descript.png', '#111827', 'creative', 'Audio/video editing and transcription.', 'custom_domain', 'active', FALSE, 370),
    ('zapier', 'Zapier', 'zapier', 'web_app', 'https://zapier.com/app/home', 'iframe', 'workflow', '/app-logos/zapier.ico', '#FF4A00', 'automation', 'Workflow automation between apps.', 'custom_domain', 'active', FALSE, 380),
    ('make', 'Make', 'make', 'web_app', 'https://www.make.com/en/hq/app', 'iframe', 'workflow', '/app-logos/make.ico', '#6D00CC', 'automation', 'Visual workflow automation.', 'custom_domain', 'active', FALSE, 390),
    ('n8n', 'n8n', 'n8n', 'web_app', 'https://app.n8n.cloud', 'iframe', 'workflow', '/app-logos/n8n.ico', '#EA4B71', 'automation', 'Workflow automation and integrations.', 'custom_domain', 'active', FALSE, 400),
    ('calendly', 'Calendly', 'calendly', 'web_app', 'https://calendly.com/app', 'iframe', 'calendar-check', '/app-logos/calendly.ico', '#006BFF', 'scheduling', 'Scheduling links and booking pages.', 'custom_domain', 'active', FALSE, 410),
    ('typeform', 'Typeform', 'typeform', 'web_app', 'https://admin.typeform.com', 'iframe', 'clipboard-list', '/app-logos/typeform.svg', '#111827', 'forms', 'Forms, surveys, and lead capture.', 'custom_domain', 'active', FALSE, 420),
    ('mailchimp', 'Mailchimp', 'mailchimp', 'web_app', 'https://mailchimp.com', 'iframe', 'mail', '/app-logos/mailchimp.ico', '#FFE01B', 'email', 'Email marketing and audience management.', 'custom_domain', 'active', FALSE, 430),
    ('beehiiv', 'beehiiv', 'beehiiv', 'web_app', 'https://app.beehiiv.com', 'iframe', 'newspaper', '/app-logos/beehiiv.png', '#FACC15', 'email', 'Newsletter publishing and growth.', 'custom_domain', 'active', FALSE, 440),
    ('meta-business', 'Meta Business Suite', 'meta', 'web_app', 'https://business.facebook.com', 'iframe', 'megaphone', '/app-logos/meta-business.ico', '#0866FF', 'ads', 'Meta ads and business management.', 'custom_domain', 'active', FALSE, 450),
    ('tiktok-ads', 'TikTok Ads Manager', 'tiktok', 'web_app', 'https://ads.tiktok.com', 'iframe', 'megaphone', '/app-logos/tiktok-ads.svg', '#111827', 'ads', 'TikTok advertising dashboard.', 'custom_domain', 'active', FALSE, 460)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    provider = EXCLUDED.provider,
    app_type = EXCLUDED.app_type,
    url = EXCLUDED.url,
    launch_mode = EXCLUDED.launch_mode,
    icon = EXCLUDED.icon,
    logo_url = EXCLUDED.logo_url,
    color = EXCLUDED.color,
    category = EXCLUDED.category,
    notes = EXCLUDED.notes,
    url_class = EXCLUDED.url_class,
    status = EXCLUDED.status,
    is_featured = EXCLUDED.is_featured,
    position_index = EXCLUDED.position_index,
    updated_at = NOW();

-- ---- 135_seed_perplexity_google_apps.sql ----
-- 135_seed_perplexity_google_apps.sql
-- Add Perplexity and Google Search to the workspace app catalog.

INSERT INTO app_catalog (
    slug, name, provider, app_type, url, launch_mode, icon, logo_url, color,
    category, notes, url_class, status, is_featured, position_index
) VALUES
    ('perplexity', 'Perplexity', 'perplexity', 'web_app', 'https://www.perplexity.ai', 'iframe', 'search', '/app-logos/perplexity.svg', '#111827', 'ai', 'AI answer engine and research app.', 'custom_domain', 'active', TRUE, 30),
    ('google', 'Google', 'google', 'web_app', 'https://www.google.com', 'iframe', 'search', '/app-logos/google.svg', '#4285F4', 'search', 'Google Search.', 'custom_domain', 'active', TRUE, 40)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    provider = EXCLUDED.provider,
    app_type = EXCLUDED.app_type,
    url = EXCLUDED.url,
    launch_mode = EXCLUDED.launch_mode,
    icon = EXCLUDED.icon,
    logo_url = EXCLUDED.logo_url,
    color = EXCLUDED.color,
    category = EXCLUDED.category,
    notes = EXCLUDED.notes,
    url_class = EXCLUDED.url_class,
    status = EXCLUDED.status,
    is_featured = EXCLUDED.is_featured,
    position_index = EXCLUDED.position_index,
    updated_at = NOW();

-- ---- 136_fix_app_catalog_logos.sql ----
-- 136_fix_app_catalog_logos.sql
-- Replace bad or missing catalog logo paths with real assets.

UPDATE app_catalog
SET logo_url = '/app-logos/gohighlevel.ico',
    updated_at = NOW()
WHERE slug = 'gohighlevel';

UPDATE app_catalog
SET logo_url = '/app-logos/google-calendar.svg',
    updated_at = NOW()
WHERE slug = 'google-calendar';

UPDATE app_catalog
SET logo_url = '/app-logos/clickup.svg',
    updated_at = NOW()
WHERE slug = 'clickup';

-- ---- 137_app_catalog_schema_guards.sql ----
-- 137_app_catalog_schema_guards.sql
-- Keep older local databases aligned with the app catalog handler schema.
-- This is intentionally idempotent and does not seed workspace app data.

ALTER TABLE IF EXISTS workspace_apps
ADD COLUMN IF NOT EXISTS catalog_app_id UUID,
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS launch_mode VARCHAR(40) DEFAULT 'iframe',
ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS icon VARCHAR(80) DEFAULT 'layout-grid',
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS color VARCHAR(40) DEFAULT '#111827',
ADD COLUMN IF NOT EXISTS category VARCHAR(80) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS show_on_desktop BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS show_in_dock BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS position_index INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
ADD COLUMN IF NOT EXISTS created_by VARCHAR(255),
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

ALTER TABLE IF EXISTS app_catalog
ADD COLUMN IF NOT EXISTS slug VARCHAR(120),
ADD COLUMN IF NOT EXISTS provider VARCHAR(60) DEFAULT 'custom',
ADD COLUMN IF NOT EXISTS app_type VARCHAR(40) DEFAULT 'web_app',
ADD COLUMN IF NOT EXISTS launch_mode VARCHAR(40) DEFAULT 'iframe',
ADD COLUMN IF NOT EXISTS icon VARCHAR(80) DEFAULT 'layout-grid',
ADD COLUMN IF NOT EXISTS logo_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS color VARCHAR(40) DEFAULT '#111827',
ADD COLUMN IF NOT EXISTS category VARCHAR(80) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS url_class VARCHAR(80) DEFAULT 'stable_sandbox_embed',
ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS position_index INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_workspace_apps_workspace ON workspace_apps(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_status ON workspace_apps(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_desktop ON workspace_apps(workspace_id, show_on_desktop);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_position ON workspace_apps(workspace_id, position_index, name);
CREATE INDEX IF NOT EXISTS idx_workspace_apps_catalog ON workspace_apps(workspace_id, catalog_app_id);
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_index i
        JOIN pg_attribute a
          ON a.attrelid = i.indrelid
         AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = 'app_catalog'::regclass
          AND i.indisunique
          AND a.attname = 'slug'
    ) THEN
        CREATE UNIQUE INDEX idx_app_catalog_slug_unique ON app_catalog(slug);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_app_catalog_status ON app_catalog(status, position_index, name);
CREATE INDEX IF NOT EXISTS idx_app_catalog_category ON app_catalog(category, position_index, name);

UPDATE app_catalog
SET logo_url = '/app-logos/google-calendar.svg',
    updated_at = NOW()
WHERE slug = 'google-calendar'
  AND logo_url = '/app-logos/google-calendar.ico';

UPDATE app_catalog
SET logo_url = '/app-logos/clickup.svg',
    updated_at = NOW()
WHERE slug = 'clickup'
  AND logo_url = '/app-logos/clickup.ico';

-- ---- 138_workspace_desktop_spaces.sql ----
-- 138_workspace_desktop_spaces.sql
-- Workspace-owned desktop spaces for shared BusinessOS desktops.

CREATE TABLE IF NOT EXISTS workspace_desktop_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    kind VARCHAR(40) NOT NULL DEFAULT 'personal' CHECK (kind IN ('personal', 'team', 'workspace')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_desktop_spaces_workspace
    ON workspace_desktop_spaces(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_workspace_desktop_spaces_kind
    ON workspace_desktop_spaces(workspace_id, kind, name);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_workspace_id_fkey'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_workspace_id_fkey
            FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_kind_check'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_kind_check
            CHECK (kind IN ('personal', 'team', 'workspace'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'workspace_desktop_spaces_config_check'
          AND conrelid = 'workspace_desktop_spaces'::regclass
    ) THEN
        ALTER TABLE workspace_desktop_spaces
            ADD CONSTRAINT workspace_desktop_spaces_config_check
            CHECK (jsonb_typeof(config) = 'object');
    END IF;
END $$;

-- ---- 139_app_catalog_embedded_launch.sql ----
-- 139_app_catalog_embedded_launch.sql
-- Curated web apps launch inside BusinessOS by default.
-- Explicit browser or external launch remains available per workspace app.

UPDATE app_catalog
SET launch_mode = 'iframe',
    updated_at = NOW()
WHERE slug IN (
    'chatgpt',
    'claude',
    'perplexity',
    'google',
    'openai-platform',
    'higgsfield',
    'slack',
    'discord',
    'notion',
    'clickup',
    'gohighlevel',
    'hubspot',
    'gmail',
    'google-calendar',
    'google-drive',
    'google-meet',
    'zoom',
    'fathom',
    'fireflies',
    'figma',
    'airtable',
    'trello',
    'asana',
    'linear',
    'github',
    'vercel',
    'netlify',
    'render',
    'replit',
    'supabase',
    'stripe',
    'shopify',
    'webflow',
    'canva',
    'capcut',
    'midjourney',
    'runway',
    'elevenlabs',
    'descript',
    'zapier',
    'make',
    'n8n',
    'calendly',
    'typeform',
    'mailchimp',
    'beehiiv',
    'meta-business',
    'tiktok-ads'
);

-- ---- 140_workspace_storage.sql ----
-- 140_workspace_storage.sql
-- Per-workspace cloud-sync storage accounting for the Knowledge module.
-- When a workspace's local knowledge is synced up (knowledge_documents), we
-- record how many bytes that workspace occupies and the quota it is allowed.
-- Free tier = 1 GB. Enforced at sync time: a sync that would push bytes_used
-- past bytes_limit is rejected (HTTP 413) and rolled back.
--
-- Additive only. Keyed by BusinessOS workspace_id (UUID), matching the
-- workspace_id retained on knowledge_documents. Idempotent (IF NOT EXISTS).
--
-- NOTE: numbered 140 (not 130) because 130_workspace_apps.sql already exists;
-- migrations apply in numeric order and 130-139 are taken.
CREATE TABLE IF NOT EXISTS workspace_storage (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    bytes_used   BIGINT NOT NULL DEFAULT 0,
    bytes_limit  BIGINT NOT NULL DEFAULT 1073741824,   -- 1 GB free tier
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- === schema-drift reconciliation part 3: module fields (124) ===
-- 124_module_fields.sql
-- Additive fields for the five Agency MIOSA marketing/ops modules, configured to
-- the real shape of the business (Growth System Audit offer ladder, VSL/ad
-- message bank, ICP fit, Robert/Roberto ownership split, funnel pages).
-- All ADD COLUMN IF NOT EXISTS so this is safe to re-run. Placed AFTER the
-- canonical-migrations block above, which is where these tables are created.

-- Offers: the Growth System Audit offer ladder.
--   tier    = where this sits in the ladder (audit | phase-1 | phase-2 | lane)
--   promise = the outcome one-liner the buyer is paying for
--   cta     = the call-to-action that sells it (default Solutions Call)
ALTER TABLE offers ADD COLUMN IF NOT EXISTS tier    VARCHAR(40)  DEFAULT 'audit';
ALTER TABLE offers ADD COLUMN IF NOT EXISTS promise TEXT         DEFAULT '';
ALTER TABLE offers ADD COLUMN IF NOT EXISTS cta     VARCHAR(120) DEFAULT 'Book a Solutions Call';

-- Campaigns: channel pushes carry the message bank hook + CTA.
--   hook = the buyer hook this push leads with
--   cta  = the call-to-action it drives to
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS hook TEXT         DEFAULT '';
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS cta  VARCHAR(120) DEFAULT 'Book a Solutions Call';

-- Personas: ICP segments are explicitly best-fit vs poor-fit in the offer doc.
--   fit = best | poor (drives who we chase vs disqualify)
ALTER TABLE personas ADD COLUMN IF NOT EXISTS fit VARCHAR(20) DEFAULT 'best';

-- Rhythm: the operating cadence has a clear Robert / Roberto ownership split.
--   owner = who owns this cadence entry (free text: Robert | Roberto | team | ...)
ALTER TABLE rhythm_entries ADD COLUMN IF NOT EXISTS owner VARCHAR(120) DEFAULT '';

-- Sites: web properties are funnels/pages, not just generic sites.
--   kind = funnel | page | form | site | app
--   cta  = the primary call-to-action on the property
ALTER TABLE sites ADD COLUMN IF NOT EXISTS kind VARCHAR(40)  DEFAULT 'page';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS cta  VARCHAR(120) DEFAULT 'Book a Solutions Call';

-- BusinessOSSync: workspace sharing for projects + tasks (mirror of migration
-- 102_businessossync_projects_tasks.sql). Declared here as ALTERs — not inline
-- in the CREATE TABLE for `projects`/`tasks` above — because `workspaces` is
-- defined later in this file, so a fresh build must create those tables first.
-- Nullable: NULL = private/user-scoped; a workspace_id lets active members see
-- shared work. The sqlc ListTasks/ListProjects queries filter on this column,
-- so it MUST exist or those endpoints 500 (SQLSTATE 42703). Idempotent.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
ALTER TABLE tasks    ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_projects_workspace_updated ON projects(workspace_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_tasks_workspace_updated ON tasks(workspace_id, updated_at);

-- ============================================================================
-- CANONICAL FOUNDATION MIGRATIONS (090, 100, 101, 103) — folded in 2026-07-10.
-- These create the org/team/sync/governance foundation that schema.sql had
-- fallen behind on. Without them a fresh install has NO organizations/
-- organization_members/teams/sync_policies tables, so provisionDefaultWorkspace
-- crashes on signup (INSERT organizations) and users get no workspace — every
-- workspace-scoped module then dead-ends. governance_events (100) backs the
-- engine's homeostatic loop; missing it spams the log. All idempotent; placed
-- at the END because they ALTER tables (workspaces, contexts, osa_workspaces)
-- defined earlier in this file.
-- ============================================================================

-- ---- 090_tenant_org_foundation.sql ----
-- ═══════════════════════════════════════════════════════════════════════════════
-- Tenant / Organization Foundation Migration
-- ═══════════════════════════════════════════════════════════════════════════════
-- Created: 2026-02-23
-- Description: Establishes the multi-tenant organization layer.
--              Creates organizations and organization_members tables,
--              then links osa_workspaces and custom_modules via a nullable
--              organization_id FK so both tables remain backward-compatible.
--
-- Design notes:
--   • No RLS policies are applied here; enforcement is left to the
--     application layer or a follow-up migration.
--   • organization_id is nullable on both linked tables so existing rows
--     continue to work without backfill.
--   • plan is a free-form text column; a CHECK constraint can be added
--     in a later migration once the plan list stabilizes.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: organizations
-- Top-level tenant entity. One organization can contain many workspaces/modules.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS organizations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identity
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL,   -- URL-safe handle, e.g. "acme-corp"

    -- Billing / feature tier
    plan        TEXT        NOT NULL DEFAULT 'free',  -- 'free', 'pro', 'business', 'enterprise'

    -- Flexible org-level configuration (theme, feature flags, limits, etc.)
    settings    JSONB       NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT organizations_slug_unique UNIQUE (slug),
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9][a-z0-9_-]*[a-z0-9]$')
);

-- Indexes for organizations
CREATE INDEX IF NOT EXISTS idx_organizations_slug       ON organizations (slug);
CREATE INDEX IF NOT EXISTS idx_organizations_plan       ON organizations (plan);
CREATE INDEX IF NOT EXISTS idx_organizations_created_at ON organizations (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_organizations_settings   ON organizations USING GIN (settings);

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: keep organizations.updated_at current
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_organizations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
CREATE TRIGGER trigger_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_organizations_updated_at();
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: organization_members
-- Maps users to organizations with a role.
-- user_id is TEXT to match the "user".id column type used elsewhere.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS organization_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    -- NOTE: column is organization_id (its FINAL name). Migration 090 originally
    -- created this as org_id and migration 103 renamed it -> organization_id;
    -- folding both verbatim made schema.sql non-idempotent (re-apply hit
    -- "column org_id does not exist" on the org_id indexes after 103's rename).
    -- Declaring the final name here makes 103's guarded RENAME a no-op.
    organization_id UUID     NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL,  -- matches "user".id VARCHAR(255) / TEXT type

    -- Access control
    role        TEXT        NOT NULL DEFAULT 'member',  -- 'owner', 'admin', 'member', 'viewer'

    -- Timestamps
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT organization_members_unique UNIQUE (organization_id, user_id),
    CONSTRAINT organization_members_role_check CHECK (
        role IN ('owner', 'admin', 'member', 'viewer')
    )
);

-- Indexes for organization_members
CREATE INDEX IF NOT EXISTS idx_org_members_org_id  ON organization_members (organization_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON organization_members (user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_role    ON organization_members (organization_id, role);

-- ───────────────────────────────────────────────────────────────────────────────
-- Alter: osa_workspaces — add organization_id (nullable)
-- Allows a workspace to be scoped to an organization while keeping all
-- existing rows intact (no backfill required).
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE osa_workspaces
    ADD COLUMN IF NOT EXISTS organization_id UUID
        REFERENCES organizations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_osa_workspaces_org
    ON osa_workspaces (organization_id)
    WHERE organization_id IS NOT NULL;

-- ───────────────────────────────────────────────────────────────────────────────
-- Alter: custom_modules — add organization_id (nullable)
-- Allows modules to be shared across a whole organization, not just a workspace.
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE custom_modules
    ADD COLUMN IF NOT EXISTS organization_id UUID
        REFERENCES organizations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_custom_modules_org
    ON custom_modules (organization_id)
    WHERE organization_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- Comments for documentation
-- ═══════════════════════════════════════════════════════════════════════════════

COMMENT ON TABLE organizations IS
    'Top-level tenant entity. Groups users, workspaces, and modules under a shared billing/plan context.';

COMMENT ON COLUMN organizations.slug IS
    'URL-safe, lowercase handle unique to each organization (e.g. "acme-corp").';

COMMENT ON COLUMN organizations.plan IS
    'Billing tier: free | pro | business | enterprise. Enforced by application layer.';

COMMENT ON COLUMN organizations.settings IS
    'Flexible JSONB bag for org-level config: feature flags, theme overrides, usage limits, etc.';

COMMENT ON TABLE organization_members IS
    'Many-to-many join between users and organizations with role-based access control.';

COMMENT ON COLUMN organization_members.user_id IS
    'References "user".id. Stored as TEXT to match the existing user ID column type.';

COMMENT ON COLUMN organization_members.role IS
    'Access tier: owner (full control), admin (manage members/settings), member (standard), viewer (read-only).';

COMMENT ON COLUMN osa_workspaces.organization_id IS
    'Optional FK to organizations. NULL means the workspace belongs to an individual user only.';

COMMENT ON COLUMN custom_modules.organization_id IS
    'Optional FK to organizations. NULL means the module is scoped to its workspace only.';

-- ═══════════════════════════════════════════════════════════════════════════════
-- END OF MIGRATION
-- ═══════════════════════════════════════════════════════════════════════════════

-- ---- 100_governance_events.sql ----
-- Governance Events audit trail
-- Beer's VSM algedonic channel and governance event logging

CREATE TABLE IF NOT EXISTS governance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL DEFAULT 'algedonic',   -- 'algedonic', 'setpoint_adjustment', 'policy_change'
    source TEXT NOT NULL,                            -- Component that fired the event
    severity TEXT NOT NULL DEFAULT 'INFO',           -- CRITICAL, HIGH, MEDIUM, LOW, INFO
    description TEXT NOT NULL DEFAULT '',             -- Human-readable description
    metadata JSONB DEFAULT '{}',                     -- Event-specific data
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_governance_events_type ON governance_events (event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_governance_events_severity ON governance_events (severity, created_at DESC);

COMMENT ON TABLE governance_events IS 'Audit trail for governance decisions, algedonic signals, and policy changes';

-- ---- 101_businessossync_foundation.sql ----
-- ═══════════════════════════════════════════════════════════════════════════════
-- BusinessOSSync Foundation Migration
-- ═══════════════════════════════════════════════════════════════════════════════
-- Created: 2026-06-24
-- Description: Adds workspace scoping to shareable content (starting with contexts)
--              and a per-module sync policy table (canonical = sync to workspace,
--              one-off = local only). This is the data-model foundation for
--              BusinessOSSync (local-first + selective cloud sync).
--
-- Safety: fully idempotent (IF NOT EXISTS / ON CONFLICT DO NOTHING). Safe to run
--         against the existing populated cloud DB. NULL workspace_id = local-only,
--         so nothing leaks across the team until a row is deliberately stamped.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- 1. Workspace scoping on contexts (knowledge docs).
--    Nullable on purpose: NULL = private/local-only. A row only becomes
--    team-visible when stamped with a workspace_id by a workspace-mode module.
--    ON DELETE SET NULL: deleting a workspace demotes its docs to local, never
--    destroys content.
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE contexts
    ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_contexts_workspace_updated
    ON contexts(workspace_id, updated_at);

-- ───────────────────────────────────────────────────────────────────────────────
-- 2. Per-module sync policy.
--    One row per (workspace, module). sync_mode='workspace' => rows of that module
--    sync to the shared cloud DB for the team; sync_mode='local' => stay on device.
--    is_published gates whether the workspace-mode data is currently shared.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sync_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module VARCHAR(100) NOT NULL,
    sync_mode VARCHAR(20) NOT NULL DEFAULT 'local'
        CHECK (sync_mode IN ('workspace', 'local')),
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, module)
);

CREATE INDEX IF NOT EXISTS idx_sync_policies_workspace ON sync_policies(workspace_id);

-- ───────────────────────────────────────────────────────────────────────────────
-- 3. Backfill existing contexts -> the owner's workspace, ONLY when unambiguous
--    (the user belongs to exactly one active workspace). Multi-workspace users are
--    left NULL (local-only) to avoid guessing. user_id is VARCHAR on both sides.
-- ───────────────────────────────────────────────────────────────────────────────
UPDATE contexts c
SET workspace_id = m.workspace_id
FROM (
    -- exactly one active workspace per user here, so (array_agg)[1] is unambiguous
    -- (uuid has no MIN() aggregate, hence array_agg instead)
    SELECT user_id, (array_agg(workspace_id))[1] AS workspace_id
    FROM workspace_members
    WHERE status = 'active'
    GROUP BY user_id
    HAVING COUNT(*) = 1
) m
WHERE c.user_id = m.user_id
  AND c.workspace_id IS NULL;

-- ───────────────────────────────────────────────────────────────────────────────
-- 4. Seed canonical module policies for every existing workspace so shared modules
--    sync out of the box. One-off / mini-app modules are intentionally NOT seeded
--    (absence => treated as local by the application).
-- ───────────────────────────────────────────────────────────────────────────────
INSERT INTO sync_policies (workspace_id, module, sync_mode, is_published)
SELECT w.id, m.module, 'workspace', TRUE
FROM workspaces w
CROSS JOIN (VALUES ('knowledge'), ('contexts'), ('projects'), ('tasks')) AS m(module)
ON CONFLICT (workspace_id, module) DO NOTHING;

-- ---- 103_organizations_teams.sql ----
-- ─────────────────────────────────────────────────────────────────────────────
-- 103: Organizations + Teams hierarchy (adaptive — partial org tables pre-existed)
--
--   Organization (the account)
--     ├── organization_members  (everyone in the account; org role)
--     ├── organization_invites   (invite people to the org)
--     └── Workspace (workspaces.organization_id)
--           ├── workspace_members  (GRANULAR access — which org people see this workspace)
--           └── Team (teams.workspace_id) · team_memberships (move users between teams)
--
-- Org membership != workspace access. Role-scoped module data stays on
-- workspace_roles.permissions JSONB (unchanged here).
-- ─────────────────────────────────────────────────────────────────────────────

-- 1. Organizations — bring the pre-existing (empty) table up to spec.
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS owner_id VARCHAR(255);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500);
CREATE INDEX IF NOT EXISTS idx_organizations_owner ON organizations(owner_id);

-- 2. Organization members — normalize column name to organization_id (table is empty).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='organization_members' AND column_name='org_id')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='organization_members' AND column_name='organization_id') THEN
        ALTER TABLE organization_members RENAME COLUMN org_id TO organization_id;
    END IF;
END $$;
CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS invited_by VARCHAR(255);
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
CREATE UNIQUE INDEX IF NOT EXISTS uq_org_members_org_user ON organization_members(organization_id, user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);

-- 3. Organization invites
CREATE TABLE IF NOT EXISTS organization_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    invited_by VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_org_invites_org ON organization_invites(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_invites_token ON organization_invites(token);

-- 4. Link workspaces to their organization
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_workspaces_org ON workspaces(organization_id);

-- 5. Teams — inside a workspace
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(20),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_teams_workspace ON teams(workspace_id);

-- 6. Team memberships — move users between teams
CREATE TABLE IF NOT EXISTS team_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'member',
    added_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_team_memberships_team ON team_memberships(team_id);
CREATE INDEX IF NOT EXISTS idx_team_memberships_user ON team_memberships(user_id);

-- 7. Backfill: each workspace owner gets a personal org; workspaces link to it;
--    existing workspace members become org members.
DO $$
DECLARE
    o RECORD;
    new_org_id UUID;
BEGIN
    FOR o IN SELECT DISTINCT owner_id FROM workspaces WHERE organization_id IS NULL LOOP
        INSERT INTO organizations (name, slug, owner_id)
        VALUES (
            'My Organization',
            'org-' || substr(md5(o.owner_id || clock_timestamp()::text), 1, 12),
            o.owner_id
        )
        RETURNING id INTO new_org_id;

        INSERT INTO organization_members (organization_id, user_id, role, status)
        VALUES (new_org_id, o.owner_id, 'owner', 'active')
        ON CONFLICT (organization_id, user_id) DO NOTHING;

        UPDATE workspaces
        SET organization_id = new_org_id
        WHERE owner_id = o.owner_id AND organization_id IS NULL;

        INSERT INTO organization_members (organization_id, user_id, role, status)
        SELECT DISTINCT new_org_id, wm.user_id, 'member', 'active'
        FROM workspace_members wm
        JOIN workspaces ws ON wm.workspace_id = ws.id
        WHERE ws.organization_id = new_org_id
          AND wm.user_id <> o.owner_id
          AND wm.status = 'active'
        ON CONFLICT (organization_id, user_id) DO NOTHING;
    END LOOP;
END $$;


-- Agent runs: execution history for Dalya/Dany OS agents, workspace-scoped.
-- Backs the Agents module (roster from team_members, runs from here). Mirrors
-- Len OS /api/agent/runs. Idempotent.
CREATE TABLE IF NOT EXISTS agent_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_name    VARCHAR(120),
    task_name     TEXT,
    company       VARCHAR(120),
    status        VARCHAR(40) DEFAULT 'pending',
    plan          TEXT,
    destination   TEXT,
    external_id   VARCHAR(120),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_runs_workspace ON agent_runs(workspace_id, status);

-- ===== Assets module: per-workspace brand/media library =====
CREATE TABLE IF NOT EXISTS assets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    kind         TEXT NOT NULL DEFAULT 'file',
    url          TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    mime_type    TEXT NOT NULL DEFAULT '',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    tags         TEXT NOT NULL DEFAULT '',
    created_by   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_assets_workspace ON assets(workspace_id);

-- ===== Intelligence module: cached AI synthesis per workspace =====
CREATE TABLE IF NOT EXISTS intelligence_reports (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    insights     JSONB NOT NULL DEFAULT '[]'::jsonb,
    model        TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===== Modules built 2026-07-11 (Drive/Agents/Builders/Engines/Resources) =====
CREATE TABLE IF NOT EXISTS drive_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    folder       TEXT NOT NULL DEFAULT '',
    mime_type    TEXT NOT NULL DEFAULT '',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    storage_path TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    created_by   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_drive_files_workspace ON drive_files (workspace_id);
CREATE INDEX IF NOT EXISTS idx_drive_files_folder ON drive_files (workspace_id, folder);
CREATE TABLE IF NOT EXISTS workspace_agents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT 'claude-sonnet-4-5-20250929',
    system_prompt TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active',
    created_by    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS workspace_agent_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id      UUID NOT NULL REFERENCES workspace_agents(id) ON DELETE CASCADE,
    input         TEXT NOT NULL DEFAULT '',
    output        TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'done',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_workspace_agents_workspace ON workspace_agents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_agent_runs_workspace ON workspace_agent_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_agent_runs_agent ON workspace_agent_runs(agent_id, created_at DESC);
CREATE TABLE IF NOT EXISTS builders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(300) NOT NULL,
    kind TEXT DEFAULT 'form',
    description TEXT DEFAULT '',
    config JSONB DEFAULT '{}',
    status TEXT DEFAULT 'draft',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_builders_workspace ON builders(workspace_id);
CREATE TABLE IF NOT EXISTS engines_workflows (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    trigger      TEXT NOT NULL DEFAULT 'manual',
    steps        JSONB NOT NULL DEFAULT '[]',
    status       TEXT NOT NULL DEFAULT 'active',
    created_by   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS engines_runs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    workflow_id  UUID NOT NULL REFERENCES engines_workflows(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'running',
    output       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_engines_workflows_workspace ON engines_workflows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_engines_runs_workspace ON engines_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_engines_runs_workflow ON engines_runs(workflow_id, created_at DESC);
CREATE TABLE IF NOT EXISTS resources (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT '',
    notes        TEXT NOT NULL DEFAULT '',
    created_by   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_resources_workspace ON resources(workspace_id);

-- ============================================================================
-- MIOSA workspace links (folded from 142_miosa_workspace_links.sql)
-- Raw MIOSA tenant key stays in credential_vault; this stores only
-- non-secret workspace attribution.
-- ============================================================================
CREATE TABLE IF NOT EXISTS miosa_workspace_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    miosa_workspace_id TEXT NOT NULL,
    external_workspace_id TEXT NOT NULL,
    external_user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'linked',
    created_by_user_id TEXT REFERENCES "user"(id) ON DELETE SET NULL,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id),
    UNIQUE (miosa_workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_miosa_workspace_links_external_workspace
    ON miosa_workspace_links(external_workspace_id);
CREATE INDEX IF NOT EXISTS idx_miosa_workspace_links_external_user
    ON miosa_workspace_links(external_user_id);

-- ============================================================================
-- 3D Desktop saved layouts (folded from 145_desktop3d_layouts.sql)
-- ============================================================================
CREATE TABLE IF NOT EXISTS desktop3d_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(40) NOT NULL DEFAULT 'custom' CHECK (type IN ('custom')),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    modules JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(modules) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_desktop3d_layouts_user
    ON desktop3d_layouts(user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_desktop3d_layouts_one_active
    ON desktop3d_layouts(user_id)
    WHERE is_active = TRUE;

-- ============================================================================
-- Knowledge cloud-sync opt-in gate (folded from 146_cloud_sync_activation.sql)
-- ============================================================================
ALTER TABLE workspace_storage
    ADD COLUMN IF NOT EXISTS cloud_sync_activated BOOLEAN NOT NULL DEFAULT false;

-- ============================================================================
-- Current module runtime contract
-- These relations depend on workspaces, so they must remain after the base
-- workspace table even though their primary modules are declared earlier.
-- ============================================================================
ALTER TABLE pipelines
    ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_workspace ON pipelines(workspace_id);

ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS pipeline_stage TEXT
        GENERATED ALWAYS AS (custom_fields ->> 'pipeline_stage') STORED,
    ADD COLUMN IF NOT EXISTS agency_type TEXT
        GENERATED ALWAYS AS (custom_fields ->> 'agency_type') STORED,
    ADD COLUMN IF NOT EXISTS outreach_status TEXT
        GENERATED ALWAYS AS (custom_fields ->> 'outreach_status') STORED,
    ADD COLUMN IF NOT EXISTS offer_fit TEXT
        GENERATED ALWAYS AS (custom_fields ->> 'offer_fit') STORED,
    ADD COLUMN IF NOT EXISTS fit_score INTEGER
        GENERATED ALWAYS AS (
            CASE
                WHEN (custom_fields ->> 'fit_score') ~ '^[0-9]+$'
                    THEN (custom_fields ->> 'fit_score')::INTEGER
                ELSE NULL
            END
        ) STORED,
    ADD COLUMN IF NOT EXISTS next_step_date TEXT
        GENERATED ALWAYS AS (NULLIF(custom_fields ->> 'next_step_date', '')) STORED;

CREATE INDEX IF NOT EXISTS idx_clients_workspace_pipeline_stage
    ON clients(workspace_id, pipeline_stage);
CREATE INDEX IF NOT EXISTS idx_clients_workspace_outreach_status
    ON clients(workspace_id, outreach_status);
CREATE INDEX IF NOT EXISTS idx_clients_workspace_fit_score
    ON clients(workspace_id, fit_score);

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS estimated_hours NUMERIC(10, 2);

CREATE TABLE IF NOT EXISTS deliverables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    kind VARCHAR(50) NOT NULL DEFAULT 'other'
        CHECK (kind IN ('package', 'document', 'deck', 'script', 'report', 'video', 'other')),
    status VARCHAR(50) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'in_progress', 'delivered')),
    client TEXT NOT NULL DEFAULT '',
    project TEXT NOT NULL DEFAULT '',
    link TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_deliverables_workspace_updated
    ON deliverables(workspace_id, updated_at DESC);

-- ============================================================================
-- FounderOS v1 bridge tables (spec criterion 3.2; design in docs/founderos/table-map.md)
--
-- 51 founderos_* tables that hold FounderOS v1's SQLite data after the ETL
-- (cmd/founderos-etl). No BusinessOS table is altered or dropped. Every
-- statement is IF NOT EXISTS, so this file is safe to re-apply and is mirrored
-- verbatim at the end of schema.sql.
--
-- Conventions (table-map.md "Conventions"):
--   * workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
--     indexed; resolved by slug by the ETL.
--   * The operator's own key is the primary key, verbatim (TEXT ids stay TEXT).
--   * imported_at is set by the ETL, not copied from the source.
--   * JSON-in-TEXT becomes JSONB; calendar days become DATE; instants become
--     TIMESTAMPTZ.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Org and agent roster
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_departments (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    tagline TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL,
    ord INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_departments_workspace ON founderos_departments(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_agents (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active', 'idle', 'training', 'planned')),
    tier TEXT NOT NULL CHECK (tier IN ('lead', 'specialist', 'worker')),
    description TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    tools JSONB NOT NULL DEFAULT '[]',
    parent_id TEXT,
    instance TEXT NOT NULL DEFAULT 'builtin',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agents_workspace ON founderos_agents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agents_department ON founderos_agents(department_id);

CREATE TABLE IF NOT EXISTS founderos_people (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    tools JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_people_workspace ON founderos_people(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_sop_tasks (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    steps JSONB NOT NULL DEFAULT '[]',
    assignee_kind TEXT NOT NULL CHECK (assignee_kind IN ('agent', 'person')),
    assignee_id TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_sop_tasks_workspace ON founderos_sop_tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_sop_tasks_assignee ON founderos_sop_tasks(assignee_kind, assignee_id);

CREATE TABLE IF NOT EXISTS founderos_tools (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('connected', 'available', 'planned')),
    color TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_tools_workspace ON founderos_tools(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_skills (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_agent_id TEXT,
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('live', 'learning', 'planned')),
    tools JSONB NOT NULL DEFAULT '[]',
    markdown TEXT NOT NULL DEFAULT '',
    ord INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_skills_workspace ON founderos_skills(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_personas (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    ord INTEGER NOT NULL,
    name TEXT NOT NULL,
    archetype TEXT NOT NULL,
    tagline TEXT NOT NULL,
    summary TEXT NOT NULL,
    accent TEXT NOT NULL,
    north_star TEXT NOT NULL,
    pillars JSONB NOT NULL DEFAULT '[]',
    connectors JSONB NOT NULL DEFAULT '[]',
    metrics JSONB NOT NULL DEFAULT '[]',
    brain_use TEXT NOT NULL,
    signature_play TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_personas_workspace ON founderos_personas(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_workflows (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    subtitle TEXT NOT NULL DEFAULT '',
    revenue_usd INTEGER NOT NULL DEFAULT 0,
    ord INTEGER NOT NULL DEFAULT 0,
    steps JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_workflows_workspace ON founderos_workflows(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_metrics (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL DEFAULT '',
    delta DOUBLE PRECISION NOT NULL DEFAULT 0,
    period TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_metrics_workspace ON founderos_metrics(workspace_id);

-- ----------------------------------------------------------------------------
-- Agent runtime
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_agent_runs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    ok BOOLEAN NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    model TEXT,
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost_usd DOUBLE PRECISION,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_workspace ON founderos_agent_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_started ON founderos_agent_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_agent ON founderos_agent_runs(agent_id, started_at DESC);

CREATE TABLE IF NOT EXISTS founderos_agent_messages (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content TEXT NOT NULL DEFAULT '',
    tool_calls JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_messages_workspace ON founderos_agent_messages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_messages_agent ON founderos_agent_messages(agent_id, created_at);

CREATE TABLE IF NOT EXISTS founderos_agent_tasks (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'doing', 'review', 'done')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_tasks_workspace ON founderos_agent_tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_tasks_agent ON founderos_agent_tasks(agent_id, status);

CREATE TABLE IF NOT EXISTS founderos_agent_crons (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    schedule TEXT NOT NULL,
    description TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_crons_workspace ON founderos_agent_crons(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_cron_runs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cron_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    ok BOOLEAN NOT NULL,
    summary TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_workspace ON founderos_cron_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_cron ON founderos_cron_runs(cron_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_started ON founderos_cron_runs(started_at DESC);

CREATE TABLE IF NOT EXISTS founderos_broadcasts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcasts_workspace ON founderos_broadcasts(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_broadcast_replies (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    broadcast_id TEXT NOT NULL REFERENCES founderos_broadcasts(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    ok BOOLEAN NOT NULL,
    reply TEXT NOT NULL DEFAULT '',
    finished_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcast_replies_workspace ON founderos_broadcast_replies(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcast_replies_broadcast ON founderos_broadcast_replies(broadcast_id);

-- ----------------------------------------------------------------------------
-- Comms and knowledge
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_comms_digests (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    generated_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_comms_digests_workspace ON founderos_comms_digests(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_comms_digests_generated ON founderos_comms_digests(generated_at DESC);

CREATE TABLE IF NOT EXISTS founderos_digest_reads (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_digest_reads_workspace ON founderos_digest_reads(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_digest_reads_read_at ON founderos_digest_reads(read_at);

CREATE TABLE IF NOT EXISTS founderos_contact_tags (
    person TEXT NOT NULL,
    channel TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    tier INTEGER NOT NULL CHECK (tier BETWEEN 1 AND 3),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (person, channel)
);
CREATE INDEX IF NOT EXISTS idx_founderos_contact_tags_workspace ON founderos_contact_tags(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_plaud_ingests (
    file_id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    recorded_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL,
    via TEXT NOT NULL CHECK (via IN ('gbrain', 'store')),
    slug TEXT NOT NULL,
    claims INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_plaud_ingests_workspace ON founderos_plaud_ingests(workspace_id);

-- ----------------------------------------------------------------------------
-- Social and content
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_social_accounts (
    platform TEXT PRIMARY KEY CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    handle TEXT NOT NULL,
    url TEXT,
    ord INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_accounts_workspace ON founderos_social_accounts(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_snapshots (
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    captured_on DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    followers INTEGER NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform, captured_on)
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_snapshots_workspace ON founderos_social_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dms (
    platform TEXT PRIMARY KEY CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    count INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dms_workspace ON founderos_social_dms(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dm_snapshots (
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    captured_on DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    count INTEGER NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform, captured_on)
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_snapshots_workspace ON founderos_social_dm_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dm_messages (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    subscriber_id TEXT NOT NULL,
    name TEXT NOT NULL,
    handle TEXT,
    text TEXT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    tag TEXT,
    ts TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_workspace ON founderos_social_dm_messages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_ts ON founderos_social_dm_messages(ts);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_subscriber ON founderos_social_dm_messages(platform, subscriber_id);

CREATE TABLE IF NOT EXISTS founderos_social_posts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    caption TEXT NOT NULL,
    media_url TEXT,
    platforms JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'published', 'failed')),
    scheduled_for TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_posts_workspace ON founderos_social_posts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_social_posts_status ON founderos_social_posts(status);

CREATE TABLE IF NOT EXISTS founderos_email_list_snapshots (
    captured_on DATE PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subscribers INTEGER NOT NULL,
    source TEXT NOT NULL,
    publication_id TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL DEFAULT '',
    quality TEXT NOT NULL DEFAULT 'ok' CHECK (quality IN ('ok', 'suspect')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_email_list_snapshots_workspace ON founderos_email_list_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_email_list_sync_misses (
    attempted_on DATE PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_email_list_sync_misses_workspace ON founderos_email_list_sync_misses(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_lead_magnets (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    offer TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('live', 'draft', 'paused', 'archived')),
    captures TEXT NOT NULL CHECK (captures IN ('email', 'booking', 'none')),
    destination TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    launched_at TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'os')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_lead_magnets_workspace ON founderos_lead_magnets(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_brand_deals (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    brand TEXT NOT NULL,
    status TEXT NOT NULL,
    tier TEXT,
    deal_value_usd DOUBLE PRECISION,
    budget_usd DOUBLE PRECISION,
    amount_agreed_usd DOUBLE PRECISION,
    suggested_rate_usd DOUBLE PRECISION,
    paid_in_full BOOLEAN NOT NULL DEFAULT false,
    deadline TIMESTAMPTZ,
    follow_up_date TIMESTAMPTZ,
    contact_name TEXT,
    contact_email TEXT,
    main_channel TEXT,
    video_type TEXT,
    source TEXT,
    icp_fit TEXT,
    notion_url TEXT NOT NULL,
    last_edited TIMESTAMPTZ NOT NULL,
    seeded BOOLEAN NOT NULL DEFAULT false,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_brand_deals_workspace ON founderos_brand_deals(workspace_id);

-- ----------------------------------------------------------------------------
-- Sales and clients
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_funnel_contacts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    venture TEXT NOT NULL CHECK (venture IN ('vantage', 'launchpad-cohort')),
    status TEXT NOT NULL CHECK (status IN ('first_touch', 'engaged', 'nurtured', 'opted_in', 'converted')),
    product TEXT,
    amount_usd DOUBLE PRECISION,
    relationship TEXT NOT NULL DEFAULT 'warm' CHECK (relationship IN ('cold', 'warm', 'hot')),
    likelihood INTEGER NOT NULL DEFAULT 50 CHECK (likelihood BETWEEN 0 AND 100),
    email TEXT,
    phone TEXT,
    person TEXT,
    company TEXT,
    role TEXT,
    linkedin TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_contacts_workspace ON founderos_funnel_contacts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_contacts_venture ON founderos_funnel_contacts(venture, status);

CREATE TABLE IF NOT EXISTS founderos_funnel_touches (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id TEXT NOT NULL REFERENCES founderos_funnel_contacts(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL CHECK (seq > 0),
    stage TEXT NOT NULL CHECK (stage IN ('first_touch', 'engaged', 'nurtured', 'opted_in', 'converted')),
    channel TEXT NOT NULL CHECK (channel IN ('organic', 'ads', 'dm', 'email', 'webinar', 'call', 'checkout', 'crm')),
    label TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('typeform', 'calendar', 'fathom', 'trakyo', 'meta-ads', 'stripe', 'manual', 'attio', 'ghl')),
    at DATE NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_touches_workspace ON founderos_funnel_touches(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_touches_contact ON founderos_funnel_touches(contact_id, seq);

CREATE TABLE IF NOT EXISTS founderos_proposals (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client TEXT NOT NULL,
    brand TEXT NOT NULL CHECK (brand IN ('vantage', 'launchpad-cohort')),
    url TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'sent', 'won', 'lost')),
    amount_usd DOUBLE PRECISION,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    origin TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'os')),
    -- The StatiCrypt code itself lives in credential_vault
    -- (provider_id 'founderos-staticrypt'); only its presence is recorded here.
    has_access_code BOOLEAN NOT NULL DEFAULT false,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_proposals_workspace ON founderos_proposals(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_deliverable_decisions (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'dismissed')),
    decided_at TIMESTAMPTZ NOT NULL,
    decided_revision TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_deliverable_decisions_workspace ON founderos_deliverable_decisions(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_client_work (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    brief TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('saved', 'launching', 'launched', 'needs_attention')),
    created_at TIMESTAMPTZ NOT NULL,
    superset_workspace_id TEXT,
    detail TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_client_work_workspace ON founderos_client_work(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_client_work_created ON founderos_client_work(created_at DESC);

CREATE TABLE IF NOT EXISTS founderos_vsl_snapshots (
    video_id TEXT NOT NULL,
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (video_id, date_from, date_to),
    CHECK (date_from <= date_to)
);
CREATE INDEX IF NOT EXISTS idx_founderos_vsl_snapshots_workspace ON founderos_vsl_snapshots(workspace_id);

-- ----------------------------------------------------------------------------
-- Trading
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_trading_snapshots (
    account_id TEXT NOT NULL DEFAULT 'individual',
    captured_at TIMESTAMPTZ NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_label TEXT NOT NULL DEFAULT 'Individual',
    account_value_usd DOUBLE PRECISION NOT NULL,
    buying_power_usd DOUBLE PRECISION NOT NULL,
    cash_usd DOUBLE PRECISION NOT NULL,
    day_pnl_usd DOUBLE PRECISION NOT NULL,
    total_pnl_usd DOUBLE PRECISION NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, captured_at)
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_snapshots_workspace ON founderos_trading_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_trading_positions (
    account_id TEXT NOT NULL DEFAULT 'individual',
    captured_at TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    quantity DOUBLE PRECISION NOT NULL,
    avg_cost_usd DOUBLE PRECISION NOT NULL,
    market_value_usd DOUBLE PRECISION NOT NULL,
    unrealized_pnl_usd DOUBLE PRECISION NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, captured_at, symbol)
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_positions_workspace ON founderos_trading_positions(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_trading_orders (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL DEFAULT 'agentic',
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    type TEXT NOT NULL,
    state TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    filled_quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    dollar_amount_usd DOUBLE PRECISION,
    limit_price_usd DOUBLE PRECISION,
    placed_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_orders_workspace ON founderos_trading_orders(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_orders_account ON founderos_trading_orders(account_id);

CREATE TABLE IF NOT EXISTS founderos_trading_analysis (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    at TIMESTAMPTZ NOT NULL,
    account_id TEXT NOT NULL DEFAULT 'agentic',
    agent TEXT NOT NULL,
    examined INTEGER NOT NULL DEFAULT 0,
    signals INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    rows JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_analysis_workspace ON founderos_trading_analysis(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_analysis_account ON founderos_trading_analysis(account_id, at DESC);

CREATE TABLE IF NOT EXISTS founderos_trading_activity (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    at TIMESTAMPTZ NOT NULL,
    account_id TEXT NOT NULL DEFAULT 'individual',
    agent TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('buy', 'sell')),
    symbol TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price_usd DOUBLE PRECISION NOT NULL,
    rationale TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('filled', 'pending', 'cancelled', 'rejected')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_activity_workspace ON founderos_trading_activity(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_activity_account ON founderos_trading_activity(account_id, at DESC);

-- One row per workspace (replaces the source's id = 1 singleton). autopilot is
-- always imported OFF by the ETL: the bridge never inherits a live trading switch.
CREATE TABLE IF NOT EXISTS founderos_trading_limits (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    max_notional_per_trade_usd DOUBLE PRECISION NOT NULL CHECK (max_notional_per_trade_usd > 0),
    max_position_pct_of_sleeve DOUBLE PRECISION NOT NULL CHECK (max_position_pct_of_sleeve > 0 AND max_position_pct_of_sleeve <= 100),
    max_risk_pct_per_trade DOUBLE PRECISION NOT NULL CHECK (max_risk_pct_per_trade > 0 AND max_risk_pct_per_trade <= 100),
    max_concurrent_positions INTEGER NOT NULL CHECK (max_concurrent_positions > 0),
    max_trades_per_day INTEGER NOT NULL CHECK (max_trades_per_day > 0),
    min_sleeve_value_usd DOUBLE PRECISION NOT NULL CHECK (min_sleeve_value_usd > 0),
    max_deployed_capital_usd DOUBLE PRECISION NOT NULL CHECK (max_deployed_capital_usd > 0),
    autopilot BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------------------
-- Analytics, usage and bookkeeping
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_metric_snapshots (
    metric_id TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (metric_id, captured_at)
);
CREATE INDEX IF NOT EXISTS idx_founderos_metric_snapshots_workspace ON founderos_metric_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_usage_snapshots (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_usage_snapshots_workspace ON founderos_usage_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_ollama_snapshots (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_ollama_snapshots_workspace ON founderos_ollama_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_seed_meta (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_seed_meta_workspace ON founderos_seed_meta(workspace_id);

-- ----------------------------------------------------------------------------
-- Side databases: bank.db, ledger.db, paykit.db, slack-bridge state.db
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_bank_summaries (
    account TEXT NOT NULL,
    month DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    business TEXT NOT NULL,
    credits_cents BIGINT NOT NULL,
    debits_cents BIGINT NOT NULL,
    net_cents BIGINT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account, month)
);
CREATE INDEX IF NOT EXISTS idx_founderos_bank_summaries_workspace ON founderos_bank_summaries(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_ledger_rows (
    hash TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    description TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    category TEXT NOT NULL,
    card TEXT NOT NULL DEFAULT 'platinum' CHECK (card IN ('gold', 'platinum', 'blue')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_ledger_rows_workspace ON founderos_ledger_rows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_ledger_rows_card_date ON founderos_ledger_rows(card, date);

CREATE TABLE IF NOT EXISTS founderos_paykit_customer_snapshots (
    account TEXT NOT NULL,
    captured_on DATE NOT NULL,
    customer_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    total_spent_cents BIGINT NOT NULL,
    total_transactions INTEGER NOT NULL,
    last_transaction_date TIMESTAMPTZ,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account, captured_on, customer_id)
);
CREATE INDEX IF NOT EXISTS idx_founderos_paykit_customer_snapshots_workspace ON founderos_paykit_customer_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_slack_bridge_jobs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    seq BIGINT NOT NULL,
    payload JSONB NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('queued', 'processing', 'done', 'uncertain')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_jobs_workspace ON founderos_slack_bridge_jobs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_jobs_state ON founderos_slack_bridge_jobs(state, seq);

CREATE TABLE IF NOT EXISTS founderos_slack_bridge_sessions (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_sessions_workspace ON founderos_slack_bridge_sessions(workspace_id);

-- 161: the operator memory writes land in Optimal Engine (GBrain retired).
-- Plaud ingests may now record via='oe', and Fathom call archiving gets its
-- own idempotency ledger. Idempotent: safe to re-apply (check-schema).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'founderos_plaud_ingests_via_check') THEN
        ALTER TABLE founderos_plaud_ingests DROP CONSTRAINT founderos_plaud_ingests_via_check;
    END IF;
    ALTER TABLE founderos_plaud_ingests
        ADD CONSTRAINT founderos_plaud_ingests_via_check CHECK (via IN ('gbrain', 'store', 'oe'));
END $$;

CREATE TABLE IF NOT EXISTS founderos_call_archive (
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    signal_id TEXT,
    entry JSONB NOT NULL DEFAULT '{}',
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source, external_id)
);
CREATE INDEX IF NOT EXISTS idx_founderos_call_archive_workspace ON founderos_call_archive(workspace_id);

-- 162: the latest device-collector push per Mac, so a bridge restart keeps
-- WhatsApp/Wispr/Obsidian/local-stack/usage readings instead of reading
-- "not configured" until the next push. Bridge-native (not an ETL target).
CREATE TABLE IF NOT EXISTS founderos_device_pushes (
    device TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_founderos_device_pushes_workspace ON founderos_device_pushes(workspace_id);

-- 163: PayKit last_transaction_date is kept as the raw text PayKit sent,
-- as FounderOS v1 stores it. As TIMESTAMPTZ, values PayKit sends without an
-- offset ("2026-10-01 09:00:00") became NULL and the rest read back rewritten.
-- Existing instants become RFC 3339 in PayKit's zone (what the bridge used
-- to return). Idempotent: a TEXT column is left alone.
DO $$
DECLARE
    tz TEXT := current_setting('TimeZone');
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'founderos_paykit_customer_snapshots'
                 AND column_name = 'last_transaction_date'
                 AND data_type <> 'text') THEN
        PERFORM set_config('TimeZone', 'America/Chicago', true);
        ALTER TABLE founderos_paykit_customer_snapshots
            ALTER COLUMN last_transaction_date TYPE TEXT
            USING to_char(last_transaction_date, 'YYYY-MM-DD"T"HH24:MI:SSTZH:TZM');
        PERFORM set_config('TimeZone', tz, true);
    END IF;
END $$;

-- 164: the retired CRM archive on /os/funnel (FounderOS v1 funnel_archive,
-- docs/funnel-crm-archive.md): Attio + GoHighLevel journeys restored
-- 2026-09-26, each stored whole as the FunnelJourney JSON. ETL target of
-- FounderOS v1's SQLite funnel_archive (id, venture, journey TEXT); the
-- workspace is the journey's venture workspace. Read-only for the page.
CREATE TABLE IF NOT EXISTS founderos_funnel_archive (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    venture TEXT NOT NULL CHECK (venture IN ('vantage', 'launchpad-cohort')),
    journey JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_archive_workspace ON founderos_funnel_archive(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_archive_venture ON founderos_funnel_archive(venture);

-- 165: /os/roadmap and /os/reference (FounderOS v1 app/roadmap, app/reference).
-- ETL targets of FounderOS v1's SQLite phases, roadmap_items and domains,
-- which the ETL used to drop as retired. All three live in the FounderOS HQ
-- workspace; source keys are kept verbatim. The CHECKs mirror v1's Zod
-- (lib/schemas.ts RoadmapStatusSchema and the 2026-Q2 quarter shape).
-- phase_id and department_id are soft references, as in v1. Idempotent.
CREATE TABLE IF NOT EXISTS founderos_phases (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    items JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_phases_workspace ON founderos_phases(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_roadmap_items (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    quarter TEXT NOT NULL CHECK (quarter ~ '^[0-9]{4}-Q[1-4]$'),
    status TEXT NOT NULL CHECK (status IN ('done', 'now', 'next', 'later')),
    department_id TEXT,
    description TEXT NOT NULL DEFAULT '',
    phase_id TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_roadmap_items_workspace ON founderos_roadmap_items(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_domains (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    color TEXT NOT NULL,
    items JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_domains_workspace ON founderos_domains(workspace_id);
