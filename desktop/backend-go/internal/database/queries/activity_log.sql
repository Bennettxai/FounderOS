-- ============================================================================
-- ACTIVITY LOG QUERIES
-- ============================================================================

-- name: ListUserActivityLog :many
SELECT * FROM activity_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: GetEntityActivityLog :many
SELECT * FROM activity_log
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: GetActorActivityLog :many
SELECT * FROM activity_log
WHERE actor_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: CreateActivityLog :one
INSERT INTO activity_log (
    user_id, entity_type, entity_id, entity_name,
    action, action_detail, actor_id, actor_name,
    changes, related_entity_type, related_entity_id, related_entity_name,
    metadata
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, $12,
    $13
)
RETURNING *;

-- name: ListRecentActivity :many
SELECT * FROM activity_log
WHERE user_id = $1
  AND created_at > NOW() - INTERVAL '7 days'
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListActivityByAction :many
SELECT * FROM activity_log
WHERE user_id = $1
  AND action = $2
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListActivityByEntityType :many
SELECT * FROM activity_log
WHERE user_id = $1
  AND entity_type = $2
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: CountUserActivity :one
SELECT COUNT(*) as count FROM activity_log
WHERE user_id = $1;

-- name: CountEntityActivity :one
SELECT COUNT(*) as count FROM activity_log
WHERE entity_type = $1 AND entity_id = $2;

-- name: GetRelatedActivity :many
SELECT * FROM activity_log
WHERE related_entity_type = $1 AND related_entity_id = $2
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: DeleteOldActivityLogs :exec
DELETE FROM activity_log
WHERE created_at < NOW() - (sqlc.arg(retention_days)::int || ' days')::INTERVAL;
