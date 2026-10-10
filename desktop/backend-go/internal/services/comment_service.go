package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// CommentService handles comment operations with mention parsing and notifications
type CommentService struct {
	db                  *pgxpool.Pool
	queries             *sqlc.Queries
	notificationService *NotificationService
}

// NewCommentService creates a new comment service
func NewCommentService(db *pgxpool.Pool, queries *sqlc.Queries, notificationService *NotificationService) *CommentService {
	return &CommentService{
		db:                  db,
		queries:             queries,
		notificationService: notificationService,
	}
}

// CommentWithAuthor represents a comment with author information
type CommentWithAuthor struct {
	ID          uuid.UUID           `json:"id"`
	UserID      string              `json:"user_id"`
	AuthorName  string              `json:"author_name"`
	AuthorEmail string              `json:"author_email"`
	AvatarURL   *string             `json:"avatar_url,omitempty"`
	EntityType  string              `json:"entity_type"`
	EntityID    uuid.UUID           `json:"entity_id"`
	Content     string              `json:"content"`
	ParentID    *uuid.UUID          `json:"parent_id,omitempty"`
	IsEdited    bool                `json:"is_edited"`
	EditedAt    *time.Time          `json:"edited_at,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Replies     []CommentWithAuthor `json:"replies,omitempty"`
	Reactions   []ReactionSummary   `json:"reactions,omitempty"`
	Mentions    []MentionInfo       `json:"mentions,omitempty"`
}

// ReactionSummary represents aggregated reactions
type ReactionSummary struct {
	Emoji string   `json:"emoji"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

// MentionInfo represents a mention in a comment
type MentionInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Position int    `json:"position"`
}

// ParsedMention represents a parsed @mention from text
type ParsedMention struct {
	UserID    string
	Username  string
	Position  int
	MatchText string
}

// CreateCommentInput is the input for creating a comment
type CreateCommentInput struct {
	UserID     string
	EntityType string // "task", "project", "note", etc.
	EntityID   uuid.UUID
	Content    string
	ParentID   *uuid.UUID
}

