package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// CRMHandler handles CRM operations (companies, pipelines, deals, activities)
type CRMHandler struct {
	EngineSyncHook
	pool *pgxpool.Pool
}

// NewCRMHandler creates a new CRMHandler
func NewCRMHandler(pool *pgxpool.Pool) *CRMHandler {
	return &CRMHandler{pool: pool}
}

// ============================================================================
// COMPANIES HANDLERS
// ============================================================================

// ListCompanies returns all companies for the current user
func (h *CRMHandler) ListCompanies(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse query params
	industry := c.Query("industry")
	lifecycleStage := c.Query("lifecycle_stage")

	pg := ParsePagination(c)

	companies, err := queries.ListCompanies(c.Request.Context(), sqlc.ListCompaniesParams{
		UserID:         user.ID,
		Industry:       crmToNullString(industry),
		LifecycleStage: crmToNullString(lifecycleStage),
		LimitVal:       pg.Limit,
		OffsetVal:      pg.Offset,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list companies", nil)
		return
	}

	total := int64(pg.Offset) + int64(len(companies))
	if int64(len(companies)) == int64(pg.Limit) {
		total++
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(transformCompanies(companies), total, pg))
}

// GetCompany returns a single company by ID
func (h *CRMHandler) GetCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "company_id")
		return
	}

	queries := sqlc.New(h.pool)
	company, err := queries.GetCompany(c.Request.Context(), sqlc.GetCompanyParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Company")
		return
	}

	c.JSON(http.StatusOK, transformCompany(company))
}

