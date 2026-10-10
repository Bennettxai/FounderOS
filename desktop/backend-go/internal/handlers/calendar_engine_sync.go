package handlers

import (
	"context"

	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
	"github.com/rhl/businessos-backend/internal/services"
)

// newGoogleCalendarEngineHook returns a closure that mirrors created Google
// calendar events into the OptimalEngine knowledge graph. nil when the
// engine isn't wired so the underlying handler stays a no-op caller.
func newGoogleCalendarEngineHook(sync *services.EngineSync) google.EventCreatedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, event *google.CalendarEvent, userID string) {
		if event == nil {
			return
		}
		meta := map[string]string{
			"provider": "google",
			"status":   event.Status,
		}
		if event.Location != "" {
			meta["location"] = event.Location
		}
		if event.MeetingLink != "" {
			meta["meeting_link"] = event.MeetingLink
		}
		body := event.Title
		if event.Description != "" {
			body += "\n\n" + event.Description
		}
		sync.Enqueue(ctx, services.Signal{
			Module:     services.ModuleCalendar,
			ID:         "event-google-" + calendarEventKey(event.ID, event.GoogleID),
			AuthorID:   userID,
			Title:      event.Title,
			Body:       body,
			Genre:      "event",
			ModifiedAt: event.UpdatedAt,
			Metadata:   meta,
		})
	}
}

// newMicrosoftCalendarEngineHook is the Outlook equivalent.
func newMicrosoftCalendarEngineHook(sync *services.EngineSync) microsoft.EventCreatedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, event *microsoft.OutlookEvent, userID string) {
		if event == nil {
			return
		}
		meta := map[string]string{"provider": "microsoft"}
		if event.LocationDisplayName != "" {
			meta["location"] = event.LocationDisplayName
		}
		if event.OnlineMeetingURL != "" {
			meta["meeting_link"] = event.OnlineMeetingURL
		}
		body := event.Subject
		if event.BodyPreview != "" {
			body += "\n\n" + event.BodyPreview
		}
		sync.Enqueue(ctx, services.Signal{
			Module:     services.ModuleCalendar,
			ID:         "event-microsoft-" + calendarEventKey(event.ID, event.EventID),
			AuthorID:   userID,
			Title:      event.Subject,
			Body:       body,
			Genre:      "event",
			ModifiedAt: event.LastModifiedDateTime,
			Metadata:   meta,
		})
	}
}

func calendarEventKey(localID, providerID string) string {
	if localID != "" {
		return localID
	}
	return providerID
}
