package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

// ErrCalendarTokenExpired indicates that the Google Calendar token has expired
var ErrCalendarTokenExpired = errors.New("calendar token expired")

func GetCalendarTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "calendar_list_events",
			Description: "List events from Google Calendar within a date range. Use this to check what meetings or events the user has scheduled.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (e.g., '2024-12-21'). Defaults to today if not provided.",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (e.g., '2024-12-28'). Defaults to 7 days from start if not provided.",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of events to return (default 10, max 50)",
					},
				},
			},
			Source: "builtin",
		},
		{
			Name:        "calendar_create_event",
			Description: "Create a new event in Google Calendar. Supports single events, recurring events, attendees (sends invites), and Google Meet video conferencing.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Event title/summary (required)",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "Start time in ISO 8601 format: 'YYYY-MM-DDTHH:MM:SS' (e.g., '2024-12-21T14:00:00'). For all-day events, use 'YYYY-MM-DD'.",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "End time in ISO 8601 format: 'YYYY-MM-DDTHH:MM:SS' (e.g., '2024-12-21T15:00:00'). For all-day events, use 'YYYY-MM-DD'.",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Event description/notes (optional)",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Event location - physical address or room name (optional)",
					},
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "Timezone for the event (e.g., 'America/New_York', 'Europe/London'). Defaults to user's calendar timezone.",
					},
					"attendees": map[string]interface{}{
						"type":        "array",
						"description": "List of attendee email addresses. Google will send calendar invitations to these people.",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"add_meet_link": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, automatically creates a Google Meet video conference link for the event.",
					},
					"recurrence": map[string]interface{}{
						"type":        "string",
						"description": "Recurrence rule in RRULE format (e.g., 'RRULE:FREQ=WEEKLY;COUNT=10' for 10 weekly occurrences). Common patterns: 'daily', 'weekly', 'monthly', 'weekdays' are also accepted as shortcuts.",
						"enum":        []string{"daily", "weekly", "monthly", "weekdays", "RRULE:..."},
					},
					"recurrence_count": map[string]interface{}{
						"type":        "integer",
						"description": "Number of times the event should repeat (used with recurrence shortcuts like 'weekly'). Default is 10.",
					},
				},
				"required": []string{"title", "start_time", "end_time"},
			},
			Source: "builtin",
		},
		{
			Name:        "calendar_update_event",
			Description: "Update an existing event in Google Calendar. Only provide the fields you want to change.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "The Google Calendar event ID (required). Get this from calendar_list_events.",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "New event title/summary",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "New start time in ISO 8601 format",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "New end time in ISO 8601 format",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New event description",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "New event location",
					},
					"attendees": map[string]interface{}{
						"type":        "array",
						"description": "Updated list of attendee email addresses (replaces existing attendees)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"add_meet_link": map[string]interface{}{
						"type":        "boolean",
						"description": "If true and event doesn't have one, adds a Google Meet link",
					},
				},
				"required": []string{"event_id"},
			},
			Source: "builtin",
		},
		{
			Name:        "calendar_delete_event",
			Description: "Delete an event from Google Calendar. This action cannot be undone.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "The Google Calendar event ID to delete (required). Get this from calendar_list_events.",
					},
					"notify_attendees": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, sends cancellation emails to all attendees. Default is true.",
					},
				},
				"required": []string{"event_id"},
			},
			Source: "builtin",
		},
	}
}

