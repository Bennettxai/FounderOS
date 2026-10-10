-- Usage Analytics Queries

-- name: CreateAIUsageLog :one
INSERT INTO ai_usage_logs (
    user_id, conversation_id, provider, model,
    input_tokens, output_tokens, total_tokens, thinking_tokens,
    agent_name, delegated_to, parent_request_id,
    request_type, node_id, project_id,
    duration_ms, started_at, completed_at, estimated_cost
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) RETURNING *;

-- name: GetAIUsageLogs :many
SELECT * FROM ai_usage_logs
WHERE user_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAIUsageLogsByDateRange :many
SELECT * FROM ai_usage_logs
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3
ORDER BY started_at DESC;

-- name: GetAIUsageSummaryByProvider :many
SELECT
    provider,
    COUNT(*) as request_count,
    COALESCE(SUM(input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(total_tokens), 0)::bigint as total_tokens,
    COALESCE(SUM(estimated_cost), 0)::bigint as total_cost
FROM ai_usage_logs
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3
GROUP BY provider;

-- name: GetAIUsageSummaryByModel :many
SELECT
    model,
    provider,
    COUNT(*) as request_count,
    COALESCE(SUM(input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(total_tokens), 0)::bigint as total_tokens,
    COALESCE(SUM(estimated_cost), 0)::bigint as total_cost
FROM ai_usage_logs
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3
GROUP BY model, provider;

-- name: GetAIUsageSummaryByAgent :many
SELECT
    agent_name,
    COUNT(*) as request_count,
    COALESCE(SUM(input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(total_tokens), 0)::bigint as total_tokens,
    COALESCE(AVG(duration_ms), 0)::float8 as avg_duration_ms
FROM ai_usage_logs
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3
  AND agent_name IS NOT NULL
GROUP BY agent_name;

-- name: GetTotalTokensForPeriod :one
SELECT
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens,
    COALESCE(SUM(total_tokens), 0) as total_tokens,
    COALESCE(SUM(thinking_tokens), 0) as total_thinking_tokens,
    COALESCE(SUM(estimated_cost), 0) as total_cost,
    COUNT(*) as total_requests
FROM ai_usage_logs
WHERE user_id = $1
  AND started_at >= $2
  AND started_at < $3;

-- name: CreateMCPUsageLog :one
INSERT INTO mcp_usage_logs (
    user_id, tool_name, server_name,
    input_params, output_result, success, error_message,
    duration_ms, conversation_id, ai_request_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetMCPUsageLogs :many
SELECT * FROM mcp_usage_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMCPUsageSummaryByTool :many
SELECT
    tool_name,
    server_name,
    COUNT(*) as request_count,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
    AVG(duration_ms) as avg_duration_ms
FROM mcp_usage_logs
WHERE user_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY tool_name, server_name;

-- name: UpsertDailySummary :one
INSERT INTO usage_daily_summary (
    user_id, date,
    ai_requests, ai_input_tokens, ai_output_tokens, ai_total_tokens, ai_thinking_tokens, ai_estimated_cost,
    provider_breakdown, model_breakdown, agent_breakdown,
    mcp_requests, mcp_tool_breakdown,
    conversations_created, messages_sent, artifacts_created, documents_created,
    contexts_accessed, nodes_accessed, projects_accessed
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
)
ON CONFLICT (user_id, date) DO UPDATE SET
    ai_requests = usage_daily_summary.ai_requests + EXCLUDED.ai_requests,
    ai_input_tokens = usage_daily_summary.ai_input_tokens + EXCLUDED.ai_input_tokens,
    ai_output_tokens = usage_daily_summary.ai_output_tokens + EXCLUDED.ai_output_tokens,
    ai_total_tokens = usage_daily_summary.ai_total_tokens + EXCLUDED.ai_total_tokens,
    ai_thinking_tokens = usage_daily_summary.ai_thinking_tokens + EXCLUDED.ai_thinking_tokens,
    ai_estimated_cost = usage_daily_summary.ai_estimated_cost + EXCLUDED.ai_estimated_cost,
    mcp_requests = usage_daily_summary.mcp_requests + EXCLUDED.mcp_requests,
    conversations_created = usage_daily_summary.conversations_created + EXCLUDED.conversations_created,
    messages_sent = usage_daily_summary.messages_sent + EXCLUDED.messages_sent,
    artifacts_created = usage_daily_summary.artifacts_created + EXCLUDED.artifacts_created,
    documents_created = usage_daily_summary.documents_created + EXCLUDED.documents_created,
    updated_at = NOW()
RETURNING *;

-- name: GetDailySummary :one
SELECT * FROM usage_daily_summary
WHERE user_id = $1 AND date = $2;

-- name: GetDailySummaries :many
SELECT * FROM usage_daily_summary
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
ORDER BY date DESC;

-- name: GetUsageTrend :many
SELECT
    date,
    ai_requests,
    ai_total_tokens,
    ai_thinking_tokens,
    ai_estimated_cost,
    mcp_requests,
    messages_sent
FROM usage_daily_summary
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
ORDER BY date ASC;

-- name: CreateSystemEvent :exec
INSERT INTO system_event_logs (
    user_id, event_type, event_name, event_data,
    module, resource_type, resource_id, session_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetRecentSystemEvents :many
SELECT * FROM system_event_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetSystemEventsByModule :many
SELECT
    module,
    event_name,
    COUNT(*) as event_count
FROM system_event_logs
WHERE user_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY module, event_name
ORDER BY event_count DESC;
