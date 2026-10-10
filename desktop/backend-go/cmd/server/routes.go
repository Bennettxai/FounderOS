package main

import (
	founderosapi "github.com/rhl/businessos-backend/internal/founderos/api"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/handlers"
	"github.com/rhl/businessos-backend/internal/integrations/miosa"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/middleware"
	redisClient "github.com/rhl/businessos-backend/internal/redis"
)

// httpMaxBytesReader is a thin wrapper around http.MaxBytesReader so that
// bootstrap.go does not need to import "net/http" for a single call.
func httpMaxBytesReader(w http.ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser {
	return http.MaxBytesReader(w, r, n)
}

// buildCSRFConfig constructs the CSRF middleware config, applying development
// overrides when the server is not in production mode.
func buildCSRFConfig(cfg *config.Config) middleware.CSRFConfig {
	csrfConfig := middleware.DefaultCSRFConfig()
	if cfg.IsProduction() {
		// The packaged renderer is app://localhost and calls the cloud API
		// cross-site. Double-submit CSRF still requires the matching custom header,
		// while SameSite=None allows Electron to send the Secure cookie itself.
		csrfConfig.CookieSameSite = http.SameSiteNoneMode
	} else {
		csrfConfig.CookieSecure = false
		csrfConfig.CookieDomain = os.Getenv("COOKIE_DOMAIN")
	}
	csrfConfig.Skipper = func(c *gin.Context) bool {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/webhooks/") ||
			strings.HasPrefix(path, "/api/webhooks/") ||
			strings.HasPrefix(path, "/api/v1/webhooks/") {
			return true
		}
		if strings.HasPrefix(path, "/api/osa/webhooks/") ||
			strings.HasPrefix(path, "/api/v1/osa/webhooks/") {
			return true
		}
		if path == "/api/sorx/callback" || path == "/api/v1/sorx/callback" {
			return true
		}
		// The operator machine routes authenticate with bearer tokens.
		if founderosapi.CSRFExempt(path) {
			return true
		}
		if strings.HasPrefix(path, "/api/optimal/") ||
			strings.HasPrefix(path, "/api/v1/optimal/") {
			return !cfg.IsProduction() && !cfg.IsCloudDeployment()
		}
		if strings.HasPrefix(path, "/api/internal/osa/") ||
			strings.HasPrefix(path, "/api/v1/internal/osa/") {
			return true
		}
		if path == "/health" || path == "/ready" || path == "/health/detailed" ||
			path == "/healthz" || path == "/readyz" {
			return true
		}
		if path == "/api/osa/health" || path == "/api/v1/osa/health" {
			return true
		}
		return false
	}
	return csrfConfig
}

// registerRoutes attaches all routes to app.router.
// It is called at the end of bootstrap() after all services are initialized.
func registerRoutes(app *AppServices, skillsHandler *handlers.SkillsHandler, osaClient *osa.ResilientClient) {
	router := app.router
	cfg := app.cfg

	// ── Health endpoints (no auth) ────────────────────────────────────────────
	deps := healthDeps{
		instanceID:     app.instanceID,
		dbConnected:    app.dbConnected,
		dbErr:          app.dbErr,
		redisConnected: app.redisConnected,
	}
	// Pass containerMgr as containerManagerInterface (nil-safe interface check).
	if app.containerMgr != nil {
		deps.containerMgr = app.containerMgr
	}

	router.GET("/", newRootHandler(app.instanceID))
	router.GET("/health", newHealthHandler(cfg, app.startTime))
	router.GET("/ready", newReadinessHandler(deps, cfg.DatabaseRequired))
	router.GET("/health/detailed", newDetailedHealthHandler(deps, cfg.DatabaseRequired))

	// ── Kubernetes-standard probes ────────────────────────────────────────────
	// /healthz  — liveness probe  (is the process alive?)
	// /readyz   — readiness probe (can the process serve traffic?)
	var redisPinger handlers.RedisPinger
	if app.redisConnected {
		redisPinger = handlers.NewRedisPinger(redisClient.IsConnected)
	}
	healthHandler := handlers.NewHealthHandler(app.pool, redisPinger)
	healthHandler.RegisterRoutes(router)

	// Static file serving — profile photos, backgrounds, etc.
	router.Static("/uploads", "./uploads")

	// ── API versioning ─────────────────────────────────────────────────────────
	versioningConfig := middleware.DefaultVersioningConfig()
	v1Config := versioningConfig.Versions["v1"]

	apiv1 := router.Group("/api/v1")
	apiv1.Use(middleware.DeprecationHeaders(v1Config))

	// Backward-compat: /api/* → /api/v1/* (deprecated)
	api := router.Group("/api")
	api.Use(middleware.VersionRedirect("v1", false))

	// ── Degraded mode — only status endpoint ──────────────────────────────────
	if !app.dbConnected || app.pool == nil {
		api.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":            "degraded",
				"database":          "unavailable",
				"database_required": cfg.DatabaseRequired,
			})
		})
		return
	}

	// ── Main handler routes ────────────────────────────────────────────────────
	app.handlers.RegisterRoutes(api)   // Deprecated path with warning headers
	app.handlers.RegisterRoutes(apiv1) // Current versioned path

	auth := app.handlers.Auth()

	// ── Skills routes ──────────────────────────────────────────────────────────
	if skillsHandler != nil {
		skillsHandler.RegisterRoutes(api)
		skillsHandler.RegisterRoutes(apiv1)
	}

	// ── Background jobs routes ─────────────────────────────────────────────────
	if app.jobsHandler != nil {
		app.jobsHandler.RegisterRoutes(api)
		app.jobsHandler.RegisterRoutes(apiv1)
	}

	// ── Computer + Billing endpoints ─────────────────────────────────────────
	var miosaComputeClient *miosa.ComputeClient
	if cfg.MIOSAAPIKey != "" && cfg.MIOSAAPIUrl != "" {
		miosaComputeClient = miosa.NewComputeClient(cfg.MIOSAAPIUrl, cfg.MIOSAAPIKey)
	}
	computerHandler := handlers.NewComputerHandler(cfg, miosaComputeClient)
	billingHandler := handlers.NewBillingHandler(cfg, miosaComputeClient)
	registerComputerRoutes(api, auth, computerHandler, billingHandler)
	registerComputerRoutes(apiv1, auth, computerHandler, billingHandler)

	miosaHandler := handlers.NewMIOSAHandler(app.pool, cfg)
	handlers.RegisterMIOSARoutes(api, miosaHandler, auth)
	handlers.RegisterMIOSARoutes(apiv1, miosaHandler, auth)

	platformAdminHandler := handlers.NewPlatformAdminHandler(app.pool, cfg, miosaComputeClient)
	handlers.RegisterPlatformAdminRoutes(api, platformAdminHandler, auth)
	handlers.RegisterPlatformAdminRoutes(apiv1, platformAdminHandler, auth)

	platformAdminControlHandler := handlers.NewPlatformAdminControlHandler(app.pool)
	handlers.RegisterPlatformAdminControlRoutes(api, platformAdminControlHandler, auth)
	handlers.RegisterPlatformAdminControlRoutes(apiv1, platformAdminControlHandler, auth)

	// ── Cloud sync endpoints (push/pull/status) ───────────────────────────────
	cloudSyncHandler := handlers.NewCloudSyncHandler(app.pool)
	registerSyncRoutes(apiv1, auth, cloudSyncHandler)

	// ── OSA health/config endpoints ───────────────────────────────────────────
	osaHealthH := handlers.NewOSAAPIHandler(osaClient, app.pool)
	router.GET("/api/osa/health", osaHealthH.HandleOSAHealth)
	router.GET("/api/v1/osa/health", osaHealthH.HandleOSAHealth)
	router.GET("/api/osa/modes", osaHealthH.HandleOSAModes)
	router.GET("/api/v1/osa/modes", osaHealthH.HandleOSAModes)
	router.POST("/api/osa/config", auth, middleware.RequireAuth(), osaHealthH.HandleOSAConfig)
	router.POST("/api/v1/osa/config", auth, middleware.RequireAuth(), osaHealthH.HandleOSAConfig)
}
