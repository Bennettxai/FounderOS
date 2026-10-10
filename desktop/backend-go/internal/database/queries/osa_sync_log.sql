-- name: CreateSyncLog :one
INSERT INTO osa_sync_log (
    sync_session_id,
    entity_type,
    entity_id,
    operation,
    direction,
    before_snapshot,
    after_snapshot,
    changes,
    status,
    conflict_type,
    conflict_resolution,
    vector_clock,
    duration_ms,
    payload_size_bytes,
    error_message,
    error_stack,
    retry_count,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) RETURNING *;

-- name: GetSyncLog :one
SELECT * FROM osa_sync_log
WHERE id = $1;

-- name: ListSyncLogsBySession :many
SELECT * FROM osa_sync_log
WHERE sync_session_id = $1
ORDER BY created_at ASC;

-- name: ListSyncLogsByEntity :many
SELECT * FROM osa_sync_log
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListSyncLogsByOperation :many
SELECT * FROM osa_sync_log
WHERE operation = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListSyncLogsByStatus :many
SELECT * FROM osa_sync_log
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListConflicts :many
SELECT * FROM osa_sync_log
WHERE conflict_type IS NOT NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnresolvedConflicts :many
SELECT * FROM osa_sync_log
WHERE conflict_type IS NOT NULL
AND conflict_resolution IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetRecentSyncActivity :many
SELECT * FROM osa_sync_log
ORDER BY created_at DESC
LIMIT $1;

-- name: GetSyncStats :one
SELECT
    COUNT(*) as total_syncs,
    COUNT(*) FILTER (WHERE status = 'success') as success_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'conflict') as conflict_count,
    COUNT(*) FILTER (WHERE conflict_type IS NOT NULL) as total_conflicts,
    COUNT(*) FILTER (WHERE conflict_resolution IS NOT NULL) as resolved_conflicts,
    AVG(duration_ms) as avg_duration_ms,
    SUM(payload_size_bytes) as total_data_synced
FROM osa_sync_log
WHERE created_at >= $1;

-- name: GetSyncStatsByEntity :one
SELECT
    entity_type,
    COUNT(*) as sync_count,
    COUNT(*) FILTER (WHERE status = 'success') as success_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    MAX(created_at) as last_sync_at
FROM osa_sync_log
WHERE entity_id = $1
GROUP BY entity_type;

-- name: GetSyncSessionStats :one
SELECT
    COUNT(*) as total_operations,
    COUNT(*) FILTER (WHERE status = 'success') as success_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE conflict_type IS NOT NULL) as conflict_count,
    MIN(created_at) as session_started_at,
    MAX(created_at) as session_ended_at,
    SUM(duration_ms) as total_duration_ms
FROM osa_sync_log
WHERE sync_session_id = $1;

-- name: GetFailedSyncsByDateRange :many
SELECT * FROM osa_sync_log
WHERE status = 'failed'
AND created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC;

-- name: UpdateSyncLogRetry :one
UPDATE osa_sync_log
SET
    retry_count = retry_count + 1,
    error_message = $2
WHERE id = $1
RETURNING *;

-- name: ResolveSyncConflict :one
UPDATE osa_sync_log
SET
    status = 'success',
    conflict_resolution = $2
WHERE id = $1 AND conflict_type IS NOT NULL
RETURNING *;

-- name: DeleteOldSyncLogs :exec
DELETE FROM osa_sync_log
WHERE created_at < $1;
