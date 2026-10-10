-- name: GetGoogleOAuthToken :one
SELECT * FROM google_oauth_tokens
WHERE user_id = $1;

-- name: CreateGoogleOAuthToken :one
INSERT INTO google_oauth_tokens (
    user_id, access_token, refresh_token, token_type, expiry, scopes, google_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateGoogleOAuthToken :one
UPDATE google_oauth_tokens
SET access_token = $2,
    refresh_token = COALESCE($3, refresh_token),
    expiry = $4,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: DeleteGoogleOAuthToken :exec
DELETE FROM google_oauth_tokens
WHERE user_id = $1;

-- name: GetGoogleOAuthStatus :one
SELECT id, user_id, google_email, expiry, created_at
FROM google_oauth_tokens
WHERE user_id = $1;
