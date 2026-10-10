// Package services provides the ConversationIntelligenceService which analyses
// conversation message streams and persists structured summaries.
//
// The implementation is split across three files:
//   - conversation_intelligence.go        — service struct, constructor, orchestration
//   - conversation_intelligence_types.go  — all public/private data types
//   - conversation_intelligence_analysis.go — extraction and scoring helpers
//   - conversation_intelligence_persistence.go — DB read/write/search + backfill
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConversationIntelligenceService provides intelligent analysis of conversations.
type ConversationIntelligenceService struct {
	pool   *pgxpool.Pool
	embed  *EmbeddingService
	logger *slog.Logger
}

// NewConversationIntelligenceService creates a new ConversationIntelligenceService.
func NewConversationIntelligenceService(pool *pgxpool.Pool, embeddingService *EmbeddingService) *ConversationIntelligenceService {
	return &ConversationIntelligenceService{
		pool:   pool,
		embed:  embeddingService,
		logger: slog.Default().With("service", "conversation_intelligence"),
	}
}

// AnalyzeConversation performs a full analysis pass on messages and persists the result.
// The returned ConversationAnalysis is populated even when persistence fails — the
// warning is logged and the caller receives the in-memory result.
func (s *ConversationIntelligenceService) AnalyzeConversation(ctx context.Context, conversationID, userID string, messages []Message) (*ConversationAnalysis, error) {
	analysis := &ConversationAnalysis{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		UserID:         userID,
		KeyPoints:      make([]string, 0),
		Topics:         make([]ConversationTopic, 0),
		Entities:       make([]ConversationEntity, 0),
		ActionItems:    make([]ActionItem, 0),
		Questions:      make([]Question, 0),
		Decisions:      make([]ConversationDecision, 0),
		CodeMentions:   make([]CodeMention, 0),
		Metadata:       make(map[string]interface{}),
		MessageCount:   len(messages),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if len(messages) == 0 {
		return analysis, nil
	}

	// Duration of the conversation window.
	if len(messages) > 1 {
		duration := messages[len(messages)-1].Timestamp.Sub(messages[0].Timestamp)
		analysis.Duration = duration.Round(time.Second).String()
	}
	analysis.Metadata["time_range_start"] = messages[0].Timestamp
	analysis.Metadata["time_range_end"] = messages[len(messages)-1].Timestamp

	// Run all extraction passes (implemented in conversation_intelligence_analysis.go).
	analysis.Topics = s.extractTopics(messages)
	analysis.Entities = s.extractEntities(messages)
	analysis.Questions = s.extractQuestions(messages)
	analysis.ActionItems = s.extractActionItems(messages)
	analysis.Decisions = s.extractDecisions(messages)
	analysis.CodeMentions = s.extractCodeMentions(messages)
	analysis.Sentiment = s.analyzeSentiment(messages)
	analysis.Title = s.generateTitle(messages, analysis.Topics)
	analysis.Summary = s.generateSummary(messages, analysis)
	analysis.KeyPoints = s.extractKeyPoints(messages, analysis)

	// Approximate token count (4 chars ≈ 1 token).
	for _, msg := range messages {
		analysis.TokenCount += len(msg.Content) / 4
	}

	// Persist (implemented in conversation_intelligence_persistence.go).
	if err := s.saveAnalysis(ctx, analysis); err != nil {
		s.logger.Warn("failed to save conversation analysis", "error", err)
	}

	return analysis, nil
}

// BackfillStaleSummaries generates or refreshes conversation summaries that are
// missing or older than the conversation's latest message timestamp.
// When force is true it re-analyses the most recent conversations unconditionally.
// Returns the number of conversations successfully processed.
func (s *ConversationIntelligenceService) BackfillStaleSummaries(ctx context.Context, limit int, maxMessages int, force bool) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	if maxMessages <= 0 {
		maxMessages = 200
	}

	rows, err := s.pool.Query(ctx, `
		WITH last_msg AS (
			SELECT conversation_id, MAX(created_at) AS last_at
			FROM messages
			GROUP BY conversation_id
		)
		SELECT c.id::text, c.user_id
		FROM conversations c
		JOIN last_msg lm ON lm.conversation_id = c.id
		LEFT JOIN conversation_summaries cs ON cs.conversation_id = c.id
		WHERE ($1::boolean = true) OR cs.conversation_id IS NULL OR cs.updated_at < lm.last_at
		ORDER BY lm.last_at DESC
		LIMIT $2
	`, force, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var conversationID, userID string
		if err := rows.Scan(&conversationID, &userID); err != nil {
			continue
		}

		messages, err := s.fetchConversationMessages(ctx, conversationID, maxMessages)
		if err != nil || len(messages) == 0 {
			continue
		}

		if _, err = s.AnalyzeConversation(ctx, conversationID, userID, messages); err != nil {
			s.logger.Warn("conversation analysis failed", "conversation_id", conversationID, "error", err)
			continue
		}
		processed++
	}

	return processed, nil
}
