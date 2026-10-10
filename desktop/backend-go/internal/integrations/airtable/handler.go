package airtable

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	integrations "github.com/rhl/businessos-backend/internal/integrations"
)

// Handler provides HTTP handlers for Airtable integration routes.
type Handler struct {
	provider *Provider
}

// NewHandler creates a new Airtable integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
	}
}

// RegisterRoutes registers all Airtable integration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// OAuth routes
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Bases routes
	bases := r.Group("/bases")
	{
		bases.GET("", h.GetBases)
		bases.GET("/:id", h.GetBase)
		bases.POST("/sync", h.SyncBases)
	}

	// Tables routes
	tables := r.Group("/tables")
	{
		tables.GET("/:id", h.GetTable)
	}

	// Base-specific tables routes
	r.GET("/bases/:base_id/tables", h.GetTables)

	// Records routes
	records := r.Group("/records")
	{
		records.GET("/:id", h.GetRecord)
		records.PUT("/:id", h.UpdateRecord)
		records.DELETE("/:id", h.DeleteRecord)
	}

	// Table-specific records routes
	tableRecords := r.Group("/tables/:table_id/records")
	{
		tableRecords.GET("", h.GetRecords)
		tableRecords.POST("", h.CreateRecord)
		tableRecords.POST("/sync", h.SyncRecords)
	}
}

// ============================================================================
// OAuth Handlers
// ============================================================================

// GetAuthURL returns the OAuth authorization URL.
func (h *Handler) GetAuthURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	state := integrations.GenerateUserState(userID)
	authURL := h.provider.GetAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// HandleCallback handles the OAuth callback.
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	userID := integrations.ExtractUserIDFromState(state)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	token, err := h.provider.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Info("Failed to exchange code", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code"})
		return
	}

	if err := h.provider.SaveToken(c.Request.Context(), userID, token); err != nil {
		slog.Info("Failed to save token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"email":   token.AccountEmail,
		"scopes":  token.Scopes,
	})
}

