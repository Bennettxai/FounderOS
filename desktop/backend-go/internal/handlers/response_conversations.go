package handlers

import (
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Message response transformation
type MessageResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

func TransformMessage(m sqlc.Message) MessageResponse {
	return MessageResponse{
		ID:             pgtypeUUIDToStringRequired(m.ID),
		ConversationID: pgtypeUUIDToStringRequired(m.ConversationID),
		Role:           strings.ToLower(string(m.Role)),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.Time.Format(time.RFC3339),
	}
}

func TransformMessages(messages []sqlc.Message) []MessageResponse {
	result := make([]MessageResponse, len(messages))
	for i, m := range messages {
		result[i] = TransformMessage(m)
	}
	return result
}

// Conversation response transformation
type ConversationResponse struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	Title        string  `json:"title"`
	ContextID    *string `json:"context_id"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	MessageCount int64   `json:"message_count"`
}

func TransformConversation(c sqlc.Conversation) ConversationResponse {
	title := "New Conversation"
	if c.Title != nil {
		title = *c.Title
	}

	return ConversationResponse{
		ID:        pgtypeUUIDToStringRequired(c.ID),
		UserID:    c.UserID,
		Title:     title,
		ContextID: pgtypeUUIDToString(c.ContextID),
		CreatedAt: c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformConversationListRow(c sqlc.ListConversationsRow) ConversationResponse {
	title := "New Conversation"
	if c.Title != nil {
		title = *c.Title
	}

	return ConversationResponse{
		ID:           pgtypeUUIDToStringRequired(c.ID),
		UserID:       c.UserID,
		Title:        title,
		ContextID:    pgtypeUUIDToString(c.ContextID),
		CreatedAt:    c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Time.Format(time.RFC3339),
		MessageCount: c.MessageCount,
	}
}

func TransformConversationListRows(conversations []sqlc.ListConversationsRow) []ConversationResponse {
	result := make([]ConversationResponse, len(conversations))
	for i, c := range conversations {
		result[i] = TransformConversationListRow(c)
	}
	return result
}

func TransformConversationByContextRow(c sqlc.ListConversationsByContextRow) ConversationResponse {
	title := "New Conversation"
	if c.Title != nil {
		title = *c.Title
	}

	return ConversationResponse{
		ID:           pgtypeUUIDToStringRequired(c.ID),
		UserID:       c.UserID,
		Title:        title,
		ContextID:    pgtypeUUIDToString(c.ContextID),
		CreatedAt:    c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Time.Format(time.RFC3339),
		MessageCount: c.MessageCount,
	}
}

func TransformConversationsByContextRows(conversations []sqlc.ListConversationsByContextRow) []ConversationResponse {
	result := make([]ConversationResponse, len(conversations))
	for i, c := range conversations {
		result[i] = TransformConversationByContextRow(c)
	}
	return result
}
