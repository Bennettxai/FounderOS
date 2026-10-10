package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// WORKSPACE AUDIT LOG HANDLERS
// =====================================================================

// ListAuditLogs lists audit logs for a workspace with filtering.
// GET /api/workspaces/:id/audit-logs
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) ListAuditLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Build filter from query parameters
	filter := services.AuditLogFilter{
		WorkspaceID: workspaceID,
	}

	if userIDParam := c.Query("user_id"); userIDParam != "" {
		filter.UserID = &userIDParam
	}

	if actionParam := c.Query("action"); actionParam != "" {
		filter.Action = &actionParam
	}

	if resourceTypeParam := c.Query("resource_type"); resourceTypeParam != "" {
		filter.ResourceType = &resourceTypeParam
	}

	if resourceIDParam := c.Query("resource_id"); resourceIDParam != "" {
		filter.ResourceID = &resourceIDParam
	}

	if startDateParam := c.Query("start_date"); startDateParam != "" {
		startDate, err := time.Parse(time.RFC3339, startDateParam)
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), "Invalid start_date format, use RFC3339")
			return
		}
		filter.StartDate = &startDate
	}

	if endDateParam := c.Query("end_date"); endDateParam != "" {
		endDate, err := time.Parse(time.RFC3339, endDateParam)
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), "Invalid end_date format, use RFC3339")
			return
		}
		filter.EndDate = &endDate
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), "Invalid limit parameter")
			return
		}
		filter.Limit = limit
	}

	if offsetParam := c.Query("offset"); offsetParam != "" {
		offset, err := strconv.Atoi(offsetParam)
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), "Invalid offset parameter")
			return
		}
		filter.Offset = offset
	}

	logs, err := h.auditService.GetLogs(c.Request.Context(), filter)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get audit logs", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "count": len(logs)})
}

// GetAuditLog retrieves a specific audit log by ID.
// GET /api/workspaces/:id/audit-logs/:logId
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) GetAuditLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	logID, err := uuid.Parse(c.Param("logId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "log ID")
		return
	}

	log, err := h.auditService.GetLogByID(c.Request.Context(), logID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Audit log")
		return
	}

	// Verify the log belongs to the workspace
	if log.WorkspaceID != workspaceID {
		utils.RespondForbidden(c, slog.Default(), "Log does not belong to this workspace")
		return
	}

	c.JSON(http.StatusOK, log)
}

// GetUserActivity retrieves recent activity for a specific user.
// GET /api/workspaces/:id/audit-logs/user/:userId
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) GetUserActivity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	userID := c.Param("userId")

	limit := 50 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	logs, err := h.auditService.GetUserActivity(c.Request.Context(), workspaceID, userID, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get user activity", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID, "activity": logs, "count": len(logs)})
}

// GetResourceHistory retrieves history for a specific resource.
// GET /api/workspaces/:id/audit-logs/resource/:resourceType/:resourceId
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) GetResourceHistory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")

	limit := 50 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	logs, err := h.auditService.GetResourceHistory(c.Request.Context(), workspaceID, resourceType, resourceID, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get resource history", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"history":       logs,
		"count":         len(logs),
	})
}

// GetActionStats retrieves statistics about actions within a time period.
// GET /api/workspaces/:id/audit-logs/stats/actions
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) GetActionStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateParam := c.Query("start_date"); startDateParam != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateParam); err == nil {
			startDate = parsed
		}
	}

	if endDateParam := c.Query("end_date"); endDateParam != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateParam); err == nil {
			endDate = parsed
		}
	}

	counts, err := h.auditService.GetActionCount(c.Request.Context(), workspaceID, startDate, endDate)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get action count", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_date":    startDate,
		"end_date":      endDate,
		"action_counts": counts,
	})
}

// GetMostActiveUsers retrieves the most active users within a time period.
// GET /api/workspaces/:id/audit-logs/stats/active-users
// Required permission: manage_workspace or admin+ role
func (h *WorkspaceHandler) GetMostActiveUsers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateParam := c.Query("start_date"); startDateParam != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateParam); err == nil {
			startDate = parsed
		}
	}

	if endDateParam := c.Query("end_date"); endDateParam != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateParam); err == nil {
			endDate = parsed
		}
	}

	limit := 10 // Default top 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	users, err := h.auditService.GetMostActiveUsers(c.Request.Context(), workspaceID, startDate, endDate, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get most active users", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_date":   startDate,
		"end_date":     endDate,
		"active_users": users,
		"count":        len(users),
	})
}
