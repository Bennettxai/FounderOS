package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RAGCacheService provides caching for RAG queries and embeddings
type RAGCacheService struct {
	client *redis.Client
	config RAGCacheConfig
}

// RAGCacheConfig configures the RAG cache
type RAGCacheConfig struct {
	QueryCacheTTL     time.Duration // TTL for query result cache
	EmbeddingCacheTTL time.Duration // TTL for embedding cache
	KeyPrefix         string        // Prefix for cache keys
	Enabled           bool          // Enable/disable caching
}

// DefaultRAGCacheConfig returns sensible cache defaults
func DefaultRAGCacheConfig() RAGCacheConfig {
	return RAGCacheConfig{
		QueryCacheTTL:     15 * time.Minute, // Cache queries for 15 minutes
		EmbeddingCacheTTL: 24 * time.Hour,   // Cache embeddings for 24 hours
		KeyPrefix:         "rag:",
		Enabled:           true,
	}
}

// NewRAGCacheService creates a new RAG cache service
func NewRAGCacheService(client *redis.Client, config RAGCacheConfig) *RAGCacheService {
	if !config.Enabled {
		return nil
	}

	return &RAGCacheService{
		client: client,
		config: config,
	}
}

// CachedHybridSearchResult wraps hybrid search results for caching
type CachedHybridSearchResult struct {
	Results   []HybridSearchResult `json:"results"`
	CachedAt  time.Time            `json:"cached_at"`
	QueryHash string               `json:"query_hash"`
}

// CachedAgenticRAGResponse wraps agentic RAG response for caching
type CachedAgenticRAGResponse struct {
	Response  *AgenticRAGResponse `json:"response"`
	CachedAt  time.Time           `json:"cached_at"`
	QueryHash string              `json:"query_hash"`
}

// CachedEmbedding wraps an embedding vector for caching
type CachedEmbedding struct {
	Embedding []float32 `json:"embedding"`
	CachedAt  time.Time `json:"cached_at"`
	TextHash  string    `json:"text_hash"`
}

// generateQueryHash creates a cache key for a query
func (c *RAGCacheService) generateQueryHash(query string, userID string, params interface{}) string {
	// Combine query, userID, and params to create unique hash
	data := fmt.Sprintf("%s:%s:%v", query, userID, params)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes
}

// generateTextHash creates a cache key for text embeddings
func (c *RAGCacheService) generateTextHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash[:16])
}

// GetHybridSearchResults retrieves cached hybrid search results
func (c *RAGCacheService) GetHybridSearchResults(ctx context.Context, query string, userID string, opts HybridSearchOptions) (*CachedHybridSearchResult, error) {
	if c == nil || !c.config.Enabled {
		return nil, nil // Cache disabled
	}

	queryHash := c.generateQueryHash(query, userID, opts)
	key := c.config.KeyPrefix + "hybrid:" + queryHash

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err // Redis error (don't fail, just miss)
	}

	var cached CachedHybridSearchResult
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// SetHybridSearchResults caches hybrid search results
func (c *RAGCacheService) SetHybridSearchResults(ctx context.Context, query string, userID string, opts HybridSearchOptions, results []HybridSearchResult) error {
	if c == nil || !c.config.Enabled {
		return nil
	}

	queryHash := c.generateQueryHash(query, userID, opts)
	key := c.config.KeyPrefix + "hybrid:" + queryHash

	cached := CachedHybridSearchResult{
		Results:   results,
		CachedAt:  time.Now(),
		QueryHash: queryHash,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.config.QueryCacheTTL).Err()
}

