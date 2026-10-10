package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- mock Redis piners ----

type alwaysOKRedisPinger struct{}

func (alwaysOKRedisPinger) Ping(_ context.Context) bool { return true }

type alwaysFailRedisPinger struct{}

func (alwaysFailRedisPinger) Ping(_ context.Context) bool { return false }

// ---- helpers ----

// livenessRequest fires GET /healthz against the given handler and returns the recorder.
func livenessRequest(t *testing.T, h *HealthHandler) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// readinessRequest fires GET /readyz against the given handler and returns the recorder.
func readinessRequest(t *testing.T, h *HealthHandler) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- tests ----

func TestHealthHandler_Liveness_AlwaysReturns200(t *testing.T) {
	// pool is nil — liveness must never touch dependencies.
	h := NewHealthHandler(nil, nil)
	w := livenessRequest(t, h)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestHealthHandler_Readiness_NilPool_Returns503(t *testing.T) {
	h := NewHealthHandler(nil, nil)
	w := readinessRequest(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])

	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok, "expected 'checks' to be an object")
	assert.Equal(t, "error", checks["database"])
}

func TestHealthHandler_Readiness_ResponseShape(t *testing.T) {
	// Verify the exact JSON structure: status + checks map.
	h := NewHealthHandler(nil, nil)
	w := readinessRequest(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body.Status)
	_, hasDatabase := body.Checks["database"]
	assert.True(t, hasDatabase, "response must include 'database' key in checks")
}

func TestHealthHandler_Readiness_WithRedisOK_NilPool_Returns503(t *testing.T) {
	// Even if redis is healthy, a nil pool must still cause 503.
	h := NewHealthHandler(nil, alwaysOKRedisPinger{})
	w := readinessRequest(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	checks := body["checks"].(map[string]any)
	assert.Equal(t, "error", checks["database"])
	assert.Equal(t, "ok", checks["redis"])
}

func TestHealthHandler_Readiness_BothFail_Returns503(t *testing.T) {
	// nil pool + redis failure → both reported as error, 503.
	h := NewHealthHandler(nil, alwaysFailRedisPinger{})
	w := readinessRequest(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	checks := body["checks"].(map[string]any)
	assert.Equal(t, "error", checks["database"])
	assert.Equal(t, "error", checks["redis"])
}

func TestHealthHandler_Readiness_NoRedis_ChecksOnlyDatabase(t *testing.T) {
	// When redis pinger is nil, "redis" must be absent from the checks map.
	h := NewHealthHandler(nil, nil)
	w := readinessRequest(t, h)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	checks := body["checks"].(map[string]any)
	_, hasRedis := checks["redis"]
	assert.False(t, hasRedis, "redis key must be absent when pinger is not configured")
}

func TestNewRedisPinger_WrapsFunction(t *testing.T) {
	called := false
	fn := func(_ context.Context) bool {
		called = true
		return true
	}
	pinger := NewRedisPinger(fn)
	result := pinger.Ping(context.Background())
	assert.True(t, result)
	assert.True(t, called)
}
