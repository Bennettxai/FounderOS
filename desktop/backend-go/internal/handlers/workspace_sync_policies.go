package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// PER-MODULE SYNC POLICIES (local-first, sync the whole module or not)
//
// Each module's data is LOCAL by default (lives on the user's machine). Flip a
// module to "workspace" and its whole dataset replicates to the cloud DB so the
// web app and teammates can see it. All-or-nothing per module - no per-item
// sync. Stored in sync_policies (workspace_id, module, sync_mode); the write-side
// stamping (contexts_workspace.go) reads it to decide what becomes shared.
// =====================================================================

// syncableModule is a module that can be local-only or synced to the workspace.
// Keys match the module names used by the write-side stamping.
type syncableModule struct {
	Key   string
	Label string
}

var syncableModules = []syncableModule{
	{"knowledge", "Knowledge"},
	{"projects", "Projects"},
	{"tasks", "Tasks"},
	{"clients", "Relationships"},
	{"calendar", "Calendar"},
	{"deals", "Pipelines"},
}

type syncPolicyItem struct {
	Module   string `json:"module"`
	Label    string `json:"label"`
	SyncMode string `json:"sync_mode"` // "local" | "workspace"
}

// GetWorkspaceSyncPolicies returns each module's sync mode (defaulting unset
// modules to "local"), plus auto_sync_all = true when every module is synced.
// GET /api/workspaces/:id/sync-policies  (any member)
func (h *WorkspaceHandler) GetWorkspaceSyncPolicies(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}
	if _, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	modes := map[string]string{}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT module, sync_mode FROM sync_policies WHERE workspace_id=$1`, workspaceID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list sync policies", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var m, mode string
		if err := rows.Scan(&m, &mode); err == nil {
			modes[m] = mode
		}
	}

	items := make([]syncPolicyItem, 0, len(syncableModules))
	allSynced := true
	for _, sm := range syncableModules {
		mode := modes[sm.Key]
		if mode == "" {
			mode = "local"
		}
		if mode != "workspace" {
			allSynced = false
		}
		items = append(items, syncPolicyItem{Module: sm.Key, Label: sm.Label, SyncMode: mode})
	}
	c.JSON(http.StatusOK, gin.H{"policies": items, "auto_sync_all": allSynced})
}

// SetWorkspaceSyncPolicies flips a single module (or all modules) between local
// and workspace (synced). Owner/admin/manager only.
// PUT /api/workspaces/:id/sync-policies
//
//	Body: { "module": "projects", "sync_mode": "workspace" }   (single)
//	  or: { "all": "workspace" }   |   { "all": "local" }       (every module)
func (h *WorkspaceHandler) SetWorkspaceSyncPolicies(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners, admins, and managers can change sync settings"})
		return
	}

	var body struct {
		Module   string `json:"module"`
		SyncMode string `json:"sync_mode"`
		All      string `json:"all"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondBadRequest(c, slog.Default(), "invalid request body")
		return
	}

	normalize := func(m string) string {
		if m == "workspace" {
			return "workspace"
		}
		return "local"
	}

	upsert := func(module, mode string) error {
		_, err := h.pool.Exec(c.Request.Context(), `
			INSERT INTO sync_policies (workspace_id, module, sync_mode, is_published)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (workspace_id, module)
			DO UPDATE SET sync_mode = EXCLUDED.sync_mode,
			              is_published = EXCLUDED.is_published,
			              updated_at = NOW()
		`, workspaceID, module, mode, mode == "workspace")
		return err
	}

	if body.All != "" {
		mode := normalize(body.All)
		for _, sm := range syncableModules {
			if err := upsert(sm.Key, mode); err != nil {
				utils.RespondInternalError(c, slog.Default(), "set sync policy", err)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if body.Module == "" {
		utils.RespondBadRequest(c, slog.Default(), "module is required")
		return
	}
	if err := upsert(body.Module, normalize(body.SyncMode)); err != nil {
		utils.RespondInternalError(c, slog.Default(), "set sync policy", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
