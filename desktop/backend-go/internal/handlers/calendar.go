package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	gcal "github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CalendarStatsHandler handles calendar statistics endpoints.
type CalendarStatsHandler struct {
	pool           *pgxpool.Pool
	suggestions    *services.SuggestionService
	googleCalendar *gcal.CalendarService
}

// NewCalendarStatsHandler constructs a CalendarStatsHandler.
func NewCalendarStatsHandler(pool *pgxpool.Pool) *CalendarStatsHandler {
	return &CalendarStatsHandler{
		pool:           pool,
		suggestions:    services.NewSuggestionService(pool),
		googleCalendar: gcal.NewCalendarService(gcal.NewProviderWithAllFeatures(pool)),
	}
}

// RegisterCalendarStatsRoutes wires /api/calendar routes.
func RegisterCalendarStatsRoutes(api *gin.RouterGroup, h *CalendarStatsHandler, auth gin.HandlerFunc) {
	calendar := api.Group("/calendar")
	{
		calendar.GET("/stats", auth, h.GetCalendarStats)
		calendar.GET("/upcoming", auth, h.GetUpcomingCalendarEvents)
		calendar.GET("/today", auth, h.GetTodayCalendarEvents)
		calendar.GET("/task-suggestions", auth, h.GetTaskSuggestions)

		// Native event CRUD - the calendar works standalone (no Google needed);
		// Google/Outlook sync into the same table. Workspace-scoped + personal.
		calendar.GET("/events", auth, h.ListEvents)
		calendar.POST("/events", auth, h.CreateEvent)
		calendar.GET("/events/:id", auth, h.GetEvent)
		calendar.PUT("/events/:id", auth, h.UpdateEvent)
		calendar.DELETE("/events/:id", auth, h.DeleteEvent)
	}
}

// CalendarStatsResponse is the API response for calendar stats
type CalendarStatsResponse struct {
	TotalEvents  int64           `json:"total_events"`
	GoogleEvents int64           `json:"google_events"`
	LocalEvents  int64           `json:"local_events"`
	DateRange    *DateRangeStats `json:"date_range,omitempty"`
	LastSync     *time.Time      `json:"last_sync,omitempty"`
}

type DateRangeStats struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// GetCalendarStats returns calendar statistics for the user
func (h *CalendarStatsHandler) GetCalendarStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	stats, err := queries.GetCalendarStats(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get calendar stats", err)
		return
	}

	response := CalendarStatsResponse{
		TotalEvents:  stats.TotalEvents,
		GoogleEvents: stats.GoogleEvents,
		LocalEvents:  stats.LocalEvents,
	}

	// Handle date range
	if stats.DateRangeFrom != nil || stats.DateRangeTo != nil {
		response.DateRange = &DateRangeStats{}
		if stats.DateRangeFrom != nil {
			if t, ok := stats.DateRangeFrom.(time.Time); ok {
				response.DateRange.From = &t
			} else if ts, ok := stats.DateRangeFrom.(pgtype.Timestamptz); ok && ts.Valid {
				response.DateRange.From = &ts.Time
			}
		}
		if stats.DateRangeTo != nil {
			if t, ok := stats.DateRangeTo.(time.Time); ok {
				response.DateRange.To = &t
			} else if ts, ok := stats.DateRangeTo.(pgtype.Timestamptz); ok && ts.Valid {
				response.DateRange.To = &ts.Time
			}
		}
	}

	// Handle last sync
	if stats.LastSync != nil {
		if t, ok := stats.LastSync.(time.Time); ok {
			response.LastSync = &t
		} else if ts, ok := stats.LastSync.(pgtype.Timestamptz); ok && ts.Valid {
			response.LastSync = &ts.Time
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetUpcomingCalendarEvents returns upcoming calendar events
func (h *CalendarStatsHandler) GetUpcomingCalendarEvents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse limit from query params, default to 10
	limit := int32(10)
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	queries := sqlc.New(h.pool)
	events, err := queries.GetUpcomingEvents(c.Request.Context(), sqlc.GetUpcomingEventsParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get upcoming events", err)
		return
	}

	response := make([]CalendarEventResponse, len(events))
	for i, e := range events {
		response[i] = TransformCalendarEvent(e)
	}

	c.JSON(http.StatusOK, response)
}

// GetTodayCalendarEvents returns today's calendar events
func (h *CalendarStatsHandler) GetTodayCalendarEvents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	events, err := queries.GetTodayEvents(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get today's events", err)
		return
	}

	response := make([]CalendarEventResponse, len(events))
	for i, e := range events {
		response[i] = TransformCalendarEvent(e)
	}

	c.JSON(http.StatusOK, response)
}

// GetTaskSuggestions returns calendar-derived task recommendations for the user.
func (h *CalendarStatsHandler) GetTaskSuggestions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	response, err := h.suggestions.GetTaskSuggestions(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get task suggestions", err)
		return
	}

	c.JSON(http.StatusOK, response)
}
