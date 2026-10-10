package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

type DeliverablesHandler struct {
	pool *pgxpool.Pool
}

func NewDeliverablesHandler(pool *pgxpool.Pool) *DeliverablesHandler {
	return &DeliverablesHandler{pool: pool}
}

type deliverable struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Client      string    `json:"client"`
	Project     string    `json:"project"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	CreatedBy   *string   `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type deliverableInput struct {
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Client      string `json:"client"`
	Project     string `json:"project"`
	Link        string `json:"link"`
	Description string `json:"description"`
}

func (h *DeliverablesHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	workspaceID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		workspaceID, userID).Scan(&member)
	return workspaceID, err == nil && member
}

func normalizeDeliverableKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "package", "document", "deck", "script", "report", "video":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "other"
	}
}

func normalizeDeliverableStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "in_progress", "delivered":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "draft"
	}
}

func scanDeliverable(row pgx.Row) (deliverable, error) {
	var result deliverable
	var id uuid.UUID
	err := row.Scan(
		&id,
		&result.Title,
		&result.Kind,
		&result.Status,
		&result.Client,
		&result.Project,
		&result.Link,
		&result.Description,
		&result.CreatedBy,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	result.ID = id.String()
	return result, err
}

func (h *DeliverablesHandler) ListDeliverables(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	workspaceID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, title, kind, status, client, project, link, description,
		       created_by, created_at, updated_at
		FROM deliverables
		WHERE workspace_id = $1
		  AND ($2 = '' OR title ILIKE '%'||$2||'%' OR client ILIKE '%'||$2||'%'
		       OR project ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
		ORDER BY updated_at DESC
	`, workspaceID, query)
	if err != nil {
		RespondInternalErr(c, "list deliverables", err)
		return
	}
	defer rows.Close()

	result := make([]deliverable, 0)
	for rows.Next() {
		item, err := scanDeliverable(rows)
		if err != nil {
			RespondInternalErr(c, "scan deliverable", err)
			return
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate deliverables", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliverables": result, "count": len(result)})
}

func (h *DeliverablesHandler) CreateDeliverable(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	workspaceID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	var input deliverableInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" {
		RespondBadRequestErr(c, "title is required")
		return
	}

	item, err := scanDeliverable(h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO deliverables (
			workspace_id, title, kind, status, client, project, link, description, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, title, kind, status, client, project, link, description,
		          created_by, created_at, updated_at
	`, workspaceID, strings.TrimSpace(input.Title), normalizeDeliverableKind(input.Kind),
		normalizeDeliverableStatus(input.Status), strings.TrimSpace(input.Client),
		strings.TrimSpace(input.Project), strings.TrimSpace(input.Link), input.Description, user.ID))
	if err != nil {
		RespondInternalErr(c, "create deliverable", err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *DeliverablesHandler) UpdateDeliverable(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	workspaceID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid deliverable id")
		return
	}
	var input deliverableInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" {
		RespondBadRequestErr(c, "title is required")
		return
	}

	item, err := scanDeliverable(h.pool.QueryRow(c.Request.Context(), `
		UPDATE deliverables
		SET title=$3, kind=$4, status=$5, client=$6, project=$7, link=$8,
		    description=$9, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2
		RETURNING id, title, kind, status, client, project, link, description,
		          created_by, created_at, updated_at
	`, id, workspaceID, strings.TrimSpace(input.Title), normalizeDeliverableKind(input.Kind),
		normalizeDeliverableStatus(input.Status), strings.TrimSpace(input.Client),
		strings.TrimSpace(input.Project), strings.TrimSpace(input.Link), input.Description))
	if err != nil {
		if err == pgx.ErrNoRows {
			RespondNotFoundErr(c, "deliverable")
			return
		}
		RespondInternalErr(c, "update deliverable", err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *DeliverablesHandler) DeleteDeliverable(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	workspaceID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid deliverable id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM deliverables WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		RespondInternalErr(c, "delete deliverable", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "deliverable")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deliverable deleted"})
}
