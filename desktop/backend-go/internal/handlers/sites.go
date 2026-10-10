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

// SitesHandler manages the per-workspace registry of the business's web
// properties (landing pages, marketing sites, apps), so humans and AI agents
// share one canonical list of where the business lives online. Workspace-scoped
// via X-Workspace-ID.
type SitesHandler struct {
	pool *pgxpool.Pool
}

func NewSitesHandler(pool *pgxpool.Pool) *SitesHandler {
	return &SitesHandler{pool: pool}
}

type site struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CTA       string    `json:"cta"`
	Notes     string    `json:"notes"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// normalizeSiteKind keeps the web-property kind within the allowed set.
func normalizeSiteKind(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "funnel":
		return "funnel"
	case "form":
		return "form"
	case "site":
		return "site"
	case "app":
		return "app"
	default:
		return "page"
	}
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *SitesHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListSites returns the workspace's sites, optionally filtered by ?q=.
// GET /api/v1/sites
func (h *SitesHandler) ListSites(c *gin.Context) {
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
			SELECT id, name, kind, url, status, cta, notes, created_by, created_at, updated_at
			FROM   sites
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR url ILIKE '%'||$2||'%' OR notes ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, url, status, cta, notes, created_by, created_at, updated_at
			FROM   sites
			WHERE  workspace_id = $1
			ORDER  BY name ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list sites", err)
		return
	}
	defer rows.Close()

	sites := make([]site, 0)
	for rows.Next() {
		var s site
		var id uuid.UUID
		if err := rows.Scan(&id, &s.Name, &s.Kind, &s.URL, &s.Status, &s.CTA, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan site", err)
			return
		}
		s.ID = id.String()
		sites = append(sites, s)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate sites", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sites": sites, "count": len(sites)})
}

type siteInput struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Status string `json:"status"`
	CTA    string `json:"cta"`
	Notes  string `json:"notes"`
}

// CreateSite adds a site. POST /api/v1/sites
func (h *SitesHandler) CreateSite(c *gin.Context) {
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
	var in siteInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "live"
	}

	var s site
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO sites (workspace_id, name, kind, url, status, cta, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, kind, url, status, cta, notes, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), normalizeSiteKind(in.Kind), in.URL, status, in.CTA, in.Notes, user.ID).
		Scan(&id, &s.Name, &s.Kind, &s.URL, &s.Status, &s.CTA, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create site", err)
		return
	}
	s.ID = id.String()
	c.JSON(http.StatusCreated, s)
}

// UpdateSite edits a site. PUT /api/v1/sites/:id
func (h *SitesHandler) UpdateSite(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid site id")
		return
	}
	var in siteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var s site
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE sites
		SET    name       = COALESCE(NULLIF($3,''), name),
		       kind       = $4,
		       url        = $5,
		       status     = COALESCE(NULLIF($6,''), status),
		       cta        = $7,
		       notes      = $8,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, kind, url, status, cta, notes, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), normalizeSiteKind(in.Kind), in.URL, strings.TrimSpace(in.Status), in.CTA, in.Notes).
		Scan(&rid, &s.Name, &s.Kind, &s.URL, &s.Status, &s.CTA, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "site")
		return
	}
	s.ID = rid.String()
	c.JSON(http.StatusOK, s)
}

// DeleteSite removes a site. DELETE /api/v1/sites/:id
func (h *SitesHandler) DeleteSite(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid site id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM sites WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete site", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "site")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}
