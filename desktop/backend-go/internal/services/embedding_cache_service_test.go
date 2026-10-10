package services

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a Redis client for testing
// Set REDIS_URL environment variable or use default localhost:6379
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests to avoid conflicts
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	// Clear test database before tests
	require.NoError(t, client.FlushDB(ctx).Err())

	return client
}

func TestNewEmbeddingCacheService(t *testing.T) {
	t.Run("creates service with default config", func(t *testing.T) {
		client := setupTestRedis(t)
		defer client.Close()

		service := NewEmbeddingCacheService(client, nil, nil)
		assert.NotNil(t, service)
		assert.True(t, service.IsEnabled())
		assert.Equal(t, "embedding:", service.keyPrefix)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		client := setupTestRedis(t)
		defer client.Close()

		cfg := &EmbeddingCacheConfig{
			KeyPrefix:  "test_embed:",
			DefaultTTL: 1 * time.Hour,
			Enabled:    true,
		}

		service := NewEmbeddingCacheService(client, nil, cfg)
		assert.NotNil(t, service)
		assert.True(t, service.IsEnabled())
		assert.Equal(t, "test_embed:", service.keyPrefix)
	})

	t.Run("disables service when Redis is nil", func(t *testing.T) {
		service := NewEmbeddingCacheService(nil, nil, nil)
		assert.NotNil(t, service)
		assert.False(t, service.IsEnabled())
	})

	t.Run("disables service when config disabled", func(t *testing.T) {
		client := setupTestRedis(t)
		defer client.Close()

		cfg := &EmbeddingCacheConfig{
			Enabled: false,
		}

		service := NewEmbeddingCacheService(client, nil, cfg)
		assert.NotNil(t, service)
		assert.False(t, service.IsEnabled())
	})
}

func TestGetAndSetEmbedding(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("cache miss returns not found", func(t *testing.T) {
		embedding, found, err := service.GetEmbedding(ctx, "test content", "text")
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, embedding)
	})

	t.Run("set and get embedding successfully", func(t *testing.T) {
		content := "hello world"
		embeddingType := "text"
		expectedEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

		// Set embedding
		err := service.SetEmbedding(ctx, content, expectedEmbedding, embeddingType, 1*time.Minute)
		require.NoError(t, err)

		// Get embedding
		actualEmbedding, found, err := service.GetEmbedding(ctx, content, embeddingType)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expectedEmbedding, actualEmbedding)
	})

	t.Run("same content generates same cache key", func(t *testing.T) {
		content := "deterministic content"
		embedding1 := []float64{1.0, 2.0, 3.0}
		embedding2 := []float64{4.0, 5.0, 6.0}

		// Set first embedding
		err := service.SetEmbedding(ctx, content, embedding1, "text", 1*time.Minute)
		require.NoError(t, err)

		// Overwrite with second embedding
		err = service.SetEmbedding(ctx, content, embedding2, "text", 1*time.Minute)
		require.NoError(t, err)

		// Should get second embedding
		retrieved, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, embedding2, retrieved)
	})

	t.Run("different types use different cache keys", func(t *testing.T) {
		content := "multi-modal content"
		textEmbedding := []float64{1.0, 2.0}
		imageEmbedding := []float64{3.0, 4.0}

		// Set text embedding
		err := service.SetEmbedding(ctx, content, textEmbedding, "text", 1*time.Minute)
		require.NoError(t, err)

		// Set image embedding
		err = service.SetEmbedding(ctx, content, imageEmbedding, "image", 1*time.Minute)
		require.NoError(t, err)

		// Get both
		retrievedText, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, textEmbedding, retrievedText)

		retrievedImage, found, err := service.GetEmbedding(ctx, content, "image")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, imageEmbedding, retrievedImage)
	})

	t.Run("handles large embeddings", func(t *testing.T) {
		content := "large embedding test"
		// Create a large embedding (e.g., 1536 dimensions like OpenAI)
		largeEmbedding := make([]float64, 1536)
		for i := range largeEmbedding {
			largeEmbedding[i] = float64(i) * 0.001
		}

		err := service.SetEmbedding(ctx, content, largeEmbedding, "text", 1*time.Minute)
		require.NoError(t, err)

		retrieved, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, len(largeEmbedding), len(retrieved))
		assert.Equal(t, largeEmbedding, retrieved)
	})
}

