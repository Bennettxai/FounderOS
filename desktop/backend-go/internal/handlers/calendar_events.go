package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rhl/businessos-backend/internal/middleware"
	googlecalendar "google.golang.org/api/calendar/v3"
)

// Native calendar event CRUD over the calendar_events table.
//
// This makes the calendar a real primitive that works WITHOUT any external
// connection: a user can create/list/edit/delete events directly. Google and
// Outlook sync into the SAME table (source column), so synced events and native
// events appear together. Events are personal by default (workspace_id NULL) and
// become shared when created with an X-Workspace-ID header for a workspace the
// user belongs to - the same model as docs and relations.

// calendarEventCols is the canonical SELECT column list (meeting_type cast to
// text so it scans into a string).
const calendarEventCols = `
	id, user_id, google_event_id, calendar_id, title, description,
	start_time, end_time, all_day, location, attendees, status, visibility,
	html_link, source, meeting_type::text, context_id, project_id, client_id,
	(SELECT color_id FROM calendar_event_colors WHERE event_id = calendar_events.id),
	(SELECT color_hex FROM calendar_event_colors WHERE event_id = calendar_events.id),
	recording_url, meeting_link, external_links, meeting_notes, action_items,
	synced_at, created_at, updated_at`

// workspaceScope resolves the X-Workspace-ID header and confirms active
// membership. Returns (workspaceID, true) for a workspace-shared event, or
// (uuid.Nil, false) for a personal event.
func (h *CalendarStatsHandler) workspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// calendarVisibilityPredicate keeps personal and workspace calendars separate
// by default. A workspace may explicitly opt into overlaying the current
// member's personal calendar without moving or duplicating those events.
func calendarVisibilityPredicate(scoped, includePersonal bool) string {
	if scoped && includePersonal {
		return "(workspace_id = $3 OR (user_id = $4 AND workspace_id IS NULL))"
	}
	if scoped {
		return "workspace_id = $3"
	}
	return "user_id = $3 AND workspace_id IS NULL"
}

func (h *CalendarStatsHandler) workspaceIncludesPersonalCalendar(c *gin.Context, workspaceID uuid.UUID) bool {
	if workspaceID == uuid.Nil {
		return false
	}
	var include bool
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT COALESCE(settings #> '{calendar,include_personal}', 'false'::jsonb) = 'true'::jsonb
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(&include)
	return err == nil && include
}

// ListEvents returns events overlapping [start,end] in exactly one scope:
// the active workspace, or the caller's personal calendar when unscoped.
// GET /api/calendar/events?start=<rfc3339>&end=<rfc3339>
func (h *CalendarStatsHandler) ListEvents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 3, 0)
	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}

	wsID, scoped := h.workspaceScope(c, user.ID)
	var visibilityArg interface{}
	if scoped {
		visibilityArg = wsID
	} else {
		visibilityArg = user.ID
	}

	includePersonal := scoped && h.workspaceIncludesPersonalCalendar(c, wsID)
	visibility := calendarVisibilityPredicate(scoped, includePersonal)
	query := `
		SELECT ` + calendarEventCols + `
		FROM   calendar_events
		WHERE  start_time < $1 AND end_time > $2
		AND    ` + visibility + `
		ORDER  BY start_time ASC
	`
	var rows pgx.Rows
	var err error
	if includePersonal {
		rows, err = h.pool.Query(c.Request.Context(), query, end, start, visibilityArg, user.ID)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), query, end, start, visibilityArg)
	}
	if err != nil {
		RespondInternalErr(c, "list calendar events", err)
		return
	}
	defer rows.Close()

	events := make([]CalendarEventResponse, 0)
	for rows.Next() {
		ev, err := scanCalendarEvent(rows)
		if err != nil {
			RespondInternalErr(c, "scan calendar event", err)
			return
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate calendar events", err)
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetEvent returns a single event the caller can see.
// GET /api/calendar/events/:id
func (h *CalendarStatsHandler) GetEvent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid event id")
		return
	}

	wsID, scoped := h.workspaceScope(c, user.ID)
	var wsArg interface{}
	if scoped {
		wsArg = wsID
	} else {
		wsArg = nil
	}

	var row scanRow
	if scoped {
		row = h.pool.QueryRow(c.Request.Context(), `
			SELECT `+calendarEventCols+`
			FROM   calendar_events
			WHERE  id = $1 AND workspace_id = $2
		`, id, wsArg)
	} else {
		row = h.pool.QueryRow(c.Request.Context(), `
			SELECT `+calendarEventCols+`
			FROM   calendar_events
			WHERE  id = $1 AND user_id = $2 AND workspace_id IS NULL
		`, id, user.ID)
	}
	ev, err := scanCalendarEvent(row)
	if err != nil {
		RespondNotFoundErr(c, "event")
		return
	}
	c.JSON(http.StatusOK, ev)
}