// CreateComment creates a new comment with mention parsing and notifications
func (s *CommentService) CreateComment(ctx context.Context, input CreateCommentInput) (*CommentWithAuthor, error) {
	// 1. Create the comment
	parentID := pgtype.UUID{}
	if input.ParentID != nil {
		parentID.Bytes = *input.ParentID
		parentID.Valid = true
	}

	entityID := pgtype.UUID{Bytes: input.EntityID, Valid: true}

	comment, err := s.queries.CreateComment(ctx, sqlc.CreateCommentParams{
		UserID:     input.UserID,
		EntityType: input.EntityType,
		EntityID:   entityID,
		Content:    input.Content,
		ParentID:   parentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	commentID := uuid.UUID(comment.ID.Bytes)

	// 2. Parse and store mentions
	mentions := s.ParseMentions(input.Content)
	for _, mention := range mentions {
		position := int32(mention.Position)
		matchText := mention.MatchText
		_, err := s.queries.CreateEntityMention(ctx, sqlc.CreateEntityMentionParams{
			SourceType:      "comment",
			SourceID:        comment.ID,
			MentionedUserID: mention.UserID,
			MentionText:     &matchText,
			PositionInText:  &position,
			EntityType:      &input.EntityType,
			EntityID:        entityID,
			MentionedBy:     input.UserID,
		})
		if err != nil {
			// Log but don't fail the whole operation
			slog.Info("[CommentService] Failed to store mention", "error", err)
		}
	}

	// 3. Trigger notifications
	go s.triggerCommentNotifications(context.Background(), input, commentID, mentions)

	// 4. Get author info and return full comment
	return s.GetCommentByID(ctx, commentID)
}

// GetCommentByID retrieves a single comment with author info
func (s *CommentService) GetCommentByID(ctx context.Context, id uuid.UUID) (*CommentWithAuthor, error) {
	commentID := pgtype.UUID{Bytes: id, Valid: true}

	row, err := s.queries.GetCommentWithAuthor(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("comment not found: %w", err)
	}

	var avatarURL *string
	if row.AvatarUrl != nil {
		avatarURL = row.AvatarUrl
	}

	entityID := uuid.UUID(row.EntityID.Bytes)
	var parentID *uuid.UUID
	if row.ParentID.Valid {
		pid := uuid.UUID(row.ParentID.Bytes)
		parentID = &pid
	}

	var editedAt *time.Time
	if row.EditedAt.Valid {
		t := row.EditedAt.Time
		editedAt = &t
	}

	// Safely dereference IsEdited and AuthorName
	isEdited := false
	if row.IsEdited != nil {
		isEdited = *row.IsEdited
	}

	authorName := ""
	if row.AuthorName != nil {
		authorName = *row.AuthorName
	}

	authorEmail := ""
	if row.AuthorEmail != nil {
		authorEmail = *row.AuthorEmail
	}

	return &CommentWithAuthor{
		ID:          uuid.UUID(row.ID.Bytes),
		UserID:      row.UserID,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
		AvatarURL:   avatarURL,
		EntityType:  row.EntityType,
		EntityID:    entityID,
		Content:     row.Content,
		ParentID:    parentID,
		IsEdited:    isEdited,
		EditedAt:    editedAt,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

// GetCommentsByEntity retrieves all comments for an entity with author info
func (s *CommentService) GetCommentsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]CommentWithAuthor, error) {
	eid := pgtype.UUID{Bytes: entityID, Valid: true}

	// Get top-level comments with author info
	rows, err := s.queries.ListCommentsWithAuthor(ctx, sqlc.ListCommentsWithAuthorParams{
		EntityType: entityType,
		EntityID:   eid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// Build comment list and fetch replies
	var result []CommentWithAuthor

	for _, row := range rows {
		comment := s.rowToCommentWithAuthor(row)

		// Fetch replies for each top-level comment
		replies, err := s.getRepliesWithAuthor(ctx, comment.ID)
		if err == nil {
			comment.Replies = replies
		}

		result = append(result, comment)
	}

	return result, nil
}

// rowToCommentWithAuthor converts a ListCommentsWithAuthorRow to CommentWithAuthor
func (s *CommentService) rowToCommentWithAuthor(row sqlc.ListCommentsWithAuthorRow) CommentWithAuthor {
	var avatarURL *string
	if row.AvatarUrl != nil {
		avatarURL = row.AvatarUrl
	}

	var parentID *uuid.UUID
	if row.ParentID.Valid {
		pid := uuid.UUID(row.ParentID.Bytes)
		parentID = &pid
	}

	var editedAt *time.Time
	if row.EditedAt.Valid {
		t := row.EditedAt.Time
		editedAt = &t
	}

	isEdited := false
	if row.IsEdited != nil {
		isEdited = *row.IsEdited
	}

	authorName := ""
	if row.AuthorName != nil {
		authorName = *row.AuthorName
	}

	authorEmail := ""
	if row.AuthorEmail != nil {
		authorEmail = *row.AuthorEmail
	}

	return CommentWithAuthor{
		ID:          uuid.UUID(row.ID.Bytes),
		UserID:      row.UserID,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
		AvatarURL:   avatarURL,
		EntityType:  row.EntityType,
		EntityID:    uuid.UUID(row.EntityID.Bytes),
		Content:     row.Content,
		ParentID:    parentID,
		IsEdited:    isEdited,
		EditedAt:    editedAt,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		Replies:     []CommentWithAuthor{},
	}
}

// getRepliesWithAuthor fetches replies for a comment with author info
func (s *CommentService) getRepliesWithAuthor(ctx context.Context, parentID uuid.UUID) ([]CommentWithAuthor, error) {
	pid := pgtype.UUID{Bytes: parentID, Valid: true}

	rows, err := s.queries.ListRepliesWithAuthor(ctx, pid)
	if err != nil {
		return nil, err
	}

	var replies []CommentWithAuthor
	for _, row := range rows {
		var avatarURL *string
		if row.AvatarUrl != nil {
			avatarURL = row.AvatarUrl
		}

		var parentPtr *uuid.UUID
		if row.ParentID.Valid {
			p := uuid.UUID(row.ParentID.Bytes)
			parentPtr = &p
		}

		var editedAt *time.Time
		if row.EditedAt.Valid {
			t := row.EditedAt.Time
			editedAt = &t
		}

		isEdited := false
		if row.IsEdited != nil {
			isEdited = *row.IsEdited
		}

		authorName := ""
		if row.AuthorName != nil {
			authorName = *row.AuthorName
		}

		authorEmail := ""
		if row.AuthorEmail != nil {
			authorEmail = *row.AuthorEmail
		}

		replies = append(replies, CommentWithAuthor{
			ID:          uuid.UUID(row.ID.Bytes),
			UserID:      row.UserID,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
			AvatarURL:   avatarURL,
			EntityType:  row.EntityType,
			EntityID:    uuid.UUID(row.EntityID.Bytes),
			Content:     row.Content,
			ParentID:    parentPtr,
			IsEdited:    isEdited,
			EditedAt:    editedAt,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		})
	}

	return replies, nil
}

// UpdateComment updates a comment's content
func (s *CommentService) UpdateComment(ctx context.Context, id uuid.UUID, userID string, content string) (*CommentWithAuthor, error) {
	commentID := pgtype.UUID{Bytes: id, Valid: true}

	// First verify ownership
	existing, err := s.queries.GetComment(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("comment not found: %w", err)
	}
	if existing.UserID != userID {
		return nil, fmt.Errorf("unauthorized: not comment owner")
	}

	_, err = s.queries.UpdateCommentContent(ctx, sqlc.UpdateCommentContentParams{
		ID:      commentID,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return s.GetCommentByID(ctx, id)
}

// DeleteComment soft-deletes a comment
func (s *CommentService) DeleteComment(ctx context.Context, id uuid.UUID, userID string) error {
	commentID := pgtype.UUID{Bytes: id, Valid: true}

	// First verify ownership
	existing, err := s.queries.GetComment(ctx, commentID)
	if err != nil {
		return fmt.Errorf("comment not found: %w", err)
	}
	if existing.UserID != userID {
		return fmt.Errorf("unauthorized: not comment owner")
	}

	return s.queries.SoftDeleteComment(ctx, commentID)
}

// AddReaction adds a reaction to a comment
func (s *CommentService) AddReaction(ctx context.Context, commentID uuid.UUID, userID string, emoji string) error {
	cid := pgtype.UUID{Bytes: commentID, Valid: true}

	_, err := s.queries.AddCommentReaction(ctx, sqlc.AddCommentReactionParams{
		CommentID: cid,
		UserID:    userID,
		Emoji:     emoji,
	})
	return err
}

// RemoveReaction removes a reaction from a comment
func (s *CommentService) RemoveReaction(ctx context.Context, commentID uuid.UUID, userID string, emoji string) error {
	cid := pgtype.UUID{Bytes: commentID, Valid: true}

	return s.queries.RemoveCommentReaction(ctx, sqlc.RemoveCommentReactionParams{
		CommentID: cid,
		UserID:    userID,
		Emoji:     emoji,
	})
}

// ParseMentions extracts @mentions from text
// Supports formats: @username, @[User Name](user_id)
func (s *CommentService) ParseMentions(content string) []ParsedMention {
	var mentions []ParsedMention

	// Pattern 1: @[Display Name](user_id) - Markdown-style
	markdownPattern := regexp.MustCompile(`@\[([^\]]+)\]\(([^)]+)\)`)
	markdownMatches := markdownPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range markdownMatches {
		if len(match) >= 6 {
			fullMatch := content[match[0]:match[1]]
			displayName := content[match[2]:match[3]]
			userID := content[match[4]:match[5]]
			mentions = append(mentions, ParsedMention{
				UserID:    userID,
				Username:  displayName,
				Position:  match[0],
				MatchText: fullMatch,
			})
		}
	}

	// Pattern 2: @username (simple) - only if no markdown mentions found
	if len(mentions) == 0 {
		simplePattern := regexp.MustCompile(`@(\w+)`)
		simpleMatches := simplePattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range simpleMatches {
			if len(match) >= 4 {
				fullMatch := content[match[0]:match[1]]
				username := content[match[2]:match[3]]
				mentions = append(mentions, ParsedMention{
					UserID:    "", // Will need to resolve from username
					Username:  username,
					Position:  match[0],
					MatchText: fullMatch,
				})
			}
		}
	}

	return mentions
}

// triggerCommentNotifications sends notifications for a new comment
func (s *CommentService) triggerCommentNotifications(ctx context.Context, input CreateCommentInput, commentID uuid.UUID, mentions []ParsedMention) {
	if s.notificationService == nil {
		return
	}

	// Get author info
	authorName := "Someone" // Default
	user, err := s.queries.GetUserByID(ctx, input.UserID)
	if err == nil && user.Name != nil {
		authorName = *user.Name
	}

	// Get entity info for context
	entityName := s.getEntityName(ctx, input.EntityType, input.EntityID)

	// 1. Notify entity owner (if not the commenter)
	ownerID := s.getEntityOwner(ctx, input.EntityType, input.EntityID)
	if ownerID != "" && ownerID != input.UserID {
		s.notificationService.OnTaskComment(ctx, OnTaskCommentEvent{
			TaskID:        input.EntityID,
			TaskTitle:     entityName,
			CommentID:     commentID,
			CommentText:   truncateCommentString(input.Content, 100),
			CommenterID:   input.UserID,
			CommenterName: authorName,
			TaskOwnerID:   ownerID,
		})
	}

	// 2. Notify mentioned users
	notifiedUsers := make(map[string]bool)
	notifiedUsers[input.UserID] = true // Don't notify commenter
	if ownerID != "" {
		notifiedUsers[ownerID] = true // Already notified above
	}

	for _, mention := range mentions {
		if mention.UserID != "" && !notifiedUsers[mention.UserID] {
			notifiedUsers[mention.UserID] = true
			s.notificationService.OnMention(ctx, OnMentionEvent{
				MentionedUserID: mention.UserID,
				MentionerID:     input.UserID,
				MentionerName:   authorName,
				SourceType:      "comment",
				SourceID:        commentID,
				EntityType:      input.EntityType,
				EntityID:        input.EntityID,
				EntityTitle:     entityName,
				Context:         truncateCommentString(input.Content, 150),
			})
		}
	}

	// 3. Notify parent comment author (for replies)
	if input.ParentID != nil {
		parentComment, err := s.GetCommentByID(ctx, *input.ParentID)
		if err == nil && parentComment.UserID != input.UserID && !notifiedUsers[parentComment.UserID] {
			s.notificationService.OnCommentReply(ctx, OnCommentReplyEvent{
				ParentCommentID: *input.ParentID,
				ParentAuthorID:  parentComment.UserID,
				ReplyID:         commentID,
				ReplyText:       truncateCommentString(input.Content, 100),
				ReplierID:       input.UserID,
				ReplierName:     authorName,
				EntityType:      input.EntityType,
				EntityID:        input.EntityID,
				EntityTitle:     entityName,
			})
		}
	}
}

// getEntityName retrieves the name/title of an entity
// For notifications, we use a simpler approach that doesn't require user_id
func (s *CommentService) getEntityName(ctx context.Context, entityType string, entityID uuid.UUID) string {
	// Use a simple query to get entity name without user_id requirement
	// For now, return a generic name - in production, add queries that don't require user_id
	switch entityType {
	case "task":
		return "task"
	case "project":
		return "project"
	}
	return "item"
}

// getEntityOwner retrieves the owner/assignee of an entity
// For notifications, we use a simpler approach
func (s *CommentService) getEntityOwner(ctx context.Context, entityType string, entityID uuid.UUID) string {
	// Return empty string - the entity owner lookup would need queries without user_id restriction
	// This is a simplification - in production, add appropriate queries
	return ""
}

// truncateCommentString truncates a string to max length
func truncateCommentString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
