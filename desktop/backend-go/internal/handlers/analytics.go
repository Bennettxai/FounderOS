package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// AnalyticsHandler handles aggregated analytics endpoints.
type AnalyticsHandler struct {
	pool *pgxpool.Pool
}

// NewAnalyticsHandler constructs an AnalyticsHandler.
func NewAnalyticsHandler(pool *pgxpool.Pool) *AnalyticsHandler {
	return &AnalyticsHandler{pool: pool}
}

// RegisterAnalyticsRoutes wires /api/analytics routes.
func RegisterAnalyticsRoutes(api *gin.RouterGroup, h *AnalyticsHandler, auth gin.HandlerFunc) {
	analytics := api.Group("/analytics")
	analytics.Use(auth, middleware.RequireAuth())
	{
		analytics.GET("/summary", h.GetAnalyticsSummary)
		analytics.GET("/task-burndown", h.GetTaskBurndown)
		analytics.GET("/workload", h.GetWorkloadHeatmap)
		analytics.GET("/upcoming-deadlines", h.GetUpcomingDeadlines)
	}
}

// ============================================================================
// ANALYTICS HANDLERS
// Aggregated data endpoints for dashboard widgets
// ============================================================================

// AnalyticsSummary represents the summary metrics response
type AnalyticsSummary struct {
	TasksDueToday          int64 `json:"tasks_due_today"`
	TasksOverdue           int64 `json:"tasks_overdue"`
	TasksCompletedThisWeek int64 `json:"tasks_completed_this_week"`
	ActiveProjects         int64 `json:"active_projects"`
}

// GetAnalyticsSummary returns aggregated counts for the metric card widgets
func (h *AnalyticsHandler) GetAnalyticsSummary(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	ctx := c.Request.Context()

	// Run all counts - could be parallelized but COUNT queries are fast
	tasksDueToday, err := queries.CountTasksDueToday(ctx, user.ID)
	if err != nil {
		slog.Info("CountTasksDueToday error", "error", err)
		tasksDueToday = 0
	}

	tasksOverdue, err := queries.CountTasksOverdue(ctx, user.ID)
	if err != nil {
		slog.Info("CountTasksOverdue error", "error", err)
		tasksOverdue = 0
	}

	tasksCompletedThisWeek, err := queries.CountTasksCompletedThisWeek(ctx, user.ID)
	if err != nil {
		slog.Info("CountTasksCompletedThisWeek error", "error", err)
		tasksCompletedThisWeek = 0
	}

	activeProjects, err := queries.CountActiveProjects(ctx, user.ID)
	if err != nil {
		slog.Info("CountActiveProjects error", "error", err)
		activeProjects = 0
	}

	c.JSON(http.StatusOK, AnalyticsSummary{
		TasksDueToday:          tasksDueToday,
		TasksOverdue:           tasksOverdue,
		TasksCompletedThisWeek: tasksCompletedThisWeek,
		ActiveProjects:         activeProjects,
	})
}

// BurndownData represents the task burndown chart data
type BurndownData struct {
	Dates     []string `json:"dates"`
	Created   []int64  `json:"created"`
	Completed []int64  `json:"completed"`
}

// GetTaskBurndown returns task creation vs completion data over time
func (h *AnalyticsHandler) GetTaskBurndown(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse query params
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	var projectID pgtype.UUID
	if pid := c.Query("project_id"); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			projectID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	queries := sqlc.New(h.pool)
	rows, err := queries.GetTaskBurndownData(c.Request.Context(), sqlc.GetTaskBurndownDataParams{
		UserID:  user.ID,
		Column2: int32(days),
		Column3: projectID,
	})
	if err != nil {
		slog.Info("GetTaskBurndown error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get burndown data", err)
		return
	}

	// Transform to response format
	data := BurndownData{
		Dates:     make([]string, len(rows)),
		Created:   make([]int64, len(rows)),
		Completed: make([]int64, len(rows)),
	}

	for i, row := range rows {
		data.Dates[i] = row.Date.Time.Format("2006-01-02")
		data.Created[i] = row.Created
		data.Completed[i] = row.Completed
	}

	c.JSON(http.StatusOK, data)
}

// WorkloadEntry represents a single day's workload data
type WorkloadEntry struct {
	TasksDue     int64 `json:"tasks_due"`
	TasksCreated int64 `json:"tasks_created"`
}

// GetWorkloadHeatmap returns task density data for a calendar heatmap
func (h *AnalyticsHandler) GetWorkloadHeatmap(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse date range
	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // Default: last month
	endDate := now

	if s := c.Query("start"); s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			startDate = parsed
		}
	}
	if e := c.Query("end"); e != "" {
		if parsed, err := time.Parse("2006-01-02", e); err == nil {
			endDate = parsed
		}
	}

	queries := sqlc.New(h.pool)
	rows, err := queries.GetWorkloadHeatmapData(c.Request.Context(), sqlc.GetWorkloadHeatmapDataParams{
		UserID:  user.ID,
		Column2: pgtype.Date{Time: startDate, Valid: true},
		Column3: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		slog.Info("GetWorkloadHeatmap error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get workload data", err)
		return
	}

	// Transform to map format
	data := make(map[string]WorkloadEntry)
	for _, row := range rows {
		dateStr := row.Date.Time.Format("2006-01-02")
		data[dateStr] = WorkloadEntry{
			TasksDue:     row.TasksDue,
			TasksCreated: row.TasksCreated,
		}
	}

	c.JSON(http.StatusOK, data)
}

// GetUpcomingDeadlines returns tasks grouped by due date for the next N days
func (h *AnalyticsHandler) GetUpcomingDeadlines(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	queries := sqlc.New(h.pool)
	rows, err := queries.GetUpcomingTasksDueByDate(c.Request.Context(), sqlc.GetUpcomingTasksDueByDateParams{
		UserID:  user.ID,
		Column2: int32(days),
	})
	if err != nil {
		slog.Info("GetUpcomingDeadlines error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "get upcoming deadlines", err)
		return
	}

	// Transform to response
	result := make([]gin.H, len(rows))
	for i, row := range rows {
		result[i] = gin.H{
			"date":       row.DueDate.Time.Format("2006-01-02"),
			"task_count": row.TaskCount,
		}
	}

	c.JSON(http.StatusOK, result)
}
