-- Migration: 025_integrations_module.sql
-- Adds comprehensive integrations support for BusinessOS + Sorx
-- This creates a unified integration system where:
-- 1. Users connect external tools (OAuth)
-- 2. Each module shows relevant integrations
-- 3. Sorx skills can use these connections

-- Integration Providers (system-defined catalog of available integrations)
CREATE TABLE IF NOT EXISTS integration_providers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,  -- communication, crm, tasks, calendar, storage, meetings, finance, code, ai
    icon_url TEXT,

    -- OAuth Configuration (stored as JSON for flexibility)
    oauth_config JSONB NOT NULL DEFAULT '{}',

    -- Which modules can use this integration
    modules TEXT[] NOT NULL DEFAULT '{}',

    -- Available skills from Sorx
    skills TEXT[] NOT NULL DEFAULT '{}',

    -- Status
    status VARCHAR(20) DEFAULT 'available',  -- available, coming_soon, beta, deprecated

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Integration Connections (when a user connects their account)
CREATE TABLE IF NOT EXISTS user_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider_id VARCHAR(50) NOT NULL REFERENCES integration_providers(id),

    -- Connection Status
    status VARCHAR(20) DEFAULT 'connected',  -- connected, disconnected, expired, error
    connected_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,

    -- OAuth Data (encrypted at application level before storing)
    access_token_encrypted BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    scopes TEXT[],

    -- External account info
    external_account_id VARCHAR(255),
    external_account_name VARCHAR(255),
    external_workspace_id VARCHAR(255),
    external_workspace_name VARCHAR(255),
    metadata JSONB DEFAULT '{}',

    -- User settings for this integration
    settings JSONB DEFAULT '{"enabledSkills": [], "notifications": true}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, provider_id)
);

-- Module Integration Settings (per-user, per-module customization)
CREATE TABLE IF NOT EXISTS module_integration_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    module_id VARCHAR(50) NOT NULL,  -- dashboard, tasks, projects, clients, etc.
    provider_id VARCHAR(50) NOT NULL REFERENCES integration_providers(id),

    -- Settings
    enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'bidirectional',  -- import, export, bidirectional
    sync_frequency VARCHAR(20) DEFAULT 'realtime',  -- realtime, hourly, daily, manual
    custom_settings JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, module_id, provider_id)
);

-- User Model Preferences (for AI tier selection)
CREATE TABLE IF NOT EXISTS user_model_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE UNIQUE,

    -- Tier assignments (user selects model for each tier)
    tier_2_model JSONB DEFAULT '{"model_id": "claude-3-5-haiku", "provider": "anthropic"}',
    tier_3_model JSONB DEFAULT '{"model_id": "claude-sonnet-4", "provider": "anthropic"}',
    tier_4_model JSONB DEFAULT '{"model_id": "claude-opus-4", "provider": "anthropic"}',

    -- Fallback chains
    tier_2_fallbacks JSONB DEFAULT '[]',
    tier_3_fallbacks JSONB DEFAULT '[]',
    tier_4_fallbacks JSONB DEFAULT '[]',

    -- Skill-specific overrides
    skill_overrides JSONB DEFAULT '{}',

    -- Behavior preferences
    allow_model_upgrade_on_failure BOOLEAN DEFAULT true,
    max_latency_ms INTEGER DEFAULT 30000,
    prefer_local BOOLEAN DEFAULT false,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pending Decisions (for human-in-the-loop skill execution)
CREATE TABLE IF NOT EXISTS pending_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id VARCHAR(255) NOT NULL,
    skill_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- The decision request
    question TEXT NOT NULL,
    description TEXT,
    options TEXT[],
    input_fields JSONB,
    context JSONB,

    -- Priority and timing
    priority VARCHAR(20) DEFAULT 'medium',  -- low, medium, high, urgent
    status VARCHAR(20) DEFAULT 'pending',  -- pending, decided, expired, cancelled

    -- Response (when decided)
    decision TEXT,
    decision_inputs JSONB,
    decided_by VARCHAR(255) REFERENCES "user"(id),
    decided_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Integration Sync Log (for tracking sync operations)
