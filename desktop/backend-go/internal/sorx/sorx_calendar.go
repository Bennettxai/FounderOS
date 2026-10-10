package sorx

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/google"
	"google.golang.org/api/calendar/v3"
)

// ============================================================================
// Google Calendar Actions
// ============================================================================

func googleCalendarListEvents(ctx context.Context, ac ActionContext) (interface{}, error) {
	daysAhead := 7
	if val, ok := ac.Params["days_ahead"].(float64); ok {
		daysAhead = int(val)
	}

	slog.Info("googleCalendarListEvents", "user_id", ac.Execution.UserID, "days_ahead", daysAhead)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	provider := google.NewProvider(pool, nil)
	calSvc := google.NewCalendarService(provider)
	srv, err := calSvc.GetCalendarAPI(ctx, ac.Execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	timeMin := time.Now()
	timeMax := timeMin.AddDate(0, 0, daysAhead)

	var events []*calendar.Event
	err = retryWithBackoff(ctx, 3, func() error {
		resp, err := srv.Events.List("primary").
			TimeMin(timeMin.Format(time.RFC3339)).
			TimeMax(timeMax.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime").
			MaxResults(100).
			Do()
		if err != nil {
			return err
		}
		events = resp.Items
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		startTime := event.Start.DateTime
		if startTime == "" {
			startTime = event.Start.Date
		}
		results = append(results, map[string]interface{}{
			"id":      event.Id,
			"summary": event.Summary,
			"start":   startTime,
		})
	}

	slog.Info("googleCalendarListEvents success", "count", len(results))
	return map[string]interface{}{
		"events":     results,
		"count":      len(results),
		"days_ahead": daysAhead,
		"provider":   creds.Provider,
	}, nil
}

func googleCalendarCreateEvent(ctx context.Context, ac ActionContext) (interface{}, error) {
	title, _ := ac.Params["title"].(string)
	startTime, _ := ac.Params["start_time"].(string)

	if title == "" || startTime == "" {
		return nil, fmt.Errorf("title and start_time are required")
	}

	slog.Info("googleCalendarCreateEvent", "user_id", ac.Execution.UserID, "title", title)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	provider := google.NewProvider(pool, nil)
	calSvc := google.NewCalendarService(provider)
	srv, err := calSvc.GetCalendarAPI(ctx, ac.Execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Parse start time
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format (use RFC3339): %w", err)
	}

	// Default 1 hour duration
	end := start.Add(1 * time.Hour)
	if endTimeStr, ok := ac.Params["end_time"].(string); ok {
		if parsedEnd, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			end = parsedEnd
		}
	}

	event := &calendar.Event{
		Summary: title,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
		},
	}

	if desc, ok := ac.Params["description"].(string); ok {
		event.Description = desc
	}

	var createdEvent *calendar.Event
	err = retryWithBackoff(ctx, 3, func() error {
		created, err := srv.Events.Insert("primary", event).Do()
		if err != nil {
			return err
		}
		createdEvent = created
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	slog.Info("googleCalendarCreateEvent success", "event_id", createdEvent.Id)
	return map[string]interface{}{
		"created":  true,
		"title":    title,
		"event_id": createdEvent.Id,
		"start":    startTime,
		"provider": creds.Provider,
	}, nil
}

func googleCalendarGetEvent(ctx context.Context, ac ActionContext) (interface{}, error) {
	eventID, _ := ac.Params["event_id"].(string)

	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}

	slog.Info("googleCalendarGetEvent", "user_id", ac.Execution.UserID, "event_id", eventID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	provider := google.NewProvider(pool, nil)
	calSvc := google.NewCalendarService(provider)
	srv, err := calSvc.GetCalendarAPI(ctx, ac.Execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	var event *calendar.Event
	err = retryWithBackoff(ctx, 3, func() error {
		e, err := srv.Events.Get("primary", eventID).Do()
		if err != nil {
			return err
		}
		event = e
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	startTime := event.Start.DateTime
	if startTime == "" {
		startTime = event.Start.Date
	}
	endTime := event.End.DateTime
	if endTime == "" {
		endTime = event.End.Date
	}

	attendees := make([]map[string]interface{}, 0, len(event.Attendees))
	for _, att := range event.Attendees {
		attendees = append(attendees, map[string]interface{}{
			"email":  att.Email,
			"status": att.ResponseStatus,
		})
	}

	return map[string]interface{}{
		"event_id":    event.Id,
		"title":       event.Summary,
		"description": event.Description,
		"start":       startTime,
		"end":         endTime,
		"attendees":   attendees,
		"provider":    creds.Provider,
	}, nil
}
