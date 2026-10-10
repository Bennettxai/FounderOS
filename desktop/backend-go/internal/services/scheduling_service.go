// Package services provides the scheduling service for intelligent meeting scheduling.
//
// This service is the "brain" of the calendar scheduling feature. It:
// 1. Queries Google Calendar's FreeBusy API to find when people are busy
// 2. Calculates available time slots where ALL attendees are free
// 3. Applies user preferences (prefer morning, avoid certain days, etc.)
// 4. Ranks and scores the available slots
// 5. Returns top proposals for user approval (NOT auto-creating events)
//
// Reference: docs/CALENDAR_VOICE_ARCHITECTURE.md - Flow 2: Agent-Driven Scheduling
package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"google.golang.org/api/calendar/v3"
)

// SchedulingService handles intelligent meeting scheduling.
type SchedulingService struct {
	calendarService CalendarAPIProvider
}

// CalendarAPIProvider interface for calendar operations (allows mocking in tests).
type CalendarAPIProvider interface {
	GetCalendarAPI(ctx context.Context, userID string) (*calendar.Service, error)
}

// NewSchedulingService creates a new scheduling service.
func NewSchedulingService(calendarService CalendarAPIProvider) *SchedulingService {
	return &SchedulingService{
		calendarService: calendarService,
	}
}

// ============================================================================
// Types for Availability & Scheduling
// ============================================================================

// TimeSlot represents a time window.
type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AvailabilityRequest contains parameters for checking availability.
type AvailabilityRequest struct {
	Attendees       []string   `json:"attendees"`        // Email addresses
	DurationMinutes int        `json:"duration_minutes"` // Meeting length
	DateRange       DateRange  `json:"date_range"`       // Search window
	WorkingHours    *TimeRange `json:"working_hours,omitempty"`
	Timezone        string     `json:"timezone,omitempty"` // e.g., "America/New_York"
}

// DateRange represents a date range for searching.
type DateRange struct {
	Start string `json:"start"` // YYYY-MM-DD
	End   string `json:"end"`   // YYYY-MM-DD
}

// TimeRange represents working hours.
type TimeRange struct {
	Start string `json:"start"` // HH:MM (24h format)
	End   string `json:"end"`   // HH:MM (24h format)
}

// AvailabilityResponse contains available time slots.
type AvailabilityResponse struct {
	AvailableSlots   []TimeSlot `json:"available_slots"`
	AttendeesChecked []string   `json:"attendees_checked"`
	WorkingHours     *TimeRange `json:"working_hours,omitempty"`
	TotalSlots       int        `json:"total_slots"`
}

// ScheduleRequest contains parameters for proposing meeting times.
type ScheduleRequest struct {
	Title           string                `json:"title"`
	Attendees       []string              `json:"attendees"`
	DurationMinutes int                   `json:"duration_minutes"`
	MeetingType     string                `json:"meeting_type,omitempty"` // kickoff, standup, etc.
	TimePreference  string                `json:"time_preference,omitempty"` // morning, afternoon, any, next_week
	Preferences     *SchedulingPreferences `json:"preferences,omitempty"`
	DateRange       *DateRange            `json:"date_range,omitempty"` // Optional: defaults to next 7 days
}

// SchedulingPreferences contains user scheduling preferences.
type SchedulingPreferences struct {
	PreferMorning   bool     `json:"prefer_morning"`
	PreferAfternoon bool     `json:"prefer_afternoon"`
	AvoidDays       []string `json:"avoid_days,omitempty"` // ["Monday", "Friday"]
	PreferredDays   []string `json:"preferred_days,omitempty"`
	BufferMinutes   int      `json:"buffer_minutes,omitempty"` // Gap before/after meetings
}

// ScheduleProposal contains proposed meeting times.
type ScheduleProposal struct {
	Proposals  []ProposedSlot `json:"proposals"`
	EventDraft EventDraft     `json:"event_draft"`
}

// ProposedSlot is a single proposed time with score and reason.
type ProposedSlot struct {
	Slot   TimeSlot `json:"slot"`
	Score  float64  `json:"score"`  // 0.0 to 1.0
	Reason string   `json:"reason"` // Human-readable explanation
}

// EventDraft contains the event details to be created after user approval.
type EventDraft struct {
	Title           string   `json:"title"`
	Attendees       []string `json:"attendees"`
	DurationMinutes int      `json:"duration_minutes"`
	MeetingType     string   `json:"meeting_type,omitempty"`
}

// ============================================================================
// Core Methods
// ============================================================================

