
-- Note: workspace_id FK constraint deferred to migration 029 (after workspaces table is created)
CREATE TABLE IF NOT EXISTS user_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID,  -- FK constraint added in 029_dashboards_workspace_fk.sql

    -- Identity
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    
    -- Layout (JSON array of widget configs)
    -- Structure: [{ widget_id, widget_type, position: {x,y,w,h}, config: {} }]
    layout JSONB NOT NULL DEFAULT '[]',
    
    -- Sharing
    visibility VARCHAR(50) DEFAULT 'private' CHECK (visibility IN ('private', 'workspace', 'public_link')),
    share_token VARCHAR(100) UNIQUE,
    is_enforced BOOLEAN DEFAULT FALSE,
    enforced_for_roles TEXT[],
    
    -- Metadata
    created_via VARCHAR(50) DEFAULT 'agent' CHECK (created_via IN ('agent', 'manual', 'template')),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for user_dashboards
CREATE INDEX IF NOT EXISTS idx_dashboards_user ON user_dashboards(user_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_workspace ON user_dashboards(workspace_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_share_token ON user_dashboards(share_token) WHERE share_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_dashboards_default ON user_dashboards(user_id, is_default) WHERE is_default = TRUE;

-- ============================================================================
-- DASHBOARD WIDGETS TABLE (Widget Type Registry)
-- Defines available widget types and their configuration schemas
-- ============================================================================
CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    widget_type VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    
    -- Configuration
    config_schema JSONB NOT NULL,           -- JSON Schema for widget config validation
    default_config JSONB DEFAULT '{}',      -- Default config values
    default_size JSONB DEFAULT '{"w": 4, "h": 3}',
    min_size JSONB DEFAULT '{"w": 2, "h": 2}',
    
    -- Real-time subscriptions
    sse_events TEXT[],                      -- Events this widget listens to
    
    -- Availability
    is_enabled BOOLEAN DEFAULT TRUE,
    requires_feature VARCHAR(100),          -- Feature flag dependency (e.g., 'team', 'client_portal')
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- DASHBOARD TEMPLATES TABLE
-- Predefined dashboard templates users can clone
-- ============================================================================
CREATE TABLE IF NOT EXISTS dashboard_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    
    -- Layout configuration (same structure as user_dashboards.layout)
    layout JSONB NOT NULL,
    
    -- Display
    thumbnail_url TEXT,
    is_default BOOLEAN DEFAULT FALSE,       -- Show as "recommended" template
    sort_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- SEED DATA: Widget Types
-- ============================================================================
INSERT INTO dashboard_widgets (widget_type, name, description, category, config_schema, default_config, default_size, min_size, sse_events, is_enabled) VALUES

-- Core Widgets (Priority 1)
('task_summary', 'Task Summary', 'Tasks grouped by status (todo/in-progress/done)', 'tasks', 
 '{"type": "object", "properties": {"group_by": {"type": "string", "enum": ["status", "project", "priority"]}, "show_completed": {"type": "boolean"}}}',
 '{"group_by": "status", "show_completed": false}',
 '{"w": 4, "h": 3}', '{"w": 2, "h": 2}',
 ARRAY['task.created', 'task.updated', 'task.deleted'], TRUE),

('task_list', 'Task List', 'Filterable, sortable task list', 'tasks',
 '{"type": "object", "properties": {"filter": {"type": "object"}, "sort": {"type": "string"}, "limit": {"type": "integer"}}}',
 '{"filter": {}, "sort": "due_date", "limit": 10}',
 '{"w": 6, "h": 4}', '{"w": 3, "h": 2}',
 ARRAY['task.created', 'task.updated', 'task.deleted'], TRUE),

('project_progress', 'Project Progress', 'Project completion percentage bars', 'projects',
 '{"type": "object", "properties": {"project_ids": {"type": "array", "items": {"type": "string"}}, "show_all": {"type": "boolean"}}}',
 '{"show_all": true}',
 '{"w": 4, "h": 3}', '{"w": 2, "h": 2}',
 ARRAY['task.completed', 'project.updated'], TRUE),

('upcoming_deadlines', 'Upcoming Deadlines', 'Tasks due in the next X days', 'tasks',
 '{"type": "object", "properties": {"days_ahead": {"type": "integer"}, "show_overdue": {"type": "boolean"}}}',
 '{"days_ahead": 7, "show_overdue": true}',
 '{"w": 4, "h": 4}', '{"w": 2, "h": 2}',
 ARRAY['task.created', 'task.updated', 'task.deleted'], TRUE),

('recent_activity', 'Recent Activity', 'Activity feed of recent actions', 'activity',
 '{"type": "object", "properties": {"limit": {"type": "integer"}, "types": {"type": "array", "items": {"type": "string"}}}}',
 '{"limit": 10, "types": []}',
 '{"w": 4, "h": 4}', '{"w": 2, "h": 3}',
 ARRAY['activity.*'], TRUE),

('metric_card', 'Metric Card', 'Single number KPI display', 'analytics',
 '{"type": "object", "properties": {"metric": {"type": "string", "enum": ["tasks_due_today", "tasks_overdue", "tasks_completed_week", "active_projects"]}, "comparison_period": {"type": "string"}}}',
 '{"metric": "tasks_due_today", "comparison_period": "yesterday"}',
 '{"w": 2, "h": 2}', '{"w": 2, "h": 2}',
 ARRAY['task.*', 'project.*'], TRUE),

-- Extended Widgets (Priority 2)
('task_burndown', 'Task Burndown', 'Chart showing tasks completed over time', 'analytics',
 '{"type": "object", "properties": {"project_id": {"type": "string"}, "days": {"type": "integer"}}}',
 '{"days": 30}',
 '{"w": 6, "h": 4}', '{"w": 4, "h": 3}',
 ARRAY['task.created', 'task.completed'], TRUE),

('workload_heatmap', 'Workload Heatmap', 'Calendar heatmap of task density', 'analytics',
 '{"type": "object", "properties": {"date_range": {"type": "string", "enum": ["month", "quarter", "year"]}}}',
 '{"date_range": "month"}',
 '{"w": 6, "h": 3}', '{"w": 4, "h": 2}',
 ARRAY['task.created', 'task.updated'], TRUE),

('client_overview', 'Client Overview', 'Client cards with project/task counts', 'clients',
 '{"type": "object", "properties": {"limit": {"type": "integer"}, "sort_by": {"type": "string"}}}',
 '{"limit": 5, "sort_by": "recent_activity"}',
 '{"w": 4, "h": 3}', '{"w": 3, "h": 2}',
 ARRAY['client.*', 'project.*'], TRUE),

('notes_pinned', 'Pinned Notes', 'Display pinned notes/artifacts', 'notes',
 '{"type": "object", "properties": {"limit": {"type": "integer"}}}',
 '{"limit": 5}',
 '{"w": 4, "h": 3}', '{"w": 2, "h": 2}',
 ARRAY['artifact.created', 'artifact.updated'], TRUE),

('quick_actions', 'Quick Actions', 'Buttons for common actions', 'utility',
 '{"type": "object", "properties": {"actions": {"type": "array", "items": {"type": "object"}}}}',
 '{"actions": [{"label": "New Task", "action": "create_task"}, {"label": "New Note", "action": "create_note"}]}',
 '{"w": 2, "h": 2}', '{"w": 2, "h": 1}',
 NULL, TRUE),

('agent_shortcuts', 'Agent Shortcuts', 'Quick prompts to OS agent', 'utility',
 '{"type": "object", "properties": {"shortcuts": {"type": "array", "items": {"type": "object"}}}}',
 '{"shortcuts": [{"label": "Summarize my day", "prompt": "What do I have on my plate today?"}, {"label": "What''s overdue?", "prompt": "Show me all overdue tasks"}]}',
 '{"w": 3, "h": 2}', '{"w": 2, "h": 2}',
 NULL, TRUE),

-- Future Widgets (Disabled until dependencies exist)
('team_capacity', 'Team Capacity', 'Shows team member availability/bandwidth', 'team',
 '{"type": "object", "properties": {"team_id": {"type": "string"}}}',
 '{}',
 '{"w": 4, "h": 3}', '{"w": 3, "h": 2}',
 ARRAY['team.*'], FALSE),

('team_workload', 'Team Workload', 'Work distribution across team members', 'team',
 '{"type": "object", "properties": {"team_id": {"type": "string"}}}',
 '{}',
 '{"w": 6, "h": 4}', '{"w": 4, "h": 3}',
 ARRAY['team.*', 'task.*'], FALSE),

('project_timeline', 'Project Timeline', 'Gantt-style project timeline view', 'projects',
 '{"type": "object", "properties": {"project_ids": {"type": "array"}}}',
 '{}',
 '{"w": 12, "h": 4}', '{"w": 6, "h": 3}',
 ARRAY['project.*', 'task.*'], FALSE),

('client_pipeline', 'Client Pipeline', 'Sales pipeline visualization', 'clients',
 '{"type": "object", "properties": {"stages": {"type": "array"}}}',
 '{}',
 '{"w": 8, "h": 4}', '{"w": 6, "h": 3}',
 ARRAY['client.*'], FALSE),

('client_activity', 'Client Activity', 'Client interaction history', 'clients',
 '{"type": "object", "properties": {"client_id": {"type": "string"}}}',
 '{}',
 '{"w": 4, "h": 4}', '{"w": 3, "h": 3}',
 ARRAY['client.*'], FALSE),

('task_calendar', 'Task Calendar', 'Calendar view of tasks by due date', 'tasks',
 '{"type": "object", "properties": {"view": {"type": "string", "enum": ["month", "week"]}}}',
 '{"view": "month"}',
 '{"w": 8, "h": 5}', '{"w": 6, "h": 4}',
 ARRAY['task.*'], FALSE),

('custom_query', 'Custom Query', 'Power user SQL/filter widget', 'advanced',
 '{"type": "object", "properties": {"query": {"type": "string"}, "display": {"type": "string"}}}',
 '{}',
 '{"w": 6, "h": 4}', '{"w": 4, "h": 3}',
 NULL, FALSE)

ON CONFLICT (widget_type) DO NOTHING;

-- ============================================================================
-- SEED DATA: Dashboard Templates
-- ============================================================================
INSERT INTO dashboard_templates (name, description, category, layout, is_default, sort_order) VALUES

('My Day', 'Perfect for daily planning and task management', 'personal',
 '[
   {"widget_id": "gen_1", "widget_type": "task_summary", "position": {"x": 0, "y": 0, "w": 4, "h": 3}, "config": {"group_by": "status"}},
   {"widget_id": "gen_2", "widget_type": "upcoming_deadlines", "position": {"x": 4, "y": 0, "w": 4, "h": 3}, "config": {"days_ahead": 3}},
   {"widget_id": "gen_3", "widget_type": "quick_actions", "position": {"x": 8, "y": 0, "w": 4, "h": 2}, "config": {}},
   {"widget_id": "gen_4", "widget_type": "recent_activity", "position": {"x": 8, "y": 2, "w": 4, "h": 3}, "config": {"limit": 5}}
 ]'::jsonb,
 TRUE, 1),

('Project Manager', 'Track project progress and team workload', 'management',
 '[
   {"widget_id": "gen_1", "widget_type": "project_progress", "position": {"x": 0, "y": 0, "w": 6, "h": 3}, "config": {"show_all": true}},
   {"widget_id": "gen_2", "widget_type": "task_burndown", "position": {"x": 6, "y": 0, "w": 6, "h": 3}, "config": {"days": 30}},
   {"widget_id": "gen_3", "widget_type": "upcoming_deadlines", "position": {"x": 0, "y": 3, "w": 6, "h": 4}, "config": {"days_ahead": 14}},
   {"widget_id": "gen_4", "widget_type": "workload_heatmap", "position": {"x": 6, "y": 3, "w": 6, "h": 3}, "config": {"date_range": "month"}}
 ]'::jsonb,
 FALSE, 2),

