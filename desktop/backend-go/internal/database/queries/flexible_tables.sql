-- ============================================================================
-- CUSTOM TABLES QUERIES
-- ============================================================================

-- name: ListCustomTables :many
SELECT * FROM custom_tables
WHERE user_id = $1
  AND (sqlc.narg(workspace_id)::uuid IS NULL OR workspace_id = sqlc.narg(workspace_id))
ORDER BY updated_at DESC;

-- name: GetCustomTable :one
SELECT * FROM custom_tables
WHERE id = $1 AND user_id = $2;

-- name: CreateCustomTable :one
INSERT INTO custom_tables (user_id, name, description, icon, color, workspace_id, settings)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateCustomTable :one
UPDATE custom_tables
SET name = $2, description = $3, icon = $4, color = $5, settings = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomTable :exec
DELETE FROM custom_tables
WHERE id = $1 AND user_id = $2;

-- ============================================================================
-- CUSTOM FIELDS QUERIES
-- ============================================================================

-- name: ListCustomFields :many
SELECT * FROM custom_fields
WHERE table_id = $1
ORDER BY position ASC;

-- name: GetCustomField :one
SELECT * FROM custom_fields
WHERE id = $1;

-- name: CreateCustomField :one
INSERT INTO custom_fields (table_id, name, field_type, description, position, config, required, unique_values)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateCustomField :one
UPDATE custom_fields
SET name = $2, description = $3, config = $4, required = $5, hidden = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCustomFieldPosition :exec
UPDATE custom_fields
SET position = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteCustomField :exec
DELETE FROM custom_fields
WHERE id = $1;

-- ============================================================================
-- CUSTOM FIELD OPTIONS QUERIES
-- ============================================================================

-- name: ListFieldOptions :many
SELECT * FROM custom_field_options
WHERE field_id = $1
ORDER BY position ASC;

-- name: CreateFieldOption :one
INSERT INTO custom_field_options (field_id, name, color, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateFieldOption :one
UPDATE custom_field_options
SET name = $2, color = $3, position = $4
WHERE id = $1
RETURNING *;

-- name: DeleteFieldOption :exec
DELETE FROM custom_field_options
WHERE id = $1;

-- ============================================================================
-- CUSTOM RECORDS QUERIES
-- ============================================================================

-- name: ListCustomRecords :many
SELECT * FROM custom_records
WHERE table_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: GetCustomRecord :one
SELECT * FROM custom_records
WHERE id = $1;

-- name: CreateCustomRecord :one
INSERT INTO custom_records (table_id, data, created_by)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateCustomRecord :one
UPDATE custom_records
SET data = $2, modified_by = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCustomRecordField :one
UPDATE custom_records
SET data = data || jsonb_build_object($2::text, $3::jsonb),
    modified_by = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomRecord :exec
DELETE FROM custom_records
WHERE id = $1;

-- name: CountCustomRecords :one
SELECT COUNT(*) as count FROM custom_records
WHERE table_id = $1;

-- name: SearchCustomRecords :many
SELECT * FROM custom_records
WHERE table_id = $1
  AND data @> $2::jsonb
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- ============================================================================
-- CUSTOM VIEWS QUERIES
-- ============================================================================

-- name: ListCustomViews :many
SELECT * FROM custom_views
WHERE table_id = $1
ORDER BY position ASC;

-- name: GetCustomView :one
SELECT * FROM custom_views
WHERE id = $1;

-- name: GetDefaultView :one
SELECT * FROM custom_views
WHERE table_id = $1 AND is_default = TRUE
LIMIT 1;

-- name: CreateCustomView :one
INSERT INTO custom_views (table_id, name, view_type, description, config, filters, sorts, group_by, view_settings, position)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateCustomView :one
UPDATE custom_views
SET name = $2, description = $3, config = $4, filters = $5, sorts = $6, group_by = $7, view_settings = $8, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetDefaultView :exec
UPDATE custom_views
SET is_default = (id = $2)
WHERE table_id = $1;

-- name: DeleteCustomView :exec
DELETE FROM custom_views
WHERE id = $1;

-- ============================================================================
-- RECORD HISTORY QUERIES
-- ============================================================================

-- name: ListRecordHistory :many
SELECT * FROM custom_record_history
WHERE record_id = $1
ORDER BY changed_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: CreateRecordHistory :one
INSERT INTO custom_record_history (record_id, field_id, action, old_value, new_value, changed_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- ============================================================================
-- WORKSPACES QUERIES
-- ============================================================================

-- name: ListCustomWorkspaces :many
SELECT * FROM custom_workspaces
WHERE user_id = $1
ORDER BY name ASC;

-- name: GetCustomWorkspace :one
SELECT * FROM custom_workspaces
WHERE id = $1 AND user_id = $2;

-- name: CreateCustomWorkspace :one
INSERT INTO custom_workspaces (user_id, name, description, icon, color, visibility)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateCustomWorkspace :one
UPDATE custom_workspaces
SET name = $2, description = $3, icon = $4, color = $5, visibility = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomWorkspace :exec
DELETE FROM custom_workspaces
WHERE id = $1 AND user_id = $2;
