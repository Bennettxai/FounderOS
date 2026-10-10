package fathom

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for Fathom Analytics integration routes.
type Handler struct {
	provider *Provider
}

// NewHandler creates a new Fathom integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
	}
}

// RegisterRoutes registers all Fathom integration routes.
// Note: Fathom uses API key authentication, NOT OAuth.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Authentication routes (API key based)
	r.POST("/connect", h.Connect)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Site routes
	sites := r.Group("/sites")
	{
		sites.GET("", h.GetSites)
		sites.GET("/:site_id", h.GetSite)
		sites.POST("/sync", h.SyncSites)
	}

	// Aggregations routes
	r.GET("/sites/:site_id/aggregations", h.GetAggregations)

	// Current visitors routes
	r.GET("/sites/:site_id/current-visitors", h.GetCurrentVisitors)

	// Events routes
	r.GET("/sites/:site_id/events", h.GetEvents)
}

// Connect handles API key connection.
// Accepts API key and validates it by making a test API call.
func (h *Handler) Connect(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required"})
		return
	}

	// SaveAPIKey validates the API key and stores it
	if err := h.provider.SaveAPIKey(c.Request.Context(), userID, req.APIKey); err != nil {
		slog.Info("Failed to save API key", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid API key or unable to connect",
		})
		return
	}

	// Get connection status to return details
	status, err := h.provider.GetConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get status after connect", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Connected but failed to retrieve status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"connected": status.Connected,
		"message":   "Successfully connected to Fathom Analytics",
	})
}

// Disconnect disconnects the Fathom integration.
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

// GetSites returns the user's Fathom sites from local database.
func (h *Handler) GetSites(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sites, err := h.provider.GetSites(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get sites", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sites": sites,
		"count": len(sites),
	})
}

// GetSite returns a specific site by ID.
func (h *Handler) GetSite(c *gin.Context) {
	userID := c.GetString("user_id")
	siteID := c.Param("site_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sites, err := h.provider.GetSites(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get sites", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get site"})
		return
	}

	// Find the specific site
	for _, site := range sites {
		if site.ID == siteID {
			c.JSON(http.StatusOK, site)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Site not found"})
}

// SyncSites syncs sites from Fathom API to local database.
func (h *Handler) SyncSites(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get API key
	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to Fathom"})
		return
	}

	// Sync sites
	stats, err := h.provider.syncSites(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync sites", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": stats.Created,
		"updated": stats.Updated,
	})
}

// GetAggregations returns aggregated analytics data for a site.
// Query parameters:
// - entity: "pageview" or "event" (required)
// - aggregates: comma-separated list (e.g., "visits,uniques,pageviews")
// - date_from: YYYY-MM-DD (optional)
// - date_to: YYYY-MM-DD (optional)
// - date_grouping: "day", "month", "year" (optional)
// - field_grouping: "pathname", "hostname", "referrer", etc. (optional)
// - sort_by: field to sort by (optional)
// - limit: max results (optional)
func (h *Handler) GetAggregations(c *gin.Context) {
	userID := c.GetString("user_id")
	siteID := c.Param("site_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse query parameters
	entity := c.DefaultQuery("entity", "pageview")
	aggregates := c.DefaultQuery("aggregates", "visits,uniques,pageviews")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	dateGrouping := c.Query("date_grouping")
	fieldGrouping := c.Query("field_grouping")
	sortBy := c.Query("sort_by")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	input := AggregationsInput{
		Entity:        entity,
		EntityID:      siteID,
		Aggregates:    aggregates,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		DateGrouping:  dateGrouping,
		FieldGrouping: fieldGrouping,
		SortBy:        sortBy,
		Limit:         limit,
	}

	aggregations, err := h.provider.GetAggregations(c.Request.Context(), userID, input)
	if err != nil {
		slog.Info("Failed to get aggregations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get aggregations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"aggregations": aggregations,
		"count":        len(aggregations),
	})
}

// GetCurrentVisitors returns real-time visitor count for a site.
// Query parameters:
// - detailed: "true" to include per-page breakdown (optional)
func (h *Handler) GetCurrentVisitors(c *gin.Context) {
	userID := c.GetString("user_id")
	siteID := c.Param("site_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	detailed := c.Query("detailed") == "true"

	visitors, err := h.provider.GetCurrentVisitors(c.Request.Context(), userID, siteID, detailed)
	if err != nil {
		slog.Info("Failed to get current visitors", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current visitors"})
		return
	}

	c.JSON(http.StatusOK, visitors)
}

// GetEvents returns custom events for a site.
func (h *Handler) GetEvents(c *gin.Context) {
	userID := c.GetString("user_id")
	siteID := c.Param("site_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	events, err := h.provider.GetEvents(c.Request.Context(), userID, siteID)
	if err != nil {
		slog.Info("Failed to get events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}
