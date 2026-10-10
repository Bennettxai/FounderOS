package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// DeltaSync returns all tasks and notifications that changed after a Unix timestamp.
// Query params: since (required, Unix timestamp), limit (1-500, default 100)
func (h *MobileHandler) DeltaSync(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	sinceStr := c.Query("since")
	if sinceStr == "" {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "since parameter required")
		return
	}

	sinceUnix, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid since timestamp")
		return
	}
	since := time.Unix(sinceUnix, 0)

	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	limit = ClampInt(limit, 1, 500)

	ctx := c.Request.Context()
	sincePg := pgtype.Timestamp{Time: since, Valid: true}
	sincePgTz := pgtype.Timestamptz{Time: since, Valid: true}

	taskRows, _ := h.queries.GetTaskChangesSince(ctx, sqlc.GetTaskChangesSinceParams{
		UserID:     user.ID,
		UpdatedAt:  sincePg,
		LimitCount: int32(limit),
	})

	notifRows, _ := h.queries.GetNotificationChangesSince(ctx, sqlc.GetNotificationChangesSinceParams{
		UserID:     user.ID,
		CreatedAt:  sincePgTz,
		LimitCount: int32(limit),
	})

	tasks := make([]MobileTaskResponse, 0, len(taskRows))
	var latestTaskTime int64
	for _, row := range taskRows {
		tasks = append(tasks, TransformSyncTaskRow(row))
		if row.UpdatedAt.Valid && row.UpdatedAt.Time.Unix() > latestTaskTime {
			latestTaskTime = row.UpdatedAt.Time.Unix()
		}
	}

	notifications := make([]MobileNotificationResponse, 0, len(notifRows))
	var latestNotifTime int64
	for _, row := range notifRows {
		notifications = append(notifications, TransformSyncNotificationRow(row))
		if row.CreatedAt.Valid && row.CreatedAt.Time.Unix() > latestNotifTime {
			latestNotifTime = row.CreatedAt.Time.Unix()
		}
	}

	// latestTaskTime and latestNotifTime are computed but currently unused in the
	// response payload; they're available for future "next_since" cursor support.
	_ = latestTaskTime
	_ = latestNotifTime

	c.JSON(http.StatusOK, MobileSyncResponse{
		Tasks:         tasks,
		Notifications: notifications,
		ServerTime:    time.Now().Unix(),
		HasMore:       len(taskRows) >= limit || len(notifRows) >= limit,
	})
}
