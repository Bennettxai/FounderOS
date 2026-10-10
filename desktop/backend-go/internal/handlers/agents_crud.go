package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ListCustomAgents returns all custom agents for the authenticated user
func (h *AgentHandler) ListCustomAgents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	// Check if we want all agents or only active ones
	includeInactive := c.Query("include_inactive") == "true"

	pg := ParsePagination(c)

	var agents []sqlc.CustomAgent
	var err error

	if includeInactive {
		agents, err = queries.GetAllCustomAgents(ctx, user.ID)
	} else {
		agents, err = queries.ListCustomAgents(ctx, user.ID)
	}

	if err != nil {
		slog.Error("Failed to list agents", "include_inactive", includeInactive, "error", err)
		utils.RespondInternalError(c, slog.Default(), "list agents", err)
		return
	}

	total := int64(len(agents))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(agents) {
		start = len(agents)
	}
	if end > len(agents) {
		end = len(agents)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(agents[start:end], total, pg))
}

// GetCustomAgent retrieves a specific custom agent
func (h *AgentHandler) GetCustomAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "agent_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	agent, err := queries.GetCustomAgent(ctx, sqlc.GetCustomAgentParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Agent")
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agent})
}

// CreateCustomAgent creates a new custom agent
func (h *AgentHandler) CreateCustomAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateCustomAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate agent name (alphanumeric + hyphens only)
	name := strings.ToLower(strings.TrimSpace(req.Name))
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Agent name can only contain lowercase letters, numbers, and hyphens"})
			return
		}
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	// Check user's agent count (rate limiting)
	count, err := queries.CountUserAgents(ctx, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "check agent count", err)
		return
	}
	if count >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 100 custom agents allowed per user. Please delete unused agents.",
		})
		return
	}

	// Validate suggested_prompts
	if len(req.SuggestedPrompts) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 10 suggested prompts allowed",
		})
		return
	}
	for i, prompt := range req.SuggestedPrompts {
		if len(strings.TrimSpace(prompt)) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Suggested prompt %d cannot be empty", i+1),
			})
			return
		}
		if len(prompt) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Suggested prompt %d exceeds 500 characters (has %d)", i+1, len(prompt)),
			})
			return
		}
	}

	// Validate welcome_message
	if len(req.WelcomeMessage) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Welcome message exceeds 2000 characters (has %d)", len(req.WelcomeMessage)),
		})
		return
	}

	// Validate category
	allowedCategories := map[string]bool{
		"general":   true,
		"coding":    true,
		"writing":   true,
		"analysis":  true,
		"research":  true,
		"support":   true,
		"sales":     true,
		"marketing": true,
	}
	if req.Category != "" && !allowedCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category. Allowed: general, coding, writing, analysis, research, support, sales, marketing",
		})
		return
	}

	// Prepare optional fields
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var avatar *string
	if req.Avatar != "" {
		avatar = &req.Avatar
	}
	var modelPref *string
	if req.ModelPreference != "" {
		modelPref = &req.ModelPreference
	}
	var maxTokens *int32
	if req.MaxTokens > 0 {
		maxTokens = &req.MaxTokens
	}
	var category *string
	if req.Category != "" {
		category = &req.Category
	}
	var welcomeMsg *string
	if req.WelcomeMessage != "" {
		welcomeMsg = &req.WelcomeMessage
	}

	// Convert temperature to pgtype.Numeric
	tempNumeric := pgtype.Numeric{}
	if req.Temperature >= 0 && req.Temperature <= 2.0 {
		tempNumeric.Scan(req.Temperature)
	} else if req.Temperature > 2.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Temperature must be between 0.0 and 2.0",
		})
		return
	}

	thinkingEnabled := &req.ThinkingEnabled
	streamingEnabled := &req.StreamingEnabled
	applyPersonalization := &req.ApplyPersonalization
	isActive := boolPtr(true)
	isPublic := &req.IsPublic
	isFeatured := &req.IsFeatured

	agent, err := queries.CreateCustomAgent(ctx, sqlc.CreateCustomAgentParams{
		UserID:               user.ID,
		Name:                 name,
		DisplayName:          req.DisplayName,
		Description:          desc,
		Avatar:               avatar,
		SystemPrompt:         req.SystemPrompt,
		ModelPreference:      modelPref,
		Temperature:          tempNumeric,
		MaxTokens:            maxTokens,
		Capabilities:         req.Capabilities,
		ToolsEnabled:         req.ToolsEnabled,
		ContextSources:       req.ContextSources,
		ThinkingEnabled:      thinkingEnabled,
		StreamingEnabled:     streamingEnabled,
		ApplyPersonalization: applyPersonalization,
		WelcomeMessage:       welcomeMsg,
		SuggestedPrompts:     req.SuggestedPrompts,
		Category:             category,
		IsActive:             isActive,
		IsPublic:             isPublic,
		IsFeatured:           isFeatured,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"agent": agent})
}

