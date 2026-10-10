-- name: GetNotionOAuthToken :one
SELECT * FROM notion_oauth_tokens
WHERE user_id = $1;

-- name: CreateNotionOAuthToken :one
INSERT INTO notion_oauth_tokens (
    user_id, workspace_id, workspace_name, workspace_icon,
    access_token, bot_id, owner_type, owner_user_id,
    owner_user_name, owner_user_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: UpdateNotionOAuthToken :one
UPDATE notion_oauth_tokens
SET access_token = $2,
    workspace_name = COALESCE($3, workspace_name),
    workspace_icon = COALESCE($4, workspace_icon),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: DeleteNotionOAuthToken :exec
DELETE FROM notion_oauth_tokens
WHERE user_id = $1;

-- name: GetNotionOAuthStatus :one
SELECT id, user_id, workspace_id, workspace_name, workspace_icon, owner_user_name, created_at
FROM notion_oauth_tokens
WHERE user_id = $1;

-- name: GetNotionWorkspaceByUser :one
SELECT workspace_id, workspace_name, access_token
FROM notion_oauth_tokens
WHERE user_id = $1;
