package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// OnboardingHandler handles conversational AI onboarding flow endpoints.
type OnboardingHandler struct {
	onboardingService *services.OnboardingService
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(onboardingService *services.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingService: onboardingService,
	}
}

// RegisterOnboardingRoutes registers onboarding routes
func (h *OnboardingHandler) RegisterOnboardingRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	onboarding := r.Group("/onboarding")
	onboarding.Use(authMiddleware)
	{
		// Status check
		onboarding.GET("/status", h.CheckOnboardingStatus)

		// Session management
		onboarding.POST("/sessions", h.CreateSession)
		onboarding.GET("/sessions/:id", h.GetSession)
		onboarding.DELETE("/sessions/:id", h.AbandonSession)

		// Resume existing session
		onboarding.GET("/resume", h.GetResumeableSession)

		// Conversation
		onboarding.POST("/sessions/:id/messages", h.SendMessage)
		onboarding.POST("/sessions/:id/messages/stream", h.SendMessageStream) // SSE streaming
		onboarding.GET("/sessions/:id/history", h.GetConversationHistory)

		// Completion
		onboarding.PUT("/sessions/:id/complete", h.CompleteOnboarding)

		// Fallback form
		onboarding.POST("/fallback", h.SubmitFallbackForm)

		// Integration selection (during onboarding)
		onboarding.POST("/sessions/:id/integrations", h.SelectIntegrations)

		// Get integration recommendations (computed from extracted data)
		onboarding.GET("/sessions/:id/recommendations", h.GetRecommendations)
	}
}

// CheckOnboardingStatus checks if user needs onboarding
// GET /api/onboarding/status
func (h *OnboardingHandler) CheckOnboardingStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	status, err := h.onboardingService.CheckOnboardingStatus(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "check onboarding status", err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// CreateSession starts a new onboarding session
// POST /api/onboarding/sessions
func (h *OnboardingHandler) CreateSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	session, err := h.onboardingService.CreateSession(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create session", err)
		return
	}

	// Also return initial conversation history
	messages, err := h.onboardingService.GetConversationHistory(c.Request.Context(), session.ID, 0)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get conversation history", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session":  session,
		"messages": messages,
	})
}

// GetSession retrieves a session with conversation history
// GET /api/onboarding/sessions/:id
func (h *OnboardingHandler) GetSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	session, messages, err := h.onboardingService.GetSessionWithHistory(c.Request.Context(), sessionID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Session")
		return
	}

	// Verify ownership
	if session.UserID != user.ID {
		utils.RespondForbidden(c, slog.Default(), "Access denied")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  session,
		"messages": messages,
	})
}

// GetResumeableSession checks for an existing session to resume
// GET /api/onboarding/resume
func (h *OnboardingHandler) GetResumeableSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	session, err := h.onboardingService.GetResumeableSession(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get resumeable session", err)
		return
	}

	if session == nil {
		c.JSON(http.StatusOK, gin.H{
			"has_session": false,
			"session":     nil,
		})
		return
	}

	// Get last few messages for context
	messages, err := h.onboardingService.GetConversationHistory(c.Request.Context(), session.ID, 6)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get conversation history", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_session": true,
		"session":     session,
		"messages":    messages,
	})
}

// AbandonSession abandons an onboarding session
// DELETE /api/onboarding/sessions/:id
func (h *OnboardingHandler) AbandonSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	err = h.onboardingService.AbandonSession(c.Request.Context(), sessionID, user.ID)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session abandoned"})
}

// SendMessage sends a user message and gets AI response
// POST /api/onboarding/sessions/:id/messages
func (h *OnboardingHandler) SendMessage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	var req services.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	response, err := h.onboardingService.ProcessUserMessage(c.Request.Context(), sessionID, user.ID, req.Content)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetConversationHistory retrieves conversation history
// GET /api/onboarding/sessions/:id/history
func (h *OnboardingHandler) GetConversationHistory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	// Verify session ownership
	session, err := h.onboardingService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Session")
		return
	}
	if session.UserID != user.ID {
		utils.RespondForbidden(c, slog.Default(), "Access denied")
		return
	}

	messages, err := h.onboardingService.GetConversationHistory(c.Request.Context(), sessionID, 0)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get conversation history", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SelectIntegrations handles integration selection
// POST /api/onboarding/sessions/:id/integrations
func (h *OnboardingHandler) SelectIntegrations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	var req struct {
		Integrations []string `json:"integrations"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// For now, just acknowledge the selection
	// OAuth flows are handled separately
	c.JSON(http.StatusOK, gin.H{
		"message":      "Integrations selected",
		"integrations": req.Integrations,
	})
}

// CompleteOnboarding completes the onboarding and creates workspace
// PUT /api/onboarding/sessions/:id/complete
func (h *OnboardingHandler) CompleteOnboarding(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	var req struct {
		Integrations []string `json:"integrations"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req.Integrations = []string{}
	}

	result, err := h.onboardingService.CompleteOnboarding(c.Request.Context(), sessionID, user.ID, req.Integrations)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// SubmitFallbackForm handles fallback form submission
// POST /api/onboarding/fallback
func (h *OnboardingHandler) SubmitFallbackForm(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		SessionID uuid.UUID                 `json:"session_id" binding:"required"`
		Data      services.FallbackFormData `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	result, err := h.onboardingService.SubmitFallbackForm(c.Request.Context(), req.SessionID, user.ID, &req.Data)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// SendMessageStream sends a user message and streams AI response via SSE
// POST /api/onboarding/sessions/:id/messages/stream
func (h *OnboardingHandler) SendMessageStream(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	var req services.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create a channel to receive streaming updates
	ctx := c.Request.Context()

	// Process message (non-streaming for now, but sends events)
	// In future, this could stream tokens as they arrive from AI
	response, err := h.onboardingService.ProcessUserMessage(ctx, sessionID, user.ID, req.Content)
	if err != nil {
		// Send error event
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}

	// Send typing event
	c.SSEvent("typing", gin.H{"typing": true})
	c.Writer.Flush()

	// Send the agent message in chunks to simulate streaming
	message := response.Message.Content
	chunkSize := 20 // characters per chunk
	for i := 0; i < len(message); i += chunkSize {
		end := i + chunkSize
		if end > len(message) {
			end = len(message)
		}
		chunk := message[i:end]
		c.SSEvent("token", gin.H{"token": chunk})
		c.Writer.Flush()
	}

	// Send complete event with full response
	c.SSEvent("complete", response)
	c.Writer.Flush()

	// Send done event
	c.SSEvent("done", gin.H{"done": true})
	c.Writer.Flush()
}

// GetRecommendations returns integration recommendations based on extracted data
// GET /api/onboarding/sessions/:id/recommendations
func (h *OnboardingHandler) GetRecommendations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "session ID")
		return
	}

	recommendations, err := h.onboardingService.GetRecommendations(c.Request.Context(), sessionID, user.ID)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": recommendations,
	})
}