func TestInvalidateEmbedding(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("invalidate removes embedding from cache", func(t *testing.T) {
		content := "content to invalidate"
		embedding := []float64{1.0, 2.0, 3.0}

		// Set embedding
		err := service.SetEmbedding(ctx, content, embedding, "text", 1*time.Minute)
		require.NoError(t, err)

		// Verify it's cached
		retrieved, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, embedding, retrieved)

		// Invalidate
		err = service.InvalidateEmbedding(ctx, content, "text")
		require.NoError(t, err)

		// Verify it's gone
		retrieved, found, err = service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, retrieved)
	})

	t.Run("invalidating non-existent key doesn't error", func(t *testing.T) {
		err := service.InvalidateEmbedding(ctx, "non-existent", "text")
		assert.NoError(t, err)
	})
}

func TestClearCache(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("clear removes all embeddings", func(t *testing.T) {
		// Add multiple embeddings
		embeddings := map[string][]float64{
			"content1": {1.0, 2.0},
			"content2": {3.0, 4.0},
			"content3": {5.0, 6.0},
		}

		for content, embedding := range embeddings {
			err := service.SetEmbedding(ctx, content, embedding, "text", 1*time.Minute)
			require.NoError(t, err)
		}

		// Verify they're cached
		stats, err := service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), stats.Size)

		// Clear cache
		err = service.ClearCache(ctx)
		require.NoError(t, err)

		// Verify all gone
		stats, err = service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Size)

		// Verify individual lookups fail
		for content := range embeddings {
			_, found, err := service.GetEmbedding(ctx, content, "text")
			require.NoError(t, err)
			assert.False(t, found)
		}
	})
}

func TestCacheStats(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("tracks hits and misses", func(t *testing.T) {
		content := "stats test"
		embedding := []float64{1.0, 2.0, 3.0}

		// Initial stats should be zero
		stats, err := service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, 0.0, stats.HitRate)

		// Cache miss
		_, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.False(t, found)

		stats, err = service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.0, stats.HitRate)

		// Set embedding
		err = service.SetEmbedding(ctx, content, embedding, "text", 1*time.Minute)
		require.NoError(t, err)

		// Cache hit
		_, found, err = service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)

		stats, err = service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.5, stats.HitRate)

		// Another hit
		_, found, err = service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)

		stats, err = service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.InDelta(t, 0.666, stats.HitRate, 0.01)
	})

	t.Run("calculates cache size", func(t *testing.T) {
		// Clear first
		err := service.ClearCache(ctx)
		require.NoError(t, err)

		// Add embeddings
		for i := 0; i < 5; i++ {
			content := string(rune('a' + i))
			embedding := []float64{float64(i)}
			err := service.SetEmbedding(ctx, content, embedding, "text", 1*time.Minute)
			require.NoError(t, err)
		}

		stats, err := service.GetCacheStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), stats.Size)
	})
}

