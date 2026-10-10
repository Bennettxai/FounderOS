-- name: CreateReasoningTemplate :one
INSERT INTO reasoning_templates (
    user_id, name, description, system_prompt, thinking_instruction,
    output_format, show_thinking, save_thinking, max_thinking_tokens, is_default
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetReasoningTemplate :one
SELECT * FROM reasoning_templates
WHERE id = $1 AND user_id = $2;

-- name: GetDefaultReasoningTemplate :one
SELECT * FROM reasoning_templates
WHERE user_id = $1 AND is_default = true
LIMIT 1;

-- name: ListReasoningTemplates :many
SELECT * FROM reasoning_templates
WHERE user_id = $1
ORDER BY is_default DESC, times_used DESC, created_at DESC;

-- name: UpdateReasoningTemplate :one
UPDATE reasoning_templates
SET name = $2, description = $3, system_prompt = $4, thinking_instruction = $5,
    output_format = $6, show_thinking = $7, save_thinking = $8,
    max_thinking_tokens = $9, updated_at = NOW()
WHERE id = $1 AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: SetDefaultReasoningTemplate :exec
UPDATE reasoning_templates
SET is_default = CASE WHEN id = $2 THEN true ELSE false END,
    updated_at = NOW()
WHERE user_id = $1;

-- name: IncrementTemplateUsage :exec
UPDATE reasoning_templates
SET times_used = times_used + 1, updated_at = NOW()
WHERE id = $1;

-- name: DeleteReasoningTemplate :exec
DELETE FROM reasoning_templates
WHERE id = $1 AND user_id = $2;

-- name: ClearDefaultTemplate :exec
UPDATE reasoning_templates
SET is_default = false, updated_at = NOW()
WHERE user_id = $1 AND is_default = true;
