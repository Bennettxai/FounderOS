package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// ManageAnalyticsHandler powers the Manage > Analytics module: workspace-wide
// counts and simple trends computed live from the operator's real data (tasks,
// projects, clients, offers, content, campaigns). It is read-only and
// deterministic — no AI, no external calls. Workspace-scoped via X-Workspace-ID.
//
// It is deliberately mounted under /manage-analytics to avoid colliding with any
// other /analytics route group.
type ManageAnalyticsHandler struct {
	pool *pgxpool.Pool
}

func NewManageAnalyticsHandler(pool *pgxpool.Pool) *ManageAnalyticsHandler {
	return &ManageAnalyticsHandler{pool: pool}
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
// Copied from offers.go so this handler stays self-contained.
func (h *ManageAnalyticsHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// statCount is a single {label,count} bucket used for the simple bar charts.
type statCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// scalarCount runs a single-column count query scoped to the workspace. Any
// error (e.g. a table without workspace_id in an older DB) resolves to 0 so the
// summary degrades gracefully rather than failing the whole request.
func (h *ManageAnalyticsHandler) scalarCount(ctx context.Context, wsID uuid.UUID, q string) int {
	var n int
	if err := h.pool.QueryRow(ctx, q, wsID).Scan(&n); err != nil {
		return 0
	}
	return n
}

// groupCounts runs a `SELECT label, count(*) ... GROUP BY label` query and
// returns the buckets in the order they arrive. Errors resolve to an empty set.
func (h *ManageAnalyticsHandler) groupCounts(ctx context.Context, wsID uuid.UUID, q string) []statCount {
	out := []statCount{}
	rows, err := h.pool.Query(ctx, q, wsID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var s statCount
		if err := rows.Scan(&s.Label, &s.Count); err != nil {
			return out
		}
		out = append(out, s)
	}
	return out
}

// GetSummary returns workspace counts + simple trends.
// GET /api/manage-analytics/summary
func (h *ManageAnalyticsHandler) GetSummary(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	ctx := c.Request.Context()

	totals := gin.H{
		"tasks":     h.scalarCount(ctx, wsID, `SELECT count(*) FROM tasks WHERE workspace_id=$1`),
		"projects":  h.scalarCount(ctx, wsID, `SELECT count(*) FROM projects WHERE workspace_id=$1`),
		"clients":   h.scalarCount(ctx, wsID, `SELECT count(*) FROM clients WHERE workspace_id=$1`),
		"offers":    h.scalarCount(ctx, wsID, `SELECT count(*) FROM offers WHERE workspace_id=$1`),
		"content":   h.scalarCount(ctx, wsID, `SELECT count(*) FROM content_items WHERE workspace_id=$1`),
		"campaigns": h.scalarCount(ctx, wsID, `SELECT count(*) FROM campaigns WHERE workspace_id=$1`),
	}

	tasksByStatus := h.groupCounts(ctx, wsID,
		`SELECT status::text, count(*) FROM tasks WHERE workspace_id=$1 GROUP BY status::text ORDER BY count(*) DESC`)
	projectsByStatus := h.groupCounts(ctx, wsID,
		`SELECT status::text, count(*) FROM projects WHERE workspace_id=$1 GROUP BY status::text ORDER BY count(*) DESC`)
	clientsByStatus := h.groupCounts(ctx, wsID,
		`SELECT status::text, count(*) FROM clients WHERE workspace_id=$1 GROUP BY status::text ORDER BY count(*) DESC`)

	trends := gin.H{
		"tasks_completed_30d": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND completed_at IS NOT NULL AND completed_at >= now() - interval '30 days'`),
		"tasks_created_30d": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND created_at >= now() - interval '30 days'`),
		"tasks_open": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status<>'done' AND status<>'cancelled' AND completed_at IS NULL`),
		"tasks_overdue": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status<>'done' AND completed_at IS NULL AND due_date IS NOT NULL AND due_date < now()`),
		"projects_active": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM projects WHERE workspace_id=$1 AND status='ACTIVE'`),
		"clients_active": h.scalarCount(ctx, wsID,
			`SELECT count(*) FROM clients WHERE workspace_id=$1 AND status='active'`),
	}

	c.JSON(http.StatusOK, gin.H{
		"totals":             totals,
		"tasks_by_status":    tasksByStatus,
		"projects_by_status": projectsByStatus,
		"clients_by_status":  clientsByStatus,
		"trends":             trends,
	})
}