('Executive', 'High-level KPIs and project overview', 'leadership',
 '[
   {"widget_id": "gen_1", "widget_type": "metric_card", "position": {"x": 0, "y": 0, "w": 3, "h": 2}, "config": {"metric": "tasks_due_today"}},
   {"widget_id": "gen_2", "widget_type": "metric_card", "position": {"x": 3, "y": 0, "w": 3, "h": 2}, "config": {"metric": "tasks_overdue"}},
   {"widget_id": "gen_3", "widget_type": "metric_card", "position": {"x": 6, "y": 0, "w": 3, "h": 2}, "config": {"metric": "tasks_completed_week"}},
   {"widget_id": "gen_4", "widget_type": "metric_card", "position": {"x": 9, "y": 0, "w": 3, "h": 2}, "config": {"metric": "active_projects"}},
   {"widget_id": "gen_5", "widget_type": "project_progress", "position": {"x": 0, "y": 2, "w": 6, "h": 3}, "config": {"show_all": true}},
   {"widget_id": "gen_6", "widget_type": "task_burndown", "position": {"x": 6, "y": 2, "w": 6, "h": 3}, "config": {"days": 30}}
 ]'::jsonb,
 FALSE, 3),

('Developer', 'Focus on your assigned tasks and notes', 'personal',
 '[
   {"widget_id": "gen_1", "widget_type": "task_list", "position": {"x": 0, "y": 0, "w": 6, "h": 4}, "config": {"filter": {"assigned_to_me": true}, "sort": "priority", "limit": 15}},
   {"widget_id": "gen_2", "widget_type": "notes_pinned", "position": {"x": 6, "y": 0, "w": 6, "h": 3}, "config": {"limit": 5}},
   {"widget_id": "gen_3", "widget_type": "recent_activity", "position": {"x": 6, "y": 3, "w": 6, "h": 3}, "config": {"limit": 8}},
   {"widget_id": "gen_4", "widget_type": "agent_shortcuts", "position": {"x": 0, "y": 4, "w": 4, "h": 2}, "config": {}}
 ]'::jsonb,
 FALSE, 4),

('Client Focus', 'Manage client relationships and projects', 'business',
 '[
   {"widget_id": "gen_1", "widget_type": "client_overview", "position": {"x": 0, "y": 0, "w": 4, "h": 3}, "config": {"limit": 5}},
   {"widget_id": "gen_2", "widget_type": "project_progress", "position": {"x": 4, "y": 0, "w": 4, "h": 3}, "config": {"show_all": true}},
   {"widget_id": "gen_3", "widget_type": "upcoming_deadlines", "position": {"x": 8, "y": 0, "w": 4, "h": 3}, "config": {"days_ahead": 14}},
   {"widget_id": "gen_4", "widget_type": "recent_activity", "position": {"x": 0, "y": 3, "w": 6, "h": 3}, "config": {"limit": 10}}
 ]'::jsonb,
 FALSE, 5)

ON CONFLICT DO NOTHING;

-- ============================================================================
-- TRIGGER: Update updated_at timestamp
-- ============================================================================
CREATE OR REPLACE FUNCTION update_dashboard_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_dashboard_timestamp
    BEFORE UPDATE ON user_dashboards
    FOR EACH ROW
    EXECUTE FUNCTION update_dashboard_updated_at();
