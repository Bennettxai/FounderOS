package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/handlers"
	"github.com/rhl/businessos-backend/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func denyingAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "test auth denied"})
	}
}

func allowingAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{
			ID:        "test-user",
			Email:     "test@example.com",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
		c.Set("user_id", "test-user")
		c.Next()
	}
}

func TestComputerAndBillingRoutesRequireAuth(t *testing.T) {
	router := gin.New()
	api := router.Group("/api")
	cfg := &config.Config{DeploymentMode: string(config.DeploymentLocal)}
	computerHandler := handlers.NewComputerHandler(cfg, nil)
	billingHandler := handlers.NewBillingHandler(cfg, nil)
	registerComputerRoutes(api, denyingAuth(), computerHandler, billingHandler)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/computer"},
		{method: http.MethodPost, path: "/api/computer"},
		{method: http.MethodGet, path: "/api/computer/metrics"},
		{method: http.MethodGet, path: "/api/computer/terminal-session"},
		{method: http.MethodGet, path: "/api/computer/desktop-stream"},
		{method: http.MethodGet, path: "/api/billing/plans"},
		{method: http.MethodPost, path: "/api/billing/subscribe"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestComputerRouteAllowsAuthenticatedLocalStatus(t *testing.T) {
	router := gin.New()
	api := router.Group("/api")
	cfg := &config.Config{DeploymentMode: string(config.DeploymentLocal)}
	computerHandler := handlers.NewComputerHandler(cfg, nil)
	billingHandler := handlers.NewBillingHandler(cfg, nil)
	registerComputerRoutes(api, allowingAuth(), computerHandler, billingHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/computer", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCloudSyncRoutesRequireAuth(t *testing.T) {
	router := gin.New()
	api := router.Group("/api/v1")
	registerSyncRoutes(api, denyingAuth(), handlers.NewCloudSyncHandler(nil))

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/sync/push"},
		{method: http.MethodGet, path: "/api/v1/sync/pull"},
		{method: http.MethodGet, path: "/api/v1/sync/status"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCSRFSkipperDoesNotExemptProtectedStateChangingRoutes(t *testing.T) {
	cfg := &config.Config{Environment: "production"}
	csrfConfig := buildCSRFConfig(cfg)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/computer"},
		{method: http.MethodDelete, path: "/api/computer"},
		{method: http.MethodPost, path: "/api/computer/runtimes/osa/start"},
		{method: http.MethodPost, path: "/api/billing/subscribe"},
		{method: http.MethodPost, path: "/api/billing/credits/purchase"},
		{method: http.MethodPost, path: "/api/v1/sync/push"},
		{method: http.MethodPost, path: "/api/osa/config"},
		{method: http.MethodPost, path: "/api/optimal/search"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(tc.method, tc.path, nil)

			if csrfConfig.Skipper(ctx) {
				t.Fatalf("expected %s %s to require CSRF", tc.method, tc.path)
			}
		})
	}
}

func TestCSRFSkipperAllowsOptimalOnlyOutsideProduction(t *testing.T) {
	prod := buildCSRFConfig(&config.Config{Environment: "production"})
	dev := buildCSRFConfig(&config.Config{Environment: "development"})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/optimal/search", nil)

	if prod.Skipper(ctx) {
		t.Fatal("expected production optimal POST to require CSRF")
	}
	if !dev.Skipper(ctx) {
		t.Fatal("expected development optimal POST to keep CSRF bypass")
	}
}

func TestProductionCSRFCookieSupportsPackagedDesktop(t *testing.T) {
	cfg := buildCSRFConfig(&config.Config{Environment: "production"})

	if !cfg.CookieSecure {
		t.Fatal("expected production CSRF cookie to remain Secure")
	}
	if cfg.CookieSameSite != http.SameSiteNoneMode {
		t.Fatalf("expected production CSRF cookie SameSite=None, got %v", cfg.CookieSameSite)
	}
}
