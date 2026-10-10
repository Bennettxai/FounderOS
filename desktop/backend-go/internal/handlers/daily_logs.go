package handlers

import (
	"encoding/json"
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
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DailyLogHandler handles daily log HTTP requests.
type DailyLogHandler struct {
	EngineSyncHook
	pool *pgxpool.Pool
}

// NewDailyLogHandler constructs a DailyLogHandler.
func NewDailyLogHandler(pool *pgxpool.Pool) *DailyLogHandler {
	return &DailyLogHandler{pool: pool}
}

// RegisterDailyLogRoutes mounts daily log routes onto the provided router group.
func RegisterDailyLogRoutes(api *gin.RouterGroup, h *DailyLogHandler, auth gin.HandlerFunc) {
	daily := api.Group("/daily")
	daily.Use(auth, middleware.RequireAuth())
	{
		daily.GET("/logs", h.ListDailyLogs)
		daily.GET("/logs/today", h.GetTodayLog)
		daily.GET("/logs/:date", h.GetDailyLogByDate)
		daily.POST("/logs", h.CreateOrUpdateDailyLog)
		daily.PUT("/logs/:id", h.UpdateDailyLog)
		daily.DELETE("/logs/:id", h.DeleteDailyLog)
	}
}

// ListDailyLogs returns daily logs for the current user.
func (h *DailyLogHandler) ListDailyLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse pagination params
	limit := int32(30)
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	offset := int32(0)
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	logs, err := queries.ListDailyLogs(c.Request.Context(), sqlc.ListDailyLogsParams{
		UserID: user.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list daily logs", err)
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetTodayLog returns today's daily log.
func (h *DailyLogHandler) GetTodayLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	log, err := queries.GetTodayLog(c.Request.Context(), user.ID)
	if err != nil {
		// Return null instead of 404 for missing today log
		c.JSON(http.StatusOK, nil)
		return
	}

	c.JSON(http.StatusOK, transformDailyLog(log))
}

// transformDailyLog converts sqlc DailyLog to JSON-friendly format.
func transformDailyLog(log sqlc.DailyLog) map[string]interface{} {
	result := map[string]interface{}{
		"id":                   uuidToString(log.ID),
		"user_id":              log.UserID,
		"date":                 log.Date.Time.Format("2006-01-02"),
		"content":              log.Content,
		"transcription_source": log.TranscriptionSource,
		"energy_level":         log.EnergyLevel,
	}

	// Handle JSONB fields
	if log.ExtractedActions != nil {
		var actions interface{}
		if err := json.Unmarshal(log.ExtractedActions, &actions); err == nil {
			result["extracted_actions"] = actions
		} else {
			result["extracted_actions"] = nil
		}
	} else {
		result["extracted_actions"] = nil
	}

	if log.ExtractedPatterns != nil {
		var patterns interface{}
		if err := json.Unmarshal(log.ExtractedPatterns, &patterns); err == nil {
			result["extracted_patterns"] = patterns
		} else {
			result["extracted_patterns"] = nil
		}
	} else {
		result["extracted_patterns"] = nil
	}

	if log.CreatedAt.Valid {
		result["created_at"] = log.CreatedAt.Time.Format(time.RFC3339)
	} else {
		result["created_at"] = nil
	}

	if log.UpdatedAt.Valid {
		result["updated_at"] = log.UpdatedAt.Time.Format(time.RFC3339)
	} else {
		result["updated_at"] = nil
	}

	return result
}

// GetDailyLogByDate returns a daily log for a specific date.
func (h *DailyLogHandler) GetDailyLogByDate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	log, err := queries.GetDailyLogByDate(c.Request.Context(), sqlc.GetDailyLogByDateParams{
		UserID: user.ID,
		Date:   pgtype.Date{Time: date, Valid: true},
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Daily log")
		return
	}

	c.JSON(http.StatusOK, log)
}

// CreateDailyLog creates a new daily log.
func (h *DailyLogHandler) CreateDailyLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Date                string   `json:"date"`
		Content             string   `json:"content" binding:"required"`
		TranscriptionSource *string  `json:"transcription_source"`
		ExtractedActions    []string `json:"extracted_actions"`
		ExtractedPatterns   []string `json:"extracted_patterns"`
		EnergyLevel         *int32   `json:"energy_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse date, default to today
	logDate := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			logDate = t
		}
	}

	// Handle JSONB arrays
	var extractedActions []byte
	if len(req.ExtractedActions) > 0 {
		if actionsJSON, err := json.Marshal(req.ExtractedActions); err == nil {
			extractedActions = actionsJSON
		}
	}

	var extractedPatterns []byte
	if len(req.ExtractedPatterns) > 0 {
		if patternsJSON, err := json.Marshal(req.ExtractedPatterns); err == nil {
			extractedPatterns = patternsJSON
		}
	}

	log, err := queries.CreateDailyLog(c.Request.Context(), sqlc.CreateDailyLogParams{
		UserID:              user.ID,
		Date:                pgtype.Date{Time: logDate, Valid: true},
		Content:             req.Content,
		TranscriptionSource: req.TranscriptionSource,
		ExtractedActions:    extractedActions,
		ExtractedPatterns:   extractedPatterns,
		EnergyLevel:         req.EnergyLevel,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create daily log", err)
		return
	}

	// Mirror into the OptimalEngine knowledge graph so the daily log
	// surfaces in the Pages graph view alongside other modules' content.
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleDailyLog,
		ID:         uuidString(log.ID.Bytes),
		AuthorID:   user.ID,
		Title:      "Daily log " + logDate.Format("2006-01-02"),
		Body:       log.Content,
		Genre:      "log_entry",
		ModifiedAt: pgPlainTimestampToTime(log.UpdatedAt),
		Metadata: map[string]string{
			"date": logDate.Format("2006-01-02"),
		},
	})

	c.JSON(http.StatusCreated, log)
}

// CreateOrUpdateDailyLog creates or updates a daily log for a date.
func (h *DailyLogHandler) CreateOrUpdateDailyLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Date                string   `json:"date"`
		Content             string   `json:"content" binding:"required"`
		TranscriptionSource *string  `json:"transcription_source"`
		ExtractedActions    []string `json:"extracted_actions"`
		ExtractedPatterns   []string `json:"extracted_patterns"`
		EnergyLevel         *int32   `json:"energy_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse date, default to today
	logDate := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			logDate = t
		}
	}

	// Handle JSONB arrays
	var extractedActions []byte
	if len(req.ExtractedActions) > 0 {
		if actionsJSON, err := json.Marshal(req.ExtractedActions); err == nil {
			extractedActions = actionsJSON
		}
	}

	var extractedPatterns []byte
	if len(req.ExtractedPatterns) > 0 {
		if patternsJSON, err := json.Marshal(req.ExtractedPatterns); err == nil {
			extractedPatterns = patternsJSON
		}
	}

	log, err := queries.UpsertDailyLog(c.Request.Context(), sqlc.UpsertDailyLogParams{
		UserID:              user.ID,
		Date:                pgtype.Date{Time: logDate, Valid: true},
		Content:             req.Content,
		TranscriptionSource: req.TranscriptionSource,
		ExtractedActions:    extractedActions,
		ExtractedPatterns:   extractedPatterns,
		EnergyLevel:         req.EnergyLevel,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "upsert daily log", err)
		return
	}

	c.JSON(http.StatusOK, log)
}

