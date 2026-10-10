package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ListClientInteractions returns all interactions for a client
func (h *ClientHandler) ListClientInteractions(c *gin.Context) {
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

	interactions, err := queries.ListClientInteractions(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list interactions", nil)
		return
	}

	c.JSON(http.StatusOK, TransformInteractions(interactions))
}

// CreateClientInteraction creates a new interaction for a client
func (h *ClientHandler) CreateClientInteraction(c *gin.Context) {
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
		ContactID   *string `json:"contact_id"`
		Type        string  `json:"type" binding:"required"`
		Subject     string  `json:"subject" binding:"required"`
		Description *string `json:"description"`
		Outcome     *string `json:"outcome"`
		OccurredAt  *string `json:"occurred_at"`
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

	// Parse contact ID if provided
	var contactID pgtype.UUID
	if req.ContactID != nil {
		if parsed, err := uuid.Parse(*req.ContactID); err == nil {
			contactID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Parse occurred_at or use now
	occurredAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	if req.OccurredAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.OccurredAt); err == nil {
			occurredAt = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	interaction, err := queries.CreateClientInteraction(c.Request.Context(), sqlc.CreateClientInteractionParams{
		ClientID:    pgtype.UUID{Bytes: id, Valid: true},
		ContactID:   contactID,
		Type:        stringToInteractionType(req.Type),
		Subject:     req.Subject,
		Description: req.Description,
		Outcome:     req.Outcome,
		OccurredAt:  occurredAt,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create interaction", nil)
		return
	}

	c.JSON(http.StatusCreated, TransformInteraction(interaction))
}
