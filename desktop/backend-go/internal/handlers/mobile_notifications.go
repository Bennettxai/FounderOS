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

// ListNotifications returns a paginated, optionally filtered list of notifications.
func (h *MobileHandler) ListNotifications(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	limit = ClampInt(limit, 1, 50)
	unreadOnly := c.Query("unread") == "true"

	var cursorTime pgtype.Timestamptz
	var cursorID pgtype.UUID
	if cursor := c.Query("cursor"); cursor != "" {
		id, ts, err := DecodeCursor(cursor)
		if err == nil && id != uuid.Nil {
			cursorTime = pgtype.Timestamptz{Time: ts, Valid: true}
			cursorID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	rows, err := h.queries.ListNotificationsForMobile(c.Request.Context(), sqlc.ListNotificationsForMobileParams{
		UserID:          user.ID,
		UnreadOnly:      &unreadOnly,
		CursorCreatedAt: cursorTime,
		CursorID:        cursorID,
		LimitCount:      int32(limit + 1),
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to fetch notifications")
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	notifications := make([]MobileNotificationResponse, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, TransformToMobileNotification(row))
	}

	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = EncodeCursor(uuid.UUID(last.ID.Bytes), last.CreatedAt.Time)
	}

	unreadCount, _ := h.queries.GetUnreadNotificationCount(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, MobileNotificationListResponse{
		Notifications: notifications,
		Cursor:        nextCursor,
		HasMore:       hasMore,
		UnreadCount:   int(unreadCount),
	})
}

// GetNotificationCount returns the unread notification badge count.
func (h *MobileHandler) GetNotificationCount(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	count, err := h.queries.GetUnreadNotificationCount(c.Request.Context(), user.ID)
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to get count")
		return
	}

	c.JSON(http.StatusOK, MobileNotificationCountResponse{
		UnreadCount: int(count),
	})
}

// MarkNotificationsRead marks notifications as read — either a specific set or all.
func (h *MobileHandler) MarkNotificationsRead(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	var req MobileNotificationReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid request body")
		return
	}

	var markedCount int64
	var err error

	if req.All {
		markedCount, err = h.queries.MarkAllNotificationsAsRead(c.Request.Context(), user.ID)
		if err != nil {
			MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to mark notifications")
			return
		}
	} else if len(req.IDs) > 0 {
		ids := make([]pgtype.UUID, len(req.IDs))
		for i, id := range req.IDs {
			ids[i] = pgtype.UUID{Bytes: id, Valid: true}
		}
		err = h.queries.MarkNotificationsAsRead(c.Request.Context(), sqlc.MarkNotificationsAsReadParams{
			UserID: user.ID,
			Ids:    ids,
		})
		if err != nil {
			MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to mark notifications")
			return
		}
		markedCount = int64(len(req.IDs))
	}

	unreadCount, _ := h.queries.GetUnreadNotificationCount(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, MobileNotificationReadResponse{
		MarkedCount: int(markedCount),
		UnreadCount: int(unreadCount),
	})
}
