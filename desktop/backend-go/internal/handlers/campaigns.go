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

// CampaignsHandler manages the per-workspace marketing/outreach campaign
// registry (email, ads, sms, organic) with lifecycle status. Workspace-scoped
// via X-Workspace-ID.
type CampaignsHandler struct {
	pool *pgxpool.Pool
}

func NewCampaignsHandler(pool *pgxpool.Pool) *CampaignsHandler {
	return &CampaignsHandler{pool: pool}
}

type campaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Channel     string     `json:"channel"`
	Status      string     `json:"status"`
	Hook        string     `json:"hook"`
	Description string     `json:"description"`
	CTA         string     `json:"cta"`
	StartDate   *time.Time `json:"start_date"`
	CreatedBy   *string    `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *CampaignsHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListCampaigns returns the workspace's campaigns, optionally filtered by ?q=.
// GET /api/v1/campaigns
func (h *CampaignsHandler) ListCampaigns(c *gin.Context) {
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
			SELECT id, name, channel, status, hook, description, cta, start_date, created_by, created_at, updated_at
			FROM   campaigns
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%' OR hook ILIKE '%'||$2||'%')
			ORDER  BY name ASC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, channel, status, hook, description, cta, start_date, created_by, created_at, updated_at
			FROM   campaigns
			WHERE  workspace_id = $1
			ORDER  BY name ASC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list campaigns", err)
		return
	}
	defer rows.Close()

	campaigns := make([]campaign, 0)
	for rows.Next() {
		var cp campaign
		var id uuid.UUID
		if err := rows.Scan(&id, &cp.Name, &cp.Channel, &cp.Status, &cp.Hook, &cp.Description, &cp.CTA, &cp.StartDate, &cp.CreatedBy, &cp.CreatedAt, &cp.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan campaign", err)
			return
		}
		cp.ID = id.String()
		campaigns = append(campaigns, cp)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate campaigns", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": campaigns, "count": len(campaigns)})
}

type campaignInput struct {
	Name        string     `json:"name"`
	Channel     string     `json:"channel"`
	Status      string     `json:"status"`
	Hook        string     `json:"hook"`
	Description string     `json:"description"`
	CTA         string     `json:"cta"`
	StartDate   *time.Time `json:"start_date"`
}

// CreateCampaign adds a campaign. POST /api/v1/campaigns
func (h *CampaignsHandler) CreateCampaign(c *gin.Context) {
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
	var in campaignInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}
	channel := strings.TrimSpace(in.Channel)
	if channel == "" {
		channel = "email"
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "draft"
	}

	var cp campaign
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO campaigns (workspace_id, name, channel, status, hook, description, cta, start_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, channel, status, hook, description, cta, start_date, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), channel, status, in.Hook, in.Description, in.CTA, in.StartDate, user.ID).
		Scan(&id, &cp.Name, &cp.Channel, &cp.Status, &cp.Hook, &cp.Description, &cp.CTA, &cp.StartDate, &cp.CreatedBy, &cp.CreatedAt, &cp.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create campaign", err)
		return
	}
	cp.ID = id.String()
	c.JSON(http.StatusCreated, cp)
}

// UpdateCampaign edits a campaign. PUT /api/v1/campaigns/:id
func (h *CampaignsHandler) UpdateCampaign(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid campaign id")
		return
	}
	var in campaignInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var cp campaign
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE campaigns
		SET    name        = COALESCE(NULLIF($3,''), name),
		       channel     = COALESCE(NULLIF($4,''), channel),
		       status      = COALESCE(NULLIF($5,''), status),
		       hook        = $6,
		       description = $7,
		       cta         = $8,
		       start_date  = $9,
		       updated_at  = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, channel, status, hook, description, cta, start_date, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Channel), strings.TrimSpace(in.Status), in.Hook, in.Description, in.CTA, in.StartDate).
		Scan(&rid, &cp.Name, &cp.Channel, &cp.Status, &cp.Hook, &cp.Description, &cp.CTA, &cp.StartDate, &cp.CreatedBy, &cp.CreatedAt, &cp.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "campaign")
		return
	}
	cp.ID = rid.String()
	c.JSON(http.StatusOK, cp)
}

// DeleteCampaign removes a campaign. DELETE /api/v1/campaigns/:id
func (h *CampaignsHandler) DeleteCampaign(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid campaign id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM campaigns WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete campaign", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "campaign")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "campaign deleted"})
}
