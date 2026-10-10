package microsoft

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
// Microsoft integration. The BO server uses this to mirror events into
// the OptimalEngine knowledge graph. nil = no-op.
type EventCreatedHook func(ctx context.Context, event *OutlookEvent, userID string)

// Handler provides HTTP handlers for Microsoft integration routes.
type Handler struct {
	provider       *Provider
	outlook        *OutlookService
	onedrive       *OneDriveService
	todo           *ToDoService
	teams          *TeamsService
	OnEventCreated EventCreatedHook
}

// NewHandler creates a new Microsoft integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
		outlook:  NewOutlookService(provider),
		onedrive: NewOneDriveService(provider),
		todo:     NewToDoService(provider),
		teams:    NewTeamsService(provider),
	}
}

// Outlook returns the handler's internal OutlookService so callers can
// attach hooks (OnEmailSaved). Mirrors google.ToolHandler.Gmail().
func (h *Handler) Outlook() *OutlookService { return h.outlook }

// Teams returns the handler's internal TeamsService so callers can
// attach hooks (OnMessageSaved).
func (h *Handler) Teams() *TeamsService { return h.teams }

// RegisterRoutes registers all Microsoft integration routes.
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
			calendar.POST("/sync", h.SyncCalendar)
		}
	}

	// Mail routes
	if h.provider.HasFeature("mail") {
		mail := r.Group("/mail")
		{
			mail.GET("/emails", h.GetEmails)
			mail.GET("/emails/:id", h.GetEmail)
			mail.POST("/send", h.SendEmail)
			mail.POST("/sync", h.SyncMail)
		}
	}

	// Files routes (OneDrive)
	if h.provider.HasFeature("files") {
		files := r.Group("/files")
		{
			files.GET("", h.GetFiles)
			files.GET("/:id", h.GetFile)
			files.POST("/sync", h.SyncFiles)
		}
	}

	// Tasks routes (Microsoft To Do)
	if h.provider.HasFeature("tasks") {
		tasks := r.Group("/tasks")
		{
			tasks.GET("/lists", h.GetTaskLists)
			tasks.GET("/:list_id/tasks", h.GetTasks)
			tasks.POST("/:list_id/tasks", h.CreateTask)
			tasks.POST("/:list_id/tasks/:task_id/complete", h.CompleteTask)
			tasks.POST("/sync", h.SyncTasks)
		}
	}

	// Teams routes (channels + messages, Wave 2)
	if h.provider.HasFeature("teams") {
		teams := r.Group("/teams")
		{
			teams.GET("/channels", h.GetTeamsChannels)
			teams.POST("/channels/sync", h.SyncTeamsChannels)
			teams.GET("/channels/:id/messages", h.GetTeamsChannelMessages)
			teams.POST("/channels/:id/messages", h.SendTeamsChannelMessage)
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

	// Generate state with user ID for callback
	state := integrations.GenerateUserState(userID)

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

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"account_email": token.AccountEmail,
		"scopes":        token.Scopes,
	})
}

// Disconnect disconnects the Microsoft integration.
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

// ============================================================================
// CALENDAR HANDLERS
// ============================================================================

