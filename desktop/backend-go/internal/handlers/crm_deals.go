package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// CRM DEALS HANDLERS
// ============================================================================

// ListCRMDeals returns all CRM deals for the current user
func (h *CRMHandler) ListCRMDeals(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse query params
	pipelineID := c.Query("pipeline_id")
	stageID := c.Query("stage_id")
	status := c.Query("status")
	ownerID := c.Query("owner_id")

	pg := ParsePagination(c)

	deals, err := queries.ListCRMDeals(c.Request.Context(), sqlc.ListCRMDealsParams{
		UserID:     user.ID,
		PipelineID: crmToNullUUID(pipelineID),
		StageID:    crmToNullUUID(stageID),
		Status:     crmToNullString(status),
		OwnerID:    crmToNullString(ownerID),
		LimitVal:   pg.Limit,
		OffsetVal:  pg.Offset,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list deals", err)
		return
	}

	// The SQL query does not expose total count; use len(deals) as an approximation
	// so that callers know whether more pages may exist.
	total := int64(pg.Offset) + int64(len(deals))
	if int64(len(deals)) == int64(pg.Limit) {
		// Might have more — signal HasMore=true by adding 1 to total estimate
		total++
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(transformCRMDeals(deals), total, pg))
}

// GetCRMDeal returns a single CRM deal by ID
func (h *CRMHandler) GetCRMDeal(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	queries := sqlc.New(h.pool)
	deal, err := queries.GetCRMDeal(c.Request.Context(), sqlc.GetCRMDealParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Deal")
		return
	}

	c.JSON(http.StatusOK, transformCRMDealRow(deal))
}

// CreateCRMDealRequest represents the request to create a CRM deal
type CreateCRMDealRequest struct {
	PipelineID        string                 `json:"pipeline_id" binding:"required"`
	StageID           string                 `json:"stage_id" binding:"required"`
	Name              string                 `json:"name" binding:"required"`
	Description       *string                `json:"description"`
	Amount            *float64               `json:"amount"`
	Currency          *string                `json:"currency"`
	Probability       *int32                 `json:"probability"`
	ExpectedCloseDate *string                `json:"expected_close_date"`
	OwnerID           *string                `json:"owner_id"`
	CompanyID         *string                `json:"company_id"`
	PrimaryContactID  *string                `json:"primary_contact_id"`
	Status            *string                `json:"status"`
	Priority          *string                `json:"priority"`
	LeadSource        *string                `json:"lead_source"`
	CustomFields      map[string]interface{} `json:"custom_fields"`
}

// CreateCRMDeal creates a new CRM deal
func (h *CRMHandler) CreateCRMDeal(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateCRMDealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	pipelineID, err := uuid.Parse(req.PipelineID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "pipeline_id")
		return
	}

	stageID, err := uuid.Parse(req.StageID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "stage_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Convert custom fields to JSON
	customFields := crmJSONBytes(req.CustomFields)

	// Parse expected close date
	var expectedCloseDate pgtype.Date
	if req.ExpectedCloseDate != nil {
		if t, err := time.Parse("2006-01-02", *req.ExpectedCloseDate); err == nil {
			expectedCloseDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	deal, err := queries.CreateCRMDeal(c.Request.Context(), sqlc.CreateCRMDealParams{
		UserID:            user.ID,
		PipelineID:        pgtype.UUID{Bytes: pipelineID, Valid: true},
		StageID:           pgtype.UUID{Bytes: stageID, Valid: true},
		Name:              req.Name,
		Description:       req.Description,
		Amount:            crmToNumeric(req.Amount),
		Currency:          req.Currency,
		Probability:       req.Probability,
		ExpectedCloseDate: expectedCloseDate,
		OwnerID:           req.OwnerID,
		CompanyID:         crmToNullUUID(crmPtrToString(req.CompanyID)),
		PrimaryContactID:  crmToNullUUID(crmPtrToString(req.PrimaryContactID)),
		Status:            req.Status,
		Priority:          req.Priority,
		LeadSource:        req.LeadSource,
		CustomFields:      customFields,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create deal", err)
		return
	}

	dealMeta := map[string]string{"crm_kind": "deal"}
	if req.Status != nil {
		dealMeta["status"] = *req.Status
	}
	if req.Priority != nil {
		dealMeta["priority"] = *req.Priority
	}
	dealBody := req.Name
	if req.Description != nil {
		dealBody += "\n\n" + *req.Description
	}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleCRM,
		ID:         "deal-" + uuidString(deal.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Name,
		Body:       dealBody,
		Genre:      "deal",
		ModifiedAt: pgTimestampToTime(deal.UpdatedAt),
		Metadata:   dealMeta,
	})

	c.JSON(http.StatusCreated, transformCRMDealBasic(deal))
}

// UpdateCRMDealRequest represents the request to update a CRM deal
type UpdateCRMDealRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Description       *string                `json:"description"`
	Amount            *float64               `json:"amount"`
	Probability       *int32                 `json:"probability"`
	ExpectedCloseDate *string                `json:"expected_close_date"`
	OwnerID           *string                `json:"owner_id"`
	CompanyID         *string                `json:"company_id"`
	PrimaryContactID  *string                `json:"primary_contact_id"`
	Priority          *string                `json:"priority"`
	CustomFields      map[string]interface{} `json:"custom_fields"`
}

// UpdateCRMDeal updates an existing CRM deal
func (h *CRMHandler) UpdateCRMDeal(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	var req UpdateCRMDealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Convert custom fields to JSON
	customFields := crmJSONBytes(req.CustomFields)

	// Parse expected close date
	var expectedCloseDate pgtype.Date
	if req.ExpectedCloseDate != nil {
		if t, err := time.Parse("2006-01-02", *req.ExpectedCloseDate); err == nil {
			expectedCloseDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	deal, err := queries.UpdateCRMDeal(c.Request.Context(), sqlc.UpdateCRMDealParams{
		ID:                pgtype.UUID{Bytes: id, Valid: true},
		UserID:            user.ID,
		Name:              req.Name,
		Description:       req.Description,
		Amount:            crmToNumeric(req.Amount),
		Probability:       req.Probability,
		ExpectedCloseDate: expectedCloseDate,
		OwnerID:           req.OwnerID,
		CompanyID:         crmToNullUUID(crmPtrToString(req.CompanyID)),
		PrimaryContactID:  crmToNullUUID(crmPtrToString(req.PrimaryContactID)),
		Priority:          req.Priority,
		CustomFields:      customFields,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update deal", nil)
		return
	}

	c.JSON(http.StatusOK, transformCRMDealBasic(deal))
}

// MoveCRMDealStage moves a deal to a different stage
func (h *CRMHandler) MoveCRMDealStage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	var req struct {
		StageID string `json:"stage_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	stageID, err := uuid.Parse(req.StageID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "stage_id")
		return
	}

	queries := sqlc.New(h.pool)
	deal, err := queries.UpdateCRMDealStage(c.Request.Context(), sqlc.UpdateCRMDealStageParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		UserID:  user.ID,
		StageID: pgtype.UUID{Bytes: stageID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "move deal", nil)
		return
	}

	c.JSON(http.StatusOK, transformCRMDealBasic(deal))
}

// UpdateCRMDealStatus updates the status of a deal (open, won, lost)
func (h *CRMHandler) UpdateCRMDealStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	var req struct {
		Status     string  `json:"status" binding:"required"`
		LostReason *string `json:"lost_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	deal, err := queries.UpdateCRMDealStatus(c.Request.Context(), sqlc.UpdateCRMDealStatusParams{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		UserID:     user.ID,
		Status:     &req.Status,
		LostReason: req.LostReason,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update deal status", nil)
		return
	}

	c.JSON(http.StatusOK, transformCRMDealBasic(deal))
}

// DeleteCRMDeal deletes a CRM deal
func (h *CRMHandler) DeleteCRMDeal(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteCRMDeal(c.Request.Context(), sqlc.DeleteCRMDealParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete deal", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deal deleted"})
}

// numericToFloat64 converts COALESCE(SUM(...), 0) scan results (interface{}) to
// float64. pgx v5 returns pgtype.Numeric for numeric/decimal columns; fallbacks
// cover int64 (when the value is the COALESCE default 0) and string.
func numericToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case pgtype.Numeric:
		f, err := val.Float64Value()
		if err != nil || !f.Valid {
			return 0
		}
		return f.Float64
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// GetCRMDealStats returns deal statistics for the current user
func (h *CRMHandler) GetCRMDealStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pipelineID := c.Query("pipeline_id")

	queries := sqlc.New(h.pool)
	stats, err := queries.GetCRMDealStats(c.Request.Context(), sqlc.GetCRMDealStatsParams{
		UserID:     user.ID,
		PipelineID: crmToNullUUID(pipelineID),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get deal stats", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_deals": stats.TotalDeals,
		"open_deals":  stats.OpenDeals,
		"won_deals":   stats.WonDeals,
		"lost_deals":  stats.LostDeals,
		"open_value":  numericToFloat64(stats.OpenValue),
		"won_value":   numericToFloat64(stats.WonValue),
		"lost_value":  numericToFloat64(stats.LostValue),
	})
}
