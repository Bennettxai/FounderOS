-- ============================================================================
-- DASHBOARD & ANALYTICS ENHANCEMENTS
-- Migration: 023_dashboard_analytics_enhancements.sql
-- Created: January 7, 2026
-- Author: Javaris Tavel
-- 
-- This migration adds:
-- 1. analytics_snapshots - Historical metrics tracking for trends
-- 2. dashboard_views - Dashboard usage tracking
-- 3. dashboard_shares - Granular sharing permissions
-- 4. widget_data_cache - Performance optimization for expensive queries
-- ============================================================================

-- ============================================================================
-- 1. ANALYTICS SNAPSHOTS TABLE
-- Stores daily snapshots of user metrics for trend analysis
-- ============================================================================

CREATE TABLE IF NOT EXISTS analytics_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Snapshot date (one per day per user)
    snapshot_date DATE NOT NULL,
    
    -- Metrics captured at snapshot time
    -- Example: {
    --   "tasks_total": 50,
    --   "tasks_completed": 30,
    --   "tasks_overdue": 5,
    --   "tasks_in_progress": 10,
    --   "tasks_todo": 5,
    --   "projects_active": 3,
    --   "projects_completed": 2,
    --   "avg_task_completion_days": 2.5,
    --   "tasks_completed_today": 5,
    --   "tasks_created_today": 3
    -- }
    metrics JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, snapshot_date)
);

