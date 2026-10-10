-- Fathom Analytics Queries
-- SQLC queries for Fathom Analytics integration

-- ============================================================================
-- Fathom Sites
-- ============================================================================

-- name: UpsertFathomSite :one
INSERT INTO fathom_sites (
    user_id, site_id, name, sharing_url, share_config, synced_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (user_id, site_id) DO UPDATE SET
    name = EXCLUDED.name,
    sharing_url = EXCLUDED.sharing_url,
    share_config = EXCLUDED.share_config,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetFathomSite :one
SELECT * FROM fathom_sites
WHERE user_id = $1 AND site_id = $2;

-- name: GetFathomSitesByUser :many
SELECT * FROM fathom_sites
WHERE user_id = $1
ORDER BY name;

-- name: DeleteFathomSite :exec
DELETE FROM fathom_sites
WHERE user_id = $1 AND site_id = $2;

-- name: DeleteFathomSitesByUser :exec
DELETE FROM fathom_sites WHERE user_id = $1;

-- ============================================================================
-- Fathom Aggregations (Daily Analytics)
-- ============================================================================

-- name: UpsertFathomAggregation :one
INSERT INTO fathom_aggregations (
    user_id, site_id, date, visits, uniques, pageviews,
    avg_duration, bounce_rate, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
ON CONFLICT (user_id, site_id, date) DO UPDATE SET
    visits = EXCLUDED.visits,
    uniques = EXCLUDED.uniques,
    pageviews = EXCLUDED.pageviews,
    avg_duration = EXCLUDED.avg_duration,
    bounce_rate = EXCLUDED.bounce_rate,
    synced_at = NOW()
RETURNING *;

-- name: GetFathomAggregation :one
SELECT * FROM fathom_aggregations
WHERE user_id = $1 AND site_id = $2 AND date = $3;

-- name: GetFathomAggregationsByDateRange :many
SELECT * FROM fathom_aggregations
WHERE user_id = $1 AND site_id = $2
  AND date >= $3 AND date <= $4
ORDER BY date DESC;

-- name: GetFathomAggregationsRecent :many
SELECT * FROM fathom_aggregations
WHERE user_id = $1 AND site_id = $2
ORDER BY date DESC
LIMIT $3;

-- name: GetFathomAggregationsSummary :one
SELECT
    SUM(visits) as total_visits,
    SUM(uniques) as total_uniques,
    SUM(pageviews) as total_pageviews,
    AVG(avg_duration) as avg_duration,
    AVG(bounce_rate) as avg_bounce_rate
FROM fathom_aggregations
WHERE user_id = $1 AND site_id = $2
  AND date >= $3 AND date <= $4;

-- name: DeleteFathomAggregationsByUser :exec
DELETE FROM fathom_aggregations WHERE user_id = $1;

-- ============================================================================
-- Fathom Pages
-- ============================================================================

-- name: UpsertFathomPage :one
INSERT INTO fathom_pages (
    user_id, site_id, pathname, hostname, visits, uniques,
    pageviews, avg_duration, bounce_rate, period_start, period_end, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (user_id, site_id, pathname, period_start, period_end) DO UPDATE SET
    hostname = EXCLUDED.hostname,
    visits = EXCLUDED.visits,
    uniques = EXCLUDED.uniques,
    pageviews = EXCLUDED.pageviews,
    avg_duration = EXCLUDED.avg_duration,
    bounce_rate = EXCLUDED.bounce_rate,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetFathomPagesByPeriod :many
SELECT * FROM fathom_pages
WHERE user_id = $1 AND site_id = $2
  AND period_start = $3 AND period_end = $4
ORDER BY pageviews DESC;

-- name: GetFathomTopPages :many
SELECT * FROM fathom_pages
WHERE user_id = $1 AND site_id = $2
ORDER BY pageviews DESC
LIMIT $3;

-- name: DeleteFathomPagesByUser :exec
DELETE FROM fathom_pages WHERE user_id = $1;

-- ============================================================================
-- Fathom Referrers
-- ============================================================================

-- name: UpsertFathomReferrer :one
INSERT INTO fathom_referrers (
    user_id, site_id, referrer, visits, uniques,
    period_start, period_end, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, site_id, referrer, period_start, period_end) DO UPDATE SET
    visits = EXCLUDED.visits,
    uniques = EXCLUDED.uniques,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetFathomReferrersByPeriod :many
SELECT * FROM fathom_referrers
WHERE user_id = $1 AND site_id = $2
  AND period_start = $3 AND period_end = $4
ORDER BY visits DESC;

-- name: GetFathomTopReferrers :many
SELECT * FROM fathom_referrers
WHERE user_id = $1 AND site_id = $2
ORDER BY visits DESC
LIMIT $3;

-- name: DeleteFathomReferrersByUser :exec
DELETE FROM fathom_referrers WHERE user_id = $1;

-- ============================================================================
-- Fathom Events
-- ============================================================================

-- name: UpsertFathomEvent :one
INSERT INTO fathom_events (
    user_id, site_id, event_id, event_name, count,
    period_start, period_end, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, site_id, event_id, period_start, period_end) DO UPDATE SET
    event_name = EXCLUDED.event_name,
    count = EXCLUDED.count,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetFathomEventsByPeriod :many
SELECT * FROM fathom_events
WHERE user_id = $1 AND site_id = $2
  AND period_start = $3 AND period_end = $4
ORDER BY count DESC;

-- name: GetFathomTopEvents :many
SELECT * FROM fathom_events
WHERE user_id = $1 AND site_id = $2
ORDER BY count DESC
LIMIT $3;

-- name: DeleteFathomEventsByUser :exec
DELETE FROM fathom_events WHERE user_id = $1;
