package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// RouterHandler handles intent classification endpoints
type RouterHandler struct {
	routerService *services.RouterService
}

// NewRouterHandler creates a new router handler
func NewRouterHandler(pool *pgxpool.Pool) *RouterHandler {
	return &RouterHandler{
		routerService: services.NewRouterService(pool),
	}
}

// RegisterRoutes registers router-related routes
func (h *RouterHandler) RegisterRoutes(r *gin.RouterGroup) {
	router := r.Group("/router")
	{
		router.POST("/analyze", h.AnalyzeMessage)
		router.POST("/batch", h.AnalyzeBatch)
		router.GET("/intents", h.ListIntents)
	}
}

// AnalyzeMessageRequest represents a request to analyze a message
type AnalyzeMessageRequest struct {
	Message        string `json:"message" binding:"required"`
	ConversationID string `json:"conversation_id"`
}

// AnalyzeMessage analyzes a single message and returns routing decision
// @Summary Analyze message intent
// @Description Classifies the intent of a message and provides routing recommendations
// @Tags router
// @Accept json
// @Produce json
// @Param request body AnalyzeMessageRequest true "Message to analyze"
// @Success 200 {object} services.RoutingDecision
// @Router /api/router/analyze [post]
func (h *RouterHandler) AnalyzeMessage(c *gin.Context) {
	var req AnalyzeMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	decision, err := h.routerService.Route(c.Request.Context(), req.Message, req.ConversationID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "analyze message", err)
		return
	}

	c.JSON(http.StatusOK, decision)
}

// AnalyzeBatchRequest represents a request to analyze multiple messages
type AnalyzeBatchRequest struct {
	Messages       []string `json:"messages" binding:"required,min=1"`
	ConversationID string   `json:"conversation_id"`
}

// AnalyzeBatch analyzes multiple messages
// @Summary Analyze multiple messages
// @Description Classifies the intent of multiple messages at once
// @Tags router
// @Accept json
// @Produce json
// @Param request body AnalyzeBatchRequest true "Messages to analyze"
// @Success 200 {array} services.RoutingDecision
// @Router /api/router/batch [post]
func (h *RouterHandler) AnalyzeBatch(c *gin.Context) {
	var req AnalyzeBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	decisions, err := h.routerService.RouteMultiple(c.Request.Context(), req.Messages, req.ConversationID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "analyze batch", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// IntentInfo provides information about an intent type
type IntentInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	FocusMode   string `json:"suggested_focus_mode,omitempty"`
	Agent       string `json:"suggested_agent,omitempty"`
}

// ListIntents returns available intent types and their descriptions
// @Summary List intent types
// @Description Returns information about all available intent classifications
// @Tags router
// @Produce json
// @Success 200 {array} IntentInfo
// @Router /api/router/intents [get]
func (h *RouterHandler) ListIntents(c *gin.Context) {
	intents := []IntentInfo{
		{
			Type:        "chat",
			Description: "Standard conversational response",
		},
		{
			Type:        "search",
			Description: "Web search is required to answer",
			FocusMode:   "quick",
		},
		{
			Type:        "agent",
			Description: "Delegation to a specific agent via @mention",
		},
		{
			Type:        "command",
			Description: "Slash command execution",
		},
		{
			Type:        "code",
			Description: "Code generation, review, or debugging",
			FocusMode:   "code",
			Agent:       "coder",
		},
		{
			Type:        "analysis",
			Description: "Data analysis or evaluation task",
			FocusMode:   "analyze",
			Agent:       "analyst",
		},
		{
			Type:        "writing",
			Description: "Document creation or editing",
			FocusMode:   "write",
			Agent:       "writer",
		},
		{
			Type:        "planning",
			Description: "Strategic planning or brainstorming",
			FocusMode:   "plan",
			Agent:       "planner",
		},
		{
			Type:        "research",
			Description: "Deep research requiring search and synthesis",
			FocusMode:   "deep",
			Agent:       "researcher",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"intents": intents,
		"count":   len(intents),
	})
}

// QuickAnalyzeResponse is a simplified routing response
type QuickAnalyzeResponse struct {
	Intent         string `json:"intent"`
	RequiresSearch bool   `json:"requires_search"`
	SuggestedAgent string `json:"suggested_agent,omitempty"`
}

// QuickAnalyze provides a simplified intent analysis (can be used inline)
func (h *RouterHandler) QuickAnalyze(message string) (*QuickAnalyzeResponse, error) {
	intent, requiresSearch, agent := h.routerService.QuickRoute(nil, message)
	return &QuickAnalyzeResponse{
		Intent:         string(intent),
		RequiresSearch: requiresSearch,
		SuggestedAgent: agent,
	}, nil
}
