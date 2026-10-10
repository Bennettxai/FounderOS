-- name: ListProjects :many
SELECT p.*, c.name as client_company_name
FROM projects p
LEFT JOIN clients c ON p.client_id = c.id
WHERE (p.user_id = $1 OR p.workspace_id IN (SELECT workspace_id FROM workspace_members WHERE user_id = $1 AND status = 'active'))
  AND (sqlc.narg(status)::projectstatus IS NULL OR p.status = sqlc.narg(status))
  AND (sqlc.narg(priority)::projectpriority IS NULL OR p.priority = sqlc.narg(priority))
  AND (sqlc.narg(client_id)::uuid IS NULL OR p.client_id = sqlc.narg(client_id))
  AND (sqlc.narg(workspace_id)::uuid IS NULL OR p.workspace_id = sqlc.narg(workspace_id))
ORDER BY p.updated_at DESC;

-- name: GetProject :one
SELECT p.*, c.name as client_company_name
FROM projects p
LEFT JOIN clients c ON p.client_id = c.id
WHERE p.id = $1 AND p.user_id = $2;

-- name: CreateProject :one
INSERT INTO projects (id, user_id, name, description, status, priority, client_name, client_id, project_type, project_metadata, start_date, due_date, visibility, owner_id)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, description = $3, status = $4, priority = $5, client_name = $6, client_id = $7, project_type = $8, project_metadata = $9, start_date = $10, due_date = $11, visibility = $12, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProjectStatus :one
UPDATE projects
SET status = $2, completed_at = CASE WHEN $2 = 'COMPLETED' THEN NOW() ELSE NULL END, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1 AND user_id = $2;

-- name: GetProjectStats :one
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'ACTIVE') as active,
    COUNT(*) FILTER (WHERE status = 'COMPLETED') as completed,
    COUNT(*) FILTER (WHERE status = 'PAUSED') as paused,
    COUNT(*) FILTER (WHERE status = 'ARCHIVED') as archived
FROM projects
WHERE user_id = $1;

-- name: GetProjectsByClient :many
SELECT * FROM projects
WHERE client_id = $1
ORDER BY updated_at DESC;

-- name: GetOverdueProjects :many
SELECT * FROM projects
WHERE user_id = $1
  AND due_date < CURRENT_DATE
  AND status NOT IN ('COMPLETED', 'ARCHIVED')
ORDER BY due_date ASC;

-- name: GetUpcomingProjects :many
SELECT * FROM projects
WHERE user_id = $1
  AND due_date >= CURRENT_DATE
  AND due_date <= CURRENT_DATE + INTERVAL '7 days'
  AND status NOT IN ('COMPLETED', 'ARCHIVED')
ORDER BY due_date ASC;

-- name: GetProjectNotes :many
SELECT * FROM project_notes
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: AddProjectNote :one
INSERT INTO project_notes (project_id, content)
VALUES ($1, $2)
RETURNING *;
