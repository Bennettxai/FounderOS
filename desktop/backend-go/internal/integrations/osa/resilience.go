package osa

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	maxFailures      uint32
	timeout          time.Duration
	halfOpenMaxCalls uint32
	maxRetryTime     time.Duration

	// State
	state            CircuitState
	failures         uint32
	lastFailureTime  time.Time
	halfOpenCalls    uint32
	consecutiveSucc  uint32
	nextAttemptTime  time.Time

	// Metrics
	metrics *CircuitMetrics
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed allows all requests through
	StateClosed CircuitState = iota
	// StateOpen rejects all requests
	StateOpen
	// StateHalfOpen allows limited requests to test if service recovered
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitMetrics tracks circuit breaker performance
type CircuitMetrics struct {
	mu                sync.RWMutex
	totalRequests     uint64
	successfulRequests uint64
	failedRequests    uint64
	rejectedRequests  uint64
	stateChanges      uint64
	lastStateChange   time.Time
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// MaxFailures before opening the circuit
	MaxFailures uint32
	// Timeout before attempting half-open state
	Timeout time.Duration
	// HalfOpenMaxCalls allowed in half-open state before closing
	HalfOpenMaxCalls uint32
	// MaxRetryTime is the maximum time to retry a single request (0 = no retries)
	MaxRetryTime time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,                // Open after 5 consecutive failures
		Timeout:          30 * time.Second, // Try half-open after 30 seconds
		HalfOpenMaxCalls: 3,                // Allow 3 calls in half-open state
		MaxRetryTime:     2 * time.Minute,  // Retry for up to 2 minutes
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
		maxRetryTime:     config.MaxRetryTime,
		state:            StateClosed,
		metrics:          &CircuitMetrics{},
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := fn()
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.mu.Lock()
	cb.metrics.totalRequests++
	cb.metrics.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout expired
		if time.Now().After(cb.nextAttemptTime) {
			cb.setState(StateHalfOpen)
			cb.halfOpenCalls = 0
			return nil
		}

		cb.metrics.mu.Lock()
		cb.metrics.rejectedRequests++
		cb.metrics.mu.Unlock()

		return fmt.Errorf("circuit breaker is open (state: %s, next attempt: %s)",
			cb.state, cb.nextAttemptTime.Format(time.RFC3339))

	case StateHalfOpen:
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			cb.metrics.mu.Lock()
			cb.metrics.rejectedRequests++
			cb.metrics.mu.Unlock()

			return fmt.Errorf("circuit breaker half-open limit reached")
		}

		cb.halfOpenCalls++
		return nil

	default:
		return fmt.Errorf("unknown circuit breaker state: %d", cb.state)
	}
}

