package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AutoLearningTriggers handles automatic learning from conversations
type AutoLearningTriggers struct {
	learningSvc *LearningService
	memorySvc   *MemoryService
	embeddingSvc *EmbeddingService
	logger      *slog.Logger
}

// NewAutoLearningTriggers creates a new auto-learning triggers service
func NewAutoLearningTriggers(
	learningSvc *LearningService,
	memorySvc *MemoryService,
	embeddingSvc *EmbeddingService,
) *AutoLearningTriggers {
	return &AutoLearningTriggers{
		learningSvc:  learningSvc,
		memorySvc:    memorySvc,
		embeddingSvc: embeddingSvc,
		logger:       slog.Default().With("service", "auto_learning"),
	}
}

// LearningConversationContext holds context for learning from a conversation
type LearningConversationContext struct {
	UserID          string
	WorkspaceID     *uuid.UUID // Added for workspace_memories support
	ConversationID  uuid.UUID
	UserMessage     string
	AgentResponse   string
	AgentType       string
	FocusMode       string
	ProjectID       *uuid.UUID
	NodeID          *uuid.UUID
	ContextIDs      []uuid.UUID
	Timestamp       time.Time
}

// ProcessConversationTurn analyzes a conversation turn and extracts learnings
// This runs asynchronously after each agent response
func (a *AutoLearningTriggers) ProcessConversationTurn(ctx context.Context, conv LearningConversationContext) error {
	a.logger.Info("Processing conversation for auto-learning",
		"user_id", conv.UserID,
		"conv_id", conv.ConversationID,
		"agent_type", conv.AgentType,
	)

	// Run learning extraction in background
	// Use background context to avoid cancellation when HTTP request completes
	go func() {
		// Create independent context that won't be canceled when HTTP request ends
		bgCtx := context.Background()

		// 1. Extract patterns
		if err := a.extractPatterns(bgCtx, conv); err != nil {
			a.logger.Error("Failed to extract patterns", "err", err)
		}

		// 2. Detect preferences
		if err := a.detectPreferences(bgCtx, conv); err != nil {
			a.logger.Error("Failed to detect preferences", "err", err)
		}

		// 3. Create memory if significant
		if err := a.createMemoryIfSignificant(bgCtx, conv); err != nil {
			a.logger.Error("Failed to create memory", "err", err)
		}

		// 4. Extract facts
		if err := a.extractFacts(bgCtx, conv); err != nil {
			a.logger.Error("Failed to extract facts", "err", err)
		}
	}()

	return nil
}

// extractPatterns identifies behavioral patterns from the conversation
func (a *AutoLearningTriggers) extractPatterns(ctx context.Context, conv LearningConversationContext) error {
	// Pattern: Topic interest
	topics := a.extractTopics(conv.UserMessage)
	if len(topics) > 0 {
		// Record topic interest pattern
		for _, topic := range topics {
			if err := a.recordBehaviorPattern(ctx, conv.UserID, "topic_interest", topic, fmt.Sprintf("User shows interest in %s", topic)); err != nil {
				a.logger.Warn("Failed to record topic pattern", "topic", topic, "err", err)
			}
		}
	}

	// Pattern: Communication style
	if isAskingQuestion := strings.Contains(conv.UserMessage, "?"); isAskingQuestion {
		a.recordBehaviorPattern(ctx, conv.UserID, "communication_style", "questions", "User prefers asking questions")
	}

	// Pattern: Focus mode preference
	if conv.FocusMode != "" {
		a.recordBehaviorPattern(ctx, conv.UserID, "focus_preference", conv.FocusMode, fmt.Sprintf("User uses %s focus mode", conv.FocusMode))
	}

	// Pattern: Agent type preference
	if conv.AgentType != "" {
		a.recordBehaviorPattern(ctx, conv.UserID, "agent_preference", conv.AgentType, fmt.Sprintf("User engages with %s agent", conv.AgentType))
	}

	return nil
}

// detectPreferences identifies user preferences from conversation
func (a *AutoLearningTriggers) detectPreferences(ctx context.Context, conv LearningConversationContext) error {
	// Preference: Detailed vs concise responses
	if len(conv.AgentResponse) > 1000 {
		// User got a detailed response - track if they continue conversation
		a.logger.Debug("Detected detailed response preference signal")
	}

	// Preference: Code examples
	if strings.Contains(conv.UserMessage, "example") || strings.Contains(conv.UserMessage, "show me") {
		a.recordLearning(ctx, conv.UserID, "preference", "User prefers examples", "explicit_behavior", nil)
	}

	// Preference: Step-by-step instructions
	if strings.Contains(conv.UserMessage, "step") || strings.Contains(conv.UserMessage, "how to") {
		a.recordLearning(ctx, conv.UserID, "preference", "User prefers step-by-step instructions", "explicit_behavior", nil)
	}

	return nil
}

