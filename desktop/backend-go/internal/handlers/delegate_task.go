package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// delegateTask turns a Dalya request into real work: it creates a task in the
// active workspace AND an agent run (status awaiting_approval) so the item shows
// up in Tasks and in the Agents module's approval queue — mirroring how Dany OS
// Dalya hands work to the agent daemon.
// POST /api/ai/delegate  body: {"task": "..."}  (X-Workspace-ID header required)
func (h *Handlers) delegateTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	wsID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Workspace-ID header required"})
		return
	}
	var req struct {
		Task string `json:"task"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Task) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task text required"})
		return
	}
	task := strings.TrimSpace(req.Task)
	ctx := c.Request.Context()

	var taskID string
	if err := h.pool.QueryRow(ctx, `
		INSERT INTO tasks (user_id, workspace_id, title, status, priority, description)
		VALUES ($1, $2, $3, 'todo', 'medium', 'Delegated to Dalya from Command')
		RETURNING id::text`, user.ID, wsID, task).Scan(&taskID); err != nil {
		slog.Default().Error("delegateTask: insert task", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	var runID string
	if err := h.pool.QueryRow(ctx, `
		INSERT INTO agent_runs (workspace_id, agent_name, task_name, company, status)
		VALUES ($1, 'Agent Daemon', $2, 'dany-os', 'awaiting_approval')
		RETURNING id::text`, wsID, task).Scan(&runID); err != nil {
		slog.Default().Error("delegateTask: insert agent run", "error", err)
		// task was created; still report partial success
		c.JSON(http.StatusOK, gin.H{"ok": true, "task_id": taskID, "run_id": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "task_id": taskID, "run_id": runID})
}
