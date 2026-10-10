-- name: ListMCPServers :many
SELECT * FROM mcp_servers
WHERE user_id = $1
ORDER BY name ASC;

-- name: ListEnabledMCPServers :many
SELECT * FROM mcp_servers
WHERE user_id = $1 AND enabled = TRUE
ORDER BY name ASC;

-- name: GetMCPServer :one
SELECT * FROM mcp_servers
WHERE id = $1 AND user_id = $2;

-- name: CreateMCPServer :one
INSERT INTO mcp_servers (
    user_id, name, description, server_url, transport,
    auth_type, auth_token_enc, custom_headers, enabled
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: UpdateMCPServer :one
UPDATE mcp_servers
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    server_url = COALESCE(sqlc.narg('server_url'), server_url),
    transport = COALESCE(sqlc.narg('transport'), transport),
    auth_type = COALESCE(sqlc.narg('auth_type'), auth_type),
    auth_token_enc = COALESCE(sqlc.narg('auth_token_enc'), auth_token_enc),
    custom_headers = COALESCE(sqlc.narg('custom_headers'), custom_headers),
    enabled = COALESCE(sqlc.narg('enabled'), enabled),
    updated_at = NOW()
WHERE id = $1 AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: UpdateMCPServerStatus :exec
UPDATE mcp_servers
SET
    status = $3,
    last_error = $4,
    last_connected = CASE WHEN $3 = 'connected' THEN NOW() ELSE last_connected END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateMCPServerToolsCache :exec
UPDATE mcp_servers
SET
    tools_cache = $3,
    status = 'connected',
    last_error = NULL,
    last_connected = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteMCPServer :exec
DELETE FROM mcp_servers
WHERE id = $1 AND user_id = $2;

-- name: CountUserMCPServers :one
SELECT COUNT(*) FROM mcp_servers
WHERE user_id = $1;
