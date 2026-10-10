package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// ManageDataHandler powers the Manage > Data module: a read-only inventory of
// every workspace-scoped data entity and how many records it holds. It performs
// no destructive actions — export is surfaced as "coming soon" in the UI.
// Workspace-scoped via X-Workspace-ID.
//
// Mounted under /manage-data to avoid colliding with any other /data route group.
type ManageDataHandler struct {
	pool *pgxpool.Pool
}

func NewManageDataHandler(pool *pgxpool.Pool) *ManageDataHandler {
	return &ManageDataHandler{pool: pool}
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
// Copied from offers.go so this handler stays self-contained.
func (h *ManageDataHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// dataEntity describes one workspace data table and its live record count.
type dataEntity struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Table string `json:"table"`
	Count int    `json:"count"`
}

// managedEntities is the fixed catalog of workspace-scoped tables surfaced in
// the Data module. Keeping it as a static list (rather than reading
// information_schema) means we only ever count tables we know are workspace
// scoped and safe to display.
var managedEntities = []dataEntity{
	{Key: "tasks", Label: "Tasks", Table: "tasks"},
	{Key: "projects", Label: "Projects", Table: "projects"},
	{Key: "clients", Label: "Clients", Table: "clients"},
	{Key: "offers", Label: "Offers", Table: "offers"},
	{Key: "content", Label: "Content", Table: "content_items"},
	{Key: "campaigns", Label: "Campaigns", Table: "campaigns"},
	{Key: "personas", Label: "Personas", Table: "personas"},
	{Key: "assets", Label: "Assets", Table: "assets"},
	{Key: "resources", Label: "Resources", Table: "resources"},
}

// countTable returns the workspace-scoped row count for a table. Any error
// (missing table/column) resolves to 0 so the inventory never fails wholesale.
func (h *ManageDataHandler) countTable(ctx context.Context, wsID uuid.UUID, table string) int {
	var n int
	// #nosec G201 — table names come only from the static managedEntities list.
	q := "SELECT count(*) FROM " + table + " WHERE workspace_id=$1"
	if err := h.pool.QueryRow(ctx, q, wsID).Scan(&n); err != nil {
		return 0
	}
	return n
}

// GetSummary returns a row per data entity with its record count.
// GET /api/manage-data/summary
func (h *ManageDataHandler) GetSummary(c *gin.Context) {
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

	entities := make([]dataEntity, 0, len(managedEntities))
	total := 0
	for _, e := range managedEntities {
		e.Count = h.countTable(ctx, wsID, e.Table)
		total += e.Count
		entities = append(entities, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"entities":     entities,
		"total":        total,
		"export_ready": false,
	})
}
