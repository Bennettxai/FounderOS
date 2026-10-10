package terminal

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.MessagesPerSecond != 1000 {
		t.Errorf("Expected MessagesPerSecond=1000, got %f", config.MessagesPerSecond)
	}
	if config.BurstSize != 200 {
		t.Errorf("Expected BurstSize=200, got %d", config.BurstSize)
	}
	if config.MaxMessageSize != 16384 {
		t.Errorf("Expected MaxMessageSize=16384, got %d", config.MaxMessageSize)
	}
	if config.MaxConnectionsPerUser != 5 {
		t.Errorf("Expected MaxConnectionsPerUser=5, got %d", config.MaxConnectionsPerUser)
	}
}

func TestRateLimiter_AllowMessage(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     10,  // 10 per second
		BurstSize:             5,   // Allow 5 burst
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 3,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "test-user-1"

	// Should allow burst messages
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.AllowMessage(userID) {
			allowed++
		}
	}

	// Should have allowed at least the burst size
	if allowed < 5 {
		t.Errorf("Expected at least 5 messages allowed (burst), got %d", allowed)
	}

	// After burst, should throttle
	if allowed > 7 {
		t.Errorf("Expected throttling after burst, but allowed %d messages", allowed)
	}
}

func TestRateLimiter_ConnectionLimit(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 3, // Max 3 connections
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "test-user-conn"

	// Should allow up to max connections
	for i := 0; i < 3; i++ {
		if !rl.AddConnection(userID) {
			t.Errorf("Connection %d should be allowed", i+1)
		}
	}

	// Verify connection count
	if count := rl.GetConnectionCount(userID); count != 3 {
		t.Errorf("Expected 3 connections, got %d", count)
	}

	// 4th connection should be denied
	if rl.AddConnection(userID) {
		t.Error("4th connection should be denied")
	}

	// Remove a connection
	rl.RemoveConnection(userID)
	if count := rl.GetConnectionCount(userID); count != 2 {
		t.Errorf("Expected 2 connections after removal, got %d", count)
	}

	// Now a new connection should be allowed
	if !rl.AddConnection(userID) {
		t.Error("Connection should be allowed after removal")
	}
}

func TestRateLimiter_NoConnectionLimit(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 0, // No limit
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "unlimited-user"

	// Should allow unlimited connections
	for i := 0; i < 100; i++ {
		if !rl.AddConnection(userID) {
			t.Errorf("Connection %d should be allowed with no limit", i+1)
		}
	}
}

func TestRateLimiter_CheckMessageSize(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        1024, // 1KB limit
		MaxConnectionsPerUser: 5,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Under limit
	if !rl.CheckMessageSize(500) {
		t.Error("500 bytes should be under limit")
	}

	// At limit
	if !rl.CheckMessageSize(1024) {
		t.Error("1024 bytes should be at limit")
	}

	// Over limit
	if rl.CheckMessageSize(1025) {
		t.Error("1025 bytes should be over limit")
	}
}

func TestRateLimiter_NoSizeLimit(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        0, // No limit
		MaxConnectionsPerUser: 5,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Any size should be allowed
	if !rl.CheckMessageSize(1000000) {
		t.Error("Should allow any size when MaxMessageSize=0")
	}
}

func TestRateLimiter_MultipleUsers(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     10,
		BurstSize:             5,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 2,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Each user should have independent limits
	users := []string{"user1", "user2", "user3"}

	for _, user := range users {
		// Each user gets their own burst
		allowed := 0
		for i := 0; i < 5; i++ {
			if rl.AllowMessage(user) {
				allowed++
			}
		}
		if allowed < 4 {
			t.Errorf("User %s should get at least 4 burst messages, got %d", user, allowed)
		}

		// Each user gets their own connection limit
		if !rl.AddConnection(user) {
			t.Errorf("User %s should be able to add connection", user)
		}
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     1000, // High rate to avoid throttling during test
		BurstSize:             100,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 100,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	var wg sync.WaitGroup
	userID := "concurrent-user"

	// Concurrent message sends
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.AllowMessage(userID)
		}()
	}

	// Concurrent connection adds/removes
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.AddConnection(userID)
			time.Sleep(time.Millisecond)
			rl.RemoveConnection(userID)
		}()
	}

	wg.Wait()
	// Test passes if no race conditions or panics
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 5,
		CleanupInterval:       100 * time.Millisecond, // Fast cleanup for testing
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "cleanup-user"

	// Create a limiter for the user
	rl.AllowMessage(userID)

	// User entry should exist
	rl.mu.RLock()
	_, exists := rl.userLimiters[userID]
	rl.mu.RUnlock()
	if !exists {
		t.Error("User limiter should exist after message")
	}

	// Wait for cleanup cycle + buffer
	time.Sleep(250 * time.Millisecond)

	// User entry should be cleaned up (no connections)
	rl.mu.RLock()
	_, exists = rl.userLimiters[userID]
	rl.mu.RUnlock()
	if exists {
		t.Error("User limiter should be cleaned up after inactivity")
	}
}

