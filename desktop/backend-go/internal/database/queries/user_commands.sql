-- name: ListUserCommands :many
SELECT * FROM user_commands
WHERE user_id = $1 AND is_active = TRUE
ORDER BY display_name ASC;

-- name: GetUserCommand :one
SELECT * FROM user_commands
WHERE id = $1 AND user_id = $2;

-- name: GetUserCommandByName :one
SELECT * FROM user_commands
WHERE name = $1 AND user_id = $2 AND is_active = TRUE;

-- name: CreateUserCommand :one
INSERT INTO user_commands (
    user_id, name, display_name, description, icon, system_prompt, context_sources, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateUserCommand :one
UPDATE user_commands
SET
    name = COALESCE($3, name),
    display_name = COALESCE($4, display_name),
    description = COALESCE($5, description),
    icon = COALESCE($6, icon),
    system_prompt = COALESCE($7, system_prompt),
    context_sources = COALESCE($8, context_sources),
    is_active = COALESCE($9, is_active),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteUserCommand :exec
DELETE FROM user_commands
WHERE id = $1 AND user_id = $2;

-- name: GetAllUserCommands :many
SELECT * FROM user_commands
WHERE user_id = $1
ORDER BY created_at DESC;
