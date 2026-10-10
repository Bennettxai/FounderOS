-- Integrations queries for SQLC

-- ============================================================================
-- Integration Providers
-- ============================================================================

-- name: GetAllIntegrationProviders :many
SELECT * FROM integration_providers
ORDER BY category, name;

-- name: GetIntegrationProvidersByCategory :many
SELECT * FROM integration_providers
WHERE category = $1 AND status != 'deprecated'
ORDER BY name;

-- name: GetIntegrationProvidersByModule :many
SELECT * FROM integration_providers
WHERE $1 = ANY(modules) AND status != 'deprecated'
ORDER BY category, name;

-- name: GetIntegrationProvider :one
SELECT * FROM integration_providers
WHERE id = $1;

-- name: GetAvailableIntegrationProviders :many
SELECT * FROM integration_providers
WHERE status = 'available'
ORDER BY category, name;

-- ============================================================================
-- User Integrations
-- ============================================================================

-- name: GetUserIntegrations :many
SELECT ui.*, ip.name as provider_name, ip.category, ip.icon_url, ip.skills
FROM user_integrations ui
JOIN integration_providers ip ON ui.provider_id = ip.id
WHERE ui.user_id = $1
ORDER BY ui.connected_at DESC;

-- name: GetUserIntegration :one
SELECT ui.*, ip.name as provider_name, ip.category, ip.icon_url, ip.skills
FROM user_integrations ui
JOIN integration_providers ip ON ui.provider_id = ip.id
WHERE ui.id = $1 AND ui.user_id = $2;

-- name: GetUserIntegrationByProvider :one
SELECT * FROM user_integrations
WHERE user_id = $1 AND provider_id = $2;

-- name: UpsertUserIntegrationConnection :one
-- Used by unified provider system
INSERT INTO user_integrations (
    user_id, provider_id, status,
    external_account_id, external_account_name,
    scopes, connected_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id, provider_id)
DO UPDATE SET
    status = EXCLUDED.status,
    external_account_id = COALESCE(EXCLUDED.external_account_id, user_integrations.external_account_id),
    external_account_name = COALESCE(EXCLUDED.external_account_name, user_integrations.external_account_name),
    scopes = COALESCE(EXCLUDED.scopes, user_integrations.scopes),
    connected_at = CASE
        WHEN EXCLUDED.status = 'connected' AND user_integrations.status != 'connected'
        THEN NOW()
        ELSE user_integrations.connected_at
    END,
    updated_at = NOW()
RETURNING *;

-- name: UpdateUserIntegrationDisconnect :exec
-- Disconnect an integration
UPDATE user_integrations SET
    status = 'disconnected',
    updated_at = NOW()
WHERE user_id = $1 AND provider_id = $2;

-- name: UpdateUserIntegrationSyncTime :exec
-- Update last used/sync time
UPDATE user_integrations SET
    last_used_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1 AND provider_id = $2;

-- name: GetUserIntegrationStats :one
-- Get sync stats for an integration
SELECT
    ui.id,
    ui.status,
    ui.connected_at,
    ui.last_used_at,
    ui.external_account_name,
    ui.scopes,
    COALESCE(
        (SELECT COUNT(*) FROM calendar_events WHERE user_id = ui.user_id AND source = 'google'),
        0
    )::int as calendar_events_count
FROM user_integrations ui
WHERE ui.user_id = $1 AND ui.provider_id = $2;

