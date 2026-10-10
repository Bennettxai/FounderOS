-- =============================================================================
-- OSA Queue Worker Test Script
-- =============================================================================
-- This script tests the OSA queue worker implementation by:
-- 1. Creating test data (app_templates)
-- 2. Inserting test items into app_generation_queue
-- 3. Providing queries to monitor worker activity
-- =============================================================================

-- Step 1: Check if migration has been run
SELECT
    CASE
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'app_templates')
        THEN '✅ app_templates table exists'
        ELSE '❌ app_templates table NOT FOUND - run migration first!'
    END as app_templates_status,
    CASE
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'app_generation_queue')
        THEN '✅ app_generation_queue table exists'
        ELSE '❌ app_generation_queue table NOT FOUND - run migration first!'
    END as queue_status;

-- =============================================================================
-- Step 2: Seed app_templates with test data
-- =============================================================================

-- Clear existing test data (optional - comment out if you want to keep data)
-- DELETE FROM app_generation_queue WHERE workspace_id IN (SELECT id FROM workspaces LIMIT 1);
-- DELETE FROM app_templates WHERE template_name LIKE 'test_%';

-- Insert test templates
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
    required_modules,
    optional_features,
    generation_prompt,
    scaffold_type
) VALUES
(
    'test_crm_basic',
    'Business Management',
    'Test CRM System',
    'A basic CRM system for testing OSA worker functionality',
    'users',
    ARRAY['saas', 'startup', 'small_business'],
    ARRAY['customer_management', 'sales_tracking'],
    ARRAY['solo', 'small'],
    80,
    '{"features": ["contacts", "deals", "tasks"], "complexity": "medium"}'::jsonb,
    ARRAY['auth', 'database', 'api'],
    ARRAY['email_integration', 'calendar', 'reports'],
    'Generate a CRM application with contact management, deal tracking, and task management. Use Go backend with PostgreSQL and Svelte frontend. Include authentication, REST API, and responsive UI.',
    'full-stack'
),
(
    'test_todo_app',
    'Productivity',
    'Test Todo Application',
    'A simple todo app for testing worker',
    'check-square',
    ARRAY['personal', 'startup'],
    ARRAY['task_management', 'productivity'],
    ARRAY['solo'],
    60,
    '{"features": ["tasks", "lists", "tags"], "complexity": "simple"}'::jsonb,
    ARRAY['auth', 'database'],
    ARRAY['reminders', 'collaboration'],
    'Generate a todo list application with task creation, categorization, and completion tracking. Use modern web technologies with a clean, intuitive UI.',
    'frontend-focused'
),
(
    'test_analytics_dashboard',
    'Analytics',
    'Test Analytics Dashboard',
    'Analytics dashboard for testing',
    'chart-bar',
    ARRAY['saas', 'enterprise'],
    ARRAY['data_visualization', 'reporting'],
    ARRAY['small', 'medium'],
    70,
    '{"features": ["charts", "widgets", "exports"], "complexity": "high"}'::jsonb,
    ARRAY['database', 'api', 'charting'],
    ARRAY['real_time_updates', 'custom_widgets'],
    'Generate an analytics dashboard with interactive charts, customizable widgets, and data export capabilities. Include PostgreSQL integration and real-time data updates.',
    'full-stack'
)
ON CONFLICT (template_name) DO UPDATE SET
    description = EXCLUDED.description,
    generation_prompt = EXCLUDED.generation_prompt,
    updated_at = NOW();

-- Verify templates were inserted
SELECT
    id,
    template_name,
    display_name,
    category,
    priority_score,
    CASE
        WHEN generation_prompt IS NOT NULL THEN '✅ Has prompt'
        ELSE '❌ No prompt'
    END as prompt_status
FROM app_templates
WHERE template_name LIKE 'test_%'
ORDER BY priority_score DESC;

-- =============================================================================
-- Step 3: Get a test workspace_id
-- =============================================================================

-- Get the first workspace for testing
DO $$
DECLARE
    test_workspace_id UUID;
    test_template_id UUID;
BEGIN
    -- Get first workspace
    SELECT id INTO test_workspace_id FROM workspaces LIMIT 1;

    IF test_workspace_id IS NULL THEN
        RAISE EXCEPTION 'No workspaces found! Create a workspace first.';
    END IF;

    RAISE NOTICE 'Using workspace_id: %', test_workspace_id;

    -- Get test template ID
    SELECT id INTO test_template_id FROM app_templates WHERE template_name = 'test_crm_basic';
    RAISE NOTICE 'Using template_id: %', test_template_id;
END $$;

-- Show available workspaces
SELECT
    id,
    name,
    created_at
FROM workspaces
ORDER BY created_at DESC
LIMIT 5;

-- =============================================================================
-- Step 4: Insert test queue items
-- =============================================================================