func (m *MCPService) ExecuteCalendarTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "calendar_list_events":
		if m.calendarService == nil {
			return nil, fmt.Errorf("Google Calendar is not configured. Please connect your Google account first.")
		}

		startDate, endDate := parseCalendarDateRange(arguments)

		maxResults := 10
		if mr, ok := arguments["max_results"].(float64); ok {
			maxResults = int(mr)
			if maxResults > 50 {
				maxResults = 50
			}
		}

		events, err := m.calendarService.FetchEvents(ctx, m.userID, startDate, endDate)
		if err != nil {
			// Check for token expiration using proper sentinel error
			if errors.Is(err, ErrCalendarTokenExpired) {
				return nil, fmt.Errorf("Google Calendar access expired. Please reconnect your Google account.")
			}
			return nil, fmt.Errorf("failed to fetch calendar events: %w", err)
		}

		if len(events) > maxResults {
			events = events[:maxResults]
		}

		formattedEvents := formatCalendarEventsForAI(events)

		return map[string]interface{}{
			"events":     formattedEvents,
			"count":      len(formattedEvents),
			"date_range": map[string]string{"start": startDate.Format("2006-01-02"), "end": endDate.Format("2006-01-02")},
		}, nil

	case "calendar_create_event":
		if m.calendarService == nil {
			return nil, fmt.Errorf("Google Calendar is not configured. Please connect your Google account first.")
		}

		title, _ := arguments["title"].(string)
		startTimeStr, _ := arguments["start_time"].(string)
		endTimeStr, _ := arguments["end_time"].(string)

		if title == "" || startTimeStr == "" || endTimeStr == "" {
			return nil, fmt.Errorf("title, start_time, and end_time are required")
		}

		event, err := buildCalendarEvent(arguments)
		if err != nil {
			return nil, fmt.Errorf("invalid event data: %w", err)
		}

		createdEvent, err := m.calendarService.CreateGoogleEvent(ctx, m.userID, event)
		if err != nil {
			return nil, fmt.Errorf("failed to create calendar event: %w", err)
		}

		response := map[string]interface{}{
			"success":   true,
			"event_id":  createdEvent.Id,
			"title":     createdEvent.Summary,
			"start":     createdEvent.Start.DateTime,
			"end":       createdEvent.End.DateTime,
			"html_link": createdEvent.HtmlLink,
		}

		if createdEvent.ConferenceData != nil && len(createdEvent.ConferenceData.EntryPoints) > 0 {
			for _, ep := range createdEvent.ConferenceData.EntryPoints {
				if ep.EntryPointType == "video" {
					response["meet_link"] = ep.Uri
					break
				}
			}
		}

		if len(createdEvent.Attendees) > 0 {
			attendeeEmails := make([]string, len(createdEvent.Attendees))
			for i, a := range createdEvent.Attendees {
				attendeeEmails[i] = a.Email
			}
			response["attendees"] = attendeeEmails
			response["invitations_sent"] = true
		}

		return response, nil

	case "calendar_update_event":
		if m.calendarService == nil {
			return nil, fmt.Errorf("Google Calendar is not configured. Please connect your Google account first.")
		}

		eventID, _ := arguments["event_id"].(string)
		if eventID == "" {
			return nil, fmt.Errorf("event_id is required")
		}

		srv, err := m.calendarService.GetCalendarAPI(ctx, m.userID)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Google Calendar: %w", err)
		}

		existingEvent, err := srv.Events.Get("primary", eventID).Do()
		if err != nil {
			return nil, fmt.Errorf("event not found: %s", eventID)
		}

		applyEventUpdates(existingEvent, arguments)

		updatedEvent, err := m.calendarService.UpdateGoogleEvent(ctx, m.userID, eventID, existingEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to update calendar event: %w", err)
		}

		return map[string]interface{}{
			"success":   true,
			"event_id":  updatedEvent.Id,
			"title":     updatedEvent.Summary,
			"start":     updatedEvent.Start.DateTime,
			"end":       updatedEvent.End.DateTime,
			"html_link": updatedEvent.HtmlLink,
			"updated":   true,
		}, nil

	case "calendar_delete_event":
		if m.calendarService == nil {
			return nil, fmt.Errorf("Google Calendar is not configured. Please connect your Google account first.")
		}

		eventID, _ := arguments["event_id"].(string)
		if eventID == "" {
			return nil, fmt.Errorf("event_id is required")
		}

		err := m.calendarService.DeleteEvent(ctx, m.userID, eventID)
		if err != nil {
			// Check for 404/not found errors using type assertion for Google API errors
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) && apiErr.Code == 404 {
				return nil, fmt.Errorf("event not found: %s", eventID)
			}
			return nil, fmt.Errorf("failed to delete calendar event: %w", err)
		}

		return map[string]interface{}{
			"success":  true,
			"event_id": eventID,
			"deleted":  true,
			"message":  "Event deleted. Attendees will receive cancellation notices.",
		}, nil

	default:
		return nil, fmt.Errorf("unknown calendar tool: %s", toolName)
	}
}

