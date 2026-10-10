-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = $1
  AND (sqlc.narg(status)::taskstatus IS NULL OR status = sqlc.narg(status))
  AND (sqlc.narg(priority)::taskpriority IS NULL OR priority = sqlc.narg(priority))
  AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
ORDER BY
  CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
  due_date ASC NULLS LAST,
  created_at DESC;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1 AND user_id = $2;

-- name: CreateTask :one
INSERT INTO tasks (user_id, title, description, status, priority, due_date, project_id, assignee_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET
  title = COALESCE(sqlc.narg(title), title),
  description = COALESCE(sqlc.narg(description), description),
  status = COALESCE(sqlc.narg(status), status),
  priority = COALESCE(sqlc.narg(priority), priority),
  due_date = COALESCE(sqlc.narg(due_date), due_date),
  project_id = COALESCE(sqlc.narg(project_id), project_id),
  assignee_id = COALESCE(sqlc.narg(assignee_id), assignee_id),
  position = COALESCE(sqlc.narg(position), position),
  updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ToggleTaskStatus :one
UPDATE tasks
SET status = CASE WHEN status = 'done' THEN 'todo'::taskstatus ELSE 'done'::taskstatus END,
    completed_at = CASE WHEN status = 'done' THEN NULL ELSE NOW() END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;

-- name: ListFocusItems :many
SELECT * FROM focus_items
WHERE user_id = $1
  AND DATE(focus_date) = sqlc.arg(focus_date)::date
ORDER BY created_at ASC;

-- name: CreateFocusItem :one
INSERT INTO focus_items (user_id, text, focus_date)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateFocusItem :one
UPDATE focus_items
SET text = $2, completed = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteFocusItem :exec
DELETE FROM focus_items
WHERE id = $1 AND user_id = $2;
