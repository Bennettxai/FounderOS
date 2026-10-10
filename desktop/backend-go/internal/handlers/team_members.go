package handlers

import (
	"context"
	"fmt"
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

// UpdateTeamMemberStatus updates a team member's status.
func (h *TeamHandler) UpdateTeamMemberStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	_, err = queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	member, err := queries.UpdateTeamMemberStatus(c.Request.Context(), sqlc.UpdateTeamMemberStatusParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
		Status: sqlc.NullMemberstatus{
			Memberstatus: stringToMemberStatus(req.Status),
			Valid:        true,
		},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update team member status", err)
		return
	}

	if h.queryCache != nil {
		pattern := fmt.Sprintf("team:user:%s*", user.ID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if deleted, err := h.queryCache.DeleteByPattern(ctx, pattern); err == nil {
				slog.Info("UpdateTeamMemberStatus: invalidated  cache entries for user", "id", deleted, "id", user.ID)
			}
		}()
	}

	c.JSON(http.StatusOK, member)
}

// UpdateTeamMemberCapacity updates a team member's capacity.
func (h *TeamHandler) UpdateTeamMemberCapacity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	var req struct {
		Capacity int32 `json:"capacity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	_, err = queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	member, err := queries.UpdateTeamMemberCapacity(c.Request.Context(), sqlc.UpdateTeamMemberCapacityParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		Capacity: &req.Capacity,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update team member capacity", err)
		return
	}

	if h.queryCache != nil {
		pattern := fmt.Sprintf("team:user:%s*", user.ID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if deleted, err := h.queryCache.DeleteByPattern(ctx, pattern); err == nil {
				slog.Info("UpdateTeamMemberCapacity: invalidated  cache entries for user", "id", deleted, "id", user.ID)
			}
		}()
	}

	c.JSON(http.StatusOK, member)
}

// AddTeamMemberActivity adds an activity to a team member.
func (h *TeamHandler) AddTeamMemberActivity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	var req struct {
		ActivityType string `json:"activity_type" binding:"required"`
		Description  string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	_, err = queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	activity, err := queries.CreateTeamMemberActivity(c.Request.Context(), sqlc.CreateTeamMemberActivityParams{
		MemberID:     pgtype.UUID{Bytes: id, Valid: true},
		ActivityType: req.ActivityType,
		Description:  req.Description,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "add activity", err)
		return
	}

	c.JSON(http.StatusCreated, activity)
}

// GetTeamMemberActivities returns activities for a team member.
func (h *TeamHandler) GetTeamMemberActivities(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	queries := sqlc.New(h.pool)

	_, err = queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	limit := int32(50)
	activities, err := queries.GetTeamMemberActivities(c.Request.Context(), sqlc.GetTeamMemberActivitiesParams{
		MemberID: pgtype.UUID{Bytes: id, Valid: true},
		Limit:    limit,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get activities", err)
		return
	}

	c.JSON(http.StatusOK, activities)
}
