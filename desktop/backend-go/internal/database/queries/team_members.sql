-- name: ListTeamMembers :many
SELECT tm.*,
       (SELECT COUNT(*) FROM tasks t WHERE t.assignee_id = tm.id AND t.status != 'done') as active_task_count,
       (SELECT COUNT(*) FROM project_members pm JOIN projects p ON pm.project_id = p.id WHERE pm.team_member_id = tm.id AND p.status = 'ACTIVE') as active_project_count
FROM team_members tm
WHERE tm.user_id = $1
ORDER BY tm.name ASC;

-- name: GetTeamMember :one
SELECT * FROM team_members
WHERE id = $1 AND user_id = $2;

-- name: CreateTeamMember :one
INSERT INTO team_members (user_id, name, email, role, avatar_url, status, capacity, manager_id, skills, hourly_rate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateTeamMember :one
UPDATE team_members
SET name = $2, email = $3, role = $4, avatar_url = $5, status = $6, capacity = $7, manager_id = $8, skills = $9, hourly_rate = $10, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateTeamMemberStatus :one
UPDATE team_members
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateTeamMemberCapacity :one
UPDATE team_members
SET capacity = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTeamMember :exec
DELETE FROM team_members
WHERE id = $1 AND user_id = $2;

-- name: CreateTeamMemberActivity :one
INSERT INTO team_member_activities (member_id, activity_type, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTeamMemberActivities :many
SELECT * FROM team_member_activities
WHERE member_id = $1
ORDER BY created_at DESC
LIMIT $2;
