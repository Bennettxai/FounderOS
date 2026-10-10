package notion

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	integrations "github.com/rhl/businessos-backend/internal/integrations"
)

// Handler provides HTTP handlers for Notion integration routes.
type Handler struct {
	provider  *Provider
	databases *DatabaseService
}

// NewHandler creates a new Notion integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider:  provider,
		databases: NewDatabaseService(provider),
	}
}

// RegisterRoutes registers all Notion integration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// OAuth routes
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Database routes
	databases := r.Group("/databases")
	{
		databases.GET("", h.GetDatabases)
		databases.POST("/sync", h.SyncDatabases)
	}

	// Page routes
	pages := r.Group("/pages")
	{
		pages.GET("/:database_id", h.GetPages)
		pages.POST("/:database_id/sync", h.SyncPages)
	}
}

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
		"success":        true,
		"workspace_name": token.AccountName,
	})
}

// Disconnect disconnects the Notion integration.
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

// GetDatabases returns the user's Notion databases.
func (h *Handler) GetDatabases(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	databases, err := h.databases.GetDatabases(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get databases", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get databases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"databases": databases,
		"count":     len(databases),
	})
}

// SyncDatabases syncs databases from Notion.
func (h *Handler) SyncDatabases(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.databases.SyncDatabases(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to sync databases", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync databases"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPages returns pages for a database.
func (h *Handler) GetPages(c *gin.Context) {
	userID := c.GetString("user_id")
	databaseID := c.Param("database_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	pages, err := h.databases.GetPages(c.Request.Context(), userID, databaseID, limit, offset)
	if err != nil {
		slog.Info("Failed to get pages", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pages": pages,
		"count": len(pages),
	})
}

// SyncPages syncs pages for a database from Notion.
func (h *Handler) SyncPages(c *gin.Context) {
	userID := c.GetString("user_id")
	databaseID := c.Param("database_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.databases.SyncPages(c.Request.Context(), userID, databaseID)
	if err != nil {
		slog.Info("Failed to sync pages", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync pages"})
		return
	}

	c.JSON(http.StatusOK, result)
}
