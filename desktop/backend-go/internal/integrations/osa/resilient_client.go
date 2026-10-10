package osa

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ResilientClient wraps the OSA client with circuit breaker, retry, and fallback
type ResilientClient struct {
	client             *Client
	circuitBreaker     *CircuitBreaker
	healthCheckCache   *HealthCheckCache
	fallbackClient     *FallbackClient
	requestQueue       *RequestQueue
	enableAutoRecovery bool
}

// ResilientClientConfig holds configuration for the resilient client
type ResilientClientConfig struct {
	OSAConfig            *Config
	CircuitBreakerConfig *CircuitBreakerConfig
	FallbackStrategy     FallbackStrategy
	CacheTTL             time.Duration
	HealthCheckCacheTTL  time.Duration
	QueueSize            int
	EnableAutoRecovery   bool
}

// DefaultResilientClientConfig returns sensible defaults
func DefaultResilientClientConfig() *ResilientClientConfig {
	return &ResilientClientConfig{
		OSAConfig:            DefaultConfig(),
		CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
		FallbackStrategy:     FallbackStale,
		CacheTTL:             5 * time.Minute,
		HealthCheckCacheTTL:  30 * time.Second,
		QueueSize:            1000,
		EnableAutoRecovery:   true,
	}
}

// NewResilientClient creates a new resilient OSA client
func NewResilientClient(config *ResilientClientConfig) (*ResilientClient, error) {
	if config == nil {
		config = DefaultResilientClientConfig()
	}

	client, err := NewClient(config.OSAConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSA client: %w", err)
	}

	circuitBreaker := NewCircuitBreaker(config.CircuitBreakerConfig)
	fallbackClient := NewFallbackClient(client, config.CacheTTL, config.FallbackStrategy)
	requestQueue := NewRequestQueue(config.QueueSize)
	healthCheckCache := NewHealthCheckCache(config.HealthCheckCacheTTL, client.HealthCheck)

	resilientClient := &ResilientClient{
		client:             client,
		circuitBreaker:     circuitBreaker,
		healthCheckCache:   healthCheckCache,
		fallbackClient:     fallbackClient,
		requestQueue:       requestQueue,
		enableAutoRecovery: config.EnableAutoRecovery,
	}

	if config.EnableAutoRecovery {
		go resilientClient.autoRecoveryLoop()
	}

	return resilientClient, nil
}

// HealthCheck performs a health check with caching
func (r *ResilientClient) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	return r.healthCheckCache.Check(ctx)
}

// Metrics returns circuit breaker metrics
func (r *ResilientClient) Metrics() CircuitMetrics {
	return r.circuitBreaker.Metrics()
}

// State returns the current circuit breaker state
func (r *ResilientClient) State() CircuitState {
	return r.circuitBreaker.State()
}

// QueueSize returns the current request queue size
func (r *ResilientClient) QueueSize() int {
	return r.requestQueue.Size()
}

// InvalidateHealthCache invalidates the health check cache
func (r *ResilientClient) InvalidateHealthCache() {
	r.healthCheckCache.Invalidate()
}

// InvalidateCache invalidates all response caches
func (r *ResilientClient) InvalidateCache() {
	r.fallbackClient.cache.Clear()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (r *ResilientClient) ResetCircuitBreaker() {
	r.circuitBreaker.Reset()
	slog.Info("circuit breaker manually reset")
}

// GetClient returns the underlying OSA client for direct access
func (r *ResilientClient) GetClient() *Client {
	return r.client
}

// Close closes the client and cleans up resources
func (r *ResilientClient) Close() error {
	return r.client.Close()
}
