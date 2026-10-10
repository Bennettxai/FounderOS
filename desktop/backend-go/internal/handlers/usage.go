package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// UsageHandler handles usage analytics HTTP requests.
type UsageHandler struct {
	pool *pgxpool.Pool
}

// NewUsageHandler constructs a UsageHandler.
func NewUsageHandler(pool *pgxpool.Pool) *UsageHandler {
	return &UsageHandler{pool: pool}
}

// RegisterUsageRoutes mounts usage routes onto the provided router group.
func RegisterUsageRoutes(api *gin.RouterGroup, h *UsageHandler, auth gin.HandlerFunc) {
	usage := api.Group("/usage")
	usage.Use(auth, middleware.RequireAuth())
	{
		usage.GET("/summary", h.GetUsageSummary)
		usage.GET("/providers", h.GetUsageByProvider)
		usage.GET("/models", h.GetUsageByModel)
		usage.GET("/agents", h.GetUsageByAgent)
		usage.GET("/trend", h.GetUsageTrend)
		usage.GET("/recent", h.GetRecentAIUsage)
		usage.GET("/mcp", h.GetMCPUsage)
	}
}

// UsageSummaryResponse represents the usage summary.
type UsageSummaryResponse struct {
	TotalRequests     int64   `json:"total_requests"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalCost         float64 `json:"total_cost"`
	Period            string  `json:"period"`
	StartDate         string  `json:"start_date"`
	EndDate           string  `json:"end_date"`
}

// ProviderUsageResponse represents usage by provider.
type ProviderUsageResponse struct {
	Provider          string  `json:"provider"`
	RequestCount      int64   `json:"request_count"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalCost         float64 `json:"total_cost"`
}

// ModelUsageResponse represents usage by model.
type ModelUsageResponse struct {
	Model             string  `json:"model"`
	Provider          string  `json:"provider"`
	RequestCount      int64   `json:"request_count"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalCost         float64 `json:"total_cost"`
}

// AgentUsageResponse represents usage by agent.
type AgentUsageResponse struct {
	AgentName         string  `json:"agent_name"`
	RequestCount      int64   `json:"request_count"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
}

// UsageTrendResponse represents daily usage trend.
type UsageTrendResponse struct {
	Date          string  `json:"date"`
	AIRequests    int32   `json:"ai_requests"`
	TotalTokens   int32   `json:"total_tokens"`
	EstimatedCost float64 `json:"estimated_cost"`
	MCPRequests   int32   `json:"mcp_requests"`
	MessagesSent  int32   `json:"messages_sent"`
}

// GetUsageSummary returns usage summary for a period.
func (h *UsageHandler) GetUsageSummary(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	period := c.DefaultQuery("period", "today")

	var startTime, endTime time.Time
	now := time.Now()

	switch period {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 0, 1)
	case "week":
		weekday := int(now.Weekday())
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 0, 7)
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	case "all":
		startTime = time.Date(2020, 1, 1, 0, 0, 0, 0, now.Location())
		endTime = now.AddDate(1, 0, 0)
	default:
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	queries := sqlc.New(h.pool)

	result, err := queries.GetTotalTokensForPeriod(c.Request.Context(), sqlc.GetTotalTokensForPeriodParams{
		UserID:      user.ID,
		StartedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get usage summary", err)
		return
	}

	// Convert interface{} values to int64
	inputTokens := int64(0)
	outputTokens := int64(0)
	totalTokens := int64(0)
	totalCost := float64(0)

	if v, ok := result.TotalInputTokens.(int64); ok {
		inputTokens = v
	}
	if v, ok := result.TotalOutputTokens.(int64); ok {
		outputTokens = v
	}
	if v, ok := result.TotalTokens.(int64); ok {
		totalTokens = v
	}
	switch v := result.TotalCost.(type) {
	case float64:
		totalCost = v
	case int64:
		totalCost = float64(v)
	}

	c.JSON(http.StatusOK, UsageSummaryResponse{
		TotalRequests:     result.TotalRequests,
		TotalInputTokens:  inputTokens,
		TotalOutputTokens: outputTokens,
		TotalTokens:       totalTokens,
		TotalCost:         totalCost,
		Period:            period,
		StartDate:         startTime.Format("2006-01-02"),
		EndDate:           endTime.Format("2006-01-02"),
	})
}

