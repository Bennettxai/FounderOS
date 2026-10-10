-- ============================================================================
-- ANALYTICS & DASHBOARD ENHANCEMENT QUERIES
-- CRUD operations for analytics_snapshots, dashboard_views, dashboard_shares, widget_data_cache
-- ============================================================================

-- ============================================================================
-- ANALYTICS SNAPSHOTS
-- ============================================================================

-- name: CreateAnalyticsSnapshot :one
INSERT INTO analytics_snapshots (user_id, workspace_id, snapshot_date, metrics)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, snapshot_date) 
DO UPDATE SET metrics = EXCLUDED.metrics
RETURNING *;

-- name: GetAnalyticsSnapshot :one
SELECT * FROM analytics_snapshots
WHERE user_id = $1 AND snapshot_date = $2;

-- name: GetAnalyticsSnapshotRange :many
SELECT * FROM analytics_snapshots
WHERE user_id = $1 
  AND snapshot_date >= $2 
  AND snapshot_date <= $3
ORDER BY snapshot_date DESC;

-- name: GetLatestAnalyticsSnapshot :one
SELECT * FROM analytics_snapshots
WHERE user_id = $1
ORDER BY snapshot_date DESC
LIMIT 1;

-- name: GetAnalyticsTrend :many
SELECT 
    snapshot_date,
    metrics->>'tasks_total' as tasks_total,
    metrics->>'tasks_completed' as tasks_completed,
    metrics->>'tasks_overdue' as tasks_overdue,
    metrics->>'projects_active' as projects_active
FROM analytics_snapshots
WHERE user_id = $1 
  AND snapshot_date >= CURRENT_DATE - ($2::int || ' days')::interval
ORDER BY snapshot_date ASC;

-- name: DeleteOldAnalyticsSnapshots :execrows
DELETE FROM analytics_snapshots
WHERE snapshot_date < CURRENT_DATE - ($1::int || ' days')::interval;

-- ============================================================================
-- DASHBOARD VIEWS
-- ============================================================================