func TestRateLimiter_NoCleanupWithActiveConnections(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             20,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 5,
		CleanupInterval:       100 * time.Millisecond,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "active-user"

	// Create limiter and add connection
	rl.AllowMessage(userID)
	rl.AddConnection(userID)

	// Wait for cleanup cycle
	time.Sleep(250 * time.Millisecond)

	// User entry should NOT be cleaned up (has active connection)
	rl.mu.RLock()
	_, exists := rl.userLimiters[userID]
	rl.mu.RUnlock()
	if !exists {
		t.Error("User limiter should NOT be cleaned up with active connection")
	}
}

func TestRateLimiter_UpdateConfig(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     10,
		BurstSize:             5,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 3,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Verify initial config applied (use thread-safe accessor)
	currentConfig := rl.GetConfig()
	if currentConfig.MaxConnectionsPerUser != 3 {
		t.Errorf("Expected initial MaxConnectionsPerUser=3, got %d", currentConfig.MaxConnectionsPerUser)
	}

	// Update to more permissive config
	newConfig := &RateLimitConfig{
		MessagesPerSecond:     100,
		BurstSize:             50,
		MaxMessageSize:        2048,
		MaxConnectionsPerUser: 10,
		CleanupInterval:       time.Minute,
	}
	rl.UpdateConfig(newConfig)

	// Verify new config applied (use thread-safe accessor)
	updatedConfig := rl.GetConfig()
	if updatedConfig.MaxConnectionsPerUser != 10 {
		t.Errorf("Expected updated MaxConnectionsPerUser=10, got %d", updatedConfig.MaxConnectionsPerUser)
	}
	if updatedConfig.MaxMessageSize != 2048 {
		t.Errorf("Expected updated MaxMessageSize=2048, got %d", updatedConfig.MaxMessageSize)
	}

	// A new user should get the new burst size
	newUserID := "new-config-user"
	allowed := 0
	for i := 0; i < 60; i++ {
		if rl.AllowMessage(newUserID) {
			allowed++
		}
	}

	// With burst=50 and rate=100/sec, should allow a lot more initially
	if allowed < 45 {
		t.Errorf("Expected at least 45 messages allowed with new config, got %d", allowed)
	}
}

func TestHTTP429Handler(t *testing.T) {
	w := httptest.NewRecorder()
	HTTP429Handler(w, "Test rate limit message")

	// Check status code
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter != "1" {
		t.Errorf("Expected Retry-After=1, got %s", retryAfter)
	}

	// Check body contains error info
	body := w.Body.String()
	if !contains(body, "rate_limit_exceeded") {
		t.Error("Response should contain rate_limit_exceeded")
	}
	if !contains(body, "Test rate limit message") {
		t.Error("Response should contain the error message")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Reset global rate limiter for this test
	// (In production, you'd use dependency injection)

	handlerCalled := false
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	wrapped := RateLimitMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	w := httptest.NewRecorder()
	wrapped(w, req)

	if !handlerCalled {
		t.Error("Handler should be called when under limit")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetRateLimiter_Singleton(t *testing.T) {
	rl1 := GetRateLimiter()
	rl2 := GetRateLimiter()

	if rl1 != rl2 {
		t.Error("GetRateLimiter should return the same instance")
	}
}

func TestRateLimiter_Reserve(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerSecond:     10,
		BurstSize:             5,
		MaxMessageSize:        1024,
		MaxConnectionsPerUser: 3,
		CleanupInterval:       time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "reserve-user"

	// Reserve should return a valid reservation
	reservation := rl.Reserve(userID)
	if reservation == nil {
		t.Error("Reserve should return a non-nil reservation")
	}

	// Reservation should have a delay
	delay := reservation.Delay()
	if delay < 0 {
		t.Error("Reservation delay should be non-negative")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmarks

func BenchmarkAllowMessage(b *testing.B) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.AllowMessage(userID)
	}
}

func BenchmarkAllowMessage_MultiUser(b *testing.B) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	users := make([]string, 100)
	for i := range users {
		users[i] = "user-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.AllowMessage(users[i%100])
	}
}

func BenchmarkConnectionAddRemove(b *testing.B) {
	config := DefaultRateLimitConfig()
	config.MaxConnectionsPerUser = 1000 // High limit to avoid blocking
	rl := NewRateLimiter(config)
	defer rl.Stop()

	userID := "bench-conn-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.AddConnection(userID)
		rl.RemoveConnection(userID)
	}
}