// UpdateCustomAgent updates an existing custom agent
func (h *AgentHandler) UpdateCustomAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "agent_id")
		return
	}

	var req UpdateCustomAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	// Validate name if provided
	if req.Name != nil {
		name := strings.ToLower(strings.TrimSpace(*req.Name))
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Agent name can only contain lowercase letters, numbers, and hyphens"})
				return
			}
		}
		req.Name = &name
	}

	// Validate suggested_prompts if provided
	if req.SuggestedPrompts != nil {
		if len(req.SuggestedPrompts) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Maximum 10 suggested prompts allowed",
			})
			return
		}
		for i, prompt := range req.SuggestedPrompts {
			if len(strings.TrimSpace(prompt)) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Suggested prompt %d cannot be empty", i+1),
				})
				return
			}
			if len(prompt) > 500 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Suggested prompt %d exceeds 500 characters (has %d)", i+1, len(prompt)),
				})
				return
			}
		}
	}

	// Validate welcome_message if provided
	if req.WelcomeMessage != nil && len(*req.WelcomeMessage) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Welcome message exceeds 2000 characters (has %d)", len(*req.WelcomeMessage)),
		})
		return
	}

	// Validate category if provided
	if req.Category != nil && *req.Category != "" {
		allowedCategories := map[string]bool{
			"general":   true,
			"coding":    true,
			"writing":   true,
			"analysis":  true,
			"research":  true,
			"support":   true,
			"sales":     true,
			"marketing": true,
		}
		if !allowedCategories[*req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid category. Allowed: general, coding, writing, analysis, research, support, sales, marketing",
			})
			return
		}
	}

	// Convert temperature to pgtype.Numeric
	tempNumeric := pgtype.Numeric{}
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2.0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Temperature must be between 0.0 and 2.0",
			})
			return
		}
		tempNumeric.Scan(*req.Temperature)
	}

	agent, err := queries.UpdateCustomAgent(ctx, sqlc.UpdateCustomAgentParams{
		ID:                   pgtype.UUID{Bytes: id, Valid: true},
		UserID:               user.ID,
		Name:                 req.Name,
		DisplayName:          req.DisplayName,
		Description:          req.Description,
		Avatar:               req.Avatar,
		SystemPrompt:         req.SystemPrompt,
		ModelPreference:      req.ModelPreference,
		Temperature:          tempNumeric,
		MaxTokens:            req.MaxTokens,
		Capabilities:         req.Capabilities,
		ToolsEnabled:         req.ToolsEnabled,
		ContextSources:       req.ContextSources,
		ThinkingEnabled:      req.ThinkingEnabled,
		StreamingEnabled:     req.StreamingEnabled,
		ApplyPersonalization: req.ApplyPersonalization,
		WelcomeMessage:       req.WelcomeMessage,
		SuggestedPrompts:     req.SuggestedPrompts,
		Category:             req.Category,
		IsActive:             req.IsActive,
		IsPublic:             req.IsPublic,
		IsFeatured:           req.IsFeatured,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agent})
}

// DeleteCustomAgent deletes a custom agent
func (h *AgentHandler) DeleteCustomAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "agent_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	err = queries.DeleteCustomAgent(ctx, sqlc.DeleteCustomAgentParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
