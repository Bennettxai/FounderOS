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

// RhythmHandler manages the per-workspace operating rhythm: daily/weekly/monthly
// entries (focuses, blockers, priorities, notes) so a team's cadence is captured
// and decodable by humans and AI agents. Workspace-scoped via X-Workspace-ID.
type RhythmHandler struct {
	pool *pgxpool.Pool
}

func NewRhythmHandler(pool *pgxpool.Pool) *RhythmHandler {
	return &RhythmHandler{pool: pool}
}

type rhythmEntry struct {
	ID        string    `json:"id"`
	Period    string    `json:"period"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	Owner     string    `json:"owner"`
	EntryDate *string   `json:"entry_date"`
	Position  int       `json:"position"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *RhythmHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListEntries returns the workspace's rhythm entries, optionally filtered by ?period=.
// GET /api/v1/rhythm
func (h *RhythmHandler) ListEntries(c *gin.Context) {
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

	period := strings.TrimSpace(c.Query("period"))
	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	var err error
	if period != "" {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, period, kind, content, owner, entry_date, position, created_by, created_at, updated_at
			FROM   rhythm_entries
			WHERE  workspace_id = $1 AND period = $2
			ORDER  BY position ASC, created_at ASC
		`, wsID, period)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, period, kind, content, owner, entry_date, position, created_by, created_at, updated_at
			FROM   rhythm_entries
			WHERE  workspace_id = $1
			ORDER  BY period ASC, position ASC, created_at ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list rhythm entries", err)
		return
	}
	defer rows.Close()

	entries := make([]rhythmEntry, 0)
	for rows.Next() {
		var e rhythmEntry
		var id uuid.UUID
		var entryDate *time.Time
		if err := rows.Scan(&id, &e.Period, &e.Kind, &e.Content, &e.Owner, &entryDate, &e.Position, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan rhythm entry", err)
			return
		}
		e.ID = id.String()
		if entryDate != nil {
			d := entryDate.Format("2006-01-02")
			e.EntryDate = &d
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate rhythm entries", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "count": len(entries)})
}

type rhythmInput struct {
	Period    string  `json:"period"`
	Kind      string  `json:"kind"`
	Content   string  `json:"content"`
	Owner     string  `json:"owner"`
	EntryDate *string `json:"entry_date"`
	Position  int     `json:"position"`
}

func normalizePeriod(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "weekly":
		return "weekly"
	case "monthly":
		return "monthly"
	default:
		return "daily"
	}
}

func normalizeKind(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "blocker":
		return "blocker"
	case "priority":
		return "priority"
	case "note":
		return "note"
	default:
		return "focus"
	}
}

// CreateEntry adds an entry. POST /api/v1/rhythm
func (h *RhythmHandler) CreateEntry(c *gin.Context) {
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
	var in rhythmInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Content) == "" {
		RespondBadRequestErr(c, "content is required")
		return
	}

	var e rhythmEntry
	var id uuid.UUID
	var entryDate *time.Time
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO rhythm_entries (workspace_id, period, kind, content, owner, entry_date, position, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, period, kind, content, owner, entry_date, position, created_by, created_at, updated_at
	`, wsID, normalizePeriod(in.Period), normalizeKind(in.Kind), strings.TrimSpace(in.Content), strings.TrimSpace(in.Owner), in.EntryDate, in.Position, user.ID).
		Scan(&id, &e.Period, &e.Kind, &e.Content, &e.Owner, &entryDate, &e.Position, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create rhythm entry", err)
		return
	}
	e.ID = id.String()
	if entryDate != nil {
		d := entryDate.Format("2006-01-02")
		e.EntryDate = &d
	}
	c.JSON(http.StatusCreated, e)
}

// UpdateEntry edits an entry. PUT /api/v1/rhythm/:id
func (h *RhythmHandler) UpdateEntry(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid entry id")
		return
	}
	var in rhythmInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var e rhythmEntry
	var rid uuid.UUID
	var entryDate *time.Time
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE rhythm_entries
		SET    period     = $3,
		       kind       = $4,
		       content    = $5,
		       owner      = $6,
		       entry_date = $7,
		       position   = $8,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, period, kind, content, owner, entry_date, position, created_by, created_at, updated_at
	`, id, wsID, normalizePeriod(in.Period), normalizeKind(in.Kind), strings.TrimSpace(in.Content), strings.TrimSpace(in.Owner), in.EntryDate, in.Position).
		Scan(&rid, &e.Period, &e.Kind, &e.Content, &e.Owner, &entryDate, &e.Position, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "entry")
		return
	}
	e.ID = rid.String()
	if entryDate != nil {
		d := entryDate.Format("2006-01-02")
		e.EntryDate = &d
	}
	c.JSON(http.StatusOK, e)
}

// DeleteEntry removes an entry. DELETE /api/v1/rhythm/:id
func (h *RhythmHandler) DeleteEntry(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid entry id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM rhythm_entries WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete rhythm entry", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "entry")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "entry deleted"})
}
