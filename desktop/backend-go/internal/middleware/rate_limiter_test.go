package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewHTTPRateLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	if rl == nil {
		t.Fatal("Expected rate limiter to be created")
	}

	if len(rl.ipLimiters) != 0 {
		t.Errorf("Expected empty ipLimiters map, got %d entries", len(rl.ipLimiters))
	}

	if len(rl.userLimiters) != 0 {
		t.Errorf("Expected empty userLimiters map, got %d entries", len(rl.userLimiters))
	}
}

func TestHTTPRateLimiter_IPBasedLimiting(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     10, // 10 req/sec
		BurstSize:             2,  // burst of 2
		UserRequestsPerSecond: 20,
		UserBurstSize:         4,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.1"

	// First 2 requests should be allowed (burst)
	if !rl.Allow(ip, "") {
		t.Error("Expected first request to be allowed")
	}
	if !rl.Allow(ip, "") {
		t.Error("Expected second request to be allowed")
	}

	// Third request should be rate limited
	if rl.Allow(ip, "") {
		t.Error("Expected third request to be rate limited")
	}

	// Wait for token bucket to refill
	time.Sleep(150 * time.Millisecond) // 1/10 sec = 100ms, add buffer

	// Should be allowed again
	if !rl.Allow(ip, "") {
		t.Error("Expected request to be allowed after waiting")
	}
}

func TestHTTPRateLimiter_UserBasedLimiting(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     100, // High IP limit to not interfere
		BurstSize:             50,  // High IP burst to not interfere
		UserRequestsPerSecond: 20,  // 20 req/sec for user
		UserBurstSize:         5,   // burst of 5 for user
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.1"
	userID := "user123"

	// First 5 requests should be allowed (user burst)
	for i := 0; i < 5; i++ {
		if !rl.Allow(ip, userID) {
			t.Errorf("Expected request %d to be allowed (within user burst)", i+1)
		}
	}

	// Sixth request should be rate limited
	if rl.Allow(ip, userID) {
		t.Error("Expected sixth request to be rate limited")
	}

	// Wait for token bucket to refill
	time.Sleep(100 * time.Millisecond) // 1/20 sec = 50ms, add buffer

	// Should be allowed again
	if !rl.Allow(ip, userID) {
		t.Error("Expected request to be allowed after waiting")
	}
}

func TestHTTPRateLimiter_MultipleIPs(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             2,
		UserRequestsPerSecond: 20,
		UserBurstSize:         4,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Each IP should have its own limit
	if !rl.Allow(ip1, "") {
		t.Error("Expected IP1 first request to be allowed")
	}
	if !rl.Allow(ip1, "") {
		t.Error("Expected IP1 second request to be allowed")
	}

	if !rl.Allow(ip2, "") {
		t.Error("Expected IP2 first request to be allowed")
	}
	if !rl.Allow(ip2, "") {
		t.Error("Expected IP2 second request to be allowed")
	}

	// Third request should be rate limited for both
	if rl.Allow(ip1, "") {
		t.Error("Expected IP1 third request to be rate limited")
	}
	if rl.Allow(ip2, "") {
		t.Error("Expected IP2 third request to be rate limited")
	}
}

func TestHTTPRateLimiter_ConcurrentAccess(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     100,
		BurstSize:             10,
		UserRequestsPerSecond: 200,
		UserBurstSize:         20,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	var wg sync.WaitGroup
	numGoroutines := 50
	requestsPerGoroutine := 10

	// Launch multiple goroutines making concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1.1"
			for j := 0; j < requestsPerGoroutine; j++ {
				rl.Allow(ip, "")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// No crashes = success
}

func TestHTTPRateLimiter_Cleanup(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             2,
		UserRequestsPerSecond: 20,
		UserBurstSize:         4,
		CleanupInterval:       100 * time.Millisecond,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.1"
	userID := "user123"

	// Make some requests
	rl.Allow(ip, "")
	rl.Allow("192.168.1.2", userID)

	// Check that limiters exist (with lock)
	rl.mu.RLock()
	ipCount := len(rl.ipLimiters)
	userCount := len(rl.userLimiters)
	rl.mu.RUnlock()

	if ipCount != 2 {
		t.Errorf("Expected 2 IP limiters, got %d", ipCount)
	}
	if userCount != 1 {
		t.Errorf("Expected 1 user limiter, got %d", userCount)
	}

	// Wait for cleanup to occur
	time.Sleep(200 * time.Millisecond)

	// Limiters should be cleaned up
	rl.mu.RLock()
	ipCount = len(rl.ipLimiters)
	userCount = len(rl.userLimiters)
	rl.mu.RUnlock()

	if ipCount != 0 {
		t.Errorf("Expected 0 IP limiters after cleanup, got %d", ipCount)
	}
	if userCount != 0 {
		t.Errorf("Expected 0 user limiters after cleanup, got %d", userCount)
	}
}

func TestHTTPRateLimiter_UpdateConfig(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             2,
		UserRequestsPerSecond: 20,
		UserBurstSize:         4,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.1"

	// Make a request to create a limiter
	rl.Allow(ip, "")

	// Update config
	newConfig := &RateLimiterConfig{
		RequestsPerSecond:     100,
		BurstSize:             20,
		UserRequestsPerSecond: 200,
		UserBurstSize:         40,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl.UpdateConfig(newConfig)

	// Check that config was updated
	currentConfig := rl.GetConfig()
	if currentConfig.RequestsPerSecond != 100 {
		t.Errorf("Expected RequestsPerSecond to be 100, got %.0f", currentConfig.RequestsPerSecond)
	}
	if currentConfig.BurstSize != 20 {
		t.Errorf("Expected BurstSize to be 20, got %d", currentConfig.BurstSize)
	}
}

func TestRateLimitMiddleware_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             5,
		UserRequestsPerSecond: 20,
		UserBurstSize:         10,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should be allowed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header to be set")
	}
}

func TestRateLimitMiddleware_Block(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             2, // Very small burst
		UserRequestsPerSecond: 20,
		UserBurstSize:         4,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	ip := "192.168.1.1:12345"

	// Exhaust the burst limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	// Check rate limit headers
	if w.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestRateLimitMiddleware_ExcludedPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &RateLimiterConfig{
		RequestsPerSecond:     10,
		BurstSize:             1, // Very small burst
		UserRequestsPerSecond: 20,
		UserBurstSize:         2,
		CleanupInterval:       5 * time.Minute,
		ExcludePaths:          []string{"/health"},
	}
	rl := NewHTTPRateLimiter(config)
	defer rl.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	ip := "192.168.1.1:12345"

	// Exhaust the burst limit on /test
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// /health should still work (excluded from rate limiting)
	req := httptest.NewRequest("GET", "/health", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for excluded path, got %d", w.Code)
	}
}

func TestStrictRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(StrictRateLimitMiddleware())
	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	ip := "192.168.1.1:12345"

	// First few requests should be allowed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// Next request should be rate limited (strict limits)
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 for strict rate limit, got %d", w.Code)
	}
}

func TestGetGlobalHTTPRateLimiter_Singleton(t *testing.T) {
	rl1 := GetGlobalHTTPRateLimiter()
	rl2 := GetGlobalHTTPRateLimiter()

	if rl1 != rl2 {
		t.Error("Expected GetGlobalHTTPRateLimiter to return the same instance")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "127.0.0.1:12345",
			xRealIP:    "203.0.113.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For multiple IPs",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.1, 198.51.100.1, 192.168.1.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:          "Untrusted peer cannot spoof X-Forwarded-For",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
