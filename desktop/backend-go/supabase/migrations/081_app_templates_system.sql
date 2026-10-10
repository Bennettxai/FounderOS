-- ================================================================
-- Migration: 081_app_templates_system.sql
-- Description: App templates system for personalized app generation
-- Created: 2026-01-20
-- ================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================================
-- TABLE: app_templates
-- Description: Blueprints for generating apps based on user profile
-- ================================================================
CREATE TABLE IF NOT EXISTS app_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL, -- 'productivity', 'crm', 'analytics', 'communication', 'finance'
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    icon_type VARCHAR(50), -- Icon identifier for frontend (lucide icon name)

    -- Targeting criteria (arrays for flexible matching)
    target_business_types TEXT[], -- ['agency', 'startup', 'freelance', 'ecommerce', 'consulting', 'saas']
    target_challenges TEXT[], -- ['time_management', 'client_communication', 'project_management', 'sales', 'growth']
    target_team_sizes TEXT[], -- ['solo', '2-5', '6-15', '16-50', '50+']
    priority_score INT DEFAULT 50, -- Base priority (0-100, higher = more likely to be generated)

    -- Template configuration
    template_config JSONB DEFAULT '{}'::jsonb, -- App-specific configuration
    required_modules TEXT[] DEFAULT '{}', -- Module dependencies
    optional_features TEXT[] DEFAULT '{}', -- Optional features that can be toggled

    -- Generation metadata
    generation_prompt TEXT, -- AI prompt for generating this app via OSA
    scaffold_type VARCHAR(50) DEFAULT 'svelte', -- 'react', 'svelte', 'static', 'iframe'

    -- Audit timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT app_templates_category_check CHECK (category IN ('productivity', 'crm', 'analytics', 'communication', 'finance', 'marketing', 'operations'))
);

-- ================================================================
-- TABLE: user_generated_apps
-- Description: Track what apps were generated for each user
-- ================================================================
CREATE TABLE IF NOT EXISTS user_generated_apps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id UUID REFERENCES app_templates(id) ON DELETE SET NULL,
    app_name VARCHAR(200) NOT NULL,

    -- Link to OSA generated app if applicable
    osa_app_id UUID REFERENCES osa_generated_apps(id) ON DELETE SET NULL,

    -- Visibility and state
    is_visible BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_favorite BOOLEAN DEFAULT FALSE,
    position_index INT, -- For custom ordering in UI

    -- Customization
    custom_config JSONB DEFAULT '{}'::jsonb, -- User-specific overrides
    custom_icon VARCHAR(50), -- Override default icon

    -- Usage tracking
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    access_count INT DEFAULT 0,

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE(workspace_id, app_name)
);

-- ================================================================
-- TABLE: app_generation_queue
-- Description: Track pending app generation tasks
-- ================================================================
CREATE TABLE IF NOT EXISTS app_generation_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES app_templates(id) ON DELETE CASCADE,

    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    priority INT DEFAULT 50, -- Higher priority processed first

    -- Generation context (snapshot of onboarding data at time of queuing)
    generation_context JSONB DEFAULT '{}'::jsonb,

    -- Error handling
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT app_generation_queue_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- ================================================================
-- INDEXES
-- ================================================================

-- app_templates indexes
CREATE INDEX idx_app_templates_business_types ON app_templates USING GIN(target_business_types);
CREATE INDEX idx_app_templates_challenges ON app_templates USING GIN(target_challenges);
CREATE INDEX idx_app_templates_team_sizes ON app_templates USING GIN(target_team_sizes);
CREATE INDEX idx_app_templates_category ON app_templates(category);
CREATE INDEX idx_app_templates_priority ON app_templates(priority_score DESC);

-- user_generated_apps indexes
CREATE INDEX idx_user_generated_apps_workspace ON user_generated_apps(workspace_id);
CREATE INDEX idx_user_generated_apps_template ON user_generated_apps(template_id);
CREATE INDEX idx_user_generated_apps_osa_app ON user_generated_apps(osa_app_id);
CREATE INDEX idx_user_generated_apps_visibility ON user_generated_apps(workspace_id, is_visible) WHERE is_visible = TRUE;
CREATE INDEX idx_user_generated_apps_position ON user_generated_apps(workspace_id, position_index);

-- app_generation_queue indexes
CREATE INDEX idx_app_generation_queue_status ON app_generation_queue(status, priority DESC);
CREATE INDEX idx_app_generation_queue_workspace ON app_generation_queue(workspace_id);
CREATE INDEX idx_app_generation_queue_pending ON app_generation_queue(workspace_id, status) WHERE status = 'pending';

