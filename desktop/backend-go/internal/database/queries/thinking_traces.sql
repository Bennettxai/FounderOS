-- name: CreateThinkingTrace :one
INSERT INTO thinking_traces (
    user_id, conversation_id, message_id, thinking_content, thinking_type,
    step_number, started_at, thinking_tokens, model_used, reasoning_template_id, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateThinkingTraceComplete :one
UPDATE thinking_traces
SET completed_at = $2, duration_ms = $3, thinking_tokens = $4
WHERE id = $1
RETURNING *;

-- name: AppendThinkingContent :one
UPDATE thinking_traces
SET thinking_content = thinking_content || $2, thinking_tokens = $3
WHERE id = $1
RETURNING *;

-- name: GetThinkingTrace :one
SELECT * FROM thinking_traces
WHERE id = $1 AND user_id = $2;

-- name: GetThinkingTraceByMessage :one
SELECT * FROM thinking_traces
WHERE message_id = $1 AND user_id = $2;

-- name: ListThinkingTracesByConversation :many
SELECT * FROM thinking_traces
WHERE conversation_id = $1 AND user_id = $2
ORDER BY step_number ASC, created_at ASC;

-- name: ListThinkingTracesByMessage :many
SELECT * FROM thinking_traces
WHERE message_id = $1
ORDER BY step_number ASC;

-- name: DeleteThinkingTrace :exec
DELETE FROM thinking_traces
WHERE id = $1 AND user_id = $2;

-- name: DeleteThinkingTracesByConversation :exec
DELETE FROM thinking_traces
WHERE conversation_id = $1 AND user_id = $2;

-- name: GetThinkingStats :one
SELECT
    COUNT(*) as total_traces,
    COALESCE(SUM(thinking_tokens), 0) as total_tokens,
    COALESCE(AVG(duration_ms), 0) as avg_duration_ms
FROM thinking_traces
WHERE user_id = $1
    AND (sqlc.narg(conversation_id)::uuid IS NULL OR conversation_id = sqlc.narg(conversation_id))
    AND created_at >= COALESCE(sqlc.narg(since)::timestamptz, NOW() - INTERVAL '30 days');
