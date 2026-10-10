package instagram

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	provider *Provider
}

func NewHandler(provider *Provider) *Handler {
	return &Handler{provider: provider}
}

func (h *Handler) GetAuthURL(c *gin.Context) {
	userID, workspaceID, ok := h.activeWorkspace(c, true)
	if !ok {
		return
	}
	if !h.provider.Configured() {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"error":   "Instagram connector is not configured",
			"message": "Set INSTAGRAM_CLIENT_ID, INSTAGRAM_CLIENT_SECRET, INSTAGRAM_REDIRECT_URI, and TOKEN_ENCRYPTION_KEY in the backend environment.",
		})
		return
	}

	state, err := h.provider.CreateOAuthState(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.writeProviderError(c, err, "Failed to start Instagram connection")
		return
	}
	c.JSON(http.StatusOK, gin.H{"auth_url": h.provider.GetAuthURL(state)})
}

func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	userID, workspaceID, err := h.provider.ConsumeOAuthState(c.Request.Context(), state)
	if err != nil {
		h.writeProviderError(c, err, "Invalid state")
		return
	}

	token, err := h.provider.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Info("Failed to exchange Instagram code", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect Instagram"})
		return
	}
	if err := h.provider.SaveToken(c.Request.Context(), userID, workspaceID, token); err != nil {
		slog.Info("Failed to save Instagram token", "error", err)
		h.writeProviderError(c, err, "Failed to save Instagram connection")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(`<!doctype html>
<html>
	<head><title>Instagram connected</title></head>
	<body style="font-family: system-ui; padding: 32px;">
		<h1>Instagram connected</h1>
		<p>%s is now connected to BusinessOS. You can close this tab and return to Connectors.</p>
	</body>
</html>`, token.AccountName))
}

func (h *Handler) GetStatus(c *gin.Context) {
	userID, workspaceID, ok := h.activeWorkspace(c, false)
	if !ok {
		return
	}
	status, err := h.provider.GetConnectionStatus(c.Request.Context(), userID, workspaceID)
	if err != nil {
		slog.Info("Failed to get Instagram status", "error", err)
		h.writeProviderError(c, err, "Failed to get Instagram status")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"connected":    status.Connected,
		"configured":   h.provider.Configured(),
		"account_id":   status.AccountID,
		"account_name": status.AccountName,
		"connected_at": status.ConnectedAt,
		"scopes":       status.Scopes,
		"error":        status.Error,
	})
}

func (h *Handler) Disconnect(c *gin.Context) {
	userID, workspaceID, ok := h.activeWorkspace(c, true)
	if !ok {
		return
	}
	if err := h.provider.Disconnect(c.Request.Context(), userID, workspaceID); err != nil {
		slog.Info("Failed to disconnect Instagram", "error", err)
		h.writeProviderError(c, err, "Failed to disconnect Instagram")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) GetAuditPreview(c *gin.Context) {
	userID, workspaceID, ok := h.activeWorkspace(c, false)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	preview, err := h.provider.BuildAuditPreview(c.Request.Context(), userID, workspaceID, limit)
	if err != nil {
		slog.Info("Failed to build Instagram audit preview", "error", err)
		if errors.Is(err, ErrWorkspaceAccess) || errors.Is(err, ErrEncryptionUnavailable) {
			h.writeProviderError(c, err, "Failed to build Instagram audit preview")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) activeWorkspace(c *gin.Context, requireManagement bool) (string, string, bool) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return "", "", false
	}
	workspaceID := strings.TrimSpace(c.GetHeader("X-Workspace-ID"))
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Select an active workspace"})
		return "", "", false
	}
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid active workspace"})
		return "", "", false
	}
	workspaceID = parsedWorkspaceID.String()
	var accessErr error
	if requireManagement {
		accessErr = h.provider.RequireWorkspaceManagement(c.Request.Context(), userID, workspaceID)
	} else {
		accessErr = h.provider.RequireWorkspaceMembership(c.Request.Context(), userID, workspaceID)
	}
	if accessErr != nil {
		h.writeProviderError(c, accessErr, "Failed to verify workspace access")
		return "", "", false
	}
	return userID, workspaceID, true
}

func (h *Handler) writeProviderError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, ErrWorkspaceAccess):
		c.JSON(http.StatusForbidden, gin.H{"error": "Select a workspace you are an active member of"})
	case errors.Is(err, ErrWorkspaceManagement):
		c.JSON(http.StatusForbidden, gin.H{"error": "Only workspace owners and admins can manage Instagram"})
	case errors.Is(err, ErrInvalidOAuthState):
		c.JSON(http.StatusBadRequest, gin.H{"error": fallback})
	case errors.Is(err, ErrEncryptionUnavailable):
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "Instagram credential encryption is not configured"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}
