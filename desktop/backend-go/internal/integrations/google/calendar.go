package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	GoogleID    string     `json:"google_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Location    string     `json:"location,omitempty"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	AllDay      bool       `json:"all_day"`
	Status      string     `json:"status"`
	MeetingLink string     `json:"meeting_link,omitempty"`
	MeetingType string     `json:"meeting_type,omitempty"`
	Attendees   []Attendee `json:"attendees,omitempty"`
	Recurrence  string     `json:"recurrence,omitempty"`
	ColorID     string     `json:"color_id,omitempty"`
	ColorHex    string     `json:"color_hex,omitempty"`
	Source      string     `json:"source"` // "local" or "google"
	CalendarID  string     `json:"calendar_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Attendee represents an event attendee.
type Attendee struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"` // "accepted", "declined", "tentative", "needsAction"
	Optional bool   `json:"optional,omitempty"`
}

// CalendarService handles Google Calendar operations.
type CalendarService struct {
	provider googleServiceProvider
}

// NewCalendarService creates a new calendar service.
func NewCalendarService(provider googleServiceProvider) *CalendarService {
	return &CalendarService{provider: provider}
}

// GetCalendarAPI returns a Google Calendar API service for a user.
func (s *CalendarService) GetCalendarAPI(ctx context.Context, userID string) (*calendar.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := calendar.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return srv, nil
}

// SyncEvents syncs calendar events from Google Calendar.
func (s *CalendarService) SyncEvents(ctx context.Context, userID string, timeMin, timeMax time.Time) (*SyncEventsResult, error) {
	slog.Info("Calendar sync starting for user", "user_id", userID)

	srv, err := s.GetCalendarAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar API: %w", err)
	}

	result := &SyncEventsResult{}
	const maxEvents = 5000 // safety cap so a runaway calendar can't loop forever

	calendars, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}
	type selectedCalendar struct {
		id       string
		colorHex string
	}
	selectedCalendars := make([]selectedCalendar, 0, len(calendars.Items))
	for _, cal := range calendars.Items {
		if cal == nil || cal.Id == "" || cal.Deleted || cal.Hidden {
			continue
		}
		// Google Calendar marks visible/checked calendars as selected. Include
		// primary even if the selected flag is absent.
		if cal.Primary || cal.Selected {
			selectedCalendars = append(selectedCalendars, selectedCalendar{
				id:       cal.Id,
				colorHex: cal.BackgroundColor,
			})
		}
	}
	if len(selectedCalendars) == 0 {
		selectedCalendars = append(selectedCalendars, selectedCalendar{id: "primary"})
	}

	eventColors := map[string]string{}
	if colors, colorErr := srv.Colors.Get().Do(); colorErr == nil {
		for id, definition := range colors.Event {
			if definition.Background != "" {
				eventColors[id] = definition.Background
			}
		}
	} else {
		slog.Info("Failed to load Google Calendar colors", "error", colorErr)
	}

	// Paginate through every selected calendar. A single 250-result page ordered
	// by startTime can be consumed by dense recurring events, so future events
	// would never sync without paging.
	for _, selected := range selectedCalendars {
		calendarID := selected.id
		pageToken := ""
		for {
			call := srv.Events.List(calendarID).
				TimeMin(timeMin.Format(time.RFC3339)).
				TimeMax(timeMax.Format(time.RFC3339)).
				SingleEvents(true).
				OrderBy("startTime").
				MaxResults(250)
			if pageToken != "" {
				call = call.PageToken(pageToken)
			}
			events, err := call.Do()
			if err != nil {
				slog.Info("Failed to list calendar events", "calendar_id", calendarID, "error", err)
				break
			}

			result.TotalEvents += len(events.Items)
			for _, event := range events.Items {
				colorHex := selected.colorHex
				if eventColor, ok := eventColors[event.ColorId]; ok {
					colorHex = eventColor
				}
				if err := s.saveEvent(ctx, userID, calendarID, colorHex, event); err != nil {
					slog.Info("Failed to save event", "calendar_id", calendarID, "id", event.Id, "error", err)
					result.FailedEvents++
				} else {
					result.SyncedEvents++
				}
			}

			if events.NextPageToken == "" || result.TotalEvents >= maxEvents {
				break
			}
			pageToken = events.NextPageToken
		}
		if result.TotalEvents >= maxEvents {
			break
		}
	}

	slog.Info("Calendar sync complete for user : synced / events", "user_id", userID, "synced", result.SyncedEvents, "total", result.TotalEvents)

	return result, nil
}

