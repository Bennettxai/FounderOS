-- ================================================================
-- App Templates Queries
-- Description: Queries for personalized app generation system
-- ================================================================

-- ================================================================
-- APP TEMPLATES
-- ================================================================

-- name: GetAppTemplate :one
SELECT * FROM app_templates
WHERE id = $1 LIMIT 1;

-- name: GetAppTemplateByName :one
SELECT * FROM app_templates
WHERE template_name = $1 LIMIT 1;

-- name: ListAppTemplates :many
SELECT * FROM app_templates
ORDER BY priority_score DESC, created_at DESC;

-- name: ListAppTemplatesByCategory :many
SELECT * FROM app_templates
WHERE category = $1
ORDER BY priority_score DESC;

-- name: GetTemplatesByBusinessType :many
SELECT * FROM app_templates
WHERE $1 = ANY(target_business_types)
ORDER BY priority_score DESC;

-- name: GetTemplatesByChallenge :many
SELECT * FROM app_templates
WHERE $1 = ANY(target_challenges)
ORDER BY priority_score DESC;

-- name: GetTemplatesByTeamSize :many
SELECT * FROM app_templates
WHERE $1 = ANY(target_team_sizes)
ORDER BY priority_score DESC;

-- name: MatchTemplatesByProfile :many
-- Match templates based on business type, challenge, and team size
SELECT
    t.*,
    -- Calculate matching score
    CASE
        WHEN $1::text = ANY(t.target_business_types) THEN 40
        ELSE 0
    END +
    CASE
        WHEN $2::text = ANY(t.target_challenges) THEN 30
        ELSE 0
    END +
    CASE
        WHEN $3::text = ANY(t.target_team_sizes) THEN 20
        ELSE 0
    END +
    t.priority_score as match_score
FROM app_templates t
WHERE
    $1::text = ANY(t.target_business_types)
    OR $2::text = ANY(t.target_challenges)
    OR $3::text = ANY(t.target_team_sizes)
ORDER BY match_score DESC
LIMIT $4;

