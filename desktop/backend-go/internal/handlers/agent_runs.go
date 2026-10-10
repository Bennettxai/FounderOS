package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// listAgentRuns returns agent execution history for the active workspace, newest
// first, plus a per-status count. Backs the Agents module's runs view.
// GET /api/ai/agent-runs  (X-Workspace-ID header required)
func (h *Handlers) listAgentRuns(c *gin.Context) {
	wsID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Workspace-ID header required"})
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id::text AS id, agent_name, task_name, company, status,
		       plan, destination, created_at
		FROM agent_runs
		WHERE workspace_id = $1
		ORDER BY created_at DESC`, wsID)
	if err != nil {
		slog.Default().Error("listAgentRuns query", "error", err, "workspace_id", wsID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agent runs"})
		return
	}
	defer rows.Close()

	runs := []map[string]interface{}{}
	counts := map[string]int{}
	fields := rows.FieldDescriptions()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			continue
		}
		m := make(map[string]interface{}, len(fields))
		for i, f := range fields {
			m[string(f.Name)] = vals[i]
		}
		if s, ok := m["status"].(string); ok {
			counts[s]++
		}
		runs = append(runs, m)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read agent runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runs": runs, "total": len(runs), "counts": counts})
}
