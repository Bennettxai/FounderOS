package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ListWidgetTypes returns all available widget types
func (h *DashboardCRUDHandler) ListWidgetTypes(c *gin.Context) {
	queries := sqlc.New(h.pool)
	widgets, err := queries.ListWidgetTypes(c.Request.Context())
	if err != nil {
		slog.Default().Error("Failed to list widget types", "error", err)
		utils.RespondInternalError(c, slog.Default(), "list widget types", err)
		return
	}

	c.JSON(http.StatusOK, transformWidgetTypes(widgets))
}

// GetWidgetSchema returns the config schema for a specific widget type
func (h *DashboardCRUDHandler) GetWidgetSchema(c *gin.Context) {
	widgetType := c.Param("type")
	if widgetType == "" {
		utils.RespondBadRequest(c, slog.Default(), "Widget type required")
		return
	}

	queries := sqlc.New(h.pool)
	widget, err := queries.GetWidgetTypeByName(c.Request.Context(), widgetType)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Widget type")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"widget_type":    widget.WidgetType,
		"name":           widget.Name,
		"description":    widget.Description,
		"category":       widget.Category,
		"config_schema":  json.RawMessage(widget.ConfigSchema),
		"default_config": json.RawMessage(widget.DefaultConfig),
		"default_size":   json.RawMessage(widget.DefaultSize),
		"min_size":       json.RawMessage(widget.MinSize),
		"sse_events":     widget.SseEvents,
	})
}

func (h *DashboardCRUDHandler) ListDashboardTemplates(c *gin.Context) {
	queries := sqlc.New(h.pool)
	templates, err := queries.ListDashboardTemplates(c.Request.Context())
	if err != nil {
		slog.Default().Error("Failed to list templates", "error", err)
		utils.RespondInternalError(c, slog.Default(), "list templates", err)
		return
	}

	c.JSON(http.StatusOK, transformTemplates(templates))
}

func (h *DashboardCRUDHandler) CreateDashboardFromTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template ID")
		return
	}

	var req struct {
		Name        string  `json:"name"`
		WorkspaceID *string `json:"workspace_id"`
	}
	_ = c.ShouldBindJSON(&req)

	var workspaceID pgtype.UUID
	if req.WorkspaceID != nil {
		if id, err := uuid.Parse(*req.WorkspaceID); err == nil {
			workspaceID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.CreateDashboardFromTemplate(c.Request.Context(), sqlc.CreateDashboardFromTemplateParams{
		ID:          pgtype.UUID{Bytes: templateID, Valid: true},
		UserID:      user.ID,
		WorkspaceID: workspaceID,
		Column4:     req.Name,
	})
	if err != nil {
		slog.Default().Error("Failed to create dashboard from template", "error", err, "template_id", templateID.String())
		utils.RespondInternalError(c, slog.Default(), "create dashboard from template", err)
		return
	}

	c.JSON(http.StatusCreated, transformDashboard(sqlc.UserDashboard(dashboard)))
}
