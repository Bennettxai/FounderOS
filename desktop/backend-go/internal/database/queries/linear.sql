-- Linear Integration queries for SQLC

-- ============================================================================
-- Linear Issues
-- ============================================================================

-- name: GetLinearIssues :many
SELECT * FROM linear_issues
WHERE user_id = $1
ORDER BY external_updated_at DESC
LIMIT $2;

-- name: GetLinearIssueByExternalID :one
SELECT * FROM linear_issues
WHERE user_id = $1 AND external_id = $2;

-- name: GetLinearIssuesByState :many
SELECT * FROM linear_issues
WHERE user_id = $1 AND state = $2
ORDER BY external_updated_at DESC
LIMIT $3;

-- name: GetLinearIssuesByTeam :many
SELECT * FROM linear_issues
WHERE user_id = $1 AND team = $2
ORDER BY external_updated_at DESC
LIMIT $3;

-- name: UpsertLinearIssue :one
INSERT INTO linear_issues (
    user_id, external_id, identifier, title, description,
    state, priority, assignee, project, team,
    due_date, external_created_at, external_updated_at, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
ON CONFLICT (user_id, external_id)
DO UPDATE SET
    identifier = EXCLUDED.identifier,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    state = EXCLUDED.state,
    priority = EXCLUDED.priority,
    assignee = EXCLUDED.assignee,
    project = EXCLUDED.project,
    team = EXCLUDED.team,
    due_date = EXCLUDED.due_date,
    external_created_at = EXCLUDED.external_created_at,
    external_updated_at = EXCLUDED.external_updated_at,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: DeleteLinearIssue :exec
DELETE FROM linear_issues
WHERE user_id = $1 AND external_id = $2;

-- name: DeleteLinearIssuesForUser :exec
DELETE FROM linear_issues
WHERE user_id = $1;

-- name: CountLinearIssuesByUser :one
SELECT COUNT(*) FROM linear_issues
WHERE user_id = $1;

-- ============================================================================
-- Linear Projects
-- ============================================================================

-- name: GetLinearProjects :many
SELECT * FROM linear_projects
WHERE user_id = $1
ORDER BY name ASC;

-- name: GetLinearProjectByExternalID :one
SELECT * FROM linear_projects
WHERE user_id = $1 AND external_id = $2;

-- name: GetLinearProjectsByState :many
SELECT * FROM linear_projects
WHERE user_id = $1 AND state = $2
ORDER BY name ASC;

-- name: UpsertLinearProject :one
INSERT INTO linear_projects (
    user_id, external_id, name, description, state,
    progress, start_date, target_date, team, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (user_id, external_id)
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    state = EXCLUDED.state,
    progress = EXCLUDED.progress,
    start_date = EXCLUDED.start_date,
    target_date = EXCLUDED.target_date,
    team = EXCLUDED.team,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: DeleteLinearProject :exec
DELETE FROM linear_projects
WHERE user_id = $1 AND external_id = $2;

-- name: DeleteLinearProjectsForUser :exec
DELETE FROM linear_projects
WHERE user_id = $1;

-- ============================================================================
-- Linear Teams
-- ============================================================================

-- name: GetLinearTeams :many
SELECT * FROM linear_teams
WHERE user_id = $1
ORDER BY name ASC;

-- name: GetLinearTeamByExternalID :one
SELECT * FROM linear_teams
WHERE user_id = $1 AND external_id = $2;

-- name: GetLinearTeamByKey :one
SELECT * FROM linear_teams
WHERE user_id = $1 AND key = $2;

-- name: UpsertLinearTeam :one
INSERT INTO linear_teams (
    user_id, external_id, key, name, description, issue_count, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id, external_id)
DO UPDATE SET
    key = EXCLUDED.key,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    issue_count = EXCLUDED.issue_count,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: DeleteLinearTeam :exec
DELETE FROM linear_teams
WHERE user_id = $1 AND external_id = $2;

-- name: DeleteLinearTeamsForUser :exec
DELETE FROM linear_teams
WHERE user_id = $1;
