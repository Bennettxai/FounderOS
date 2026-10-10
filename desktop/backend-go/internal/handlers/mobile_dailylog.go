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

// GetTodayLog returns the current day's daily log for the authenticated user.
func (h *MobileHandler) GetTodayLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	row, err := h.queries.GetTodayDailyLogForMobile(c.Request.Context(), user.ID)
	if err != nil {
		// No log for today yet — return an empty shell rather than an error.
		c.JSON(http.StatusOK, MobileDailyLogTodayResponse{
			Date:    time.Now().Format("2006-01-02"),
			Entries: []MobileDailyLogEntryResponse{},
		})
		return
	}

	c.JSON(http.StatusOK, MobileDailyLogTodayResponse{
		Date:        row.Date.Time.Format("2006-01-02"),
		Content:     row.Content,
		EnergyLevel: row.EnergyLevel,
	})
}

// GetLogHistory returns a paginated list of past daily log summaries.
func (h *MobileHandler) GetLogHistory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	limitStr := c.DefaultQuery("limit", "14")
	limit, _ := strconv.Atoi(limitStr)
	limit = ClampInt(limit, 1, 30)

	var beforeDate pgtype.Date
	if before := c.Query("before"); before != "" {
		if t, err := time.Parse("2006-01-02", before); err == nil {
			beforeDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	rows, err := h.queries.GetDailyLogHistoryForMobile(c.Request.Context(), sqlc.GetDailyLogHistoryForMobileParams{
		UserID:     user.ID,
		BeforeDate: beforeDate,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to fetch history")
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	logs := make([]MobileDailyLogHistoryItem, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, TransformToDailyLogHistoryItem(row))
	}

	var nextBefore string
	if hasMore && len(rows) > 0 {
		nextBefore = rows[len(rows)-1].Date.Time.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, MobileDailyLogHistoryResponse{
		Logs:    logs,
		HasMore: hasMore,
		Before:  nextBefore,
	})
}
