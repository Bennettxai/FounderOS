-- Credential Vault Queries
-- SQLC queries for the credential_vault table

-- name: StoreCredential :one
-- Store or update a credential in the vault
INSERT INTO credential_vault (
    user_id,
    provider_id,
    credential_type,
    encrypted_data,
    encryption_version,
    expires_at,
    external_account_id,
    external_account_email,
    external_workspace_id,
    external_workspace_name,
    scopes,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (user_id, provider_id)
DO UPDATE SET
    credential_type = EXCLUDED.credential_type,
    encrypted_data = EXCLUDED.encrypted_data,
    encryption_version = EXCLUDED.encryption_version,
    expires_at = EXCLUDED.expires_at,
    external_account_id = EXCLUDED.external_account_id,
    external_account_email = EXCLUDED.external_account_email,
    external_workspace_id = EXCLUDED.external_workspace_id,
    external_workspace_name = EXCLUDED.external_workspace_name,
    scopes = EXCLUDED.scopes,
    metadata = EXCLUDED.metadata,
    updated_at = NOW(),
    last_rotated_at = CASE
        WHEN credential_vault.encrypted_data != EXCLUDED.encrypted_data THEN NOW()
        ELSE credential_vault.last_rotated_at
    END
RETURNING *;

-- name: GetCredential :one
-- Get a credential from the vault
SELECT * FROM credential_vault
WHERE user_id = $1 AND provider_id = $2;

-- name: GetCredentialsByUser :many
-- Get all credentials for a user
SELECT * FROM credential_vault
WHERE user_id = $1
ORDER BY provider_id;

-- name: GetCredentialsByProvider :many
-- Get all credentials for a provider (admin use)
SELECT * FROM credential_vault
WHERE provider_id = $1
ORDER BY created_at DESC;

-- name: UpdateCredentialLastUsed :exec
-- Update last_used_at timestamp
UPDATE credential_vault
SET last_used_at = NOW()
WHERE user_id = $1 AND provider_id = $2;

-- name: UpdateCredentialExpiry :exec
-- Update token expiry after refresh
UPDATE credential_vault
SET
    encrypted_data = $3,
    expires_at = $4,
    updated_at = NOW(),
    last_rotated_at = NOW()
WHERE user_id = $1 AND provider_id = $2;

-- name: DeleteCredential :exec
-- Delete a credential from the vault
DELETE FROM credential_vault
WHERE user_id = $1 AND provider_id = $2;

-- name: DeleteAllUserCredentials :exec
-- Delete all credentials for a user (account deletion)
DELETE FROM credential_vault
WHERE user_id = $1;

-- name: GetExpiringCredentials :many
-- Get credentials expiring within a time window (for proactive refresh)
SELECT * FROM credential_vault
WHERE expires_at IS NOT NULL
  AND expires_at < NOW() + $1::interval
  AND expires_at > NOW()
ORDER BY expires_at ASC;

-- name: GetExpiredCredentials :many
-- Get already expired credentials
SELECT * FROM credential_vault
WHERE expires_at IS NOT NULL
  AND expires_at <= NOW()
ORDER BY expires_at ASC;

-- name: CountUserCredentials :one
-- Count credentials for a user
SELECT COUNT(*) FROM credential_vault
WHERE user_id = $1;

-- name: CredentialExists :one
-- Check if a credential exists
SELECT EXISTS(
    SELECT 1 FROM credential_vault
    WHERE user_id = $1 AND provider_id = $2
) as exists;

-- ============================================================================
-- Webhook Queries
-- ============================================================================

-- name: CreateWebhook :one
INSERT INTO integration_webhooks (
    user_id,
    provider_id,
    webhook_url,
    webhook_secret_encrypted,
    events,
    status
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, provider_id, webhook_url)
DO UPDATE SET
    webhook_secret_encrypted = EXCLUDED.webhook_secret_encrypted,
    events = EXCLUDED.events,
    status = EXCLUDED.status,
    updated_at = NOW()
RETURNING *;

-- name: GetWebhook :one
SELECT * FROM integration_webhooks
WHERE id = $1;

-- name: GetWebhooksByUser :many
SELECT * FROM integration_webhooks
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetWebhooksByProvider :many
SELECT * FROM integration_webhooks
WHERE user_id = $1 AND provider_id = $2
ORDER BY created_at DESC;

-- name: UpdateWebhookStatus :exec
UPDATE integration_webhooks
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: RecordWebhookTrigger :exec
UPDATE integration_webhooks
SET
    last_triggered_at = NOW(),
    failure_count = CASE WHEN $2 THEN 0 ELSE failure_count + 1 END,
    last_error = CASE WHEN $2 THEN NULL ELSE $3 END,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteWebhook :exec
DELETE FROM integration_webhooks WHERE id = $1;

-- name: DeleteUserWebhooks :exec
DELETE FROM integration_webhooks WHERE user_id = $1;

-- ============================================================================
-- Data Sync Mapping Queries
-- ============================================================================

-- name: CreateSyncMapping :one
INSERT INTO data_sync_mappings (
    user_id,
    source_provider,
    source_entity,
    target_module,
    target_entity,
    field_mappings,
    transform_rules,
    enabled,
    sync_direction,
    sync_frequency
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id, source_provider, source_entity, target_module)
DO UPDATE SET
    target_entity = EXCLUDED.target_entity,
    field_mappings = EXCLUDED.field_mappings,
    transform_rules = EXCLUDED.transform_rules,
    enabled = EXCLUDED.enabled,
    sync_direction = EXCLUDED.sync_direction,
    sync_frequency = EXCLUDED.sync_frequency,
    updated_at = NOW()
RETURNING *;

-- name: GetSyncMapping :one
SELECT * FROM data_sync_mappings WHERE id = $1;

-- name: GetSyncMappingsByUser :many
SELECT * FROM data_sync_mappings
WHERE user_id = $1
ORDER BY source_provider, source_entity;

-- name: GetSyncMappingsByProvider :many
SELECT * FROM data_sync_mappings
WHERE user_id = $1 AND source_provider = $2
ORDER BY source_entity;

-- name: GetEnabledSyncMappings :many
SELECT * FROM data_sync_mappings
WHERE enabled = true
ORDER BY user_id, source_provider;

-- name: UpdateSyncMappingLastSynced :exec
UPDATE data_sync_mappings
SET
    last_synced_at = NOW(),
    records_synced = records_synced + $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ToggleSyncMapping :exec
UPDATE data_sync_mappings
SET enabled = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteSyncMapping :exec
DELETE FROM data_sync_mappings WHERE id = $1;

-- name: DeleteUserSyncMappings :exec
DELETE FROM data_sync_mappings WHERE user_id = $1;
