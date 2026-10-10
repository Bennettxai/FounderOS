-- ================================================================
-- App Versions Queries
-- Description: Queries for app versioning and snapshot management
-- ================================================================

-- ================================================================
-- CREATE / SNAPSHOT
-- ================================================================

-- name: CreateAppVersion :one
INSERT INTO app_versions (
    app_id,
    version_number,
    snapshot_data,
    snapshot_metadata,
    change_summary,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: CreateAutoSnapshot :one
-- Create automatic snapshot with generated version number
INSERT INTO app_versions (
    app_id,
    version_number,
    snapshot_data,
    snapshot_metadata,
    change_summary,
    created_by
)
SELECT
    $1::uuid as app_id,
    $2::varchar as version_number,
    $3::jsonb as snapshot_data,
    $4::jsonb as snapshot_metadata,
    'Auto-snapshot' as change_summary,
    $5::uuid as created_by
RETURNING *;

-- ================================================================
-- READ / GET
-- ================================================================

-- name: GetAppVersion :one
SELECT * FROM app_versions
WHERE id = $1 LIMIT 1;

-- name: GetAppVersionByNumber :one
SELECT * FROM app_versions
WHERE app_id = $1 AND version_number = $2
LIMIT 1;

-- name: GetLatestAppVersion :one
SELECT * FROM app_versions
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListAppVersions :many
SELECT * FROM app_versions
WHERE app_id = $1
ORDER BY created_at DESC;

-- name: ListAppVersionsPaginated :many
SELECT * FROM app_versions
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAppVersions :one
SELECT COUNT(*) FROM app_versions
WHERE app_id = $1;

-- name: ListVersionsByUser :many
-- Get all versions created by a specific user
SELECT * FROM app_versions
WHERE created_by = $1
ORDER BY created_at DESC
LIMIT $2;

-- ================================================================
-- VERSION HISTORY
-- ================================================================

-- name: GetVersionHistory :many
-- Get detailed version history with user info
SELECT
    av.*,
    u.name as creator_name,
    u.email as creator_email
FROM app_versions av
LEFT JOIN "user" u ON av.created_by = u.id
WHERE av.app_id = $1
ORDER BY av.created_at DESC;

-- name: GetVersionDiff :one
-- Get two versions for comparison (returns both as array)
SELECT
    jsonb_build_object(
        'old_version', (SELECT row_to_json(v1) FROM app_versions v1 WHERE v1.app_id = $1 AND v1.version_number = $2),
        'new_version', (SELECT row_to_json(v2) FROM app_versions v2 WHERE v2.app_id = $1 AND v2.version_number = $3)
    ) as diff_data;

-- ================================================================
-- RESTORE / ROLLBACK
-- ================================================================

-- name: GetRestoreData :one
-- Get snapshot data for restoring to a specific version
SELECT
    id,
    version_number,
    snapshot_data,
    snapshot_metadata,
    change_summary,
    created_at
FROM app_versions
WHERE app_id = $1 AND version_number = $2
LIMIT 1;

-- name: GetLatestRestoreData :one
-- Get latest snapshot data for restoring
SELECT
    id,
    version_number,
    snapshot_data,
    snapshot_metadata,
    change_summary,
    created_at
FROM app_versions
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- ================================================================
-- UPDATE
-- ================================================================

-- name: UpdateVersionSummary :one
UPDATE app_versions
SET change_summary = $2
WHERE id = $1
RETURNING *;

-- name: UpdateVersionMetadata :one
UPDATE app_versions
SET snapshot_metadata = $2
WHERE id = $1
RETURNING *;

-- ================================================================
-- DELETE / CLEANUP
-- ================================================================

-- name: DeleteAppVersion :exec
DELETE FROM app_versions
WHERE id = $1;

-- name: DeleteAppVersionByNumber :exec
DELETE FROM app_versions
WHERE app_id = $1 AND version_number = $2;

-- name: DeleteOldVersions :exec
-- Clean up old versions, keeping only the most recent N versions per app
DELETE FROM app_versions av
WHERE av.app_id = $1
AND av.id NOT IN (
    SELECT sub.id FROM app_versions sub
    WHERE sub.app_id = $1
    ORDER BY sub.created_at DESC
    LIMIT $2
);

-- name: DeleteVersionsOlderThan :exec
-- Delete versions older than a specific date
DELETE FROM app_versions
WHERE app_id = $1
AND created_at < $2;

-- ================================================================
-- ANALYTICS
-- ================================================================

-- name: GetVersionStats :one
-- Get statistics about versions for an app
SELECT
    COUNT(*) as total_versions,
    MIN(created_at) as first_version_at,
    MAX(created_at) as latest_version_at,
    COUNT(DISTINCT created_by) as unique_creators
FROM app_versions
WHERE app_id = $1;

-- name: GetVersionSize :one
-- Get approximate size of version data
SELECT
    id,
    version_number,
    pg_column_size(snapshot_data) as snapshot_size_bytes,
    pg_column_size(snapshot_metadata) as metadata_size_bytes
FROM app_versions
WHERE id = $1;

-- name: ListLargeVersions :many
-- Find versions with large snapshot data (for cleanup)
SELECT
    id,
    app_id,
    version_number,
    pg_column_size(snapshot_data) as snapshot_size_bytes,
    created_at
FROM app_versions
WHERE app_id = $1
AND pg_column_size(snapshot_data) > $2
ORDER BY snapshot_size_bytes DESC;

-- ================================================================
-- SEARCH
-- ================================================================

-- name: SearchVersionsByChangeSummary :many
-- Search versions by change summary text
SELECT * FROM app_versions
WHERE app_id = $1
AND change_summary ILIKE '%' || $2 || '%'
ORDER BY created_at DESC;

-- name: FindVersionBySnapshotContent :many
-- Search versions by content in snapshot_data (JSONB query)
SELECT * FROM app_versions av
WHERE av.app_id = $1
AND av.snapshot_data @> $2::jsonb
ORDER BY av.created_at DESC;