// Disconnect disconnects the Airtable integration.
func (h *Handler) Disconnect(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.provider.Disconnect(c.Request.Context(), userID); err != nil {
		slog.Info("Failed to disconnect", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStatus returns the connection status.
func (h *Handler) GetStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	status, err := h.provider.GetConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ============================================================================
// Bases Handlers
// ============================================================================

// GetBases returns the user's Airtable bases.
func (h *Handler) GetBases(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	bases, err := h.provider.GetBases(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get bases", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bases": bases,
		"count": len(bases),
	})
}

// GetBase returns a specific base by ID.
func (h *Handler) GetBase(c *gin.Context) {
	userID := c.GetString("user_id")
	baseID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	base, err := h.provider.GetBase(c.Request.Context(), userID, baseID)
	if err != nil {
		slog.Info("Failed to get base", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get base"})
		return
	}

	c.JSON(http.StatusOK, base)
}

// SyncBases syncs all bases from Airtable.
func (h *Handler) SyncBases(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Sync bases using provider method
	synced, err := h.provider.SyncBases(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to sync bases", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync bases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"synced":  synced,
	})
}

// ============================================================================
// Tables Handlers
// ============================================================================

// GetTables returns all tables in a specific base.
func (h *Handler) GetTables(c *gin.Context) {
	userID := c.GetString("user_id")
	baseID := c.Param("base_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tables, err := h.provider.GetTables(c.Request.Context(), userID, baseID)
	if err != nil {
		slog.Info("Failed to get tables", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tables"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tables": tables,
		"count":  len(tables),
	})
}

// GetTable returns a specific table by ID.
func (h *Handler) GetTable(c *gin.Context) {
	userID := c.GetString("user_id")
	tableID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Note: To get a specific table, we need the base ID
	// This is a limitation - the table ID alone isn't enough
	// Client should provide base_id as query param
	baseID := c.Query("base_id")
	if baseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id query parameter"})
		return
	}

	table, err := h.provider.GetTable(c.Request.Context(), userID, baseID, tableID)
	if err != nil {
		slog.Info("Failed to get table", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get table"})
		return
	}

	c.JSON(http.StatusOK, table)
}

// ============================================================================
// Records Handlers
// ============================================================================

// GetRecords returns records from a specific table.
func (h *Handler) GetRecords(c *gin.Context) {
	userID := c.GetString("user_id")
	tableID := c.Param("table_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID is required for Airtable API
	baseID := c.Query("base_id")
	if baseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id query parameter"})
		return
	}

	// Parse query options
	options := &RecordQueryOptions{}

	if maxRecords := c.Query("max_records"); maxRecords != "" {
		if val, err := strconv.Atoi(maxRecords); err == nil {
			options.MaxRecords = val
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if val, err := strconv.Atoi(pageSize); err == nil {
			options.PageSize = val
		}
	}

	options.Offset = c.Query("offset")
	options.View = c.Query("view")
	options.FilterByFormula = c.Query("filter")
	options.Sort = c.Query("sort")

	recordList, err := h.provider.GetRecords(c.Request.Context(), userID, baseID, tableID, options)
	if err != nil {
		slog.Info("Failed to get records", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": recordList.Records,
		"count":   len(recordList.Records),
		"offset":  recordList.Offset,
	})
}

// GetRecord returns a specific record by ID.
func (h *Handler) GetRecord(c *gin.Context) {
	userID := c.GetString("user_id")
	recordID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID and Table ID are required
	baseID := c.Query("base_id")
	tableID := c.Query("table_id")

	if baseID == "" || tableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id or table_id query parameters"})
		return
	}

	// Get all records and filter
	// Note: Airtable doesn't have a direct "get single record by ID" endpoint
	// We use the list endpoint with formula filter
	options := &RecordQueryOptions{
		MaxRecords:      1,
		FilterByFormula: "RECORD_ID() = '" + recordID + "'",
	}

	recordList, err := h.provider.GetRecords(c.Request.Context(), userID, baseID, tableID, options)
	if err != nil {
		slog.Info("Failed to get record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get record"})
		return
	}

	if len(recordList.Records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, recordList.Records[0])
}

// CreateRecord creates a new record in a table.
func (h *Handler) CreateRecord(c *gin.Context) {
	userID := c.GetString("user_id")
	tableID := c.Param("table_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID is required
	baseID := c.Query("base_id")
	if baseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id query parameter"})
		return
	}

	var req struct {
		Fields map[string]interface{} `json:"fields" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	record, err := h.provider.CreateRecord(c.Request.Context(), userID, baseID, tableID, req.Fields)
	if err != nil {
		slog.Info("Failed to create record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// UpdateRecord updates an existing record.
func (h *Handler) UpdateRecord(c *gin.Context) {
	userID := c.GetString("user_id")
	recordID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID and Table ID are required
	baseID := c.Query("base_id")
	tableID := c.Query("table_id")

	if baseID == "" || tableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id or table_id query parameters"})
		return
	}

	var req struct {
		Fields map[string]interface{} `json:"fields" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	record, err := h.provider.UpdateRecord(c.Request.Context(), userID, baseID, tableID, recordID, req.Fields)
	if err != nil {
		slog.Info("Failed to update record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteRecord deletes a record from a table.
func (h *Handler) DeleteRecord(c *gin.Context) {
	userID := c.GetString("user_id")
	recordID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID and Table ID are required
	baseID := c.Query("base_id")
	tableID := c.Query("table_id")

	if baseID == "" || tableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id or table_id query parameters"})
		return
	}

	if err := h.provider.DeleteRecord(c.Request.Context(), userID, baseID, tableID, recordID); err != nil {
		slog.Info("Failed to delete record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncRecords syncs records for a table from Airtable.
func (h *Handler) SyncRecords(c *gin.Context) {
	userID := c.GetString("user_id")
	tableID := c.Param("table_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Base ID is required
	baseID := c.Query("base_id")
	if baseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing base_id query parameter"})
		return
	}

	// Get all records (with pagination support)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	synced, err := h.provider.SyncRecords(c.Request.Context(), userID, baseID, tableID, limit)
	if err != nil {
		slog.Info("Failed to sync records", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"synced":  synced,
	})
}

