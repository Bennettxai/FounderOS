-- name: ListProjectMembers :many
SELECT pm.*, tm.name as member_name, tm.email as member_email, tm.avatar_url, tm.role as member_role
FROM project_members pm
LEFT JOIN team_members tm ON pm.team_member_id = tm.id
WHERE pm.project_id = $1
ORDER BY pm.assigned_at DESC;

-- name: GetProjectMember :one
SELECT pm.*, tm.name as member_name, tm.email as member_email, tm.avatar_url
FROM project_members pm
LEFT JOIN team_members tm ON pm.team_member_id = tm.id
WHERE pm.id = $1;

-- name: AddProjectMember :one
INSERT INTO project_members (project_id, user_id, team_member_id, role, assigned_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateProjectMemberRole :one
UPDATE project_members
SET role = $2
WHERE id = $1
RETURNING *;

-- name: RemoveProjectMember :exec
DELETE FROM project_members
WHERE id = $1;

-- name: RemoveProjectMemberByTeamMember :exec
DELETE FROM project_members
WHERE project_id = $1 AND team_member_id = $2;

-- name: GetProjectsByMember :many
SELECT p.*
FROM projects p
JOIN project_members pm ON p.id = pm.project_id
WHERE pm.team_member_id = $1
ORDER BY p.updated_at DESC;

-- name: CountProjectMembers :one
SELECT COUNT(*) FROM project_members WHERE project_id = $1;
