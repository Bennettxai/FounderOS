package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueryCache provides query result caching with automatic serialization
type QueryCache struct {
	redis          *redis.Client
	logger         *slog.Logger
	cacheSemaphore chan struct{} // Limits concurrent cache write goroutines
	hits           atomic.Int64  // Cache hit counter
	misses         atomic.Int64  // Cache miss counter
}

// NewQueryCache creates a new query cache instance
func NewQueryCache(redis *redis.Client, logger *slog.Logger) *QueryCache {
	return &QueryCache{
		redis:          redis,
		logger:         logger,
		cacheSemaphore: make(chan struct{}, 100), // Max 100 concurrent cache writes
	}
}

// CacheConfig contains caching configuration for a query
type CacheConfig struct {
	TTL             time.Duration // Cache lifetime
	Prefix          string        // Cache key prefix
	IncludeUserID   bool          // Include user ID in cache key
	IncludeFilters  bool          // Include filter params in cache key
	CompressionSize int           // Compress results larger than this (0 = no compression)
}

// DefaultCacheConfig returns sensible defaults
func DefaultCacheConfig(prefix string) CacheConfig {
	return CacheConfig{
		TTL:            10 * time.Minute,
		Prefix:         prefix,
		IncludeUserID:  true,
		IncludeFilters: true,
	}
}

// =============================================================================
// GENERIC QUERY CACHING
// =============================================================================

// Get retrieves a cached query result
func (q *QueryCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	start := time.Now()

	data, err := q.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		q.misses.Add(1) // Atomic increment
		q.logger.Debug("Cache miss",
			"key", key,
			"duration", time.Since(start))
		return false, nil
	}
	if err != nil {
		q.logger.Error("Cache get error",
			"key", key,
			"error", err)
		return false, fmt.Errorf("cache get: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		q.logger.Error("Cache unmarshal error",
			"key", key,
			"error", err)
		return false, fmt.Errorf("unmarshal cached data: %w", err)
	}

	q.hits.Add(1) // Atomic increment
	q.logger.Debug("Cache hit",
		"key", key,
		"duration", time.Since(start))

	return true, nil
}

// Set stores a query result in cache
func (q *QueryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()

	data, err := json.Marshal(value)
	if err != nil {
		q.logger.Error("Cache marshal error",
			"key", key,
			"error", err)
		return fmt.Errorf("marshal value: %w", err)
	}

	if err := q.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		q.logger.Error("Cache set error",
			"key", key,
			"error", err)
		return fmt.Errorf("cache set: %w", err)
	}

	q.logger.Debug("Cache set",
		"key", key,
		"size", len(data),
		"ttl", ttl,
		"duration", time.Since(start))

	return nil
}

