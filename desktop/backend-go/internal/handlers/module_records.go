package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// ModuleRecordsHandler handles record CRUD for "kind: records" custom modules.
type ModuleRecordsHandler struct {
	pool *pgxpool.Pool
}

// NewModuleRecordsHandler returns a handler backed by the given pool.
func NewModuleRecordsHandler(pool *pgxpool.Pool) *ModuleRecordsHandler {
	return &ModuleRecordsHandler{pool: pool}
}

// moduleRecordRow is the canonical JSON shape returned to callers.
type moduleRecordRow struct {
	ID          string          `json:"id"`
	ModuleID    string          `json:"module_id"`
	WorkspaceID string          `json:"workspace_id"`
	CreatedBy   *string         `json:"created_by"`
	Data        json.RawMessage `json:"data"`
	Position    int             `json:"position"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// operatingWorkspace resolves the workspace the caller is currently working in
// from the X-Workspace-ID header (sent by the frontend on every request) and
// verifies active membership. This is the PARTITION KEY for module data: a
// module can live in / be installed into several workspaces, but each
// workspace's records are isolated by this id, so data never bleeds across
// workspaces. Returns (nil,false) if the header is absent/invalid or the caller
// is not an active member — callers translate that to 404.
func (h *ModuleRecordsHandler) operatingWorkspace(c *gin.Context, userID string) (uuid.UUID, bool) {
	wsID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		return uuid.Nil, false
	}
	var isMember bool
	if err := h.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members
			WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
		)`, wsID, userID).Scan(&isMember); err != nil || !isMember {
		return uuid.Nil, false
	}
	return wsID, true
}

// moduleAvailableInWorkspace reports whether module m is usable from workspace w:
// w is the module's home workspace, an enabled installation exists for (m, w),
// or the module is org-shared and w is in the module's organization.
func (h *ModuleRecordsHandler) moduleAvailableInWorkspace(ctx context.Context, moduleID, wsID uuid.UUID) bool {
	var available bool
	err := h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM custom_modules m
			LEFT JOIN module_installations i
			       ON i.module_id = m.id AND i.workspace_id = $2 AND i.is_enabled = TRUE
			LEFT JOIN workspaces wm ON wm.id = m.workspace_id
			LEFT JOIN workspaces ww ON ww.id = $2
			WHERE m.id = $1
			  AND (
			      m.workspace_id = $2
			      OR i.module_id IS NOT NULL
			      OR (m.share_scope = 'organization' AND wm.organization_id = ww.organization_id)
			  )
		)`, moduleID, wsID).Scan(&available)
	return err == nil && available
}

// ListRecords returns all records for a module, ordered by position then created_at.
// GET /api/modules/:id/records
func (h *ModuleRecordsHandler) ListRecords(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.operatingWorkspace(c, user.ID)
	if !ok || !h.moduleAvailableInWorkspace(c.Request.Context(), moduleID, wsID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, module_id, workspace_id, created_by, data, position, created_at, updated_at
		FROM   module_records
		WHERE  module_id = $1 AND workspace_id = $2
		ORDER  BY position ASC, created_at ASC
	`, moduleID, wsID)
	if err != nil {
		RespondInternalErr(c, "list module records", err)
		return
	}
	defer rows.Close()

	records := make([]moduleRecordRow, 0)
	for rows.Next() {
		var r moduleRecordRow
		var rawData []byte
		if err := rows.Scan(&r.ID, &r.ModuleID, &r.WorkspaceID, &r.CreatedBy, &rawData, &r.Position, &r.CreatedAt, &r.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan module record", err)
			return
		}
		r.Data = json.RawMessage(rawData)
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate module records", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records, "count": len(records)})
}

// CreateRecord inserts a new record into the module's record store.
// POST /api/modules/:id/records  body: { "data": { ... } }
func (h *ModuleRecordsHandler) CreateRecord(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.operatingWorkspace(c, user.ID)
	if !ok || !h.moduleAvailableInWorkspace(c.Request.Context(), moduleID, wsID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Data) == 0 {
		body.Data = json.RawMessage("{}")
	}

	// Records are stamped with the OPERATING workspace (not the module's home),
	// so an installed module accrues its own isolated data in each workspace.
	var r moduleRecordRow
	var rawData []byte
	err = h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO module_records (module_id, workspace_id, created_by, data)
		VALUES ($1, $2, $3, $4)
		RETURNING id, module_id, workspace_id, created_by, data, position, created_at, updated_at
	`, moduleID, wsID, user.ID, []byte(body.Data)).Scan(
		&r.ID, &r.ModuleID, &r.WorkspaceID, &r.CreatedBy, &rawData, &r.Position, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		RespondInternalErr(c, "create module record", err)
		return
	}
	r.Data = json.RawMessage(rawData)

	c.JSON(http.StatusCreated, r)
}

// UpdateRecord replaces the data (and optionally position) of an existing record.
// PUT /api/modules/:id/records/:recordId  body: { "data": { ... }, "position": 2 }
func (h *ModuleRecordsHandler) UpdateRecord(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}
	recordID, err := uuid.Parse(c.Param("recordId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.operatingWorkspace(c, user.ID)
	if !ok || !h.moduleAvailableInWorkspace(c.Request.Context(), moduleID, wsID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	var body struct {
		Data     json.RawMessage `json:"data"`
		Position *int            `json:"position"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var r moduleRecordRow
	var rawData []byte
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE module_records
		SET    data       = COALESCE($3, data),
		       position   = COALESCE($4, position),
		       updated_at = NOW()
		WHERE  id         = $1
		AND    module_id  = $2
		AND    workspace_id = $5
		RETURNING id, module_id, workspace_id, created_by, data, position, created_at, updated_at
	`, recordID, moduleID, nullableJSON(body.Data), body.Position, wsID).Scan(
		&r.ID, &r.ModuleID, &r.WorkspaceID, &r.CreatedBy, &rawData, &r.Position, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		RespondNotFoundErr(c, "record")
		return
	}
	r.Data = json.RawMessage(rawData)

	c.JSON(http.StatusOK, r)
}

// DeleteRecord removes a record from the module's record store.
// DELETE /api/modules/:id/records/:recordId
func (h *ModuleRecordsHandler) DeleteRecord(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}
	recordID, err := uuid.Parse(c.Param("recordId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.operatingWorkspace(c, user.ID)
	if !ok || !h.moduleAvailableInWorkspace(c.Request.Context(), moduleID, wsID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	tag, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM module_records WHERE id = $1 AND module_id = $2 AND workspace_id = $3
	`, recordID, moduleID, wsID)
	if err != nil {
		RespondInternalErr(c, "delete module record", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "record")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "record deleted"})
}

// nullableJSON returns nil when raw is empty/null so that COALESCE in the UPDATE
// leaves the existing column value untouched when data is not provided.
func nullableJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return []byte(raw)
}
