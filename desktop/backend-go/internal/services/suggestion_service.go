// Package services provides the suggestion service for calendar-based task recommendations.
//
// This service analyzes the user's calendar and intelligently suggests tasks:
// 1. Meeting prep tasks ("Prepare agenda for Design Kickoff")
// 2. Follow-up tasks ("Send meeting notes from Client Call")
// 3. Deadline reminders ("Project X due in 2 days")
//
// Key behaviors:
// - Does NOT auto-create tasks, only shows recommendations
// - Checks against existing tasks to avoid duplicates
// - Ranks suggestions by priority and urgency
//
// Reference: docs/CALENDAR_VOICE_ARCHITECTURE.md - Flow 3: Calendar-Based Task Suggestions
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SuggestionService generates task suggestions based on calendar events.
type SuggestionService struct {
	pool *pgxpool.Pool
}

// NewSuggestionService creates a new suggestion service.
func NewSuggestionService(pool *pgxpool.Pool) *SuggestionService {
	return &SuggestionService{pool: pool}
}

// ============================================================================
// Types
// ============================================================================

// TaskSuggestion represents a suggested task based on calendar analysis.
type TaskSuggestion struct {
	Type           string    `json:"type"`             // meeting_prep, follow_up, deadline, reminder
	Title          string    `json:"title"`            // Suggested task title
	RelatedEventID string    `json:"related_event_id,omitempty"`
	RelatedEvent   string    `json:"related_event,omitempty"` // Event title for context
	SuggestedDue   time.Time `json:"suggested_due"`
	Priority       string    `json:"priority"` // high, medium, low
	Reason         string    `json:"reason"`   // Why this is suggested
}

// TaskSuggestionsResponse contains all suggestions for a user.
type TaskSuggestionsResponse struct {
	Suggestions []TaskSuggestion `json:"suggestions"`
	GeneratedAt time.Time        `json:"generated_at"`
}

// CalendarEventForSuggestion is a simplified event structure for suggestion logic.
type CalendarEventForSuggestion struct {
	ID          string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	MeetingType string
	Attendees   int
}

// ============================================================================
// Core Methods
// ============================================================================

// GetTaskSuggestions analyzes calendar and returns task suggestions.
func (s *SuggestionService) GetTaskSuggestions(ctx context.Context, userID string) (*TaskSuggestionsResponse, error) {
	suggestions := []TaskSuggestion{}
	now := time.Now()

	// 1. Get upcoming events (next 7 days) for meeting prep suggestions
	upcomingEvents, err := s.getUpcomingEvents(ctx, userID, now, now.AddDate(0, 0, 7))
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	// 2. Get recent past events (last 24 hours) for follow-up suggestions
	recentEvents, err := s.getRecentEvents(ctx, userID, now.Add(-24*time.Hour), now)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	// 3. Get existing tasks to avoid duplicates
	existingTasks, err := s.getExistingTaskTitles(ctx, userID)
	if err != nil {
		// Non-fatal: continue without dedup
		existingTasks = make(map[string]bool)
	}

	// 4. Generate meeting prep suggestions
	for _, event := range upcomingEvents {
		prepSuggestions := s.generatePrepSuggestions(event, now, existingTasks)
		suggestions = append(suggestions, prepSuggestions...)
	}

	// 5. Generate follow-up suggestions
	for _, event := range recentEvents {
		followUpSuggestions := s.generateFollowUpSuggestions(event, now, existingTasks)
		suggestions = append(suggestions, followUpSuggestions...)
	}

	// 6. Sort by priority (high first) then by due date
	sortSuggestions(suggestions)

	// 7. Limit to top 10 suggestions
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return &TaskSuggestionsResponse{
		Suggestions: suggestions,
		GeneratedAt: now,
	}, nil
}

// ============================================================================
// Database Queries
// ============================================================================