// createMemoryIfSignificant creates a memory if the conversation is important
func (a *AutoLearningTriggers) createMemoryIfSignificant(ctx context.Context, conv LearningConversationContext) error {
	// Significance heuristics:
	significance := 0.0

	// 1. Length of conversation (longer = more significant)
	if len(conv.UserMessage) > 200 {
		significance += 0.2
	}
	if len(conv.AgentResponse) > 500 {
		significance += 0.3
	}

	// 2. Presence of questions (indicates information seeking)
	if strings.Contains(conv.UserMessage, "?") {
		significance += 0.2
	}

	// 3. Project/Node context (work-related = more significant)
	if conv.ProjectID != nil || conv.NodeID != nil {
		significance += 0.3
	}

	// 4. Technical terms (indicates specialized knowledge)
	technicalTerms := []string{"database", "api", "function", "code", "implement", "architecture", "schema"}
	for _, term := range technicalTerms {
		if strings.Contains(strings.ToLower(conv.UserMessage), term) {
			significance += 0.1
			break
		}
	}

	a.logger.Debug("Memory significance calculated", "score", significance)

	// Only create memory if significance > threshold
	if significance >= 0.5 {
		// Generate title from user message
		title := a.generateMemoryTitle(conv.UserMessage)

		// Create summary from response
		summary := conv.AgentResponse
		if len(summary) > 200 {
			summary = summary[:197] + "..."
		}

		// Determine memory type
		memoryType := "interaction"
		if conv.ProjectID != nil {
			memoryType = "project_context"
		} else if strings.Contains(conv.UserMessage, "?") {
			memoryType = "fact"
		}

		// Determine scope type
		scopeType := "workspace"
		var scopeID *uuid.UUID
		if conv.ProjectID != nil {
			scopeType = "project"
			scopeID = conv.ProjectID
		} else if conv.NodeID != nil {
			scopeType = "node"
			scopeID = conv.NodeID
		}

		// Use workspace_memories if workspace_id is available
		if conv.WorkspaceID != nil {
			// Create workspace memory (new table)
			workspaceMemory := WorkspaceMemoryRequest{
				WorkspaceID:     *conv.WorkspaceID,
				UserID:          conv.UserID,
				Title:           title,
				Summary:         summary,
				Content:         fmt.Sprintf("User: %s\n\nAgent: %s", conv.UserMessage, conv.AgentResponse),
				MemoryType:      memoryType,
				Visibility:      "private", // Auto-generated memories are private by default
				Tags:            a.extractTags(conv.UserMessage),
				ImportanceScore: significance,
				ScopeType:       &scopeType,
				ScopeID:         scopeID,
			}

			_, err := a.memorySvc.CreateWorkspaceMemory(ctx, workspaceMemory)
			if err != nil {
				return fmt.Errorf("failed to create workspace memory: %w", err)
			}

			a.logger.Info("Created workspace memory from conversation",
				"user_id", conv.UserID,
				"workspace_id", conv.WorkspaceID,
				"title", title,
				"significance", significance,
				"visibility", "private",
			)
		} else {
			// Fallback to legacy memories table if no workspace_id
			memory := &Memory{
				UserID:          conv.UserID,
				Title:           title,
				Summary:         summary,
				Content:         fmt.Sprintf("User: %s\n\nAgent: %s", conv.UserMessage, conv.AgentResponse),
				MemoryType:      memoryType,
				SourceType:      "conversation",
				SourceID:        &conv.ConversationID,
				ProjectID:       conv.ProjectID,
				NodeID:          conv.NodeID,
				ImportanceScore: significance,
				Tags:            a.extractTags(conv.UserMessage),
			}

			if err := a.memorySvc.CreateMemory(ctx, memory); err != nil {
				return fmt.Errorf("failed to create memory: %w", err)
			}

			a.logger.Info("Created legacy memory from conversation (no workspace_id)",
				"user_id", conv.UserID,
				"title", title,
				"significance", significance,
			)
		}
	}

	return nil
}

// extractFacts extracts factual information from conversation
func (a *AutoLearningTriggers) extractFacts(ctx context.Context, conv LearningConversationContext) error {
	// Pattern: "I prefer X" or "I like X"
	preferencePatterns := []string{
		`I prefer ([^.!?]+)`,
		`I like ([^.!?]+)`,
		`I usually ([^.!?]+)`,
		`I always ([^.!?]+)`,
		`My ([^.!?]+) is ([^.!?]+)`,
	}

	for _, pattern := range preferencePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(conv.UserMessage); len(matches) > 1 {
			fact := matches[1]
			a.recordUserFact(ctx, conv.UserID, "preference", fact)
		}
	}

	// Pattern: "My name is X"
	namePattern := regexp.MustCompile(`(?i)my name is ([A-Za-z]+)`)
	if matches := namePattern.FindStringSubmatch(conv.UserMessage); len(matches) > 1 {
		name := matches[1]
		a.recordUserFact(ctx, conv.UserID, "name", name)
	}

	return nil
}

// Helper methods

