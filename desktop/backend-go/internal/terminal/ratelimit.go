package terminal

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"github.com/rhl/businessos-backend/internal/logging"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Messages per second allowed (token bucket refill rate)
	MessagesPerSecond float64

	// Burst size - max messages allowed in a burst
	BurstSize int

	// Max message size in bytes (0 = no limit)
	MaxMessageSize int64

	// Max connections per user (0 = no limit)
	MaxConnectionsPerUser int

	// How long to keep rate limiter in memory after last activity
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns production-safe defaults
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MessagesPerSecond:     1000,   // 1000 messages per second (increased for development)
		BurstSize:             200,    // Allow burst of 200 messages (increased for terminal responsiveness)
		MaxMessageSize:        16384,  // 16KB max message size
		MaxConnectionsPerUser: 5,      // Max 5 concurrent connections per user
		CleanupInterval:       5 * time.Minute,
	}
}

// RateLimiter manages per-user and per-connection rate limiting
type RateLimiter struct {
	config          *RateLimitConfig
	mu              sync.RWMutex
	userLimiters    map[string]*rate.Limiter     // Per-user message rate
	userConnections map[string]int               // Connection count per user
	lastActivity    map[string]time.Time         // For cleanup
	stopCleanup     chan struct{}
}

// Global rate limiter instance
var (
	globalRateLimiter     *RateLimiter
	globalRateLimiterOnce sync.Once
)

// GetRateLimiter returns the global rate limiter instance
func GetRateLimiter() *RateLimiter {
	globalRateLimiterOnce.Do(func() {
		globalRateLimiter = NewRateLimiter(DefaultRateLimitConfig())
	})
	return globalRateLimiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:          config,
		userLimiters:    make(map[string]*rate.Limiter),
		userConnections: make(map[string]int),
		lastActivity:    make(map[string]time.Time),
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// getLimiter returns the rate limiter for a user, creating if needed
func (rl *RateLimiter) getLimiter(userID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.userLimiters[userID]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Limit(rl.config.MessagesPerSecond),
			rl.config.BurstSize,
		)
		rl.userLimiters[userID] = limiter
	}

	rl.lastActivity[userID] = time.Now()
	return limiter
}

// AllowMessage checks if a message is allowed under rate limiting
func (rl *RateLimiter) AllowMessage(userID string) bool {
	limiter := rl.getLimiter(userID)
	return limiter.Allow()
}

// WaitForMessage blocks until a message is allowed or context expires
func (rl *RateLimiter) Reserve(userID string) *rate.Reservation {
	limiter := rl.getLimiter(userID)
	return limiter.Reserve()
}

// CheckMessageSize validates message size
func (rl *RateLimiter) CheckMessageSize(size int64) bool {
	rl.mu.RLock()
	maxSize := rl.config.MaxMessageSize
	rl.mu.RUnlock()

	if maxSize <= 0 {
		return true
	}
	return size <= maxSize
}

// AddConnection increments connection count for a user
// Returns true if connection is allowed, false if limit exceeded
func (rl *RateLimiter) AddConnection(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.config.MaxConnectionsPerUser <= 0 {
		return true // No limit
	}

	current := rl.userConnections[userID]
	if current >= rl.config.MaxConnectionsPerUser {
		logging.Security("Rate limit: max connections exceeded for user %s (current: %d, max: %d)",
			logging.MaskSessionID(userID), current, rl.config.MaxConnectionsPerUser)
		return false
	}

	rl.userConnections[userID] = current + 1
	rl.lastActivity[userID] = time.Now()
	logging.Debug("Connection added for user %s (total: %d)", logging.MaskSessionID(userID), current+1)
	return true
}

// RemoveConnection decrements connection count for a user
func (rl *RateLimiter) RemoveConnection(userID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	current := rl.userConnections[userID]
	if current > 0 {
		rl.userConnections[userID] = current - 1
		logging.Debug("Connection removed for user %s (remaining: %d)", logging.MaskSessionID(userID), current-1)
	}

	if rl.userConnections[userID] == 0 {
		delete(rl.userConnections, userID)
	}
}

// GetConnectionCount returns current connection count for a user
func (rl *RateLimiter) GetConnectionCount(userID string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.userConnections[userID]
}

// cleanupLoop periodically removes inactive rate limiters to prevent memory leaks
func (rl *RateLimiter) cleanupLoop() {
	// Read config under lock to avoid race condition
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
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.config.CleanupInterval)
	removed := 0

	for userID, lastActive := range rl.lastActivity {
		if lastActive.Before(cutoff) && rl.userConnections[userID] == 0 {
			delete(rl.userLimiters, userID)
			delete(rl.lastActivity, userID)
			removed++
		}
	}

	if removed > 0 {
		logging.Debug("Rate limiter cleanup: removed %d inactive entries", removed)
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// UpdateConfig updates rate limiter configuration
func (rl *RateLimiter) UpdateConfig(config *RateLimitConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.config = config

	// Update existing limiters with new rate
	for _, limiter := range rl.userLimiters {
		limiter.SetLimit(rate.Limit(config.MessagesPerSecond))
		limiter.SetBurst(config.BurstSize)
	}
}

// GetConfig returns a copy of the current configuration (thread-safe)
func (rl *RateLimiter) GetConfig() RateLimitConfig {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return *rl.config
}

// RateLimitMiddleware returns an HTTP middleware that checks connection limits
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by auth middleware)
		userID, ok := r.Context().Value("user").(string)
		if !ok || userID == "" {
			userID = r.RemoteAddr // Fallback to IP
		}

		rl := GetRateLimiter()

		// Check connection limit
		if !rl.AddConnection(userID) {
			logging.Security("Rate limit: connection refused for user %s", logging.MaskSessionID(userID))
			http.Error(w, "Too many connections", http.StatusTooManyRequests)
			return
		}

		// Connection will be removed when websocket closes
		// (handled in WebSocketHandler.HandleConnection)
		next(w, r)
	}
}

// HTTP429Handler sends a proper 429 response
func HTTP429Handler(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1") // Suggest retry after 1 second
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate_limit_exceeded","message":"` + message + `","retry_after":1}`))
}
