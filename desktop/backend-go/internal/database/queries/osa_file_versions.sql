-- name: CreateFileVersion :one
INSERT INTO osa_file_versions (
    file_id,
    workflow_id,
    version_number,
    content,
    content_hash,
    file_size_bytes,
    change_type,
    change_summary,
    diff_from_previous,
    lines_added,
    lines_removed,
    created_by_workflow_type,
    created_by_user_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetFileVersion :one
SELECT * FROM osa_file_versions
WHERE id = $1;

-- name: GetFileVersionByNumber :one
SELECT * FROM osa_file_versions
WHERE file_id = $1 AND version_number = $2;

-- name: ListFileVersions :many
SELECT * FROM osa_file_versions
WHERE file_id = $1
ORDER BY version_number DESC;

-- name: ListFileVersionsByWorkflow :many
SELECT * FROM osa_file_versions
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestFileVersion :one
SELECT * FROM osa_file_versions
WHERE file_id = $1
ORDER BY version_number DESC
LIMIT 1;

-- name: GetFileVersionByHash :one
SELECT * FROM osa_file_versions
WHERE file_id = $1 AND content_hash = $2
LIMIT 1;

-- name: CountFileVersions :one
SELECT COUNT(*) FROM osa_file_versions
WHERE file_id = $1;

-- name: GetOSAFileVersionDiff :one
SELECT
    v1.version_number as from_version,
    v2.version_number as to_version,
    v1.content as from_content,
    v2.content as to_content,
    v2.diff_from_previous,
    v2.lines_added,
    v2.lines_removed,
    v2.change_type,
    v2.change_summary
FROM osa_file_versions v1
JOIN osa_file_versions v2 ON v1.file_id = v2.file_id
WHERE v1.file_id = $1
AND v1.version_number = $2
AND v2.version_number = $3;

-- name: ListRecentVersions :many
SELECT
    fv.*,
    f.file_path,
    f.file_name,
    f.file_type
FROM osa_file_versions fv
JOIN osa_module_files f ON fv.file_id = f.id
WHERE fv.workflow_id = $1
ORDER BY fv.created_at DESC
LIMIT $2;

-- name: GetOSAFileVersionStats :one
SELECT
    COUNT(*) as total_versions,
    COUNT(DISTINCT file_id) as unique_files,
    SUM(lines_added) as total_lines_added,
    SUM(lines_removed) as total_lines_removed,
    AVG(file_size_bytes) as avg_file_size
FROM osa_file_versions
WHERE workflow_id = $1;

-- name: DeleteOldFileVersions :exec
DELETE FROM osa_file_versions fv_del
WHERE fv_del.file_id = $1
AND fv_del.version_number < (
    SELECT MAX(fv_sub.version_number) - $2
    FROM osa_file_versions fv_sub
    WHERE fv_sub.file_id = $1
);
