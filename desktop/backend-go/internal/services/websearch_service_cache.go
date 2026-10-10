package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// ============================================================================
// CACHED SEARCH SERVICE
// ============================================================================

// CachedWebSearchService wraps WebSearchService with caching capabilities
type CachedWebSearchService struct {
	*WebSearchService
	pool         *pgxpool.Pool
	cacheTTL     time.Duration
	newsCacheTTL time.Duration
}

// NewCachedWebSearchService creates a new cached web search service
func NewCachedWebSearchService(pool *pgxpool.Pool) *CachedWebSearchService {
	return &CachedWebSearchService{
		WebSearchService: NewWebSearchService(),
		pool:             pool,
		cacheTTL:         1 * time.Hour,
		newsCacheTTL:     15 * time.Minute,
	}
}

// NewCachedWebSearchServiceWithConfig creates a cached search service with explicit config
func NewCachedWebSearchServiceWithConfig(pool *pgxpool.Pool, cfg *config.Config) *CachedWebSearchService {
	return &CachedWebSearchService{
		WebSearchService: NewWebSearchServiceWithConfig(cfg),
		pool:             pool,
		cacheTTL:         1 * time.Hour,
		newsCacheTTL:     15 * time.Minute,
	}
}

// hashQuery creates a SHA256 hash of the normalized query
func (s *CachedWebSearchService) hashQuery(query string) string {
	return hashQueryString(query)
}

// cachedResult holds the structure of a cached search result
type cachedResult struct {
	ID            pgtype.UUID
	QueryHash     string
	OriginalQuery string
	Results       []byte
	ResultCount   int32
	Provider      string
	HitCount      int32
}

// SearchWithCache performs a search with caching
func (s *CachedWebSearchService) SearchWithCache(ctx context.Context, query string, maxResults int, userID string, conversationID *uuid.UUID) (*WebSearchResponse, error) {
	if s.pool == nil {
		// No pool available, fall back to direct search
		return s.Search(ctx, query, maxResults)
	}

	queryHash := s.hashQuery(query)

	// Try to get cached result
	cached, err := s.getCachedResult(ctx, queryHash, conversationID)
	if err == nil && cached != nil {
		slog.Debug("Web search cache hit", "queryHash", queryHash, "hitCount", cached.HitCount)

		// Increment hit count asynchronously
		go func() {
			_, _ = s.pool.Exec(context.Background(),
				"UPDATE web_search_results SET hit_count = hit_count + 1, last_hit_at = NOW() WHERE id = $1",
				cached.ID)
		}()

		// Parse cached results
		var results []WebSearchResult
		if err := json.Unmarshal(cached.Results, &results); err == nil {
			return &WebSearchResponse{
				Query:        cached.OriginalQuery,
				Results:      results,
				TotalResults: int(cached.ResultCount),
				SearchTime:   0, // Cached, no search time
				Provider:     cached.Provider,
				Cached:       true,
			}, nil
		}
	}

	// Cache miss - perform search
	response, err := s.SearchWithOptimization(ctx, query, maxResults)
	if err != nil {
		return nil, err
	}

	// Save to cache asynchronously
	go func() {
		s.saveToCache(context.Background(), query, response, userID, conversationID)
	}()

	return response, nil
}

// getCachedResult retrieves a cached result from the database
func (s *CachedWebSearchService) getCachedResult(ctx context.Context, queryHash string, conversationID *uuid.UUID) (*cachedResult, error) {
	var query string
	var args []interface{}

	if conversationID != nil {
		query = `SELECT id, query_hash, original_query, results, result_count, provider, hit_count
				 FROM web_search_results
				 WHERE query_hash = $1 AND conversation_id = $2 AND expires_at > NOW()
				 ORDER BY created_at DESC LIMIT 1`
		args = []interface{}{queryHash, *conversationID}
	} else {
		query = `SELECT id, query_hash, original_query, results, result_count, provider, hit_count
				 FROM web_search_results
				 WHERE query_hash = $1 AND expires_at > NOW()
				 ORDER BY created_at DESC LIMIT 1`
		args = []interface{}{queryHash}
	}

	row := s.pool.QueryRow(ctx, query, args...)

	var result cachedResult
	var provider *string
	err := row.Scan(&result.ID, &result.QueryHash, &result.OriginalQuery, &result.Results, &result.ResultCount, &provider, &result.HitCount)
	if err != nil {
		return nil, err
	}

	if provider != nil {
		result.Provider = *provider
	} else {
		result.Provider = "unknown"
	}

	return &result, nil
}

// saveToCache saves search results to the cache
func (s *CachedWebSearchService) saveToCache(ctx context.Context, originalQuery string, response *WebSearchResponse, userID string, conversationID *uuid.UUID) {
	queryHash := s.hashQuery(originalQuery)

	// Determine TTL based on query type
	optimizer := NewQueryOptimizer()
	queryType := optimizer.DetectQueryType(originalQuery)
	ttl := s.cacheTTL
	if queryType == QueryTypeNews {
		ttl = s.newsCacheTTL
	}

	// Serialize results
	resultsJSON, err := json.Marshal(response.Results)
	if err != nil {
		slog.Error("Failed to marshal search results for cache", "err", err)
		return
	}

	expiresAt := time.Now().Add(ttl)

	query := `INSERT INTO web_search_results (
		query_hash, original_query, optimized_query, user_id, conversation_id,
		results, result_count, provider, search_time_ms, expires_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	_, err = s.pool.Exec(ctx, query,
		queryHash,
		originalQuery,
		response.Query, // optimized query
		userIDPtr,
		conversationID,
		resultsJSON,
		response.TotalResults,
		response.Provider,
		response.SearchTime,
		expiresAt,
	)

	if err != nil {
		slog.Error("Failed to save search result to cache", "err", err)
	} else {
		slog.Debug("Saved search result to cache", "queryHash", queryHash, "provider", response.Provider, "ttl", ttl)
	}
}

// CleanupCache removes expired cache entries
func (s *CachedWebSearchService) CleanupCache(ctx context.Context) (int64, error) {
	if s.pool == nil {
		return 0, nil
	}

	result, err := s.pool.Exec(ctx, "DELETE FROM web_search_results WHERE expires_at < NOW()")
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
