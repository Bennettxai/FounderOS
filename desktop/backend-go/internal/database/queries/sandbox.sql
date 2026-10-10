-- =============================================================================
-- SANDBOX CONTAINER QUERIES
-- Queries for managing Docker sandbox deployments
-- =============================================================================

-- name: UpdateAppSandboxInfo :one
-- Updates sandbox deployment information for an app
UPDATE osa_module_instances
SET
    container_id = $2,
    sandbox_port = $3,
    sandbox_url = $4,
    sandbox_status = $5,
    container_image = $6,
    app_type = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetAppSandboxInfo :one
-- Gets sandbox information for a specific app
SELECT
    id,
    workspace_id,
    name,
    display_name,
    container_id,
    sandbox_port,
    sandbox_url,
    sandbox_status,
    container_image,
    app_type,
    last_health_check,
    health_status,
    created_at,
    updated_at
FROM osa_module_instances
WHERE id = $1;

-- name: UpdateAppSandboxStatus :one
-- Updates only the sandbox status
UPDATE osa_module_instances
SET
    sandbox_status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateAppHealthStatus :one
-- Updates health check information
UPDATE osa_module_instances
SET
    health_status = $2,
    last_health_check = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListRunningSandboxes :many
-- Lists all apps with running sandbox containers
SELECT
    id,
    workspace_id,
    name,
    display_name,
    container_id,
    sandbox_port,
    sandbox_url,
    sandbox_status,
    container_image,
    app_type,
    last_health_check,
    health_status,
    created_at,
    updated_at
FROM osa_module_instances
WHERE sandbox_status = 'running'
ORDER BY updated_at DESC;

-- name: ListSandboxesByStatus :many
-- Lists all apps with a specific sandbox status
SELECT
    id,
    workspace_id,
    name,
    display_name,
    container_id,
    sandbox_port,
    sandbox_url,
    sandbox_status,
    container_image,
    app_type,
    last_health_check,
    health_status,
    created_at,
    updated_at
FROM osa_module_instances
WHERE sandbox_status = $1
ORDER BY updated_at DESC;

-- name: ListStoppedSandboxes :many
-- Lists all stopped sandboxes (for cleanup)
SELECT
    id,
    workspace_id,
    name,
    container_id,
    sandbox_port,
    sandbox_status,
    updated_at
FROM osa_module_instances
WHERE sandbox_status = 'stopped'
ORDER BY updated_at ASC;

-- name: ListUserSandboxes :many
-- Lists all sandbox apps for a specific user (via workspace)
SELECT
    a.id,
    a.workspace_id,
    a.name,
    a.display_name,
    a.container_id,
    a.sandbox_port,
    a.sandbox_url,
    a.sandbox_status,
    a.container_image,
    a.app_type,
    a.last_health_check,
    a.health_status,
    a.created_at,
    a.updated_at
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE w.user_id = $1
  AND a.sandbox_status != 'none'
ORDER BY a.updated_at DESC;

-- name: CountUserRunningSandboxes :one
-- Counts running sandboxes for quota enforcement
SELECT COUNT(*) as count
FROM osa_module_instances a
JOIN osa_workspaces w ON a.workspace_id = w.id
WHERE w.user_id = $1
  AND a.sandbox_status = 'running';

-- name: GetSandboxByPort :one
-- Gets app info by allocated port (for conflict detection)
SELECT
    id,
    workspace_id,
    name,
    container_id,
    sandbox_port,
    sandbox_status
FROM osa_module_instances
WHERE sandbox_port = $1
  AND sandbox_status IN ('running', 'deploying', 'pending')
LIMIT 1;

-- name: GetSandboxByContainerID :one
-- Gets app info by Docker container ID
SELECT
    id,
    workspace_id,
    name,
    display_name,
    container_id,
    sandbox_port,
    sandbox_url,
    sandbox_status,
    container_image,
    app_type,
    last_health_check,
    health_status
FROM osa_module_instances
WHERE container_id = $1;

-- name: ClearSandboxInfo :exec
-- Clears sandbox information when container is removed
UPDATE osa_module_instances
SET
    container_id = NULL,
    sandbox_port = NULL,
    sandbox_url = NULL,
    sandbox_status = 'none',
    health_status = 'unknown',
    last_health_check = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: ListStaleHealthChecks :many
-- Lists sandboxes that haven't had a health check recently (for monitoring)
SELECT
    id,
    workspace_id,
    name,
    container_id,
    sandbox_port,
    sandbox_status,
    last_health_check,
    health_status
FROM osa_module_instances
WHERE sandbox_status = 'running'
  AND (last_health_check IS NULL OR last_health_check < NOW() - INTERVAL '5 minutes')
ORDER BY last_health_check ASC NULLS FIRST;

-- =============================================================================
-- SANDBOX EVENTS QUERIES
-- =============================================================================

-- name: InsertSandboxEvent :one
-- Records a sandbox lifecycle event
INSERT INTO sandbox_events (
    module_instance_id,
    event_type,
    container_id,
    details
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetSandboxEvents :many
-- Gets events for a specific app
SELECT
    id,
    module_instance_id,
    event_type,
    container_id,
    details,
    created_at
FROM sandbox_events
WHERE module_instance_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetRecentSandboxEvents :many
-- Gets recent events across all apps (for admin/debugging)
SELECT
    e.id,
    e.module_instance_id,
    e.event_type,
    e.container_id,
    e.details,
    e.created_at,
    a.name as app_name,
    a.display_name as app_display_name
FROM sandbox_events e
JOIN osa_module_instances a ON e.module_instance_id = a.id
ORDER BY e.created_at DESC
LIMIT $1;

-- name: GetSandboxEventsByType :many
-- Gets events of a specific type for an app
SELECT
    id,
    module_instance_id,
    event_type,
    container_id,
    details,
    created_at
FROM sandbox_events
WHERE module_instance_id = $1
  AND event_type = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: DeleteOldSandboxEvents :exec
-- Cleanup: Delete events older than specified interval
DELETE FROM sandbox_events
WHERE created_at < NOW() - $1::interval;

-- name: CountSandboxEventsByType :one
-- Count events by type for an app (useful for metrics)
SELECT
    event_type,
    COUNT(*) as count
FROM sandbox_events
WHERE module_instance_id = $1
GROUP BY event_type;
