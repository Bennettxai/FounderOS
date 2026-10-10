package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/container"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/terminal"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TerminalHandler handles terminal-related HTTP requests
type TerminalHandler struct {
	wsHandler    *terminal.WebSocketHandler
	manager      *terminal.Manager
	containerMgr *container.ContainerManager
	pubsub       *terminal.TerminalPubSub
}

// NewTerminalHandler creates a new terminal handler
// pubsub is optional - if nil, terminal works without horizontal scaling
func NewTerminalHandler(containerMgr *container.ContainerManager, pubsub *terminal.TerminalPubSub) *TerminalHandler {
	manager := terminal.NewManager(containerMgr)

	// Wire up pub/sub if available
	if pubsub != nil {
		manager.SetPubSub(pubsub)
		slog.Info("[Terminal] Pub/sub enabled for horizontal scaling", "instance", pubsub.InstanceID())
	} else {
		slog.Info("[Terminal] Pub/sub disabled - single instance mode")
	}

	wsHandler := terminal.NewWebSocketHandler(manager)
	return &TerminalHandler{
		wsHandler:    wsHandler,
		manager:      manager,
		containerMgr: containerMgr,
		pubsub:       pubsub,
	}
}

// HandleWebSocket handles WebSocket terminal connections
// @Summary Connect to terminal via WebSocket
// @Description Establishes a WebSocket connection for real-time terminal I/O
// @Tags terminal
// @Produce json
// @Param cols query int false "Terminal columns" default(80)
// @Param rows query int false "Terminal rows" default(24)
// @Param shell query string false "Shell to use" default(zsh)
// @Param cwd query string false "Working directory"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/terminal/ws [get]
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	slog.Info("[Terminal] HandleWebSocket called from", "value", c.Request.RemoteAddr)
	slog.Info("[Terminal] Request headers", "value", c.Request.Header)

	// Get authenticated user from context (set by AuthMiddleware as "user")
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Info("[Terminal] No authenticated user found in context")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	slog.Info("[Terminal] User authenticated", "name", user.Name, "id", user.ID)

	// Upgrade to WebSocket and handle connection
	h.wsHandler.HandleConnection(c.Writer, c.Request, user.ID)
}

// ListSessions lists all active terminal sessions for the user
// @Summary List terminal sessions
// @Description Returns all active terminal sessions for the authenticated user
// @Tags terminal
// @Produce json
// @Success 200 {object} map[string]interface{} "Sessions list"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/terminal/sessions [get]
func (h *TerminalHandler) ListSessions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessions := h.manager.GetUserSessions(user.ID)

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// CloseSession closes a specific terminal session
// @Summary Close terminal session
// @Description Closes and cleans up a terminal session owned by the authenticated user
// @Tags terminal
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]string "Session closed"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not your session"
// @Failure 404 {object} map[string]string "Session not found"
// @Router /api/terminal/sessions/{id} [delete]
func (h *TerminalHandler) CloseSession(c *gin.Context) {
	// Get authenticated user from context
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID := c.Param("id")

	// Retrieve session to verify ownership
	session, err := h.manager.GetSession(sessionID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to retrieve terminal session",
			"session_id", sessionID,
			"error", err)
		utils.RespondNotFound(c, slog.Default(), "Terminal session")
		return
	}

	// CRITICAL SECURITY CHECK: Verify session ownership
	// Prevents users from closing other users' sessions (IDOR vulnerability - OWASP A01)
	if session.UserID != user.ID {
		slog.WarnContext(c.Request.Context(), "Unauthorized session close attempt - ownership mismatch",
			"session_id", sessionID,
			"session_owner", session.UserID,
			"requesting_user", user.ID,
			"remote_addr", c.Request.RemoteAddr)
		utils.RespondForbidden(c, slog.Default(), "You can only close your own terminal sessions")
		return
	}

	// Close the session after ownership verification
	if err := h.manager.CloseSession(sessionID); err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to close terminal session",
			"session_id", sessionID,
			"error", err)
		utils.RespondNotFound(c, slog.Default(), "Terminal session")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session closed"})
}

// GetSandboxChanges returns the git diff summary for a sandbox session
// @Summary Get sandbox changes
// @Description Returns git diff summary for a sandbox terminal session
// @Tags terminal
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} terminal.SandboxChanges
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Session not found"
// @Router /api/terminal/sessions/{id}/changes [get]
func (h *TerminalHandler) GetSandboxChanges(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID := c.Param("id")
	changes, err := h.manager.GetSandboxChanges(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, changes)
}

// GetManager returns the terminal manager (for cleanup)
func (h *TerminalHandler) GetManager() *terminal.Manager {
	return h.manager
}

// Shutdown gracefully shuts down the terminal handler
func (h *TerminalHandler) Shutdown() {
	if h.pubsub != nil {
		slog.Info("[Terminal] Closing pub/sub connections...")
		h.pubsub.Close()
	}
	h.manager.Shutdown()
}
