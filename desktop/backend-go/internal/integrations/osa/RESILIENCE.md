# OSA Integration Resilience Patterns

Production-ready error handling patterns for external API integration with the OSA (Open Source Automation) API.

## Table of Contents

1. [Overview](#overview)
2. [Circuit Breaker Pattern](#circuit-breaker-pattern)
3. [Exponential Backoff with Jitter](#exponential-backoff-with-jitter)
4. [Fallback Mechanisms](#fallback-mechanisms)
5. [Health Check Caching](#health-check-caching)
6. [Request Queueing](#request-queueing)
7. [Complete Usage Example](#complete-usage-example)
8. [Configuration](#configuration)
9. [Monitoring and Metrics](#monitoring-and-metrics)

## Overview

The resilience system provides multiple layers of protection against external API failures:

```
Request → Circuit Breaker → Retry with Backoff → Fallback → Queue
            ↓                      ↓                  ↓          ↓
         Reject if open      Exponential wait    Cached data  Store for later
```

### Key Features

- **Circuit Breaker**: Prevents cascading failures by stopping requests to failing services
- **Exponential Backoff**: Retries with increasing delays and jitter to avoid thundering herd
- **Response Caching**: Serves stale data when primary service is unavailable
- **Health Check Caching**: Avoids hammering health endpoints
- **Request Queueing**: Stores requests for processing when service recovers
- **Auto-Recovery**: Automatically processes queued requests when circuit closes

## Circuit Breaker Pattern

The circuit breaker prevents cascading failures by monitoring request success/failure rates and opening the circuit when thresholds are exceeded.

### States

1. **Closed (Normal)**: All requests pass through
2. **Open (Failing)**: All requests are rejected immediately
3. **Half-Open (Testing)**: Limited requests allowed to test recovery

### State Transitions

```
Closed --[Max Failures]--> Open --[Timeout]--> Half-Open
                            ↑                      |
                            |                      |
                            +------[Failure]-------+
                            |
                            +---[Success]----> Closed
```

### Usage

```go
// Create circuit breaker
config := &CircuitBreakerConfig{
    MaxFailures:      5,                // Open after 5 consecutive failures
    Timeout:          30 * time.Second, // Try half-open after 30s
    HalfOpenMaxCalls: 3,                // Allow 3 test requests in half-open
}
cb := NewCircuitBreaker(config)

// Execute with circuit breaker protection
err := cb.Execute(ctx, func() error {
    return someRiskyOperation()
})

if err != nil {
    log.Printf("Circuit state: %s", cb.State())
}
```

### Monitoring

```go
// Get circuit breaker metrics
metrics := cb.Metrics()
log.Printf("Total: %d, Success: %d, Failed: %d, Rejected: %d",
    metrics.totalRequests,
    metrics.successfulRequests,
    metrics.failedRequests,
    metrics.rejectedRequests)
```

## Exponential Backoff with Jitter

Implements intelligent retry logic with exponentially increasing delays and randomization to prevent thundering herd problems.

### Features

- Initial interval: 500ms
- Max interval: 30s
- Max elapsed time: 2 minutes
- Multiplier: 2.0
- Randomization factor: ±50% (jitter)

### Retry Decision Logic

```go
func IsRetryableError(err error) bool {
    // Network errors are retryable
    if contains(err, "timeout") ||
       contains(err, "connection refused") ||
       contains(err, "connection reset") {
        return true
    }

    // HTTP 5xx and 429 are retryable
    if contains(err, "status 500") ||
       contains(err, "status 502") ||
       contains(err, "status 503") ||
       contains(err, "status 504") ||
       contains(err, "status 429") {
        return true
    }

    // All other errors are not retryable
    return false
}
```

### Usage

```go
// Automatic retry with exponential backoff
err := RetryWithBackoff(ctx, func() error {
    return client.GenerateApp(ctx, req)
})

if err != nil {
    log.Printf("Failed after all retries: %v", err)
}
```

### Retry Timeline Example

```
Attempt 1: Immediate
Attempt 2: ~500ms  (250-750ms with jitter)
Attempt 3: ~1000ms (500-1500ms with jitter)
Attempt 4: ~2000ms (1000-3000ms with jitter)
Attempt 5: ~4000ms (2000-6000ms with jitter)
...
Max:       30000ms
```

## Fallback Mechanisms

Provides graceful degradation when primary service fails.

### Strategies

1. **FallbackNone**: Return error immediately (no fallback)
2. **FallbackCache**: Return cached response if available (not expired)
3. **FallbackStale**: Return stale cached response even if expired
4. **FallbackDefault**: Return default/degraded response

### Usage

```go
// Create fallback client
fallbackClient := NewFallbackClient(
    client,
    5 * time.Minute,      // Cache TTL
    FallbackStale,        // Strategy
)

// Automatic fallback on failure
resp, err := fallbackClient.GenerateAppWithFallback(ctx, req)
if err != nil {
    // All strategies failed
    log.Printf("Primary and fallback failed: %v", err)
}
```

### Response Cache

```go
// The cache automatically stores successful responses
cache := NewResponseCache(5 * time.Minute)

// Set response
cache.Set("key", response)

// Get response (allowStale=false)
if resp, ok := cache.Get("key", false); ok {
    // Fresh cached response
}

// Get response (allowStale=true)
if resp, ok := cache.Get("key", true); ok {
    // Potentially stale response
}

// Invalidate specific entry
cache.Invalidate("key")

// Clear all entries
cache.Clear()
```

## Health Check Caching

Prevents excessive health check requests that can overwhelm the service.

### Features

- Configurable TTL (default: 30 seconds)
- Thread-safe
- Automatic cache refresh

### Usage

```go
// Create health check cache
healthCache := NewHealthCheckCache(
    30 * time.Second,  // Cache TTL
    client.HealthCheck, // Health check function
)

// Cached health check
resp, err := healthCache.Check(ctx)
if err != nil {
    log.Printf("Service unhealthy: %v", err)
}

// Force refresh
healthCache.Invalidate()
```

### Benefits

- Reduces load on health endpoints
- Faster health check responses
- Prevents health check storms during incidents

## Request Queueing

Stores failed requests for processing when service recovers.

### Features

- Configurable max size
- Automatic retry on recovery
- Request deduplication
- Metrics and monitoring

### Usage

```go
// Create request queue
queue := NewRequestQueue(1000) // Max 1000 queued requests

// Enqueue request
reqID, err := queue.Enqueue("generate_app", req, userID)
if err != nil {
    log.Printf("Queue full: %v", err)
}

// Check queue size
size := queue.Size()

// Process queued requests (usually automatic)
if req, ok := queue.Dequeue(); ok {
    // Process request
}

// Clear queue
queue.Clear()
```

### Auto-Recovery

The resilient client automatically processes queued requests:

```go
// Auto-recovery runs every 30 seconds
// 1. Check if circuit is closed
// 2. Verify service health
// 3. Process queued requests
// 4. Log results
```

## Complete Usage Example

### Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "your-app/internal/integrations/osa"
)

func main() {
    // Configure resilient client
    config := &osa.ResilientClientConfig{
        OSAConfig: &osa.Config{
            BaseURL:      "https://osa-api.example.com",
            SharedSecret: "your-shared-secret",
            Timeout:      30 * time.Second,
            MaxRetries:   3,
            RetryDelay:   2 * time.Second,
        },
        CircuitBreakerConfig: &osa.CircuitBreakerConfig{
            MaxFailures:      5,
            Timeout:          30 * time.Second,
            HalfOpenMaxCalls: 3,
        },
        FallbackStrategy:    osa.FallbackStale,
        CacheTTL:            5 * time.Minute,
        HealthCheckCacheTTL: 30 * time.Second,
        QueueSize:           1000,
        EnableAutoRecovery:  true,
    }

    // Create resilient client
    client, err := osa.NewResilientClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Use the client
    ctx := context.Background()

    // Generate app with full resilience
    req := &osa.AppGenerationRequest{
        UserID:      userID,
        WorkspaceID: workspaceID,
        Prompt:      "Build a todo app",
    }

    resp, err := client.GenerateApp(ctx, req)
    if err != nil {
        log.Printf("Error: %v", err)
        log.Printf("Circuit state: %s", client.State())
        log.Printf("Queue size: %d", client.QueueSize())
        return
    }

    log.Printf("App generated: %s", resp.AppID)
}
```

### With Monitoring

```go
// Monitor circuit breaker
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        metrics := client.Metrics()

        log.Printf("Circuit Metrics:")
        log.Printf("  Total requests: %d", metrics.totalRequests)
        log.Printf("  Successful: %d", metrics.successfulRequests)
        log.Printf("  Failed: %d", metrics.failedRequests)
        log.Printf("  Rejected: %d", metrics.rejectedRequests)
        log.Printf("  State changes: %d", metrics.stateChanges)
        log.Printf("  Current state: %s", client.State())
        log.Printf("  Queue size: %d", client.QueueSize())
    }
}()
```

### Error Handling

```go
resp, err := client.GenerateApp(ctx, req)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "circuit breaker is open"):
        // Circuit is open - service is failing
        log.Printf("Service unavailable, circuit open")
        // Return 503 Service Unavailable to client

    case strings.Contains(err.Error(), "all resilience strategies failed"):
        // Everything failed - primary, retries, fallback
        log.Printf("Complete failure: %v", err)
        // Return 503 or 500 to client

    case strings.Contains(err.Error(), "context canceled"):
        // Client cancelled request
        log.Printf("Request cancelled")
        // Return 499 Client Closed Request

    default:
        // Other error
        log.Printf("Unexpected error: %v", err)
        // Return 500 Internal Server Error
    }
    return
}

// Success (may be from cache/fallback)
log.Printf("Success: %+v", resp)
```

## Configuration

### Development Configuration

```go
config := &osa.ResilientClientConfig{
    OSAConfig: &osa.Config{
        BaseURL:    "http://localhost:8089",
        Timeout:    10 * time.Second,
        MaxRetries: 2,
        RetryDelay: 1 * time.Second,
    },
    CircuitBreakerConfig: &osa.CircuitBreakerConfig{
        MaxFailures:      3,
        Timeout:          10 * time.Second,
        HalfOpenMaxCalls: 2,
    },
    FallbackStrategy:    osa.FallbackCache,
    CacheTTL:            2 * time.Minute,
    HealthCheckCacheTTL: 15 * time.Second,
    QueueSize:           100,
    EnableAutoRecovery:  true,
}
```

### Production Configuration

```go
config := &osa.ResilientClientConfig{
    OSAConfig: &osa.Config{
        BaseURL:    "https://osa-api.production.com",
        Timeout:    30 * time.Second,
        MaxRetries: 5,
        RetryDelay: 2 * time.Second,
    },
    CircuitBreakerConfig: &osa.CircuitBreakerConfig{
        MaxFailures:      10,           // Higher threshold for production
        Timeout:          60 * time.Second,
        HalfOpenMaxCalls: 5,
    },
    FallbackStrategy:    osa.FallbackStale,  // Serve stale data if needed
    CacheTTL:            10 * time.Minute,
    HealthCheckCacheTTL: 30 * time.Second,
    QueueSize:           5000,               // Larger queue
    EnableAutoRecovery:  true,
}
```

### High-Traffic Configuration

```go
config := &osa.ResilientClientConfig{
    OSAConfig: &osa.Config{
        BaseURL:    "https://osa-api.production.com",
        Timeout:    15 * time.Second,    // Lower timeout
        MaxRetries: 3,                   // Fewer retries
        RetryDelay: 1 * time.Second,
    },
    CircuitBreakerConfig: &osa.CircuitBreakerConfig{
        MaxFailures:      20,            // Very high threshold
        Timeout:          30 * time.Second,
        HalfOpenMaxCalls: 10,
    },
    FallbackStrategy:    osa.FallbackStale,
    CacheTTL:            15 * time.Minute,  // Longer cache
    HealthCheckCacheTTL: 60 * time.Second,  // Longer health cache
    QueueSize:           10000,             // Very large queue
    EnableAutoRecovery:  true,
}
```

## Monitoring and Metrics

### Circuit Breaker Metrics

```go
type CircuitMetrics struct {
    totalRequests      uint64    // Total requests attempted
    successfulRequests uint64    // Successful requests
    failedRequests     uint64    // Failed requests
    rejectedRequests   uint64    // Requests rejected by circuit
    stateChanges       uint64    // Number of state transitions
    lastStateChange    time.Time // Last state change timestamp
}
```

### Key Metrics to Monitor

1. **Circuit State**: Track time spent in each state
2. **Failure Rate**: `failedRequests / totalRequests`
3. **Rejection Rate**: `rejectedRequests / totalRequests`
4. **Queue Size**: Monitor for unbounded growth
5. **Recovery Time**: Time from open → closed

### Alerting Thresholds

```go
// Alert if circuit is open for > 5 minutes
if client.State() == osa.StateOpen &&
   time.Since(metrics.lastStateChange) > 5*time.Minute {
    alert("OSA circuit breaker stuck open")
}

// Alert if rejection rate > 50%
if metrics.rejectedRequests > metrics.totalRequests/2 {
    alert("High OSA request rejection rate")
}

// Alert if queue is > 80% full
if client.QueueSize() > config.QueueSize*80/100 {
    alert("OSA request queue nearly full")
}
```

## Best Practices

1. **Circuit Breaker Tuning**
   - Start conservative (lower thresholds)
   - Monitor and adjust based on actual failure patterns
   - Consider different thresholds for different operations

2. **Retry Strategy**
   - Only retry idempotent operations
   - Use exponential backoff with jitter
   - Set reasonable max retry counts

3. **Caching**
   - Cache frequently accessed data
   - Use stale-while-revalidate pattern
   - Implement cache invalidation strategy

4. **Health Checks**
   - Cache health check results
   - Use circuit breaker for health checks
   - Implement graceful degradation

5. **Monitoring**
   - Track all resilience metrics
   - Set up alerts for anomalies
   - Log state transitions

6. **Testing**
   - Test circuit breaker state transitions
   - Simulate network failures
   - Verify fallback behavior
   - Load test with realistic failure scenarios

## References

- [Circuit Breaker Pattern - Martin Fowler](https://martinfowler.com/bliki/CircuitBreaker.html)
- [sony/gobreaker](https://github.com/sony/gobreaker) - Circuit breaker implementation
- [cenkalti/backoff](https://github.com/cenkalti/backoff) - Exponential backoff implementation
- [API Resilience Patterns](https://api7.ai/blog/10-common-api-resilience-design-patterns)
- [Failsafe-go](https://failsafe-go.dev/) - Fault tolerance patterns for Go
