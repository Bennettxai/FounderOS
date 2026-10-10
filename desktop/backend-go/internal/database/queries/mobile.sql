-- Mobile API Queries
-- Optimized for: cursor pagination, lean payloads, efficient filtering

-- TASKS

-- name: ListTasksForMobile :many
-- Cursor pagination with optional status/due filters
SELECT 
    t.id,
    t.title,
    t.status,
    t.priority,
    t.due_date,
    t.updated_at,
    p.name AS project_name,
    tm.name AS assignee_name
FROM tasks t
LEFT JOIN projects p ON t.project_id = p.id
LEFT JOIN team_members tm ON t.assignee_id = tm.id
WHERE t.user_id = $1
  AND (
    sqlc.narg(cursor_updated_at)::timestamp IS NULL 
    OR t.updated_at < sqlc.narg(cursor_updated_at)
    OR (t.updated_at = sqlc.narg(cursor_updated_at) AND t.id < sqlc.narg(cursor_id)::uuid)
  )
  AND (sqlc.narg(status)::taskstatus IS NULL OR t.status = sqlc.narg(status))
  AND (
    sqlc.narg(due_filter)::text IS NULL
    OR (sqlc.narg(due_filter) = 'today' AND DATE(t.due_date) = CURRENT_DATE)
    OR (sqlc.narg(due_filter) = 'week' AND t.due_date >= CURRENT_DATE AND t.due_date < CURRENT_DATE + INTERVAL '7 days')
    OR (sqlc.narg(due_filter) = 'overdue' AND t.due_date < CURRENT_DATE AND t.status != 'done')
  )
ORDER BY t.updated_at DESC, t.id DESC
LIMIT sqlc.arg(limit_count);

-- name: GetTaskForMobile :one
SELECT 
    t.id,
    t.title,
    t.description,
    t.status,
    t.priority,
    t.due_date,
    t.start_date,
    t.completed_at,
    t.created_at,
    t.updated_at,
    t.project_id,
    t.assignee_id,
    p.name AS project_name,
    tm.id AS assignee_uuid,
    tm.name AS assignee_name,
    tm.avatar_url AS assignee_avatar
FROM tasks t
LEFT JOIN projects p ON t.project_id = p.id
LEFT JOIN team_members tm ON t.assignee_id = tm.id
WHERE t.id = $1 AND t.user_id = $2;

-- name: CountSubtasksForTask :one
SELECT COUNT(*) FROM tasks
WHERE parent_task_id = $1;

-- name: QuickCreateTask :one
INSERT INTO tasks (user_id, title, status, priority, due_date)
VALUES ($1, $2, 'todo', COALESCE(sqlc.narg(priority)::taskpriority, 'medium'), sqlc.narg(due_date))
RETURNING id, title, status, priority, due_date, created_at, updated_at;

-- name: UpdateTaskStatusMobile :one
UPDATE tasks
SET 
    status = sqlc.arg(status)::taskstatus,
    completed_at = CASE WHEN sqlc.arg(status)::taskstatus = 'done' THEN NOW() ELSE NULL END,
    updated_at = NOW()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING id, status, completed_at, updated_at;

-- name: ToggleTaskStatusMobile :one
UPDATE tasks
SET 
    status = CASE WHEN status = 'done' THEN 'todo'::taskstatus ELSE 'done'::taskstatus END,
    completed_at = CASE WHEN status = 'done' THEN NULL ELSE NOW() END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, status, completed_at, updated_at;

-- name: CountTasksForUser :one
SELECT COUNT(*) FROM tasks WHERE user_id = $1;

-- NOTIFICATIONS

-- name: ListNotificationsForMobile :many
SELECT 
    id,
    type,
    title,
    body,
    entity_type,
    entity_id,
    priority,
    is_read,
    created_at