-- ================================================================
-- SEED DATA: Default App Templates
-- ================================================================

INSERT INTO app_templates (
    template_name,
    category,
    display_name,
    description,
    icon_type,
    target_business_types,
    target_challenges,
    target_team_sizes,
    priority_score,
    template_config,
    scaffold_type,
    generation_prompt
) VALUES
-- Template 1: Client Portal (High Priority for Agencies)
(
    'client_portal',
    'communication',
    'Client Portal',
    'Secure portal for client communication, project updates, and file sharing',
    'users',
    ARRAY['agency', 'consulting', 'freelance']::TEXT[],
    ARRAY['client_communication', 'project_management']::TEXT[],
    ARRAY['solo', '2-5', '6-15']::TEXT[],
    90,
    '{"features": ["messaging", "file_sharing", "project_status", "invoices"], "security": "high"}'::jsonb,
    'svelte',
    'Generate a secure client portal with real-time messaging, file sharing, project status tracking, and invoice management. Use Svelte for reactive UI, implement role-based access control, and integrate with existing workspace authentication.'
),

-- Template 2: Time Tracker (Essential for Billable Work)
(
    'time_tracker',
    'productivity',
    'Time Tracker',
    'Track time spent on projects, generate reports, and create invoices',
    'clock',
    ARRAY['agency', 'freelance', 'consulting']::TEXT[],
    ARRAY['time_management', 'billing', 'productivity']::TEXT[],
    ARRAY['solo', '2-5', '6-15']::TEXT[],
    85,
    '{"features": ["timer", "manual_entry", "reporting", "invoicing", "project_breakdown"], "integrations": ["calendar"]}'::jsonb,
    'svelte',
    'Generate a time tracking application with start/stop timer, manual time entry, detailed reporting by project and client, and invoice generation. Include calendar integration and export capabilities.'
),

-- Template 3: Analytics Dashboard (Data-Driven Businesses)
(
    'analytics_dashboard',
    'analytics',
    'Analytics Dashboard',
    'Business metrics, KPI visualization, and performance tracking',
    'chart-bar',
    ARRAY['startup', 'ecommerce', 'saas', 'agency']::TEXT[],
    ARRAY['growth', 'metrics', 'data_analysis']::TEXT[],
    ARRAY['2-5', '6-15', '16-50']::TEXT[],
    80,
    '{"features": ["charts", "custom_metrics", "export", "real_time"], "chart_types": ["line", "bar", "pie", "area"]}'::jsonb,
    'react',
    'Generate an analytics dashboard with customizable charts, real-time KPI tracking, metric alerts, and data export. Use React with chart libraries, support custom metric definitions, and provide drill-down capabilities.'
),

-- Template 4: Project Kanban Board
(
    'project_kanban',
    'productivity',
    'Project Kanban',
    'Visual project management with kanban boards and task tracking',
    'layout-kanban',
    ARRAY['agency', 'startup', 'software', 'consulting']::TEXT[],
    ARRAY['project_management', 'team_collaboration', 'workflow']::TEXT[],
    ARRAY['2-5', '6-15', '16-50']::TEXT[],
    75,
    '{"features": ["boards", "cards", "assignments", "deadlines", "labels", "comments"], "views": ["kanban", "list", "calendar"]}'::jsonb,
    'svelte',
    'Generate a kanban board project management tool with drag-and-drop cards, task assignments, deadlines, labels, and comments. Support multiple boards, swimlanes, and views (kanban, list, calendar).'
),

-- Template 5: Lead Manager (Sales-Focused)
(
    'lead_manager',
    'crm',
    'Lead Manager',
    'Track and manage sales leads through pipeline',
    'user-plus',
    ARRAY['agency', 'ecommerce', 'consulting', 'saas']::TEXT[],
    ARRAY['sales', 'client_acquisition', 'growth']::TEXT[],
    ARRAY['solo', '2-5', '6-15']::TEXT[],
    70,
    '{"features": ["pipeline", "contact_info", "notes", "tasks", "email_integration"], "stages": ["lead", "qualified", "proposal", "negotiation", "closed"]}'::jsonb,
    'react',
    'Generate a lead management system with visual pipeline, contact management, activity tracking, notes, and task creation. Include email integration and customizable pipeline stages.'
),

-- Template 6: Team Calendar
(
    'team_calendar',
    'productivity',
    'Team Calendar',
    'Shared calendar for team scheduling and availability',
    'calendar',
    ARRAY['agency', 'startup', 'consulting']::TEXT[],
    ARRAY['team_collaboration', 'scheduling']::TEXT[],
    ARRAY['2-5', '6-15', '16-50']::TEXT[],
    65,
    '{"features": ["shared_calendar", "events", "availability", "reminders", "integrations"], "views": ["month", "week", "day"]}'::jsonb,
    'svelte',
    'Generate a shared team calendar with event creation, availability tracking, reminders, and calendar integrations. Support multiple views and timezone handling.'
),

