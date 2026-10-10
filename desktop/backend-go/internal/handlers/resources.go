package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// ResourcesHandler manages the per-workspace resource library: a curated list of
// links, docs, SOPs, and tools the team relies on (title, url, category, notes).
// Workspace-scoped via X-Workspace-ID.
type ResourcesHandler struct {
	pool *pgxpool.Pool
}

func NewResourcesHandler(pool *pgxpool.Pool) *ResourcesHandler {
	return &ResourcesHandler{pool: pool}
}

type resource struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Category  string    `json:"category"`
	Notes     string    `json:"notes"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
// Copied from offers.go so this handler stays self-contained.
func (h *ResourcesHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListResources returns the workspace's resources, optionally filtered by ?q=.
// GET /api/resources
func (h *ResourcesHandler) ListResources(c *gin.Context) {
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
	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	var err error
	if q != "" {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, title, url, category, notes, created_by, created_at, updated_at
			FROM   resources
			WHERE  workspace_id = $1
			AND   (title ILIKE '%'||$2||'%' OR url ILIKE '%'||$2||'%' OR category ILIKE '%'||$2||'%' OR notes ILIKE '%'||$2||'%')
			ORDER  BY created_at DESC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, title, url, category, notes, created_by, created_at, updated_at
			FROM   resources
			WHERE  workspace_id = $1
			ORDER  BY created_at DESC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list resources", err)
		return
	}
	defer rows.Close()

	resources := make([]resource, 0)
	for rows.Next() {
		var r resource
		var id uuid.UUID
		if err := rows.Scan(&id, &r.Title, &r.URL, &r.Category, &r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan resource", err)
			return
		}
		r.ID = id.String()
		resources = append(resources, r)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate resources", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": resources, "count": len(resources)})
}

type resourceInput struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Category string `json:"category"`
	Notes    string `json:"notes"`
}

// CreateResource adds a resource. POST /api/resources
func (h *ResourcesHandler) CreateResource(c *gin.Context) {
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
	var in resourceInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Title) == "" {
		RespondBadRequestErr(c, "title is required")
		return
	}

	var r resource
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO resources (workspace_id, title, url, category, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, url, category, notes, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Title), strings.TrimSpace(in.URL), strings.TrimSpace(in.Category), in.Notes, user.ID).
		Scan(&id, &r.Title, &r.URL, &r.Category, &r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create resource", err)
		return
	}
	r.ID = id.String()
	c.JSON(http.StatusCreated, r)
}

// UpdateResource edits a resource. PUT /api/resources/:id
func (h *ResourcesHandler) UpdateResource(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid resource id")
		return
	}
	var in resourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var r resource
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE resources
		SET    title      = COALESCE(NULLIF($3,''), title),
		       url        = $4,
		       category   = $5,
		       notes      = $6,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, title, url, category, notes, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Title), strings.TrimSpace(in.URL), strings.TrimSpace(in.Category), in.Notes).
		Scan(&rid, &r.Title, &r.URL, &r.Category, &r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "resource")
		return
	}
	r.ID = rid.String()
	c.JSON(http.StatusOK, r)
}

// DeleteResource removes a resource. DELETE /api/resources/:id
func (h *ResourcesHandler) DeleteResource(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid resource id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM resources WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete resource", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "resource")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resource deleted"})
}
