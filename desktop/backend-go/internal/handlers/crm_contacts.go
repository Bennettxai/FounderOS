package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// CONTACT-COMPANY RELATIONS HANDLERS
// ============================================================================

// ListCompanyContacts returns contacts associated with a company
func (h *CRMHandler) ListCompanyContacts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "company_id")
		return
	}

	queries := sqlc.New(h.pool)
	contacts, err := queries.ListCompanyContacts(c.Request.Context(), pgtype.UUID{Bytes: companyID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list contacts", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contacts": contacts,
		"count":    len(contacts),
	})
}

// LinkContactToCompanyRequest represents the request to link a contact to a company
type LinkContactToCompanyRequest struct {
	ContactID  string  `json:"contact_id" binding:"required"`
	JobTitle   *string `json:"job_title"`
	Department *string `json:"department"`
	RoleType   *string `json:"role_type"`
	IsPrimary  bool    `json:"is_primary"`
}

// LinkContactToCompany links a contact to a company
func (h *CRMHandler) LinkContactToCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "company_id")
		return
	}

	var req LinkContactToCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	contactID, err := uuid.Parse(req.ContactID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "contact_id")
		return
	}

	queries := sqlc.New(h.pool)
	relation, err := queries.CreateContactCompanyRelation(c.Request.Context(), sqlc.CreateContactCompanyRelationParams{
		ContactID:  pgtype.UUID{Bytes: contactID, Valid: true},
		CompanyID:  pgtype.UUID{Bytes: companyID, Valid: true},
		JobTitle:   req.JobTitle,
		Department: req.Department,
		RoleType:   req.RoleType,
		IsPrimary:  &req.IsPrimary,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "link contact to company", nil)
		return
	}

	c.JSON(http.StatusCreated, relation)
}

// UnlinkContactFromCompany removes a contact-company relation
func (h *CRMHandler) UnlinkContactFromCompany(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	relationID, err := uuid.Parse(c.Param("relationId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "relation_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteContactCompanyRelation(c.Request.Context(), pgtype.UUID{Bytes: relationID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unlink contact", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact unlinked from company"})
}
