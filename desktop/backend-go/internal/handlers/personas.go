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

// PersonasHandler manages the per-workspace buyer/customer personas: ideal
// customer profiles (segment, pains, objections, language) so humans and AI
// agents target messaging and offers consistently. Workspace-scoped via
// X-Workspace-ID.
type PersonasHandler struct {
	pool *pgxpool.Pool
}

func NewPersonasHandler(pool *pgxpool.Pool) *PersonasHandler {
	return &PersonasHandler{pool: pool}
}

type persona struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Segment    string    `json:"segment"`
	Fit        string    `json:"fit"`
	Pains      string    `json:"pains"`
	Objections string    `json:"objections"`
	Language   string    `json:"language"`
	Notes      string    `json:"notes"`
	CreatedBy  *string   `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// normalizeFit keeps the ICP fit flag within best | poor.
func normalizeFit(s string) string {
	if strings.ToLower(strings.TrimSpace(s)) == "poor" {
		return "poor"
	}
	return "best"
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *PersonasHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListPersonas returns the workspace's personas, optionally filtered by ?q=.
// GET /api/v1/personas
func (h *PersonasHandler) ListPersonas(c *gin.Context) {
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
			SELECT id, name, segment, fit, pains, objections, language, notes, created_by, created_at, updated_at
			FROM   personas
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR segment ILIKE '%'||$2||'%' OR pains ILIKE '%'||$2||'%' OR objections ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, segment, fit, pains, objections, language, notes, created_by, created_at, updated_at
			FROM   personas
			WHERE  workspace_id = $1
			ORDER  BY name ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list personas", err)
		return
	}
	defer rows.Close()

	personas := make([]persona, 0)
	for rows.Next() {
		var p persona
		var id uuid.UUID
		if err := rows.Scan(&id, &p.Name, &p.Segment, &p.Fit, &p.Pains, &p.Objections, &p.Language, &p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan persona", err)
			return
		}
		p.ID = id.String()
		personas = append(personas, p)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate personas", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"personas": personas, "count": len(personas)})
}

type personaInput struct {
	Name       string `json:"name"`
	Segment    string `json:"segment"`
	Fit        string `json:"fit"`
	Pains      string `json:"pains"`
	Objections string `json:"objections"`
	Language   string `json:"language"`
	Notes      string `json:"notes"`
}

// CreatePersona adds a persona. POST /api/v1/personas
func (h *PersonasHandler) CreatePersona(c *gin.Context) {
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
	var in personaInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}

	var p persona
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO personas (workspace_id, name, segment, fit, pains, objections, language, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, segment, fit, pains, objections, language, notes, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), in.Segment, normalizeFit(in.Fit), in.Pains, in.Objections, in.Language, in.Notes, user.ID).
		Scan(&id, &p.Name, &p.Segment, &p.Fit, &p.Pains, &p.Objections, &p.Language, &p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create persona", err)
		return
	}
	p.ID = id.String()
	c.JSON(http.StatusCreated, p)
}

// UpdatePersona edits a persona. PUT /api/v1/personas/:id
func (h *PersonasHandler) UpdatePersona(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid persona id")
		return
	}
	var in personaInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var p persona
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE personas
		SET    name       = COALESCE(NULLIF($3,''), name),
		       segment    = $4,
		       fit        = $5,
		       pains      = $6,
		       objections = $7,
		       language   = $8,
		       notes      = $9,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, segment, fit, pains, objections, language, notes, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), in.Segment, normalizeFit(in.Fit), in.Pains, in.Objections, in.Language, in.Notes).
		Scan(&rid, &p.Name, &p.Segment, &p.Fit, &p.Pains, &p.Objections, &p.Language, &p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "persona")
		return
	}
	p.ID = rid.String()
	c.JSON(http.StatusOK, p)
}

// DeletePersona removes a persona. DELETE /api/v1/personas/:id
func (h *PersonasHandler) DeletePersona(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid persona id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM personas WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete persona", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "persona")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "persona deleted"})
}
