-- ============================================================================
-- VERIFY DASHBOARD DATA
-- Quick script to check if widget_types are seeded and dashboards exist
-- ============================================================================

-- Check if dashboard_widgets table exists and has data
SELECT 'Widget Types Count' AS check_name, COUNT(*) AS result
FROM dashboard_widgets;

-- List all enabled widget types
SELECT 'Enabled Widget Types' AS info,
       widget_type,
       name,
       category,
       is_enabled
FROM dashboard_widgets
ORDER BY
    CASE category
        WHEN 'tasks' THEN 1
        WHEN 'projects' THEN 2
        WHEN 'analytics' THEN 3
        WHEN 'clients' THEN 4
        WHEN 'utility' THEN 5
        ELSE 6
    END,
    widget_type;

-- Check user_dashboards table
SELECT 'User Dashboards Count' AS check_name, COUNT(*) AS result
FROM user_dashboards;

-- List dashboards if any exist
SELECT
    'User Dashboards' AS info,
    id,
    user_id,
    name,
    is_default,
    jsonb_array_length(layout->'widgets') AS widget_count,
    created_at
FROM user_dashboards
ORDER BY created_at DESC
LIMIT 10;

-- Check dashboard_templates
SELECT 'Dashboard Templates Count' AS check_name, COUNT(*) AS result
FROM dashboard_templates;

-- List available templates
SELECT
    'Dashboard Templates' AS info,
    name,
    category,
    is_default,
    jsonb_array_length(layout) AS widget_count
FROM dashboard_templates
ORDER BY sort_order;

-- Summary report
SELECT
    'SUMMARY' AS report,
    (SELECT COUNT(*) FROM dashboard_widgets WHERE is_enabled = TRUE) AS enabled_widgets,
    (SELECT COUNT(*) FROM dashboard_widgets WHERE is_enabled = FALSE) AS disabled_widgets,
    (SELECT COUNT(*) FROM user_dashboards) AS user_dashboards,
    (SELECT COUNT(*) FROM dashboard_templates) AS templates;
