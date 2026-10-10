package handlers

import (
	"context"
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

// ListAgentPresets returns all available agent presets
func (h *AgentHandler) ListAgentPresets(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	presets, err := queries.ListAgentPresets(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list presets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"presets": presets})
}

// GetAgentPreset retrieves a specific agent preset
func (h *AgentHandler) GetAgentPreset(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preset ID"})
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	preset, err := queries.GetAgentPreset(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Preset")
		return
	}

	c.JSON(http.StatusOK, gin.H{"preset": preset})
}

// CreateAgentFromPreset creates a new custom agent based on a preset
func (h *AgentHandler) CreateAgentFromPreset(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	presetIDStr := c.Param("presetId")
	presetID, err := uuid.Parse(presetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preset ID"})
		return
	}

	var req CreateAgentFromPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate agent name
	name := strings.ToLower(strings.TrimSpace(req.Name))
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Agent name can only contain lowercase letters, numbers, and hyphens"})
			return
		}
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	// Create agent from preset
	agent, err := queries.CreateAgentFromPreset(ctx, sqlc.CreateAgentFromPresetParams{
		UserID: user.ID,
		Name:   name,
		ID:     pgtype.UUID{Bytes: presetID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent from preset: " + err.Error()})
		return
	}

	// Increment preset copy count (best-effort)
	if err := queries.IncrementPresetCopyCount(ctx, pgtype.UUID{Bytes: presetID, Valid: true}); err != nil {
		slog.Warn("[Agents] Failed to increment preset copy count", "error", err)
	}

	c.JSON(http.StatusCreated, gin.H{"agent": agent})
}

// ListCustomAgentsByCategory returns custom agents filtered by category
func (h *AgentHandler) ListCustomAgentsByCategory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	agents, err := queries.ListCustomAgentsByCategory(ctx, sqlc.ListCustomAgentsByCategoryParams{
		UserID:   user.ID,
		Category: &category,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}
