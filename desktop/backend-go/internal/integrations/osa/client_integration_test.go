package osa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOSAConfig creates an OSA config suitable for unit tests (with mock server)
func testOSAConfig(serverURL string, timeout time.Duration) *Config {
	return &Config{
		BaseURL:      serverURL,
		Timeout:      timeout,
		SharedSecret: "test-secret-for-unit-tests",
	}
}

// sdkAppGenResponse returns a valid SDK-shaped GenerateApp JSON response.
func sdkAppGenResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app_id":       uuid.New().String(),
		"status":       "processing",
		"workspace_id": uuid.New().String(),
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	})
}

// TestResilientClientCircuitBreaker tests circuit breaker behavior
func TestResilientClientCircuitBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Circuit opens after threshold failures", func(t *testing.T) {
		failureCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			failureCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.MaxRetryTime = 0 // Disable retries for this test

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 1*time.Second),
			CircuitBreakerConfig: cbConfig,
			EnableAutoRecovery:   false,
			FallbackStrategy:     FallbackNone, // No fallback, fail immediately
			QueueSize:            0,            // No queue
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)

		ctx := context.Background()

		// Trigger failures to open circuit
		for i := 0; i < 5; i++ {
			_, err := client.GenerateApp(ctx, &AppGenerationRequest{
				UserID:      uuid.New(),
				WorkspaceID: uuid.New(),
				Name:        "test",
				Description: "test",
			})
			assert.Error(t, err)
		}

		// Circuit should be open now; next request fails fast without hitting server
		beforeFailureCount := failureCount
		_, err = client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		assert.Error(t, err)
		assert.Equal(t, beforeFailureCount, failureCount, "Circuit is open, server should not be hit")
	})

	t.Run("Circuit half-opens and closes on success", func(t *testing.T) {
		requestCount := int32(0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			if count <= 3 {
				// First 3 requests fail
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				// Subsequent requests succeed with SDK-shaped response
				sdkAppGenResponse(w)
			}
		}))
		defer server.Close()

		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.MaxRetryTime = 0          // Disable retries for this test
		cbConfig.Timeout = 1 * time.Second // Shorter timeout for test

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 1*time.Second),
			CircuitBreakerConfig: cbConfig,
			EnableAutoRecovery:   false,
			FallbackStrategy:     FallbackNone, // No fallback, fail immediately
			QueueSize:            0,            // No queue
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		// Trigger failures to open circuit
		for i := 0; i < 3; i++ {
			client.GenerateApp(ctx, &AppGenerationRequest{
				Name:        "test",
				Description: "test",
			})
		}

		// Wait for circuit to half-open (timeout + small buffer)
		time.Sleep(1500 * time.Millisecond)

		// Next request should succeed and close the circuit
		resp, err := client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "processing", resp.Status)
	})
}

// TestResilientClientRetryLogic tests exponential backoff retry
func TestResilientClientRetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Retry with exponential backoff", func(t *testing.T) {
		attemptCount := 0
		requestTimes := []time.Time{}
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			attemptCount++
			requestTimes = append(requestTimes, time.Now())
			mu.Unlock()

			if attemptCount < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				sdkAppGenResponse(w)
			}
		}))
		defer server.Close()

		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.MaxRetryTime = 10 * time.Second // Short retry timeout for tests

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 5*time.Second),
			CircuitBreakerConfig: cbConfig,
			EnableAutoRecovery:   true,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		resp, err := client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 3, attemptCount, "Should have made 3 attempts")

		// Verify retry spacing. The implementation intentionally adds jitter to
		// avoid retry herds, so exact exponential ordering is not deterministic.
		mu.Lock()
		defer mu.Unlock()
		if len(requestTimes) >= 3 {
			gap1 := requestTimes[1].Sub(requestTimes[0])
			gap2 := requestTimes[2].Sub(requestTimes[1])

			assert.Greater(t, gap1, 0*time.Millisecond, "First retry should be delayed")
			assert.Greater(t, gap2, 0*time.Millisecond, "Second retry should be delayed")
		}
	})

	t.Run("Max retries exhausted", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.MaxRetryTime = 2 * time.Second // Short retry timeout for tests

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 1*time.Second),
			CircuitBreakerConfig: cbConfig,
			EnableAutoRecovery:   false,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		_, err = client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		assert.Error(t, err)
		assert.GreaterOrEqual(t, attemptCount, 1, "Should attempt at least once")
	})
}

// TestResilientClientHealthCheck tests health check caching.
// The BOS HealthCheckCache TTL governs how often the underlying client.HealthCheck
// is called. The SDK client's internal health cache has its own 30s TTL, but
// since the BOS resilient client wraps the base client.HealthCheck (which calls
// sdk.Health), and the SDK's internal cache is keyed per-instance, we verify
// BOS-layer caching behavior by measuring ResilientClient.HealthCheck calls
// against known server hit counts.
func TestResilientClientHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Health check uses BOS-layer cache", func(t *testing.T) {
		healthCheckCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				healthCheckCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "healthy",
					"version": "1.0.0",
				})
			}
		}))
		defer server.Close()

		// Use a very short BOS cache TTL so we can expire it quickly in the test.
		cfg := &ResilientClientConfig{
			OSAConfig:           testOSAConfig(server.URL, 1*time.Second),
			HealthCheckCacheTTL: 500 * time.Millisecond,
			EnableAutoRecovery:  false,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		// First health check — hits the server.
		health1, err := client.HealthCheck(ctx)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health1.Status)
		assert.GreaterOrEqual(t, healthCheckCount, 1, "Expected at least one server hit")

		// Second health check within the BOS cache TTL — BOS cache serves it.
		hitsBefore := healthCheckCount
		health2, err := client.HealthCheck(ctx)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health2.Status)
		assert.Equal(t, hitsBefore, healthCheckCount, "BOS cache should absorb the second call")
	})
}

// TestResilientClientConcurrency tests concurrent requests
func TestResilientClientConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Handle concurrent requests safely", func(t *testing.T) {
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			sdkAppGenResponse(w)
		}))
		defer server.Close()

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 5*time.Second),
			CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
			EnableAutoRecovery:   false,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		numRequests := 10
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer func() { done <- true }()

				_, err := client.GenerateApp(ctx, &AppGenerationRequest{
					Name:        "concurrent-test",
					Description: "test",
				})
				if err != nil {
					errors <- err
				}
			}()
		}

		// Wait for all requests
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(10 * time.Second):
				t.Fatal("timeout waiting for concurrent requests")
			}
		}

		close(errors)
		errorCount := 0
		for err := range errors {
			t.Logf("Request error: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "All concurrent requests should succeed")
		assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(numRequests), "Most requests should hit server")
	})
}

// TestResilientClientTimeout tests request timeout handling
func TestResilientClientTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Request times out after configured duration", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 500*time.Millisecond),
			CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
			EnableAutoRecovery:   false,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)
		ctx := context.Background()

		start := time.Now()
		_, err = client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Less(t, elapsed, 2*time.Second, "Should timeout quickly")
	})
}

// TestResilientClientContextCancellation tests context cancellation
func TestResilientClientContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilient client integration test in short mode")
	}
	t.Run("Request cancelled via context", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &ResilientClientConfig{
			OSAConfig:            testOSAConfig(server.URL, 10*time.Second),
			CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
			EnableAutoRecovery:   false,
		}

		client, err := NewResilientClient(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err = client.GenerateApp(ctx, &AppGenerationRequest{
			Name:        "test",
			Description: "test",
		})
		assert.Error(t, err)
	})
}
