-- ============================================================================
-- ATTACHMENTS QUERIES
-- ============================================================================

-- name: ListEntityAttachments :many
SELECT * FROM attachments
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC;

-- name: ListUserAttachments :many
SELECT * FROM attachments
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: GetAttachment :one
SELECT * FROM attachments
WHERE id = $1;

-- name: CreateAttachment :one
INSERT INTO attachments (
    user_id, entity_type, entity_id,
    file_name, file_size, mime_type, file_extension,
    storage_provider, storage_path, storage_bucket,
    thumbnail_url, preview_url,
    width, height, page_count, duration_seconds,
    processing_status, metadata, uploaded_by, folder_id
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    $8, $9, $10,
    $11, $12,
    $13, $14, $15, $16,
    $17, $18, $19, $20
)
RETURNING *;

-- name: UpdateAttachment :one
UPDATE attachments
SET file_name = $2,
    thumbnail_url = $3,
    preview_url = $4,
    processing_status = $5,
    processing_error = $6,
    metadata = $7,
    folder_id = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateAttachmentProcessing :exec
UPDATE attachments
SET processing_status = $2,
    processing_error = $3,
    thumbnail_url = $4,
    preview_url = $5,
    metadata = $6,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteAttachment :exec
DELETE FROM attachments
WHERE id = $1;

-- name: DeleteEntityAttachments :exec
DELETE FROM attachments
WHERE entity_type = $1 AND entity_id = $2;

-- name: GetPendingAttachments :many
SELECT * FROM attachments
WHERE processing_status IN ('pending', 'processing')
ORDER BY created_at ASC
LIMIT sqlc.arg(limit_val)::int;

-- name: GetUserStorageUsage :one
SELECT COALESCE(SUM(file_size), 0)::bigint as total_bytes
FROM attachments
WHERE user_id = $1;

-- name: ListAttachmentsByMimeType :many
SELECT * FROM attachments
WHERE user_id = $1 AND mime_type LIKE $2
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- ============================================================================
-- ATTACHMENT VERSIONS QUERIES
-- ============================================================================

-- name: ListAttachmentVersions :many
SELECT * FROM attachment_versions
WHERE attachment_id = $1
ORDER BY version_number DESC;

-- name: CreateAttachmentVersion :one
INSERT INTO attachment_versions (attachment_id, version_number, version_label, file_size, storage_path, storage_bucket, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLatestVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) as max_version
FROM attachment_versions
WHERE attachment_id = $1;

-- ============================================================================
-- ATTACHMENT FOLDERS QUERIES
-- ============================================================================

-- name: ListAttachmentFolders :many
SELECT * FROM attachment_folders
WHERE user_id = $1
  AND (sqlc.narg(parent_id)::uuid IS NULL AND parent_id IS NULL OR parent_id = sqlc.narg(parent_id))
ORDER BY name ASC;

-- name: GetAttachmentFolder :one
SELECT * FROM attachment_folders
WHERE id = $1 AND user_id = $2;

-- name: CreateAttachmentFolder :one
INSERT INTO attachment_folders (user_id, parent_id, name, description, color, entity_type, entity_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAttachmentFolder :one
UPDATE attachment_folders
SET name = $2, description = $3, color = $4, parent_id = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAttachmentFolder :exec
DELETE FROM attachment_folders
WHERE id = $1 AND user_id = $2;

-- name: ListFolderAttachments :many
SELECT * FROM attachments
WHERE folder_id = $1
ORDER BY created_at DESC;
