-- name: ListCalendarEvents :many
-- Get events that overlap with the date range (start_time < range_end AND end_time > range_start)
-- Params: $1=user_id, $2=range_end (EndTime), $3=range_start (StartTime)
SELECT * FROM calendar_events
WHERE user_id = $1
  AND start_time < $2
  AND end_time > $3
ORDER BY start_time ASC;

-- name: ListCalendarEventsByType :many
SELECT * FROM calendar_events
WHERE user_id = $1
  AND meeting_type = $2
  AND start_time >= $3
  AND end_time <= $4
ORDER BY start_time ASC;

-- name: GetCalendarEvent :one
SELECT * FROM calendar_events
WHERE id = $1 AND user_id = $2;

-- name: GetCalendarEventByGoogleId :one
SELECT * FROM calendar_events
WHERE google_event_id = $1 AND user_id = $2;

-- name: CreateCalendarEvent :one
INSERT INTO calendar_events (
    user_id, google_event_id, calendar_id, title, description,
    start_time, end_time, all_day, location, attendees,
    status, visibility, html_link, source,
    meeting_type, context_id, project_id, client_id,
    recording_url, meeting_link, external_links,
    meeting_notes, action_items
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19, $20, $21, $22, $23
) RETURNING *;

-- name: UpdateCalendarEvent :one
UPDATE calendar_events
SET title = $3,
    description = $4,
    start_time = $5,
    end_time = $6,
    all_day = $7,
    location = $8,
    attendees = $9,
    status = $10,
    visibility = $11,
    html_link = $12,
    meeting_type = $13,
    context_id = $14,
    project_id = $15,
    client_id = $16,
    recording_url = $17,
    meeting_link = $18,
    external_links = $19,
    meeting_notes = $20,
    action_items = $21,
    synced_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpsertCalendarEvent :one
INSERT INTO calendar_events (
    user_id, google_event_id, calendar_id, title, description,
    start_time, end_time, all_day, location, attendees,
    status, visibility, html_link, source, meeting_link, external_links
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (user_id, google_event_id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    all_day = EXCLUDED.all_day,
    location = EXCLUDED.location,
    attendees = EXCLUDED.attendees,
    status = EXCLUDED.status,
    visibility = EXCLUDED.visibility,
    html_link = EXCLUDED.html_link,
    meeting_link = EXCLUDED.meeting_link,
    external_links = EXCLUDED.external_links,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: UpdateMeetingDetails :one
UPDATE calendar_events
SET meeting_type = $3,
    context_id = $4,
    project_id = $5,
    client_id = $6,
    recording_url = $7,
    meeting_link = $8,
    external_links = $9,
    meeting_notes = $10,
    action_items = $11,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteCalendarEvent :exec
DELETE FROM calendar_events
WHERE id = $1 AND user_id = $2;

-- name: DeleteCalendarEventByGoogleId :exec
DELETE FROM calendar_events
WHERE google_event_id = $1 AND user_id = $2;

-- name: DeleteCalendarEventsForUser :exec
DELETE FROM calendar_events
WHERE user_id = $1;

-- name: GetTodayEvents :many
SELECT * FROM calendar_events
WHERE user_id = $1
  AND DATE(start_time) = CURRENT_DATE
ORDER BY start_time ASC;

-- name: GetUpcomingEvents :many
SELECT * FROM calendar_events
WHERE user_id = $1
  AND start_time > NOW()
ORDER BY start_time ASC
LIMIT $2;

-- name: GetEventsForContext :many
SELECT * FROM calendar_events
WHERE context_id = $1
ORDER BY start_time DESC;

-- name: GetEventsForProject :many
SELECT * FROM calendar_events
WHERE project_id = $1
ORDER BY start_time DESC;

-- name: GetEventsForClient :many
SELECT * FROM calendar_events
WHERE client_id = $1
ORDER BY start_time DESC;

-- name: GetTeamAvailability :many
SELECT
    tm.id as member_id,
    tm.name as member_name,
    ce.start_time,
    ce.end_time,
    ce.title
FROM team_members tm
LEFT JOIN calendar_events ce ON tm.calendar_user_id = ce.user_id
    AND ce.start_time >= $2
    AND ce.end_time <= $3
WHERE tm.user_id = $1
  AND tm.share_calendar = TRUE
ORDER BY tm.name, ce.start_time;

-- name: GetMeetingsByType :many
SELECT meeting_type, COUNT(*) as count
FROM calendar_events
WHERE user_id = $1
  AND start_time >= $2
  AND end_time <= $3
GROUP BY meeting_type
ORDER BY count DESC;

-- name: SearchMeetings :many
SELECT * FROM calendar_events
WHERE user_id = $1
  AND (
    title ILIKE '%' || $2 || '%'
    OR description ILIKE '%' || $2 || '%'
    OR meeting_notes ILIKE '%' || $2 || '%'
  )
ORDER BY start_time DESC
LIMIT $3;

-- name: GetCalendarStats :one
SELECT
    COUNT(*) as total_events,
    COUNT(CASE WHEN source = 'google' THEN 1 END) as google_events,
    COUNT(CASE WHEN source = 'local' OR source IS NULL THEN 1 END) as local_events,
    MIN(start_time) as date_range_from,
    MAX(end_time) as date_range_to,
    MAX(synced_at) as last_sync
FROM calendar_events
WHERE user_id = $1;
