package osa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	t.Run("starts in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("opens after max failures", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      3,
			Timeout:          1 * time.Second,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config)

		ctx := context.Background()

		// First 3 failures should open the circuit
		for i := 0; i < 3; i++ {
			err := cb.Execute(ctx, func() error {
				return errors.New("simulated failure")
			})
			assert.Error(t, err)
		}

		assert.Equal(t, StateOpen, cb.State())
	})

	t.Run("rejects requests when open", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			Timeout:          100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config)

		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return errors.New("failure")
			})
		}

		// Should reject the next request
		err := cb.Execute(ctx, func() error {
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			Timeout:          50 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config)

		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return errors.New("failure")
			})
		}

		assert.Equal(t, StateOpen, cb.State())

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Next request should transition to half-open
		cb.Execute(ctx, func() error {
			return nil
		})

		assert.Equal(t, StateHalfOpen, cb.State())
	})

	t.Run("closes after successful half-open requests", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			Timeout:          50 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config)

		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return errors.New("failure")
			})
		}

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Successful requests in half-open state
		for i := 0; i < 2; i++ {
			err := cb.Execute(ctx, func() error {
				return nil
			})
			assert.NoError(t, err)
		}

		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("reopens on failure in half-open state", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			Timeout:          50 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config)

		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return errors.New("failure")
			})
		}

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// First success
		cb.Execute(ctx, func() error {
			return nil
		})

		assert.Equal(t, StateHalfOpen, cb.State())

		// Then failure
		cb.Execute(ctx, func() error {
			return errors.New("failure")
		})

		assert.Equal(t, StateOpen, cb.State())
	})

	t.Run("tracks metrics correctly", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		ctx := context.Background()

		// Execute some requests
		cb.Execute(ctx, func() error { return nil })
		cb.Execute(ctx, func() error { return errors.New("error") })
		cb.Execute(ctx, func() error { return nil })

		metrics := cb.Metrics()
		assert.Equal(t, uint64(3), metrics.totalRequests)
		assert.Equal(t, uint64(2), metrics.successfulRequests)
		assert.Equal(t, uint64(1), metrics.failedRequests)
	})
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("succeeds on first try", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := RetryWithBackoff(ctx, func() error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on retryable errors", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := RetryWithBackoff(ctx, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("status 503")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("stops on non-retryable errors", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := RetryWithBackoff(ctx, func() error {
			attempts++
			return errors.New("status 400")
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		attempts := 0
		err := RetryWithBackoff(ctx, func() error {
			attempts++
			return errors.New("status 503")
		})

		assert.Error(t, err)
		// Should not retry when context is cancelled
		assert.LessOrEqual(t, attempts, 1)
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("connection timeout"), true},
		{"503 error", errors.New("status 503"), true},
		{"500 error", errors.New("status 500"), true},
		{"429 rate limit", errors.New("status 429"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"400 bad request", errors.New("status 400"), false},
		{"404 not found", errors.New("status 404"), false},
		{"generic error", errors.New("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthCheckCache(t *testing.T) {
	t.Run("caches health check results", func(t *testing.T) {
		calls := 0
		checkFunc := func(ctx context.Context) (*HealthResponse, error) {
			calls++
			return &HealthResponse{Status: "healthy"}, nil
		}

		cache := NewHealthCheckCache(100*time.Millisecond, checkFunc)

		// First call should invoke the function
		resp1, err1 := cache.Check(context.Background())
		assert.NoError(t, err1)
		assert.Equal(t, "healthy", resp1.Status)
		assert.Equal(t, 1, calls)

		// Second call should return cached result
		resp2, err2 := cache.Check(context.Background())
		assert.NoError(t, err2)
		assert.Equal(t, "healthy", resp2.Status)
		assert.Equal(t, 1, calls, "should not call function again")
	})

	t.Run("refreshes after TTL expires", func(t *testing.T) {
		calls := 0
		checkFunc := func(ctx context.Context) (*HealthResponse, error) {
			calls++
			return &HealthResponse{Status: "healthy"}, nil
		}

		cache := NewHealthCheckCache(50*time.Millisecond, checkFunc)

		// First call
		cache.Check(context.Background())
		assert.Equal(t, 1, calls)

		// Wait for TTL to expire
		time.Sleep(60 * time.Millisecond)

		// Second call should refresh
		cache.Check(context.Background())
		assert.Equal(t, 2, calls)
	})

	t.Run("invalidate clears cache", func(t *testing.T) {
		calls := 0
		checkFunc := func(ctx context.Context) (*HealthResponse, error) {
			calls++
			return &HealthResponse{Status: "healthy"}, nil
		}

		cache := NewHealthCheckCache(1*time.Second, checkFunc)

		// First call
		cache.Check(context.Background())
		assert.Equal(t, 1, calls)

		// Invalidate
		cache.Invalidate()

		// Next call should refresh
		cache.Check(context.Background())
		assert.Equal(t, 2, calls)
	})
}
