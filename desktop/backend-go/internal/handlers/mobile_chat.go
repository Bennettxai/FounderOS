package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// ListChatThreads returns recent AI conversation threads for the mobile chat view.
func (h *MobileHandler) ListChatThreads(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	limit = ClampInt(limit, 1, 50)

	rows, err := h.queries.ListConversationsForMobile(c.Request.Context(), sqlc.ListConversationsForMobileParams{
		UserID:     user.ID,
		LimitCount: int32(limit),
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to fetch threads")
		return
	}

	threads := make([]MobileChatThreadResponse, 0, len(rows))
	for _, row := range rows {
		threads = append(threads, TransformToChatThread(row))
	}

	c.JSON(http.StatusOK, MobileChatThreadListResponse{
		Threads: threads,
	})
}

// GetChatHistory returns cursor-paginated messages for a specific conversation.
func (h *MobileHandler) GetChatHistory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid conversation ID")
		return
	}

	limitStr := c.DefaultQuery("limit", "30")
	limit, _ := strconv.Atoi(limitStr)
	limit = ClampInt(limit, 1, 100)

	var cursorTime pgtype.Timestamp
	var cursorID pgtype.UUID
	if cursor := c.Query("cursor"); cursor != "" {
		id, ts, err := DecodeCursor(cursor)
		if err == nil && id != uuid.Nil {
			cursorTime = pgtype.Timestamp{Time: ts, Valid: true}
			cursorID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	rows, err := h.queries.GetMessagesForMobile(c.Request.Context(), sqlc.GetMessagesForMobileParams{
		ConversationID:  pgtype.UUID{Bytes: conversationID, Valid: true},
		CursorCreatedAt: cursorTime,
		CursorID:        cursorID,
		LimitCount:      int32(limit + 1),
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to fetch messages")
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	messages := make([]MobileChatMessageResponse, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, TransformToChatMessage(row))
	}

	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = EncodeCursor(uuid.UUID(last.ID.Bytes), last.CreatedAt.Time)
	}

	c.JSON(http.StatusOK, MobileChatHistoryResponse{
		Messages: messages,
		Cursor:   nextCursor,
		HasMore:  hasMore,
	})
}

// SendChatMessage persists a user message. The AI response is streamed separately.
func (h *MobileHandler) SendChatMessage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	var req MobileChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid request")
		return
	}

	ctx := c.Request.Context()

	_, err := h.queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  pgtype.UUID{Bytes: req.ConversationID, Valid: true},
		Role:            sqlc.MessageroleUSER,
		Content:         req.Content,
		MessageMetadata: nil,
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to save message")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "received",
		"message": "Message saved. AI response will be streamed separately.",
	})
}