// GetOrCompute retrieves from cache or computes the result
func (q *QueryCache) GetOrCompute(
	ctx context.Context,
	key string,
	ttl time.Duration,
	dest interface{},
	compute func() (interface{}, error),
) error {
	// Try cache first
	hit, err := q.Get(ctx, key, dest)
	if err != nil {
		q.logger.Warn("Cache error, computing fresh",
			"key", key,
			"error", err)
	}
	if hit {
		return nil
	}

	// Cache miss - compute result
	result, err := compute()
	if err != nil {
		return err
	}

	// Store in cache (fire and forget with bounded goroutines)
	// Use semaphore pattern to prevent unbounded goroutine spawning
	select {
	case q.cacheSemaphore <- struct{}{}: // Acquire semaphore slot
		go func() {
			defer func() { <-q.cacheSemaphore }() // Release slot when done

			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := q.Set(cacheCtx, key, result, ttl); err != nil {
				q.logger.Warn("Failed to cache result",
					"key", key,
					"error", err)
			}
		}()
	default:
		// Semaphore pool full - acceptable to drop cache operation
		// This prevents OOM under extreme load
		q.logger.Debug("Cache write pool full, dropping cache operation",
			"key", key)
	}

	// Copy result to dest
	data, err := json.Marshal(result)
	if err != nil {
		q.logger.Error("failed to marshal cached result", slog.Any("error", err))
		return fmt.Errorf("marshal cached result: %w", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		q.logger.Error("failed to unmarshal to dest", slog.Any("error", err))
		return fmt.Errorf("unmarshal to dest: %w", err)
	}

	return nil
}

// =============================================================================
// QUERY-SPECIFIC CACHING HELPERS
// =============================================================================

// ConversationListKey generates a cache key for conversation lists
func (q *QueryCache) ConversationListKey(userID string, page int, filters map[string]string) string {
	return q.buildKey("conversations", userID, page, filters)
}

// ArtifactListKey generates a cache key for artifact lists
func (q *QueryCache) ArtifactListKey(userID string, page int, filters map[string]string) string {
	return q.buildKey("artifacts", userID, page, filters)
}

// TaskListKey generates a cache key for task lists
func (q *QueryCache) TaskListKey(userID string, page int, filters map[string]string) string {
	return q.buildKey("tasks", userID, page, filters)
}

// MemoryListKey generates a cache key for memory lists
func (q *QueryCache) MemoryListKey(userID string, workspaceID *string, page int, filters map[string]string) string {
	if workspaceID != nil {
		filters["workspace_id"] = *workspaceID
	}
	return q.buildKey("memories", userID, page, filters)
}

// ProjectListKey generates a cache key for project lists
func (q *QueryCache) ProjectListKey(userID string, page int, filters map[string]string) string {
	return q.buildKey("projects", userID, page, filters)
}

// SearchResultKey generates a cache key for search results
func (q *QueryCache) SearchResultKey(userID string, query string, filters map[string]string) string {
	filters["q"] = query
	return q.buildKey("search", userID, 0, filters)
}

// =============================================================================
// CONVERSATION HISTORY CACHING (High Priority)
// =============================================================================

// GetConversationMessages retrieves cached conversation messages
func (q *QueryCache) GetConversationMessages(ctx context.Context, conversationID string, dest interface{}) (bool, error) {
	key := fmt.Sprintf("conv:%s:messages", conversationID)
	return q.Get(ctx, key, dest)
}

// SetConversationMessages caches conversation messages
func (q *QueryCache) SetConversationMessages(ctx context.Context, conversationID string, messages interface{}) error {
	key := fmt.Sprintf("conv:%s:messages", conversationID)
	return q.Set(ctx, key, messages, 1*time.Hour)
}

// =============================================================================
// RAG EMBEDDING CACHING (Critical)
// =============================================================================

// GetEmbedding retrieves a cached embedding
func (q *QueryCache) GetEmbedding(ctx context.Context, text string, dest interface{}) (bool, error) {
	hash := q.hashText(text)
	key := fmt.Sprintf("embed:%s", hash)
	return q.Get(ctx, key, dest)
}

// SetEmbedding caches an embedding result
func (q *QueryCache) SetEmbedding(ctx context.Context, text string, embedding interface{}) error {
	hash := q.hashText(text)
	key := fmt.Sprintf("embed:%s", hash)
	return q.Set(ctx, key, embedding, 24*time.Hour)
}

// =============================================================================
// AGENT STATUS CACHING (Medium Priority)
// =============================================================================

// GetAgentStatus retrieves cached agent status
func (q *QueryCache) GetAgentStatus(ctx context.Context, agentID string, dest interface{}) (bool, error) {
	key := fmt.Sprintf("agent:%s:status", agentID)
	return q.Get(ctx, key, dest)
}

// SetAgentStatus caches agent status
func (q *QueryCache) SetAgentStatus(ctx context.Context, agentID string, status interface{}) error {
	key := fmt.Sprintf("agent:%s:status", agentID)
	return q.Set(ctx, key, status, 5*time.Minute)
}

// =============================================================================
// CACHE STATISTICS
// =============================================================================

// QueryCacheStats represents query cache performance metrics
type QueryCacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	TotalKeys   int64   `json:"total_keys"`
	MemoryUsed  string  `json:"memory_used"`
	EvictedKeys int64   `json:"evicted_keys"`
}

// GetStats retrieves cache statistics
func (q *QueryCache) GetStats(ctx context.Context) (*QueryCacheStats, error) {
	// Note: Redis INFO command result is currently unused
	// but the connection check is performed
	_, err := q.redis.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("get redis info: %w", err)
	}

	// Parse stats from info string
	stats := &QueryCacheStats{}

	// Get total keys
	dbSize := q.redis.DBSize(ctx)
	stats.TotalKeys, _ = dbSize.Result()

	// Read atomic counters
	stats.Hits = q.hits.Load()
	stats.Misses = q.misses.Load()
	stats.MemoryUsed = "N/A"
	stats.EvictedKeys = 0

	// Calculate hit rate
	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
	}

	q.logger.Info("Cache stats retrieved",
		"hits", stats.Hits,
		"misses", stats.Misses,
		"hit_rate", fmt.Sprintf("%.2f%%", stats.HitRate),
		"total_keys", stats.TotalKeys)

	return stats, nil
}

