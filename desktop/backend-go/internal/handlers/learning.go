package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// LearningHandler handles learning and personalization operations
type LearningHandler struct {
	learning *services.LearningService
}

// NewLearningHandler creates a new learning handler
func NewLearningHandler(learning *services.LearningService) *LearningHandler {
	return &LearningHandler{
		learning: learning,
	}
}

// RecordFeedback records user feedback
func (h *LearningHandler) RecordFeedback(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req struct {
		TargetType      string `json:"target_type"` // message, artifact, memory, suggestion, agent_response
		TargetID        string `json:"target_id"`
		FeedbackType    string `json:"feedback_type"` // thumbs_up, thumbs_down, correction, comment, rating
		FeedbackValue   string `json:"feedback_value,omitempty"`
		Rating          *int   `json:"rating,omitempty"`
		ConversationID  string `json:"conversation_id,omitempty"`
		AgentType       string `json:"agent_type,omitempty"`
		FocusMode       string `json:"focus_mode,omitempty"`
		OriginalContent string `json:"original_content,omitempty"`
		ExpectedContent string `json:"expected_content,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "target_id")
		return
	}

	var conversationID *uuid.UUID
	if req.ConversationID != "" {
		if id, err := uuid.Parse(req.ConversationID); err == nil {
			conversationID = &id
		}
	}

	input := services.FeedbackInput{
		UserID:          userID,
		TargetType:      req.TargetType,
		TargetID:        targetID,
		FeedbackType:    req.FeedbackType,
		FeedbackValue:   req.FeedbackValue,
		Rating:          req.Rating,
		ConversationID:  conversationID,
		AgentType:       req.AgentType,
		FocusMode:       req.FocusMode,
		OriginalContent: req.OriginalContent,
		ExpectedContent: req.ExpectedContent,
	}

	feedback, err := h.learning.RecordFeedback(ctx, input)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "record feedback", err)
		return
	}

	c.JSON(http.StatusOK, feedback)
}

// ObserveBehavior records a user behavior observation
func (h *LearningHandler) ObserveBehavior(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req struct {
		PatternType  string `json:"pattern_type"`
		PatternKey   string `json:"pattern_key"`
		PatternValue string `json:"pattern_value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if err := h.learning.ObserveBehavior(ctx, userID, req.PatternType, req.PatternKey, req.PatternValue); err != nil {
		utils.RespondInternalError(c, slog.Default(), "observe behavior", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "observed"})
}

// GetPersonalizationProfile retrieves user's personalization profile
func (h *LearningHandler) GetPersonalizationProfile(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	profile, err := h.learning.GetPersonalizationProfile(ctx, userID)
	if err != nil {
		// If profile doesn't exist, return default empty profile instead of error
		if errors.Is(err, pgx.ErrNoRows) {
			defaultProfile := &services.PersonalizationProfile{
				UserID:                userID,
				PreferredTone:         "professional",
				PreferredVerbosity:    "balanced",
				PreferredFormat:       "structured",
				PrefersExamples:       true,
				PrefersAnalogies:      false,
				PrefersCodeSamples:    false,
				PrefersVisualAids:     false,
				ExpertiseAreas:        []string{},
				LearningAreas:         []string{},
				CommonTopics:          []string{},
				MostActiveHours:       []int{},
				TotalConversations:    0,
				TotalFeedbackGiven:    0,
				PositiveFeedbackRatio: 0.5,
				ProfileCompleteness:   0,
			}
			c.JSON(http.StatusOK, defaultProfile)
			return
		}
		utils.RespondInternalError(c, slog.Default(), "get profile", err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates the user's personalization profile
func (h *LearningHandler) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var profile services.PersonalizationProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	profile.UserID = userID

	if err := h.learning.UpdatePersonalizationProfile(ctx, &profile); err != nil {
		utils.RespondInternalError(c, slog.Default(), "update profile", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// DetectPatterns triggers pattern detection analysis
func (h *LearningHandler) DetectPatterns(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	patterns, err := h.learning.DetectPatternsToUserFacts(ctx, userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "detect patterns", err)
		return
	}

	c.JSON(http.StatusOK, patterns)
}

// GetLearnings retrieves learnings for a specific context
func (h *LearningHandler) GetLearnings(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	agentType := c.Query("agent_type")
	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	learnings, err := h.learning.GetLearningsForContext(ctx, userID, agentType, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get learnings", err)
		return
	}

	c.JSON(http.StatusOK, learnings)
}

// ApplyLearning marks a learning as applied
func (h *LearningHandler) ApplyLearning(c *gin.Context) {
	ctx := c.Request.Context()

	learningIDStr := c.Param("id")
	learningID, err := uuid.Parse(learningIDStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "learning ID")
		return
	}

	var req struct {
		Successful bool `json:"successful"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if err := h.learning.ApplyLearning(ctx, learningID, req.Successful); err != nil {
		utils.RespondInternalError(c, slog.Default(), "apply learning", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "applied"})
}

// RefreshProfile refreshes profile from detected patterns
func (h *LearningHandler) RefreshProfile(c *gin.Context) {
	ctx := c.Request.Context()
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	if err := h.learning.RefreshProfileFromPatterns(ctx, userID); err != nil {
		utils.RespondInternalError(c, slog.Default(), "refresh profile", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "refreshed"})
}

// RegisterLearningRoutes registers learning routes on a Gin router group
func RegisterLearningRoutes(r *gin.RouterGroup, handler *LearningHandler) {
	learning := r.Group("/learning")
	{
		learning.POST("/feedback", handler.RecordFeedback)
		learning.POST("/behavior", handler.ObserveBehavior)
		learning.GET("/profile", handler.GetPersonalizationProfile)
		learning.PUT("/profile", handler.UpdateProfile)
		learning.POST("/profile/refresh", handler.RefreshProfile)
		learning.GET("/patterns", handler.DetectPatterns)
		learning.GET("/learnings", handler.GetLearnings)
		learning.POST("/learnings/:id/apply", handler.ApplyLearning)
	}
}
