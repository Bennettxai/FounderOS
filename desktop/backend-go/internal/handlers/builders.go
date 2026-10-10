package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// BuildersHandler manages the per-workspace Builders registry: the tools for
// creating modules, apps, workflows, sites, and automations. Each builder is a
// catalog entry (name, kind, status, config) so the team and AI agents share
// one source of truth for what can be built. Workspace-scoped via X-Workspace-ID.
type BuildersHandler struct {
	pool *pgxpool.Pool
}

func NewBuildersHandler(pool *pgxpool.Pool) *BuildersHandler {
	return &BuildersHandler{pool: pool}
}

type builder struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	Status      string          `json:"status"`
	CreatedBy   *string         `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *BuildersHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// normalizeBuilderKind keeps the builder kind within the allowed set.
func normalizeBuilderKind(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "flow":
		return "flow"
	case "automation":
		return "automation"
	case "app":
		return "app"
	case "site":
		return "site"
	default:
		return "form"
	}
}

// normalizeBuilderStatus keeps status within the allowed set, defaulting to draft.
func normalizeBuilderStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active":
		return "active"
	case "archived":
		return "archived"
	default:
		return "draft"
	}
}

// ListBuilders returns the workspace's builders, optionally filtered by ?q= and ?kind=.
// GET /api/v1/builders
func (h *BuildersHandler) ListBuilders(c *gin.Context) {
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

	q := strings.TrimSpace(c.Query("q"))
	kind := strings.TrimSpace(c.Query("kind"))

	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	var err error
	switch {
	case q != "" && kind != "":
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, description, config, status, created_by, created_at, updated_at
			FROM   builders
			WHERE  workspace_id = $1
			AND    kind = $3
			AND   (name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q, normalizeBuilderKind(kind))
	case q != "":
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, description, config, status, created_by, created_at, updated_at
			FROM   builders
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q)
	case kind != "":
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, description, config, status, created_by, created_at, updated_at
			FROM   builders
			WHERE  workspace_id = $1
			AND    kind = $2
			ORDER  BY name ASC
		`, wsID, normalizeBuilderKind(kind))
	default:
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, description, config, status, created_by, created_at, updated_at
			FROM   builders
			WHERE  workspace_id = $1
			ORDER  BY name ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list builders", err)
		return
	}
	defer rows.Close()

	builders := make([]builder, 0)
	for rows.Next() {
		var b builder
		var id uuid.UUID
		if err := rows.Scan(&id, &b.Name, &b.Kind, &b.Description, &b.Config, &b.Status, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan builder", err)
			return
		}
		b.ID = id.String()
		builders = append(builders, b)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate builders", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"builders": builders, "count": len(builders)})
}

type builderInput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// CreateBuilder adds a builder. POST /api/v1/builders
func (h *BuildersHandler) CreateBuilder(c *gin.Context) {
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
	var in builderInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}

	var b builder
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO builders (workspace_id, name, kind, description, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, kind, description, config, status, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), normalizeBuilderKind(in.Kind), in.Description, normalizeBuilderStatus(in.Status), user.ID).
		Scan(&id, &b.Name, &b.Kind, &b.Description, &b.Config, &b.Status, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create builder", err)
		return
	}
	b.ID = id.String()
	c.JSON(http.StatusCreated, b)
}

// UpdateBuilder edits a builder. PUT /api/v1/builders/:id
func (h *BuildersHandler) UpdateBuilder(c *gin.Context) {
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
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid builder id")
		return
	}
	var in builderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var b builder
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE builders
		SET    name        = COALESCE(NULLIF($3,''), name),
		       kind        = $4,
		       description = $5,
		       status      = $6,
		       updated_at  = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, kind, description, config, status, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), normalizeBuilderKind(in.Kind), in.Description, normalizeBuilderStatus(in.Status)).
		Scan(&rid, &b.Name, &b.Kind, &b.Description, &b.Config, &b.Status, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "builder")
		return
	}
	b.ID = rid.String()
	c.JSON(http.StatusOK, b)
}

// DeleteBuilder removes a builder. DELETE /api/v1/builders/:id
func (h *BuildersHandler) DeleteBuilder(c *gin.Context) {
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
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid builder id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM builders WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete builder", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "builder")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "builder deleted"})
}
