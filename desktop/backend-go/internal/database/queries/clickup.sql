-- ClickUp Queries
-- SQLC queries for ClickUp integration

-- ============================================================================
-- ClickUp Workspaces
-- ============================================================================

-- name: UpsertClickUpWorkspace :one
INSERT INTO clickup_workspaces (
    user_id, workspace_id, name, color, avatar, member_count, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id, workspace_id) DO UPDATE SET
    name = EXCLUDED.name,
    color = EXCLUDED.color,
    avatar = EXCLUDED.avatar,
    member_count = EXCLUDED.member_count,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetClickUpWorkspace :one
SELECT * FROM clickup_workspaces
WHERE user_id = $1 AND workspace_id = $2;

-- name: GetClickUpWorkspacesByUser :many
SELECT * FROM clickup_workspaces
WHERE user_id = $1
ORDER BY name;

-- name: DeleteClickUpWorkspacesByUser :exec
DELETE FROM clickup_workspaces WHERE user_id = $1;

-- ============================================================================
-- ClickUp Spaces
-- ============================================================================

-- name: UpsertClickUpSpace :one
INSERT INTO clickup_spaces (
    user_id, space_id, workspace_id, name, color, private, archived, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, space_id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    name = EXCLUDED.name,
    color = EXCLUDED.color,
    private = EXCLUDED.private,
    archived = EXCLUDED.archived,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetClickUpSpace :one
SELECT * FROM clickup_spaces
WHERE user_id = $1 AND space_id = $2;

-- name: GetClickUpSpacesByWorkspace :many
SELECT * FROM clickup_spaces
WHERE user_id = $1 AND workspace_id = $2 AND archived = false
ORDER BY name;

-- name: GetClickUpSpacesByUser :many
SELECT * FROM clickup_spaces
WHERE user_id = $1 AND archived = false
ORDER BY name;

-- name: DeleteClickUpSpacesByUser :exec
DELETE FROM clickup_spaces WHERE user_id = $1;

-- ============================================================================
-- ClickUp Folders
-- ============================================================================

-- name: UpsertClickUpFolder :one
INSERT INTO clickup_folders (
    user_id, folder_id, space_id, name, hidden, archived, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id, folder_id) DO UPDATE SET
    space_id = EXCLUDED.space_id,
    name = EXCLUDED.name,
    hidden = EXCLUDED.hidden,
    archived = EXCLUDED.archived,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetClickUpFolder :one
SELECT * FROM clickup_folders
WHERE user_id = $1 AND folder_id = $2;

-- name: GetClickUpFoldersBySpace :many
SELECT * FROM clickup_folders
WHERE user_id = $1 AND space_id = $2 AND archived = false
ORDER BY name;

-- name: DeleteClickUpFoldersByUser :exec
DELETE FROM clickup_folders WHERE user_id = $1;

-- ============================================================================
-- ClickUp Lists
-- ============================================================================

-- name: UpsertClickUpList :one
INSERT INTO clickup_lists (
    user_id, list_id, folder_id, space_id, name, archived, task_count, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, list_id) DO UPDATE SET
    folder_id = EXCLUDED.folder_id,
    space_id = EXCLUDED.space_id,
    name = EXCLUDED.name,
    archived = EXCLUDED.archived,
    task_count = EXCLUDED.task_count,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetClickUpList :one
SELECT * FROM clickup_lists
WHERE user_id = $1 AND list_id = $2;

-- name: GetClickUpListsByFolder :many
SELECT * FROM clickup_lists
WHERE user_id = $1 AND folder_id = $2 AND archived = false
ORDER BY name;

-- name: GetClickUpListsBySpace :many
SELECT * FROM clickup_lists
WHERE user_id = $1 AND space_id = $2 AND archived = false
ORDER BY name;

-- name: GetClickUpListsByUser :many
SELECT * FROM clickup_lists
WHERE user_id = $1 AND archived = false
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: DeleteClickUpListsByUser :exec
DELETE FROM clickup_lists WHERE user_id = $1;

-- ============================================================================
-- ClickUp Tasks
-- ============================================================================

-- name: UpsertClickUpTask :one
INSERT INTO clickup_tasks (
    user_id, task_id, custom_id, list_id, folder_id, space_id,
    name, description, status, status_color, priority, priority_color,
    due_date, start_date, date_created, date_updated, date_closed,
    time_spent, time_estimate, parent_task_id, assignees, creator, tags, url, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NOW())
ON CONFLICT (user_id, task_id) DO UPDATE SET
    custom_id = EXCLUDED.custom_id,
    list_id = EXCLUDED.list_id,
    folder_id = EXCLUDED.folder_id,
    space_id = EXCLUDED.space_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    status_color = EXCLUDED.status_color,
    priority = EXCLUDED.priority,
    priority_color = EXCLUDED.priority_color,
    due_date = EXCLUDED.due_date,
    start_date = EXCLUDED.start_date,
    date_updated = EXCLUDED.date_updated,
    date_closed = EXCLUDED.date_closed,
    time_spent = EXCLUDED.time_spent,
    time_estimate = EXCLUDED.time_estimate,
    parent_task_id = EXCLUDED.parent_task_id,
    assignees = EXCLUDED.assignees,
    creator = EXCLUDED.creator,
    tags = EXCLUDED.tags,
    url = EXCLUDED.url,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetClickUpTask :one
SELECT * FROM clickup_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: GetClickUpTasksByList :many
SELECT * FROM clickup_tasks
WHERE user_id = $1 AND list_id = $2
ORDER BY date_created DESC
LIMIT $3 OFFSET $4;

-- name: GetClickUpTasksBySpace :many
SELECT * FROM clickup_tasks
WHERE user_id = $1 AND space_id = $2
ORDER BY date_created DESC
LIMIT $3 OFFSET $4;

-- name: GetClickUpTasksByStatus :many
SELECT * FROM clickup_tasks
WHERE user_id = $1 AND status = $2
ORDER BY due_date NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetClickUpTasksDue :many
SELECT * FROM clickup_tasks
WHERE user_id = $1
  AND due_date IS NOT NULL
  AND due_date <= $2
  AND date_closed IS NULL
ORDER BY due_date;

-- name: SearchClickUpTasks :many
SELECT * FROM clickup_tasks
WHERE user_id = $1
  AND (name ILIKE $2 OR description ILIKE $2)
ORDER BY date_updated DESC NULLS LAST
LIMIT $3;

-- name: CountClickUpTasksByStatus :many
SELECT status, COUNT(*) as count
FROM clickup_tasks
WHERE user_id = $1 AND date_closed IS NULL
GROUP BY status
ORDER BY count DESC;

-- name: DeleteClickUpTask :exec
DELETE FROM clickup_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: DeleteClickUpTasksByUser :exec
DELETE FROM clickup_tasks WHERE user_id = $1;
