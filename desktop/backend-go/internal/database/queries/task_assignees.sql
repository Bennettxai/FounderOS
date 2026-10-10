-- name: ListTaskAssignees :many
-- Lists all assignees for a task with their member info
SELECT ta.*, tm.name AS member_name, tm.email AS member_email, tm.avatar_url AS member_avatar_url, tm.role AS member_role
FROM task_assignees ta
JOIN team_members tm ON ta.team_member_id = tm.id
WHERE ta.task_id = $1
ORDER BY ta.assigned_at ASC;

-- name: GetTaskAssignee :one
SELECT * FROM task_assignees
WHERE task_id = $1 AND team_member_id = $2;

-- name: AddTaskAssignee :one
INSERT INTO task_assignees (task_id, team_member_id, role, assigned_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (task_id, team_member_id) DO UPDATE SET
    role = EXCLUDED.role,
    assigned_at = NOW()
RETURNING *;

-- name: UpdateTaskAssigneeRole :one
UPDATE task_assignees
SET role = $3
WHERE task_id = $1 AND team_member_id = $2
RETURNING *;

-- name: RemoveTaskAssignee :exec
DELETE FROM task_assignees
WHERE task_id = $1 AND team_member_id = $2;

-- name: RemoveAllTaskAssignees :exec
DELETE FROM task_assignees
WHERE task_id = $1;

-- name: SetTaskAssignees :exec
-- Replaces all assignees for a task with a new set
-- First delete existing, then insert new ones via separate calls

-- name: CountTaskAssignees :one
SELECT COUNT(*)::int AS count FROM task_assignees
WHERE task_id = $1;

-- name: ListTasksByAssignee :many
-- Lists all tasks assigned to a team member
SELECT t.*, ta.role AS assignee_role, ta.assigned_at
FROM tasks t
JOIN task_assignees ta ON t.id = ta.task_id
WHERE ta.team_member_id = $1
  AND t.user_id = $2
  AND (sqlc.narg(status)::taskstatus IS NULL OR t.status = sqlc.narg(status))
ORDER BY t.due_date ASC NULLS LAST, t.created_at DESC;

-- name: GetTasksWithAssignees :many
-- Returns tasks with their assignee count (for list views)
SELECT t.*,
  (SELECT COUNT(*)::int FROM task_assignees ta WHERE ta.task_id = t.id) AS assignee_count,
  (SELECT COUNT(*)::int FROM tasks st WHERE st.parent_task_id = t.id) AS subtask_count,
  (SELECT COUNT(*)::int FROM tasks st WHERE st.parent_task_id = t.id AND st.status = 'done') AS completed_subtask_count
FROM tasks t
WHERE t.user_id = $1
  AND t.parent_task_id IS NULL
  AND (sqlc.narg(status)::taskstatus IS NULL OR t.status = sqlc.narg(status))
  AND (sqlc.narg(priority)::taskpriority IS NULL OR t.priority = sqlc.narg(priority))
  AND (sqlc.narg(project_id)::uuid IS NULL OR t.project_id = sqlc.narg(project_id))
ORDER BY
  CASE t.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
  t.position ASC,
  t.due_date ASC NULLS LAST,
  t.created_at DESC;

-- name: GetAssigneeCountsForTasks :many
-- Returns assignee counts for multiple tasks
SELECT task_id, COUNT(*)::int AS assignee_count
FROM task_assignees
WHERE task_id = ANY($1::uuid[])
GROUP BY task_id;

-- name: GetAssigneesForTasks :many
-- Returns all assignees for multiple tasks (for batch loading)
SELECT ta.task_id, ta.team_member_id, ta.role, ta.assigned_at,
  tm.name AS member_name, tm.email AS member_email, tm.avatar_url AS member_avatar_url
FROM task_assignees ta
JOIN team_members tm ON ta.team_member_id = tm.id
WHERE ta.task_id = ANY($1::uuid[])
ORDER BY ta.task_id, ta.assigned_at ASC;

-- name: GetWorkloadByMember :many
-- Returns task counts per team member for workload view
SELECT tm.id, tm.name, tm.avatar_url, tm.capacity,
  COUNT(CASE WHEN t.status != 'done' THEN 1 END)::int AS active_task_count,
  COUNT(CASE WHEN t.status = 'done' THEN 1 END)::int AS completed_task_count,
  COUNT(*)::int AS total_task_count
FROM team_members tm
LEFT JOIN task_assignees ta ON tm.id = ta.team_member_id
LEFT JOIN tasks t ON ta.task_id = t.id
WHERE tm.user_id = $1
GROUP BY tm.id, tm.name, tm.avatar_url, tm.capacity
ORDER BY active_task_count DESC;
