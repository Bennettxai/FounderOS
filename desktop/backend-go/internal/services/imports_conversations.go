package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// IMPORTED CONVERSATIONS QUERY METHODS
// =============================================================================

// GetImportedConversations retrieves imported conversations for a user
func (s *ImportService) GetImportedConversations(ctx context.Context, userID string, limit, offset int32) ([]sqlc.ImportedConversation, error) {
	return s.queries.GetImportedConversationsByUser(ctx, sqlc.GetImportedConversationsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// GetImportedConversationsBySource retrieves imported conversations filtered by source
func (s *ImportService) GetImportedConversationsBySource(ctx context.Context, userID string, sourceType sqlc.ImportSourceType, limit, offset int32) ([]sqlc.ImportedConversation, error) {
	return s.queries.GetImportedConversationsBySource(ctx, sqlc.GetImportedConversationsBySourceParams{
		UserID:     userID,
		SourceType: sourceType,
		Limit:      limit,
		Offset:     offset,
	})
}

// GetImportedConversation retrieves a single imported conversation
func (s *ImportService) GetImportedConversation(ctx context.Context, userID string, conversationID uuid.UUID) (*sqlc.ImportedConversation, error) {
	conv, err := s.queries.GetImportedConversation(ctx, sqlc.GetImportedConversationParams{
		ID:     pgtype.UUID{Bytes: conversationID, Valid: true},
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// SearchImportedConversations searches imported conversations by content
func (s *ImportService) SearchImportedConversations(ctx context.Context, userID string, query string, limit int32) ([]sqlc.ImportedConversation, error) {
	return s.queries.SearchImportedConversations(ctx, sqlc.SearchImportedConversationsParams{
		UserID:         userID,
		PlaintoTsquery: query,
		Limit:          limit,
	})
}

// CountImportedConversations counts imported conversations for a user
func (s *ImportService) CountImportedConversations(ctx context.Context, userID string) (int64, error) {
	return s.queries.CountImportedConversations(ctx, userID)
}

// LinkConversationToContext links an imported conversation to a BusinessOS context
func (s *ImportService) LinkConversationToContext(ctx context.Context, userID string, conversationID, contextID uuid.UUID) error {
	return s.queries.LinkConversationToContext(ctx, sqlc.LinkConversationToContextParams{
		ID:              pgtype.UUID{Bytes: conversationID, Valid: true},
		UserID:          userID,
		LinkedContextID: pgtype.UUID{Bytes: contextID, Valid: true},
	})
}

// LinkConversationToProject links an imported conversation to a BusinessOS project
func (s *ImportService) LinkConversationToProject(ctx context.Context, userID string, conversationID, projectID uuid.UUID) error {
	return s.queries.LinkConversationToProject(ctx, sqlc.LinkConversationToProjectParams{
		ID:              pgtype.UUID{Bytes: conversationID, Valid: true},
		UserID:          userID,
		LinkedProjectID: pgtype.UUID{Bytes: projectID, Valid: true},
	})
}

// UpdateConversationTags updates the tags on an imported conversation
func (s *ImportService) UpdateConversationTags(ctx context.Context, userID string, conversationID uuid.UUID, tags []string) error {
	return s.queries.UpdateConversationTags(ctx, sqlc.UpdateConversationTagsParams{
		ID:     pgtype.UUID{Bytes: conversationID, Valid: true},
		UserID: userID,
		Tags:   tags,
	})
}

// DeleteImportedConversation deletes an imported conversation
func (s *ImportService) DeleteImportedConversation(ctx context.Context, userID string, conversationID uuid.UUID) error {
	return s.queries.DeleteImportedConversation(ctx, sqlc.DeleteImportedConversationParams{
		ID:     pgtype.UUID{Bytes: conversationID, Valid: true},
		UserID: userID,
	})
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// buildSearchContent creates searchable text from conversation title and messages
func (s *ImportService) buildSearchContent(title string, messages []NormalizedMessage) string {
	var parts []string
	if title != "" {
		parts = append(parts, title)
	}
	for _, msg := range messages {
		if msg.Content != "" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, " ")
}

// hashData creates a SHA256 hash of data for change detection
func (s *ImportService) hashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// unixToTimestamp converts Unix timestamp to pgtype.Timestamptz
func (s *ImportService) unixToTimestamp(unix float64) pgtype.Timestamptz {
	if unix <= 0 {
		return pgtype.Timestamptz{}
	}
	sec := int64(unix)
	nsec := int64((unix - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec)
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// parseISO8601 parses an ISO8601 timestamp to pgtype.Timestamptz
func (s *ImportService) parseISO8601(ts string) pgtype.Timestamptz {
	if ts == "" {
		return pgtype.Timestamptz{}
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}
