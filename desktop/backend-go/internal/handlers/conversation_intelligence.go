package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ConversationIntelligenceHandler handles conversation analysis operations
type ConversationIntelligenceHandler struct {
	intelligence    *services.ConversationIntelligenceService
	memoryExtractor *services.MemoryExtractorService
}

// NewConversationIntelligenceHandler creates a new conversation intelligence handler
func NewConversationIntelligenceHandler(
	intelligence *services.ConversationIntelligenceService,
	memoryExtractor *services.MemoryExtractorService,
) *ConversationIntelligenceHandler {
	return &ConversationIntelligenceHandler{
		intelligence:    intelligence,
		memoryExtractor: memoryExtractor,
	}
}

// AnalyzeConversation analyzes a conversation
func (h *ConversationIntelligenceHandler) AnalyzeConversation(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := user.ID

	var req struct {
		ConversationID string `json:"conversation_id"`
		Messages       []struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp,omitempty"`
		} `json:"messages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Convert to service message type
	messages := make([]services.Message, len(req.Messages))
	for i, m := range req.Messages {
		ts := time.Now()
		if m.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, m.Timestamp); err == nil {
				ts = parsed
			}
		}
		messages[i] = services.Message{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: ts,
		}
	}

	analysis, err := h.intelligence.AnalyzeConversation(ctx, req.ConversationID, userID, messages)
	if err != nil {
		utils.InternalError(slog.Default(), "Analysis failed").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetConversationAnalysis retrieves analysis for a conversation
func (h *ConversationIntelligenceHandler) GetConversationAnalysis(c *gin.Context) {
	ctx := c.Request.Context()
	conversationID := c.Param("id")

	analysis, err := h.intelligence.GetAnalysis(ctx, conversationID)
	if err != nil {
		utils.NotFound(slog.Default(), "Analysis").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// SearchConversations searches conversation analyses
func (h *ConversationIntelligenceHandler) SearchConversations(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := user.ID

	query := c.Query("q")
	if query == "" {
		utils.RespondBadRequest(c, slog.Default(), "Query parameter 'q' is required")
		return
	}

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := h.intelligence.SearchConversations(ctx, userID, query, limit)
	if err != nil {
		utils.InternalError(slog.Default(), "Search failed").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, results)
}

// ExtractMemories extracts memories from a conversation
func (h *ConversationIntelligenceHandler) ExtractMemories(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := user.ID

	var req struct {
		Messages []struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp,omitempty"`
		} `json:"messages"`
		Options *services.ExtractionOptions `json:"options,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Convert to service message type
	messages := make([]services.Message, len(req.Messages))
	for i, m := range req.Messages {
		ts := time.Now()
		if m.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, m.Timestamp); err == nil {
				ts = parsed
			}
		}
		messages[i] = services.Message{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: ts,
		}
	}

	result, err := h.memoryExtractor.ExtractWithLLM(ctx, userID, messages, req.Options)
	if err != nil {
		utils.InternalError(slog.Default(), "Extraction failed").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExtractFromVoiceNote extracts memories from a voice note transcript
func (h *ConversationIntelligenceHandler) ExtractFromVoiceNote(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := user.ID

	var req struct {
		Transcript string                      `json:"transcript"`
		Options    *services.ExtractionOptions `json:"options,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.Transcript == "" {
		utils.RespondBadRequest(c, slog.Default(), "Transcript is required")
		return
	}

	result, err := h.memoryExtractor.ExtractFromVoiceNote(ctx, userID, req.Transcript, req.Options)
	if err != nil {
		utils.InternalError(slog.Default(), "Extraction failed").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetExtractedMemories retrieves extracted memories
func (h *ConversationIntelligenceHandler) GetExtractedMemories(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := user.ID

	memoryType := c.Query("type")

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	memories, err := h.memoryExtractor.GetExtractedMemories(ctx, userID, memoryType, limit)
	if err != nil {
		utils.InternalError(slog.Default(), "Failed to get memories").WithError(err).Respond(c)
		return
	}

	c.JSON(http.StatusOK, memories)
}

// RegisterConversationIntelligenceRoutes registers conversation intelligence routes on a Gin router group
func RegisterConversationIntelligenceRoutes(r *gin.RouterGroup, handler *ConversationIntelligenceHandler) {
	intel := r.Group("/intelligence")
	{
		// Conversation analysis
		intel.POST("/analyze", handler.AnalyzeConversation)
		intel.GET("/conversations/:id", handler.GetConversationAnalysis)
		intel.GET("/conversations/search", handler.SearchConversations)

		// Memory extraction
		intel.POST("/extract/conversation", handler.ExtractMemories)
		intel.POST("/extract/voice-note", handler.ExtractFromVoiceNote)
		intel.GET("/memories", handler.GetExtractedMemories)
	}
}