func IsCalendarTool(toolName string) bool {
	return strings.HasPrefix(toolName, "calendar_")
}

type CalendarEventForAI struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	AllDay      bool     `json:"all_day"`
	Location    string   `json:"location,omitempty"`
	Description string   `json:"description,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	MeetLink    string   `json:"meet_link,omitempty"`
	HtmlLink    string   `json:"html_link,omitempty"`
	Status      string   `json:"status,omitempty"`
	IsRecurring bool     `json:"is_recurring,omitempty"`
}

func parseCalendarDateRange(arguments map[string]interface{}) (start, end time.Time) {
	now := time.Now()
	start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if startStr, ok := arguments["start_date"].(string); ok && startStr != "" {
		switch strings.ToLower(startStr) {
		case "today":
		case "tomorrow":
			start = start.AddDate(0, 0, 1)
		default:
			if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
				start = parsed
			}
		}
	}

	end = start.AddDate(0, 0, 7).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	if endStr, ok := arguments["end_date"].(string); ok && endStr != "" {
		switch strings.ToLower(endStr) {
		case "today":
			end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		case "tomorrow":
			end = time.Date(now.Year(), now.Month(), now.Day()+1, 23, 59, 59, 0, now.Location())
		default:
			if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
				end = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			}
		}
	}

	return start, end
}

func formatCalendarEventsForAI(events []*calendar.Event) []CalendarEventForAI {
	formatted := make([]CalendarEventForAI, len(events))

	for i, event := range events {
		e := CalendarEventForAI{
			ID:       event.Id,
			Title:    event.Summary,
			Location: event.Location,
			Status:   event.Status,
			HtmlLink: event.HtmlLink,
		}

		if event.Start.Date != "" {
			e.AllDay = true
			e.StartTime = event.Start.Date
			e.EndTime = event.End.Date
		} else {
			e.AllDay = false
			e.StartTime = event.Start.DateTime
			e.EndTime = event.End.DateTime
		}

		if len(event.Description) > 200 {
			e.Description = event.Description[:200] + "..."
		} else {
			e.Description = event.Description
		}

		if len(event.Attendees) > 0 {
			e.Attendees = make([]string, len(event.Attendees))
			for j, a := range event.Attendees {
				e.Attendees[j] = a.Email
			}
		}

		if event.ConferenceData != nil {
			for _, ep := range event.ConferenceData.EntryPoints {
				if ep.EntryPointType == "video" {
					e.MeetLink = ep.Uri
					break
				}
			}
		}

		e.IsRecurring = len(event.Recurrence) > 0 || event.RecurringEventId != ""
		formatted[i] = e
	}

	return formatted
}

func buildCalendarEvent(arguments map[string]interface{}) (*calendar.Event, error) {
	event := &calendar.Event{}

	title, _ := arguments["title"].(string)
	startTimeStr, _ := arguments["start_time"].(string)
	endTimeStr, _ := arguments["end_time"].(string)

	event.Summary = title
	isAllDay := !strings.Contains(startTimeStr, "T")

	if isAllDay {
		event.Start = &calendar.EventDateTime{Date: startTimeStr}
		event.End = &calendar.EventDateTime{Date: endTimeStr}
	} else {
		timezone, _ := arguments["timezone"].(string)
		if timezone == "" {
			timezone = "UTC"
		}

		if !strings.Contains(startTimeStr, "Z") && !strings.Contains(startTimeStr, "+") && !strings.Contains(startTimeStr, "-") {
			loc, err := time.LoadLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", startTimeStr, loc); err == nil {
				startTimeStr = t.Format(time.RFC3339)
			}
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", endTimeStr, loc); err == nil {
				endTimeStr = t.Format(time.RFC3339)
			}
		}

		event.Start = &calendar.EventDateTime{DateTime: startTimeStr, TimeZone: timezone}
		event.End = &calendar.EventDateTime{DateTime: endTimeStr, TimeZone: timezone}
	}

	if desc, ok := arguments["description"].(string); ok && desc != "" {
		event.Description = desc
	}
	if loc, ok := arguments["location"].(string); ok && loc != "" {
		event.Location = loc
	}
	if attendees, ok := arguments["attendees"].([]interface{}); ok && len(attendees) > 0 {
		event.Attendees = make([]*calendar.EventAttendee, len(attendees))
		for i, a := range attendees {
			email, _ := a.(string)
			event.Attendees[i] = &calendar.EventAttendee{Email: email}
		}
	}
	if addMeet, ok := arguments["add_meet_link"].(bool); ok && addMeet {
		event.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId:             uuid.New().String(),
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		}
	}
	if recurrence, ok := arguments["recurrence"].(string); ok && recurrence != "" {
		if rrule := buildRecurrenceRule(recurrence, arguments); rrule != "" {
			event.Recurrence = []string{rrule}
		}
	}

	return event, nil
}

func buildRecurrenceRule(recurrence string, arguments map[string]interface{}) string {
	count := 10
	if c, ok := arguments["recurrence_count"].(float64); ok {
		count = int(c)
	}

	if strings.HasPrefix(strings.ToUpper(recurrence), "RRULE:") {
		return recurrence
	}

	switch strings.ToLower(recurrence) {
	case "daily":
		return fmt.Sprintf("RRULE:FREQ=DAILY;COUNT=%d", count)
	case "weekly":
		return fmt.Sprintf("RRULE:FREQ=WEEKLY;COUNT=%d", count)
	case "monthly":
		return fmt.Sprintf("RRULE:FREQ=MONTHLY;COUNT=%d", count)
	case "weekdays":
		return fmt.Sprintf("RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR;COUNT=%d", count)
	case "biweekly":
		return fmt.Sprintf("RRULE:FREQ=WEEKLY;INTERVAL=2;COUNT=%d", count)
	default:
		return ""
	}
}

func applyEventUpdates(event *calendar.Event, arguments map[string]interface{}) {
	if title, ok := arguments["title"].(string); ok && title != "" {
		event.Summary = title
	}

	if desc, ok := arguments["description"].(string); ok {
		event.Description = desc
	}

	if loc, ok := arguments["location"].(string); ok {
		event.Location = loc
	}

	if startTimeStr, ok := arguments["start_time"].(string); ok && startTimeStr != "" {
		if strings.Contains(startTimeStr, "T") {
			event.Start.DateTime = startTimeStr
			event.Start.Date = ""
		} else {
			event.Start.Date = startTimeStr
			event.Start.DateTime = ""
		}
	}

	if endTimeStr, ok := arguments["end_time"].(string); ok && endTimeStr != "" {
		if strings.Contains(endTimeStr, "T") {
			event.End.DateTime = endTimeStr
			event.End.Date = ""
		} else {
			event.End.Date = endTimeStr
			event.End.DateTime = ""
		}
	}

	if attendees, ok := arguments["attendees"].([]interface{}); ok {
		event.Attendees = make([]*calendar.EventAttendee, len(attendees))
		for i, a := range attendees {
			email, _ := a.(string)
			event.Attendees[i] = &calendar.EventAttendee{Email: email}
		}
	}

	if addMeet, ok := arguments["add_meet_link"].(bool); ok && addMeet {
		if event.ConferenceData == nil {
			event.ConferenceData = &calendar.ConferenceData{
				CreateRequest: &calendar.CreateConferenceRequest{
					RequestId: uuid.New().String(),
					ConferenceSolutionKey: &calendar.ConferenceSolutionKey{
						Type: "hangoutsMeet",
					},
				},
			}
		}
	}
}
