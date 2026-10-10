package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/cache"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DashboardSummary represents the cached dashboard summary data
type DashboardSummary struct {
	Projects      interface{} `json:"projects"`
	Clients       interface{} `json:"clients"`
	Contexts      interface{} `json:"contexts"`
	Artifacts     interface{} `json:"artifacts"`
	Tasks         interface{} `json:"tasks"`
	FocusItems    interface{} `json:"focus_items"`
	Activities    interface{} `json:"activities"`
	EnergyLevel   int         `json:"energy_level"`
	ProjectCount  int         `json:"project_count"`
	ClientCount   int         `json:"client_count"`
	ContextCount  int         `json:"context_count"`
	ArtifactCount int         `json:"artifact_count"`
	TaskCount     int         `json:"task_count"`
}

// DashboardItemHandler handles dashboard focus items and tasks
type DashboardItemHandler struct {
	EngineSyncHook
	pool                 *pgxpool.Pool
	queryCache           *cache.QueryCache
	notificationTriggers *services.NotificationTriggers
}

// NewDashboardItemHandler creates a new DashboardItemHandler
func NewDashboardItemHandler(pool *pgxpool.Pool, queryCache *cache.QueryCache, notifTriggers *services.NotificationTriggers) *DashboardItemHandler {
	return &DashboardItemHandler{
		pool:                 pool,
		queryCache:           queryCache,
		notificationTriggers: notifTriggers,
	}
}

// RegisterDashboardItemRoutes registers all dashboard routes
func RegisterDashboardItemRoutes(api *gin.RouterGroup, h *DashboardItemHandler, auth gin.HandlerFunc) {
	dashboard := api.Group("/dashboard")
	dashboard.Use(auth, middleware.RequireAuth())
	{
		dashboard.GET("/summary", h.GetDashboardSummary)
		dashboard.GET("/focus", h.ListFocusItems)
		dashboard.POST("/focus", h.CreateFocusItem)
		dashboard.PUT("/focus/:id", h.UpdateFocusItem)
		dashboard.DELETE("/focus/:id", h.DeleteFocusItem)
		dashboard.GET("/tasks", h.ListTasks)
		dashboard.POST("/tasks", h.CreateTask)
		dashboard.PUT("/tasks/:id", h.UpdateTask)
		dashboard.POST("/tasks/:id/toggle", h.ToggleTask)
		dashboard.DELETE("/tasks/:id", h.DeleteTask)
	}

	tasks := api.Group("/tasks")
	tasks.Use(auth, middleware.RequireAuth())
	{
		tasks.GET("", h.ListTasks)
		tasks.POST("", h.CreateTask)
		tasks.PUT("/:id", h.UpdateTask)
		tasks.POST("/:id/toggle", h.ToggleTask)
		tasks.DELETE("/:id", h.DeleteTask)
	}
}

// GetDashboardSummary returns a summary of the user's data with Redis caching
func (h *DashboardItemHandler) GetDashboardSummary(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var workspaceID *uuid.UUID
	if c.GetHeader("X-Workspace-ID") != "" {
		resolved, scoped := h.taskWorkspaceScope(c, user.ID)
		if !scoped {
			c.JSON(http.StatusForbidden, gin.H{"error": "workspace access denied"})
			return
		}
		workspaceID = &resolved
	}

	// If cache is not available, fetch directly
	if h.queryCache == nil {
		h.fetchAndRespondDashboardSummary(c, user.ID, workspaceID)
		return
	}

	// Dashboard state is workspace-scoped. A user-only cache key can leak the
	// previous workspace's summary immediately after switching workspaces.
	cacheKey := fmt.Sprintf("dashboard:user:%s:summary", user.ID)
	if workspaceID != nil {
		cacheKey = fmt.Sprintf("dashboard:user:%s:workspace:%s:summary", user.ID, workspaceID.String())
	}

	// Try to get from cache
	var cachedSummary DashboardSummary
	if err := h.queryCache.GetOrCompute(
		c.Request.Context(),
		cacheKey,
		1*time.Minute, // 1 minute TTL for dashboard summary
		&cachedSummary,
		func() (interface{}, error) {
			return h.computeDashboardSummary(c, user.ID, workspaceID)
		},
	); err != nil {
		slog.Default().Error("Failed to get dashboard summary from cache", "error", err, "user_id", user.ID)
		// Fall back to fetching directly
		h.fetchAndRespondDashboardSummary(c, user.ID, workspaceID)
		return
	}

	c.JSON(http.StatusOK, cachedSummary)
}

