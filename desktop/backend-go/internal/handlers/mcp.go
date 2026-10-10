package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/integrations/notion"
	"github.com/rhl/businessos-backend/internal/integrations/slack"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// MCPHandler handles Model Context Protocol tool operations
type MCPHandler struct {
	pool *pgxpool.Pool
}

// NewMCPHandler creates a new MCPHandler
func NewMCPHandler(pool *pgxpool.Pool) *MCPHandler {
	return &MCPHandler{pool: pool}
}

func (h *MCPHandler) createMCPService(userID string) *services.MCPService {
	// Create providers and services for each integration
	googleProvider := google.NewProviderWithAllFeatures(h.pool)
	calendarService := google.NewCalendarService(googleProvider)

	slackProvider := slack.NewProvider(h.pool)
	slackChannelService := slack.NewChannelService(slackProvider)

	notionProvider := notion.NewProvider(h.pool)
	notionDBService := notion.NewDatabaseService(notionProvider)

	return services.NewMCPService(h.pool, userID, calendarService, slackChannelService, notionDBService)
}

func (h *MCPHandler) ListMCPTools(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	mcpService := h.createMCPService(user.ID)
	tools := mcpService.GetAllTools()

	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

func (h *MCPHandler) ExecuteMCPTool(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Tool      string                 `json:"tool" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid MCP tool execution request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	mcpService := h.createMCPService(user.ID)
	result, err := mcpService.ExecuteTool(c.Request.Context(), req.Tool, req.Arguments)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "MCP tool execution failed",
			"tool", req.Tool,
			"user_id", user.ID,
			"error", err)
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// MCPHealth returns the health status of MCP services
func (h *MCPHandler) MCPHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "mcp",
	})
}
