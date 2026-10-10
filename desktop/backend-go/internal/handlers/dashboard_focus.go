package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ListFocusItems returns focus items for a specific date
func (h *DashboardItemHandler) ListFocusItems(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse date from query, default to today
	dateStr := c.Query("date")
	focusDate := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			focusDate = t
		}
	}

	items, err := queries.ListFocusItems(c.Request.Context(), sqlc.ListFocusItemsParams{
		UserID:    user.ID,
		FocusDate: pgtype.Date{Time: focusDate, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list focus items", err)
		return
	}

	c.JSON(http.StatusOK, TransformFocusItems(items))
}

// CreateFocusItem creates a new focus item
func (h *DashboardItemHandler) CreateFocusItem(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Text      string  `json:"text" binding:"required"`
		FocusDate *string `json:"focus_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse focus date, default to today
	focusDate := time.Now()
	if req.FocusDate != nil {
		if t, err := time.Parse("2006-01-02", *req.FocusDate); err == nil {
			focusDate = t
		}
	}

	item, err := queries.CreateFocusItem(c.Request.Context(), sqlc.CreateFocusItemParams{
		UserID:    user.ID,
		Text:      req.Text,
		FocusDate: pgtype.Timestamp{Time: focusDate, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create focus item", err)
		return
	}

	c.JSON(http.StatusCreated, TransformFocusItem(item))
}

// UpdateFocusItem updates a focus item
func (h *DashboardItemHandler) UpdateFocusItem(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "focus item ID")
		return
	}

	var req struct {
		Text      string `json:"text" binding:"required"`
		Completed *bool  `json:"completed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	item, err := queries.UpdateFocusItem(c.Request.Context(), sqlc.UpdateFocusItemParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Text:      req.Text,
		Completed: req.Completed,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update focus item", err)
		return
	}

	c.JSON(http.StatusOK, TransformFocusItem(item))
}

// DeleteFocusItem deletes a focus item
func (h *DashboardItemHandler) DeleteFocusItem(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "focus item ID")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteFocusItem(c.Request.Context(), sqlc.DeleteFocusItemParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete focus item", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Focus item deleted"})
}