// computeDashboardSummary computes the dashboard summary data
func (h *DashboardItemHandler) computeDashboardSummary(c *gin.Context, userID string, workspaceID *uuid.UUID) (interface{}, error) {
	if workspaceID != nil {
		var projectCount, clientCount, contextCount, taskCount int
		err := h.pool.QueryRow(c.Request.Context(), `
			SELECT
				(SELECT COUNT(*) FROM projects WHERE workspace_id = $1),
				(SELECT COUNT(*) FROM clients WHERE workspace_id = $1),
				(SELECT COUNT(*) FROM contexts WHERE workspace_id = $1),
				(SELECT COUNT(*) FROM tasks WHERE workspace_id = $1)
		`, *workspaceID).Scan(&projectCount, &clientCount, &contextCount, &taskCount)
		if err != nil {
			return nil, fmt.Errorf("compute workspace dashboard summary: %w", err)
		}

		// The legacy summary arrays are user-scoped and must never be returned
		// for an explicitly selected workspace. Workspace modules load their own
		// scoped records; this surface only needs safe at-a-glance counts.
		return DashboardSummary{
			Projects:      []interface{}{},
			Clients:       []interface{}{},
			Contexts:      []interface{}{},
			Artifacts:     []interface{}{},
			Tasks:         []interface{}{},
			FocusItems:    []interface{}{},
			Activities:    []interface{}{},
			EnergyLevel:   3,
			ProjectCount:  projectCount,
			ClientCount:   clientCount,
			ContextCount:  contextCount,
			ArtifactCount: 0,
			TaskCount:     taskCount,
		}, nil
	}

	queries := sqlc.New(h.pool)

	// Get data for various entities
	projectRows, _ := queries.ListProjects(c.Request.Context(), sqlc.ListProjectsParams{UserID: userID})
	clients, _ := queries.ListClients(c.Request.Context(), sqlc.ListClientsParams{UserID: userID})
	contexts, _ := queries.ListContexts(c.Request.Context(), sqlc.ListContextsParams{UserID: userID})
	artifacts, _ := queries.ListArtifacts(c.Request.Context(), sqlc.ListArtifactsParams{UserID: userID})
	tasks, _ := queries.ListTasks(c.Request.Context(), sqlc.ListTasksParams{UserID: userID})

	// Get today's focus items
	today := time.Now()
	focusItems, _ := queries.ListFocusItems(c.Request.Context(), sqlc.ListFocusItemsParams{
		UserID:    userID,
		FocusDate: pgtype.Date{Time: today, Valid: true},
	})

	// Ensure arrays are not nil (return empty arrays instead)
	if projectRows == nil {
		projectRows = []sqlc.ListProjectsRow{}
	}
	if clients == nil {
		clients = []sqlc.Client{}
	}
	if contexts == nil {
		contexts = []sqlc.Context{}
	}
	if artifacts == nil {
		artifacts = []sqlc.Artifact{}
	}
	if tasks == nil {
		tasks = []sqlc.Task{}
	}
	if focusItems == nil {
		focusItems = []sqlc.FocusItem{}
	}

	return DashboardSummary{
		Projects:      TransformProjectRows(projectRows),
		Clients:       TransformClients(clients),
		Contexts:      TransformContexts(contexts),
		Artifacts:     TransformArtifacts(artifacts),
		Tasks:         TransformTasks(tasks),
		FocusItems:    TransformFocusItems(focusItems),
		Activities:    []interface{}{}, // Placeholder for activities
		EnergyLevel:   3,               // Default energy level (1-5 scale)
		ProjectCount:  len(projectRows),
		ClientCount:   len(clients),
		ContextCount:  len(contexts),
		ArtifactCount: len(artifacts),
		TaskCount:     len(tasks),
	}, nil
}

// fetchAndRespondDashboardSummary fetches dashboard summary without caching
func (h *DashboardItemHandler) fetchAndRespondDashboardSummary(c *gin.Context, userID string, workspaceID *uuid.UUID) {
	summary, err := h.computeDashboardSummary(c, userID, workspaceID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "compute dashboard summary", err)
		return
	}

	c.JSON(http.StatusOK, summary)
}
