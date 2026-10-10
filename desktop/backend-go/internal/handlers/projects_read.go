package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// GetProject returns a single project with notes.
func (h *ProjectHandler) GetProject(c *gin.Context) {
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
	project, err := queries.GetProject(c.Request.Context(), sqlc.GetProjectParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Project")
		return
	}

	// Check if notes are requested
	if c.Query("include_notes") == "true" {
		notes, err := queries.GetProjectNotes(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"project": project,
				"notes":   notes,
			})
			return
		}
	}

	// Get related artifacts
	if c.Query("include_artifacts") == "true" {
		artifacts, err := queries.ListArtifacts(c.Request.Context(), sqlc.ListArtifactsParams{
			UserID:    user.ID,
			ProjectID: pgtype.UUID{Bytes: id, Valid: true},
		})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"project":   project,
				"artifacts": artifacts,
			})
			return
		}
	}

	c.JSON(http.StatusOK, project)
}

// GetProjectStats returns project statistics for the user.
func (h *ProjectHandler) GetProjectStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	stats, err := queries.GetProjectStats(c.Request.Context(), user.ID)
	if err != nil {
		slog.Info("GetProjectStats error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get project stats", nil)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetOverdueProjects returns overdue projects for the user.
func (h *ProjectHandler) GetOverdueProjects(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	projects, err := queries.GetOverdueProjects(c.Request.Context(), user.ID)
	if err != nil {
		slog.Info("GetOverdueProjects error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get overdue projects", nil)
		return
	}

	c.JSON(http.StatusOK, projects)
}

// GetUpcomingProjects returns projects due within 7 days.
func (h *ProjectHandler) GetUpcomingProjects(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	projects, err := queries.GetUpcomingProjects(c.Request.Context(), user.ID)
	if err != nil {
		slog.Info("GetUpcomingProjects error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get upcoming projects", nil)
		return
	}

	c.JSON(http.StatusOK, projects)
}