// UpdateDailyLog updates an existing daily log.
func (h *DailyLogHandler) UpdateDailyLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "daily log ID")
		return
	}

	var req struct {
		Content             string   `json:"content" binding:"required"`
		TranscriptionSource *string  `json:"transcription_source"`
		ExtractedActions    []string `json:"extracted_actions"`
		ExtractedPatterns   []string `json:"extracted_patterns"`
		EnergyLevel         *int32   `json:"energy_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Handle JSONB arrays
	var extractedActions []byte
	if len(req.ExtractedActions) > 0 {
		if actionsJSON, err := json.Marshal(req.ExtractedActions); err == nil {
			extractedActions = actionsJSON
		}
	}

	var extractedPatterns []byte
	if len(req.ExtractedPatterns) > 0 {
		if patternsJSON, err := json.Marshal(req.ExtractedPatterns); err == nil {
			extractedPatterns = patternsJSON
		}
	}

	log, err := queries.UpdateDailyLog(c.Request.Context(), sqlc.UpdateDailyLogParams{
		ID:                  pgtype.UUID{Bytes: id, Valid: true},
		Content:             req.Content,
		TranscriptionSource: req.TranscriptionSource,
		ExtractedActions:    extractedActions,
		ExtractedPatterns:   extractedPatterns,
		EnergyLevel:         req.EnergyLevel,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update daily log", err)
		return
	}

	c.JSON(http.StatusOK, log)
}

// AppendToDailyLog appends content to today's daily log (or creates it).
func (h *DailyLogHandler) AppendToDailyLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Try to get today's log
	existingLog, err := queries.GetTodayLog(c.Request.Context(), user.ID)
	if err != nil {
		// Create new log for today
		log, err := queries.CreateDailyLog(c.Request.Context(), sqlc.CreateDailyLogParams{
			UserID:            user.ID,
			Date:              pgtype.Date{Time: time.Now(), Valid: true},
			Content:           req.Content,
			ExtractedActions:  nil,
			ExtractedPatterns: nil,
		})
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "create daily log", err)
			return
		}
		c.JSON(http.StatusCreated, log)
		return
	}

	// Append to existing log
	newContent := existingLog.Content + "\n\n" + req.Content
	log, err := queries.UpdateDailyLog(c.Request.Context(), sqlc.UpdateDailyLogParams{
		ID:                  existingLog.ID,
		Content:             newContent,
		TranscriptionSource: existingLog.TranscriptionSource,
		ExtractedActions:    existingLog.ExtractedActions,
		ExtractedPatterns:   existingLog.ExtractedPatterns,
		EnergyLevel:         existingLog.EnergyLevel,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "append to daily log", err)
		return
	}

	c.JSON(http.StatusOK, log)
}

// DeleteDailyLog deletes a daily log.
func (h *DailyLogHandler) DeleteDailyLog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "daily log ID")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteDailyLog(c.Request.Context(), sqlc.DeleteDailyLogParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete daily log", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Daily log deleted"})
}
