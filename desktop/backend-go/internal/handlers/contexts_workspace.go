package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stampWorkspaceOnCreate is the generic BusinessOSSync write-side helper used by
// any module's create handler (projects, tasks, ...). It resolves the
// X-Workspace-ID header, confirms active membership, checks the module's sync
// policy, and if the module is in workspace mode, stamps the just-created row's
// workspace_id so every member sees it. Returns the workspace id string (for the
// engine Signal) or "" when the row stays private. table/idColumn are caller
// constants (never user input), so the interpolation is safe.
func stampWorkspaceOnCreate(c *gin.Context, pool *pgxpool.Pool, table string, id pgtype.UUID, userID, module string) string {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return ""
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return ""
	}
	ctx := c.Request.Context()
	var member bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member); err != nil || !member {
		return ""
	}
	var mode string
	if err := pool.QueryRow(ctx,
		`SELECT sync_mode FROM sync_policies WHERE workspace_id=$1 AND module=$2`, wsID, module).Scan(&mode); err != nil || mode != "workspace" {
		return ""
	}
	if _, err := pool.Exec(ctx,
		`UPDATE `+table+` SET workspace_id=$1 WHERE id=$2 AND user_id=$3`, wsID, id, userID); err != nil {
		slog.Warn("[BusinessOSSync] failed to stamp workspace_id", "table", table, "error", err)
		return ""
	}
	return wsID.String()
}

// BusinessOSSync — workspace scoping for knowledge docs (contexts).
//
// A doc is private (user-only) by default. When the request carries an
// X-Workspace-ID header for a workspace the user actively belongs to, AND that
// workspace's "knowledge" module is in workspace sync mode (sync_policies), the
// doc is stamped with workspace_id and becomes visible to every member of that
// workspace. Local-mode modules never stamp, so their docs stay private.

const knowledgeModule = "knowledge"

// workspaceFromHeader resolves the X-Workspace-ID header and confirms the user
// is an active member. Returns (workspaceID, true) only when authorized.
func (h *ContextHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// moduleSharesToWorkspace reports whether a module syncs to the workspace
// (sync_policies.sync_mode = 'workspace'). A missing policy means local-only.
func (h *ContextHandler) moduleSharesToWorkspace(c *gin.Context, ws uuid.UUID, module string) bool {
	var mode string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT sync_mode FROM sync_policies WHERE workspace_id=$1 AND module=$2`,
		ws, module).Scan(&mode)
	return err == nil && mode == "workspace"
}

// shareContextToWorkspace stamps a context with the workspace so all members see
// it. No-op (and safe) when there is no authorized workspace or the module is
// local-only. Returns the workspace id string to attach to the engine Signal.
func (h *ContextHandler) shareContextToWorkspace(c *gin.Context, ctxID pgtype.UUID, userID string) string {
	ws, ok := h.workspaceFromHeader(c, userID)
	if !ok || !h.moduleSharesToWorkspace(c, ws, knowledgeModule) {
		return ""
	}
	if _, err := h.pool.Exec(c.Request.Context(),
		`UPDATE contexts SET workspace_id=$1 WHERE id=$2 AND user_id=$3`,
		ws, ctxID, userID); err != nil {
		slog.Warn("[Contexts] failed to stamp workspace_id", "error", err, "context_id", ctxID)
		return ""
	}
	return ws.String()
}
