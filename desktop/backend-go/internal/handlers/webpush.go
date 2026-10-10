package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// WebPushHandler handles Web Push API endpoints
type WebPushHandler struct {
	svc *services.WebPushService
}

// NewWebPushHandler creates a new Web Push handler
func NewWebPushHandler(svc *services.WebPushService) *WebPushHandler {
	return &WebPushHandler{svc: svc}
}

// GetVAPIDPublicKey returns the VAPID public key for frontend subscription
// GET /api/notifications/push/vapid-public-key
func (h *WebPushHandler) GetVAPIDPublicKey(c *gin.Context) {
	publicKey := h.svc.GetPublicKey()
	if publicKey == "" {
		utils.RespondServiceUnavailable(c, slog.Default(), "Web Push")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key": publicKey,
		"enabled":    true,
	})
}

// POST /api/notifications/push/subscribe
func (h *WebPushHandler) Subscribe(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	if !h.svc.IsEnabled() {
		utils.RespondServiceUnavailable(c, slog.Default(), "Web Push")
		return
	}

	var input services.SubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if input.Endpoint == "" || input.P256dh == "" || input.Auth == "" {
		utils.RespondBadRequest(c, slog.Default(), "endpoint, p256dh, and auth are required")
		return
	}

	// Get user agent from request if not provided
	if input.UserAgent == "" {
		input.UserAgent = c.GetHeader("User-Agent")
	}

	if err := h.svc.Subscribe(c.Request.Context(), user.ID, input); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save subscription", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// POST /api/notifications/push/unsubscribe
func (h *WebPushHandler) Unsubscribe(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondBadRequest(c, slog.Default(), "endpoint is required")
		return
	}

	if err := h.svc.Unsubscribe(c.Request.Context(), user.ID, req.Endpoint); err != nil {
		utils.RespondInternalError(c, slog.Default(), "remove subscription", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestPush sends a test push notification to the current user
// POST /api/notifications/push/test
func (h *WebPushHandler) TestPush(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	if !h.svc.IsEnabled() {
		utils.RespondServiceUnavailable(c, slog.Default(), "Web Push")
		return
	}

	payload := services.PushPayload{
		Title:    "Test Notification",
		Body:     "Web Push is working! 🎉",
		Icon:     "/icon-192.png",
		Tag:      "test",
		Priority: "normal",
		Data: map[string]interface{}{
			"type": "test",
		},
	}

	if err := h.svc.SendToUser(c.Request.Context(), user.ID, payload); err != nil {
		utils.RespondInternalError(c, slog.Default(), "send test notification", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Test notification sent"})
}