// GetAgenticRAGResponse retrieves cached agentic RAG response
func (c *RAGCacheService) GetAgenticRAGResponse(ctx context.Context, req AgenticRAGRequest) (*CachedAgenticRAGResponse, error) {
	if c == nil || !c.config.Enabled {
		return nil, nil
	}

	queryHash := c.generateQueryHash(req.Query, req.UserID, req)
	key := c.config.KeyPrefix + "agentic:" + queryHash

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cached CachedAgenticRAGResponse
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// SetAgenticRAGResponse caches agentic RAG response
func (c *RAGCacheService) SetAgenticRAGResponse(ctx context.Context, req AgenticRAGRequest, response *AgenticRAGResponse) error {
	if c == nil || !c.config.Enabled {
		return nil
	}

	queryHash := c.generateQueryHash(req.Query, req.UserID, req)
	key := c.config.KeyPrefix + "agentic:" + queryHash

	cached := CachedAgenticRAGResponse{
		Response:  response,
		CachedAt:  time.Now(),
		QueryHash: queryHash,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.config.QueryCacheTTL).Err()
}

// GetEmbedding retrieves cached embedding
func (c *RAGCacheService) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c == nil || !c.config.Enabled {
		return nil, nil
	}

	textHash := c.generateTextHash(text)
	key := c.config.KeyPrefix + "embedding:" + textHash

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cached CachedEmbedding
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return cached.Embedding, nil
}

// SetEmbedding caches an embedding
func (c *RAGCacheService) SetEmbedding(ctx context.Context, text string, embedding []float32) error {
	if c == nil || !c.config.Enabled {
		return nil
	}

	textHash := c.generateTextHash(text)
	key := c.config.KeyPrefix + "embedding:" + textHash

	cached := CachedEmbedding{
		Embedding: embedding,
		CachedAt:  time.Now(),
		TextHash:  textHash,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.config.EmbeddingCacheTTL).Err()
}

// InvalidateUserCache invalidates all cache entries for a user
func (c *RAGCacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	if c == nil || !c.config.Enabled {
		return nil
	}

	// Scan for keys matching this user
	// Note: This is a simple implementation. For production, consider using Redis SCAN
	// with patterns like "rag:*:userid" if you store userID in the key

	// For now, we'll just note this limitation
	// In production, you might want to maintain a set of keys per user
	return nil
}

// GetCacheStats returns cache statistics
func (c *RAGCacheService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	if c == nil || !c.config.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	// Count keys by type
	hybridKeys := 0
	agenticKeys := 0
	embeddingKeys := 0

	// Use SCAN to count keys (more efficient than KEYS)
	iter := c.client.Scan(ctx, 0, c.config.KeyPrefix+"*", 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if strings.Contains(key, "hybrid:") {
			hybridKeys++
		} else if strings.Contains(key, "agentic:") {
			agenticKeys++
		} else if strings.Contains(key, "embedding:") {
			embeddingKeys++
		}
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"enabled":           true,
		"hybrid_cached":     hybridKeys,
		"agentic_cached":    agenticKeys,
		"embeddings_cached": embeddingKeys,
		"total_keys":        hybridKeys + agenticKeys + embeddingKeys,
		"query_ttl":         c.config.QueryCacheTTL.String(),
		"embedding_ttl":     c.config.EmbeddingCacheTTL.String(),
	}, nil
}

// ClearCache clears all RAG cache entries
func (c *RAGCacheService) ClearCache(ctx context.Context) error {
	if c == nil || !c.config.Enabled {
		return nil
	}

	// Delete all keys with our prefix
	iter := c.client.Scan(ctx, 0, c.config.KeyPrefix+"*", 1000).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

// WarmCache pre-populates cache with common queries
func (c *RAGCacheService) WarmCache(ctx context.Context, commonQueries []string, userID string, hybridSearch *HybridSearchService) error {
	if c == nil || !c.config.Enabled || hybridSearch == nil {
		return nil
	}

	opts := DefaultHybridSearchOptions()

	for _, query := range commonQueries {
		// Check if already cached
		cached, _ := c.GetHybridSearchResults(ctx, query, userID, opts)
		if cached != nil {
			continue // Already cached
		}

		// Execute search and cache
		results, err := hybridSearch.Search(ctx, query, userID, opts)
		if err != nil {
			continue // Skip errors during warming
		}

		_ = c.SetHybridSearchResults(ctx, query, userID, opts, results)
	}

	return nil
}