// getUpcomingEvents fetches calendar events in the given time range.
func (s *SuggestionService) getUpcomingEvents(ctx context.Context, userID string, start, end time.Time) ([]CalendarEventForSuggestion, error) {
	query := `
		SELECT id, title, description, start_time, end_time, meeting_type,
			   COALESCE(jsonb_array_length(attendees), 0) as attendee_count
		FROM calendar_events
		WHERE user_id = $1 
		  AND start_time >= $2 
		  AND start_time < $3
		  AND status != 'cancelled'
		ORDER BY start_time ASC
		LIMIT 50
	`

	rows, err := s.pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEventForSuggestion
	for rows.Next() {
		var e CalendarEventForSuggestion
		var description *string
		var meetingType *string

		err := rows.Scan(&e.ID, &e.Title, &description, &e.StartTime, &e.EndTime, &meetingType, &e.Attendees)
		if err != nil {
			continue
		}

		if description != nil {
			e.Description = *description
		}
		if meetingType != nil {
			e.MeetingType = *meetingType
		}

		events = append(events, e)
	}

	return events, nil
}

// getRecentEvents fetches calendar events that recently ended.
func (s *SuggestionService) getRecentEvents(ctx context.Context, userID string, start, end time.Time) ([]CalendarEventForSuggestion, error) {
	query := `
		SELECT id, title, description, start_time, end_time, meeting_type,
			   COALESCE(jsonb_array_length(attendees), 0) as attendee_count
		FROM calendar_events
		WHERE user_id = $1 
		  AND end_time >= $2 
		  AND end_time < $3
		  AND status != 'cancelled'
		ORDER BY end_time DESC
		LIMIT 20
	`

	rows, err := s.pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEventForSuggestion
	for rows.Next() {
		var e CalendarEventForSuggestion
		var description *string
		var meetingType *string

		err := rows.Scan(&e.ID, &e.Title, &description, &e.StartTime, &e.EndTime, &meetingType, &e.Attendees)
		if err != nil {
			continue
		}

		if description != nil {
			e.Description = *description
		}
		if meetingType != nil {
			e.MeetingType = *meetingType
		}

		events = append(events, e)
	}

	return events, nil
}