// calendarEventInput is the create/update payload (matches the frontend types).
type calendarEventInput struct {
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	StartTime   *string         `json:"start_time"`
	EndTime     *string         `json:"end_time"`
	AllDay      *bool           `json:"all_day"`
	Location    *string         `json:"location"`
	MeetingType *string         `json:"meeting_type"`
	ContextID   *string         `json:"context_id"`
	ProjectID   *string         `json:"project_id"`
	ClientID    *string         `json:"client_id"`
	MeetingLink *string         `json:"meeting_link"`
	Attendees   json.RawMessage `json:"attendees"`
}

// CreateEvent inserts a native event. It is shared with the active workspace
// when an X-Workspace-ID header is present, personal otherwise.
// POST /api/calendar/events
func (h *CalendarStatsHandler) CreateEvent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}

	var in calendarEventInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	if in.Title == nil || *in.Title == "" || in.StartTime == nil || in.EndTime == nil {
		RespondBadRequestErr(c, "title, start_time and end_time are required")
		return
	}
	startT, err := time.Parse(time.RFC3339, *in.StartTime)
	if err != nil {
		RespondBadRequestErr(c, "invalid start_time")
		return
	}
	endT, err := time.Parse(time.RFC3339, *in.EndTime)
	if err != nil {
		RespondBadRequestErr(c, "invalid end_time")
		return
	}

	meetingType := "other"
	if in.MeetingType != nil && *in.MeetingType != "" {
		meetingType = *in.MeetingType
	}
	allDay := false
	if in.AllDay != nil {
		allDay = *in.AllDay
	}

	wsID, scoped := h.workspaceScope(c, user.ID)
	var wsArg interface{}
	if scoped {
		wsArg = wsID
	} else {
		wsArg = nil
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO calendar_events
			(user_id, workspace_id, title, description, start_time, end_time,
			 all_day, location, meeting_type, source, context_id, project_id,
			 client_id, meeting_link, attendees)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::text::meetingtype,'businessos',
		        $10,$11,$12,$13,COALESCE($14,'[]'::jsonb))
		RETURNING `+calendarEventCols+`
	`, user.ID, wsArg, in.Title, in.Description, startT, endT,
		allDay, in.Location, meetingType, nullableUUID(in.ContextID),
		nullableUUID(in.ProjectID), nullableUUID(in.ClientID), in.MeetingLink,
		nullableRawJSON(in.Attendees))

	ev, err := scanCalendarEvent(row)
	if err != nil {
		RespondInternalErr(c, "create calendar event", err)
		return
	}
	ev = h.syncAppointmentToGoogle(c.Request.Context(), user.ID, ev)
	c.JSON(http.StatusCreated, ev)
}

// UpdateEvent edits an event the caller owns or shares via its workspace.
// PUT /api/calendar/events/:id
func (h *CalendarStatsHandler) UpdateEvent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid event id")
		return
	}
	if !h.canEditEvent(c, user.ID, id) {
		RespondNotFoundErr(c, "event")
		return
	}

	var in calendarEventInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var startArg, endArg interface{}
	if in.StartTime != nil {
		if t, err := time.Parse(time.RFC3339, *in.StartTime); err == nil {
			startArg = t
		}
	}
	if in.EndTime != nil {
		if t, err := time.Parse(time.RFC3339, *in.EndTime); err == nil {
			endArg = t
		}
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		UPDATE calendar_events SET
			title        = COALESCE($2, title),
			description  = COALESCE($3, description),
			start_time   = COALESCE($4, start_time),
			end_time     = COALESCE($5, end_time),
			all_day      = COALESCE($6, all_day),
			location     = COALESCE($7, location),
			meeting_link = COALESCE($8, meeting_link),
			updated_at   = NOW()
		WHERE id = $1
		RETURNING `+calendarEventCols+`
	`, id, in.Title, in.Description, startArg, endArg, in.AllDay, in.Location, in.MeetingLink)

	ev, err := scanCalendarEvent(row)
	if err != nil {
		RespondInternalErr(c, "update calendar event", err)
		return
	}
	ev = h.syncAppointmentToGoogle(c.Request.Context(), user.ID, ev)
	c.JSON(http.StatusOK, ev)
}

