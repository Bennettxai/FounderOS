// Package google provides HTTP handlers for individual Google tool integrations.
package google

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

// ToolHandler provides HTTP handlers for a specific Google tool.
type ToolHandler struct {
	pool           *pgxpool.Pool
	provider       *ToolProvider
	calendar       *CalendarService
	gmail          *GmailService
	OnEventCreated EventCreatedHook
}

type googleServiceProvider interface {
	GetTokenSource(context.Context, string) (oauth2.TokenSource, error)
	Pool() *pgxpool.Pool
}

// NewToolHandler creates a new handler for a specific Google tool.
func NewToolHandler(pool *pgxpool.Pool, toolID string) (*ToolHandler, error) {
	provider, err := NewToolProvider(pool, toolID)
	if err != nil {
		return nil, err
	}

	h := &ToolHandler{
		pool:     pool,
		provider: provider,
	}

	// Initialize service-specific handlers based on tool type
	switch toolID {
	case "google_calendar":
		h.calendar = NewCalendarService(provider)
	case "google_gmail":
		h.gmail = NewGmailService(provider)
	}

	return h, nil
}

// Gmail returns the handler's internal GmailService so callers can
// attach hooks (OnEmailSaved). nil for non-Gmail tool handlers.
func (h *ToolHandler) Gmail() *GmailService { return h.gmail }

// Calendar returns the handler's internal CalendarService. nil for
// non-Calendar tool handlers.
func (h *ToolHandler) Calendar() *CalendarService { return h.calendar }

// RegisterRoutes registers routes for this tool.
func (h *ToolHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Common OAuth routes for all tools
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Tool-specific routes
	switch h.provider.ID() {
	case "google_calendar":
		h.registerCalendarRoutes(r)
	case "google_gmail":
		h.registerGmailRoutes(r)
	}
}

func (h *ToolHandler) registerCalendarRoutes(r *gin.RouterGroup) {
	calendar := r.Group("/calendar")
	{
		calendar.GET("/events", h.GetCalendarEvents)
		calendar.POST("/events", h.CreateCalendarEvent)
		calendar.DELETE("/events/:id", h.DeleteCalendarEvent)
		calendar.POST("/sync", h.SyncCalendar)
	}
}

func (h *ToolHandler) registerGmailRoutes(r *gin.RouterGroup) {
	gmail := r.Group("/gmail")
	{
		gmail.GET("/emails", h.GetEmails)
		gmail.GET("/emails/:id", h.GetEmail)
		gmail.POST("/emails/send", h.SendEmail)
		gmail.POST("/emails/:id/read", h.MarkEmailRead)
		gmail.POST("/emails/:id/archive", h.ArchiveEmail)
		gmail.DELETE("/emails/:id", h.DeleteEmail)
		gmail.POST("/sync", h.SyncGmail)
		gmail.GET("/stats", h.GetGmailStats)
	}
}

// ============================================================================
// OAuth Handlers
// ============================================================================

// GetAuthURL returns the OAuth authorization URL for this specific tool.
func (h *ToolHandler) GetAuthURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Generate state with user ID, tool ID, and the requested final redirect.
	// The desktop app passes a businessos:// deep link here so the callback can
	// bounce back into the packaged renderer (see getSafeRedirectURL).
	state := generateToolState(userID, h.provider.ID(), c.Query("redirect_uri"))
	authURL := h.provider.GetAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"tool_id":  h.provider.ID(),
		"scopes":   h.provider.tool.Scopes,
	})
}

// HandleCallback handles the OAuth callback for this tool.
func (h *ToolHandler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	// Extract user ID, tool ID, and the requested redirect from state. Providers
	// preserve `state` across the consent hop but NOT arbitrary query params, so
	// the desktop deep-link target must be read from state, not from the query.
	userID, toolID, redirect := extractToolState(state)
	frontendURL := getSafeRedirectURL(redirect, h.provider.ID())

	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"&error=missing_authorization_code")
		return
	}

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	// Verify tool ID matches
	if toolID != "" && toolID != h.provider.ID() {
		slog.Info("Tool ID mismatch", "expected", h.provider.ID(), "got", toolID)
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

	// Redirect to frontend success page
	// SECURITY: Validate redirect_uri to prevent open redirect attacks
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// Disconnect removes the user's connection to this tool.
func (h *ToolHandler) Disconnect(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tool_id": h.provider.ID(),
	})
}