// getExistingTaskTitles returns a map of existing task titles for deduplication.
func (s *SuggestionService) getExistingTaskTitles(ctx context.Context, userID string) (map[string]bool, error) {
	query := `
		SELECT LOWER(title) FROM tasks 
		WHERE user_id = $1 
		  AND status NOT IN ('completed', 'cancelled')
		  AND created_at > NOW() - INTERVAL '30 days'
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := make(map[string]bool)
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err == nil {
			titles[title] = true
		}
	}

	return titles, nil
}

// ============================================================================
// Suggestion Generators
// ============================================================================

// generatePrepSuggestions creates prep task suggestions for upcoming events.
func (s *SuggestionService) generatePrepSuggestions(event CalendarEventForSuggestion, now time.Time, existingTasks map[string]bool) []TaskSuggestion {
	var suggestions []TaskSuggestion

	hoursUntil := event.StartTime.Sub(now).Hours()
	title := event.Title
	if title == "" {
		title = "Upcoming Meeting"
	}

	// Skip very short-notice events (less than 2 hours)
	if hoursUntil < 2 {
		return suggestions
	}

	// Meeting prep based on meeting type and timing
	switch {
	case strings.Contains(strings.ToLower(event.MeetingType), "kickoff") ||
		strings.Contains(strings.ToLower(title), "kickoff"):
		suggestion := TaskSuggestion{
			Type:           "meeting_prep",
			Title:          fmt.Sprintf("Prepare kickoff agenda for %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   event.StartTime.Add(-2 * time.Hour),
			Priority:       "high",
			Reason:         fmt.Sprintf("Kickoff meeting in %.0f hours", hoursUntil),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}

	case strings.Contains(strings.ToLower(event.MeetingType), "client") ||
		strings.Contains(strings.ToLower(title), "client"):
		suggestion := TaskSuggestion{
			Type:           "meeting_prep",
			Title:          fmt.Sprintf("Review client history before %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   event.StartTime.Add(-1 * time.Hour),
			Priority:       "high",
			Reason:         fmt.Sprintf("Client meeting in %.0f hours", hoursUntil),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}

	case strings.Contains(strings.ToLower(event.MeetingType), "review") ||
		strings.Contains(strings.ToLower(title), "review"):
		suggestion := TaskSuggestion{
			Type:           "meeting_prep",
			Title:          fmt.Sprintf("Prepare materials for %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   event.StartTime.Add(-3 * time.Hour),
			Priority:       "medium",
			Reason:         fmt.Sprintf("Review meeting in %.0f hours", hoursUntil),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}

	case event.Attendees > 3 && hoursUntil > 24:
		// Large meetings need more prep
		suggestion := TaskSuggestion{
			Type:           "meeting_prep",
			Title:          fmt.Sprintf("Prepare agenda for %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   event.StartTime.Add(-24 * time.Hour),
			Priority:       "medium",
			Reason:         fmt.Sprintf("Meeting with %d attendees in %.0f hours", event.Attendees, hoursUntil),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// generateFollowUpSuggestions creates follow-up task suggestions for past events.
func (s *SuggestionService) generateFollowUpSuggestions(event CalendarEventForSuggestion, now time.Time, existingTasks map[string]bool) []TaskSuggestion {
	var suggestions []TaskSuggestion

	hoursSince := now.Sub(event.EndTime).Hours()
	title := event.Title
	if title == "" {
		title = "Recent Meeting"
	}

	// Only suggest follow-ups for meetings that ended recently
	if hoursSince > 24 {
		return suggestions
	}

	// Meeting notes suggestion for most meetings
	if event.Attendees >= 2 {
		suggestion := TaskSuggestion{
			Type:           "follow_up",
			Title:          fmt.Sprintf("Send meeting notes from %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   now.Add(4 * time.Hour), // Due within 4 hours
			Priority:       "medium",
			Reason:         fmt.Sprintf("Meeting ended %.0f hours ago", hoursSince),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}
	}

	// Action items follow-up for important meetings
	if strings.Contains(strings.ToLower(event.MeetingType), "kickoff") ||
		strings.Contains(strings.ToLower(event.MeetingType), "planning") ||
		strings.Contains(strings.ToLower(title), "planning") {
		suggestion := TaskSuggestion{
			Type:           "follow_up",
			Title:          fmt.Sprintf("Create action items from %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   now.Add(2 * time.Hour),
			Priority:       "high",
			Reason:         fmt.Sprintf("Important meeting ended %.0f hours ago", hoursSince),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}
	}

	// Client follow-up
	if strings.Contains(strings.ToLower(event.MeetingType), "client") ||
		strings.Contains(strings.ToLower(title), "client") {
		suggestion := TaskSuggestion{
			Type:           "follow_up",
			Title:          fmt.Sprintf("Send follow-up email after %s", title),
			RelatedEventID: event.ID,
			RelatedEvent:   title,
			SuggestedDue:   now.Add(3 * time.Hour),
			Priority:       "high",
			Reason:         fmt.Sprintf("Client meeting ended %.0f hours ago", hoursSince),
		}
		if !isDuplicate(suggestion.Title, existingTasks) {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// ============================================================================
// Helpers
// ============================================================================

// isDuplicate checks if a similar task already exists.
func isDuplicate(suggestionTitle string, existingTasks map[string]bool) bool {
	// Check for exact match
	if existingTasks[strings.ToLower(suggestionTitle)] {
		return true
	}

	// Check for partial match (simplified)
	lowerSuggestion := strings.ToLower(suggestionTitle)
	for existingTitle := range existingTasks {
		// If significant overlap, consider it a duplicate
		if strings.Contains(lowerSuggestion, existingTitle) || 
		   strings.Contains(existingTitle, lowerSuggestion) {
			return true
		}
	}

	return false
}

// sortSuggestions sorts by priority then due date.
func sortSuggestions(suggestions []TaskSuggestion) {
	priorityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}

	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			// Compare priority first
			pi := priorityOrder[suggestions[i].Priority]
			pj := priorityOrder[suggestions[j].Priority]
			if pi > pj || (pi == pj && suggestions[i].SuggestedDue.After(suggestions[j].SuggestedDue)) {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
}
