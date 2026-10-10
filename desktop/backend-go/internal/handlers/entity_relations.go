package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// EntityRelationsHandler is the generic relations layer. It lets any module
// relate its core primitive (a client, project, event, deal, ...) to any other
// module's primitive by ID, without either module knowing the other's internals.
// Each module still owns its own table; this only stores pointers (entity_links).
//
// A link is workspace-scoped when the request carries an X-Workspace-ID header
// for a workspace the user actively belongs to (visible to all members), and
// personal otherwise (workspace_id NULL, visible only to its creator).
type EntityRelationsHandler struct {
	pool *pgxpool.Pool
}

// NewEntityRelationsHandler returns a handler backed by the given pool.
func NewEntityRelationsHandler(pool *pgxpool.Pool) *EntityRelationsHandler {
	return &EntityRelationsHandler{pool: pool}
}

// validLinkTypes mirrors the entity_link_type enum so we reject bad values in Go
// rather than letting Postgres throw on the cast.
var validLinkTypes = map[string]bool{
	"related": true, "mentions": true,
	"parent_of": true, "child_of": true,
	"blocks": true, "blocked_by": true, "depends_on": true,
	"duplicate_of": true, "original_of": true,
	"derived_from": true, "spawned": true,
	"task_for": true, "project_for": true, "note_about": true, "meeting_about": true,
	"custom": true,
}

// entityLinkRow is the canonical JSON shape returned to callers.
type entityLinkRow struct {
	ID              string          `json:"id"`
	WorkspaceID     *string         `json:"workspace_id"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
	SourceName      *string         `json:"source_name"`
	TargetType      string          `json:"target_type"`
	TargetID        string          `json:"target_id"`
	TargetName      *string         `json:"target_name"`
	LinkType        string          `json:"link_type"`
	CustomLinkType  *string         `json:"custom_link_type"`
	IsBidirectional bool            `json:"is_bidirectional"`
	Description     *string         `json:"description"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedBy       *string         `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// workspaceScope resolves the X-Workspace-ID header and confirms active
// membership. Returns (workspaceID, true) for a workspace link, or
// (uuid.Nil, false) for a personal link (no header, bad header, or non-member).
func (h *EntityRelationsHandler) workspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListLinks returns every link touching an entity (as source OR target), within
// the caller's current scope (the active workspace, else their personal links).
// GET /api/relations?type=<entityType>&id=<entityUUID>
func (h *EntityRelationsHandler) ListLinks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}

	entityType := c.Query("type")
	entityIDStr := c.Query("id")
	if entityType == "" || entityIDStr == "" {
		RespondBadRequestErr(c, "type and id query params are required")
		return
	}
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		RespondBadRequestErr(c, "invalid entity id")
		return
	}

	ctx := c.Request.Context()
	wsID, scoped := h.workspaceScope(c, user.ID)

	// Workspace scope: all members' links in this workspace touching the entity.
	// Personal scope: only the caller's own links with no workspace.
	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	if scoped {
		rows, err = h.pool.Query(ctx, `
			SELECT id, workspace_id, source_type, source_id, source_name,
			       target_type, target_id, target_name, link_type, custom_link_type,
			       is_bidirectional, description, metadata, created_by, created_at, updated_at
			FROM   entity_links
			WHERE  workspace_id = $1
			AND   ((source_type = $2 AND source_id = $3)
			    OR (target_type = $2 AND target_id = $3))
			ORDER  BY created_at DESC
		`, wsID, entityType, entityID)
	} else {
		rows, err = h.pool.Query(ctx, `
			SELECT id, workspace_id, source_type, source_id, source_name,
			       target_type, target_id, target_name, link_type, custom_link_type,
			       is_bidirectional, description, metadata, created_by, created_at, updated_at
			FROM   entity_links
			WHERE  workspace_id IS NULL AND user_id = $1
			AND   ((source_type = $2 AND source_id = $3)
			    OR (target_type = $2 AND target_id = $3))
			ORDER  BY created_at DESC
		`, user.ID, entityType, entityID)
	}
	if err != nil {
		RespondInternalErr(c, "list entity links", err)
		return
	}
	defer rows.Close()

	links := make([]entityLinkRow, 0)
	for rows.Next() {
		r, err := scanEntityLink(rows)
		if err != nil {
			RespondInternalErr(c, "scan entity link", err)
			return
		}
		links = append(links, r)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate entity links", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"links": links, "count": len(links)})
}

// CreateLink relates two entities. The link is workspace-scoped when an
// X-Workspace-ID header for a workspace the caller belongs to is present,
// personal otherwise.
// POST /api/relations
//
//	{ "source_type":"client", "source_id":"<uuid>",
//	  "target_type":"project", "target_id":"<uuid>",
//	  "link_type":"related", "source_name":"...", "target_name":"...",
//	  "description":"...", "is_bidirectional":true }
func (h *EntityRelationsHandler) CreateLink(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}

	var body struct {
		SourceType      string  `json:"source_type"`
		SourceID        string  `json:"source_id"`
		SourceName      *string `json:"source_name"`
		TargetType      string  `json:"target_type"`
		TargetID        string  `json:"target_id"`
		TargetName      *string `json:"target_name"`
		LinkType        string  `json:"link_type"`
		CustomLinkType  *string `json:"custom_link_type"`
		Description     *string `json:"description"`
		IsBidirectional bool    `json:"is_bidirectional"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	if body.SourceType == "" || body.TargetType == "" {
		RespondBadRequestErr(c, "source_type and target_type are required")
		return
	}
	sourceID, err := uuid.Parse(body.SourceID)
	if err != nil {
		RespondBadRequestErr(c, "invalid source_id")
		return
	}
	targetID, err := uuid.Parse(body.TargetID)
	if err != nil {
		RespondBadRequestErr(c, "invalid target_id")
		return
	}
	if body.LinkType == "" {
		body.LinkType = "related"
	}
	if !validLinkTypes[body.LinkType] {
		RespondBadRequestErr(c, "invalid link_type")
		return
	}

	wsID, scoped := h.workspaceScope(c, user.ID)
	var wsArg interface{}
	if scoped {
		wsArg = wsID
	} else {
		wsArg = nil
	}

	ctx := c.Request.Context()
	// Idempotent: a repeated link (same source/target/type) returns the existing row.
	row := h.pool.QueryRow(ctx, `
		INSERT INTO entity_links
			(user_id, workspace_id, source_type, source_id, source_name,
			 target_type, target_id, target_name, link_type, custom_link_type,
			 is_bidirectional, description, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$1)
		ON CONFLICT (source_type, source_id, target_type, target_id, link_type)
		DO UPDATE SET
			source_name      = COALESCE(EXCLUDED.source_name, entity_links.source_name),
			target_name      = COALESCE(EXCLUDED.target_name, entity_links.target_name),
			description      = COALESCE(EXCLUDED.description, entity_links.description),
			is_bidirectional = EXCLUDED.is_bidirectional,
			updated_at       = NOW()
		RETURNING id, workspace_id, source_type, source_id, source_name,
		          target_type, target_id, target_name, link_type, custom_link_type,
		          is_bidirectional, description, metadata, created_by, created_at, updated_at
	`, user.ID, wsArg, body.SourceType, sourceID, body.SourceName,
		body.TargetType, targetID, body.TargetName, body.LinkType, body.CustomLinkType,
		body.IsBidirectional, body.Description)

	r, err := scanEntityLink(row)
	if err != nil {
		RespondInternalErr(c, "create entity link", err)
		return
	}
	c.JSON(http.StatusCreated, r)
}

