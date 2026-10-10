-- Airtable Queries
-- SQLC queries for Airtable integration

-- ============================================================================
-- Airtable Bases
-- ============================================================================

-- name: UpsertAirtableBase :one
INSERT INTO airtable_bases (
    user_id, base_id, name, permission_level, synced_at
) VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id, base_id) DO UPDATE SET
    name = EXCLUDED.name,
    permission_level = EXCLUDED.permission_level,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetAirtableBase :one
SELECT * FROM airtable_bases
WHERE user_id = $1 AND base_id = $2;

-- name: GetAirtableBasesByUser :many
SELECT * FROM airtable_bases
WHERE user_id = $1
ORDER BY name;

-- name: DeleteAirtableBase :exec
DELETE FROM airtable_bases
WHERE user_id = $1 AND base_id = $2;

-- name: DeleteAirtableBasesByUser :exec
DELETE FROM airtable_bases WHERE user_id = $1;

-- ============================================================================
-- Airtable Tables
-- ============================================================================

-- name: UpsertAirtableTable :one
INSERT INTO airtable_tables (
    user_id, table_id, base_id, name, primary_field_id, fields, views, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, table_id) DO UPDATE SET
    base_id = EXCLUDED.base_id,
    name = EXCLUDED.name,
    primary_field_id = EXCLUDED.primary_field_id,
    fields = EXCLUDED.fields,
    views = EXCLUDED.views,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetAirtableTable :one
SELECT * FROM airtable_tables
WHERE user_id = $1 AND table_id = $2;

-- name: GetAirtableTablesByBase :many
SELECT * FROM airtable_tables
WHERE user_id = $1 AND base_id = $2
ORDER BY name;

-- name: GetAirtableTablesByUser :many
SELECT * FROM airtable_tables
WHERE user_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: SearchAirtableTables :many
SELECT * FROM airtable_tables
WHERE user_id = $1 AND name ILIKE $2
ORDER BY name
LIMIT $3;

-- name: DeleteAirtableTable :exec
DELETE FROM airtable_tables
WHERE user_id = $1 AND table_id = $2;

-- name: DeleteAirtableTablesByBase :exec
DELETE FROM airtable_tables
WHERE user_id = $1 AND base_id = $2;

-- name: DeleteAirtableTablesByUser :exec
DELETE FROM airtable_tables WHERE user_id = $1;

-- ============================================================================
-- Airtable Records
-- ============================================================================

-- name: UpsertAirtableRecord :one
INSERT INTO airtable_records (
    user_id, record_id, table_id, base_id, fields, created_time_airtable, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id, record_id) DO UPDATE SET
    table_id = EXCLUDED.table_id,
    base_id = EXCLUDED.base_id,
    fields = EXCLUDED.fields,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetAirtableRecord :one
SELECT * FROM airtable_records
WHERE user_id = $1 AND record_id = $2;

-- name: GetAirtableRecordsByTable :many
SELECT * FROM airtable_records
WHERE user_id = $1 AND table_id = $2
ORDER BY created_time_airtable DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetAirtableRecordsByBase :many
SELECT * FROM airtable_records
WHERE user_id = $1 AND base_id = $2
ORDER BY created_time_airtable DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: CountAirtableRecordsByTable :one
SELECT COUNT(*) FROM airtable_records
WHERE user_id = $1 AND table_id = $2;

-- name: CountAirtableRecordsByBase :one
SELECT COUNT(*) FROM airtable_records
WHERE user_id = $1 AND base_id = $2;

-- name: DeleteAirtableRecord :exec
DELETE FROM airtable_records
WHERE user_id = $1 AND record_id = $2;

-- name: DeleteAirtableRecordsByTable :exec
DELETE FROM airtable_records
WHERE user_id = $1 AND table_id = $2;

-- name: DeleteAirtableRecordsByBase :exec
DELETE FROM airtable_records
WHERE user_id = $1 AND base_id = $2;

-- name: DeleteAirtableRecordsByUser :exec
DELETE FROM airtable_records WHERE user_id = $1;
