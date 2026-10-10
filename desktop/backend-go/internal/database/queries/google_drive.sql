-- Google Drive and Docs Queries
-- SQLC queries for Google Drive, Docs, Sheets, Slides integration

-- ============================================================================
-- Google Drive Files
-- ============================================================================

-- name: UpsertGoogleDriveFile :one
INSERT INTO google_drive_files (
    user_id, file_id, name, mime_type, file_extension, size_bytes,
    parent_folder_id, parent_folder_name, path, shared, sharing_user,
    permissions, web_view_link, web_content_link, thumbnail_link, icon_link,
    created_time, modified_time, viewed_by_me_time, owners, last_modifying_user,
    synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW())
ON CONFLICT (user_id, file_id) DO UPDATE SET
    name = EXCLUDED.name,
    mime_type = EXCLUDED.mime_type,
    file_extension = EXCLUDED.file_extension,
    size_bytes = EXCLUDED.size_bytes,
    parent_folder_id = EXCLUDED.parent_folder_id,
    parent_folder_name = EXCLUDED.parent_folder_name,
    path = EXCLUDED.path,
    shared = EXCLUDED.shared,
    sharing_user = EXCLUDED.sharing_user,
    permissions = EXCLUDED.permissions,
    web_view_link = EXCLUDED.web_view_link,
    web_content_link = EXCLUDED.web_content_link,
    thumbnail_link = EXCLUDED.thumbnail_link,
    icon_link = EXCLUDED.icon_link,
    modified_time = EXCLUDED.modified_time,
    viewed_by_me_time = EXCLUDED.viewed_by_me_time,
    owners = EXCLUDED.owners,
    last_modifying_user = EXCLUDED.last_modifying_user,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleDriveFile :one
SELECT * FROM google_drive_files
WHERE user_id = $1 AND file_id = $2;

-- name: GetGoogleDriveFileByID :one
SELECT * FROM google_drive_files
WHERE id = $1 AND user_id = $2;

-- name: GetGoogleDriveFilesByUser :many
SELECT * FROM google_drive_files
WHERE user_id = $1
ORDER BY modified_time DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: GetGoogleDriveFilesByFolder :many
SELECT * FROM google_drive_files
WHERE user_id = $1 AND parent_folder_id = $2
ORDER BY name;

-- name: GetGoogleDriveFilesByMimeType :many
SELECT * FROM google_drive_files
WHERE user_id = $1 AND mime_type = $2
ORDER BY modified_time DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: SearchGoogleDriveFiles :many
SELECT * FROM google_drive_files
WHERE user_id = $1 AND name ILIKE $2
ORDER BY modified_time DESC NULLS LAST
LIMIT $3;

-- name: GetGoogleDriveRecentFiles :many
SELECT * FROM google_drive_files
WHERE user_id = $1
ORDER BY viewed_by_me_time DESC NULLS LAST
LIMIT $2;

-- name: GetGoogleDriveSharedFiles :many
SELECT * FROM google_drive_files
WHERE user_id = $1 AND shared = true
ORDER BY modified_time DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: DeleteGoogleDriveFile :exec
DELETE FROM google_drive_files
WHERE user_id = $1 AND file_id = $2;

-- name: DeleteGoogleDriveFilesByUser :exec
DELETE FROM google_drive_files WHERE user_id = $1;

-- ============================================================================
-- Google Docs
-- ============================================================================

