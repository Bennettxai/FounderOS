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

// OffersHandler manages the per-workspace offers catalog: the business's
// productized offers (name, price, status, what's included) so the team and
// AI agents share one source of truth for what we sell. Workspace-scoped via
// X-Workspace-ID.
type OffersHandler struct {
	pool *pgxpool.Pool
}

func NewOffersHandler(pool *pgxpool.Pool) *OffersHandler {
	return &OffersHandler{pool: pool}
}

type offer struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Tier        string    `json:"tier"`
	Price       string    `json:"price"`
	Status      string    `json:"status"`
	Promise     string    `json:"promise"`
	Description string    `json:"description"`
	Includes    string    `json:"includes"`
	CTA         string    `json:"cta"`
	CreatedBy   *string   `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *OffersHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListOffers returns the workspace's offers, optionally filtered by ?q=.
// GET /api/v1/offers
func (h *OffersHandler) ListOffers(c *gin.Context) {
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
			SELECT id, name, tier, price, status, promise, description, includes, cta, created_by, created_at, updated_at
			FROM   offers
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%' OR includes ILIKE '%'||$2||'%' OR promise ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, tier, price, status, promise, description, includes, cta, created_by, created_at, updated_at
			FROM   offers
			WHERE  workspace_id = $1
			ORDER  BY name ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list offers", err)
		return
	}
	defer rows.Close()

	offers := make([]offer, 0)
	for rows.Next() {
		var o offer
		var id uuid.UUID
		if err := rows.Scan(&id, &o.Name, &o.Tier, &o.Price, &o.Status, &o.Promise, &o.Description, &o.Includes, &o.CTA, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan offer", err)
			return
		}
		o.ID = id.String()
		offers = append(offers, o)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate offers", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"offers": offers, "count": len(offers)})
}

type offerInput struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Price       string `json:"price"`
	Status      string `json:"status"`
	Promise     string `json:"promise"`
	Description string `json:"description"`
	Includes    string `json:"includes"`
	CTA         string `json:"cta"`
}

// normalizeStatus keeps status within the allowed set, defaulting to active.
func normalizeStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		return "draft"
	case "archived":
		return "archived"
	default:
		return "active"
	}
}

// normalizeTier keeps the offer-ladder tier within the allowed set.
func normalizeTier(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "phase-1":
		return "phase-1"
	case "phase-2":
		return "phase-2"
	case "lane":
		return "lane"
	default:
		return "audit"
	}
}

// CreateOffer adds an offer. POST /api/v1/offers
func (h *OffersHandler) CreateOffer(c *gin.Context) {
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
	var in offerInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}

	var o offer
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO offers (workspace_id, name, tier, price, status, promise, description, includes, cta, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, tier, price, status, promise, description, includes, cta, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), normalizeTier(in.Tier), in.Price, normalizeStatus(in.Status), in.Promise, in.Description, in.Includes, in.CTA, user.ID).
		Scan(&id, &o.Name, &o.Tier, &o.Price, &o.Status, &o.Promise, &o.Description, &o.Includes, &o.CTA, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create offer", err)
		return
	}
	o.ID = id.String()
	c.JSON(http.StatusCreated, o)
}

// UpdateOffer edits an offer. PUT /api/v1/offers/:id
func (h *OffersHandler) UpdateOffer(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid offer id")
		return
	}
	var in offerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var o offer
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE offers
		SET    name        = COALESCE(NULLIF($3,''), name),
		       tier        = $4,
		       price       = $5,
		       status      = $6,
		       promise     = $7,
		       description = $8,
		       includes    = $9,
		       cta         = $10,
		       updated_at  = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, tier, price, status, promise, description, includes, cta, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), normalizeTier(in.Tier), in.Price, normalizeStatus(in.Status), in.Promise, in.Description, in.Includes, in.CTA).
		Scan(&rid, &o.Name, &o.Tier, &o.Price, &o.Status, &o.Promise, &o.Description, &o.Includes, &o.CTA, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "offer")
		return
	}
	o.ID = rid.String()
	c.JSON(http.StatusOK, o)
}

// DeleteOffer removes an offer. DELETE /api/v1/offers/:id
func (h *OffersHandler) DeleteOffer(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid offer id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM offers WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete offer", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "offer")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "offer deleted"})
}
