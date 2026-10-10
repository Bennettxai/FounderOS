// Package middleware provides HTTP rate limiting for DoS attack prevention
package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds HTTP rate limiting configuration
type RateLimiterConfig struct {
	// Requests per second allowed per IP (token bucket refill rate)
	RequestsPerSecond float64

	// Burst size - max requests allowed in a burst per IP
	BurstSize int

	// Requests per second allowed per authenticated user
	UserRequestsPerSecond float64

	// Burst size for authenticated users
	UserBurstSize int

	// How long to keep IP rate limiter in memory after last activity
	CleanupInterval time.Duration

	// Paths to exclude from rate limiting (health checks, static assets, etc.)
	ExcludePaths []string
}

// DefaultRateLimiterConfig returns production-safe defaults
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond:     20,   // 20 requests/sec per IP
		BurstSize:             40,   // Allow burst of 40 requests
		UserRequestsPerSecond: 1000, // Authenticated users get higher limit
		UserBurstSize:         200,  // Larger burst for authenticated users
		CleanupInterval:       10 * time.Minute,
		ExcludePaths: []string{
			"/health",
			"/api/health",
			"/metrics",
		},
	}
}

// StrictRateLimiterConfig returns strict limits for sensitive endpoints
func StrictRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond:     10, // 10 requests/sec per IP
		BurstSize:             3,  // Small burst
		UserRequestsPerSecond: 20, // 20 requests/sec for auth users
		UserBurstSize:         5,  // Small burst for auth users
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
}

// HTTPRateLimiter manages per-IP and per-user rate limiting for HTTP requests
type HTTPRateLimiter struct {
	config       *RateLimiterConfig
	mu           sync.RWMutex
	ipLimiters   map[string]*rate.Limiter // Per-IP rate limiters
	userLimiters map[string]*rate.Limiter // Per-user rate limiters
	lastActivity map[string]time.Time     // Track last activity for cleanup
	stopCleanup  chan struct{}
}

// NewHTTPRateLimiter creates a new HTTP rate limiter
func NewHTTPRateLimiter(config *RateLimiterConfig) *HTTPRateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &HTTPRateLimiter{
		config:       config,
		ipLimiters:   make(map[string]*rate.Limiter),
		userLimiters: make(map[string]*rate.Limiter),
		lastActivity: make(map[string]time.Time),
		stopCleanup:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// getIPLimiter returns the rate limiter for an IP address, creating if needed
func (rl *HTTPRateLimiter) getIPLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.ipLimiters[ip]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Limit(rl.config.RequestsPerSecond),
			rl.config.BurstSize,
		)
		rl.ipLimiters[ip] = limiter
	}

	rl.lastActivity["ip:"+ip] = time.Now()
	return limiter
}

// getUserLimiter returns the rate limiter for a user, creating if needed
func (rl *HTTPRateLimiter) getUserLimiter(userID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.userLimiters[userID]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Limit(rl.config.UserRequestsPerSecond),
			rl.config.UserBurstSize,
		)
		rl.userLimiters[userID] = limiter
	}

	rl.lastActivity["user:"+userID] = time.Now()
	return limiter
}

// Allow checks if a request should be allowed
// Returns true if allowed, false if rate limit exceeded
func (rl *HTTPRateLimiter) Allow(ip string, userID string) bool {
	// Check user rate limit first (if authenticated)
	if userID != "" {
		userLimiter := rl.getUserLimiter(userID)
		if !userLimiter.Allow() {
			return false
		}
	}

	// Always check IP rate limit (defense in depth)
	ipLimiter := rl.getIPLimiter(ip)
	return ipLimiter.Allow()
}

// Reserve reserves a token for the request and returns a Reservation
func (rl *HTTPRateLimiter) Reserve(ip string, userID string) *rate.Reservation {
	// For authenticated users, use user limiter
	if userID != "" {
		userLimiter := rl.getUserLimiter(userID)
		return userLimiter.Reserve()
	}

	// For unauthenticated requests, use IP limiter
	ipLimiter := rl.getIPLimiter(ip)
	return ipLimiter.Reserve()
}

// cleanupLoop periodically removes inactive rate limiters to prevent memory leaks
func (rl *HTTPRateLimiter) cleanupLoop() {
	// Read cleanup interval with lock to avoid race
	rl.mu.RLock()
	interval := rl.config.CleanupInterval
	rl.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes inactive rate limiters
func (rl *HTTPRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.config.CleanupInterval)
	removedIPs := 0
	removedUsers := 0

	// Clean up IP limiters
	for key, lastActive := range rl.lastActivity {
		if lastActive.Before(cutoff) {
			if ip := extractIPFromKey(key); ip != "" {
				delete(rl.ipLimiters, ip)
				delete(rl.lastActivity, key)
				removedIPs++
			} else if userID := extractUserIDFromKey(key); userID != "" {
				delete(rl.userLimiters, userID)
				delete(rl.lastActivity, key)
				removedUsers++
			}
		}
	}

	if removedIPs > 0 || removedUsers > 0 {
		// Note: Using standard log to avoid circular dependency
		// In production, replace with your logging package
		_ = removedIPs + removedUsers // Avoid unused variable warning
	}
}

