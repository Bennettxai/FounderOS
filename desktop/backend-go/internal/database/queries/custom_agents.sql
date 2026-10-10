-- name: ListCustomAgents :many
SELECT * FROM custom_agents
WHERE user_id = $1 AND is_active = TRUE
ORDER BY times_used DESC, display_name ASC;

-- name: GetAllCustomAgents :many
SELECT * FROM custom_agents
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetCustomAgent :one
SELECT * FROM custom_agents
WHERE id = $1 AND user_id = $2;

-- name: GetCustomAgentByName :one
SELECT * FROM custom_agents
WHERE LOWER(name) = LOWER($1) AND user_id = $2 AND is_active = TRUE;

-- name: CreateCustomAgent :one
INSERT INTO custom_agents (
    user_id, name, display_name, description, avatar,
    system_prompt, model_preference, temperature, max_tokens,
    capabilities, tools_enabled, context_sources,
    thinking_enabled, streaming_enabled, apply_personalization,
    welcome_message, suggested_prompts,
    category, is_active, is_public, is_featured
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
) RETURNING *;

-- name: UpdateCustomAgent :one
UPDATE custom_agents
SET
    name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    avatar = COALESCE(sqlc.narg('avatar'), avatar),
    system_prompt = COALESCE(sqlc.narg('system_prompt'), system_prompt),
    model_preference = COALESCE(sqlc.narg('model_preference'), model_preference),
    temperature = COALESCE(sqlc.narg('temperature'), temperature),
    max_tokens = COALESCE(sqlc.narg('max_tokens'), max_tokens),
    capabilities = COALESCE(sqlc.narg('capabilities'), capabilities),
    tools_enabled = COALESCE(sqlc.narg('tools_enabled'), tools_enabled),
    context_sources = COALESCE(sqlc.narg('context_sources'), context_sources),
    thinking_enabled = COALESCE(sqlc.narg('thinking_enabled'), thinking_enabled),
    streaming_enabled = COALESCE(sqlc.narg('streaming_enabled'), streaming_enabled),
    apply_personalization = COALESCE(sqlc.narg('apply_personalization'), apply_personalization),
    welcome_message = COALESCE(sqlc.narg('welcome_message'), welcome_message),
    suggested_prompts = COALESCE(sqlc.narg('suggested_prompts'), suggested_prompts),
    category = COALESCE(sqlc.narg('category'), category),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    is_public = COALESCE(sqlc.narg('is_public'), is_public),
    is_featured = COALESCE(sqlc.narg('is_featured'), is_featured),
    updated_at = NOW()
WHERE id = $1 AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteCustomAgent :exec
DELETE FROM custom_agents
WHERE id = $1 AND user_id = $2;

-- name: IncrementAgentUsage :exec
UPDATE custom_agents
SET times_used = times_used + 1, last_used_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListCustomAgentsByCategory :many
SELECT * FROM custom_agents
WHERE user_id = $1 AND category = $2 AND is_active = TRUE
ORDER BY times_used DESC, display_name ASC;

-- ===== AGENT PRESETS =====

-- name: ListAgentPresets :many
SELECT * FROM agent_presets
ORDER BY times_copied DESC, display_name ASC;

-- name: GetAgentPreset :one
SELECT * FROM agent_presets
WHERE id = $1;

-- name: GetAgentPresetByName :one
SELECT * FROM agent_presets
WHERE name = $1;

-- name: IncrementPresetCopyCount :exec
UPDATE agent_presets
SET times_copied = times_copied + 1, updated_at = NOW()
WHERE id = $1;

-- name: CreateAgentFromPreset :one
INSERT INTO custom_agents (
    user_id, name, display_name, description, avatar,
    system_prompt, model_preference, temperature, max_tokens,
    capabilities, tools_enabled, context_sources,
    thinking_enabled, streaming_enabled, apply_personalization,
    welcome_message, suggested_prompts,
    category, is_active, is_public, is_featured
)
SELECT
    $1, -- user_id
    $2, -- custom name (user can rename)
    display_name,
    description,
    avatar,
    system_prompt,
    model_preference,
    temperature,
    max_tokens,
    capabilities,
    tools_enabled,
    context_sources,
    thinking_enabled,
    TRUE, -- streaming_enabled
    FALSE, -- apply_personalization (default off)
    welcome_message,
    suggested_prompts,
    category,
    TRUE, -- is_active
    FALSE, -- is_public (default private)
    is_featured -- inherit from preset
FROM agent_presets ap
WHERE ap.id = $3
RETURNING *;

-- name: CountUserAgents :one
SELECT COUNT(*) FROM custom_agents WHERE user_id = $1;
