package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RedisPinger is the minimal interface required by the readiness handler.
// It is satisfied by the package-level IsConnected function in internal/redis.
type RedisPinger interface {
	Ping(ctx context.Context) bool
}

// redisPingerFunc adapts a plain function to the RedisPinger interface so
// callers can pass the package-level IsConnected without an extra wrapper struct.
type redisPingerFunc func(ctx context.Context) bool

func (f redisPingerFunc) Ping(ctx context.Context) bool { return f(ctx) }

// NewRedisPinger wraps any func(context.Context) bool as a RedisPinger.
// Pass redis.IsConnected directly:
//
//	NewRedisPinger(redisClient.IsConnected)
func NewRedisPinger(fn func(ctx context.Context) bool) RedisPinger {
	return redisPingerFunc(fn)
}

// HealthHandler handles the Kubernetes-standard liveness and readiness probes.
// It is separate from the legacy /health and /ready handlers that live in
// cmd/server/health.go; those remain untouched for backward compatibility.
type HealthHandler struct {
	pool  *pgxpool.Pool // may be nil during degraded-mode boot
	redis RedisPinger   // may be nil when Redis is not configured
}

// NewHealthHandler constructs a HealthHandler.
//   - pool  — pass nil when the database is not yet available (degraded mode).
//   - redis — pass nil when Redis is not configured; NewRedisPinger() otherwise.
func NewHealthHandler(pool *pgxpool.Pool, redis RedisPinger) *HealthHandler {
	return &HealthHandler{pool: pool, redis: redis}
}

// RegisterRoutes attaches /healthz and /readyz to the provided router group.
// Both endpoints are intentionally unauthenticated.
func (h *HealthHandler) RegisterRoutes(r gin.IRouter) {
	r.GET("/healthz", h.Liveness)
	r.GET("/readyz", h.Readiness)
}

// Liveness handles GET /healthz.
// A liveness probe only answers "is the process alive?". It always returns 200
// as long as the HTTP server is running — it does not check dependencies.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// checkResult carries the outcome of a single readiness sub-check.
type checkResult struct {
	name   string
	status string // "ok" or "error"
	err    error
}

// Readiness handles GET /readyz.
// Returns 200 when all required dependencies are healthy, 503 otherwise.
// Response shape:
//
//	{"status":"ready","checks":{"database":"ok","redis":"ok"}}
//	{"status":"not_ready","checks":{"database":"error","redis":"ok"}}
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allOK := true

	// Database check.
	dbResult := h.checkDatabase(ctx)
	checks[dbResult.name] = dbResult.status
	if dbResult.err != nil {
		allOK = false
		slog.WarnContext(ctx, "healthz: database readiness check failed", "error", dbResult.err)
	}

	// Redis check (optional — only present when a pinger is configured).
	if h.redis != nil {
		redisResult := h.checkRedis(ctx)
		checks[redisResult.name] = redisResult.status
		if redisResult.err != nil {
			allOK = false
			slog.WarnContext(ctx, "healthz: redis readiness check failed", "error", redisResult.err)
		}
	}

	if allOK {
		c.JSON(http.StatusOK, gin.H{"status": "ready", "checks": checks})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "checks": checks})
}

func (h *HealthHandler) checkDatabase(ctx context.Context) checkResult {
	if h.pool == nil {
		return checkResult{name: "database", status: "error", err: errNilPool}
	}
	if err := h.pool.Ping(ctx); err != nil {
		return checkResult{name: "database", status: "error", err: err}
	}
	return checkResult{name: "database", status: "ok"}
}

func (h *HealthHandler) checkRedis(ctx context.Context) checkResult {
	if !h.redis.Ping(ctx) {
		return checkResult{name: "redis", status: "error", err: errRedisPingFailed}
	}
	return checkResult{name: "redis", status: "ok"}
}

// Sentinel errors used internally; never exposed in HTTP responses.
var (
	errNilPool         = &probeError{"database pool is nil"}
	errRedisPingFailed = &probeError{"redis ping failed"}
)

type probeError struct{ msg string }

func (e *probeError) Error() string { return e.msg }
