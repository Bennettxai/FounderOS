-- name: CreateOSAPromptTemplate :one
INSERT INTO osa_prompt_templates (
    name,
    display_name,
    description,
    scope,
    workspace_id,
    user_id,
    template_content,
    variables,
    category,
    tags,
    version,
    is_active,
    parent_template_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetOSAPromptTemplate :one
SELECT * FROM osa_prompt_templates
WHERE id = $1 AND is_active = true;

-- name: GetOSAPromptTemplateByName :one
SELECT * FROM osa_prompt_templates
WHERE name = $1
  AND scope = $2
  AND (workspace_id = $3 OR workspace_id IS NULL)
  AND (user_id = $4 OR user_id IS NULL)
  AND is_active = true
ORDER BY
  CASE scope
    WHEN 'user' THEN 1
    WHEN 'workspace' THEN 2
    WHEN 'system' THEN 3
  END
LIMIT 1;

-- name: ListOSAPromptTemplates :many
SELECT * FROM osa_prompt_templates
WHERE is_active = true
  AND ($1::varchar IS NULL OR scope = $1)
  AND ($2::varchar IS NULL OR category = $2)
  AND ($3::uuid IS NULL OR workspace_id = $3)
  AND ($4::uuid IS NULL OR user_id = $4)
ORDER BY usage_count DESC, success_rate DESC
LIMIT $5 OFFSET $6;

-- name: UpdateOSAPromptTemplate :one
UPDATE osa_prompt_templates
SET
    display_name = COALESCE($2, display_name),
    description = COALESCE($3, description),
    template_content = COALESCE($4, template_content),
    variables = COALESCE($5, variables),
    category = COALESCE($6, category),
    tags = COALESCE($7, tags),
    version = COALESCE($8, version),
    is_active = COALESCE($9, is_active),
    metadata = COALESCE($10, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOSAPromptTemplate :exec
DELETE FROM osa_prompt_templates
WHERE id = $1;

-- name: IncrementOSAPromptTemplateUsage :exec
UPDATE osa_prompt_templates
SET
    usage_count = usage_count + 1,
    last_used_at = NOW()
WHERE id = $1;

-- name: RecordOSAPromptTemplateSuccess :exec
UPDATE osa_prompt_templates
SET
    success_count = success_count + 1,
    usage_count = usage_count + 1,
    last_used_at = NOW()
WHERE id = $1;

-- name: RecordOSAPromptTemplateFailure :exec
UPDATE osa_prompt_templates
SET
    failure_count = failure_count + 1,
    usage_count = usage_count + 1,
    last_used_at = NOW()
WHERE id = $1;

-- name: CreateTemplateUsageLog :one
INSERT INTO osa_template_usage_log (
    template_id,
    user_id,
    workspace_id,
    workflow_id,
    module_instance_id,
    variables_used,
    render_time_ms,
    generation_time_sec,
    tokens_used,
    status,
    error_message,
    error_details
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetPopularTemplates :many
SELECT
    t.id,
    t.name,
    t.display_name,
    t.category,
    t.usage_count,
    t.success_rate,
    (
        SELECT ROUND(AVG(user_rating), 2)
        FROM osa_template_usage_log
        WHERE template_id = t.id AND user_rating IS NOT NULL
    ) AS avg_user_rating
FROM osa_prompt_templates t
WHERE t.is_active = true
  AND ($1::varchar IS NULL OR t.category = $1)
  AND ($2::varchar IS NULL OR t.scope = $2)
ORDER BY t.usage_count DESC, t.success_rate DESC
LIMIT $3;