func TestTTLBehavior(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("embedding expires after TTL", func(t *testing.T) {
		content := "expiring content"
		embedding := []float64{1.0, 2.0}

		// Set with very short TTL
		err := service.SetEmbedding(ctx, content, embedding, "text", 1*time.Second)
		require.NoError(t, err)

		// Should be found immediately
		_, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)

		// Wait for expiration
		time.Sleep(1500 * time.Millisecond)

		// Should not be found
		_, found, err = service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("uses default TTL when zero provided", func(t *testing.T) {
		content := "default ttl test"
		embedding := []float64{1.0}

		// Set with zero TTL (should use default)
		err := service.SetEmbedding(ctx, content, embedding, "text", 0)
		require.NoError(t, err)

		// Should be found
		_, found, err := service.GetEmbedding(ctx, content, "text")
		require.NoError(t, err)
		assert.True(t, found)

		// Check TTL in Redis (should be ~24 hours)
		key := service.getCacheKey(content, "text")
		ttl := client.TTL(ctx, key).Val()
		assert.Greater(t, ttl.Hours(), 23.0) // At least 23 hours
		assert.Less(t, ttl.Hours(), 25.0)    // At most 25 hours
	})
}

func TestGracefulDegradation(t *testing.T) {
	t.Run("disabled service handles operations gracefully", func(t *testing.T) {
		service := NewEmbeddingCacheService(nil, nil, nil)
		ctx := context.Background()

		// All operations should succeed without errors
		embedding := []float64{1.0, 2.0}

		err := service.SetEmbedding(ctx, "test", embedding, "text", 1*time.Minute)
		assert.NoError(t, err)

		_, found, err := service.GetEmbedding(ctx, "test", "text")
		assert.NoError(t, err)
		assert.False(t, found)

		err = service.InvalidateEmbedding(ctx, "test", "text")
		assert.NoError(t, err)

		err = service.ClearCache(ctx)
		assert.NoError(t, err)

		stats, err := service.GetCacheStats(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), stats.Hits)
	})
}

func TestHealthCheck(t *testing.T) {
	t.Run("health check passes with good Redis", func(t *testing.T) {
		client := setupTestRedis(t)
		defer client.Close()

		service := NewEmbeddingCacheService(client, nil, nil)
		ctx := context.Background()

		healthy := service.HealthCheck(ctx)
		assert.True(t, healthy)
	})

	t.Run("health check fails when disabled", func(t *testing.T) {
		service := NewEmbeddingCacheService(nil, nil, nil)
		ctx := context.Background()

		healthy := service.HealthCheck(ctx)
		assert.False(t, healthy)
	})
}

func TestEnableDisable(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)

	t.Run("can disable and enable service", func(t *testing.T) {
		assert.True(t, service.IsEnabled())

		service.Disable()
		assert.False(t, service.IsEnabled())

		service.Enable()
		assert.True(t, service.IsEnabled())
	})
}

func TestValidation(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	service := NewEmbeddingCacheService(client, nil, nil)
	ctx := context.Background()

	t.Run("rejects empty content", func(t *testing.T) {
		embedding := []float64{1.0, 2.0}

		err := service.SetEmbedding(ctx, "", embedding, "text", 1*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content cannot be empty")

		_, _, err = service.GetEmbedding(ctx, "", "text")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})

	t.Run("rejects empty embedding", func(t *testing.T) {
		err := service.SetEmbedding(ctx, "test", []float64{}, "text", 1*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embedding cannot be empty")
	})

	t.Run("uses default type when empty", func(t *testing.T) {
		embedding := []float64{1.0, 2.0}

		// Set with empty type (should default to "text")
		err := service.SetEmbedding(ctx, "default type test", embedding, "", 1*time.Minute)
		require.NoError(t, err)

		// Get with explicit "text" type
		retrieved, found, err := service.GetEmbedding(ctx, "default type test", "text")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, embedding, retrieved)
	})
}

func BenchmarkEmbeddingCache(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	defer client.Close()
	service := NewEmbeddingCacheService(client, nil, nil)

	embedding := make([]float64, 768) // Typical embedding size
	for i := range embedding {
		embedding[i] = float64(i) * 0.001
	}

	b.Run("SetEmbedding", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content := string(rune('a' + (i % 26)))
			_ = service.SetEmbedding(ctx, content, embedding, "text", 1*time.Minute)
		}
	})

	b.Run("GetEmbedding-Hit", func(b *testing.B) {
		// Pre-populate cache
		_ = service.SetEmbedding(ctx, "benchmark", embedding, "text", 1*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = service.GetEmbedding(ctx, "benchmark", "text")
		}
	})

	b.Run("GetEmbedding-Miss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content := string(rune('A' + (i % 26)))
			_, _, _ = service.GetEmbedding(ctx, content, "text")
		}
	})
}
