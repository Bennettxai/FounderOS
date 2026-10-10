package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ThinkingHandler handles thinking traces, reasoning templates, and thinking settings.
type ThinkingHandler struct {
	pool *pgxpool.Pool
}

// NewThinkingHandler creates a new ThinkingHandler.
func NewThinkingHandler(pool *pgxpool.Pool) *ThinkingHandler {
	return &ThinkingHandler{pool: pool}
}

// RegisterThinkingRoutes registers all thinking/reasoning routes under the given API group.
func RegisterThinkingRoutes(api *gin.RouterGroup, h *ThinkingHandler, auth gin.HandlerFunc) {
	thinking := api.Group("/thinking")
	thinking.Use(auth, middleware.RequireAuth())
	{
		thinking.GET("/traces/:conversationId", h.ListThinkingTraces)
		thinking.GET("/trace/:messageId", h.GetThinkingTraceByMessage)
		thinking.DELETE("/traces/:conversationId", h.DeleteThinkingTraces)
		thinking.GET("/settings", h.GetThinkingSettings)
		thinking.PUT("/settings", h.UpdateThinkingSettings)
	}

	reasoning := api.Group("/reasoning")
	reasoning.Use(auth, middleware.RequireAuth())
	{
		reasoning.GET("/templates", h.ListReasoningTemplates)
		reasoning.POST("/templates", h.CreateReasoningTemplate)
		reasoning.GET("/templates/:id", h.GetReasoningTemplate)
		reasoning.PUT("/templates/:id", h.UpdateReasoningTemplate)
		reasoning.DELETE("/templates/:id", h.DeleteReasoningTemplate)
		reasoning.POST("/templates/:id/default", h.SetDefaultReasoningTemplate)
	}
}

// =========================================================
// THINKING TRACES HANDLERS
// =========================================================

// ListThinkingTraces returns thinking traces for a conversation.
func (h *ThinkingHandler) ListThinkingTraces(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	conversationID := c.Param("conversationId")

	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	traces, err := queries.ListThinkingTracesByConversation(ctx, sqlc.ListThinkingTracesByConversationParams{
		ConversationID: pgtype.UUID{Bytes: convUUID, Valid: true},
		UserID:         userID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch thinking traces", err)
		return
	}

	c.JSON(http.StatusOK, traces)
}

// GetThinkingTraceByMessage returns thinking trace for a specific message.
func (h *ThinkingHandler) GetThinkingTraceByMessage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	messageID := c.Param("messageId")

	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "message ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	traces, err := queries.ListThinkingTracesByMessage(ctx, pgtype.UUID{Bytes: msgUUID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch thinking traces", err)
		return
	}

	// Filter by user_id in application layer since query doesn't have it
	var filteredTraces []sqlc.ThinkingTrace
	for _, t := range traces {
		if t.UserID == userID {
			filteredTraces = append(filteredTraces, t)
		}
	}

	c.JSON(http.StatusOK, filteredTraces)
}

// DeleteThinkingTraces deletes all thinking traces for a conversation.
func (h *ThinkingHandler) DeleteThinkingTraces(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	conversationID := c.Param("conversationId")

	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	err = queries.DeleteThinkingTracesByConversation(ctx, sqlc.DeleteThinkingTracesByConversationParams{
		ConversationID: pgtype.UUID{Bytes: convUUID, Valid: true},
		UserID:         userID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete thinking traces", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Thinking traces deleted"})
}

// =========================================================
// REASONING TEMPLATES HANDLERS
// =========================================================

type CreateReasoningTemplateRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	SystemPrompt        string `json:"system_prompt"`
	ThinkingInstruction string `json:"thinking_instruction"`
	OutputFormat        string `json:"output_format"`
	ShowThinking        bool   `json:"show_thinking"`
	SaveThinking        bool   `json:"save_thinking"`
	MaxThinkingTokens   int32  `json:"max_thinking_tokens"`
	IsDefault           bool   `json:"is_default"`
}

type UpdateReasoningTemplateRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	SystemPrompt        string `json:"system_prompt"`
	ThinkingInstruction string `json:"thinking_instruction"`
	OutputFormat        string `json:"output_format"`
	ShowThinking        bool   `json:"show_thinking"`
	SaveThinking        bool   `json:"save_thinking"`
	MaxThinkingTokens   int32  `json:"max_thinking_tokens"`
}

// ListReasoningTemplates returns all reasoning templates for the user.
func (h *ThinkingHandler) ListReasoningTemplates(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	templates, err := queries.ListReasoningTemplates(ctx, userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch reasoning templates", err)
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetReasoningTemplate returns a specific reasoning template.
func (h *ThinkingHandler) GetReasoningTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	templateID := c.Param("id")

	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	template, err := queries.GetReasoningTemplate(ctx, sqlc.GetReasoningTemplateParams{
		ID:     pgtype.UUID{Bytes: templateUUID, Valid: true},
		UserID: userID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Template")
		return
	}

	c.JSON(http.StatusOK, template)
}

// CreateReasoningTemplate creates a new reasoning template.
func (h *ThinkingHandler) CreateReasoningTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req CreateReasoningTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Default values
	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "streaming"
	}
	maxTokens := req.MaxThinkingTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)

	// If setting as default, clear existing default first
	if req.IsDefault {
		if err := queries.ClearDefaultTemplate(ctx, userID); err != nil {
			slog.Default().Warn("Failed to clear default template", "error", err, "user_id", userID)
		}
	}

	template, err := queries.CreateReasoningTemplate(ctx, sqlc.CreateReasoningTemplateParams{
		UserID:              userID,
		Name:                req.Name,
		Description:         toNullString(req.Description),
		SystemPrompt:        toNullString(req.SystemPrompt),
		ThinkingInstruction: toNullString(req.ThinkingInstruction),
		OutputFormat:        toNullString(outputFormat),
		ShowThinking:        toBoolPointer(req.ShowThinking),
		SaveThinking:        toBoolPointer(req.SaveThinking),
		MaxThinkingTokens:   toNullInt32(maxTokens),
		IsDefault:           toBoolPointer(req.IsDefault),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create template", err)
		return
	}

	c.JSON(http.StatusCreated, template)
}

