package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

type NotificationHandler struct {
	svc *services.NotificationService
}

func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// ListNotifications returns paginated notifications
// GET /api/notifications
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	notifications, err := h.svc.GetForUser(c.Request.Context(), user.ID, pg.Limit, pg.Offset)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch notifications", err)
		return
	}

	total := int64(pg.Offset) + int64(len(notifications))
	if int64(len(notifications)) == int64(pg.Limit) {
		total++
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(notifications, total, pg))
}

// GetUnreadCount returns count of unread notifications
// GET /api/notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	count, err := h.svc.GetUnreadCount(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch notification count", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkAsRead marks a single notification as read
// POST /api/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "notification ID")
		return
	}

	if err := h.svc.MarkAsRead(c.Request.Context(), user.ID, id); err != nil {
		utils.RespondInternalError(c, slog.Default(), "mark notification as read", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkMultipleAsRead marks multiple notifications as read
// POST /api/notifications/read
func (h *NotificationHandler) MarkMultipleAsRead(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if err := h.svc.MarkMultipleAsRead(c.Request.Context(), user.ID, req.IDs); err != nil {
		utils.RespondInternalError(c, slog.Default(), "mark multiple notifications as read", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkAllAsRead marks all notifications as read
// POST /api/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	if err := h.svc.MarkAllAsRead(c.Request.Context(), user.ID); err != nil {
		utils.RespondInternalError(c, slog.Default(), "mark all notifications as read", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Stream handles SSE connection for real-time notifications
// GET /api/notifications/stream
func (h *NotificationHandler) Stream(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe to events; nil is returned when the per-user connection limit is reached.
	ch := h.svc.SSE().Subscribe(user.ID)
	if ch == nil {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "Too many SSE connections for this user",
		})
		return
	}
	defer h.svc.SSE().Unsubscribe(user.ID, ch)

	// Send initial connection event
	c.SSEvent("connected", gin.H{"user_id": user.ID})
	c.Writer.Flush()

	// Stream events
	for {
		select {
		case event := <-ch:
			c.SSEvent(event.Type, event.Data)
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// GetPreferences returns notification preferences
// GET /api/notifications/preferences
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	prefs, err := h.svc.GetPreferences(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch notification preferences", err)
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences updates notification preferences
// PUT /api/notifications/preferences
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var input services.UpdatePreferencesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	prefs, err := h.svc.UpdatePreferences(c.Request.Context(), user.ID, input)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update notification preferences", err)
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// DeleteNotification deletes a notification
// DELETE /api/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "notification ID")
		return
	}

	if err := h.svc.DeleteNotification(c.Request.Context(), user.ID, id); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete notification", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