// afterRequest updates circuit breaker state based on request result
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Request failed
		cb.metrics.mu.Lock()
		cb.metrics.failedRequests++
		cb.metrics.mu.Unlock()

		cb.failures++
		cb.consecutiveSucc = 0
		cb.lastFailureTime = time.Now()

		switch cb.state {
		case StateClosed:
			// Open circuit when we reach max failures threshold
			if cb.failures >= cb.maxFailures {
				cb.setState(StateOpen)
				cb.nextAttemptTime = time.Now().Add(cb.timeout)
			}

		case StateHalfOpen:
			// Failure in half-open state reopens the circuit
			cb.setState(StateOpen)
			cb.nextAttemptTime = time.Now().Add(cb.timeout)
		}
	} else {
		// Request succeeded
		cb.metrics.mu.Lock()
		cb.metrics.successfulRequests++
		cb.metrics.mu.Unlock()

		cb.consecutiveSucc++

		switch cb.state {
		case StateHalfOpen:
			// Close circuit after enough successful half-open requests
			if cb.consecutiveSucc >= cb.halfOpenMaxCalls {
				cb.setState(StateClosed)
				cb.failures = 0
			}

		case StateClosed:
			// Reset failure count on success
			if cb.consecutiveSucc >= 1 {
				cb.failures = 0
			}
		}
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		slog.Info("circuit breaker state change",
			"old_state", cb.state,
			"new_state", newState,
			"failures", cb.failures,
			"consecutive_successes", cb.consecutiveSucc)

		cb.state = newState

		cb.metrics.mu.Lock()
		cb.metrics.stateChanges++
		cb.metrics.lastStateChange = time.Now()
		cb.metrics.mu.Unlock()
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Metrics returns a copy of the circuit breaker metrics
func (cb *CircuitBreaker) Metrics() CircuitMetrics {
	cb.metrics.mu.RLock()
	defer cb.metrics.mu.RUnlock()

	return CircuitMetrics{
		totalRequests:      cb.metrics.totalRequests,
		successfulRequests: cb.metrics.successfulRequests,
		failedRequests:     cb.metrics.failedRequests,
		rejectedRequests:   cb.metrics.rejectedRequests,
		stateChanges:       cb.metrics.stateChanges,
		lastStateChange:    cb.metrics.lastStateChange,
	}
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.consecutiveSucc = 0
	cb.halfOpenCalls = 0
}

// ExponentialBackoff creates an exponential backoff strategy with jitter
func ExponentialBackoff() *backoff.ExponentialBackOff {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = 500 * time.Millisecond
	eb.MaxInterval = 30 * time.Second
	eb.Multiplier = 2.0
	eb.RandomizationFactor = 0.5 // Add jitter: ±50%
	return eb
}

// RetryWithBackoff executes a function with exponential backoff
func RetryWithBackoff(ctx context.Context, operation func() error) error {
	return RetryWithBackoffTimeout(ctx, operation, 2*time.Minute)
}

// RetryWithBackoffTimeout executes a function with exponential backoff and configurable timeout
func RetryWithBackoffTimeout(ctx context.Context, operation func() error, maxElapsedTime time.Duration) error {
	// If maxElapsedTime is 0, don't retry at all
	if maxElapsedTime == 0 {
		return operation()
	}

	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		err := operation()
		if err != nil {
			// Check if error is retryable
			if IsRetryableError(err) {
				slog.Debug("retrying operation after error",
					"error", err)
				return struct{}{}, err
			}
			// Non-retryable error, stop retrying
			return struct{}{}, backoff.Permanent(err)
		}
		return struct{}{}, nil
	}, backoff.WithBackOff(ExponentialBackoff()), backoff.WithMaxElapsedTime(maxElapsedTime))

	return err
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add custom logic for determining retryable errors
	// Common retryable errors:
	// - Network timeouts
	// - Temporary network errors
	// - HTTP 5xx errors
	// - Rate limit errors (HTTP 429)

	errStr := err.Error()

	// Network errors
	if contains(errStr, "timeout") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "temporary failure") {
		return true
	}

	// HTTP status codes that are retryable
	if contains(errStr, "status 500") ||
		contains(errStr, "status 502") ||
		contains(errStr, "status 503") ||
		contains(errStr, "status 504") ||
		contains(errStr, "status 429") {
		return true
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// HealthCheckCache caches health check results to avoid hammering the health endpoint
type HealthCheckCache struct {
	mu          sync.RWMutex
	lastCheck   time.Time
	lastResult  *HealthResponse
	lastError   error
	cacheTTL    time.Duration
	checkFunc   func(context.Context) (*HealthResponse, error)
}

// NewHealthCheckCache creates a new health check cache
func NewHealthCheckCache(ttl time.Duration, checkFunc func(context.Context) (*HealthResponse, error)) *HealthCheckCache {
	return &HealthCheckCache{
		cacheTTL:  ttl,
		checkFunc: checkFunc,
	}
}

// Check performs a health check with caching
func (h *HealthCheckCache) Check(ctx context.Context) (*HealthResponse, error) {
	h.mu.RLock()
	if time.Since(h.lastCheck) < h.cacheTTL {
		result := h.lastResult
		err := h.lastError
		h.mu.RUnlock()
		return result, err
	}
	h.mu.RUnlock()

	// Cache expired, perform actual health check
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(h.lastCheck) < h.cacheTTL {
		return h.lastResult, h.lastError
	}

	result, err := h.checkFunc(ctx)
	h.lastCheck = time.Now()
	h.lastResult = result
	h.lastError = err

	return result, err
}

// Invalidate clears the health check cache
func (h *HealthCheckCache) Invalidate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheck = time.Time{}
	h.lastResult = nil
	h.lastError = nil
}