// DeleteEvent removes an event the caller owns or shares via its workspace.
// DELETE /api/calendar/events/:id
func (h *CalendarStatsHandler) DeleteEvent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid event id")
		return
	}
	if !h.canEditEvent(c, user.ID, id) {
		RespondNotFoundErr(c, "event")
		return
	}
	googleEventID := h.googleEventIDForEvent(c.Request.Context(), id)
	if _, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM calendar_events WHERE id = $1`, id); err != nil {
		RespondInternalErr(c, "delete calendar event", err)
		return
	}
	if googleEventID != "" {
		h.deleteGoogleAppointment(c.Request.Context(), user.ID, googleEventID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "event deleted"})
}

func (h *CalendarStatsHandler) syncAppointmentToGoogle(ctx context.Context, userID string, ev CalendarEventResponse) CalendarEventResponse {
	if h.googleCalendar == nil || !h.googleCalendar.IsConnected(ctx, userID) {
		return ev
	}

	googleEvent, err := calendarResponseToGoogleEvent(ev)
	if err != nil {
		slog.Info("calendar: skipped Google write-through", "event_id", ev.ID, "error", err)
		return ev
	}

	if ev.GoogleEventID != nil && *ev.GoogleEventID != "" {
		updated, err := h.googleCalendar.UpdateGoogleEvent(ctx, userID, *ev.GoogleEventID, googleEvent)
		if err != nil {
			slog.Info("calendar: failed to update Google event", "event_id", ev.ID, "google_event_id", *ev.GoogleEventID, "error", err)
			return ev
		}
		return h.persistGoogleMirror(ctx, ev, updated)
	}

	created, err := h.googleCalendar.CreateGoogleEvent(ctx, userID, googleEvent)
	if err != nil {
		slog.Info("calendar: failed to create Google event", "event_id", ev.ID, "error", err)
		return ev
	}
	return h.persistGoogleMirror(ctx, ev, created)
}

func calendarResponseToGoogleEvent(ev CalendarEventResponse) (*googlecalendar.Event, error) {
	start, err := time.Parse(time.RFC3339, ev.StartTime)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, ev.EndTime)
	if err != nil {
		return nil, err
	}

	googleEvent := &googlecalendar.Event{
		Summary:     derefString(ev.Title),
		Description: derefString(ev.Description),
		Location:    derefString(ev.Location),
		Start: &googlecalendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
		},
		End: &googlecalendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
		},
	}
	if ev.AllDay {
		googleEvent.Start = &googlecalendar.EventDateTime{Date: start.Format("2006-01-02")}
		googleEvent.End = &googlecalendar.EventDateTime{Date: end.Format("2006-01-02")}
	}
	return googleEvent, nil
}

func (h *CalendarStatsHandler) persistGoogleMirror(ctx context.Context, ev CalendarEventResponse, googleEvent *googlecalendar.Event) CalendarEventResponse {
	if googleEvent == nil || googleEvent.Id == "" {
		return ev
	}
	if _, err := h.pool.Exec(ctx, `
		UPDATE calendar_events
		SET google_event_id = $2,
		    html_link = NULLIF($3, ''),
		    synced_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, ev.ID, googleEvent.Id, googleEvent.HtmlLink); err != nil {
		slog.Info("calendar: failed to persist Google event mirror", "event_id", ev.ID, "google_event_id", googleEvent.Id, "error", err)
		return ev
	}
	ev.GoogleEventID = &googleEvent.Id
	if googleEvent.HtmlLink != "" {
		ev.HtmlLink = &googleEvent.HtmlLink
	}
	ev.SyncedAt = time.Now().Format(time.RFC3339)
	return ev
}

func (h *CalendarStatsHandler) googleEventIDForEvent(ctx context.Context, eventID uuid.UUID) string {
	var googleEventID *string
	_ = h.pool.QueryRow(ctx, `SELECT google_event_id FROM calendar_events WHERE id = $1`, eventID).Scan(&googleEventID)
	if googleEventID == nil {
		return ""
	}
	return *googleEventID
}

