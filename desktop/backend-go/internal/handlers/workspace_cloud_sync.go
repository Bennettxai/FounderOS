package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// WorkspaceCloudSyncHandler exposes the owner-triggered local→cloud sync.
type WorkspaceCloudSyncHandler struct {
	pool *pgxpool.Pool
}

func NewWorkspaceCloudSyncHandler(pool *pgxpool.Pool) *WorkspaceCloudSyncHandler {
	return &WorkspaceCloudSyncHandler{pool: pool}
}

// SyncToCloud pushes the active workspace + members + invites + scoped data to
// the shared cloud DB. Owner/admin only, and only works on a machine that has
// CLOUD_DATABASE_URL configured (the owner's).
// POST /api/workspace/sync-to-cloud
func (h *WorkspaceCloudSyncHandler) SyncToCloud(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	hdr := c.GetHeader("X-Workspace-ID")
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		RespondBadRequestErr(c, "no active workspace")
		return
	}

	// Owner/admin only.
	var role string
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(role, role_name, '') FROM workspace_members
		 WHERE workspace_id=$1 AND user_id=$2 AND status='active'`, wsID, user.ID).Scan(&role)
	if err != nil {
		RespondUnauthorizedErr(c, "not a member of this workspace")
		return
	}
	if role != "owner" && role != "admin" {
		RespondForbiddenErr(c, "only an owner or admin can sync to cloud")
		return
	}

	if services.CloudDatabaseURL() == "" {
		RespondBadRequestErr(c, "cloud sync is not configured on this machine")
		return
	}

	report, err := services.NewWorkspaceCloudSync(h.pool).Sync(c.Request.Context(), wsID.String())
	if err != nil {
		RespondInternalErr(c, "sync to cloud", err)
		return
	}
	c.JSON(http.StatusOK, report)
}