-- Indexes for analytics_snapshots
CREATE INDEX IF NOT EXISTS idx_analytics_snapshots_user_date 
    ON analytics_snapshots(user_id, snapshot_date DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_snapshots_workspace 
    ON analytics_snapshots(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_analytics_snapshots_date 
    ON analytics_snapshots(snapshot_date DESC);

-- ============================================================================
-- 2. DASHBOARD VIEWS TABLE
-- Tracks dashboard usage for analytics and identifying valuable dashboards
-- ============================================================================

CREATE TABLE IF NOT EXISTS dashboard_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL REFERENCES user_dashboards(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    
    -- When the view occurred
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Session tracking
    session_id VARCHAR(100),
    
    -- Engagement metrics
    duration_seconds INTEGER,
    widget_interactions JSONB DEFAULT '[]',
    -- Example: [
    --   {"widget_id": "w1", "widget_type": "task_summary", "interaction": "click", "count": 3},
    --   {"widget_id": "w2", "widget_type": "metric_card", "interaction": "hover", "count": 1}
    -- ]
    
    -- Context
    source VARCHAR(50), -- 'direct', 'navigation', 'search', 'agent'
    device_type VARCHAR(20) -- 'desktop', 'mobile', 'tablet'
);

-- Indexes for dashboard_views
CREATE INDEX IF NOT EXISTS idx_dashboard_views_dashboard 
    ON dashboard_views(dashboard_id, viewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_dashboard_views_user 
    ON dashboard_views(user_id, viewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_dashboard_views_date 
    ON dashboard_views(viewed_at DESC);

-- ============================================================================
-- 3. DASHBOARD SHARES TABLE
-- Granular sharing permissions for dashboards
-- ============================================================================

CREATE TABLE IF NOT EXISTS dashboard_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL REFERENCES user_dashboards(id) ON DELETE CASCADE,
    
    -- Share target (one of these should be set)
    shared_with_user_id VARCHAR(255),
    shared_with_role VARCHAR(100),
    shared_with_workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Permission level
    permission VARCHAR(20) DEFAULT 'view' CHECK (permission IN ('view', 'edit', 'admin')),
    
    -- Optional expiration
    expires_at TIMESTAMPTZ,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    
    -- Ensure unique shares per target
    UNIQUE(dashboard_id, shared_with_user_id),
    UNIQUE(dashboard_id, shared_with_role),
    
    -- At least one share target must be specified
    CONSTRAINT chk_share_target CHECK (
        shared_with_user_id IS NOT NULL OR 
        shared_with_role IS NOT NULL OR 
        shared_with_workspace_id IS NOT NULL
    )
);

-- Indexes for dashboard_shares
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_dashboard 
    ON dashboard_shares(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_user 
    ON dashboard_shares(shared_with_user_id) WHERE shared_with_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_role 
    ON dashboard_shares(shared_with_role) WHERE shared_with_role IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_workspace 
    ON dashboard_shares(shared_with_workspace_id) WHERE shared_with_workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_dashboard_shares_expiry 
    ON dashboard_shares(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- 4. WIDGET DATA CACHE TABLE
-- Caches expensive widget query results for faster dashboard loads
-- ============================================================================

CREATE TABLE IF NOT EXISTS widget_data_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    widget_type VARCHAR(100) NOT NULL,
    
    -- Cache key (hashed config + date range + any relevant params)
    cache_key VARCHAR(255) NOT NULL,
    
    -- Cached response data
    data JSONB NOT NULL,
    
    -- TTL management
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    hit_count INTEGER DEFAULT 0,
    last_hit_at TIMESTAMPTZ,
    
    UNIQUE(user_id, widget_type, cache_key)
);

-- Indexes for widget_data_cache
CREATE INDEX IF NOT EXISTS idx_widget_cache_lookup 
    ON widget_data_cache(user_id, widget_type, cache_key);
CREATE INDEX IF NOT EXISTS idx_widget_cache_expiry 
    ON widget_data_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_widget_cache_type 
    ON widget_data_cache(widget_type);

-- ============================================================================
-- 5. HELPER FUNCTIONS
-- ============================================================================

-- Function to clean up expired cache entries (run via cron)
CREATE OR REPLACE FUNCTION cleanup_expired_widget_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM widget_data_cache WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old dashboard views (keep 90 days)
CREATE OR REPLACE FUNCTION cleanup_old_dashboard_views()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM dashboard_views WHERE viewed_at < NOW() - INTERVAL '90 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to record a dashboard view (upsert pattern for deduplication)
CREATE OR REPLACE FUNCTION record_dashboard_view(
    p_dashboard_id UUID,
    p_user_id VARCHAR(255),
    p_session_id VARCHAR(100) DEFAULT NULL,
    p_source VARCHAR(50) DEFAULT 'direct',
    p_device_type VARCHAR(20) DEFAULT 'desktop'
)
RETURNS UUID AS $$
DECLARE
    view_id UUID;
BEGIN
    INSERT INTO dashboard_views (
        dashboard_id, 
        user_id, 
        session_id, 
        source, 
        device_type
    ) VALUES (
        p_dashboard_id, 
        p_user_id, 
        p_session_id, 
        p_source, 
        p_device_type
    )
    RETURNING id INTO view_id;
    
    RETURN view_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create daily analytics snapshot for a user
CREATE OR REPLACE FUNCTION create_analytics_snapshot(p_user_id VARCHAR(255))
RETURNS UUID AS $$
DECLARE
    snapshot_id UUID;
    metrics_data JSONB;
BEGIN
    -- Build metrics JSON
    SELECT jsonb_build_object(
        'tasks_total', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id),
        'tasks_completed', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND status = 'done'),
        'tasks_overdue', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND due_date < NOW() AND status != 'done'),
        'tasks_in_progress', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND status = 'in_progress'),
        'tasks_todo', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND status = 'todo'),
        'projects_active', (SELECT COUNT(*) FROM projects WHERE user_id = p_user_id AND status NOT IN ('completed', 'archived', 'cancelled')),
        'projects_completed', (SELECT COUNT(*) FROM projects WHERE user_id = p_user_id AND status = 'completed'),
        'tasks_completed_today', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND status = 'done' AND updated_at::date = CURRENT_DATE),
        'tasks_created_today', (SELECT COUNT(*) FROM tasks WHERE user_id = p_user_id AND created_at::date = CURRENT_DATE),
        'snapshot_timestamp', NOW()
    ) INTO metrics_data;
    
    -- Upsert snapshot (one per day)
    INSERT INTO analytics_snapshots (user_id, snapshot_date, metrics)
    VALUES (p_user_id, CURRENT_DATE, metrics_data)
    ON CONFLICT (user_id, snapshot_date) 
    DO UPDATE SET metrics = EXCLUDED.metrics
    RETURNING id INTO snapshot_id;
    
    RETURN snapshot_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 6. COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE analytics_snapshots IS 'Daily snapshots of user metrics for trend analysis and historical reporting';
COMMENT ON TABLE dashboard_views IS 'Tracks dashboard usage for analytics, engagement metrics, and identifying valuable dashboards';
COMMENT ON TABLE dashboard_shares IS 'Granular sharing permissions for dashboards with user, role, or workspace targeting';
COMMENT ON TABLE widget_data_cache IS 'Cache layer for expensive widget queries to improve dashboard load performance';

COMMENT ON FUNCTION cleanup_expired_widget_cache() IS 'Removes expired cache entries. Run via cron: SELECT cleanup_expired_widget_cache()';
COMMENT ON FUNCTION cleanup_old_dashboard_views() IS 'Removes view records older than 90 days. Run via cron: SELECT cleanup_old_dashboard_views()';
COMMENT ON FUNCTION record_dashboard_view IS 'Records a dashboard view event for analytics tracking';
COMMENT ON FUNCTION create_analytics_snapshot IS 'Creates or updates daily analytics snapshot for a user. Run nightly for all active users.';
