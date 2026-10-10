-- name: GetSlackOAuthToken :one
SELECT * FROM slack_oauth_tokens
WHERE user_id = $1;

-- name: CreateSlackOAuthToken :one
INSERT INTO slack_oauth_tokens (
    user_id, workspace_id, workspace_name, bot_token, user_token,
    bot_user_id, authed_user_id, bot_scopes, user_scopes,
    incoming_webhook_url, incoming_webhook_channel
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateSlackOAuthToken :one
UPDATE slack_oauth_tokens
SET bot_token = $2,
    user_token = COALESCE($3, user_token),
    bot_scopes = COALESCE($4, bot_scopes),
    user_scopes = COALESCE($5, user_scopes),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: DeleteSlackOAuthToken :exec
DELETE FROM slack_oauth_tokens
WHERE user_id = $1;

-- name: GetSlackOAuthStatus :one
SELECT id, user_id, workspace_id, workspace_name, bot_user_id, created_at
FROM slack_oauth_tokens
WHERE user_id = $1;

-- name: GetSlackWorkspaceByUser :one
SELECT workspace_id, workspace_name, bot_token
FROM slack_oauth_tokens
WHERE user_id = $1;
