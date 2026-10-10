package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// saveThinkingTrace saves thinking content to the database
func saveThinkingTrace(ctx context.Context, pool *pgxpool.Pool, userID string, conversationID uuid.UUID, thinkingContent string, model string, startTime time.Time) {
	if thinkingContent == "" {
		return
	}

	queries := sqlc.New(pool)

	// Estimate token count (rough approximation)
	thinkingTokens := int32(len(thinkingContent) / 4)
	stepNumber := int32(1)

	// Create thinking trace
	_, err := queries.CreateThinkingTrace(ctx, sqlc.CreateThinkingTraceParams{
		UserID:          userID,
		ConversationID:  pgtype.UUID{Bytes: conversationID, Valid: true},
		MessageID:       pgtype.UUID{Valid: false}, // Will be set later if needed
		ThinkingContent: thinkingContent,
		ThinkingType: sqlc.NullThinkingtype{
			Thinkingtype: sqlc.ThinkingtypeAnalysis,
			Valid:        true,
		},
		StepNumber: &stepNumber,
		StartedAt: pgtype.Timestamptz{
			Time:  startTime,
			Valid: true,
		},
		ThinkingTokens:      &thinkingTokens,
		ModelUsed:           &model,
		ReasoningTemplateID: pgtype.UUID{Valid: false},
		Metadata:            nil,
	})

	if err != nil {
		slog.Error("ChatV2: Failed to save thinking trace", "err", err)
	} else {
		slog.Debug("ChatV2: Saved thinking trace", "chars", len(thinkingContent), "tokens", thinkingTokens)
	}
}
