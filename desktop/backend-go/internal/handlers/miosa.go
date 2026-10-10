package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// MIOSAHandler exposes the safe BusinessOS control-plane view of MIOSA.
// It never returns raw API keys to the browser.
type MIOSAHandler struct {
	cfg  *config.Config
	pool *pgxpool.Pool
}

func NewMIOSAHandler(pool *pgxpool.Pool, cfg *config.Config) *MIOSAHandler {
	return &MIOSAHandler{pool: pool, cfg: cfg}
}

func RegisterMIOSARoutes(api *gin.RouterGroup, h *MIOSAHandler, auth gin.HandlerFunc) {
	routes := api.Group("/miosa")
	routes.Use(auth, middleware.RequireAuth())
	{
		routes.GET("/status", h.Status)
		routes.POST("/ping", h.Ping)
		routes.POST("/sync", h.Sync)
		routes.POST("/sandboxes", h.CreateSandbox)
	}
}

func (h *MIOSAHandler) Status(c *gin.Context) {
	tenantAvailable := os.Getenv("BUSINESSOS_MIOSA_TENANT_ENABLED") == "true" || h.hasPlatformMiosaCredential(c)
	userKeyAvailable := h.cfg.MIOSAAPIKey != ""
	apiKeySet := userKeyAvailable || tenantAvailable
	workspaceSandboxEnabled := h.currentWorkspaceSandboxEnabled(c)
	c.JSON(http.StatusOK, gin.H{
		"mode":                        h.miosaMode(),
		"connected":                   apiKeySet && h.miosaMode() == "cloud",
		"api_key_set":                 apiKeySet,
		"capacity_provider":           h.capacityProvider(),
		"businessos_tenant_available": tenantAvailable,
		"businessos_sandbox_enabled":  workspaceSandboxEnabled,
		"user_key_available":          userKeyAvailable,
		"workspace_quota":             h.workspaceQuota(),
		"usage":                       gin.H{"active_sandboxes": 0, "active_computers": 0, "active_desktops": 0},
	})
}

func (h *MIOSAHandler) Ping(c *gin.Context) {
	platform := services.NewMIOSAPlatformService(h.pool, h.cfg)
	key, err := platform.GetTenantKey(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"connected": false, "error": err.Error()})
		return
	}
	if key == nil || key.APIKey == "" {
		c.JSON(http.StatusOK, gin.H{"connected": false, "error": "MIOSA API key is not configured"})
		return
	}

	if _, err := platform.NewClient(key.APIKey).Tenant.Current(c.Request.Context()); err != nil {
		c.JSON(http.StatusOK, gin.H{"connected": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"connected": true})
}

func (h *MIOSAHandler) Sync(c *gin.Context) {
	var req struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.WorkspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "workspace_id is required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"synced_at":   time.Now().UTC(),
		"manifest_id": "pending-miosa-manifest-sync",
	})
}

func (h *MIOSAHandler) miosaMode() string {
	if h.cfg.OSAMode == "cloud" {
		return "cloud"
	}
	return "local"
}

func (h *MIOSAHandler) capacityProvider() string {
	if provider := os.Getenv("BUSINESSOS_MIOSA_CAPACITY_PROVIDER"); provider != "" {
		return provider
	}
	if os.Getenv("BUSINESSOS_MIOSA_TENANT_ENABLED") == "true" || h.hasPlatformMiosaCredential(nil) {
		return "businessos"
	}
	if h.cfg.MIOSAAPIKey != "" {
		return "user"
	}
	return "local"
}

func (h *MIOSAHandler) hasPlatformMiosaCredential(c *gin.Context) bool {
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	key, err := services.NewMIOSAPlatformService(h.pool, h.cfg).GetTenantKey(ctx)
	return err == nil && key != nil && key.APIKey != ""
}

func (h *MIOSAHandler) currentWorkspaceSandboxEnabled(c *gin.Context) bool {
	if c == nil || h.pool == nil {
		return false
	}
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return false
	}
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		return false
	}
	if !h.userCanAccessWorkspace(c, user.ID, workspaceID) {
		return false
	}
	enabled, err := services.NewMIOSAPlatformService(h.pool, h.cfg).IsWorkspaceSandboxEnabled(c.Request.Context(), workspaceID)
	return err == nil && enabled
}

func (h *MIOSAHandler) workspaceQuota() gin.H {
	maxSandboxes := h.cfg.SandboxMaxPerUser
	if maxSandboxes <= 0 {
		maxSandboxes = 5
	}
	return gin.H{
		"max_sandboxes": maxSandboxes,
		"max_computers": 0,
		"max_desktops":  0,
	}
}

func (h *MIOSAHandler) miosaAPIURL() string {
	if h.cfg.MIOSACloudURL != "" {
		return h.cfg.MIOSACloudURL
	}
	if h.cfg.MIOSAAPIUrl != "" {
		return h.cfg.MIOSAAPIUrl
	}
	return "https://api.miosa.ai"
}
