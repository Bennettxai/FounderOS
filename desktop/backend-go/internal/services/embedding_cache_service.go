package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// EmbeddingCacheService provides Redis-backed caching for embeddings
// with fallback to database when Redis is unavailable
type EmbeddingCacheService struct {
	redisClient *redis.Client
	pool        *pgxpool.Pool
	enabled     atomic.Bool
	keyPrefix   string
	stats       cacheStats
}

// cacheStats tracks cache performance metrics
type cacheStats struct {
	hits   atomic.Int64
	misses atomic.Int64
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	Size    int64   `json:"size"`
	HitRate float64 `json:"hit_rate"`
}

// CachedEmbeddingData wraps embedding data for caching
type CachedEmbeddingData struct {
	Embedding     []float64 `json:"embedding"`
	EmbeddingType string    `json:"embedding_type"`
	ContentHash   string    `json:"content_hash"`
	CachedAt      time.Time `json:"cached_at"`
}

// EmbeddingCacheConfig configures the embedding cache
type EmbeddingCacheConfig struct {
	KeyPrefix      string        // Redis key prefix (default: "embedding:")
	DefaultTTL     time.Duration // Default TTL for embeddings (default: 24 hours)
	TextTTL        time.Duration // TTL for text embeddings (default: 24 hours)
	ImageTTL       time.Duration // TTL for image embeddings (default: 48 hours)
	Enabled        bool          // Enable/disable caching
	GracefulFallback bool        // If true, fall back to database on Redis errors
}

// DefaultEmbeddingCacheConfig returns sensible defaults
func DefaultEmbeddingCacheConfig() *EmbeddingCacheConfig {
	return &EmbeddingCacheConfig{
		KeyPrefix:        "embedding:",
		DefaultTTL:       24 * time.Hour,
		TextTTL:          24 * time.Hour,
		ImageTTL:         48 * time.Hour,
		Enabled:          true,
		GracefulFallback: true,
	}
}

// NewEmbeddingCacheService creates a new embedding cache service
// If redisClient is nil, the service will be disabled
// pool is optional and only used for fallback queries
func NewEmbeddingCacheService(redisClient *redis.Client, pool *pgxpool.Pool, cfg *EmbeddingCacheConfig) *EmbeddingCacheService {
	if cfg == nil {
		cfg = DefaultEmbeddingCacheConfig()
	}

	// If Redis is not available, disable caching
	enabled := cfg.Enabled && redisClient != nil
	if !enabled {
		slog.Info("EmbeddingCache: Redis not available, caching disabled")
	}

	service := &EmbeddingCacheService{
		redisClient: redisClient,
		pool:        pool,
		keyPrefix:   cfg.KeyPrefix,
	}
	service.enabled.Store(enabled)

	// Test Redis connection if enabled
	if enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Info("EmbeddingCache: Redis ping failed, disabling cache", "error", err)
			service.enabled.Store(false)
		} else {
			slog.Info("EmbeddingCache: Redis connected successfully")
		}
	}

	return service
}

// generateContentHash creates a deterministic hash for content
// This ensures the same content always generates the same cache key
func (s *EmbeddingCacheService) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// getCacheKey generates Redis key for an embedding
func (s *EmbeddingCacheService) getCacheKey(content string, embeddingType string) string {
	contentHash := s.generateContentHash(content)
	return fmt.Sprintf("%s%s:%s", s.keyPrefix, embeddingType, contentHash)
}

// GetEmbedding retrieves a cached embedding
// Returns (embedding, found, error)
// - embedding: the cached embedding vector (nil if not found)
// - found: true if found in cache, false otherwise
// - error: non-nil if there was an error accessing the cache
func (s *EmbeddingCacheService) GetEmbedding(ctx context.Context, content string, embeddingType string) ([]float64, bool, error) {
	if !s.enabled.Load() {
		return nil, false, nil
	}

	// Validate inputs
	if content == "" {
		return nil, false, fmt.Errorf("content cannot be empty")
	}
	if embeddingType == "" {
		embeddingType = "text" // Default type
	}

	key := s.getCacheKey(content, embeddingType)

	// Try to get from Redis with timeout
	getCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	data, err := s.redisClient.Get(getCtx, key).Result()
	if err == redis.Nil {
		// Cache miss - normal case
		s.stats.misses.Add(1)
		return nil, false, nil
	}
	if err != nil {
		// Redis error - log and return as miss (graceful degradation)
		slog.Info("EmbeddingCache: Redis GET error for key", "value", key, "error", err)
		s.stats.misses.Add(1)

		// Optionally disable cache if Redis is completely down
		if s.isRedisDown(err) {
			slog.Info("EmbeddingCache: Redis appears down, temporarily disabling")
			s.enabled.Store(false)
		}

		return nil, false, nil
	}

	// Unmarshal cached data
	var cached CachedEmbeddingData
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		slog.Info("EmbeddingCache: Failed to unmarshal cached embedding", "error", err)
		// Delete corrupted cache entry
		_ = s.redisClient.Del(ctx, key).Err()
		s.stats.misses.Add(1)
		return nil, false, nil
	}

	// Validate cached data
	if len(cached.Embedding) == 0 {
		slog.Info("EmbeddingCache: Cached embedding is empty, invalidating")
		_ = s.redisClient.Del(ctx, key).Err()
		s.stats.misses.Add(1)
		return nil, false, nil
	}

	// Cache hit
	s.stats.hits.Add(1)
	return cached.Embedding, true, nil
}

