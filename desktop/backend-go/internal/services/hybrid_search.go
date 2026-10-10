package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// HybridSearchService combines semantic and keyword search with RRF
// Implements SORX 2.0 Hybrid Search SKILL
type HybridSearchService struct {
	pool         *pgxpool.Pool
	embeddingSvc *EmbeddingService
}

// HybridSearchOptions configures hybrid search behavior
type HybridSearchOptions struct {
	SemanticWeight float64 // 0.0 to 1.0, weight for semantic search
	KeywordWeight  float64 // 0.0 to 1.0, weight for keyword search
	MaxResults     int     // Maximum number of results to return
	RRFConstant    int     // RRF k constant (typically 60)
	MinSimilarity  float64 // Minimum similarity threshold (0.0 to 1.0)
}

// HybridSearchResult represents a result from hybrid search
type HybridSearchResult struct {
	ContextID      uuid.UUID  `json:"context_id"`
	BlockID        string     `json:"block_id"`
	BlockType      string     `json:"block_type"`
	Content        string     `json:"content"`
	ContextName    string     `json:"context_name"`
	ContextType    string     `json:"context_type"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	SemanticScore  float64    `json:"semantic_score"`  // Cosine similarity
	KeywordScore   float64    `json:"keyword_score"`   // Text search rank
	HybridScore    float64    `json:"hybrid_score"`    // Combined RRF score
	SearchStrategy string     `json:"search_strategy"` // "semantic", "keyword", or "hybrid"
}

// DefaultHybridSearchOptions returns sensible defaults
func DefaultHybridSearchOptions() HybridSearchOptions {
	return HybridSearchOptions{
		SemanticWeight: 0.7,  // Favor semantic understanding
		KeywordWeight:  0.3,  // But include keyword matches
		MaxResults:     10,
		RRFConstant:    60,   // Standard RRF constant
		MinSimilarity:  0.3,  // Filter out very weak matches
	}
}

// NewHybridSearchService creates a new hybrid search service
func NewHybridSearchService(pool *pgxpool.Pool, embeddingSvc *EmbeddingService) *HybridSearchService {
	return &HybridSearchService{
		pool:         pool,
		embeddingSvc: embeddingSvc,
	}
}

// Search performs hybrid search combining semantic and keyword approaches
func (h *HybridSearchService) Search(ctx context.Context, query string, userID string, opts HybridSearchOptions) ([]HybridSearchResult, error) {
	// Validate options
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}
	if opts.RRFConstant <= 0 {
		opts.RRFConstant = 60
	}

	// Normalize weights
	totalWeight := opts.SemanticWeight + opts.KeywordWeight
	if totalWeight == 0 {
		return nil, fmt.Errorf("at least one search weight must be > 0")
	}
	opts.SemanticWeight = opts.SemanticWeight / totalWeight
	opts.KeywordWeight = opts.KeywordWeight / totalWeight

	// Run both searches in parallel
	type searchResult struct {
		semantic []semanticResult
		keyword  []keywordResult
		err      error
	}

	resultChan := make(chan searchResult, 1)

	go func() {
		var result searchResult

		// Semantic search
		if opts.SemanticWeight > 0 {
			result.semantic, result.err = h.semanticSearch(ctx, query, userID, opts.MaxResults*2)
			if result.err != nil {
				resultChan <- result
				return
			}
		}

		// Keyword search
		if opts.KeywordWeight > 0 {
			result.keyword, result.err = h.keywordSearch(ctx, query, userID, opts.MaxResults*2)
			if result.err != nil {
				resultChan <- result
				return
			}
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.err != nil {
		return nil, result.err
	}

	// Combine results using Reciprocal Rank Fusion (RRF)
	combined := h.fuseResults(result.semantic, result.keyword, opts)

	// Apply minimum similarity filter and limit results
	var filtered []HybridSearchResult
	for _, r := range combined {
		if r.HybridScore >= opts.MinSimilarity {
			filtered = append(filtered, r)
		}
		if len(filtered) >= opts.MaxResults {
			break
		}
	}

	return filtered, nil
}

// semanticResult represents a result from semantic search
type semanticResult struct {
	ContextID   uuid.UUID
	BlockID     string
	BlockType   string
	Content     string
	ContextName string
	ContextType string
	ParentID    *uuid.UUID
	Similarity  float64
	Rank        int
}

// keywordResult represents a result from keyword search
type keywordResult struct {
	ContextID   uuid.UUID
	BlockID     string
	BlockType   string
	Content     string
	ContextName string
	ContextType string
	ParentID    *uuid.UUID
	Rank        float64
	Position    int
}

// semanticSearch performs vector similarity search
func (h *HybridSearchService) semanticSearch(ctx context.Context, query string, userID string, limit int) ([]semanticResult, error) {
	// Generate query embedding
	queryEmbedding, err := h.embeddingSvc.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	vec := pgvector.NewVector(queryEmbedding)

	// Query with cosine similarity
	rows, err := h.pool.Query(ctx, `
		SELECT
			ce.context_id,
			ce.block_id,
			ce.block_type,
			ce.content,
			c.name as context_name,
			c.type::text as context_type,
			c.parent_id,
			1 - (ce.embedding <=> $1) as similarity
		FROM context_embeddings ce
		JOIN contexts c ON c.id = ce.context_id
		WHERE c.user_id = $2
		  AND c.is_archived = false
		ORDER BY ce.embedding <=> $1
		LIMIT $3
	`, vec, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search query: %w", err)
	}
	defer rows.Close()

	var results []semanticResult
	rank := 1
	for rows.Next() {
		var r semanticResult
		var parentID uuid.NullUUID

		err := rows.Scan(
			&r.ContextID,
			&r.BlockID,
			&r.BlockType,
			&r.Content,
			&r.ContextName,
			&r.ContextType,
			&parentID,
			&r.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan semantic result: %w", err)
		}

		if parentID.Valid {
			r.ParentID = &parentID.UUID
		}

		r.Rank = rank
		rank++

		results = append(results, r)
	}

	return results, rows.Err()
}

// keywordSearch performs full-text keyword search using PostgreSQL tsvector
func (h *HybridSearchService) keywordSearch(ctx context.Context, query string, userID string, limit int) ([]keywordResult, error) {
	// Prepare query for PostgreSQL full-text search
	// Convert "machine learning model" -> "machine & learning & model"
	keywords := strings.Fields(strings.ToLower(query))
	tsQuery := strings.Join(keywords, " & ")
	if tsQuery == "" {
		tsQuery = query
	}

	// Query using PostgreSQL full-text search
	rows, err := h.pool.Query(ctx, `
		SELECT
			ce.context_id,
			ce.block_id,
			ce.block_type,
			ce.content,
			c.name as context_name,
			c.type::text as context_type,
			c.parent_id,
			ts_rank(to_tsvector('english', ce.content), to_tsquery('english', $1)) as rank
		FROM context_embeddings ce
		JOIN contexts c ON c.id = ce.context_id
		WHERE c.user_id = $2
		  AND c.is_archived = false
		  AND to_tsvector('english', ce.content) @@ to_tsquery('english', $1)
		ORDER BY rank DESC
		LIMIT $3
	`, tsQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search query: %w", err)
	}
	defer rows.Close()

	var results []keywordResult
	position := 1
	for rows.Next() {
		var r keywordResult
		var parentID uuid.NullUUID

		err := rows.Scan(
			&r.ContextID,
			&r.BlockID,
			&r.BlockType,
			&r.Content,
			&r.ContextName,
			&r.ContextType,
			&parentID,
			&r.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("scan keyword result: %w", err)
		}

		if parentID.Valid {
			r.ParentID = &parentID.UUID
		}

		r.Position = position
		position++

		results = append(results, r)
	}

	return results, rows.Err()
}

// fuseResults combines semantic and keyword results using Reciprocal Rank Fusion
// RRF formula: score(d) = Σ 1/(k + rank_i(d))
// where k is typically 60, and rank_i is the rank in each ranking
func (h *HybridSearchService) fuseResults(semantic []semanticResult, keyword []keywordResult, opts HybridSearchOptions) []HybridSearchResult {
	// Create a map to combine results by unique identifier
	type resultKey struct {
		ContextID uuid.UUID
		BlockID   string
	}

	fusedMap := make(map[resultKey]*HybridSearchResult)

	// Process semantic results
	for _, s := range semantic {
		key := resultKey{ContextID: s.ContextID, BlockID: s.BlockID}

		// Calculate RRF score for semantic ranking
		rrfScore := 1.0 / float64(opts.RRFConstant+s.Rank)
		weightedScore := rrfScore * opts.SemanticWeight

		fusedMap[key] = &HybridSearchResult{
			ContextID:      s.ContextID,
			BlockID:        s.BlockID,
			BlockType:      s.BlockType,
			Content:        s.Content,
			ContextName:    s.ContextName,
			ContextType:    s.ContextType,
			ParentID:       s.ParentID,
			SemanticScore:  s.Similarity,
			KeywordScore:   0,
			HybridScore:    weightedScore,
			SearchStrategy: "semantic",
		}
	}

	// Process keyword results and merge
	for _, k := range keyword {
		key := resultKey{ContextID: k.ContextID, BlockID: k.BlockID}

		// Calculate RRF score for keyword ranking
		rrfScore := 1.0 / float64(opts.RRFConstant+k.Position)
		weightedScore := rrfScore * opts.KeywordWeight

		if existing, exists := fusedMap[key]; exists {
			// Merge with existing semantic result
			existing.KeywordScore = k.Rank
			existing.HybridScore += weightedScore
			existing.SearchStrategy = "hybrid"
		} else {
			// New result from keyword only
			fusedMap[key] = &HybridSearchResult{
				ContextID:      k.ContextID,
				BlockID:        k.BlockID,
				BlockType:      k.BlockType,
				Content:        k.Content,
				ContextName:    k.ContextName,
				ContextType:    k.ContextType,
				ParentID:       k.ParentID,
				SemanticScore:  0,
				KeywordScore:   k.Rank,
				HybridScore:    weightedScore,
				SearchStrategy: "keyword",
			}
		}
	}

	// Convert map to sorted slice
	var results []HybridSearchResult
	for _, r := range fusedMap {
		results = append(results, *r)
	}

	// Sort by hybrid score (descending)
	// Using a simple bubble sort for small result sets
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].HybridScore > results[i].HybridScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Normalize hybrid scores to 0-1 range for consistency
	if len(results) > 0 {
		maxScore := results[0].HybridScore
		if maxScore > 0 {
			for i := range results {
				results[i].HybridScore = results[i].HybridScore / maxScore
			}
		}
	}

	return results
}

// ExplainSearch returns debug information about search strategy
func (h *HybridSearchService) ExplainSearch(ctx context.Context, query string, userID string, opts HybridSearchOptions) (map[string]interface{}, error) {
	results, err := h.Search(ctx, query, userID, opts)
	if err != nil {
		return nil, err
	}

	// Count results by strategy
	strategyCounts := make(map[string]int)
	var avgSemanticScore, avgKeywordScore, avgHybridScore float64

	for _, r := range results {
		strategyCounts[r.SearchStrategy]++
		avgSemanticScore += r.SemanticScore
		avgKeywordScore += r.KeywordScore
		avgHybridScore += r.HybridScore
	}

	count := float64(len(results))
	if count > 0 {
		avgSemanticScore /= count
		avgKeywordScore /= count
		avgHybridScore /= count
	}

	return map[string]interface{}{
		"query":               query,
		"total_results":       len(results),
		"strategy_breakdown":  strategyCounts,
		"avg_semantic_score":  math.Round(avgSemanticScore*1000) / 1000,
		"avg_keyword_score":   math.Round(avgKeywordScore*1000) / 1000,
		"avg_hybrid_score":    math.Round(avgHybridScore*1000) / 1000,
		"options": map[string]interface{}{
			"semantic_weight": opts.SemanticWeight,
			"keyword_weight":  opts.KeywordWeight,
			"rrf_constant":    opts.RRFConstant,
			"min_similarity":  opts.MinSimilarity,
		},
		"top_5_results": results[:min(5, len(results))],
	}, nil
}
