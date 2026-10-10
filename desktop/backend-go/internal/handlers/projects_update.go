package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// UpdateProject updates an existing project.
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	var req struct {
		Name            *string         `json:"name"`
		Description     *string         `json:"description"`
		Status          *string         `json:"status"`
		Priority        *string         `json:"priority"`
		ClientName      *string         `json:"client_name"`
		ClientID        *string         `json:"client_id"`
		ProjectType     *string         `json:"project_type"`
		ProjectMetadata json.RawMessage `json:"project_metadata"`
		StartDate       *string         `json:"start_date"`
		DueDate         *string         `json:"due_date"`
		Visibility      *string         `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Get existing project first
	existing, err := queries.GetProject(c.Request.Context(), sqlc.GetProjectParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Project")
		return
	}

	// Build update params with existing values as defaults
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}

	status := existing.Status
	if req.Status != nil {
		status = sqlc.NullProjectstatus{
			Projectstatus: stringToProjectStatus(*req.Status),
			Valid:         true,
		}
	}

	priority := existing.Priority
	if req.Priority != nil {
		priority = sqlc.NullProjectpriority{
			Projectpriority: stringToProjectPriority(*req.Priority),
			Valid:           true,
		}
	}

	clientName := existing.ClientName
	if req.ClientName != nil {
		clientName = req.ClientName
	}

	clientID := existing.ClientID
	if req.ClientID != nil {
		if cid, err := uuid.Parse(*req.ClientID); err == nil {
			clientID = pgtype.UUID{Bytes: cid, Valid: true}
		}
	}

	projectType := existing.ProjectType
	if req.ProjectType != nil {
		projectType = req.ProjectType
	}

	metadata := existing.ProjectMetadata
	if req.ProjectMetadata != nil {
		metadata = req.ProjectMetadata
	}

	startDate := existing.StartDate
	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	dueDate := existing.DueDate
	if req.DueDate != nil {
		if t, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			dueDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	visibility := existing.Visibility
	if req.Visibility != nil {
		visibility = req.Visibility
	}

	project, err := queries.UpdateProject(c.Request.Context(), sqlc.UpdateProjectParams{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		Name:            name,
		Description:     description,
		Status:          status,
		Priority:        priority,
		ClientName:      clientName,
		ClientID:        clientID,
		ProjectType:     projectType,
		ProjectMetadata: metadata,
		StartDate:       startDate,
		DueDate:         dueDate,
		Visibility:      visibility,
	})
	if err != nil {
		slog.Info("UpdateProject error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "update project", nil)
		return
	}

	// Invalidate projects cache for this user after successful update
	if h.queryCache != nil {
		cachePattern := fmt.Sprintf("projects:%s:*", user.ID)
		go h.invalidateProjectsCachePattern(c.Request.Context(), cachePattern)
	}

	// Trigger notification if status changed
	if h.notificationTriggers != nil && req.Status != nil && existing.Status.Valid {
		oldStatus := string(existing.Status.Projectstatus)
		newStatus := *req.Status
		if oldStatus != newStatus {
			// Get project members to notify
			members, _ := queries.ListProjectMembers(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
			var memberIDs []string
			for _, m := range members {
				if m.TeamMemberID.Valid {
					memberIDs = append(memberIDs, uuid.UUID(m.TeamMemberID.Bytes).String())
				}
			}
			go h.notificationTriggers.OnProjectStatusChanged(c.Request.Context(), services.ProjectStatusChangedInput{
				ProjectID:   id,
				ProjectName: name,
				OldStatus:   oldStatus,
				NewStatus:   newStatus,
				ChangedByID: user.ID,
				ChangedBy:   user.Name,
				MemberIDs:   memberIDs,
			})
		}
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project.
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteProject(c.Request.Context(), sqlc.DeleteProjectParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete project", nil)
		return
	}

	// Invalidate projects cache for this user after successful deletion
	if h.queryCache != nil {
		cachePattern := fmt.Sprintf("projects:%s:*", user.ID)
		go h.invalidateProjectsCachePattern(c.Request.Context(), cachePattern)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// AddProjectNote adds a note to a project.
func (h *ProjectHandler) AddProjectNote(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify project ownership
	_, err = queries.GetProject(c.Request.Context(), sqlc.GetProjectParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Project")
		return
	}

	note, err := queries.AddProjectNote(c.Request.Context(), sqlc.AddProjectNoteParams{
		ProjectID: pgtype.UUID{Bytes: id, Valid: true},
		Content:   req.Content,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "add note", nil)
		return
	}

	c.JSON(http.StatusCreated, note)
}

// invalidateProjectsCachePattern invalidates project list cache entries matching a pattern.
// This is a background helper called after mutations; errors are non-critical.
func (h *ProjectHandler) invalidateProjectsCachePattern(ctx context.Context, pattern string) {
	if h.queryCache == nil {
		return
	}

	// Use Redis SCAN to find matching keys and delete them.
	// Note: The cache package doesn't expose a wildcard deletion method directly.
	// For now, we rely on TTL expiration (5 minutes) for automatic invalidation.
	// Future improvement: implement a ScanAndDelete method in the cache package.
	slog.Debug("Cache invalidation scheduled (will expire via TTL)", "pattern", pattern)
}