-- name: CreateUserIntegration :one
INSERT INTO user_integrations (
    user_id, provider_id, status,
    access_token_encrypted, refresh_token_encrypted, token_expires_at, scopes,
    external_account_id, external_account_name,
    external_workspace_id, external_workspace_name,
    metadata, settings
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: UpdateUserIntegration :one
UPDATE user_integrations SET
    status = COALESCE($3, status),
    access_token_encrypted = COALESCE($4, access_token_encrypted),
    refresh_token_encrypted = COALESCE($5, refresh_token_encrypted),
    token_expires_at = COALESCE($6, token_expires_at),
    scopes = COALESCE($7, scopes),
    external_account_id = COALESCE($8, external_account_id),
    external_account_name = COALESCE($9, external_account_name),
    metadata = COALESCE($10, metadata),
    settings = COALESCE($11, settings),
    last_used_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateUserIntegrationLastUsed :exec
UPDATE user_integrations SET
    last_used_at = NOW()
WHERE id = $1;

-- name: DisconnectUserIntegration :exec
UPDATE user_integrations SET
    status = 'disconnected',
    access_token_encrypted = NULL,
    refresh_token_encrypted = NULL,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteUserIntegration :exec
DELETE FROM user_integrations
WHERE id = $1 AND user_id = $2;

-- name: GetConnectedIntegrationsForModule :many
SELECT ui.*, ip.name as provider_name, ip.skills
FROM user_integrations ui
JOIN integration_providers ip ON ui.provider_id = ip.id
WHERE ui.user_id = $1
  AND ui.status = 'connected'
  AND $2 = ANY(ip.modules);

-- ============================================================================
-- Module Integration Settings
-- ============================================================================

-- name: GetModuleIntegrationSettings :many
SELECT mis.*, ip.name as provider_name
FROM module_integration_settings mis
JOIN integration_providers ip ON mis.provider_id = ip.id
WHERE mis.user_id = $1 AND mis.module_id = $2;

-- name: GetModuleIntegrationSetting :one
SELECT * FROM module_integration_settings
WHERE user_id = $1 AND module_id = $2 AND provider_id = $3;

-- name: UpsertModuleIntegrationSetting :one
INSERT INTO module_integration_settings (
    user_id, module_id, provider_id,
    enabled, sync_direction, sync_frequency, custom_settings
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, module_id, provider_id)
DO UPDATE SET
    enabled = EXCLUDED.enabled,
    sync_direction = EXCLUDED.sync_direction,
    sync_frequency = EXCLUDED.sync_frequency,
    custom_settings = EXCLUDED.custom_settings,
    updated_at = NOW()
RETURNING *;

-- ============================================================================
-- User Model Preferences
-- ============================================================================

-- name: GetUserModelPreferences :one
SELECT * FROM user_model_preferences
WHERE user_id = $1;

-- name: UpsertUserModelPreferences :one
INSERT INTO user_model_preferences (
    user_id, tier_2_model, tier_3_model, tier_4_model,
    tier_2_fallbacks, tier_3_fallbacks, tier_4_fallbacks,
    skill_overrides, allow_model_upgrade_on_failure, max_latency_ms, prefer_local
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (user_id)
DO UPDATE SET
    tier_2_model = EXCLUDED.tier_2_model,
    tier_3_model = EXCLUDED.tier_3_model,
    tier_4_model = EXCLUDED.tier_4_model,
    tier_2_fallbacks = EXCLUDED.tier_2_fallbacks,
    tier_3_fallbacks = EXCLUDED.tier_3_fallbacks,
    tier_4_fallbacks = EXCLUDED.tier_4_fallbacks,
    skill_overrides = EXCLUDED.skill_overrides,
    allow_model_upgrade_on_failure = EXCLUDED.allow_model_upgrade_on_failure,
    max_latency_ms = EXCLUDED.max_latency_ms,
    prefer_local = EXCLUDED.prefer_local,
    updated_at = NOW()
RETURNING *;

-- ============================================================================
-- Pending Decisions
-- ============================================================================

-- name: GetPendingDecisions :many
SELECT * FROM pending_decisions
WHERE user_id = $1 AND status = 'pending'
ORDER BY
    CASE priority
        WHEN 'urgent' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
    END,
    created_at ASC;

-- name: GetPendingDecision :one
SELECT * FROM pending_decisions
WHERE id = $1;

-- name: GetPendingDecisionByExecution :one
SELECT * FROM pending_decisions
WHERE execution_id = $1 AND step_id = $2;

-- name: CreatePendingDecision :one
INSERT INTO pending_decisions (
    execution_id, skill_id, step_id, user_id,
    question, description, options, input_fields, context,
    priority, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: RespondToDecision :one
UPDATE pending_decisions SET
    status = 'decided',
    decision = $2,
    decision_inputs = $3,
    decided_by = $4,
    decided_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: ExpirePendingDecisions :exec
UPDATE pending_decisions SET
    status = 'expired'
WHERE status = 'pending' AND expires_at < NOW();

-- name: CancelPendingDecision :exec
UPDATE pending_decisions SET
    status = 'cancelled'
WHERE id = $1;

-- ============================================================================
-- Integration Sync Log
-- ============================================================================

-- name: CreateIntegrationSyncLog :one
INSERT INTO integration_sync_log (
    user_integration_id, module_id, sync_type, direction, status
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSyncLog :one
UPDATE integration_sync_log SET
    status = $2,
    records_processed = $3,
    records_created = $4,
    records_updated = $5,
    records_failed = $6,
    error_message = $7,
    error_details = $8,
    completed_at = CASE WHEN $2 IN ('success', 'failed') THEN NOW() ELSE completed_at END
WHERE id = $1
RETURNING *;

-- name: GetRecentSyncLogs :many
SELECT * FROM integration_sync_log
WHERE user_integration_id = $1
ORDER BY started_at DESC
LIMIT $2;

-- ============================================================================
-- Skill Executions
-- ============================================================================

-- name: CreateSkillExecution :one
INSERT INTO skill_executions (
    skill_id, user_id, status, params, context
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSkillExecution :one
UPDATE skill_executions SET
    status = COALESCE($2, status),
    current_step = COALESCE($3, current_step),
    result = COALESCE($4, result),
    error = COALESCE($5, error),
    context = COALESCE($6, context),
    step_results = COALESCE($7, step_results),
    metrics = COALESCE($8, metrics),
    completed_at = CASE WHEN $2 IN ('complete', 'failed', 'cancelled') THEN NOW() ELSE completed_at END
WHERE id = $1
RETURNING *;

-- name: GetSkillExecution :one
SELECT * FROM skill_executions
WHERE id = $1;

-- name: GetUserSkillExecutions :many
SELECT * FROM skill_executions
WHERE user_id = $1
ORDER BY started_at DESC
LIMIT $2;
