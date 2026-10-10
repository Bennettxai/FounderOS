package appgen

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota // Normal operation
	StateOpen                        // Circuit broken, rejecting requests
	StateHalfOpen                    // Testing if service recovered
)

// CircuitBreaker prevents cascading failures when Claude API is unstable
type CircuitBreaker struct {
	maxFailures     int           // Number of failures before opening circuit
	timeout         time.Duration // Time to wait before attempting half-open
	resetSuccesses  int           // Number of successes needed to close circuit
	failureCount    int           // Current failure count
	successCount    int           // Current success count (in half-open state)
	state           CircuitState
	lastFailureTime time.Time
	mu              sync.RWMutex
	logger          *slog.Logger
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures    int           // Default: 5
	Timeout        time.Duration // Default: 30s
	ResetSuccesses int           // Default: 2
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:    5,
		Timeout:        30 * time.Second,
		ResetSuccesses: 2,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	if logger == nil {
		logger = slog.Default()
	}
	return &CircuitBreaker{
		maxFailures:    config.MaxFailures,
		timeout:        config.Timeout,
		resetSuccesses: config.ResetSuccesses,
		state:          StateClosed,
		logger:         logger,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := fn(ctx)
	cb.afterRequest(err)
	return err
}

// beforeRequest checks if request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Try half-open state
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.logger.Info("circuit breaker transitioning to half-open",
				"timeout", cb.timeout,
			)
			return nil
		}
		return fmt.Errorf("circuit breaker is open (failures: %d, retry after: %v)",
			cb.failureCount,
			cb.timeout-time.Since(cb.lastFailureTime),
		)
	case StateHalfOpen, StateClosed:
		return nil
	default:
		return nil
	}
}

// afterRequest updates circuit breaker state based on result
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		switch cb.state {
		case StateHalfOpen:
			// Failure in half-open state immediately reopens circuit
			cb.state = StateOpen
			cb.logger.Warn("circuit breaker reopened after half-open failure",
				"error", err,
			)
		case StateClosed:
			if cb.failureCount >= cb.maxFailures {
				cb.state = StateOpen
				cb.logger.Error("circuit breaker opened due to failures",
					"failures", cb.failureCount,
					"max_failures", cb.maxFailures,
				)
			}
		}
	} else {
		// Success
		switch cb.state {
		case StateHalfOpen:
			cb.successCount++
			if cb.successCount >= cb.resetSuccesses {
				// Enough successes, close circuit
				cb.state = StateClosed
				cb.failureCount = 0
				cb.successCount = 0
				cb.logger.Info("circuit breaker closed after successful recovery",
					"successes", cb.successCount,
				)
			}
		case StateClosed:
			// Reset failure count on success
			cb.failureCount = 0
		}
	}
}

// GetState returns current circuit state (thread-safe)
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stateStr := "closed"
	if cb.state == StateOpen {
		stateStr = "open"
	} else if cb.state == StateHalfOpen {
		stateStr = "half-open"
	}

	return map[string]interface{}{
		"state":          stateStr,
		"failure_count":  cb.failureCount,
		"success_count":  cb.successCount,
		"max_failures":   cb.maxFailures,
		"last_failure":   cb.lastFailureTime,
		"reset_in":       cb.timeout - time.Since(cb.lastFailureTime),
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.logger.Info("circuit breaker manually reset")
}