// SyncEventsResult represents the result of a sync operation.
type SyncEventsResult struct {
	TotalEvents  int `json:"total_events"`
	SyncedEvents int `json:"synced_events"`
	FailedEvents int `json:"failed_events"`
}

// saveEvent saves a Google Calendar event to the database.
func (s *CalendarService) saveEvent(ctx context.Context, userID string, calendarID string, colorHex string, event *calendar.Event) error {
	// Parse start and end times
	var startTime, endTime time.Time
	var allDay bool

	if event.Start.DateTime != "" {
		startTime, _ = time.Parse(time.RFC3339, event.Start.DateTime)
		endTime, _ = time.Parse(time.RFC3339, event.End.DateTime)
	} else {
		// All-day event
		startTime, _ = time.Parse("2006-01-02", event.Start.Date)
		endTime, _ = time.Parse("2006-01-02", event.End.Date)
		allDay = true
	}

	// Extract the video meeting link (Google Meet / Zoom entry point), if any.
	// Do not map the conference name into meeting_type. That column is a
	// Postgres enum and the conference name is not a valid value.
	meetingLink := ""
	if event.ConferenceData != nil && len(event.ConferenceData.EntryPoints) > 0 {
		for _, ep := range event.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				meetingLink = ep.Uri
				break
			}
		}
	}

	// Get status
	status := event.Status
	if status == "" {
		status = "confirmed"
	}

	// Keep the provider color in a sidecar table so the stable calendar_events
	// shape remains compatible with generated SELECT * queries.
	tx, err := s.provider.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var calendarEventID string
	err = tx.QueryRow(ctx, `
		INSERT INTO calendar_events (
			user_id, google_event_id, calendar_id, title, description, location,
			start_time, end_time, all_day, status,
			meeting_link, source, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'google', NOW())
		ON CONFLICT (user_id, google_event_id) DO UPDATE SET
			calendar_id = EXCLUDED.calendar_id,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			location = EXCLUDED.location,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			all_day = EXCLUDED.all_day,
			status = EXCLUDED.status,
			meeting_link = EXCLUDED.meeting_link,
			synced_at = NOW(),
			updated_at = NOW()
		RETURNING id::text
	`, userID, event.Id, calendarID, event.Summary, event.Description, event.Location,
		startTime, endTime, allDay, status, meetingLink).Scan(&calendarEventID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO calendar_event_colors (event_id, color_id, color_hex, updated_at)
		VALUES ($1::uuid, NULLIF($2, ''), NULLIF($3, ''), NOW())
		ON CONFLICT (event_id) DO UPDATE SET
			color_id = EXCLUDED.color_id,
			color_hex = EXCLUDED.color_hex,
			updated_at = NOW()
	`, calendarEventID, event.ColorId, colorHex)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetEvents retrieves calendar events for a user.
func (s *CalendarService) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]*CalendarEvent, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, google_event_id, title, description, location,
			start_time, end_time, all_day, status,
			meeting_link, meeting_type,
			(SELECT color_id FROM calendar_event_colors WHERE event_id = calendar_events.id),
			(SELECT color_hex FROM calendar_event_colors WHERE event_id = calendar_events.id),
			source,
			created_at, updated_at
		FROM calendar_events
		WHERE user_id = $1 AND start_time >= $2 AND start_time < $3
		ORDER BY start_time
	`, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		var googleID, description, location, meetingLink, meetingType, colorID, colorHex pgtype.Text

		err := rows.Scan(
			&e.ID, &e.UserID, &googleID, &e.Title, &description, &location,
			&e.StartTime, &e.EndTime, &e.AllDay, &e.Status,
			&meetingLink, &meetingType, &colorID, &colorHex, &e.Source,
			&e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		e.GoogleID = googleID.String
		e.Description = description.String
		e.Location = location.String
		e.MeetingLink = meetingLink.String
		e.MeetingType = meetingType.String
		e.ColorID = colorID.String
		e.ColorHex = colorHex.String

		events = append(events, &e)
	}

	return events, nil
}

