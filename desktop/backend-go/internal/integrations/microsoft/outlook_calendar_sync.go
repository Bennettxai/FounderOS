package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// SyncEventsResult represents the result of an events sync.
type SyncEventsResult struct {
	TotalEvents  int `json:"total_events"`
	SyncedEvents int `json:"synced_events"`
	FailedEvents int `json:"failed_events"`
}

// Graph API event structure
type graphEvent struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	BodyPreview string `json:"bodyPreview"`
	Start       struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"end"`
	IsAllDay bool `json:"isAllDay"`
	Location *struct {
		DisplayName string `json:"displayName"`
	} `json:"location"`
	Attendees []struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
		Type   string `json:"type"`
		Status struct {
			Response string `json:"response"`
		} `json:"status"`
	} `json:"attendees"`
	Organizer *struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"organizer"`
	IsOnlineMeeting bool `json:"isOnlineMeeting"`
	OnlineMeeting   *struct {
		JoinUrl string `json:"joinUrl"`
	} `json:"onlineMeeting"`
	ResponseStatus *struct {
		Response string `json:"response"`
	} `json:"responseStatus"`
	Importance                 string   `json:"importance"`
	ShowAs                     string   `json:"showAs"`
	IsCancelled                bool     `json:"isCancelled"`
	IsReminderOn               bool     `json:"isReminderOn"`
	ReminderMinutesBeforeStart int      `json:"reminderMinutesBeforeStart"`
	Categories                 []string `json:"categories"`
	CreatedDateTime            string   `json:"createdDateTime"`
	LastModifiedDateTime       string   `json:"lastModifiedDateTime"`
}

// SyncEvents syncs calendar events from Outlook.
func (s *OutlookService) SyncEvents(ctx context.Context, userID string, timeMin, timeMax time.Time) (*SyncEventsResult, error) {
	slog.Info("Outlook calendar sync starting for user", "user_id", userID)

	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	result := &SyncEventsResult{}

	// Build URL with time filter
	apiURL := fmt.Sprintf("%s/me/calendarView?startDateTime=%s&endDateTime=%s&$top=100&$select=id,subject,bodyPreview,start,end,location,attendees,organizer,isOnlineMeeting,onlineMeeting,responseStatus,importance,showAs,isCancelled,isReminderOn,reminderMinutesBeforeStart,categories,createdDateTime,lastModifiedDateTime,isAllDay",
		GraphAPIBase, url.QueryEscape(timeMin.Format(time.RFC3339)), url.QueryEscape(timeMax.Format(time.RFC3339)))

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var eventResp struct {
		Value []graphEvent `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&eventResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.TotalEvents = len(eventResp.Value)

	for _, event := range eventResp.Value {
		if err := s.saveEvent(ctx, userID, &event); err != nil {
			slog.Info("Failed to save event", "id", event.ID, "error", err)
			result.FailedEvents++
		} else {
			result.SyncedEvents++
		}
	}

	slog.Info("Outlook calendar sync complete for user : synced / events", "user_id", userID, "synced", result.SyncedEvents, "total", result.TotalEvents)

	return result, nil
}

func (s *OutlookService) saveEvent(ctx context.Context, userID string, event *graphEvent) error {
	// Parse start/end times
	var startDateTime, endDateTime *time.Time
	if event.Start.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", event.Start.DateTime)
		startDateTime = &t
	}
	if event.End.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", event.End.DateTime)
		endDateTime = &t
	}

	// Parse attendees
	attendees := make([]EventAttendee, 0)
	for _, a := range event.Attendees {
		attendees = append(attendees, EventAttendee{
			Email:          a.EmailAddress.Address,
			Name:           a.EmailAddress.Name,
			Type:           a.Type,
			ResponseStatus: a.Status.Response,
		})
	}

	// Extract organizer
	var organizerEmail, organizerName string
	if event.Organizer != nil {
		organizerEmail = event.Organizer.EmailAddress.Address
		organizerName = event.Organizer.EmailAddress.Name
	}

	// Extract location
	var locationDisplayName string
	if event.Location != nil {
		locationDisplayName = event.Location.DisplayName
	}

	// Extract online meeting URL
	var onlineMeetingURL string
	if event.OnlineMeeting != nil {
		onlineMeetingURL = event.OnlineMeeting.JoinUrl
	}

	// Extract response status
	var responseStatus string
	if event.ResponseStatus != nil {
		responseStatus = event.ResponseStatus.Response
	}

	// Parse timestamps
	var createdDateTime, lastModifiedDateTime *time.Time
	if event.CreatedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, event.CreatedDateTime)
		createdDateTime = &t
	}
	if event.LastModifiedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, event.LastModifiedDateTime)
		lastModifiedDateTime = &t
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_calendar_events (
			user_id, event_id, subject, body_preview,
			location_display_name, start_datetime, start_timezone, end_datetime, end_timezone, is_all_day,
			attendees, organizer_email, organizer_name,
			is_online_meeting, online_meeting_url, response_status,
			importance, show_as, is_cancelled, is_reminder_on, reminder_minutes_before_start,
			categories, created_datetime, last_modified_datetime, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NOW())
		ON CONFLICT (user_id, event_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			body_preview = EXCLUDED.body_preview,
			location_display_name = EXCLUDED.location_display_name,
			start_datetime = EXCLUDED.start_datetime,
			start_timezone = EXCLUDED.start_timezone,
			end_datetime = EXCLUDED.end_datetime,
			end_timezone = EXCLUDED.end_timezone,
			is_all_day = EXCLUDED.is_all_day,
			attendees = EXCLUDED.attendees,
			organizer_email = EXCLUDED.organizer_email,
			organizer_name = EXCLUDED.organizer_name,
			is_online_meeting = EXCLUDED.is_online_meeting,
			online_meeting_url = EXCLUDED.online_meeting_url,
			response_status = EXCLUDED.response_status,
			importance = EXCLUDED.importance,
			show_as = EXCLUDED.show_as,
			is_cancelled = EXCLUDED.is_cancelled,
			is_reminder_on = EXCLUDED.is_reminder_on,
			reminder_minutes_before_start = EXCLUDED.reminder_minutes_before_start,
			categories = EXCLUDED.categories,
			last_modified_datetime = EXCLUDED.last_modified_datetime,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, event.ID, event.Subject, event.BodyPreview,
		locationDisplayName, startDateTime, event.Start.TimeZone, endDateTime, event.End.TimeZone, event.IsAllDay,
		attendees, organizerEmail, organizerName,
		event.IsOnlineMeeting, onlineMeetingURL, responseStatus,
		event.Importance, event.ShowAs, event.IsCancelled, event.IsReminderOn, event.ReminderMinutesBeforeStart,
		event.Categories, createdDateTime, lastModifiedDateTime)

	return err
}
