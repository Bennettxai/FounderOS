-- ===== NOTIFICATION QUERIES =====

-- name: CreateNotification :one
INSERT INTO notifications (
    user_id, workspace_id, type, title, body,
    entity_type, entity_id, sender_id, sender_name, sender_avatar_url,
    batch_id, batch_count, priority, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetNotification :one
SELECT * FROM notifications
WHERE id = $1 AND user_id = $2;

-- name: GetNotificationsForUser :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUnreadNotificationsForUser :many
SELECT * FROM notifications
WHERE user_id = $1 AND is_read = FALSE
ORDER BY created_at DESC
LIMIT $2;

-- name: GetUnreadCount :one
SELECT COUNT(*)::bigint FROM notifications
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationAsRead :one
UPDATE notifications
SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: MarkMultipleAsRead :exec
UPDATE notifications
SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
WHERE id = ANY($1::uuid[]) AND user_id = $2;

-- name: MarkAllAsRead :exec
UPDATE notifications
SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND is_read = FALSE;

-- name: UpdateNotificationChannelsSent :exec
UPDATE notifications
SET channels_sent = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1 AND user_id = $2;

-- name: DeleteOldNotifications :exec
DELETE FROM notifications
WHERE created_at < NOW() - INTERVAL '90 days';

-- name: GetNotificationsByEntity :many
SELECT * FROM notifications
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC
LIMIT $3;

-- ===== NOTIFICATION PREFERENCES QUERIES =====

-- name: GetNotificationPreferences :one
SELECT * FROM notification_preferences
WHERE user_id = $1
  AND (workspace_id = $2 OR (workspace_id IS NULL AND $2::uuid IS NULL))
LIMIT 1;

-- name: GetNotificationPreferencesByUser :one
SELECT * FROM notification_preferences
WHERE user_id = $1 AND workspace_id IS NULL
LIMIT 1;

-- name: CreateNotificationPreferences :one
INSERT INTO notification_preferences (
    user_id, workspace_id, email_enabled, push_enabled, in_app_enabled,
    type_settings, quiet_hours_enabled, quiet_hours_start, quiet_hours_end,
    quiet_hours_timezone, email_digest_enabled, email_digest_time, email_digest_timezone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: UpdateNotificationPreferences :one
UPDATE notification_preferences
SET
    email_enabled = COALESCE(sqlc.narg(email_enabled), email_enabled),
    push_enabled = COALESCE(sqlc.narg(push_enabled), push_enabled),
    in_app_enabled = COALESCE(sqlc.narg(in_app_enabled), in_app_enabled),
    type_settings = COALESCE(sqlc.narg(type_settings), type_settings),
    quiet_hours_enabled = COALESCE(sqlc.narg(quiet_hours_enabled), quiet_hours_enabled),
    quiet_hours_start = COALESCE(sqlc.narg(quiet_hours_start), quiet_hours_start),
    quiet_hours_end = COALESCE(sqlc.narg(quiet_hours_end), quiet_hours_end),
    quiet_hours_timezone = COALESCE(sqlc.narg(quiet_hours_timezone), quiet_hours_timezone),
    email_digest_enabled = COALESCE(sqlc.narg(email_digest_enabled), email_digest_enabled),
    email_digest_time = COALESCE(sqlc.narg(email_digest_time), email_digest_time),
    email_digest_timezone = COALESCE(sqlc.narg(email_digest_timezone), email_digest_timezone),
    updated_at = NOW()
WHERE user_id = $1 AND (workspace_id = sqlc.narg(workspace_id) OR (workspace_id IS NULL AND sqlc.narg(workspace_id) IS NULL))
RETURNING *;

-- name: UpsertNotificationPreferences :one
INSERT INTO notification_preferences (
    user_id, workspace_id, email_enabled, push_enabled, in_app_enabled,
    type_settings, quiet_hours_enabled, quiet_hours_start, quiet_hours_end,
    quiet_hours_timezone, email_digest_enabled, email_digest_time, email_digest_timezone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) ON CONFLICT (user_id, workspace_id) DO UPDATE SET
    email_enabled = EXCLUDED.email_enabled,
    push_enabled = EXCLUDED.push_enabled,
    in_app_enabled = EXCLUDED.in_app_enabled,
    type_settings = EXCLUDED.type_settings,
    quiet_hours_enabled = EXCLUDED.quiet_hours_enabled,
    quiet_hours_start = EXCLUDED.quiet_hours_start,
    quiet_hours_end = EXCLUDED.quiet_hours_end,
    quiet_hours_timezone = EXCLUDED.quiet_hours_timezone,
    email_digest_enabled = EXCLUDED.email_digest_enabled,
    email_digest_time = EXCLUDED.email_digest_time,
    email_digest_timezone = EXCLUDED.email_digest_timezone,
    updated_at = NOW()
RETURNING *;

-- ===== NOTIFICATION BATCHES QUERIES =====

-- name: GetPendingBatch :one
SELECT * FROM notification_batches
WHERE user_id = $1 AND batch_key = $2 AND status = 'pending';

-- name: CreateBatch :one
INSERT INTO notification_batches (
    user_id, batch_key, type, entity_type, entity_id,
    pending_ids, pending_count, dispatch_at
) VALUES (
    $1, $2, $3, $4, $5, $6, 1, $7
) RETURNING *;

-- name: AddToBatch :one
UPDATE notification_batches
SET pending_ids = array_append(pending_ids, $2),
    pending_count = pending_count + 1
WHERE id = $1
RETURNING *;

-- name: GetBatchesReadyToDispatch :many
SELECT * FROM notification_batches
WHERE status = 'pending' AND dispatch_at <= NOW()
ORDER BY dispatch_at ASC
LIMIT 100;

-- name: MarkBatchDispatched :exec
UPDATE notification_batches
SET status = 'dispatched'
WHERE id = $1;

-- name: DeleteOldBatches :exec
DELETE FROM notification_batches
WHERE status = 'dispatched' AND dispatch_at < NOW() - INTERVAL '7 days';

-- name: GetBatchNotifications :many
SELECT * FROM notifications
WHERE batch_id = $1
ORDER BY created_at ASC;

-- ===== PUSH SUBSCRIPTION QUERIES =====

-- name: CreatePushSubscription :one
INSERT INTO push_subscriptions (
    user_id, endpoint, p256dh, auth, user_agent
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (endpoint) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth,
    user_agent = EXCLUDED.user_agent,
    updated_at = NOW()
RETURNING *;

-- name: GetPushSubscriptionsByUser :many
SELECT * FROM push_subscriptions
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions
WHERE endpoint = $1 AND user_id = $2;

-- name: DeletePushSubscriptionByEndpoint :exec
DELETE FROM push_subscriptions
WHERE endpoint = $1;

-- name: DeleteAllPushSubscriptionsForUser :exec
DELETE FROM push_subscriptions
WHERE user_id = $1;

-- name: GetAllPushSubscriptions :many
SELECT * FROM push_subscriptions
ORDER BY created_at DESC
LIMIT $1;