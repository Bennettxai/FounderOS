package osa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FallbackStrategy defines how to handle failures
type FallbackStrategy int

const (
	// FallbackNone returns error immediately
	FallbackNone FallbackStrategy = iota
	// FallbackCache returns cached response if available
	FallbackCache
	// FallbackStale returns stale cached response even if expired
	FallbackStale
	// FallbackDefault returns a default/degraded response
	FallbackDefault
)

// ResponseCache caches API responses with TTL
type ResponseCache struct {
	mu     sync.RWMutex
	cache  map[string]*CacheEntry
	maxAge time.Duration
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Response  interface{}
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache(maxAge time.Duration) *ResponseCache {
	cache := &ResponseCache{
		cache:  make(map[string]*CacheEntry),
		maxAge: maxAge,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Set stores a response in the cache
func (c *ResponseCache) Set(key string, response interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.cache[key] = &CacheEntry{
		Response:  response,
		CachedAt:  now,
		ExpiresAt: now.Add(c.maxAge),
	}
}

// Get retrieves a response from the cache
func (c *ResponseCache) Get(key string, allowStale bool) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		if allowStale {
			slog.Warn("returning stale cached response",
				"key", key,
				"age", time.Since(entry.CachedAt))
			return entry.Response, true
		}
		return nil, false
	}

	return entry.Response, true
}

// Invalidate removes a specific cache entry
func (c *ResponseCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
}

// Clear removes all cache entries
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

// cleanup periodically removes expired entries
func (c *ResponseCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			// Remove entries expired more than 1 hour ago
			if now.After(entry.ExpiresAt.Add(1 * time.Hour)) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// FallbackClient wraps the OSA client with fallback capabilities
type FallbackClient struct {
	client   *Client
	cache    *ResponseCache
	strategy FallbackStrategy
}

// NewFallbackClient creates a new fallback-enabled client
func NewFallbackClient(client *Client, cacheTTL time.Duration, strategy FallbackStrategy) *FallbackClient {
	return &FallbackClient{
		client:   client,
		cache:    NewResponseCache(cacheTTL),
		strategy: strategy,
	}
}

// GenerateAppWithFallback applies fallback strategy without retrying the request
// This should be called AFTER the primary request has already failed
func (f *FallbackClient) GenerateAppWithFallback(ctx context.Context, req *AppGenerationRequest) (*AppGenerationResponse, error) {
	cacheKey := f.cacheKey("generate_app", req.UserID, req.WorkspaceID)

	// Apply fallback strategy (DO NOT retry the request)
	slog.Info("applying fallback strategy for app generation",
		"strategy", f.strategy)

	return f.applyFallback(cacheKey, fmt.Errorf("primary request failed"), func() *AppGenerationResponse {
		return &AppGenerationResponse{
			AppID:   uuid.New().String(),
			Status:  "failed",
			Message: "Service temporarily unavailable",
		}
	})
}

// GetAppStatusWithFallback applies fallback strategy without retrying the request
func (f *FallbackClient) GetAppStatusWithFallback(ctx context.Context, appID string, userID uuid.UUID) (*AppStatusResponse, error) {
	cacheKey := f.cacheKey("app_status", userID, uuid.Nil) + ":" + appID

	// Apply fallback strategy (DO NOT retry the request)
	slog.Info("applying fallback strategy for app status",
		"app_id", appID,
		"strategy", f.strategy)

	return applyFallback[AppStatusResponse](f, cacheKey, fmt.Errorf("primary request failed"), func() *AppStatusResponse {
		return &AppStatusResponse{
			AppID:  appID,
			Status:  "unknown",
			Error:  "Service temporarily unavailable",
		}
	})
}

// OrchestrateWithFallback applies fallback strategy without retrying the request
func (f *FallbackClient) OrchestrateWithFallback(ctx context.Context, req *OrchestrateRequest) (*OrchestrateResponse, error) {
	cacheKey := f.cacheKey("orchestrate", req.UserID, req.WorkspaceID)

	// Apply fallback strategy (DO NOT retry the request)
	slog.Info("applying fallback strategy for orchestrate",
		"strategy", f.strategy)

	return applyFallback[OrchestrateResponse](f, cacheKey, fmt.Errorf("primary request failed"), func() *OrchestrateResponse {
		return &OrchestrateResponse{
			Success: false,
			Output:  "Service temporarily unavailable",
		}
	})
}

// GetWorkspacesWithFallback applies fallback strategy without retrying the request
func (f *FallbackClient) GetWorkspacesWithFallback(ctx context.Context, userID uuid.UUID) (*WorkspacesResponse, error) {
	cacheKey := f.cacheKey("workspaces", userID, uuid.Nil)

	// Apply fallback strategy (DO NOT retry the request)
	slog.Info("applying fallback strategy for workspaces",
		"strategy", f.strategy)

	return applyFallback[WorkspacesResponse](f, cacheKey, fmt.Errorf("primary request failed"), func() *WorkspacesResponse {
		return &WorkspacesResponse{
			Workspaces: []Workspace{},
		}
	})
}

// applyFallback applies the configured fallback strategy
func applyFallback[T any](f *FallbackClient, cacheKey string, err error, defaultFunc func() *T) (*T, error) {
	switch f.strategy {
	case FallbackNone:
		return nil, err

	case FallbackCache:
		// Try to get cached response (not expired)
		if cached, ok := f.cache.Get(cacheKey, false); ok {
			slog.Info("returning cached response",
				"cache_key", cacheKey)
			if resp, ok := cached.(*T); ok {
				return resp, nil
			}
		}
		return nil, err

	case FallbackStale:
		// Try to get cached response (even if expired)
		if cached, ok := f.cache.Get(cacheKey, true); ok {
			slog.Warn("returning stale cached response due to service unavailability",
				"cache_key", cacheKey)
			if resp, ok := cached.(*T); ok {
				return resp, nil
			}
		}
		return nil, err

	case FallbackDefault:
		// Return default/degraded response
		slog.Warn("returning default response due to service unavailability",
			"cache_key", cacheKey)
		return defaultFunc(), nil

	default:
		return nil, err
	}
}

// Helper method using the generic function
func (f *FallbackClient) applyFallback(cacheKey string, err error, defaultFunc func() *AppGenerationResponse) (*AppGenerationResponse, error) {
	return applyFallback[AppGenerationResponse](f, cacheKey, err, defaultFunc)
}

// cacheKey generates a cache key for a request
func (f *FallbackClient) cacheKey(operation string, userID, workspaceID uuid.UUID) string {
	if workspaceID == uuid.Nil {
		return fmt.Sprintf("%s:%s", operation, userID)
	}
	return fmt.Sprintf("%s:%s:%s", operation, userID, workspaceID)
}

// RequestQueue queues requests when service is unavailable
type RequestQueue struct {
	mu       sync.RWMutex
	queue    []QueuedRequest
	maxSize  int
	enabled  bool
}

// QueuedRequest represents a queued request
type QueuedRequest struct {
	ID        string
	Operation string
	Payload   json.RawMessage
	QueuedAt  time.Time
	UserID    uuid.UUID
	RetryCount int
}

// NewRequestQueue creates a new request queue
func NewRequestQueue(maxSize int) *RequestQueue {
	return &RequestQueue{
		queue:   make([]QueuedRequest, 0),
		maxSize: maxSize,
		enabled: true,
	}
}

// Enqueue adds a request to the queue
func (q *RequestQueue) Enqueue(operation string, payload interface{}, userID uuid.UUID) (string, error) {
	if !q.enabled {
		return "", fmt.Errorf("queue is disabled")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) >= q.maxSize {
		return "", fmt.Errorf("queue is full (max: %d)", q.maxSize)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req := QueuedRequest{
		ID:        uuid.New().String(),
		Operation: operation,
		Payload:   payloadBytes,
		QueuedAt:  time.Now(),
		UserID:    userID,
	}

	q.queue = append(q.queue, req)

	slog.Info("request queued",
		"request_id", req.ID,
		"operation", operation,
		"queue_size", len(q.queue))

	return req.ID, nil
}

// Dequeue removes and returns the oldest request from the queue
func (q *RequestQueue) Dequeue() (*QueuedRequest, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return nil, false
	}

	req := q.queue[0]
	q.queue = q.queue[1:]

	return &req, true
}

// Size returns the current queue size
func (q *RequestQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// Clear removes all requests from the queue
func (q *RequestQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = make([]QueuedRequest, 0)
}

// Enable enables the queue
func (q *RequestQueue) Enable() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = true
}

// Disable disables the queue
func (q *RequestQueue) Disable() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enabled = false
}
