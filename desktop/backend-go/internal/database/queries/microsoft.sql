-- Microsoft 365 Queries
-- SQLC queries for Microsoft 365 integrations (Outlook, OneDrive, To Do)

-- ============================================================================
-- Microsoft OAuth Tokens
-- ============================================================================

-- name: UpsertMicrosoftToken :one
INSERT INTO microsoft_oauth_tokens (
    user_id, access_token, refresh_token, expiry, scopes,
    microsoft_id, microsoft_email
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    access_token = EXCLUDED.access_token,
    refresh_token = EXCLUDED.refresh_token,
    expiry = EXCLUDED.expiry,
    scopes = EXCLUDED.scopes,
    microsoft_id = EXCLUDED.microsoft_id,
    microsoft_email = EXCLUDED.microsoft_email,
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftToken :one
SELECT * FROM microsoft_oauth_tokens WHERE user_id = $1;

-- name: DeleteMicrosoftToken :exec
DELETE FROM microsoft_oauth_tokens WHERE user_id = $1;

-- name: GetMicrosoftTokensByScope :many
SELECT * FROM microsoft_oauth_tokens
WHERE $1 = ANY(scopes);

-- ============================================================================
-- Microsoft Mail Messages
-- ============================================================================

-- name: UpsertMicrosoftMailMessage :one
INSERT INTO microsoft_mail_messages (
    user_id, message_id, conversation_id, subject, body_preview,
    body_content, body_content_type, importance, from_email, from_name,
    to_recipients, cc_recipients, bcc_recipients, reply_to,
    is_read, is_draft, has_attachments, folder_id, folder_name,
    categories, flag_status, attachments, received_datetime, sent_datetime,
    created_datetime, last_modified_datetime
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
ON CONFLICT (user_id, message_id) DO UPDATE SET
    conversation_id = EXCLUDED.conversation_id,
    subject = EXCLUDED.subject,
    body_preview = EXCLUDED.body_preview,
    body_content = EXCLUDED.body_content,
    body_content_type = EXCLUDED.body_content_type,
    importance = EXCLUDED.importance,
    is_read = EXCLUDED.is_read,
    is_draft = EXCLUDED.is_draft,
    has_attachments = EXCLUDED.has_attachments,
    folder_id = EXCLUDED.folder_id,
    folder_name = EXCLUDED.folder_name,
    categories = EXCLUDED.categories,
    flag_status = EXCLUDED.flag_status,
    attachments = EXCLUDED.attachments,
    last_modified_datetime = EXCLUDED.last_modified_datetime,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftMailMessage :one
SELECT * FROM microsoft_mail_messages
WHERE user_id = $1 AND message_id = $2;

-- name: GetMicrosoftMailMessages :many
SELECT * FROM microsoft_mail_messages
WHERE user_id = $1
ORDER BY received_datetime DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: GetMicrosoftMailMessagesByFolder :many
SELECT * FROM microsoft_mail_messages
WHERE user_id = $1 AND folder_id = $2
ORDER BY received_datetime DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetMicrosoftMailConversation :many
SELECT * FROM microsoft_mail_messages
WHERE user_id = $1 AND conversation_id = $2
ORDER BY received_datetime ASC;

-- name: SearchMicrosoftMail :many
SELECT * FROM microsoft_mail_messages
WHERE user_id = $1
  AND (subject ILIKE $2 OR body_preview ILIKE $2 OR from_email ILIKE $2)
ORDER BY received_datetime DESC NULLS LAST
LIMIT $3;

-- name: GetUnreadMicrosoftMailCount :one
SELECT COUNT(*) FROM microsoft_mail_messages
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkMicrosoftMailAsRead :exec
UPDATE microsoft_mail_messages
SET is_read = TRUE, updated_at = NOW()
WHERE user_id = $1 AND message_id = $2;

-- name: DeleteMicrosoftMailMessage :exec
DELETE FROM microsoft_mail_messages
WHERE user_id = $1 AND message_id = $2;

-- name: DeleteMicrosoftMailByUser :exec
DELETE FROM microsoft_mail_messages WHERE user_id = $1;

-- ============================================================================
-- Microsoft Calendar Events
-- ============================================================================

-- name: UpsertMicrosoftCalendarEvent :one
INSERT INTO microsoft_calendar_events (
    user_id, event_id, calendar_id, calendar_name, subject, body_preview,
    body_content, body_content_type, location_display_name, location_address,
    location_coordinates, start_datetime, start_timezone, end_datetime, end_timezone,
    is_all_day, recurrence, series_master_id, type, attendees, organizer_email,
    organizer_name, is_online_meeting, online_meeting_provider, online_meeting_url,
    online_meeting_join_url, response_status, response_time, importance, sensitivity,
    show_as, is_cancelled, is_reminder_on, reminder_minutes_before_start, categories,
    created_datetime, last_modified_datetime
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37)
ON CONFLICT (user_id, event_id) DO UPDATE SET
    calendar_id = EXCLUDED.calendar_id,
    calendar_name = EXCLUDED.calendar_name,
    subject = EXCLUDED.subject,
    body_preview = EXCLUDED.body_preview,
    body_content = EXCLUDED.body_content,
    location_display_name = EXCLUDED.location_display_name,
    location_address = EXCLUDED.location_address,
    start_datetime = EXCLUDED.start_datetime,
    start_timezone = EXCLUDED.start_timezone,
    end_datetime = EXCLUDED.end_datetime,
    end_timezone = EXCLUDED.end_timezone,
    is_all_day = EXCLUDED.is_all_day,
    attendees = EXCLUDED.attendees,
    response_status = EXCLUDED.response_status,
    response_time = EXCLUDED.response_time,
    is_online_meeting = EXCLUDED.is_online_meeting,
    online_meeting_url = EXCLUDED.online_meeting_url,
    online_meeting_join_url = EXCLUDED.online_meeting_join_url,
    is_cancelled = EXCLUDED.is_cancelled,
    categories = EXCLUDED.categories,
    last_modified_datetime = EXCLUDED.last_modified_datetime,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftCalendarEvent :one
SELECT * FROM microsoft_calendar_events
WHERE user_id = $1 AND event_id = $2;

-- name: GetMicrosoftCalendarEvents :many
SELECT * FROM microsoft_calendar_events
WHERE user_id = $1
ORDER BY start_datetime ASC
LIMIT $2 OFFSET $3;

-- name: GetMicrosoftCalendarEventsByDateRange :many
SELECT * FROM microsoft_calendar_events
WHERE user_id = $1
  AND start_datetime >= $2
  AND end_datetime <= $3
ORDER BY start_datetime ASC;

-- name: GetMicrosoftCalendarEventsByCalendar :many
SELECT * FROM microsoft_calendar_events
WHERE user_id = $1 AND calendar_id = $2
ORDER BY start_datetime ASC
LIMIT $3 OFFSET $4;

-- name: GetUpcomingMicrosoftEvents :many
SELECT * FROM microsoft_calendar_events
WHERE user_id = $1
  AND start_datetime >= NOW()
  AND is_cancelled = FALSE
ORDER BY start_datetime ASC
LIMIT $2;

-- name: DeleteMicrosoftCalendarEvent :exec
DELETE FROM microsoft_calendar_events
WHERE user_id = $1 AND event_id = $2;

-- name: DeleteMicrosoftCalendarByUser :exec
DELETE FROM microsoft_calendar_events WHERE user_id = $1;

-- ============================================================================
-- Microsoft Contacts
-- ============================================================================

-- name: UpsertMicrosoftContact :one
INSERT INTO microsoft_contacts (
    user_id, contact_id, display_name, given_name, surname, middle_name, nickname,
    title, email_addresses, phone_numbers, addresses, im_addresses, websites,
    company_name, department, job_title, office_location, profession, manager,
    assistant_name, birthday, spouse_name, personal_notes, photo_url, categories,
    parent_folder_id, created_datetime, last_modified_datetime
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
ON CONFLICT (user_id, contact_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    given_name = EXCLUDED.given_name,
    surname = EXCLUDED.surname,
    middle_name = EXCLUDED.middle_name,
    nickname = EXCLUDED.nickname,
    title = EXCLUDED.title,
    email_addresses = EXCLUDED.email_addresses,
    phone_numbers = EXCLUDED.phone_numbers,
    addresses = EXCLUDED.addresses,
    im_addresses = EXCLUDED.im_addresses,
    websites = EXCLUDED.websites,
    company_name = EXCLUDED.company_name,
    department = EXCLUDED.department,
    job_title = EXCLUDED.job_title,
    office_location = EXCLUDED.office_location,
    manager = EXCLUDED.manager,
    birthday = EXCLUDED.birthday,
    spouse_name = EXCLUDED.spouse_name,
    personal_notes = EXCLUDED.personal_notes,
    photo_url = EXCLUDED.photo_url,
    categories = EXCLUDED.categories,
    last_modified_datetime = EXCLUDED.last_modified_datetime,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftContact :one
SELECT * FROM microsoft_contacts
WHERE user_id = $1 AND contact_id = $2;

-- name: GetMicrosoftContacts :many
SELECT * FROM microsoft_contacts
WHERE user_id = $1
ORDER BY display_name NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchMicrosoftContacts :many
SELECT * FROM microsoft_contacts
WHERE user_id = $1
  AND (display_name ILIKE $2 OR company_name ILIKE $2)
ORDER BY display_name NULLS LAST
LIMIT $3;

-- name: GetMicrosoftContactsByCompany :many
SELECT * FROM microsoft_contacts
WHERE user_id = $1 AND company_name = $2
ORDER BY display_name NULLS LAST;

-- name: GetMicrosoftContactsCount :one
SELECT COUNT(*) FROM microsoft_contacts WHERE user_id = $1;

-- name: DeleteMicrosoftContact :exec
DELETE FROM microsoft_contacts
WHERE user_id = $1 AND contact_id = $2;

-- name: DeleteMicrosoftContactsByUser :exec
DELETE FROM microsoft_contacts WHERE user_id = $1;

-- ============================================================================
-- Microsoft OneDrive Files
-- ============================================================================

-- name: UpsertMicrosoftOneDriveFile :one
INSERT INTO microsoft_onedrive_files (
    user_id, item_id, name, description, mime_type, size_bytes,
    parent_reference_id, parent_reference_path, web_url, is_folder,
    folder_child_count, file_hash, shared, shared_scope, shared_link,
    permissions, created_by_user_email, created_by_user_name,
    last_modified_by_user_email, last_modified_by_user_name,
    created_datetime, last_modified_datetime, download_url, thumbnails
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
ON CONFLICT (user_id, item_id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    mime_type = EXCLUDED.mime_type,
    size_bytes = EXCLUDED.size_bytes,
    parent_reference_id = EXCLUDED.parent_reference_id,
    parent_reference_path = EXCLUDED.parent_reference_path,
    web_url = EXCLUDED.web_url,
    is_folder = EXCLUDED.is_folder,
    folder_child_count = EXCLUDED.folder_child_count,
    file_hash = EXCLUDED.file_hash,
    shared = EXCLUDED.shared,
    shared_scope = EXCLUDED.shared_scope,
    shared_link = EXCLUDED.shared_link,
    permissions = EXCLUDED.permissions,
    last_modified_by_user_email = EXCLUDED.last_modified_by_user_email,
    last_modified_by_user_name = EXCLUDED.last_modified_by_user_name,
    last_modified_datetime = EXCLUDED.last_modified_datetime,
    download_url = EXCLUDED.download_url,
    thumbnails = EXCLUDED.thumbnails,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftOneDriveFile :one
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1 AND item_id = $2;

-- name: GetMicrosoftOneDriveFiles :many
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1
ORDER BY last_modified_datetime DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: GetMicrosoftOneDriveFilesByFolder :many
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1 AND parent_reference_id = $2
ORDER BY is_folder DESC, name ASC
LIMIT $3 OFFSET $4;

-- name: GetMicrosoftOneDriveFolders :many
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1 AND is_folder = TRUE
ORDER BY name ASC;

-- name: SearchMicrosoftOneDriveFiles :many
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1 AND name ILIKE $2
ORDER BY last_modified_datetime DESC NULLS LAST
LIMIT $3;

-- name: GetMicrosoftOneDriveRecentFiles :many
SELECT * FROM microsoft_onedrive_files
WHERE user_id = $1 AND is_folder = FALSE
ORDER BY last_modified_datetime DESC NULLS LAST
LIMIT $2;

-- name: GetMicrosoftOneDriveStorageUsed :one
SELECT COALESCE(SUM(size_bytes), 0)::BIGINT as total_bytes
FROM microsoft_onedrive_files
WHERE user_id = $1 AND is_folder = FALSE;

-- name: DeleteMicrosoftOneDriveFile :exec
DELETE FROM microsoft_onedrive_files
WHERE user_id = $1 AND item_id = $2;

-- name: DeleteMicrosoftOneDriveByUser :exec
DELETE FROM microsoft_onedrive_files WHERE user_id = $1;

-- ============================================================================
-- Microsoft To Do Lists
-- ============================================================================

-- name: UpsertMicrosoftToDoList :one
INSERT INTO microsoft_todo_lists (
    user_id, list_id, display_name, is_owner, is_shared, wellknown_list_name
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, list_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    is_owner = EXCLUDED.is_owner,
    is_shared = EXCLUDED.is_shared,
    wellknown_list_name = EXCLUDED.wellknown_list_name,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftToDoList :one
SELECT * FROM microsoft_todo_lists
WHERE user_id = $1 AND list_id = $2;

-- name: GetMicrosoftToDoLists :many
SELECT * FROM microsoft_todo_lists
WHERE user_id = $1
ORDER BY display_name ASC;

-- name: DeleteMicrosoftToDoList :exec
DELETE FROM microsoft_todo_lists
WHERE user_id = $1 AND list_id = $2;

-- name: DeleteMicrosoftToDoListsByUser :exec
DELETE FROM microsoft_todo_lists WHERE user_id = $1;

-- ============================================================================
-- Microsoft To Do Tasks
-- ============================================================================

-- name: UpsertMicrosoftToDoTask :one
INSERT INTO microsoft_todo_tasks (
    user_id, task_id, list_id, title, body_content, body_content_type,
    importance, status, due_datetime, due_timezone, start_datetime, start_timezone,
    completed_datetime, completed_timezone, recurrence, is_reminder_on,
    reminder_datetime, categories, linked_resources, checklist_items,
    created_datetime, last_modified_datetime
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
ON CONFLICT (user_id, task_id) DO UPDATE SET
    list_id = EXCLUDED.list_id,
    title = EXCLUDED.title,
    body_content = EXCLUDED.body_content,
    body_content_type = EXCLUDED.body_content_type,
    importance = EXCLUDED.importance,
    status = EXCLUDED.status,
    due_datetime = EXCLUDED.due_datetime,
    due_timezone = EXCLUDED.due_timezone,
    start_datetime = EXCLUDED.start_datetime,
    start_timezone = EXCLUDED.start_timezone,
    completed_datetime = EXCLUDED.completed_datetime,
    completed_timezone = EXCLUDED.completed_timezone,
    recurrence = EXCLUDED.recurrence,
    is_reminder_on = EXCLUDED.is_reminder_on,
    reminder_datetime = EXCLUDED.reminder_datetime,
    categories = EXCLUDED.categories,
    linked_resources = EXCLUDED.linked_resources,
    checklist_items = EXCLUDED.checklist_items,
    last_modified_datetime = EXCLUDED.last_modified_datetime,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetMicrosoftToDoTask :one
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: GetMicrosoftToDoTasks :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1
ORDER BY due_datetime ASC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: GetMicrosoftToDoTasksByList :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1 AND list_id = $2
ORDER BY due_datetime ASC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetMicrosoftToDoTasksByStatus :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1 AND status = $2
ORDER BY due_datetime ASC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetMicrosoftToDoIncompleteTasks :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1 AND status != 'completed'
ORDER BY due_datetime ASC NULLS LAST
LIMIT $2;

-- name: GetMicrosoftToDoOverdueTasks :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1
  AND status != 'completed'
  AND due_datetime < NOW()
ORDER BY due_datetime ASC;

-- name: GetMicrosoftToDoTasksDueToday :many
SELECT * FROM microsoft_todo_tasks
WHERE user_id = $1
  AND status != 'completed'
  AND due_datetime >= CURRENT_DATE
  AND due_datetime < CURRENT_DATE + INTERVAL '1 day'
ORDER BY due_datetime ASC;

-- name: MarkMicrosoftToDoTaskComplete :exec
UPDATE microsoft_todo_tasks
SET status = 'completed',
    completed_datetime = NOW(),
    updated_at = NOW()
WHERE user_id = $1 AND task_id = $2;

-- name: DeleteMicrosoftToDoTask :exec
DELETE FROM microsoft_todo_tasks
WHERE user_id = $1 AND task_id = $2;

-- name: DeleteMicrosoftToDoTasksByList :exec
DELETE FROM microsoft_todo_tasks
WHERE user_id = $1 AND list_id = $2;

-- name: DeleteMicrosoftToDoTasksByUser :exec
DELETE FROM microsoft_todo_tasks WHERE user_id = $1;