// GetAvailability finds available time slots for all attendees.
// This uses Google Calendar's FreeBusy API to check everyone's calendars.
func (s *SchedulingService) GetAvailability(ctx context.Context, userID string, req AvailabilityRequest) (*AvailabilityResponse, error) {
	// Get the calendar API for the user
	srv, err := s.calendarService.GetCalendarAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar API: %w", err)
	}

	// Parse date range
	startDate, err := time.Parse("2006-01-02", req.DateRange.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", req.DateRange.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	// Include the full end day
	endDate = endDate.Add(24 * time.Hour)

	// Default working hours
	workingHours := &TimeRange{Start: "09:00", End: "17:00"}
	if req.WorkingHours != nil {
		workingHours = req.WorkingHours
	}

	// Build FreeBusy request
	// Include the current user plus all specified attendees
	freeBusyItems := []*calendar.FreeBusyRequestItem{
		{Id: "primary"}, // Current user
	}
	for _, email := range req.Attendees {
		freeBusyItems = append(freeBusyItems, &calendar.FreeBusyRequestItem{Id: email})
	}

	freeBusyReq := &calendar.FreeBusyRequest{
		TimeMin: startDate.Format(time.RFC3339),
		TimeMax: endDate.Format(time.RFC3339),
		Items:   freeBusyItems,
	}

	// Call FreeBusy API
	freeBusyResp, err := srv.Freebusy.Query(freeBusyReq).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query freebusy: %w", err)
	}

	// Collect all busy periods
	allBusyPeriods := []TimeSlot{}
	for calID, busyInfo := range freeBusyResp.Calendars {
		slog.Info("Calendar  has  busy periods", "value0", calID, "value1", len(busyInfo.Busy))
		for _, busy := range busyInfo.Busy {
			start, _ := time.Parse(time.RFC3339, busy.Start)
			end, _ := time.Parse(time.RFC3339, busy.End)
			allBusyPeriods = append(allBusyPeriods, TimeSlot{Start: start, End: end})
		}
	}

	// Find available slots
	availableSlots := s.findAvailableSlots(
		startDate,
		endDate,
		allBusyPeriods,
		req.DurationMinutes,
		workingHours,
	)

	return &AvailabilityResponse{
		AvailableSlots:   availableSlots,
		AttendeesChecked: append([]string{"primary"}, req.Attendees...),
		WorkingHours:     workingHours,
		TotalSlots:       len(availableSlots),
	}, nil
}