// extractIPFromKey extracts IP from activity key
func extractIPFromKey(key string) string {
	if len(key) > 3 && key[:3] == "ip:" {
		return key[3:]
	}
	return ""
}

// extractUserIDFromKey extracts user ID from activity key
func extractUserIDFromKey(key string) string {
	if len(key) > 5 && key[:5] == "user:" {
		return key[5:]
	}
	return ""
}

// Stop stops the cleanup goroutine
func (rl *HTTPRateLimiter) Stop() {
	close(rl.stopCleanup)
}

// UpdateConfig updates the rate limiter configuration
func (rl *HTTPRateLimiter) UpdateConfig(config *RateLimiterConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.config = config

	// Update existing IP limiters
	for _, limiter := range rl.ipLimiters {
		limiter.SetLimit(rate.Limit(config.RequestsPerSecond))
		limiter.SetBurst(config.BurstSize)
	}

	// Update existing user limiters
	for _, limiter := range rl.userLimiters {
		limiter.SetLimit(rate.Limit(config.UserRequestsPerSecond))
		limiter.SetBurst(config.UserBurstSize)
	}
}

// GetConfig returns a copy of the current configuration
func (rl *HTTPRateLimiter) GetConfig() RateLimiterConfig {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return *rl.config
}

// isExcludedPath checks if the request path should be excluded from rate limiting
func (rl *HTTPRateLimiter) isExcludedPath(path string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for _, excludedPath := range rl.config.ExcludePaths {
		if path == excludedPath {
			return true
		}
	}
	return false
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting
func RateLimitMiddleware(rl *HTTPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path is excluded
		if rl.isExcludedPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract client IP
		clientIP := getClientIP(c.Request)

		// Extract user ID if authenticated (set by auth middleware)
		var userID string
		if user, exists := c.Get(UserContextKey); exists {
			if authUser, ok := user.(*BetterAuthUser); ok {
				userID = authUser.ID
			}
		}

		// Check rate limit
		if !rl.Allow(clientIP, userID) {
			// Add rate limit headers
			c.Header("X-RateLimit-Limit", formatLimit(rl.config.RequestsPerSecond))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", formatResetTime(time.Now().Add(time.Second)))
			c.Header("Retry-After", "1")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please slow down.",
				"retry_after": 1,
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", formatLimit(rl.config.RequestsPerSecond))

		c.Next()
	}
}

// StrictRateLimitMiddleware returns a Gin middleware with strict rate limiting for sensitive endpoints
func StrictRateLimitMiddleware() gin.HandlerFunc {
	strictLimiter := NewHTTPRateLimiter(StrictRateLimiterConfig())
	return RateLimitMiddleware(strictLimiter)
}

// CredentialRateLimiterConfig returns highly restrictive limits for credential operations.
// Enforces a maximum of 10 requests per minute per authenticated user to prevent
// brute-force credential harvesting (OWASP A07: Authentication Failures).
func CredentialRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		// 10 requests per minute per IP (unauthenticated fallback) = 10/60 req/sec
		RequestsPerSecond: float64(10) / 60,
		BurstSize:         2, // Allow a tiny burst for near-simultaneous requests
		// 10 requests per minute per authenticated user
		UserRequestsPerSecond: float64(10) / 60,
		UserBurstSize:         3, // Small burst for authenticated users
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
}

// CredentialRateLimitMiddleware returns a Gin middleware with tightly restricted rate
// limiting for credential-ticket endpoints (10 req/min per user, burst 3).
func CredentialRateLimitMiddleware() gin.HandlerFunc {
	credLimiter := NewHTTPRateLimiter(CredentialRateLimiterConfig())
	return RateLimitMiddleware(credLimiter)
}

// getClientIP extracts the real client IP from the request.
// SECURITY (HIGH-1): Only trusts proxy headers (X-Forwarded-For, X-Real-IP) when
// the direct connection comes from a trusted proxy address (loopback / local reverse
// proxy). This prevents rate-limit bypass via IP spoofing from untrusted clients.
// In production behind a cloud load balancer, expand trustedProxyIPs to include
// the load balancer's egress CIDR range.
func getClientIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// Only honour proxy headers when the TCP connection itself arrives from a
	// trusted source (loopback = local reverse proxy such as nginx or Caddy).
	parsedIP := net.ParseIP(remoteIP)
	if parsedIP != nil && (parsedIP.IsLoopback() || remoteIP == "::1") {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For: client, proxy1, proxy2 — take the leftmost entry.
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	return remoteIP
}

// formatLimit formats requests per second as a string
func formatLimit(rps float64) string {
	return fmt.Sprintf("%.0f", rps)
}

// formatResetTime formats reset time as Unix timestamp
func formatResetTime(t time.Time) string {
	return fmt.Sprintf("%d", t.Unix())
}

// Global rate limiter instance
var (
	globalHTTPRateLimiter     *HTTPRateLimiter
	globalHTTPRateLimiterOnce sync.Once
)

// GetGlobalHTTPRateLimiter returns the global HTTP rate limiter instance
func GetGlobalHTTPRateLimiter() *HTTPRateLimiter {
	globalHTTPRateLimiterOnce.Do(func() {
		globalHTTPRateLimiter = NewHTTPRateLimiter(DefaultRateLimiterConfig())
	})
	return globalHTTPRateLimiter
}