func (a *AutoLearningTriggers) extractTopics(message string) []string {
	// Simple topic extraction based on keywords
	topics := []string{}
	keywords := map[string]string{
		"database": "databases",
		"api":      "api_development",
		"react":    "frontend",
		"go":       "backend",
		"design":   "design",
		"test":     "testing",
	}

	lowerMsg := strings.ToLower(message)
	for keyword, topic := range keywords {
		if strings.Contains(lowerMsg, keyword) {
			topics = append(topics, topic)
		}
	}

	return topics
}

func (a *AutoLearningTriggers) generateMemoryTitle(message string) string {
	// Take first sentence or first 50 chars
	firstSentence := strings.Split(message, ".")[0]
	firstSentence = strings.Split(firstSentence, "?")[0]
	firstSentence = strings.Split(firstSentence, "!")[0]

	if len(firstSentence) > 50 {
		return firstSentence[:47] + "..."
	}
	return firstSentence
}

func (a *AutoLearningTriggers) extractTags(message string) []string {
	tags := []string{}

	// Extract hashtags if present
	hashtagPattern := regexp.MustCompile(`#(\w+)`)
	matches := hashtagPattern.FindAllStringSubmatch(message, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	return tags
}

// recordBehaviorPattern records or updates a behavior pattern
func (a *AutoLearningTriggers) recordBehaviorPattern(ctx context.Context, userID, patternType, patternKey, description string) error {
	a.logger.Debug("Recording behavior pattern",
		"user_id", userID,
		"type", patternType,
		"key", patternKey,
	)

	// Use LearningService.ObserveBehavior to record the pattern
	// This handles the INSERT/UPDATE logic with ON CONFLICT
	if a.learningSvc != nil {
		err := a.learningSvc.ObserveBehavior(ctx, userID, patternType, patternKey, description)
		if err != nil {
			a.logger.Error("Failed to record behavior pattern",
				"error", err,
				"user_id", userID,
				"pattern_type", patternType,
			)
			return fmt.Errorf("failed to record behavior pattern: %w", err)
		}
		a.logger.Info("Behavior pattern recorded successfully",
			"user_id", userID,
			"pattern_type", patternType,
			"pattern_key", patternKey,
		)
	}

	return nil
}

// recordLearning records a learning event
func (a *AutoLearningTriggers) recordLearning(ctx context.Context, userID, learningType, content, sourceType string, sourceID *uuid.UUID) error {
	a.logger.Debug("Recording learning event",
		"user_id", userID,
		"type", learningType,
		"content", content,
	)

	// Record learning event to database
	if a.learningSvc != nil {
		learning := &LearningEvent{
			ID:              uuid.New(),
			UserID:          userID,
			LearningType:    learningType,
			LearningContent: content,
			LearningSummary: fmt.Sprintf("Auto-learned %s from conversation", learningType),
			SourceType:      sourceType,
			SourceID:        sourceID,
			ConfidenceScore: 0.6, // Medium confidence for auto-learning
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Insert into learning_events table
		_, err := a.learningSvc.pool.Exec(ctx, `
			INSERT INTO learning_events (
				id, user_id, learning_type, learning_content, learning_summary,
				source_type, source_id, confidence_score, is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, learning.ID, learning.UserID, learning.LearningType, learning.LearningContent,
			learning.LearningSummary, learning.SourceType, learning.SourceID,
			learning.ConfidenceScore, learning.IsActive, learning.CreatedAt, learning.UpdatedAt)

		if err != nil {
			a.logger.Error("Failed to record learning event",
				"error", err,
				"user_id", userID,
				"learning_type", learningType,
			)
			return fmt.Errorf("failed to record learning event: %w", err)
		}

		a.logger.Info("Learning event recorded successfully",
			"user_id", userID,
			"learning_id", learning.ID,
			"learning_type", learningType,
		)
	}

	return nil
}

// recordUserFact records a user fact
func (a *AutoLearningTriggers) recordUserFact(ctx context.Context, userID, factType, value string) error {
	a.logger.Info("Recording user fact",
		"user_id", userID,
		"type", factType,
		"value", value,
	)

	// Generate normalized fact key from type
	factKey := fmt.Sprintf("%s", strings.ToLower(strings.ReplaceAll(factType, " ", "_")))

	// Insert or update user fact in database
	if a.learningSvc != nil {
		_, err := a.learningSvc.pool.Exec(ctx, `
			INSERT INTO user_facts (
				user_id, fact_key, fact_value, fact_type, confidence_score,
				is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, 0.7, true, NOW(), NOW())
			ON CONFLICT (user_id, fact_key)
			DO UPDATE SET
				fact_value = EXCLUDED.fact_value,
				fact_type = EXCLUDED.fact_type,
				confidence_score = LEAST(1.0, user_facts.confidence_score + 0.1),
				updated_at = NOW(),
				last_confirmed_at = NOW()
		`, userID, factKey, value, factType)

		if err != nil {
			a.logger.Error("Failed to record user fact",
				"error", err,
				"user_id", userID,
				"fact_type", factType,
			)
			return fmt.Errorf("failed to record user fact: %w", err)
		}

		a.logger.Info("User fact recorded successfully",
			"user_id", userID,
			"fact_key", factKey,
			"fact_type", factType,
		)
	}

	return nil
}