CREATE TABLE IF NOT EXISTS integration_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id UUID NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    module_id VARCHAR(50),

    -- Sync details
    sync_type VARCHAR(50) NOT NULL,
    direction VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,  -- pending, running, success, failed

    -- Statistics
    records_processed INT DEFAULT 0,
    records_created INT DEFAULT 0,
    records_updated INT DEFAULT 0,
    records_failed INT DEFAULT 0,

    -- Error info
    error_message TEXT,
    error_details JSONB,

    -- Timestamps
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Skill Executions (track Sorx skill runs)
CREATE TABLE IF NOT EXISTS skill_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Execution state
    status VARCHAR(20) NOT NULL,  -- pending, running, waiting_callback, complete, failed, cancelled
    current_step INT DEFAULT 0,

    -- Input/Output
    params JSONB DEFAULT '{}',
    result JSONB,
    error TEXT,

    -- Context
    context JSONB DEFAULT '{}',
    step_results JSONB DEFAULT '{}',

    -- Metrics
    metrics JSONB DEFAULT '{}',

    -- Timestamps
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_integrations_user ON user_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_provider ON user_integrations(provider_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_status ON user_integrations(status);

CREATE INDEX IF NOT EXISTS idx_module_integration_settings_user ON module_integration_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_module_integration_settings_module ON module_integration_settings(module_id);

CREATE INDEX IF NOT EXISTS idx_pending_decisions_user_status ON pending_decisions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_pending_decisions_execution ON pending_decisions(execution_id);
CREATE INDEX IF NOT EXISTS idx_pending_decisions_expires ON pending_decisions(expires_at) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_integration_sync_log_integration ON integration_sync_log(user_integration_id);
CREATE INDEX IF NOT EXISTS idx_integration_sync_log_started ON integration_sync_log(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_skill_executions_user ON skill_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_skill_executions_status ON skill_executions(status);

-- Seed integration providers
INSERT INTO integration_providers (id, name, description, category, modules, skills, status) VALUES
-- Communication
('slack', 'Slack', 'Team messaging and collaboration', 'communication',
 ARRAY['chat', 'tasks', 'projects', 'team', 'daily_log'],
 ARRAY['slack.send_message', 'slack.message_to_task', 'slack.create_channel'],
 'available'),

('teams', 'Microsoft Teams', 'Microsoft team collaboration', 'communication',
 ARRAY['chat', 'tasks', 'projects', 'team'],
 ARRAY['teams.send_message', 'teams.create_channel'],
 'coming_soon'),

('discord', 'Discord', 'Community messaging', 'communication',
 ARRAY['chat', 'team'],
 ARRAY['discord.send_message'],
 'coming_soon'),

-- CRM
('hubspot', 'HubSpot', 'CRM and sales automation', 'crm',
 ARRAY['clients', 'projects', 'tasks'],
 ARRAY['hubspot.qualify_lead', 'hubspot.deal_won_onboarding', 'hubspot.sync_contacts'],
 'available'),

('salesforce', 'Salesforce', 'Enterprise CRM', 'crm',
 ARRAY['clients', 'projects'],
 ARRAY['salesforce.sync_contacts', 'salesforce.sync_opportunities'],
 'coming_soon'),

('pipedrive', 'Pipedrive', 'Sales CRM', 'crm',
 ARRAY['clients', 'projects'],
 ARRAY['pipedrive.sync_deals'],
 'coming_soon'),

-- Tasks/Projects
('clickup', 'ClickUp', 'Project and task management', 'tasks',
 ARRAY['tasks', 'projects'],
 ARRAY['clickup.sync_tasks', 'clickup.create_task', 'clickup.sync_projects'],
 'available'),

('asana', 'Asana', 'Work management platform', 'tasks',
 ARRAY['tasks', 'projects'],
 ARRAY['asana.sync_tasks', 'asana.create_task'],
 'coming_soon'),

('linear', 'Linear', 'Modern project management', 'tasks',
 ARRAY['tasks', 'projects'],
 ARRAY['linear.sync_issues', 'linear.create_issue'],
 'coming_soon'),

('jira', 'Jira', 'Issue and project tracking', 'tasks',
 ARRAY['tasks', 'projects'],
 ARRAY['jira.sync_issues'],
 'coming_soon'),

-- Calendar
('google_calendar', 'Google Calendar', 'Google calendar integration', 'calendar',
 ARRAY['calendar', 'daily_log', 'projects'],
 ARRAY['google_calendar.sync_daily_log', 'google_calendar.create_event', 'google_calendar.get_events'],
 'available'),

('outlook', 'Outlook Calendar', 'Microsoft Outlook calendar', 'calendar',
 ARRAY['calendar', 'daily_log'],
 ARRAY['outlook.sync_events'],
 'coming_soon'),

('calendly', 'Calendly', 'Scheduling automation', 'calendar',
 ARRAY['calendar', 'clients'],
 ARRAY['calendly.sync_bookings'],
 'coming_soon'),

-- Storage/Docs
('notion', 'Notion', 'Connected workspace', 'storage',
 ARRAY['contexts', 'projects', 'tasks'],
 ARRAY['notion.sync_pages', 'notion.create_page', 'notion.search'],
 'available'),

('google_drive', 'Google Drive', 'Cloud file storage', 'storage',
 ARRAY['contexts', 'projects'],
 ARRAY['google_drive.list_files', 'google_drive.upload_file'],
 'coming_soon'),

-- Meetings
('fathom', 'Fathom', 'AI meeting notes', 'meetings',
 ARRAY['daily_log', 'contexts', 'tasks'],
 ARRAY['fathom.process_meeting', 'fathom.get_transcript'],
 'available'),

('fireflies', 'Fireflies.ai', 'Meeting transcription', 'meetings',
 ARRAY['daily_log', 'contexts', 'tasks'],
 ARRAY['fireflies.process_meeting'],
 'coming_soon'),

('zoom', 'Zoom', 'Video meetings', 'meetings',
 ARRAY['calendar', 'daily_log'],
 ARRAY['zoom.create_meeting', 'zoom.get_recordings'],
 'coming_soon'),

-- Finance
('stripe', 'Stripe', 'Payment processing', 'finance',
 ARRAY['clients', 'projects'],
 ARRAY['stripe.sync_customers', 'stripe.sync_payments'],
 'coming_soon'),

('quickbooks', 'QuickBooks', 'Accounting software', 'finance',
 ARRAY['clients', 'projects'],
 ARRAY['quickbooks.sync_invoices'],
 'coming_soon'),

-- Code
('github', 'GitHub', 'Code hosting and collaboration', 'code',
 ARRAY['tasks', 'projects'],
 ARRAY['github.sync_issues', 'github.create_issue'],
 'coming_soon'),

('gitlab', 'GitLab', 'DevOps platform', 'code',
 ARRAY['tasks', 'projects'],
 ARRAY['gitlab.sync_issues'],
 'coming_soon'),

-- AI Models (special category)
('anthropic', 'Anthropic', 'Claude AI models', 'ai',
 ARRAY['ai'],
 ARRAY[]::TEXT[],
 'available'),

('openai', 'OpenAI', 'GPT models', 'ai',
 ARRAY['ai'],
 ARRAY[]::TEXT[],
 'available'),

('groq', 'Groq', 'Fast inference', 'ai',
 ARRAY['ai'],
 ARRAY[]::TEXT[],
 'available'),

('ollama', 'Ollama', 'Local models', 'ai',
 ARRAY['ai'],
 ARRAY[]::TEXT[],
 'available')

ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    modules = EXCLUDED.modules,
    skills = EXCLUDED.skills,
    status = EXCLUDED.status,
    updated_at = NOW();