// GetStatus returns the connection status for this tool.
func (h *ToolHandler) GetStatus(c *gin.Context) {
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

	authorized := status.Connected && h.provider.HasAuthorizedCredential(c.Request.Context(), userID)
	c.JSON(http.StatusOK, gin.H{
		"tool_id":    h.provider.ID(),
		"tool_name":  h.provider.Name(),
		"connected":  status.Connected,
		"authorized": authorized,
		"account":    status.AccountName,
		"email":      status.AccountName,
		"scopes":     status.Scopes,
		"last_sync":  status.LastSyncAt,
	})
}

// ============================================================================
// Calendar Handlers
// ============================================================================

// GetCalendarEvents returns calendar events.
func (h *ToolHandler) GetCalendarEvents(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if connected
	if !h.provider.IsConnected(c.Request.Context(), userID) {
		c.JSON(http.StatusOK, gin.H{
			"events":    []interface{}{},
			"count":     0,
			"connected": false,
			"message":   "Google Calendar not connected",
		})
		return
	}
	if !h.provider.HasAuthorizedCredential(c.Request.Context(), userID) {
		c.JSON(http.StatusPreconditionRequired, gin.H{
			"error": "Google Calendar permission is required. Reconnect Google Calendar and approve calendar access.",
			"code":  "google_calendar_reauth_required",
		})
		return
	}

	// Parse date range
	start := time.Now().AddDate(0, 0, -7)
	end := time.Now().AddDate(0, 1, 0)

	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr := c.Query("end"); endStr != "" {
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
func (h *ToolHandler) CreateCalendarEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if !h.provider.IsConnected(c.Request.Context(), userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google Calendar not connected"})
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
func (h *ToolHandler) DeleteCalendarEvent(c *gin.Context) {
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
func (h *ToolHandler) SyncCalendar(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if !h.provider.IsConnected(c.Request.Context(), userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google Calendar not connected"})
		return
	}
	if !h.provider.HasAuthorizedCredential(c.Request.Context(), userID) {
		c.JSON(http.StatusPreconditionRequired, gin.H{
			"error": "Google Calendar permission is required. Reconnect Google Calendar and approve calendar access.",
			"code":  "google_calendar_reauth_required",
		})
		return
	}

	timeMin := time.Now().AddDate(0, -1, 0)
	timeMax := time.Now().AddDate(0, 3, 0)

	result, err := h.calendar.SyncEvents(c.Request.Context(), userID, timeMin, timeMax)
	if err != nil {
		slog.Info("Failed to sync calendar", "error", err)
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusForbidden {
			c.JSON(http.StatusPreconditionRequired, gin.H{
				"error": "Google Calendar permission was rejected. Reconnect Google Calendar and approve calendar access.",
				"code":  "google_calendar_reauth_required",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync calendar"})
		return
	}
	_, _ = h.pool.Exec(c.Request.Context(), `
		UPDATE user_integrations SET last_used_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND provider_id = $2
	`, userID, h.provider.ID())

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Gmail Handlers
// ============================================================================

// GetEmails returns emails from a folder.
func (h *ToolHandler) GetEmails(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if !h.provider.IsConnected(c.Request.Context(), userID) {
		c.JSON(http.StatusOK, gin.H{
			"emails":    []interface{}{},
			"count":     0,
			"connected": false,
			"message":   "Gmail not connected",
		})
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
		"emails":    emails,
		"count":     len(emails),
		"folder":    folder,
		"connected": true,
	})
}

// GetEmail returns a single email.
func (h *ToolHandler) GetEmail(c *gin.Context) {
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

// SendEmail sends an email.
func (h *ToolHandler) SendEmail(c *gin.Context) {
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

// MarkEmailRead marks an email as read in the DB and Gmail.
func (h *ToolHandler) MarkEmailRead(c *gin.Context) {
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
func (h *ToolHandler) ArchiveEmail(c *gin.Context) {
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

// DeleteEmail moves an email to trash.
func (h *ToolHandler) DeleteEmail(c *gin.Context) {
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
func (h *ToolHandler) SyncGmail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if !h.provider.IsConnected(c.Request.Context(), userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gmail not connected"})
		return
	}

	maxResults := int64(100)
	if mr := c.Query("max_results"); mr != "" {
		if n, err := strconv.ParseInt(mr, 10, 64); err == nil {
			maxResults = n
		}
	} else if mr := c.Query("max"); mr != "" {
		// frontend sends "max" — accept both
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync complete",
		"result":  result,
	})
}

// GetGmailStats returns aggregate counts and last-sync time for the user's Gmail.
// Shape matches frontend GmailStats: {has_access, total_emails, unread_count, last_sync}.
func (h *ToolHandler) GetGmailStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	hasAccess := h.gmail.IsConnected(c.Request.Context(), userID)
	if !hasAccess {
		c.JSON(http.StatusOK, gin.H{
			"has_access":   false,
			"total_emails": 0,
			"unread_count": 0,
			"last_sync":    nil,
		})
		return
	}

	var total, unread int
	var lastSync *time.Time
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE NOT is_read),
			MAX(updated_at)
		FROM emails
		WHERE user_id = $1 AND provider = 'gmail'
	`, userID).Scan(&total, &unread, &lastSync)
	if err != nil {
		slog.Info("Failed to load gmail stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_access":   true,
		"total_emails": total,
		"unread_count": unread,
		"last_sync":    lastSync,
	})
}

// ============================================================================
// State Helpers
// ============================================================================

func generateToolState(userID, toolID, redirect string) string {
	data := map[string]string{
		"user_id":   userID,
		"tool_id":   toolID,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if redirect != "" {
		data["redirect"] = redirect
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func extractToolState(state string) (userID, toolID, redirect string) {
	var data map[string]string
	if err := json.Unmarshal([]byte(state), &data); err != nil {
		return "", "", ""
	}
	return data["user_id"], data["tool_id"], data["redirect"]
}

// getSafeRedirectURL validates and returns a safe redirect URL.
// SECURITY: Prevents open redirect attacks by only allowing known-safe origins.
func getSafeRedirectURL(requestedURL string, toolID string) string {
	frontendURL := getConfiguredFrontendURL()
	base := frontendURL + "/connectors"

	if allowed := validateRedirectTarget(requestedURL, frontendURL); allowed != "" {
		base = allowed
	}

	// Always attach the connected marker. Appending here (rather than baking it
	// into a default) guarantees a query string exists, so the caller's
	// "&error=..." append stays well-formed for every target, including the
	// query-less businessos:// deep link.
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "connected=" + url.QueryEscape(toolID)
}

// validateRedirectTarget returns the requested redirect URL if it targets a
// trusted origin, or "" if it should be rejected. It allows the desktop
// businessos:// deep-link scheme (how a packaged Electron app is re-entered
// from the external browser) plus the http(s) localhost dev origins and the
// configured/production frontend URLs.
func validateRedirectTarget(requestedURL, frontendURL string) string {
	if requestedURL == "" {
		return ""
	}

	parsed, err := url.Parse(requestedURL)
	if err != nil {
		slog.Info("Invalid redirect URL", "id", err)
		return ""
	}

	// Desktop deep link: the OS routes businessos:// back into the app.
	if parsed.Scheme == "businessos" {
		return requestedURL
	}

	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:5273",
		"http://localhost:3000",
		"https://localhost:5173",
		"https://localhost:5273",
		"https://localhost:3000",
	}
	if frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	if prodURL := os.Getenv("PRODUCTION_FRONTEND_URL"); prodURL != "" {
		allowedOrigins = append(allowedOrigins, prodURL)
	}

	requestedOrigin := parsed.Scheme + "://" + parsed.Host
	for _, allowed := range allowedOrigins {
		if strings.HasPrefix(allowed, requestedOrigin) || requestedOrigin == allowed {
			return requestedURL
		}
	}

	slog.Warn("Blocked redirect to untrusted origin", "origin", requestedOrigin, "allowed", allowedOrigins)
	return ""
}

func getConfiguredFrontendURL() string {
	if frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/"); frontendURL != "" {
		return frontendURL
	}

	if frontendPort := strings.TrimSpace(os.Getenv("FRONTEND_PORT")); frontendPort != "" {
		return "http://localhost:" + frontendPort
	}

	return "http://localhost:5173"
}

// Note: decodeJSON is defined in helpers.go
