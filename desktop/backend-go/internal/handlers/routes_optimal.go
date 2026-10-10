package handlers

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerOptimalRoutes wires up all OptimalOS data endpoints at
// /api/optimal/*. The runtime is the canonical Elixir engine:
//
//	OPTIMAL_ENGINE_URL set     → Elixir engine via reverse proxy
//	OPTIMAL_ENGINE_URL unset   → explicit 503 for every /api/optimal/* request
//
// Older builds mounted an in-process Go port as a fallback. That made fresh
// clones depend on a sibling checkout and let runtime behavior drift from the
// Elixir engine. The backend now either talks to Elixir or fails loudly with
// setup guidance.
//
// These routes remain unauthenticated only for local desktop/development
// scenarios. Production and cloud deployments require the same session auth as
// the typed engine endpoints.
func (h *Handlers) registerOptimalRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	routes := api
	if h.cfg != nil && (h.cfg.IsProduction() || h.cfg.IsCloudDeployment()) {
		protected := api.Group("")
		protected.Use(auth, middleware.RequireAuth())
		routes = protected
	}

	// Per-workspace proxy: the DB pool lets each request route to its workspace's
	// own configured engine; OPTIMAL_ENGINE_URL (if set) is the global fallback.
	engineURL := os.Getenv("OPTIMAL_ENGINE_URL")
	proxy, err := NewOptimalProxy(engineURL, h.pool)
	if err != nil {
		slog.Error("OptimalOS proxy: init failed", "engine_url", engineURL, "error", err)
		routes.Any("/optimal/*path", func(c *gin.Context) {
			c.JSON(503, gin.H{
				"error": "OptimalEngine is not configured",
				"hint":  "connect an engine in Settings > Optimal Engine, or set OPTIMAL_ENGINE_URL",
			})
		})
		return
	}

	RegisterOptimalProxyRoutes(routes, proxy)
	routes.Any("/optimal-engine/*path", proxy.Forward)
	slog.Info("OptimalEngine proxied per-workspace",
		"routes", "/api/optimal/*, /api/optimal-engine/*",
		"global_fallback", engineURL)
}

// registerOptimalEngineRoutes mounts /api/optimal/{ask,recall/*,memory/*}
// and /api/brief/{morning,evening}. Distinct from registerOptimalRoutes
// (which is the catch-all reverse proxy) — these are typed BO endpoints
// that call the engine's HTTP API via the shared optimalengine.Client
// and inject BusinessOS user context (auth, default workspace, etc.).
func (h *Handlers) registerOptimalEngineRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	handler := h.buildOptimalEngineHandler()
	RegisterOptimalEngineRoutes(api, handler, auth)
	if handler != nil && handler.engine != nil && handler.engine.Enabled() {
		slog.Info("Engine-backed routes registered",
			"routes", "/api/engine/{ask,recall/*,memory}, /api/brief/{morning,evening}")
	} else {
		slog.Info("Engine-backed routes registered (will return 503 — OPTIMAL_ENGINE_URL not set)")
	}
}
