-- name: ListProjectStatuses :many
-- Lists all statuses for a project ordered by position
SELECT * FROM project_statuses
WHERE project_id = $1
ORDER BY position ASC;

-- name: GetProjectStatus :one
SELECT * FROM project_statuses
WHERE id = $1;

-- name: GetProjectStatusByName :one
SELECT * FROM project_statuses
WHERE project_id = $1 AND name = $2;

-- name: GetDefaultProjectStatus :one
SELECT * FROM project_statuses
WHERE project_id = $1 AND is_default = TRUE
LIMIT 1;

-- name: GetDoneProjectStatus :one
SELECT * FROM project_statuses
WHERE project_id = $1 AND is_done_state = TRUE
LIMIT 1;

-- name: CreateProjectStatus :one
INSERT INTO project_statuses (project_id, name, color, position, is_done_state, is_default)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: CreateProjectStatusAutoPosition :one
-- Creates a status with automatic position assignment
INSERT INTO project_statuses (project_id, name, color, is_done_state, is_default, position)
SELECT $1, $2, $3, $4, $5,
  COALESCE((SELECT MAX(position) + 1 FROM project_statuses WHERE project_id = $1), 0)
RETURNING *;

-- name: UpdateProjectCustomStatus :one
UPDATE project_statuses
SET name = $2, color = $3, is_done_state = $4, is_default = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProjectStatusPosition :exec
UPDATE project_statuses
SET position = $2, updated_at = NOW()
WHERE id = $1;

-- name: SetDefaultProjectStatus :exec
-- Sets a status as the default and unsets any other default for the project
UPDATE project_statuses ps
SET is_default = (ps.id = $1), updated_at = NOW()
WHERE ps.project_id = (SELECT sub.project_id FROM project_statuses sub WHERE sub.id = $1);

-- name: ReorderProjectStatuses :exec
-- Updates positions for all statuses after reorder
UPDATE project_statuses AS ps
SET position = new_positions.new_position, updated_at = NOW()
FROM (
  SELECT s.id AS status_id, ROW_NUMBER() OVER (ORDER BY s.position) - 1 AS new_position
  FROM project_statuses s
  WHERE s.project_id = $1
) AS new_positions
WHERE ps.id = new_positions.status_id;

-- name: DeleteProjectStatus :exec
DELETE FROM project_statuses
WHERE id = $1;

-- name: DeleteProjectStatusWithFallback :exec
-- Deletes a status and moves tasks to the default status
-- First, update any tasks using this status to use the default status
WITH default_status AS (
  SELECT ps.id FROM project_statuses ps
  WHERE ps.project_id = (SELECT sub.project_id FROM project_statuses sub WHERE sub.id = $1)
    AND ps.is_default = TRUE
  LIMIT 1
)
UPDATE tasks
SET custom_status_id = (SELECT ds.id FROM default_status ds), updated_at = NOW()
WHERE custom_status_id = $1;

-- name: CountTasksByProjectStatus :many
-- Returns task counts for each status in a project
SELECT ps.id, ps.name, ps.color, ps.position, ps.is_done_state, ps.is_default,
  COUNT(t.id)::int AS task_count
FROM project_statuses ps
LEFT JOIN tasks t ON t.custom_status_id = ps.id
WHERE ps.project_id = $1
GROUP BY ps.id, ps.name, ps.color, ps.position, ps.is_done_state, ps.is_default
ORDER BY ps.position ASC;

-- name: GetProjectStatusesWithTaskCounts :many
-- Returns all statuses for a project with task counts
SELECT ps.*,
  COALESCE((SELECT COUNT(*)::int FROM tasks t WHERE t.custom_status_id = ps.id), 0) AS task_count
FROM project_statuses ps
WHERE ps.project_id = $1
ORDER BY ps.position ASC;

-- name: CreateDefaultStatusesForProject :exec
-- Creates the default set of statuses for a project
INSERT INTO project_statuses (project_id, name, color, position, is_done_state, is_default)
VALUES
    ($1, 'To Do', '#6B7280', 0, FALSE, TRUE),
    ($1, 'In Progress', '#3B82F6', 1, FALSE, FALSE),
    ($1, 'In Review', '#8B5CF6', 2, FALSE, FALSE),
    ($1, 'Done', '#10B981', 3, TRUE, FALSE),
    ($1, 'Blocked', '#F59E0B', 4, FALSE, FALSE)
ON CONFLICT (project_id, name) DO NOTHING;
