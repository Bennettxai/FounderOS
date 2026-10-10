-- name: CreateGeneratedFile :one
INSERT INTO osa_module_files (
    workflow_id,
    module_instance_id,
    file_path,
    file_name,
    file_type,
    language,
    content,
    content_hash,
    file_size_bytes,
    line_count,
    encoding,
    purpose,
    dependencies,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetGeneratedFile :one
SELECT * FROM osa_module_files
WHERE id = $1;

-- name: GetFileByPath :one
SELECT * FROM osa_module_files
WHERE workflow_id = $1 AND file_path = $2
LIMIT 1;

-- name: GetFileByHash :one
SELECT * FROM osa_module_files
WHERE content_hash = $1
LIMIT 1;

-- name: ListFilesByWorkflow :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1
ORDER BY file_path ASC;

-- name: ListFilesByApp :many
SELECT * FROM osa_module_files
WHERE module_instance_id = $1
ORDER BY created_at DESC;

-- name: ListFilesByType :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1 AND file_type = $2
ORDER BY file_path ASC;

-- name: ListFilesByLanguage :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1 AND language = $2
ORDER BY file_path ASC;

-- name: ListFilesByStatus :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1 AND installation_status = $2
ORDER BY file_path ASC;

-- name: ListPendingFiles :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1 AND installation_status = 'pending'
ORDER BY file_path ASC;

-- name: ListLatestFiles :many
SELECT * FROM osa_module_files
WHERE is_latest = true AND workflow_id = $1
ORDER BY file_path ASC;

-- name: UpdateFileContent :one
UPDATE osa_module_files
SET
    content = $2,
    content_hash = $3,
    file_size_bytes = $4,
    line_count = $5
WHERE id = $1
RETURNING *;

-- name: UpdateFileInstallationStatus :one
UPDATE osa_module_files
SET
    installation_status = $2,
    installation_path = COALESCE($3, installation_path),
    installed_at = CASE WHEN $2 = 'installed' THEN NOW() ELSE installed_at END
WHERE id = $1
RETURNING *;

-- name: MarkFileConflict :one
UPDATE osa_module_files
SET
    installation_status = 'conflict',
    conflict_reason = $2,
    conflict_resolution = $3
WHERE id = $1
RETURNING *;

-- name: ResolveFileConflict :one
UPDATE osa_module_files
SET
    installation_status = $2,
    conflict_resolved_by = $3,
    conflict_resolution = $4
WHERE id = $1 AND installation_status = 'conflict'
RETURNING *;

-- name: MarkFileAsLatest :one
UPDATE osa_module_files
SET is_latest = true
WHERE id = $1
RETURNING *;

-- name: UnmarkPreviousVersions :exec
UPDATE osa_module_files
SET is_latest = false
WHERE file_path = $1 AND workflow_id = $2 AND id != $3;

-- name: DeleteGeneratedFile :exec
DELETE FROM osa_module_files
WHERE id = $1;

-- name: CountFilesByWorkflow :one
SELECT COUNT(*) FROM osa_module_files
WHERE workflow_id = $1;

-- name: CountFilesByType :one
SELECT file_type, COUNT(*) as count
FROM osa_module_files
WHERE workflow_id = $1
GROUP BY file_type;

-- name: GetFileStats :one
SELECT
    COUNT(*) as total_files,
    SUM(file_size_bytes) as total_size_bytes,
    SUM(line_count) as total_lines,
    COUNT(DISTINCT file_type) as file_types_count,
    COUNT(DISTINCT language) as languages_count,
    COUNT(*) FILTER (WHERE installation_status = 'installed') as installed_count,
    COUNT(*) FILTER (WHERE installation_status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE installation_status = 'conflict') as conflict_count
FROM osa_module_files
WHERE workflow_id = $1;

-- name: SearchFiles :many
SELECT * FROM osa_module_files
WHERE
    workflow_id = $1 AND
    (
        file_path ILIKE '%' || $2 || '%' OR
        file_name ILIKE '%' || $2 || '%' OR
        content ILIKE '%' || $2 || '%'
    )
ORDER BY file_path ASC
LIMIT $3 OFFSET $4;

-- name: GetFileDependencies :many
SELECT f.* FROM osa_module_files f
WHERE f.workflow_id = $1
AND f.file_path = ANY(
    SELECT unnest(dependencies)
    FROM osa_module_files dep
    WHERE dep.id = $2
);

-- name: GetFileDependents :many
SELECT * FROM osa_module_files
WHERE workflow_id = $1
AND $2 = ANY(dependencies);

-- name: ListFileMetadataByApp :many
-- Fetches file metadata (no content) for an app, for efficient snapshot comparison
SELECT id, module_instance_id, file_path, file_name, file_type, language,
       content_hash, file_size_bytes, line_count, created_at
FROM osa_module_files
WHERE module_instance_id = $1
ORDER BY file_path ASC;

-- name: GetFileContentByAppAndPath :one
-- Fetches file content for a specific path within an app (for diff generation)
SELECT id, file_path, content, content_hash, file_size_bytes, line_count
FROM osa_module_files
WHERE module_instance_id = $1 AND file_path = $2
LIMIT 1;