// GetUsageByProvider returns usage breakdown by provider.
func (h *UsageHandler) GetUsageByProvider(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	period := c.DefaultQuery("period", "month")
	startTime, endTime := getPeriodRange(period)

	queries := sqlc.New(h.pool)

	results, err := queries.GetAIUsageSummaryByProvider(c.Request.Context(), sqlc.GetAIUsageSummaryByProviderParams{
		UserID:      user.ID,
		StartedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get provider usage", err)
		return
	}

	response := make([]ProviderUsageResponse, len(results))
	for i, r := range results {
		response[i] = ProviderUsageResponse{
			Provider:          r.Provider,
			RequestCount:      r.RequestCount,
			TotalInputTokens:  r.TotalInputTokens,
			TotalOutputTokens: r.TotalOutputTokens,
			TotalTokens:       r.TotalTokens,
			TotalCost:         float64(r.TotalCost) / 100, // Convert cents to dollars
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetUsageByModel returns usage breakdown by model.
func (h *UsageHandler) GetUsageByModel(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	period := c.DefaultQuery("period", "month")
	startTime, endTime := getPeriodRange(period)

	queries := sqlc.New(h.pool)

	results, err := queries.GetAIUsageSummaryByModel(c.Request.Context(), sqlc.GetAIUsageSummaryByModelParams{
		UserID:      user.ID,
		StartedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get model usage", err)
		return
	}

	response := make([]ModelUsageResponse, len(results))
	for i, r := range results {
		response[i] = ModelUsageResponse{
			Model:             r.Model,
			Provider:          r.Provider,
			RequestCount:      r.RequestCount,
			TotalInputTokens:  r.TotalInputTokens,
			TotalOutputTokens: r.TotalOutputTokens,
			TotalTokens:       r.TotalTokens,
			TotalCost:         float64(r.TotalCost) / 100,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetUsageByAgent returns usage breakdown by agent.
func (h *UsageHandler) GetUsageByAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	period := c.DefaultQuery("period", "month")
	startTime, endTime := getPeriodRange(period)

	queries := sqlc.New(h.pool)

	results, err := queries.GetAIUsageSummaryByAgent(c.Request.Context(), sqlc.GetAIUsageSummaryByAgentParams{
		UserID:      user.ID,
		StartedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get agent usage", err)
		return
	}

	response := make([]AgentUsageResponse, len(results))
	for i, r := range results {
		agentName := "unknown"
		if r.AgentName != nil {
			agentName = *r.AgentName
		}
		response[i] = AgentUsageResponse{
			AgentName:         agentName,
			RequestCount:      r.RequestCount,
			TotalInputTokens:  r.TotalInputTokens,
			TotalOutputTokens: r.TotalOutputTokens,
			TotalTokens:       r.TotalTokens,
			AvgDurationMs:     r.AvgDurationMs,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetUsageTrend returns daily usage trend.
func (h *UsageHandler) GetUsageTrend(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Default to last 30 days
	days := 30
	now := time.Now()
	startDate := now.AddDate(0, 0, -days)
	endDate := now

	queries := sqlc.New(h.pool)

	results, err := queries.GetUsageTrend(c.Request.Context(), sqlc.GetUsageTrendParams{
		UserID: user.ID,
		Date:   pgtype.Date{Time: startDate, Valid: true},
		Date_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get usage trend", err)
		return
	}

	response := make([]UsageTrendResponse, len(results))
	for i, r := range results {
		aiRequests := int32(0)
		if r.AiRequests != nil {
			aiRequests = *r.AiRequests
		}
		totalTokens := int32(0)
		if r.AiTotalTokens != nil {
			totalTokens = *r.AiTotalTokens
		}
		mcpRequests := int32(0)
		if r.McpRequests != nil {
			mcpRequests = *r.McpRequests
		}
		messagesSent := int32(0)
		if r.MessagesSent != nil {
			messagesSent = *r.MessagesSent
		}

		cost := float64(0)
		if r.AiEstimatedCost.Valid {
			f, _ := r.AiEstimatedCost.Float64Value()
			if f.Valid {
				cost = f.Float64
			}
		}

		response[i] = UsageTrendResponse{
			Date:          r.Date.Time.Format("2006-01-02"),
			AIRequests:    aiRequests,
			TotalTokens:   totalTokens,
			EstimatedCost: cost,
			MCPRequests:   mcpRequests,
			MessagesSent:  messagesSent,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetRecentAIUsage returns recent AI usage logs.
func (h *UsageHandler) GetRecentAIUsage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	results, err := queries.GetAIUsageLogs(c.Request.Context(), sqlc.GetAIUsageLogsParams{
		UserID: user.ID,
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get AI usage logs", err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetMCPUsage returns MCP tool usage summary.
func (h *UsageHandler) GetMCPUsage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	period := c.DefaultQuery("period", "month")
	startTime, endTime := getPeriodRange(period)

	queries := sqlc.New(h.pool)

	results, err := queries.GetMCPUsageSummaryByTool(c.Request.Context(), sqlc.GetMCPUsageSummaryByToolParams{
		UserID:      user.ID,
		CreatedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get MCP usage", err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// getPeriodRange returns start and end times for a named period.
func getPeriodRange(period string) (time.Time, time.Time) {
	now := time.Now()
	var startTime, endTime time.Time

	switch period {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 0, 1)
	case "week":
		weekday := int(now.Weekday())
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 0, 7)
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	case "year":
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(1, 0, 0)
	default:
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	}

	return startTime, endTime
}