// DeleteLink removes a relation. The caller must own it (personal link) or be an
// active member of its workspace.
// DELETE /api/relations/:id
func (h *EntityRelationsHandler) DeleteLink(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	linkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid link id")
		return
	}

	ctx := c.Request.Context()
	// Load the link's owning workspace + creator so we can authorize.
	var (
		linkWS  *uuid.UUID
		linkUID string
	)
	err = h.pool.QueryRow(ctx,
		`SELECT workspace_id, user_id FROM entity_links WHERE id = $1`, linkID,
	).Scan(&linkWS, &linkUID)
	if err != nil {
		RespondNotFoundErr(c, "link")
		return
	}

	authorized := false
	if linkWS == nil {
		authorized = linkUID == user.ID // personal link: only the owner
	} else {
		var member bool
		_ = h.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
			*linkWS, user.ID).Scan(&member)
		authorized = member
	}
	if !authorized {
		RespondNotFoundErr(c, "link")
		return
	}

	if _, err := h.pool.Exec(ctx, `DELETE FROM entity_links WHERE id = $1`, linkID); err != nil {
		RespondInternalErr(c, "delete entity link", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "link deleted"})
}

// scanRow is the minimal surface shared by pgx.Row and pgx.Rows.
type scanRow interface{ Scan(...any) error }

// scanEntityLink maps a row into entityLinkRow, normalizing nullable columns.
func scanEntityLink(row scanRow) (entityLinkRow, error) {
	var (
		r       entityLinkRow
		wsID    *uuid.UUID
		rawMeta []byte
	)
	err := row.Scan(
		&r.ID, &wsID, &r.SourceType, &r.SourceID, &r.SourceName,
		&r.TargetType, &r.TargetID, &r.TargetName, &r.LinkType, &r.CustomLinkType,
		&r.IsBidirectional, &r.Description, &rawMeta, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return r, err
	}
	if wsID != nil {
		s := wsID.String()
		r.WorkspaceID = &s
	}
	if len(rawMeta) == 0 {
		rawMeta = []byte("{}")
	}
	r.Metadata = json.RawMessage(rawMeta)
	return r, nil
}