// SetEmbedding caches an embedding with specified TTL
// If ttl is 0, uses default TTL based on embedding type
func (s *EmbeddingCacheService) SetEmbedding(ctx context.Context, content string, embedding []float64, embeddingType string, ttl time.Duration) error {
	if !s.enabled.Load() {
		return nil // Silently skip if disabled
	}

	// Validate inputs
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if len(embedding) == 0 {
		return fmt.Errorf("embedding cannot be empty")
	}
	if embeddingType == "" {
		embeddingType = "text"
	}

	// Determine TTL
	if ttl == 0 {
		// Use default TTL based on type
		switch embeddingType {
		case "text":
			ttl = 24 * time.Hour
		case "image":
			ttl = 48 * time.Hour
		default:
			ttl = 24 * time.Hour
		}
	}

	key := s.getCacheKey(content, embeddingType)
	contentHash := s.generateContentHash(content)

	// Create cached data
	cached := CachedEmbeddingData{
		Embedding:     embedding,
		EmbeddingType: embeddingType,
		ContentHash:   contentHash,
		CachedAt:      time.Now(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	// Set in Redis with timeout
	setCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.redisClient.Set(setCtx, key, data, ttl).Err(); err != nil {
		slog.Info("EmbeddingCache: Redis SET error for key", "value", key, "error", err)

		// Check if Redis is down
		if s.isRedisDown(err) {
			slog.Info("EmbeddingCache: Redis appears down, temporarily disabling")
			s.enabled.Store(false)
		}

		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// InvalidateEmbedding removes a specific embedding from cache
func (s *EmbeddingCacheService) InvalidateEmbedding(ctx context.Context, content string, embeddingType string) error {
	if !s.enabled.Load() {
		return nil
	}

	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if embeddingType == "" {
		embeddingType = "text"
	}

	key := s.getCacheKey(content, embeddingType)

	delCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.redisClient.Del(delCtx, key).Err(); err != nil {
		slog.Info("EmbeddingCache: Failed to delete key", "id", key, "error", err)
		return fmt.Errorf("redis delete failed: %w", err)
	}

	return nil
}

// ClearCache removes all embeddings from cache
func (s *EmbeddingCacheService) ClearCache(ctx context.Context) error {
	if !s.enabled.Load() {
		return nil
	}

	clearCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use SCAN to find all embedding keys
	var cursor uint64
	var deletedCount int64

	for {
		var keys []string
		var err error

		keys, cursor, err = s.redisClient.Scan(clearCtx, cursor, s.keyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Delete keys in batch
		if len(keys) > 0 {
			pipe := s.redisClient.Pipeline()
			for _, key := range keys {
				pipe.Del(clearCtx, key)
			}

			cmds, err := pipe.Exec(clearCtx)
			if err != nil {
				slog.Info("EmbeddingCache: Pipeline delete error", "error", err)
			} else {
				deletedCount += int64(len(cmds))
			}
		}

		// Break when cursor returns to 0 (full scan complete)
		if cursor == 0 {
			break
		}
	}

	slog.Info("EmbeddingCache: Cleared  cached embeddings", "count", deletedCount)

	// Reset stats
	s.stats.hits.Store(0)
	s.stats.misses.Store(0)

	return nil
}

// GetCacheStats returns cache statistics
func (s *EmbeddingCacheService) GetCacheStats(ctx context.Context) (CacheStats, error) {
	stats := CacheStats{
		Hits:   s.stats.hits.Load(),
		Misses: s.stats.misses.Load(),
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	// Get cache size if Redis is enabled
	if s.enabled.Load() {
		sizeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Count keys using SCAN
		var cursor uint64
		var count int64

		for {
			keys, nextCursor, err := s.redisClient.Scan(sizeCtx, cursor, s.keyPrefix+"*", 100).Result()
			if err != nil {
				slog.Info("EmbeddingCache: Failed to scan keys for stats", "error", err)
				break
			}

			count += int64(len(keys))
			cursor = nextCursor

			if cursor == 0 {
				break
			}
		}

		stats.Size = count
	}

	return stats, nil
}

// isRedisDown checks if the error indicates Redis is completely unavailable
func (s *EmbeddingCacheService) isRedisDown(err error) bool {
	if err == nil {
		return false
	}

	// Check for common Redis connection errors
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "i/o timeout") ||
		contains(errStr, "no route to host")
}

// contains is a helper to check if a string contains a substring
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

// HealthCheck verifies Redis connection
func (s *EmbeddingCacheService) HealthCheck(ctx context.Context) bool {
	if !s.enabled.Load() || s.redisClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := s.redisClient.Ping(ctx).Err()
	if err != nil {
		slog.Info("EmbeddingCache: Health check failed", "error", err)
		return false
	}

	return true
}

// Enable re-enables the cache after it was disabled
func (s *EmbeddingCacheService) Enable() {
	if s.redisClient != nil {
		// Test connection first
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.redisClient.Ping(ctx).Err(); err == nil {
			s.enabled.Store(true)
			slog.Info("EmbeddingCache: Re-enabled successfully")
		} else {
			slog.Info("EmbeddingCache: Cannot enable, Redis still unavailable", "error", err)
		}
	}
}

// Disable temporarily disables the cache
func (s *EmbeddingCacheService) Disable() {
	s.enabled.Store(false)
	slog.Info("EmbeddingCache: Disabled")
}

// IsEnabled returns whether the cache is currently enabled
func (s *EmbeddingCacheService) IsEnabled() bool {
	return s.enabled.Load()
}
