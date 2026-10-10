package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
)

func TestOptimalRoutesRequireAuthInProduction(t *testing.T) {
	t.Setenv("OPTIMAL_ENGINE_URL", "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api")
	h := &Handlers{cfg: &config.Config{Environment: "production"}}
	h.registerOptimalRoutes(api, func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "test auth denied"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/optimal/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOptimalRoutesStayOpenInDevelopment(t *testing.T) {
	t.Setenv("OPTIMAL_ENGINE_URL", "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api")
	h := &Handlers{cfg: &config.Config{Environment: "development"}}
	h.registerOptimalRoutes(api, func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "test auth denied"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/optimal/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 disabled-engine response, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOptimalProxyTranslatesPostSearchToEngineGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotMethod, gotPath, gotQ, gotWorkspace, gotLimit string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQ = r.URL.Query().Get("q")
		gotWorkspace = r.URL.Query().Get("workspace")
		gotLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer upstream.Close()

	proxy, err := NewOptimalProxy(upstream.URL, nil)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	router := gin.New()
	api := router.Group("/api/v1")
	RegisterOptimalProxyRoutes(api, proxy)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/optimal/search",
		strings.NewReader(`{"query":"platform","workspace":"agency-miosa","limit":2}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("expected upstream GET, got %s", gotMethod)
	}
	if gotPath != "/api/search" {
		t.Fatalf("expected upstream /api/search, got %s", gotPath)
	}
	if gotQ != "platform" || gotWorkspace != "agency-miosa" || gotLimit != "2" {
		t.Fatalf("unexpected upstream query q=%q workspace=%q limit=%q", gotQ, gotWorkspace, gotLimit)
	}
}

func TestOptimalProxyPostSearchUsesResolvedWorkspaceOverBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotWorkspace string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotWorkspace = r.URL.Query().Get("workspace")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer upstream.Close()

	proxy, err := NewOptimalProxy(upstream.URL, nil)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/optimal/search",
		strings.NewReader(`{"query":"platform","workspace":"wrong-workspace"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), proxyTargetKey{}, &proxyTarget{
		url:       upstreamURL,
		workspace: "agency-miosa",
	}))
	rec := httptest.NewRecorder()

	proxy.rp.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotWorkspace != "agency-miosa" {
		t.Fatalf("expected resolved workspace agency-miosa, got %q", gotWorkspace)
	}
}