// GetCalendarEvents returns calendar events.
func (h *Handler) GetCalendarEvents(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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

	events, err := h.outlook.GetEvents(c.Request.Context(), userID, start, end)
	if err != nil {
		slog.Info("Failed to get events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// CreateCalendarEvent creates a new calendar event.
func (h *Handler) CreateCalendarEvent(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var event OutlookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	created, err := h.outlook.CreateEvent(c.Request.Context(), userID, &event)
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

// SyncCalendar syncs calendar events from Outlook.
func (h *Handler) SyncCalendar(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse sync range
	timeMin := time.Now().AddDate(0, -1, 0) // 1 month ago
	timeMax := time.Now().AddDate(0, 3, 0)  // 3 months from now

	result, err := h.outlook.SyncEvents(c.Request.Context(), userID, timeMin, timeMax)
	if err != nil {
		slog.Info("Failed to sync calendar", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync calendar"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// MAIL HANDLERS
// ============================================================================

// GetEmails returns emails from Outlook.
func (h *Handler) GetEmails(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	folderID := c.Query("folder_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.outlook.GetMessages(c.Request.Context(), userID, folderID, limit, offset)
	if err != nil {
		slog.Info("Failed to get emails", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get emails"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"emails": messages,
		"count":  len(messages),
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

	// Get from database by ID
	var message OutlookMessage
	var conversationID, bodyPreview, bodyContent, bodyContentType, importance, folderID *string
	err := h.provider.Pool().QueryRow(c.Request.Context(), `
		SELECT id, user_id, message_id, conversation_id, subject, body_preview, body_content, body_content_type, importance,
			from_email, from_name, is_read, is_draft, has_attachments, folder_id,
			received_datetime, sent_datetime, synced_at
		FROM microsoft_mail_messages
		WHERE user_id = $1 AND message_id = $2
	`, userID, emailID).Scan(
		&message.ID, &message.UserID, &message.MessageID, &conversationID,
		&message.Subject, &bodyPreview, &bodyContent, &bodyContentType, &importance,
		&message.FromEmail, &message.FromName, &message.IsRead, &message.IsDraft,
		&message.HasAttachments, &folderID,
		&message.ReceivedDateTime, &message.SentDateTime, &message.SyncedAt,
	)

	if err != nil {
		slog.Info("Failed to get email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get email"})
		return
	}

	if conversationID != nil {
		message.ConversationID = *conversationID
	}
	if bodyPreview != nil {
		message.BodyPreview = *bodyPreview
	}
	if bodyContent != nil {
		message.BodyContent = *bodyContent
	}
	if bodyContentType != nil {
		message.BodyContentType = *bodyContentType
	}
	if importance != nil {
		message.Importance = *importance
	}
	if folderID != nil {
		message.FolderID = *folderID
	}

	c.JSON(http.StatusOK, message)
}

// SendEmail sends a new email via Outlook.
func (h *Handler) SendEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		To      []string `json:"to" binding:"required"`
		Subject string   `json:"subject" binding:"required"`
		Body    string   `json:"body" binding:"required"`
		IsHTML  bool     `json:"is_html"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.outlook.SendMessage(c.Request.Context(), userID, req.To, req.Subject, req.Body, req.IsHTML); err != nil {
		slog.Info("Failed to send email", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncMail syncs emails from Outlook.
func (h *Handler) SyncMail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	maxResults := 100
	if mr := c.Query("max_results"); mr != "" {
		if n, err := strconv.Atoi(mr); err == nil {
			maxResults = n
		}
	}

	result, err := h.outlook.SyncMessages(c.Request.Context(), userID, maxResults)
	if err != nil {
		slog.Info("Failed to sync mail", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync mail"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// FILES HANDLERS (OneDrive)
// ============================================================================

// GetFiles returns OneDrive files.
func (h *Handler) GetFiles(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	parentID := c.Query("parent_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	files, err := h.onedrive.GetFiles(c.Request.Context(), userID, parentID, limit, offset)
	if err != nil {
		slog.Info("Failed to get files", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"count": len(files),
	})
}

// GetFile returns a single OneDrive file.
func (h *Handler) GetFile(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var file OneDriveFile
	err := h.provider.Pool().QueryRow(c.Request.Context(), `
		SELECT id, user_id, item_id, name, description, mime_type, size_bytes,
			web_url, is_folder, folder_child_count, synced_at
		FROM microsoft_onedrive_files
		WHERE user_id = $1 AND item_id = $2
	`, userID, fileID).Scan(
		&file.ID, &file.UserID, &file.ItemID, &file.Name, &file.Description,
		&file.MimeType, &file.SizeBytes, &file.WebURL, &file.IsFolder,
		&file.FolderChildCount, &file.SyncedAt,
	)

	if err != nil {
		slog.Info("Failed to get file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file"})
		return
	}

	c.JSON(http.StatusOK, file)
}

// SyncFiles syncs files from OneDrive.
func (h *Handler) SyncFiles(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	maxResults := 100
	if mr := c.Query("max_results"); mr != "" {
		if n, err := strconv.Atoi(mr); err == nil {
			maxResults = n
		}
	}

	result, err := h.onedrive.SyncFiles(c.Request.Context(), userID, maxResults)
	if err != nil {
		slog.Info("Failed to sync files", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync files"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// TASKS HANDLERS (Microsoft To Do)
// ============================================================================

// GetTaskLists returns Microsoft To Do task lists.
func (h *Handler) GetTaskLists(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	lists, err := h.todo.GetLists(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get task lists", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task lists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lists": lists,
		"count": len(lists),
	})
}

// GetTasks returns tasks from a specific list.
func (h *Handler) GetTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	listID := c.Param("list_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	includeCompleted := c.Query("include_completed") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, err := h.todo.GetTasks(c.Request.Context(), userID, listID, includeCompleted, limit, offset)
	if err != nil {
		slog.Info("Failed to get tasks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// CreateTask creates a new task in a Microsoft To Do list.
func (h *Handler) CreateTask(c *gin.Context) {
	userID := c.GetString("user_id")
	listID := c.Param("list_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var task ToDoTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	created, err := h.todo.CreateTask(c.Request.Context(), userID, listID, &task)
	if err != nil {
		slog.Info("Failed to create task", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// CompleteTask marks a task as completed.
func (h *Handler) CompleteTask(c *gin.Context) {
	userID := c.GetString("user_id")
	listID := c.Param("list_id")
	taskID := c.Param("task_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.todo.CompleteTask(c.Request.Context(), userID, listID, taskID); err != nil {
		slog.Info("Failed to complete task", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncTasks syncs all tasks from all lists.
func (h *Handler) SyncTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.todo.SyncAllTasks(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to sync tasks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync tasks"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// TEAMS HANDLERS
// ============================================================================

// GetTeamsChannels returns every Teams channel the user has synced.
func (h *Handler) GetTeamsChannels(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	channels, err := h.teams.GetChannels(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get teams channels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"count":    len(channels),
	})
}

// SyncTeamsChannels triggers a Graph sync of joined teams + their channels.
func (h *Handler) SyncTeamsChannels(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.teams.SyncChannels(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to sync teams channels", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync channels"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTeamsChannelMessages lists messages in a channel by BO uuid. The
// `:id` is microsoft_teams_channels.id; we resolve to (team_id, channel_id)
// before reading microsoft_teams_messages.
func (h *Handler) GetTeamsChannelMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	channelRowID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ch, err := h.teams.GetChannelByID(c.Request.Context(), userID, channelRowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// If the client passes ?sync=true, trigger a fresh Graph fetch first
	// so the response reflects the latest state. Otherwise we read the
	// last-synced rows directly.
	if c.Query("sync") == "true" {
		if _, serr := h.teams.SyncChannelMessages(c.Request.Context(), userID, ch.TeamID, ch.ChannelID, limit); serr != nil {
			slog.Info("Teams channel message sync failed (continuing with cached)", "error", serr)
		}
	}

	messages, err := h.teams.GetChannelMessages(c.Request.Context(), userID, ch.TeamID, ch.ChannelID, limit, offset)
	if err != nil {
		slog.Info("Failed to get teams messages", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
		"channel":  ch,
	})
}

// SendTeamsChannelMessage posts a message to a Teams channel. The `:id`
// is microsoft_teams_channels.id; we resolve to Graph identifiers.
func (h *Handler) SendTeamsChannelMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	channelRowID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		IsHTML  bool   `json:"is_html"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ch, err := h.teams.GetChannelByID(c.Request.Context(), userID, channelRowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	msgID, err := h.teams.SendChannelMessage(c.Request.Context(), userID, ch.TeamID, ch.ChannelID, req.Content, req.IsHTML)
	if err != nil {
		slog.Info("Failed to send teams message", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// After Graph confirms send, sync the channel so the new message
	// lands in microsoft_teams_messages and fires the engine hook.
	if _, serr := h.teams.SyncChannelMessages(c.Request.Context(), userID, ch.TeamID, ch.ChannelID, 20); serr != nil {
		slog.Info("Teams post-send sync failed", "error", serr)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message_id": msgID,
	})
}
