package services

import (
	"context"
	"time"
)

// EmbeddingCacheAdapter adapts EmbeddingCacheService to work with float32 embeddings
// This provides compatibility with existing code that uses []float32
type EmbeddingCacheAdapter struct {
	cache *EmbeddingCacheService
}

// NewEmbeddingCacheAdapter creates an adapter for float32 compatibility
func NewEmbeddingCacheAdapter(cache *EmbeddingCacheService) *EmbeddingCacheAdapter {
	return &EmbeddingCacheAdapter{
		cache: cache,
	}
}

// GetEmbedding retrieves a cached embedding and converts to float32
func (a *EmbeddingCacheAdapter) GetEmbedding(ctx context.Context, content string, embeddingType string) ([]float32, bool, error) {
	if a.cache == nil {
		return nil, false, nil
	}

	// Get float64 embedding from cache
	embedding64, found, err := a.cache.GetEmbedding(ctx, content, embeddingType)
	if err != nil || !found {
		return nil, found, err
	}

	// Convert []float64 to []float32
	embedding32 := make([]float32, len(embedding64))
	for i, v := range embedding64 {
		embedding32[i] = float32(v)
	}

	return embedding32, true, nil
}

// SetEmbedding caches a float32 embedding (converts to float64)
func (a *EmbeddingCacheAdapter) SetEmbedding(ctx context.Context, content string, embedding []float32, embeddingType string, ttl time.Duration) error {
	if a.cache == nil {
		return nil
	}

	// Convert []float32 to []float64
	embedding64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embedding64[i] = float64(v)
	}

	return a.cache.SetEmbedding(ctx, content, embedding64, embeddingType, ttl)
}

// InvalidateEmbedding invalidates a cached embedding
func (a *EmbeddingCacheAdapter) InvalidateEmbedding(ctx context.Context, content string, embeddingType string) error {
	if a.cache == nil {
		return nil
	}
	return a.cache.InvalidateEmbedding(ctx, content, embeddingType)
}

// ClearCache clears all cached embeddings
func (a *EmbeddingCacheAdapter) ClearCache(ctx context.Context) error {
	if a.cache == nil {
		return nil
	}
	return a.cache.ClearCache(ctx)
}

// GetCacheStats returns cache statistics
func (a *EmbeddingCacheAdapter) GetCacheStats(ctx context.Context) (CacheStats, error) {
	if a.cache == nil {
		return CacheStats{}, nil
	}
	return a.cache.GetCacheStats(ctx)
}

// HealthCheck checks cache health
func (a *EmbeddingCacheAdapter) HealthCheck(ctx context.Context) bool {
	if a.cache == nil {
		return false
	}
	return a.cache.HealthCheck(ctx)
}

// IsEnabled checks if cache is enabled
func (a *EmbeddingCacheAdapter) IsEnabled() bool {
	if a.cache == nil {
		return false
	}
	return a.cache.IsEnabled()
}

// Enable enables the cache
func (a *EmbeddingCacheAdapter) Enable() {
	if a.cache != nil {
		a.cache.Enable()
	}
}

// Disable disables the cache
func (a *EmbeddingCacheAdapter) Disable() {
	if a.cache != nil {
		a.cache.Disable()
	}
}

// GetUnderlyingCache returns the underlying float64 cache
// Use this if you need direct access to the float64 version
func (a *EmbeddingCacheAdapter) GetUnderlyingCache() *EmbeddingCacheService {
	return a.cache
}
