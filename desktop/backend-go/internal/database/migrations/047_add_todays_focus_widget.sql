-- ============================================================================
-- ADD TODAYS_FOCUS WIDGET TYPE
-- Migration: 047_add_todays_focus_widget.sql
-- Created: January 19, 2026
-- 
-- Adds the todays_focus widget type to align with frontend dashboard
-- ============================================================================

-- Add todays_focus widget type if it doesn't exist
INSERT INTO dashboard_widgets (widget_type, name, description, category, config_schema, default_config, default_size, min_size, sse_events, is_enabled) 
VALUES (
    'todays_focus', 
    'Today''s Focus', 
    'Track your daily priorities and focus items', 
    'productivity',
    '{"type": "object", "properties": {"max_items": {"type": "integer", "default": 5}, "show_completed": {"type": "boolean", "default": true}}}',
    '{"max_items": 5, "show_completed": true}',
    '{"w": 4, "h": 3}', 
    '{"w": 2, "h": 2}',
    ARRAY['focus.created', 'focus.updated', 'focus.completed'],
    TRUE
)
ON CONFLICT (widget_type) DO NOTHING;

-- Verify all expected widget types exist (no-op if already present)
DO $$
BEGIN
    -- Log which widget types are available
    RAISE NOTICE 'Available widget types:';
    PERFORM widget_type FROM dashboard_widgets WHERE is_enabled = TRUE;
END $$;
