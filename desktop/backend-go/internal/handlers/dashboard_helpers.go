package handlers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// stringToTaskStatus converts a string to sqlc.Taskstatus
func stringToTaskStatus(s string) sqlc.Taskstatus {
	typeMap := map[string]sqlc.Taskstatus{
		"todo":        sqlc.TaskstatusTodo,
		"in_progress": sqlc.TaskstatusInProgress,
		"done":        sqlc.TaskstatusDone,
		"cancelled":   sqlc.TaskstatusCancelled,
	}
	if enum, ok := typeMap[strings.ToLower(s)]; ok {
		return enum
	}
	return sqlc.TaskstatusTodo
}

// stringToTaskPriority converts a string to sqlc.Taskpriority
func stringToTaskPriority(p string) sqlc.Taskpriority {
	typeMap := map[string]sqlc.Taskpriority{
		"critical": sqlc.TaskpriorityCritical,
		"high":     sqlc.TaskpriorityHigh,
		"medium":   sqlc.TaskpriorityMedium,
		"low":      sqlc.TaskpriorityLow,
	}
	if enum, ok := typeMap[strings.ToLower(p)]; ok {
		return enum
	}
	return sqlc.TaskpriorityMedium
}

// invalidateDashboardCache invalidates all dashboard caches for a user
// This should be called whenever dashboard-related data changes (tasks, projects, etc.)
func (h *DashboardItemHandler) invalidateDashboardCache(c *gin.Context, userID string) {
	if h.queryCache == nil {
		return // Cache not available, nothing to invalidate
	}

	// Invalidate all dashboard cache keys for the user
	cachePatterns := []string{
		fmt.Sprintf("dashboard:user:%s:summary", userID),    // Dashboard summary (1min TTL)
		fmt.Sprintf("dashboard:user:%s:tasks", userID),      // Dashboard tasks (2min TTL)
		fmt.Sprintf("dashboard:user:%s:activities", userID), // Recent activities (5min TTL)
	}

	for _, pattern := range cachePatterns {
		if err := h.queryCache.Delete(c.Request.Context(), pattern); err != nil {
			slog.Default().Warn("Failed to invalidate dashboard cache",
				"user_id", userID,
				"pattern", pattern,
				"error", err)
		}
	}

	slog.Default().Debug("Dashboard cache invalidated",
		"user_id", userID,
		"patterns_invalidated", len(cachePatterns))
}