// UpdateReasoningTemplate updates a reasoning template.
func (h *ThinkingHandler) UpdateReasoningTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	templateID := c.Param("id")

	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template ID")
		return
	}

	var req UpdateReasoningTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "streaming"
	}
	maxTokens := req.MaxThinkingTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	template, err := queries.UpdateReasoningTemplate(ctx, sqlc.UpdateReasoningTemplateParams{
		ID:                  pgtype.UUID{Bytes: templateUUID, Valid: true},
		Name:                req.Name,
		Description:         toNullString(req.Description),
		SystemPrompt:        toNullString(req.SystemPrompt),
		ThinkingInstruction: toNullString(req.ThinkingInstruction),
		OutputFormat:        toNullString(outputFormat),
		ShowThinking:        toBoolPointer(req.ShowThinking),
		SaveThinking:        toBoolPointer(req.SaveThinking),
		MaxThinkingTokens:   toNullInt32(maxTokens),
		UserID:              userID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update template", err)
		return
	}

	c.JSON(http.StatusOK, template)
}

// DeleteReasoningTemplate deletes a reasoning template.
func (h *ThinkingHandler) DeleteReasoningTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	templateID := c.Param("id")

	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	err = queries.DeleteReasoningTemplate(ctx, sqlc.DeleteReasoningTemplateParams{
		ID:     pgtype.UUID{Bytes: templateUUID, Valid: true},
		UserID: userID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete template", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}

// SetDefaultReasoningTemplate sets a template as the default.
func (h *ThinkingHandler) SetDefaultReasoningTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	templateID := c.Param("id")

	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	err = queries.SetDefaultReasoningTemplate(ctx, sqlc.SetDefaultReasoningTemplateParams{
		UserID: userID,
		ID:     pgtype.UUID{Bytes: templateUUID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "set default template", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default template updated"})
}

// =========================================================
// THINKING SETTINGS HANDLERS
// =========================================================

type UpdateThinkingSettingsRequest struct {
	Enabled           bool    `json:"enabled"`
	ShowInUI          bool    `json:"show_in_ui"`
	SaveTraces        bool    `json:"save_traces"`
	DefaultTemplateID *string `json:"default_template_id"`
	MaxTokens         int32   `json:"max_tokens"`
}

// GetThinkingSettings returns the user's thinking settings.
func (h *ThinkingHandler) GetThinkingSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)
	settings, err := queries.GetThinkingSettings(ctx, userID)
	if err != nil {
		// Return defaults if no settings exist
		c.JSON(http.StatusOK, gin.H{
			"enabled":             false,
			"show_in_ui":          true,
			"save_traces":         true,
			"default_template_id": nil,
			"max_tokens":          4096,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":             settings.ThinkingEnabled,
		"show_in_ui":          settings.ThinkingShowInUi,
		"save_traces":         settings.ThinkingSaveTraces,
		"default_template_id": settings.ThinkingDefaultTemplateID,
		"max_tokens":          settings.ThinkingMaxTokens,
	})
}

// UpdateThinkingSettings updates the user's thinking settings.
func (h *ThinkingHandler) UpdateThinkingSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req UpdateThinkingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	queries := sqlc.New(h.pool)

	var templateID pgtype.UUID
	if req.DefaultTemplateID != nil && *req.DefaultTemplateID != "" {
		if parsed, err := uuid.Parse(*req.DefaultTemplateID); err == nil {
			templateID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	_, err := queries.UpdateThinkingSettings(ctx, sqlc.UpdateThinkingSettingsParams{
		UserID:                    userID,
		ThinkingEnabled:           toBoolPointer(req.Enabled),
		ThinkingShowInUi:          toBoolPointer(req.ShowInUI),
		ThinkingSaveTraces:        toBoolPointer(req.SaveTraces),
		ThinkingDefaultTemplateID: templateID,
		ThinkingMaxTokens:         toNullInt32(req.MaxTokens),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update thinking settings", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":             req.Enabled,
		"show_in_ui":          req.ShowInUI,
		"save_traces":         req.SaveTraces,
		"default_template_id": req.DefaultTemplateID,
		"max_tokens":          req.MaxTokens,
	})
}

// Helper functions (package-level, shared within handlers package)
func toNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toBoolPointer(b bool) *bool {
	return &b
}

func toNullInt32(i int32) *int32 {
	if i == 0 {
		return nil
	}
	return &i
}
