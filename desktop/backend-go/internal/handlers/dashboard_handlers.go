package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DashboardCRUDHandler handles all user dashboard, widget, and template operations.
type DashboardCRUDHandler struct {
	pool                *pgxpool.Pool
	notificationService *services.NotificationService
}

// NewDashboardCRUDHandler constructs a DashboardCRUDHandler.
func NewDashboardCRUDHandler(pool *pgxpool.Pool, notifService *services.NotificationService) *DashboardCRUDHandler {
	return &DashboardCRUDHandler{
		pool:                pool,
		notificationService: notifService,
	}
}

// RegisterDashboardCRUDRoutes registers all dashboard CRUD, widget, and template routes.
func RegisterDashboardCRUDRoutes(api *gin.RouterGroup, h *DashboardCRUDHandler, auth gin.HandlerFunc) {
	// Custom Dashboards - /api/user-dashboards
	userDashboards := api.Group("/user-dashboards")
	userDashboards.Use(auth, middleware.RequireAuth())
	{
		userDashboards.GET("", h.ListUserDashboards)
		userDashboards.POST("", h.CreateUserDashboard)
		userDashboards.GET("/:id", h.GetUserDashboard)
		userDashboards.PUT("/:id", h.UpdateUserDashboard)
		userDashboards.DELETE("/:id", h.DeleteUserDashboard)
		userDashboards.POST("/:id/duplicate", h.DuplicateUserDashboard)
		userDashboards.PUT("/:id/layout", h.UpdateDashboardLayout)
		userDashboards.POST("/:id/default", h.SetDefaultUserDashboard)
		userDashboards.POST("/:id/share", h.ShareUserDashboard)
	}
	// Public shared dashboard access (no auth)
	api.GET("/user-dashboards/shared/:token", h.GetSharedDashboard)

	// Widgets - /api/widgets
	widgets := api.Group("/widgets")
	widgets.Use(auth, middleware.RequireAuth())
	{
		widgets.GET("", h.ListWidgetTypes)
		widgets.GET("/:type/schema", h.GetWidgetSchema)
	}

	// Dashboard Templates - /api/dashboard-templates
	dashboardTemplates := api.Group("/dashboard-templates")
	dashboardTemplates.Use(auth, middleware.RequireAuth())
	{
		dashboardTemplates.GET("", h.ListDashboardTemplates)
		dashboardTemplates.POST("/create-from/:id", h.CreateDashboardFromTemplate)
	}
}

func (h *DashboardCRUDHandler) ListUserDashboards(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	dashboards, err := queries.ListUserDashboards(c.Request.Context(), user.ID)
	if err != nil {
		slog.Default().Error("Failed to list dashboards", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "list dashboards", err)
		return
	}

	c.JSON(http.StatusOK, transformDashboards(dashboards))
}

func (h *DashboardCRUDHandler) GetUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.GetDashboard(c.Request.Context(), sqlc.GetDashboardParams{
		ID:     pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Dashboard")
		return
	}

	c.JSON(http.StatusOK, transformDashboard(dashboard))
}

func (h *DashboardCRUDHandler) CreateUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name        string          `json:"name" binding:"required"`
		Description *string         `json:"description"`
		Layout      json.RawMessage `json:"layout"`
		Visibility  *string         `json:"visibility"`
		WorkspaceID *string         `json:"workspace_id"`
		CreatedVia  *string         `json:"created_via"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Default layout to nil if not provided (SimpleProtocol compatibility)
	layout := req.Layout

	// Parse workspace ID if provided
	var workspaceID pgtype.UUID
	if req.WorkspaceID != nil {
		if id, err := uuid.Parse(*req.WorkspaceID); err == nil {
			workspaceID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	// Default visibility
	visibility := "private"
	if req.Visibility != nil {
		visibility = *req.Visibility
	}

	// Default created_via
	createdVia := "manual"
	if req.CreatedVia != nil {
		createdVia = *req.CreatedVia
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.CreateDashboard(c.Request.Context(), sqlc.CreateDashboardParams{
		UserID:      user.ID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		Layout:      layout,
		Visibility:  &visibility,
		CreatedVia:  &createdVia,
	})
	if err != nil {
		slog.Default().Error("Failed to create dashboard", "error", err, "user_id", user.ID, "name", req.Name)
		utils.RespondInternalError(c, slog.Default(), "create dashboard", err)
		return
	}

	c.JSON(http.StatusCreated, transformDashboard(dashboard))
}

func (h *DashboardCRUDHandler) UpdateUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	var req struct {
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		Layout      json.RawMessage `json:"layout"`
		Visibility  *string         `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.UpdateDashboard(c.Request.Context(), sqlc.UpdateDashboardParams{
		ID:          pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID:      user.ID,
		Name:        req.Name,
		Description: req.Description,
		Layout:      req.Layout,
		Visibility:  req.Visibility,
	})
	if err != nil {
		slog.Default().Error("Failed to update dashboard", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "update dashboard", err)
		return
	}

	c.JSON(http.StatusOK, transformDashboard(dashboard))
}

func (h *DashboardCRUDHandler) DeleteUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteDashboard(c.Request.Context(), sqlc.DeleteDashboardParams{
		ID:     pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		slog.Default().Error("Failed to delete dashboard", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "delete dashboard", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dashboard deleted"})
}

// DuplicateUserDashboard creates a copy of a dashboard
func (h *DashboardCRUDHandler) DuplicateUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	// Name is optional, will default to "Copy of X"
	_ = c.ShouldBindJSON(&req)

	// Get original dashboard to build name if not provided
	queries := sqlc.New(h.pool)
	original, err := queries.GetDashboardByID(c.Request.Context(), pgtype.UUID{Bytes: dashboardID, Valid: true})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Dashboard")
		return
	}

	name := req.Name
	if name == "" {
		name = "Copy of " + original.Name
	}

	dashboard, err := queries.DuplicateDashboard(c.Request.Context(), sqlc.DuplicateDashboardParams{
		ID:     pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID: user.ID,
		Name:   name,
	})
	if err != nil {
		slog.Default().Error("Failed to duplicate dashboard", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "duplicate dashboard", err)
		return
	}

	c.JSON(http.StatusCreated, transformDashboard(sqlc.UserDashboard(dashboard)))
}

func (h *DashboardCRUDHandler) SetDefaultUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	queries := sqlc.New(h.pool)

	// First clear existing default
	err = queries.ClearDefaultDashboard(c.Request.Context(), user.ID)
	if err != nil {
		slog.Default().Error("Failed to clear default dashboard", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "clear default dashboard", err)
		return
	}

	// Set new default
	err = queries.SetDefaultDashboard(c.Request.Context(), sqlc.SetDefaultDashboardParams{
		ID:     pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		slog.Default().Error("Failed to set default dashboard", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "set default dashboard", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default dashboard updated"})
}