// CreateCompanyRequest represents the request to create a company
type CreateCompanyRequest struct {
	Name           string                 `json:"name" binding:"required"`
	LegalName      *string                `json:"legal_name"`
	Industry       *string                `json:"industry"`
	CompanySize    *string                `json:"company_size"`
	Website        *string                `json:"website"`
	Email          *string                `json:"email"`
	Phone          *string                `json:"phone"`
	AddressLine1   *string                `json:"address_line1"`
	AddressLine2   *string                `json:"address_line2"`
	City           *string                `json:"city"`
	State          *string                `json:"state"`
	PostalCode     *string                `json:"postal_code"`
	Country        *string                `json:"country"`
	AnnualRevenue  *float64               `json:"annual_revenue"`
	Currency       *string                `json:"currency"`
	TaxID          *string                `json:"tax_id"`
	LinkedinURL    *string                `json:"linkedin_url"`
	TwitterHandle  *string                `json:"twitter_handle"`
	OwnerID        *string                `json:"owner_id"`
	LifecycleStage *string                `json:"lifecycle_stage"`
	LeadSource     *string                `json:"lead_source"`
	LogoURL        *string                `json:"logo_url"`
	CustomFields   map[string]interface{} `json:"custom_fields"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CreateCompany creates a new company
func (h *CRMHandler) CreateCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Convert custom fields and metadata to JSON
	customFields := crmJSONBytes(req.CustomFields)
	metadata := crmJSONBytes(req.Metadata)

	company, err := queries.CreateCompany(c.Request.Context(), sqlc.CreateCompanyParams{
		UserID:         user.ID,
		Name:           req.Name,
		LegalName:      req.LegalName,
		Industry:       req.Industry,
		CompanySize:    req.CompanySize,
		Website:        req.Website,
		Email:          req.Email,
		Phone:          req.Phone,
		AddressLine1:   req.AddressLine1,
		AddressLine2:   req.AddressLine2,
		City:           req.City,
		State:          req.State,
		PostalCode:     req.PostalCode,
		Country:        req.Country,
		AnnualRevenue:  crmToNumeric(req.AnnualRevenue),
		Currency:       req.Currency,
		TaxID:          req.TaxID,
		LinkedinUrl:    req.LinkedinURL,
		TwitterHandle:  req.TwitterHandle,
		OwnerID:        req.OwnerID,
		LifecycleStage: req.LifecycleStage,
		LeadSource:     req.LeadSource,
		LogoUrl:        req.LogoURL,
		CustomFields:   customFields,
		Metadata:       metadata,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create company", err)
		return
	}

	// Mirror into the OptimalEngine knowledge graph so CRM records show
	// up in the Pages graph view alongside everything else the user owns.
	companyBody := req.Name
	if req.Industry != nil {
		companyBody += " — " + *req.Industry
	}
	if req.LegalName != nil {
		companyBody += " (" + *req.LegalName + ")"
	}
	companyMeta := map[string]string{"crm_kind": "company"}
	if req.LifecycleStage != nil {
		companyMeta["lifecycle_stage"] = *req.LifecycleStage
	}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleCRM,
		ID:         "company-" + uuidString(company.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Name,
		Body:       companyBody,
		Genre:      "company",
		ModifiedAt: pgTimestampToTime(company.UpdatedAt),
		Metadata:   companyMeta,
	})

	c.JSON(http.StatusCreated, transformCompany(company))
}

// UpdateCompanyRequest represents the request to update a company
type UpdateCompanyRequest struct {
	Name           string                 `json:"name" binding:"required"`
	LegalName      *string                `json:"legal_name"`
	Industry       *string                `json:"industry"`
	CompanySize    *string                `json:"company_size"`
	Website        *string                `json:"website"`
	Email          *string                `json:"email"`
	Phone          *string                `json:"phone"`
	AddressLine1   *string                `json:"address_line1"`
	AddressLine2   *string                `json:"address_line2"`
	City           *string                `json:"city"`
	State          *string                `json:"state"`
	PostalCode     *string                `json:"postal_code"`
	Country        *string                `json:"country"`
	AnnualRevenue  *float64               `json:"annual_revenue"`
	LifecycleStage *string                `json:"lifecycle_stage"`
	LinkedinURL    *string                `json:"linkedin_url"`
	TwitterHandle  *string                `json:"twitter_handle"`
	LogoURL        *string                `json:"logo_url"`
	CustomFields   map[string]interface{} `json:"custom_fields"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// UpdateCompany updates an existing company
func (h *CRMHandler) UpdateCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "company_id")
		return
	}

	var req UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Convert custom fields and metadata to JSON
	customFields := crmJSONBytes(req.CustomFields)
	metadata := crmJSONBytes(req.Metadata)

	company, err := queries.UpdateCompany(c.Request.Context(), sqlc.UpdateCompanyParams{
		ID:             pgtype.UUID{Bytes: id, Valid: true},
		UserID:         user.ID,
		Name:           req.Name,
		LegalName:      req.LegalName,
		Industry:       req.Industry,
		CompanySize:    req.CompanySize,
		Website:        req.Website,
		Email:          req.Email,
		Phone:          req.Phone,
		AddressLine1:   req.AddressLine1,
		AddressLine2:   req.AddressLine2,
		City:           req.City,
		State:          req.State,
		PostalCode:     req.PostalCode,
		Country:        req.Country,
		AnnualRevenue:  crmToNumeric(req.AnnualRevenue),
		LifecycleStage: req.LifecycleStage,
		LinkedinUrl:    req.LinkedinURL,
		TwitterHandle:  req.TwitterHandle,
		LogoUrl:        req.LogoURL,
		CustomFields:   customFields,
		Metadata:       metadata,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update company", nil)
		return
	}

	c.JSON(http.StatusOK, transformCompany(company))
}

// DeleteCompany deletes a company
func (h *CRMHandler) DeleteCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "company_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteCompany(c.Request.Context(), sqlc.DeleteCompanyParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete company", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Company deleted"})
}

// SearchCompanies searches companies by name or website
func (h *CRMHandler) SearchCompanies(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	queries := sqlc.New(h.pool)
	companies, err := queries.SearchCompanies(c.Request.Context(), sqlc.SearchCompaniesParams{
		UserID:   user.ID,
		Column2:  &query,
		LimitVal: int32(limit),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "search companies", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": transformCompanies(companies),
		"count":     len(companies),
	})
}