-- name: RecordDashboardView :one
INSERT INTO dashboard_views (
    dashboard_id, user_id, session_id, source, device_type
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateDashboardViewDuration :exec
UPDATE dashboard_views
SET duration_seconds = $2, widget_interactions = $3
WHERE id = $1;

-- name: GetDashboardViews :many
SELECT * FROM dashboard_views
WHERE dashboard_id = $1
ORDER BY viewed_at DESC
LIMIT $2;

-- name: GetUserDashboardViews :many
SELECT * FROM dashboard_views
WHERE user_id = $1
ORDER BY viewed_at DESC
LIMIT $2;

-- name: GetDashboardViewStats :one
SELECT 
    COUNT(*) as total_views,
    COUNT(DISTINCT user_id) as unique_viewers,
    AVG(duration_seconds) as avg_duration_seconds,
    MAX(viewed_at) as last_viewed_at
FROM dashboard_views
WHERE dashboard_id = $1;

-- name: GetPopularDashboards :many
SELECT 
    dashboard_id,
    COUNT(*) as view_count,
    COUNT(DISTINCT user_id) as unique_viewers,
    MAX(viewed_at) as last_viewed
FROM dashboard_views
WHERE viewed_at >= CURRENT_DATE - ($1::int || ' days')::interval
GROUP BY dashboard_id
ORDER BY view_count DESC
LIMIT $2;

-- name: GetUserDashboardActivity :many
SELECT 
    dv.dashboard_id,
    ud.name as dashboard_name,
    COUNT(*) as view_count,
    MAX(dv.viewed_at) as last_viewed
FROM dashboard_views dv
JOIN user_dashboards ud ON dv.dashboard_id = ud.id
WHERE dv.user_id = $1
  AND dv.viewed_at >= CURRENT_DATE - ($2::int || ' days')::interval
GROUP BY dv.dashboard_id, ud.name
ORDER BY view_count DESC;

-- name: CleanupOldDashboardViews :execrows
DELETE FROM dashboard_views
WHERE viewed_at < NOW() - INTERVAL '90 days';

-- ============================================================================
-- DASHBOARD SHARES
-- ============================================================================

-- name: CreateDashboardShare :one
INSERT INTO dashboard_shares (
    dashboard_id, shared_with_user_id, shared_with_role, 
    shared_with_workspace_id, permission, expires_at, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDashboardShare :one
SELECT * FROM dashboard_shares
WHERE id = $1;

-- name: ListDashboardShares :many
SELECT * FROM dashboard_shares
WHERE dashboard_id = $1
ORDER BY created_at DESC;

-- name: ListUserSharedDashboards :many
SELECT ds.*, ud.name as dashboard_name, ud.description, ud.layout
FROM dashboard_shares ds
JOIN user_dashboards ud ON ds.dashboard_id = ud.id
WHERE ds.shared_with_user_id = $1
  AND (ds.expires_at IS NULL OR ds.expires_at > NOW())
ORDER BY ds.created_at DESC;

-- name: ListRoleSharedDashboards :many
SELECT ds.*, ud.name as dashboard_name, ud.description, ud.layout
FROM dashboard_shares ds
JOIN user_dashboards ud ON ds.dashboard_id = ud.id
WHERE ds.shared_with_role = $1
  AND (ds.expires_at IS NULL OR ds.expires_at > NOW())
ORDER BY ds.created_at DESC;

-- name: UpdateDashboardShare :one
UPDATE dashboard_shares
SET 
    permission = COALESCE(sqlc.narg('permission'), permission),
    expires_at = sqlc.narg('expires_at')
WHERE id = $1
RETURNING *;

-- name: DeleteDashboardShare :exec
DELETE FROM dashboard_shares
WHERE id = $1;

-- name: DeleteDashboardShareByUser :exec
DELETE FROM dashboard_shares
WHERE dashboard_id = $1 AND shared_with_user_id = $2;

-- name: CheckDashboardAccess :one
SELECT EXISTS (
    SELECT 1 FROM dashboard_shares
    WHERE dashboard_id = $1 
      AND (
          shared_with_user_id = $2 
          OR shared_with_role = ANY($3::text[])
          OR shared_with_workspace_id = $4
      )
      AND (expires_at IS NULL OR expires_at > NOW())
) as has_access;

-- name: CleanupExpiredShares :execrows
DELETE FROM dashboard_shares
WHERE expires_at IS NOT NULL AND expires_at < NOW();

-- ============================================================================
-- WIDGET DATA CACHE
-- ============================================================================

-- name: GetWidgetCache :one
SELECT * FROM widget_data_cache
WHERE user_id = $1 AND widget_type = $2 AND cache_key = $3
  AND expires_at > NOW();

-- name: SetWidgetCache :one
INSERT INTO widget_data_cache (user_id, widget_type, cache_key, data, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, widget_type, cache_key)
DO UPDATE SET 
    data = EXCLUDED.data, 
    expires_at = EXCLUDED.expires_at,
    created_at = NOW()
RETURNING *;

-- name: IncrementWidgetCacheHit :exec
UPDATE widget_data_cache
SET hit_count = hit_count + 1, last_hit_at = NOW()
WHERE user_id = $1 AND widget_type = $2 AND cache_key = $3;

-- name: InvalidateWidgetCache :exec
DELETE FROM widget_data_cache
WHERE user_id = $1 AND widget_type = $2;

-- name: InvalidateAllUserCache :exec
DELETE FROM widget_data_cache
WHERE user_id = $1;

-- name: CleanupExpiredWidgetCache :execrows
DELETE FROM widget_data_cache
WHERE expires_at < NOW();

-- name: GetWidgetCacheStats :many
SELECT 
    widget_type,
    COUNT(*) as cache_entries,
    SUM(hit_count) as total_hits,
    AVG(hit_count) as avg_hits_per_entry
FROM widget_data_cache
WHERE user_id = $1
GROUP BY widget_type;