-- name: UpsertGoogleDoc :one
INSERT INTO google_docs (
    user_id, document_id, drive_file_id, title, body_text, word_count,
    headers, locale, created_time, modified_time, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (user_id, document_id) DO UPDATE SET
    drive_file_id = EXCLUDED.drive_file_id,
    title = EXCLUDED.title,
    body_text = EXCLUDED.body_text,
    word_count = EXCLUDED.word_count,
    headers = EXCLUDED.headers,
    locale = EXCLUDED.locale,
    modified_time = EXCLUDED.modified_time,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleDoc :one
SELECT * FROM google_docs
WHERE user_id = $1 AND document_id = $2;

-- name: GetGoogleDocsByUser :many
SELECT * FROM google_docs
WHERE user_id = $1
ORDER BY modified_time DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGoogleDocs :many
SELECT * FROM google_docs
WHERE user_id = $1
  AND to_tsvector('english', body_text) @@ plainto_tsquery('english', $2)
ORDER BY ts_rank(to_tsvector('english', body_text), plainto_tsquery('english', $2)) DESC
LIMIT $3;

-- name: SearchGoogleDocsByTitle :many
SELECT * FROM google_docs
WHERE user_id = $1 AND title ILIKE $2
ORDER BY modified_time DESC NULLS LAST
LIMIT $3;

-- name: DeleteGoogleDoc :exec
DELETE FROM google_docs
WHERE user_id = $1 AND document_id = $2;

-- name: DeleteGoogleDocsByUser :exec
DELETE FROM google_docs WHERE user_id = $1;

-- ============================================================================
-- Google Sheets
-- ============================================================================

-- name: UpsertGoogleSheet :one
INSERT INTO google_sheets (
    user_id, spreadsheet_id, drive_file_id, title, locale, time_zone,
    sheet_count, sheets, named_ranges, created_time, modified_time, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (user_id, spreadsheet_id) DO UPDATE SET
    drive_file_id = EXCLUDED.drive_file_id,
    title = EXCLUDED.title,
    locale = EXCLUDED.locale,
    time_zone = EXCLUDED.time_zone,
    sheet_count = EXCLUDED.sheet_count,
    sheets = EXCLUDED.sheets,
    named_ranges = EXCLUDED.named_ranges,
    modified_time = EXCLUDED.modified_time,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleSheet :one
SELECT * FROM google_sheets
WHERE user_id = $1 AND spreadsheet_id = $2;

-- name: GetGoogleSheetsByUser :many
SELECT * FROM google_sheets
WHERE user_id = $1
ORDER BY modified_time DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGoogleSheets :many
SELECT * FROM google_sheets
WHERE user_id = $1 AND title ILIKE $2
ORDER BY modified_time DESC NULLS LAST
LIMIT $3;

-- name: DeleteGoogleSheet :exec
DELETE FROM google_sheets
WHERE user_id = $1 AND spreadsheet_id = $2;

-- name: DeleteGoogleSheetsByUser :exec
DELETE FROM google_sheets WHERE user_id = $1;

-- ============================================================================
-- Google Slides
-- ============================================================================

-- name: UpsertGoogleSlide :one
INSERT INTO google_slides (
    user_id, presentation_id, drive_file_id, title, locale,
    slide_count, slides, page_width, page_height, created_time, modified_time, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (user_id, presentation_id) DO UPDATE SET
    drive_file_id = EXCLUDED.drive_file_id,
    title = EXCLUDED.title,
    locale = EXCLUDED.locale,
    slide_count = EXCLUDED.slide_count,
    slides = EXCLUDED.slides,
    page_width = EXCLUDED.page_width,
    page_height = EXCLUDED.page_height,
    modified_time = EXCLUDED.modified_time,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleSlide :one
SELECT * FROM google_slides
WHERE user_id = $1 AND presentation_id = $2;

-- name: GetGoogleSlidesByUser :many
SELECT * FROM google_slides
WHERE user_id = $1
ORDER BY modified_time DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGoogleSlides :many
SELECT * FROM google_slides
WHERE user_id = $1 AND title ILIKE $2
ORDER BY modified_time DESC NULLS LAST
LIMIT $3;

-- name: DeleteGoogleSlide :exec
DELETE FROM google_slides
WHERE user_id = $1 AND presentation_id = $2;

-- name: DeleteGoogleSlidesByUser :exec
DELETE FROM google_slides WHERE user_id = $1;