FROM notifications
WHERE user_id = $1
  AND (sqlc.narg(unread_only)::boolean IS NULL OR sqlc.narg(unread_only) = FALSE OR is_read = FALSE)
  AND (
    sqlc.narg(cursor_created_at)::timestamptz IS NULL 
    OR created_at < sqlc.narg(cursor_created_at)
    OR (created_at = sqlc.narg(cursor_created_at) AND id < sqlc.narg(cursor_id)::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) FROM notifications 
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationsAsRead :exec
UPDATE notifications
SET is_read = TRUE, read_at = NOW()
WHERE user_id = $1 AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: MarkAllNotificationsAsRead :execrows
UPDATE notifications
SET is_read = TRUE, read_at = NOW()
WHERE user_id = $1 AND is_read = FALSE;

-- DAILY LOGS

-- name: GetTodayDailyLogForMobile :one
SELECT id, date, content, energy_level, extracted_actions, extracted_patterns, transcription_source
FROM daily_logs
WHERE user_id = $1 AND date = CURRENT_DATE;

-- name: GetDailyLogHistoryForMobile :many
SELECT 
    id,
    date,
    SUBSTRING(content FROM 1 FOR 200) AS summary,
    energy_level
FROM daily_logs
WHERE user_id = $1
  AND (sqlc.narg(before_date)::date IS NULL OR date < sqlc.narg(before_date))
ORDER BY date DESC
LIMIT sqlc.arg(limit_count);

-- CHAT

-- name: ListConversationsForMobile :many
SELECT 
    c.id,
    c.title,
    c.updated_at,
    (
        SELECT SUBSTRING(content FROM 1 FOR 100)
        FROM messages 
        WHERE conversation_id = c.id 
        ORDER BY created_at DESC 
        LIMIT 1
    ) AS last_message
FROM conversations c
WHERE c.user_id = $1
ORDER BY c.updated_at DESC
LIMIT sqlc.arg(limit_count);

-- name: GetMessagesForMobile :many
SELECT 
    id,
    role,
    content,
    created_at
FROM messages
WHERE conversation_id = $1
  AND (
    sqlc.narg(cursor_created_at)::timestamp IS NULL 
    OR created_at < sqlc.narg(cursor_created_at)
    OR (created_at = sqlc.narg(cursor_created_at) AND id < sqlc.narg(cursor_id)::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- SYNC

-- name: GetTaskChangesSince :many
SELECT 
    t.id,
    t.title,
    t.status,
    t.priority,
    t.due_date,
    t.updated_at,
    p.name AS project_name,
    tm.name AS assignee_name
FROM tasks t
LEFT JOIN projects p ON t.project_id = p.id
LEFT JOIN team_members tm ON t.assignee_id = tm.id
WHERE t.user_id = $1
  AND t.updated_at > $2
ORDER BY t.updated_at ASC
LIMIT sqlc.arg(limit_count);

-- name: GetNotificationChangesSince :many
SELECT id, type, title, body, entity_type, entity_id, priority, is_read, created_at
FROM notifications
WHERE user_id = $1 AND created_at > $2
ORDER BY created_at ASC
LIMIT sqlc.arg(limit_count);

-- PUSH DEVICES

-- name: RegisterPushDevice :one
INSERT INTO push_devices (user_id, device_id, platform, push_token, app_version, os_version, device_model)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, device_id) DO UPDATE SET
    push_token = EXCLUDED.push_token,
    platform = EXCLUDED.platform,
    app_version = EXCLUDED.app_version,
    os_version = EXCLUDED.os_version,
    device_model = EXCLUDED.device_model,
    is_active = true,
    last_used_at = NOW(),
    updated_at = NOW()
RETURNING id, device_id, platform, created_at;

-- name: UnregisterPushDevice :exec
UPDATE push_devices SET is_active = false, updated_at = NOW()
WHERE user_id = $1 AND device_id = $2;

-- name: GetUserPushDevices :many
SELECT id, device_id, platform, push_token, is_active, last_used_at
FROM push_devices
WHERE user_id = $1 AND is_active = true;
