-- name: CreateOSAModule :one
INSERT INTO osa_modules (
    name,
    display_name,
    description,
    module_type,
    schema_definition,
    api_definition,
    ui_definition,
    created_by,
    workspace_id,
    status,
    version,
    metadata,
    tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetOSAModule :one
SELECT * FROM osa_modules
WHERE id = $1;

-- name: GetOSAModuleByName :one
SELECT * FROM osa_modules
WHERE name = $1 AND workspace_id = $2;

-- name: ListOSAModules :many
SELECT * FROM osa_modules
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: ListOSAModulesByUser :many
SELECT * FROM osa_modules
WHERE created_by = $1
ORDER BY created_at DESC;

-- name: UpdateOSAModuleStatus :one
UPDATE osa_modules
SET status = $2,
    deployed_at = CASE WHEN $2 = 'active' THEN NOW() ELSE deployed_at END
WHERE id = $1
RETURNING *;

-- name: DeleteOSAModule :exec
DELETE FROM osa_modules
WHERE id = $1;

-- ============================================================================
-- OSA WORKSPACES
-- ============================================================================

-- name: CreateOSAWorkspace :one
INSERT INTO osa_workspaces (
    user_id,
    name,
    mode,
    layout,
    active_modules,
    template_type,
    settings
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetOSAWorkspace :one
SELECT * FROM osa_workspaces
WHERE id = $1;

-- name: GetOSAWorkspaceByUserAndName :one
SELECT * FROM osa_workspaces
WHERE user_id = $1 AND name = $2;

-- name: ListOSAWorkspacesByUser :many
SELECT * FROM osa_workspaces
WHERE user_id = $1
ORDER BY last_accessed_at DESC;

-- name: UpdateOSAWorkspaceLayout :one
UPDATE osa_workspaces
SET layout = $2,
    active_modules = $3,
    last_accessed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOSAWorkspaceMode :one
UPDATE osa_workspaces
SET mode = $2,
    last_accessed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOSAWorkspace :exec
DELETE FROM osa_workspaces
WHERE id = $1;

-- name: AddOSAWorkspaceModule :one
UPDATE osa_workspaces
SET active_modules = array_append(active_modules, $2),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- ============================================================================
-- OSA MODULE INSTANCES
-- ============================================================================

-- name: CreateOSAModuleInstance :one
INSERT INTO osa_module_instances (
    workspace_id,
    module_id,
    name,
    display_name,
    description,
    osa_workflow_id,
    osa_sandbox_id,
    code_repository,
    deployment_url,
    status,
    files_created,
    tests_passed,
    build_status,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetOSAModuleInstance :one
SELECT * FROM osa_module_instances
WHERE id = $1;

-- name: GetOSAModuleInstanceByWorkflowID :one
SELECT * FROM osa_module_instances
WHERE osa_workflow_id = $1;

-- name: ListOSAModuleInstancesByWorkspace :many
SELECT * FROM osa_module_instances
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: UpdateOSAModuleInstanceStatus :one
UPDATE osa_module_instances
SET status = $2,
    error_message = $3,
    deployment_url = COALESCE($4, deployment_url),
    generated_at = CASE WHEN $2 = 'generated' THEN NOW() ELSE generated_at END,
    deployed_at = CASE WHEN $2 = 'deployed' THEN NOW() ELSE deployed_at END
WHERE id = $1
RETURNING *;

-- name: UpdateOSAModuleInstanceBuildStatus :one
UPDATE osa_module_instances
SET build_status = $2,
    tests_passed = $3,
    files_created = $4,
    last_build_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOSAModuleInstance :exec
DELETE FROM osa_module_instances
WHERE id = $1;

-- ============================================================================
-- OSA EXECUTION HISTORY
-- ============================================================================

-- name: CreateOSAExecutionHistory :one
INSERT INTO osa_execution_history (
    user_id,
    module_instance_id,
    workspace_id,
    command,
    working_directory,
    environment_vars,
    output,
    error_output,
    exit_code,
    duration_ms,
    triggered_by,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: ListOSAExecutionHistory :many
SELECT * FROM osa_execution_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListOSAExecutionHistoryByApp :many
SELECT * FROM osa_execution_history
WHERE module_instance_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- ============================================================================
-- OSA SYNC STATUS
-- ============================================================================

-- name: CreateOSASyncStatus :one
INSERT INTO osa_sync_status (
    entity_type,
    entity_id,
    osa_entity_id,
    osa_entity_type,
    sync_status,
    sync_direction,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (entity_type, entity_id)
DO UPDATE SET
    osa_entity_id = EXCLUDED.osa_entity_id,
    osa_entity_type = EXCLUDED.osa_entity_type,
    sync_status = EXCLUDED.sync_status,
    updated_at = NOW()
RETURNING *;

-- name: GetOSASyncStatus :one
SELECT * FROM osa_sync_status
WHERE entity_type = $1 AND entity_id = $2;

-- name: UpdateOSASyncStatus :one
UPDATE osa_sync_status
SET sync_status = $2,
    last_sync_at = NOW(),
    error_count = CASE WHEN $2 = 'failed' THEN error_count + 1 ELSE 0 END,
    last_error = $3
WHERE entity_type = $1 AND entity_id = $4
RETURNING *;

-- name: ListPendingOSASyncs :many
SELECT * FROM osa_sync_status
WHERE sync_status IN ('pending', 'failed')
  AND (next_sync_at IS NULL OR next_sync_at <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- ============================================================================
-- OSA BUILD EVENTS
-- ============================================================================

-- name: CreateOSABuildEvent :one
INSERT INTO osa_build_events (
    module_instance_id,
    workspace_id,
    event_type,
    event_data,
    build_id,
    phase,
    progress_percent,
    status_message
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListOSABuildEvents :many
SELECT * FROM osa_build_events
WHERE module_instance_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListOSABuildEventsByBuildID :many
SELECT * FROM osa_build_events
WHERE build_id = $1
ORDER BY created_at ASC;

-- name: GetLatestOSABuildEvent :one
SELECT * FROM osa_build_events
WHERE module_instance_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- ============================================================================
-- OSA WEBHOOKS
-- ============================================================================

-- name: CreateOSAWebhook :one
INSERT INTO osa_webhooks (
    workspace_id,
    module_instance_id,
    event_type,
    webhook_url,
    secret_key,
    enabled,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetOSAWebhook :one
SELECT * FROM osa_webhooks
WHERE id = $1;

-- name: ListOSAWebhooksByWorkspace :many
SELECT * FROM osa_webhooks
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: ListActiveOSAWebhooksByEvent :many
SELECT * FROM osa_webhooks
WHERE event_type = $1 AND enabled = true;

-- name: UpdateOSAWebhookStatus :one
UPDATE osa_webhooks
SET enabled = $2
WHERE id = $1
RETURNING *;

-- name: IncrementOSAWebhookSuccess :exec
UPDATE osa_webhooks
SET success_count = success_count + 1,
    last_triggered_at = NOW()
WHERE id = $1;

-- name: IncrementOSAWebhookFailure :exec
UPDATE osa_webhooks
SET failure_count = failure_count + 1,
    last_triggered_at = NOW()
WHERE id = $1;

-- name: DeleteOSAWebhook :exec
DELETE FROM osa_webhooks
WHERE id = $1;

-- ============================================================================
-- OSA APP MANAGEMENT (ISR-4)
-- ============================================================================

-- name: ListOSAModuleInstancesByUser :many
SELECT
    a.*,
    w.user_id,
    (SELECT COUNT(*) FROM osa_build_events WHERE module_instance_id = a.id) as build_events_count
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE w.user_id = $1
    AND ($2::UUID IS NULL OR a.workspace_id = $2)
    AND ($3::TEXT = '' OR a.status = $3)
ORDER BY a.created_at DESC
LIMIT $4
OFFSET $5;

-- name: GetOSAModuleInstanceByID :one
SELECT
    a.*,
    w.user_id,
    (SELECT COUNT(*) FROM osa_build_events WHERE module_instance_id = a.id) as build_events_count
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE a.id = $1;

-- name: GetOSAModuleInstanceByIDWithAuth :one
SELECT
    a.*,
    w.user_id
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE a.id = $1 AND w.user_id = $2;

-- name: UpdateOSAModuleInstanceMetadata :one
UPDATE osa_module_instances
SET
    display_name = COALESCE($2, display_name),
    description = COALESCE($3, description),
    metadata = COALESCE($4, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountOSAModuleInstancesByUser :one
SELECT COUNT(DISTINCT a.id)
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE w.user_id = $1
    AND ($2::UUID IS NULL OR a.workspace_id = $2)
    AND ($3::TEXT = '' OR a.status = $3);

-- name: GetOSAAppLogs :many
SELECT *
FROM osa_build_events
WHERE module_instance_id = $1
    AND ($2::TEXT = '' OR event_type = $2)
ORDER BY created_at DESC
LIMIT $3
OFFSET $4;

-- name: ListOSAAppSnapshots :many
-- Returns all generation snapshots for apps with the same name in the same workspace
SELECT a.id, a.name, a.display_name, a.status, a.files_created,
       a.created_at, a.generated_at,
       (SELECT COUNT(*) FROM osa_module_files f WHERE f.module_instance_id = a.id) as file_count
FROM osa_module_instances a
WHERE a.workspace_id = (SELECT ws.workspace_id FROM osa_module_instances ws WHERE ws.id = $1)
  AND a.name = (SELECT n.name FROM osa_module_instances n WHERE n.id = $1)
ORDER BY a.created_at DESC;
