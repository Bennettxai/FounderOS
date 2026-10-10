-- name: CreateInstallationLog :one
INSERT INTO osa_installation_log (
    workflow_id,
    file_id,
    user_id,
    action,
    source_path,
    destination_path,
    status,
    backup_path,
    backup_content,
    backup_hash,
    error_message,
    error_details,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetInstallationLog :one
SELECT * FROM osa_installation_log
WHERE id = $1;

-- name: ListInstallationLogsByWorkflow :many
SELECT * FROM osa_installation_log
WHERE workflow_id = $1
ORDER BY created_at DESC;

-- name: ListInstallationLogsByFile :many
SELECT * FROM osa_installation_log
WHERE file_id = $1
ORDER BY created_at DESC;

-- name: ListInstallationLogsByUser :many
SELECT * FROM osa_installation_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListInstallationLogsByStatus :many
SELECT * FROM osa_installation_log
WHERE workflow_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: ListFailedInstallations :many
SELECT * FROM osa_installation_log
WHERE status = 'failed'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetInstallationLogStats :one
SELECT
    COUNT(*) as total_installations,
    COUNT(*) FILTER (WHERE status = 'success') as success_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'partial') as partial_count,
    COUNT(*) FILTER (WHERE action = 'install') as install_count,
    COUNT(*) FILTER (WHERE action = 'overwrite') as overwrite_count,
    COUNT(*) FILTER (WHERE action = 'merge') as merge_count,
    COUNT(*) FILTER (WHERE action = 'skip') as skip_count,
    COUNT(*) FILTER (WHERE action = 'rollback') as rollback_count
FROM osa_installation_log
WHERE workflow_id = $1;

-- name: GetRecentInstallations :many
SELECT
    il.*,
    f.file_path,
    f.file_name,
    f.file_type
FROM osa_installation_log il
LEFT JOIN osa_module_files f ON il.file_id = f.id
WHERE il.user_id = $1
ORDER BY il.created_at DESC
LIMIT $2;

-- name: GetInstallationsByDateRange :many
SELECT * FROM osa_installation_log
WHERE created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC;

-- name: CountInstallationsByAction :many
SELECT
    action,
    COUNT(*) as count
FROM osa_installation_log
WHERE workflow_id = $1
GROUP BY action;
