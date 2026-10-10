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

// GlossaryHandler manages the per-workspace business glossary: the team's own
// definitions of terms (what "proposal" means for us, etc.) so humans and AI
// agents decode language the same way. Workspace-scoped via X-Workspace-ID.
type GlossaryHandler struct {
	pool *pgxpool.Pool
}

func NewGlossaryHandler(pool *pgxpool.Pool) *GlossaryHandler {
	return &GlossaryHandler{pool: pool}
}

type glossaryTerm struct {
	ID         string    `json:"id"`
	Term       string    `json:"term"`
	Definition string    `json:"definition"`
	Category   *string   `json:"category"`
	Aliases    *string   `json:"aliases"`
	CreatedBy  *string   `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *GlossaryHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListTerms returns the workspace's glossary, optionally filtered by ?q=.
// GET /api/v1/glossary
func (h *GlossaryHandler) ListTerms(c *gin.Context) {
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
			SELECT id, term, definition, category, aliases, created_by, created_at, updated_at
			FROM   glossary_terms
			WHERE  workspace_id = $1
			AND   (term ILIKE '%'||$2||'%' OR definition ILIKE '%'||$2||'%' OR COALESCE(aliases,'') ILIKE '%'||$2||'%')
			ORDER  BY term ASC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, term, definition, category, aliases, created_by, created_at, updated_at
			FROM   glossary_terms
			WHERE  workspace_id = $1
			ORDER  BY term ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list glossary terms", err)
		return
	}
	defer rows.Close()

	terms := make([]glossaryTerm, 0)
	for rows.Next() {
		var t glossaryTerm
		var id uuid.UUID
		if err := rows.Scan(&id, &t.Term, &t.Definition, &t.Category, &t.Aliases, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan glossary term", err)
			return
		}
		t.ID = id.String()
		terms = append(terms, t)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate glossary terms", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"terms": terms, "count": len(terms)})
}

type glossaryInput struct {
	Term       string  `json:"term"`
	Definition string  `json:"definition"`
	Category   *string `json:"category"`
	Aliases    *string `json:"aliases"`
}

// CreateTerm adds a term. POST /api/v1/glossary
func (h *GlossaryHandler) CreateTerm(c *gin.Context) {
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
	var in glossaryInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Term) == "" {
		RespondBadRequestErr(c, "term is required")
		return
	}

	var t glossaryTerm
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO glossary_terms (workspace_id, term, definition, category, aliases, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (workspace_id, term) DO UPDATE SET
			definition = EXCLUDED.definition,
			category   = EXCLUDED.category,
			aliases    = EXCLUDED.aliases,
			updated_at = NOW()
		RETURNING id, term, definition, category, aliases, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Term), in.Definition, in.Category, in.Aliases, user.ID).
		Scan(&id, &t.Term, &t.Definition, &t.Category, &t.Aliases, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create glossary term", err)
		return
	}
	t.ID = id.String()
	c.JSON(http.StatusCreated, t)
}

// UpdateTerm edits a term. PUT /api/v1/glossary/:id
func (h *GlossaryHandler) UpdateTerm(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid term id")
		return
	}
	var in glossaryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var t glossaryTerm
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE glossary_terms
		SET    term       = COALESCE(NULLIF($3,''), term),
		       definition = $4,
		       category   = $5,
		       aliases    = $6,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, term, definition, category, aliases, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Term), in.Definition, in.Category, in.Aliases).
		Scan(&rid, &t.Term, &t.Definition, &t.Category, &t.Aliases, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "term")
		return
	}
	t.ID = rid.String()
	c.JSON(http.StatusOK, t)
}

// DeleteTerm removes a term. DELETE /api/v1/glossary/:id
func (h *GlossaryHandler) DeleteTerm(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid term id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM glossary_terms WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete glossary term", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "term")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "term deleted"})
}
