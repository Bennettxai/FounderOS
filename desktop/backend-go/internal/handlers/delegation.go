package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DelegationHandler handles agent delegation endpoints
type DelegationHandler struct {
	delegationService *services.DelegationService
}

// NewDelegationHandler creates a new delegation handler
func NewDelegationHandler(delegationService *services.DelegationService) *DelegationHandler {
	return &DelegationHandler{
		delegationService: delegationService,
	}
}

// ListAgents returns all available agents for delegation
// GET /api/agents/available
func (h *DelegationHandler) ListAgents(c *gin.Context) {
	userID := "default"
	if user := middleware.GetCurrentUser(c); user != nil {
		userID = user.ID
	}

	agents, err := h.delegationService.ListAvailableAgents(c.Request.Context(), userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list agents", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// ResolveAgentMention resolves an @mention to an agent
// GET /api/agents/resolve/:mention
func (h *DelegationHandler) ResolveAgentMention(c *gin.Context) {
	userID := "default"
	if user := middleware.GetCurrentUser(c); user != nil {
		userID = user.ID
	}

	mention := c.Param("mention")
	if mention == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	agent, err := h.delegationService.ResolveAgentMention(c.Request.Context(), userID, mention)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Agent")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent":    agent,
		"mention":  mention,
		"resolved": true,
	})
}

// ExtractMentions extracts all @mentions from a message
// POST /api/agents/mentions
func (h *DelegationHandler) ExtractMentions(c *gin.Context) {
	userID := "default"
	if user := middleware.GetCurrentUser(c); user != nil {
		userID = user.ID
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	mentions := h.delegationService.ExtractMentions(c.Request.Context(), userID, req.Message)

	c.JSON(http.StatusOK, gin.H{
		"mentions": mentions,
		"count":    len(mentions),
	})
}

// DelegateRequest represents a delegation request
type DelegateRequest struct {
	FromAgent     string            `json:"from_agent"`
	ToAgent       string            `json:"to_agent" binding:"required"`
	Reason        string            `json:"reason"`
	Context       string            `json:"context"`
	OriginalQuery string            `json:"original_query"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Delegate initiates a delegation to another agent
// POST /api/agents/delegate
func (h *DelegationHandler) Delegate(c *gin.Context) {
	userID := "default"
	if user := middleware.GetCurrentUser(c); user != nil {
		userID = user.ID
	}

	var req DelegateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	result, err := h.delegationService.Delegate(c.Request.Context(), services.DelegationRequest{
		FromAgent:     req.FromAgent,
		ToAgent:       req.ToAgent,
		Reason:        req.Reason,
		Context:       req.Context,
		OriginalQuery: req.OriginalQuery,
		UserID:        userID,
		Metadata:      req.Metadata,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delegate request", err)
		return
	}

	c.JSON(http.StatusOK, result)
}
