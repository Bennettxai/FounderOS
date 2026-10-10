package google

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	integrations "github.com/rhl/businessos-backend/internal/integrations"
)

// EventCreatedHook is called after a calendar event is created via the
// Google integration. The BO server uses this to mirror events into
// the OptimalEngine knowledge graph. nil = no-op.
type EventCreatedHook func(ctx context.Context, event *CalendarEvent, userID string)

// Handler provides HTTP handlers for Google integration routes.
type Handler struct {
	provider       *Provider
	calendar       *CalendarService
	gmail          *GmailService
	OnEventCreated EventCreatedHook
}

// NewHandler creates a new Google integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
		calendar: NewCalendarService(provider),
		gmail:    NewGmailService(provider),
	}
}

// Gmail returns the handler's internal GmailService so callers can
// attach hooks (e.g. OnEmailSaved to mirror saved emails into the
// OptimalEngine knowledge graph).
func (h *Handler) Gmail() *GmailService { return h.gmail }

// Calendar returns the handler's internal CalendarService.
func (h *Handler) Calendar() *CalendarService { return h.calendar }

// RegisterRoutes registers all Google integration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// OAuth routes
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Calendar routes
	if h.provider.HasFeature("calendar") {
		calendar := r.Group("/calendar")
		{
			calendar.GET("/events", h.GetCalendarEvents)
			calendar.POST("/events", h.CreateCalendarEvent)
			calendar.DELETE("/events/:id", h.DeleteCalendarEvent)
			calendar.POST("/sync", h.SyncCalendar)
		}
	}

	// Gmail routes
	if h.provider.HasFeature("gmail") {
		gmail := r.Group("/gmail")
		{
			gmail.GET("/emails", h.GetEmails)
			gmail.GET("/emails/:id", h.GetEmail)
			gmail.POST("/emails/send", h.SendEmail)
			gmail.POST("/emails/:id/read", h.MarkEmailRead)
			gmail.POST("/emails/:id/archive", h.ArchiveEmail)
			gmail.DELETE("/emails/:id", h.DeleteEmail)
			gmail.POST("/sync", h.SyncGmail)
		}
	}
}

// GetAuthURL returns the OAuth authorization URL.
func (h *Handler) GetAuthURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Generate state with user ID (and desktop deep-link redirect) for callback
	state := integrations.GenerateUserStateWithRedirect(userID, c.Query("redirect_uri"))

	// Get features from query params (optional)
	features := c.QueryArray("features")
	if len(features) == 0 {
		features = h.provider.Features()
	}

	authURL := h.provider.GetAuthURLWithFeatures(state, features)
	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"features": features,
	})
}

// HandleCallback handles the OAuth callback.
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	// Extract user ID from state
	userID := integrations.ExtractUserIDFromState(state)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	// Exchange code for tokens
	token, err := h.provider.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Info("Failed to exchange code", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code"})
		return
	}

	// Save tokens
	if err := h.provider.SaveToken(c.Request.Context(), userID, token); err != nil {
		slog.Info("Failed to save token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	// Redirect to frontend success page (or the desktop deep link carried in state)
	frontendURL := getSafeRedirectURL(integrations.ExtractRedirectFromState(state), "google")
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// Disconnect disconnects the Google integration.
func (h *Handler) Disconnect(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.provider.Disconnect(c.Request.Context(), userID); err != nil {
		slog.Info("Failed to disconnect", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStatus returns the connection status.
func (h *Handler) GetStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	status, err := h.provider.GetConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Calendar Handlers

// GetCalendarEvents returns calendar events.
func (h *Handler) GetCalendarEvents(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user has Google Calendar connected
	if !h.calendar.IsConnected(c.Request.Context(), userID) {
		// Not connected - return empty events, not an error
		c.JSON(http.StatusOK, gin.H{
			"events":    []interface{}{},
			"count":     0,
			"connected": false,
			"message":   "Google Calendar not connected",
		})
		return
	}

	// Parse date range
	startStr := c.Query("start")
	endStr := c.Query("end")

	start := time.Now().AddDate(0, 0, -7) // Default: 7 days ago
	end := time.Now().AddDate(0, 1, 0)    // Default: 1 month from now

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	events, err := h.calendar.GetEvents(c.Request.Context(), userID, start, end)
	if err != nil {
		slog.Info("Failed to get events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"count":     len(events),
		"connected": true,
	})
}

// CreateCalendarEvent creates a new calendar event.
func (h *Handler) CreateCalendarEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var event CalendarEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	created, err := h.calendar.CreateEvent(c.Request.Context(), userID, &event)
	if err != nil {
		slog.Info("Failed to create event", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	if h.OnEventCreated != nil && created != nil {
		h.OnEventCreated(c.Request.Context(), created, userID)
	}

	c.JSON(http.StatusCreated, created)
}

// DeleteCalendarEvent deletes a calendar event.
func (h *Handler) DeleteCalendarEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	eventID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.calendar.DeleteEvent(c.Request.Context(), userID, eventID); err != nil {
		slog.Info("Failed to delete event", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncCalendar syncs calendar events from Google.
func (h *Handler) SyncCalendar(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse sync range
	timeMin := time.Now().AddDate(0, -1, 0) // 1 month ago
	timeMax := time.Now().AddDate(0, 3, 0)  // 3 months from now

	result, err := h.calendar.SyncEvents(c.Request.Context(), userID, timeMin, timeMax)
	if err != nil {
		slog.Info("Failed to sync calendar", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync calendar"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Gmail Handlers

// GetEmails returns emails from a folder.
func (h *Handler) GetEmails(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	folder := EmailFolder(c.DefaultQuery("folder", "inbox"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	emails, err := h.gmail.GetEmails(c.Request.Context(), userID, folder, limit, offset)
	if err != nil {
		slog.Info("Failed to get emails", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get emails"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"emails": emails,
		"count":  len(emails),
		"folder": folder,
	})
}

// GetEmail returns a single email.
func (h *Handler) GetEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	emailID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	email, err := h.gmail.GetEmailByID(c.Request.Context(), userID, emailID)
	if err != nil {
		slog.Info("Failed to get email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get email"})
		return
	}

	c.JSON(http.StatusOK, email)
}

// SendEmail sends a new email.
func (h *Handler) SendEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var email ComposeEmail
	if err := c.ShouldBindJSON(&email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.gmail.SendEmail(c.Request.Context(), userID, &email); err != nil {
		slog.Info("Failed to send email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkEmailRead marks an email as read.
func (h *Handler) MarkEmailRead(c *gin.Context) {
	userID := c.GetString("user_id")
	emailID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.gmail.MarkAsRead(c.Request.Context(), userID, emailID); err != nil {
		slog.Info("Failed to mark email as read", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ArchiveEmail archives an email.
func (h *Handler) ArchiveEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	emailID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.gmail.ArchiveEmail(c.Request.Context(), userID, emailID); err != nil {
		slog.Info("Failed to archive email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteEmail deletes an email.
func (h *Handler) DeleteEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	emailID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.gmail.DeleteEmail(c.Request.Context(), userID, emailID); err != nil {
		slog.Info("Failed to delete email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncGmail syncs emails from Gmail.
func (h *Handler) SyncGmail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	maxResults := int64(100)
	if mr := c.Query("max_results"); mr != "" {
		if n, err := strconv.ParseInt(mr, 10, 64); err == nil {
			maxResults = n
		}
	}

	result, err := h.gmail.SyncEmails(c.Request.Context(), userID, maxResults)
	if err != nil {
		slog.Info("Failed to sync gmail", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync gmail"})
		return
	}

	c.JSON(http.StatusOK, result)
}
