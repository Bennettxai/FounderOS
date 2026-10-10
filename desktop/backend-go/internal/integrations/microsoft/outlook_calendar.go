package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// GetEvents retrieves calendar events for a user.
func (s *OutlookService) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]*OutlookEvent, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, event_id, calendar_id, subject, body_preview,
			location_display_name, start_datetime, start_timezone, end_datetime, end_timezone, is_all_day,
			organizer_email, organizer_name, is_online_meeting, online_meeting_url, response_status,
			importance, show_as, is_cancelled, is_reminder_on, reminder_minutes_before_start,
			categories, created_datetime, last_modified_datetime, synced_at
		FROM microsoft_calendar_events
		WHERE user_id = $1 AND start_datetime >= $2 AND start_datetime < $3
		ORDER BY start_datetime
	`, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*OutlookEvent
	for rows.Next() {
		var e OutlookEvent
		var calendarID, bodyPreview, locationDisplayName, startTZ, endTZ *string
		var organizerEmail, organizerName, onlineMeetingURL, responseStatus, importance, showAs *string
		var createdDateTime, lastModifiedDateTime *time.Time
		var categories []string

		err := rows.Scan(
			&e.ID, &e.UserID, &e.EventID, &calendarID, &e.Subject, &bodyPreview,
			&locationDisplayName, &e.StartDateTime, &startTZ, &e.EndDateTime, &endTZ, &e.IsAllDay,
			&organizerEmail, &organizerName, &e.IsOnlineMeeting, &onlineMeetingURL, &responseStatus,
			&importance, &showAs, &e.IsCancelled, &e.IsReminderOn, &e.ReminderMinutesBefore,
			&categories, &createdDateTime, &lastModifiedDateTime, &e.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		if calendarID != nil {
			e.CalendarID = *calendarID
		}
		if bodyPreview != nil {
			e.BodyPreview = *bodyPreview
		}
		if locationDisplayName != nil {
			e.LocationDisplayName = *locationDisplayName
		}
		if startTZ != nil {
			e.StartTimeZone = *startTZ
		}
		if endTZ != nil {
			e.EndTimeZone = *endTZ
		}
		if organizerEmail != nil {
			e.OrganizerEmail = *organizerEmail
		}
		if organizerName != nil {
			e.OrganizerName = *organizerName
		}
		if onlineMeetingURL != nil {
			e.OnlineMeetingURL = *onlineMeetingURL
		}
		if responseStatus != nil {
			e.ResponseStatus = *responseStatus
		}
		if importance != nil {
			e.Importance = *importance
		}
		if showAs != nil {
			e.ShowAs = *showAs
		}
		e.Categories = categories
		if createdDateTime != nil {
			e.CreatedDateTime = *createdDateTime
		}
		if lastModifiedDateTime != nil {
			e.LastModifiedDateTime = *lastModifiedDateTime
		}

		events = append(events, &e)
	}

	return events, nil
}

// CreateEvent creates a new calendar event.
func (s *OutlookService) CreateEvent(ctx context.Context, userID string, event *OutlookEvent) (*OutlookEvent, error) {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	eventData := map[string]interface{}{
		"subject": event.Subject,
		"start": map[string]string{
			"dateTime": event.StartDateTime.Format("2006-01-02T15:04:05"),
			"timeZone": event.StartTimeZone,
		},
		"end": map[string]string{
			"dateTime": event.EndDateTime.Format("2006-01-02T15:04:05"),
			"timeZone": event.EndTimeZone,
		},
		"isAllDay": event.IsAllDay,
	}

	if event.LocationDisplayName != "" {
		eventData["location"] = map[string]string{
			"displayName": event.LocationDisplayName,
		}
	}

	if event.BodyContent != "" {
		eventData["body"] = map[string]string{
			"contentType": "text",
			"content":     event.BodyContent,
		}
	}

	jsonBody, _ := json.Marshal(eventData)

	resp, err := client.Post(GraphAPIBase+"/me/events", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create event: %s", resp.Status)
	}

	var created graphEvent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Save to database
	if err := s.saveEvent(ctx, userID, &created); err != nil {
		slog.Info("Failed to save created event to database", "error", err)
	}

	event.EventID = created.ID
	return event, nil
}

// IsMailConnected checks if Outlook Mail is connected for a user.
func (s *OutlookService) IsMailConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM microsoft_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsMailScope(scope) {
			return true
		}
	}
	return false
}

// IsCalendarConnected checks if Outlook Calendar is connected for a user.
func (s *OutlookService) IsCalendarConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM microsoft_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsCalendarScope(scope) {
			return true
		}
	}
	return false
}

func containsMailScope(scope string) bool {
	mailScopes := []string{"Mail.Read", "Mail.ReadWrite", "Mail.Send"}
	for _, s := range mailScopes {
		if scope == s {
			return true
		}
	}
	return false
}

func containsCalendarScope(scope string) bool {
	calendarScopes := []string{"Calendars.Read", "Calendars.ReadWrite"}
	for _, s := range calendarScopes {
		if scope == s {
			return true
		}
	}
	return false
}
