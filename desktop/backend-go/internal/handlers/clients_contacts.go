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

// ListClientContacts returns all contacts for a client
func (h *ClientHandler) ListClientContacts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify client access (owner or active workspace member)
	_, err = h.getClientShared(c, pgtype.UUID{Bytes: id, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	contacts, err := queries.ListClientContacts(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list contacts", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContacts(contacts))
}

// CreateClientContact creates a new contact for a client
func (h *ClientHandler) CreateClientContact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Email     *string `json:"email"`
		Phone     *string `json:"phone"`
		Role      *string `json:"role"`
		IsPrimary *bool   `json:"is_primary"`
		Notes     *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify client access (owner or active workspace member)
	_, err = h.getClientShared(c, pgtype.UUID{Bytes: id, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	contact, err := queries.CreateClientContact(c.Request.Context(), sqlc.CreateClientContactParams{
		ClientID:  pgtype.UUID{Bytes: id, Valid: true},
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
		Notes:     req.Notes,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create contact", nil)
		return
	}

	c.JSON(http.StatusCreated, TransformContact(contact))
}

// UpdateClientContact updates a client contact
func (h *ClientHandler) UpdateClientContact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	contactID, err := uuid.Parse(c.Param("contactId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "contact_id")
		return
	}

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Email     *string `json:"email"`
		Phone     *string `json:"phone"`
		Role      *string `json:"role"`
		IsPrimary *bool   `json:"is_primary"`
		Notes     *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify client access (owner or active workspace member)
	_, err = h.getClientShared(c, pgtype.UUID{Bytes: clientID, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	contact, err := queries.UpdateClientContact(c.Request.Context(), sqlc.UpdateClientContactParams{
		ID:        pgtype.UUID{Bytes: contactID, Valid: true},
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
		Notes:     req.Notes,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update contact", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContact(contact))
}

// DeleteClientContact deletes a client contact
func (h *ClientHandler) DeleteClientContact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	contactID, err := uuid.Parse(c.Param("contactId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "contact_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify client access (owner or active workspace member)
	_, err = h.getClientShared(c, pgtype.UUID{Bytes: clientID, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	err = queries.DeleteClientContact(c.Request.Context(), sqlc.DeleteClientContactParams{
		ID:       pgtype.UUID{Bytes: contactID, Valid: true},
		ClientID: pgtype.UUID{Bytes: clientID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete contact", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}