func (h *CalendarStatsHandler) deleteGoogleAppointment(ctx context.Context, userID string, googleEventID string) {
	if h.googleCalendar == nil || !h.googleCalendar.IsConnected(ctx, userID) {
		return
	}
	srv, err := h.googleCalendar.GetCalendarAPI(ctx, userID)
	if err != nil {
		slog.Info("calendar: failed to get Google API for delete", "google_event_id", googleEventID, "error", err)
		return
	}
	if err := srv.Events.Delete("primary", googleEventID).Do(); err != nil {
		slog.Info("calendar: failed to delete Google event", "google_event_id", googleEventID, "error", err)
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// canEditEvent authorizes against the active calendar scope. A personal event
// cannot be edited while operating inside a workspace, and workspace events
// cannot be edited from the personal calendar.
func (h *CalendarStatsHandler) canEditEvent(c *gin.Context, userID string, eventID uuid.UUID) bool {
	var (
		ownerID string
		wsID    *uuid.UUID
	)
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT user_id, workspace_id FROM calendar_events WHERE id = $1`, eventID,
	).Scan(&ownerID, &wsID)
	if err != nil {
		return false
	}
	activeWorkspaceID, scoped := h.workspaceScope(c, userID)
	if scoped {
		return wsID != nil && *wsID == activeWorkspaceID
	}
	return wsID == nil && ownerID == userID
}

// scanCalendarEvent maps a calendar_events row into the canonical response shape
// (identical to TransformCalendarEvent so the frontend sees one consistent type).
func scanCalendarEvent(row scanRow) (CalendarEventResponse, error) {
	var (
		r            CalendarEventResponse
		id           uuid.UUID
		startT, endT time.Time
		allDay       *bool
		ctxID        *uuid.UUID
		projID       *uuid.UUID
		cliID        *uuid.UUID
		meetingType  *string
		syncedAt     *time.Time
		createdAt    time.Time
		updatedAt    time.Time
		attendees    []byte
		extLinks     []byte
		actionItems  []byte
	)
	err := row.Scan(
		&id, &r.UserID, &r.GoogleEventID, &r.CalendarID, &r.Title, &r.Description,
		&startT, &endT, &allDay, &r.Location, &attendees, &r.Status, &r.Visibility,
		&r.HtmlLink, &r.Source, &meetingType, &ctxID, &projID, &cliID,
		&r.ColorID, &r.ColorHex,
		&r.RecordingURL, &r.MeetingLink, &extLinks, &r.MeetingNotes, &actionItems,
		&syncedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return r, err
	}

	r.ID = id.String()
	r.StartTime = startT.Format(time.RFC3339)
	r.EndTime = endT.Format(time.RFC3339)
	if allDay != nil {
		r.AllDay = *allDay
	}
	r.MeetingType = "other"
	if meetingType != nil && *meetingType != "" {
		r.MeetingType = *meetingType
	}
	r.ContextID = uuidPtrToStrPtr(ctxID)
	r.ProjectID = uuidPtrToStrPtr(projID)
	r.ClientID = uuidPtrToStrPtr(cliID)
	if syncedAt != nil {
		r.SyncedAt = syncedAt.Format(time.RFC3339)
	}
	r.CreatedAt = createdAt.Format(time.RFC3339)
	r.UpdatedAt = updatedAt.Format(time.RFC3339)

	r.Attendees = []map[string]any{}
	if len(attendees) > 0 {
		_ = json.Unmarshal(attendees, &r.Attendees)
		if r.Attendees == nil {
			r.Attendees = []map[string]any{}
		}
	}
	r.ExternalLinks = []string{}
	if len(extLinks) > 0 {
		_ = json.Unmarshal(extLinks, &r.ExternalLinks)
		if r.ExternalLinks == nil {
			r.ExternalLinks = []string{}
		}
	}
	r.ActionItems = []string{}
	if len(actionItems) > 0 {
		_ = json.Unmarshal(actionItems, &r.ActionItems)
		if r.ActionItems == nil {
			r.ActionItems = []string{}
		}
	}
	return r, nil
}

// nullableUUID parses an optional UUID string, returning nil for empty/invalid
// so the column stays NULL instead of erroring.
func nullableUUID(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return id
}

// nullableRawJSON returns nil for empty JSON so COALESCE applies the default.
func nullableRawJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return []byte(raw)
}

// uuidPtrToStrPtr converts an optional UUID to an optional string.
func uuidPtrToStrPtr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}