// CreateEvent creates a new calendar event.
func (s *CalendarService) CreateEvent(ctx context.Context, userID string, event *CalendarEvent) (*CalendarEvent, error) {
	// Insert into database
	var id string
	err := s.provider.Pool().QueryRow(ctx, `
		INSERT INTO calendar_events (
			user_id, title, description, location,
			start_time, end_time, all_day, status,
			meeting_link, meeting_type, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'local')
		RETURNING id
	`, userID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.AllDay, "confirmed",
		event.MeetingLink, event.MeetingType).Scan(&id)

	if err != nil {
		return nil, err
	}

	event.ID = id
	event.UserID = userID
	event.Source = "local"
	event.Status = "confirmed"

	// Optionally push to Google Calendar
	if event.GoogleID == "" {
		go s.pushEventToGoogle(context.Background(), userID, event)
	}

	return event, nil
}

// pushEventToGoogle pushes a local event to Google Calendar.
func (s *CalendarService) pushEventToGoogle(ctx context.Context, userID string, event *CalendarEvent) {
	srv, err := s.GetCalendarAPI(ctx, userID)
	if err != nil {
		slog.Info("Failed to get calendar API for push", "error", err)
		return
	}

	googleEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime.Format(time.RFC3339),
		},
	}

	if event.AllDay {
		googleEvent.Start = &calendar.EventDateTime{Date: event.StartTime.Format("2006-01-02")}
		googleEvent.End = &calendar.EventDateTime{Date: event.EndTime.Format("2006-01-02")}
	}

	created, err := srv.Events.Insert("primary", googleEvent).Do()
	if err != nil {
		slog.Info("Failed to push event to Google", "error", err)
		return
	}

	// Update local event with Google ID
	s.provider.Pool().Exec(ctx, `
		UPDATE calendar_events SET google_event_id = $1, updated_at = NOW()
		WHERE id = $2
	`, created.Id, event.ID)
}

// DeleteEvent deletes a calendar event.
func (s *CalendarService) DeleteEvent(ctx context.Context, userID, eventID string) error {
	// Get the event to check for Google ID
	var googleID pgtype.Text
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT google_event_id FROM calendar_events WHERE id = $1 AND user_id = $2
	`, eventID, userID).Scan(&googleID)

	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	// Delete from database
	_, err = s.provider.Pool().Exec(ctx, `
		DELETE FROM calendar_events WHERE id = $1 AND user_id = $2
	`, eventID, userID)
	if err != nil {
		return err
	}

	// Delete from Google if it was synced
	if googleID.Valid && googleID.String != "" {
		go func() {
			srv, err := s.GetCalendarAPI(context.Background(), userID)
			if err != nil {
				slog.Info("Failed to get calendar API for delete", "error", err)
				return
			}
			srv.Events.Delete("primary", googleID.String).Do()
		}()
	}

	return nil
}

// IsConnected checks if the user has Google Calendar connected.
func (s *CalendarService) IsConnected(ctx context.Context, userID string) bool {
	var count int
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&count)
	return err == nil && count > 0
}

// ============================================
// MCP-Compatible Methods (Google Native Types)
// These methods work with Google's native calendar.Event type
// for advanced features like attendees, recurrence, and Meet links.
// ============================================

// FetchEvents returns events from Google Calendar (native Google types).
// Used by MCP tools that need full Google Calendar features.
func (s *CalendarService) FetchEvents(ctx context.Context, userID string, start, end time.Time) ([]*calendar.Event, error) {
	srv, err := s.GetCalendarAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar API: %w", err)
	}

	events, err := srv.Events.List("primary").
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(100).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events.Items, nil
}

// CreateGoogleEvent creates an event using Google's native event type.
// Supports all Google Calendar features: attendees, Meet links, recurrence.
func (s *CalendarService) CreateGoogleEvent(ctx context.Context, userID string, event *calendar.Event) (*calendar.Event, error) {
	srv, err := s.GetCalendarAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar API: %w", err)
	}

	// If event has conference data request, add conferenceDataVersion
	call := srv.Events.Insert("primary", event)
	if event.ConferenceData != nil && event.ConferenceData.CreateRequest != nil {
		call = call.ConferenceDataVersion(1)
	}

	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return created, nil
}

// UpdateGoogleEvent updates an event using Google's native event type.
func (s *CalendarService) UpdateGoogleEvent(ctx context.Context, userID, eventID string, event *calendar.Event) (*calendar.Event, error) {
	srv, err := s.GetCalendarAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar API: %w", err)
	}

	call := srv.Events.Update("primary", eventID, event)
	if event.ConferenceData != nil && event.ConferenceData.CreateRequest != nil {
		call = call.ConferenceDataVersion(1)
	}

	updated, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return updated, nil
}