// =============================================================================
// INTERNAL HELPERS
// =============================================================================

// buildKey constructs a cache key from components
func (q *QueryCache) buildKey(prefix string, userID string, page int, filters map[string]string) string {
	// Start with prefix and user ID
	key := fmt.Sprintf("%s:%s", prefix, userID)

	// Add page if not first page
	if page > 0 {
		key += fmt.Sprintf(":page:%d", page)
	}

	// Add filters hash if any
	if len(filters) > 0 {
		filtersHash := q.hashFilters(filters)
		key += fmt.Sprintf(":filters:%s", filtersHash)
	}

	return key
}

// hashText generates a SHA256 hash of text
func (q *QueryCache) hashText(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// hashFilters generates a deterministic hash of filters
func (q *QueryCache) hashFilters(filters map[string]string) string {
	// Sort keys for deterministic hashing
	data, err := json.Marshal(filters)
	if err != nil {
		// Fallback to string representation if marshal fails
		data = []byte(fmt.Sprintf("%v", filters))
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter keys
}

// =============================================================================
// BATCH OPERATIONS
// =============================================================================

// MGet retrieves multiple keys at once
func (q *QueryCache) MGet(ctx context.Context, keys []string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	values, err := q.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("mget: %w", err)
	}

	return values, nil
}

// MSet sets multiple key-value pairs at once
func (q *QueryCache) MSet(ctx context.Context, pairs map[string]interface{}, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	pipe := q.redis.Pipeline()

	for key, value := range pairs {
		data, err := json.Marshal(value)
		if err != nil {
			q.logger.Error("Marshal error in MSet",
				"key", key,
				"error", err)
			continue
		}
		pipe.Set(ctx, key, data, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("mset pipeline: %w", err)
	}

	return nil
}

// =============================================================================
// CACHE INVALIDATION / DELETION
// =============================================================================

// Delete removes a cache entry by exact key
func (q *QueryCache) Delete(ctx context.Context, key string) error {
	start := time.Now()

	if err := q.redis.Del(ctx, key).Err(); err != nil {
		q.logger.Error("Cache delete error",
			"key", key,
			"error", err)
		return fmt.Errorf("cache delete: %w", err)
	}

	q.logger.Debug("Cache deleted",
		"key", key,
		"duration", time.Since(start))

	return nil
}

// DeleteByPattern removes all cache entries matching a pattern
func (q *QueryCache) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	start := time.Now()

	var deletedCount int64
	iter := q.redis.Scan(ctx, 0, pattern, 100).Iterator()

	var keysToDelete []string
	for iter.Next(ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
	}

	if err := iter.Err(); err != nil {
		q.logger.Error("Cache pattern scan error",
			"pattern", pattern,
			"error", err)
		return 0, fmt.Errorf("scan cache pattern: %w", err)
	}

	if len(keysToDelete) == 0 {
		q.logger.Debug("No cache keys found for pattern",
			"pattern", pattern)
		return 0, nil
	}

	// Delete all matched keys in a pipeline for efficiency
	pipe := q.redis.Pipeline()
	for _, key := range keysToDelete {
		pipe.Del(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		q.logger.Error("Cache pattern deletion error",
			"pattern", pattern,
			"count", len(keysToDelete),
			"error", err)
		return 0, fmt.Errorf("delete cache by pattern: %w", err)
	}

	deletedCount = int64(len(keysToDelete))
	q.logger.Debug("Cache entries deleted by pattern",
		"pattern", pattern,
		"count", deletedCount,
		"duration", time.Since(start))

	return deletedCount, nil
}

// MDel removes multiple cache entries at once
func (q *QueryCache) MDel(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	start := time.Now()

	result, err := q.redis.Del(ctx, keys...).Result()
	if err != nil {
		q.logger.Error("Cache mdel error",
			"count", len(keys),
			"error", err)
		return 0, fmt.Errorf("cache mdel: %w", err)
	}

	q.logger.Debug("Cache entries deleted",
		"count", result,
		"duration", time.Since(start))

	return result, nil
}