-- Template 7: Invoice Generator
(
    'invoice_generator',
    'finance',
    'Invoice Generator',
    'Create and send professional invoices to clients',
    'file-text',
    ARRAY['freelance', 'agency', 'consulting']::TEXT[],
    ARRAY['billing', 'finance', 'client_communication']::TEXT[],
    ARRAY['solo', '2-5']::TEXT[],
    70,
    '{"features": ["invoice_creation", "pdf_export", "email_sending", "payment_tracking", "templates"], "currency": "multi"}'::jsonb,
    'react',
    'Generate an invoice creation tool with customizable templates, PDF export, email sending, payment tracking, and multi-currency support. Include recurring invoice functionality.'
),

-- Template 8: Knowledge Base
(
    'knowledge_base',
    'communication',
    'Knowledge Base',
    'Internal wiki and documentation system',
    'book-open',
    ARRAY['agency', 'startup', 'software', 'saas']::TEXT[],
    ARRAY['documentation', 'team_collaboration', 'knowledge_sharing']::TEXT[],
    ARRAY['2-5', '6-15', '16-50', '50+']::TEXT[],
    60,
    '{"features": ["wiki", "search", "categories", "permissions", "versioning"], "editor": "markdown"}'::jsonb,
    'svelte',
    'Generate a knowledge base system with wiki-style pages, full-text search, categorization, permission controls, and version history. Use markdown editor with rich formatting.'
),

-- Template 9: Feedback Collector
(
    'feedback_collector',
    'communication',
    'Feedback Collector',
    'Gather and manage customer feedback',
    'message-circle',
    ARRAY['startup', 'saas', 'ecommerce']::TEXT[],
    ARRAY['customer_feedback', 'product_development', 'user_research']::TEXT[],
    ARRAY['solo', '2-5', '6-15']::TEXT[],
    55,
    '{"features": ["feedback_forms", "voting", "status_tracking", "categorization"], "public_board": true}'::jsonb,
    'react',
    'Generate a feedback collection system with customizable forms, voting, status tracking, categorization, and public feedback boards. Include prioritization features.'
),

-- Template 10: Expense Tracker
(
    'expense_tracker',
    'finance',
    'Expense Tracker',
    'Track business expenses and generate reports',
    'receipt',
    ARRAY['freelance', 'agency', 'consulting', 'startup']::TEXT[],
    ARRAY['finance', 'budgeting', 'tax_preparation']::TEXT[],
    ARRAY['solo', '2-5', '6-15']::TEXT[],
    65,
    '{"features": ["expense_entry", "receipts", "categories", "reports", "export"], "currency": "multi"}'::jsonb,
    'svelte',
    'Generate an expense tracking application with receipt uploads, categorization, reporting, and export for accounting. Support multiple currencies and tax categories.'
);

-- ================================================================
-- FUNCTIONS: Auto-update timestamp
-- ================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_app_templates_updated_at
    BEFORE UPDATE ON app_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_generated_apps_updated_at
    BEFORE UPDATE ON user_generated_apps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================================
-- COMMENTS
-- ================================================================

COMMENT ON TABLE app_templates IS 'Blueprints for generating personalized apps based on user onboarding profile';
COMMENT ON TABLE user_generated_apps IS 'Tracks apps that were generated for each workspace with customization and usage data';
COMMENT ON TABLE app_generation_queue IS 'Queue for background processing of app generation requests';

COMMENT ON COLUMN app_templates.target_business_types IS 'Array of business types this template is designed for (agency, startup, freelance, etc.)';
COMMENT ON COLUMN app_templates.target_challenges IS 'Array of challenges this template helps solve';
COMMENT ON COLUMN app_templates.priority_score IS 'Base priority score (0-100) - higher scores more likely to be selected';
COMMENT ON COLUMN app_templates.generation_prompt IS 'AI prompt sent to OSA for generating this app';

COMMENT ON COLUMN user_generated_apps.custom_config IS 'User-specific customization overrides (features enabled, theme, etc.)';
COMMENT ON COLUMN user_generated_apps.position_index IS 'Custom ordering for desktop icon display';

COMMENT ON COLUMN app_generation_queue.generation_context IS 'Snapshot of user onboarding data at time of generation request';
COMMENT ON COLUMN app_generation_queue.priority IS 'Higher priority items processed first in queue';
