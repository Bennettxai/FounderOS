package hubspot

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	integrations "github.com/rhl/businessos-backend/internal/integrations"
)

// Handler provides HTTP handlers for HubSpot integration routes.
type Handler struct {
	provider *Provider
}

// NewHandler creates a new HubSpot integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
	}
}

// RegisterRoutes registers all HubSpot integration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// OAuth routes
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Contact routes
	contacts := r.Group("/contacts")
	{
		contacts.GET("", h.GetContacts)
		contacts.GET("/:id", h.GetContact)
		contacts.POST("", h.CreateContact)
		contacts.PUT("/:id", h.UpdateContact)
		contacts.POST("/sync", h.SyncContacts)
	}

	// Company routes
	companies := r.Group("/companies")
	{
		companies.GET("", h.GetCompanies)
		companies.GET("/:id", h.GetCompany)
		companies.POST("", h.CreateCompany)
		companies.POST("/sync", h.SyncCompanies)
	}

	// Deal routes
	deals := r.Group("/deals")
	{
		deals.GET("", h.GetDeals)
		deals.GET("/:id", h.GetDeal)
		deals.POST("", h.CreateDeal)
		deals.POST("/sync", h.SyncDeals)
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
		"success":      true,
		"account_id":   token.AccountID,
		"account_name": token.AccountName,
		"scopes":       token.Scopes,
	})
}

// Disconnect disconnects the HubSpot integration.
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
// Contact Handlers
// ============================================================================

// GetContacts returns the user's HubSpot contacts.
func (h *Handler) GetContacts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	contacts, err := h.provider.GetContacts(c.Request.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		slog.Info("Failed to get contacts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contacts": contacts,
		"count":    len(contacts),
	})
}

// GetContact returns a single contact by ID.
func (h *Handler) GetContact(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = c.Param("id") // contactID - not implemented yet

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// For now, return an error - would need to add GetContactByID method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// CreateContact creates a new contact in HubSpot.
func (h *Handler) CreateContact(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Email     string `json:"email" binding:"required"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.provider.CreateContact(c.Request.Context(), userID, req.Email, req.FirstName, req.LastName); err != nil {
		slog.Info("Failed to create contact", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contact"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true})
}

// UpdateContact updates an existing contact in HubSpot.
func (h *Handler) UpdateContact(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = c.Param("id") // contactID - not implemented yet

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// For now, return an error - would need to add UpdateContact method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// SyncContacts syncs contacts from HubSpot.
func (h *Handler) SyncContacts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncContacts(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync contacts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Company Handlers
// ============================================================================

// GetCompanies returns the user's HubSpot companies.
func (h *Handler) GetCompanies(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	companies, err := h.provider.GetCompanies(c.Request.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		slog.Info("Failed to get companies", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
		"count":     len(companies),
	})
}

// GetCompany returns a single company by ID.
func (h *Handler) GetCompany(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = c.Param("id") // companyID - not implemented yet

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// For now, return an error - would need to add GetCompanyByID method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// CreateCompany creates a new company in HubSpot.
func (h *Handler) CreateCompany(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// For now, return an error - would need to add CreateCompany method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// SyncCompanies syncs companies from HubSpot.
func (h *Handler) SyncCompanies(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncCompanies(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync companies", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Deal Handlers
// ============================================================================

// GetDeals returns the user's HubSpot deals.
func (h *Handler) GetDeals(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	deals, err := h.provider.GetDeals(c.Request.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		slog.Info("Failed to get deals", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get deals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deals": deals,
		"count": len(deals),
	})
}

// GetDeal returns a single deal by ID.
func (h *Handler) GetDeal(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = c.Param("id") // dealID - not implemented yet

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// For now, return an error - would need to add GetDealByID method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// CreateDeal creates a new deal in HubSpot.
func (h *Handler) CreateDeal(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		DealName  string  `json:"deal_name" binding:"required"`
		Amount    float64 `json:"amount"`
		Pipeline  string  `json:"pipeline"`
		DealStage string  `json:"deal_stage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// For now, return an error - would need to add CreateDeal method to provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// SyncDeals syncs deals from HubSpot.
func (h *Handler) SyncDeals(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncDeals(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync deals", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync deals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================