-- Insert test items into app_generation_queue
-- REPLACE the workspace_id with actual value from above query
WITH test_workspace AS (
    SELECT id FROM workspaces LIMIT 1
),
test_template AS (
    SELECT id FROM app_templates WHERE template_name = 'test_crm_basic'
)
INSERT INTO app_generation_queue (
    workspace_id,
    template_id,
    status,
    priority,
    generation_context
)
SELECT
    tw.id,
    tt.id,
    'pending',
    8,
    jsonb_build_object(
        'app_name', 'Test CRM from Worker',
        'description', 'Testing OSA queue worker functionality',
        'user_config', jsonb_build_object(
            'company_name', 'Test Corp',
            'industry', 'Technology'
        )
    )
FROM test_workspace tw, test_template tt
RETURNING
    id,
    workspace_id,
    template_id,
    status,
    priority,
    created_at;

-- Insert a second test item with different template
WITH test_workspace AS (
    SELECT id FROM workspaces LIMIT 1
),
test_template AS (
    SELECT id FROM app_templates WHERE template_name = 'test_todo_app'
)
INSERT INTO app_generation_queue (
    workspace_id,
    template_id,
    status,
    priority,
    generation_context
)
SELECT
    tw.id,
    tt.id,
    'pending',
    5,
    jsonb_build_object(
        'app_name', 'My Todo List',
        'description', 'Personal task manager'
    )
FROM test_workspace tw, test_template tt
RETURNING
    id,
    workspace_id,
    template_id,
    status,
    priority,
    created_at;

-- =============================================================================
-- Step 5: Monitor queue status
-- =============================================================================

-- View all queue items with details
SELECT
    q.id,
    w.name as workspace_name,
    t.template_name,
    t.display_name as template_display_name,
    q.status,
    q.priority,
    q.generation_context->>'app_name' as app_name,
    q.error_message,
    q.retry_count,
    q.created_at,
    q.started_at,
    q.completed_at,
    CASE
        WHEN q.completed_at IS NOT NULL THEN
            EXTRACT(EPOCH FROM (q.completed_at - q.started_at)) || ' seconds'
        WHEN q.started_at IS NOT NULL THEN
            EXTRACT(EPOCH FROM (NOW() - q.started_at)) || ' seconds (in progress)'
        ELSE
            'Not started'
    END as processing_time
FROM app_generation_queue q
LEFT JOIN workspaces w ON q.workspace_id = w.id
LEFT JOIN app_templates t ON q.template_id = t.id
ORDER BY q.created_at DESC;

-- Queue statistics by status
SELECT
    status,
    COUNT(*) as count,
    AVG(retry_count) as avg_retries,
    MAX(retry_count) as max_retries
FROM app_generation_queue
GROUP BY status
ORDER BY
    CASE status
        WHEN 'processing' THEN 1
        WHEN 'pending' THEN 2
        WHEN 'completed' THEN 3
        WHEN 'failed' THEN 4
    END;

-- =============================================================================
-- Step 6: Watch for worker activity (run this repeatedly)
-- =============================================================================

-- Real-time monitoring query (run every few seconds)
SELECT
    '=== OSA WORKER MONITOR ===' as header,
    NOW() as timestamp
UNION ALL
SELECT
    'Queue Status: ' || status || ' = ' || COUNT(*)::text,
    NULL
FROM app_generation_queue
GROUP BY status
ORDER BY header DESC, timestamp DESC NULLS LAST;

-- Details of processing/pending items
SELECT
    id,
    status,
    priority,
    generation_context->>'app_name' as app_name,
    retry_count,
    CASE
        WHEN started_at IS NOT NULL THEN
            'Started ' || EXTRACT(EPOCH FROM (NOW() - started_at))::int || 's ago'
        ELSE
            'Waiting'
    END as status_info,
    SUBSTRING(error_message, 1, 100) as error_preview
FROM app_generation_queue
WHERE status IN ('pending', 'processing')
ORDER BY priority DESC, created_at ASC;

-- =============================================================================
-- Step 7: Check for completed items
-- =============================================================================

SELECT
    q.id,
    q.generation_context->>'app_name' as app_name,
    t.template_name,
    q.status,
    q.started_at,
    q.completed_at,
    EXTRACT(EPOCH FROM (q.completed_at - q.started_at)) as duration_seconds,
    q.error_message
FROM app_generation_queue q
LEFT JOIN app_templates t ON q.template_id = t.id
WHERE q.status IN ('completed', 'failed')
ORDER BY q.completed_at DESC
LIMIT 10;

-- =============================================================================
-- Step 8: Cleanup (optional - run after testing)
-- =============================================================================

/*
-- Delete test queue items
DELETE FROM app_generation_queue
WHERE template_id IN (
    SELECT id FROM app_templates WHERE template_name LIKE 'test_%'
);

-- Delete test templates
DELETE FROM app_templates WHERE template_name LIKE 'test_%';

-- Verify cleanup
SELECT COUNT(*) as remaining_test_templates
FROM app_templates
WHERE template_name LIKE 'test_%';

SELECT COUNT(*) as remaining_test_queue_items
FROM app_generation_queue
WHERE template_id IN (
    SELECT id FROM app_templates WHERE template_name LIKE 'test_%'
);
*/