// ProposeSchedule finds optimal meeting times and returns ranked proposals.
// This is the main method used by the agent for intelligent scheduling.
func (s *SchedulingService) ProposeSchedule(ctx context.Context, userID string, req ScheduleRequest) (*ScheduleProposal, error) {
	// Set default date range (next 7 days) if not provided
	dateRange := req.DateRange
	if dateRange == nil {
		now := time.Now()
		dateRange = &DateRange{
			Start: now.Format("2006-01-02"),
			End:   now.AddDate(0, 0, 7).Format("2006-01-02"),
		}
	}

	// Adjust date range based on time preference
	if req.TimePreference == "next_week" {
		now := time.Now()
		// Find next Monday
		daysUntilMonday := (8 - int(now.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		nextMonday := now.AddDate(0, 0, daysUntilMonday)
		dateRange = &DateRange{
			Start: nextMonday.Format("2006-01-02"),
			End:   nextMonday.AddDate(0, 0, 5).Format("2006-01-02"), // Mon-Fri
		}
	}

	// Get availability
	availResp, err := s.GetAvailability(ctx, userID, AvailabilityRequest{
		Attendees:       req.Attendees,
		DurationMinutes: req.DurationMinutes,
		DateRange:       *dateRange,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get availability: %w", err)
	}

	// Score and rank slots
	scoredSlots := s.scoreSlots(availResp.AvailableSlots, req)

	// Sort by score (highest first)
	sort.Slice(scoredSlots, func(i, j int) bool {
		return scoredSlots[i].Score > scoredSlots[j].Score
	})

	// Take top 3 proposals
	maxProposals := 3
	if len(scoredSlots) < maxProposals {
		maxProposals = len(scoredSlots)
	}
	proposals := scoredSlots[:maxProposals]

	return &ScheduleProposal{
		Proposals: proposals,
		EventDraft: EventDraft{
			Title:           req.Title,
			Attendees:       req.Attendees,
			DurationMinutes: req.DurationMinutes,
			MeetingType:     req.MeetingType,
		},
	}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// findAvailableSlots calculates free time slots from busy periods.
func (s *SchedulingService) findAvailableSlots(
	rangeStart, rangeEnd time.Time,
	busyPeriods []TimeSlot,
	durationMinutes int,
	workingHours *TimeRange,
) []TimeSlot {
	var availableSlots []TimeSlot
	duration := time.Duration(durationMinutes) * time.Minute

	// Parse working hours
	workStart, _ := time.Parse("15:04", workingHours.Start)
	workEnd, _ := time.Parse("15:04", workingHours.End)

	// Sort busy periods by start time
	sort.Slice(busyPeriods, func(i, j int) bool {
		return busyPeriods[i].Start.Before(busyPeriods[j].Start)
	})

	// Merge overlapping busy periods
	mergedBusy := mergeBusyPeriods(busyPeriods)

	// Iterate through each day in the range
	for day := rangeStart; day.Before(rangeEnd); day = day.AddDate(0, 0, 1) {
		// Skip weekends
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}

		// Set working hours for this day
		dayStart := time.Date(day.Year(), day.Month(), day.Day(),
			workStart.Hour(), workStart.Minute(), 0, 0, day.Location())
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(),
			workEnd.Hour(), workEnd.Minute(), 0, 0, day.Location())

		// Find free slots during working hours
		freeSlots := s.findFreeSlotsInDay(dayStart, dayEnd, mergedBusy, duration)
		availableSlots = append(availableSlots, freeSlots...)
	}

	return availableSlots
}

// findFreeSlotsInDay finds available slots within a single day.
func (s *SchedulingService) findFreeSlotsInDay(
	dayStart, dayEnd time.Time,
	busyPeriods []TimeSlot,
	minDuration time.Duration,
) []TimeSlot {
	var freeSlots []TimeSlot

	current := dayStart
	for _, busy := range busyPeriods {
		// Skip busy periods outside this day
		if busy.End.Before(dayStart) || busy.Start.After(dayEnd) {
			continue
		}

		// Adjust busy period to day boundaries
		busyStart := busy.Start
		busyEnd := busy.End
		if busyStart.Before(dayStart) {
			busyStart = dayStart
		}
		if busyEnd.After(dayEnd) {
			busyEnd = dayEnd
		}

		// If there's a gap before this busy period
		if current.Before(busyStart) {
			gap := busyStart.Sub(current)
			if gap >= minDuration {
				// Create slots in 30-minute increments
				for slotStart := current; slotStart.Add(minDuration).Before(busyStart) || slotStart.Add(minDuration).Equal(busyStart); slotStart = slotStart.Add(30 * time.Minute) {
					freeSlots = append(freeSlots, TimeSlot{
						Start: slotStart,
						End:   slotStart.Add(minDuration),
					})
				}
			}
		}

		// Move current to end of busy period
		if busyEnd.After(current) {
			current = busyEnd
		}
	}

	// Check for remaining time at end of day
	if current.Before(dayEnd) {
		gap := dayEnd.Sub(current)
		if gap >= minDuration {
			for slotStart := current; slotStart.Add(minDuration).Before(dayEnd) || slotStart.Add(minDuration).Equal(dayEnd); slotStart = slotStart.Add(30 * time.Minute) {
				freeSlots = append(freeSlots, TimeSlot{
					Start: slotStart,
					End:   slotStart.Add(minDuration),
				})
			}
		}
	}

	return freeSlots
}

// scoreSlots assigns scores to available slots based on preferences.
func (s *SchedulingService) scoreSlots(slots []TimeSlot, req ScheduleRequest) []ProposedSlot {
	var scored []ProposedSlot

	for _, slot := range slots {
		score := 1.0
		reasons := []string{"All attendees free"}

		hour := slot.Start.Hour()
		dayName := slot.Start.Weekday().String()

		// Apply time preference scoring
		if req.TimePreference == "morning" || (req.Preferences != nil && req.Preferences.PreferMorning) {
			if hour >= 9 && hour < 12 {
				score += 0.2
				reasons = append(reasons, "morning slot")
			} else if hour >= 12 {
				score -= 0.1
			}
		} else if req.TimePreference == "afternoon" || (req.Preferences != nil && req.Preferences.PreferAfternoon) {
			if hour >= 13 && hour < 17 {
				score += 0.2
				reasons = append(reasons, "afternoon slot")
			} else if hour < 12 {
				score -= 0.1
			}
		}

		// Avoid specified days
		if req.Preferences != nil {
			for _, avoidDay := range req.Preferences.AvoidDays {
				if dayName == avoidDay {
					score -= 0.3
					break
				}
			}
			for _, preferDay := range req.Preferences.PreferredDays {
				if dayName == preferDay {
					score += 0.15
					reasons = append(reasons, fmt.Sprintf("preferred day (%s)", dayName))
					break
				}
			}
		}

		// Prefer slots not too early or too late
		if hour == 9 || hour == 10 {
			score += 0.05 // Slight preference for standard meeting times
		}
		if hour >= 16 {
			score -= 0.05 // Slight penalty for late day meetings
		}

		// Cap score between 0 and 1
		if score > 1.0 {
			score = 1.0
		}
		if score < 0.0 {
			score = 0.0
		}

		// Build reason string
		reasonStr := reasons[0]
		if len(reasons) > 1 {
			reasonStr = fmt.Sprintf("%s, %s", reasons[0], reasons[len(reasons)-1])
		}

		scored = append(scored, ProposedSlot{
			Slot:   slot,
			Score:  score,
			Reason: reasonStr,
		})
	}

	return scored
}

// mergeBusyPeriods combines overlapping busy periods.
func mergeBusyPeriods(periods []TimeSlot) []TimeSlot {
	if len(periods) == 0 {
		return periods
	}

	// Sort by start time
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].Start.Before(periods[j].Start)
	})

	merged := []TimeSlot{periods[0]}
	for i := 1; i < len(periods); i++ {
		last := &merged[len(merged)-1]
		current := periods[i]

		// If overlapping or adjacent, merge
		if current.Start.Before(last.End) || current.Start.Equal(last.End) {
			if current.End.After(last.End) {
				last.End = current.End
			}
		} else {
			merged = append(merged, current)
		}
	}

	return merged
}