-- name: CreateAppTemplate :one
INSERT INTO app_templates (
    template_name,
    category,
    display_name,
    description,
    icon_type,
    target_business_types,
    target_challenges,
    target_team_sizes,
    priority_score,
    template_config,
    required_modules,
    optional_features,
    generation_prompt,
    scaffold_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: UpdateAppTemplate :one
UPDATE app_templates
SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    icon_type = COALESCE(sqlc.narg('icon_type'), icon_type),
    target_business_types = COALESCE(sqlc.narg('target_business_types'), target_business_types),
    target_challenges = COALESCE(sqlc.narg('target_challenges'), target_challenges),
    target_team_sizes = COALESCE(sqlc.narg('target_team_sizes'), target_team_sizes),
    priority_score = COALESCE(sqlc.narg('priority_score'), priority_score),
    template_config = COALESCE(sqlc.narg('template_config'), template_config),
    generation_prompt = COALESCE(sqlc.narg('generation_prompt'), generation_prompt),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteAppTemplate :exec
DELETE FROM app_templates
WHERE id = $1;

-- ================================================================
-- USER GENERATED APPS
-- ================================================================

-- name: GetUserGeneratedApp :one
SELECT * FROM user_generated_apps
WHERE id = $1 LIMIT 1;

-- name: ListUserGeneratedApps :many
SELECT * FROM user_generated_apps
WHERE workspace_id = $1
ORDER BY
    is_pinned DESC,
    position_index ASC,
    generated_at DESC;

-- name: ListVisibleUserApps :many
SELECT * FROM user_generated_apps
WHERE workspace_id = $1 AND is_visible = true
ORDER BY
    is_pinned DESC,
    position_index ASC,
    generated_at DESC;

-- name: GetUserAppsByTemplate :many
SELECT * FROM user_generated_apps
WHERE workspace_id = $1 AND template_id = $2
ORDER BY generated_at DESC;

-- name: CountUserAppsByWorkspace :one
SELECT COUNT(*) FROM user_generated_apps
WHERE workspace_id = $1;

-- name: CreateUserGeneratedApp :one
INSERT INTO user_generated_apps (
    workspace_id,
    template_id,
    app_name,
    module_instance_id,
    is_visible,
    is_pinned,
    is_favorite,
    position_index,
    custom_config,
    custom_icon
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: UpdateUserGeneratedApp :one
UPDATE user_generated_apps
SET
    app_name = COALESCE(sqlc.narg('app_name'), app_name),
    is_visible = COALESCE(sqlc.narg('is_visible'), is_visible),
    is_pinned = COALESCE(sqlc.narg('is_pinned'), is_pinned),
    is_favorite = COALESCE(sqlc.narg('is_favorite'), is_favorite),
    position_index = COALESCE(sqlc.narg('position_index'), position_index),
    custom_config = COALESCE(sqlc.narg('custom_config'), custom_config),
    custom_icon = COALESCE(sqlc.narg('custom_icon'), custom_icon),
    last_accessed_at = COALESCE(sqlc.narg('last_accessed_at'), last_accessed_at),
    access_count = COALESCE(sqlc.narg('access_count'), access_count),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: IncrementAppAccessCount :one
UPDATE user_generated_apps
SET
    access_count = access_count + 1,
    last_accessed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUserGeneratedApp :exec
DELETE FROM user_generated_apps
WHERE id = $1;

-- name: LinkUserAppToOSAApp :one
UPDATE user_generated_apps
SET
    module_instance_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- ================================================================
-- APP GENERATION QUEUE
-- ================================================================

-- name: GetQueueItem :one
SELECT id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at
FROM app_generation_queue
WHERE id = $1 LIMIT 1;

-- name: ListPendingQueueItems :many
SELECT id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at
FROM app_generation_queue
WHERE status = 'pending'
ORDER BY priority DESC, created_at ASC
LIMIT $1;

-- name: ListQueueItemsByWorkspace :many
SELECT id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at
FROM app_generation_queue
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: GetNextPendingItem :one
SELECT
    id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at
FROM app_generation_queue
WHERE status = 'pending'
ORDER BY priority DESC, created_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: CreateQueueItem :one
INSERT INTO app_generation_queue (
    workspace_id,
    template_id,
    status,
    priority,
    generation_context
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at;

-- name: UpdateQueueItemStatus :one
UPDATE app_generation_queue
SET
    status = sqlc.arg(status)::varchar,
    started_at = CASE WHEN sqlc.arg(status)::varchar = 'processing' THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN sqlc.arg(status)::varchar IN ('completed', 'failed') THEN NOW() ELSE completed_at END
WHERE id = sqlc.arg(id)
RETURNING id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at;

-- name: UpdateQueueItemError :one
UPDATE app_generation_queue
SET
    error_message = $2,
    retry_count = retry_count + 1,
    status = CASE
        WHEN retry_count + 1 >= max_retries THEN 'failed'::varchar
        ELSE 'pending'::varchar
    END
WHERE id = $1
RETURNING id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at;

-- name: MarkQueueItemCompleted :one
UPDATE app_generation_queue
SET
    status = 'completed',
    completed_at = NOW()
WHERE id = $1
RETURNING id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at;

-- name: MarkQueueItemFailed :one
UPDATE app_generation_queue
SET
    status = 'failed',
    error_message = $2,
    completed_at = NOW()
WHERE id = $1
RETURNING id, workspace_id, template_id, status, priority,
    generation_context, error_message, retry_count, max_retries,
    created_at, started_at, completed_at;

-- name: DeleteCompletedQueueItems :exec
-- Clean up completed items older than 7 days
DELETE FROM app_generation_queue
WHERE status = 'completed'
AND completed_at < NOW() - INTERVAL '7 days';

-- name: CountQueueItemsByStatus :one
SELECT COUNT(*) FROM app_generation_queue
WHERE workspace_id = $1 AND status = $2;

-- name: GetQueueStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'processing') as processing_count,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) as total_count
FROM app_generation_queue
WHERE workspace_id = $1;
