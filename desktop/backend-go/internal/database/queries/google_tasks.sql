-- Google Tasks Queries
-- SQLC queries for Google Tasks integration

-- ============================================================================
-- Google Task Lists
-- ============================================================================

-- name: UpsertGoogleTaskList :one
INSERT INTO google_task_lists (
    user_id, task_list_id, title, kind, updated, synced_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (user_id, task_list_id) DO UPDATE SET
    title = EXCLUDED.title,
    kind = EXCLUDED.kind,
    updated = EXCLUDED.updated,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleTaskList :one
SELECT * FROM google_task_lists
WHERE user_id = $1 AND task_list_id = $2;

-- name: GetGoogleTaskListsByUser :many
SELECT * FROM google_task_lists
WHERE user_id = $1
ORDER BY title;

-- name: DeleteGoogleTaskList :exec
DELETE FROM google_task_lists
WHERE user_id = $1 AND task_list_id = $2;

-- name: DeleteGoogleTaskListsByUser :exec
DELETE FROM google_task_lists WHERE user_id = $1;

-- ============================================================================
-- Google Tasks
-- ============================================================================

-- name: UpsertGoogleTask :one
INSERT INTO google_tasks (
    user_id, task_id, task_list_id, title, notes, status,
    due, completed, deleted, hidden, parent_task_id, position,
    links, updated, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
ON CONFLICT (user_id, task_id) DO UPDATE SET
    task_list_id = EXCLUDED.task_list_id,
    title = EXCLUDED.title,
    notes = EXCLUDED.notes,
    status = EXCLUDED.status,
    due = EXCLUDED.due,
    completed = EXCLUDED.completed,
    deleted = EXCLUDED.deleted,
    hidden = EXCLUDED.hidden,
    parent_task_id = EXCLUDED.parent_task_id,
    position = EXCLUDED.position,
    links = EXCLUDED.links,
    updated = EXCLUDED.updated,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleTask :one
SELECT * FROM google_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: GetGoogleTasksByList :many
SELECT * FROM google_tasks
WHERE user_id = $1 AND task_list_id = $2 AND deleted = false
ORDER BY position;

-- name: GetGoogleTasksByUser :many
SELECT * FROM google_tasks
WHERE user_id = $1 AND deleted = false
ORDER BY due NULLS LAST, position
LIMIT $2 OFFSET $3;

-- name: GetGoogleTasksDue :many
SELECT * FROM google_tasks
WHERE user_id = $1
  AND status = 'needsAction'
  AND deleted = false
  AND due IS NOT NULL
  AND due <= $2
ORDER BY due;

-- name: GetGoogleTasksPending :many
SELECT * FROM google_tasks
WHERE user_id = $1
  AND status = 'needsAction'
  AND deleted = false
ORDER BY due NULLS LAST, position
LIMIT $2;

-- name: GetGoogleTasksCompleted :many
SELECT * FROM google_tasks
WHERE user_id = $1
  AND status = 'completed'
  AND deleted = false
ORDER BY completed DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGoogleTasks :many
SELECT * FROM google_tasks
WHERE user_id = $1
  AND deleted = false
  AND (title ILIKE $2 OR notes ILIKE $2)
ORDER BY due NULLS LAST
LIMIT $3;

-- name: CountGoogleTasksPending :one
SELECT COUNT(*) FROM google_tasks
WHERE user_id = $1
  AND status = 'needsAction'
  AND deleted = false;

-- name: DeleteGoogleTask :exec
DELETE FROM google_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: DeleteGoogleTasksByList :exec
DELETE FROM google_tasks
WHERE user_id = $1 AND task_list_id = $2;

-- name: DeleteGoogleTasksByUser :exec
DELETE FROM google_tasks WHERE user_id = $1;
