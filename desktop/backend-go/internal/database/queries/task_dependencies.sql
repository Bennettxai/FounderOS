-- name: ListTaskDependencies :many
-- Lists all dependencies for a task (both as predecessor and successor)
SELECT td.*,
  pt.title AS predecessor_title,
  st.title AS successor_title
FROM task_dependencies td
JOIN tasks pt ON td.predecessor_id = pt.id
JOIN tasks st ON td.successor_id = st.id
WHERE td.predecessor_id = $1 OR td.successor_id = $1
ORDER BY td.created_at ASC;

-- name: GetTaskPredecessors :many
-- Gets all tasks that must complete before this task can start
SELECT td.*, t.title, t.status, t.due_date, t.start_date
FROM task_dependencies td
JOIN tasks t ON td.predecessor_id = t.id
WHERE td.successor_id = $1
ORDER BY t.due_date ASC NULLS LAST;

-- name: GetTaskSuccessors :many
-- Gets all tasks that depend on this task
SELECT td.*, t.title, t.status, t.due_date, t.start_date
FROM task_dependencies td
JOIN tasks t ON td.successor_id = t.id
WHERE td.predecessor_id = $1
ORDER BY t.due_date ASC NULLS LAST;

-- name: AddTaskDependency :one
-- Creates a new dependency between tasks
INSERT INTO task_dependencies (predecessor_id, successor_id, dependency_type, lag_days)
VALUES ($1, $2, $3, $4)
ON CONFLICT (predecessor_id, successor_id) DO UPDATE SET
    dependency_type = EXCLUDED.dependency_type,
    lag_days = EXCLUDED.lag_days
RETURNING *;

-- name: UpdateTaskDependency :one
-- Updates an existing dependency
UPDATE task_dependencies
SET dependency_type = $3, lag_days = $4
WHERE predecessor_id = $1 AND successor_id = $2
RETURNING *;

-- name: RemoveTaskDependency :exec
-- Removes a dependency between tasks
DELETE FROM task_dependencies
WHERE predecessor_id = $1 AND successor_id = $2;

-- name: RemoveAllTaskDependencies :exec
-- Removes all dependencies for a task (as both predecessor and successor)
DELETE FROM task_dependencies
WHERE predecessor_id = $1 OR successor_id = $1;

-- name: GetDependencyChain :many
-- Gets the full dependency chain for visualization (recursive CTE)
WITH RECURSIVE dependency_chain AS (
    -- Base case: start with the given task
    SELECT
        $1::uuid AS task_id,
        t.title,
        t.status,
        t.start_date,
        t.due_date,
        0 AS depth,
        ARRAY[$1::uuid] AS path
    FROM tasks t
    WHERE t.id = $1

    UNION ALL

    -- Recursive case: follow predecessors
    SELECT
        td.predecessor_id AS task_id,
        t.title,
        t.status,
        t.start_date,
        t.due_date,
        dc.depth + 1,
        dc.path || td.predecessor_id
    FROM dependency_chain dc
    JOIN task_dependencies td ON td.successor_id = dc.task_id
    JOIN tasks t ON t.id = td.predecessor_id
    WHERE NOT (td.predecessor_id = ANY(dc.path))  -- Prevent cycles
    AND dc.depth < 20  -- Limit depth
)
SELECT * FROM dependency_chain
ORDER BY depth DESC;

-- name: GetGanttTasks :many
-- Gets tasks with dependency info for Gantt chart
SELECT
    t.id,
    t.title,
    t.status,
    t.priority,
    t.start_date,
    t.due_date,
    t.parent_task_id,
    t.position,
    t.project_id,
    p.name AS project_name,
    (SELECT COUNT(*)::int FROM task_dependencies WHERE successor_id = t.id) AS predecessor_count,
    (SELECT COUNT(*)::int FROM task_dependencies WHERE predecessor_id = t.id) AS successor_count,
    (SELECT ARRAY_AGG(predecessor_id) FROM task_dependencies WHERE successor_id = t.id) AS predecessor_ids
FROM tasks t
LEFT JOIN projects p ON t.project_id = p.id
WHERE t.user_id = $1
  AND t.parent_task_id IS NULL
  AND (sqlc.narg(project_id)::uuid IS NULL OR t.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(status)::taskstatus IS NULL OR t.status = sqlc.narg(status))
ORDER BY
    COALESCE(t.start_date, t.created_at) ASC,
    t.position ASC;

-- name: GetDependenciesForTasks :many
-- Gets all dependencies between a set of tasks (for batch loading)
SELECT td.*,
  pt.title AS predecessor_title,
  st.title AS successor_title
FROM task_dependencies td
JOIN tasks pt ON td.predecessor_id = pt.id
JOIN tasks st ON td.successor_id = st.id
WHERE td.predecessor_id = ANY($1::uuid[]) OR td.successor_id = ANY($1::uuid[]);
