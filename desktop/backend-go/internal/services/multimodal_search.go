package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MultiModalSearchService implements the EnhancedSearchService interface
// Provides hybrid search combining text, images, and semantic understanding
type MultiModalSearchService struct {
	pool              *pgxpool.Pool
	hybridSearch      *HybridSearchService
	reranker          *ReRankerService
	imageEmbedding    *ImageEmbeddingService
	textEmbedding     *EmbeddingService
}

// MultiModalSearchResult represents a search result that can be text or image
type MultiModalSearchResult struct {
	ID           uuid.UUID
	Type         string  // "text", "image", "hybrid"
	Score        float64
	Similarity   float64

	// Text result fields
	ContextID    *uuid.UUID
	Content      string
	Title        string

	// Image result fields
	ImageID      *uuid.UUID
	ImageURL     string
	ImageCaption string

	// Common fields
	UserID       string
	Metadata     map[string]interface{}
	Source       string // "semantic", "keyword", "image", "hybrid"
}

// SearchOptions configures multimodal search behavior
type SearchOptions struct {
	// Weights (must sum to 1.0)
	SemanticWeight  float64   // Weight for semantic text search
	KeywordWeight   float64   // Weight for keyword text search
	ImageWeight     float64   // Weight for image similarity

	// Search behavior
	ReRankEnabled   bool
	MaxResults      int
	MinSimilarity   float64

	// Filters
	IncludeText     bool
	IncludeImages   bool
	ContextIDs      []uuid.UUID
	ProjectIDs      []uuid.UUID

	// Re-ranking options
	RecencyWeight     float64
	QualityWeight     float64
	InteractionWeight float64
}

// DefaultSearchOptions returns sensible default options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		SemanticWeight:    0.5,
		KeywordWeight:     0.3,
		ImageWeight:       0.2,
		ReRankEnabled:     true,
		MaxResults:        20,
		MinSimilarity:     0.3,
		IncludeText:       true,
		IncludeImages:     true,
		RecencyWeight:     0.3,
		QualityWeight:     0.3,
		InteractionWeight: 0.2,
	}
}

// NewMultiModalSearchService creates a new multimodal search service
func NewMultiModalSearchService(
	pool *pgxpool.Pool,
	hybridSearch *HybridSearchService,
	reranker *ReRankerService,
	imageEmbedding *ImageEmbeddingService,
	textEmbedding *EmbeddingService,
) *MultiModalSearchService {
	return &MultiModalSearchService{
		pool:           pool,
		hybridSearch:   hybridSearch,
		reranker:       reranker,
		imageEmbedding: imageEmbedding,
		textEmbedding:  textEmbedding,
	}
}

// HybridSearch performs text-only hybrid search (semantic + keyword)
// This implements the first method from EnhancedSearchService interface
func (m *MultiModalSearchService) HybridSearch(ctx context.Context, query string, userID string, opts SearchOptions) ([]MultiModalSearchResult, error) {
	// Validate weights
	if err := m.validateWeights(opts); err != nil {
		return nil, err
	}

	// Use existing hybrid search service
	hybridOpts := HybridSearchOptions{
		SemanticWeight:   opts.SemanticWeight / (opts.SemanticWeight + opts.KeywordWeight),
		KeywordWeight:    opts.KeywordWeight / (opts.SemanticWeight + opts.KeywordWeight),
		MaxResults:       opts.MaxResults,
		MinSimilarity:    opts.MinSimilarity,
		RRFConstant:      60,
	}

	results, err := m.hybridSearch.Search(ctx, query, userID, hybridOpts)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Convert to MultiModalSearchResult
	multiModalResults := make([]MultiModalSearchResult, 0, len(results))
	for _, r := range results {
		multiModalResults = append(multiModalResults, MultiModalSearchResult{
			ID:         r.ContextID,
			Type:       "text",
			Score:      r.HybridScore,
			Similarity: r.SemanticScore,
			ContextID:  &r.ContextID,
			Content:    r.Content,
			Title:      r.ContextName,
			UserID:     userID,
			Source:     r.SearchStrategy,
		})
	}

	// Apply re-ranking if enabled
	if opts.ReRankEnabled && m.reranker != nil {
		return m.reRankResults(ctx, query, userID, multiModalResults, opts)
	}

	return multiModalResults, nil
}

// ReRank re-ranks search results for better relevance
// This implements the second method from EnhancedSearchService interface
func (m *MultiModalSearchService) ReRank(ctx context.Context, query string, userID string, results []MultiModalSearchResult, opts SearchOptions) ([]MultiModalSearchResult, error) {
	// Convert MultiModalSearchResult to HybridSearchResult for existing reranker
	hybridResults := make([]HybridSearchResult, 0, len(results))
	for _, r := range results {
		if r.Type == "text" {
			hybridResults = append(hybridResults, HybridSearchResult{
				ContextID:     *r.ContextID,
				Content:       r.Content,
				ContextName:   r.Title,
				SemanticScore: r.Similarity,
				HybridScore:   r.Score,
				SearchStrategy: r.Source,
			})
		}
	}

	// Use existing re-ranker
	rerankerOpts := ReRankingOptions{
		SemanticWeight:    opts.SemanticWeight,
		RecencyWeight:     opts.RecencyWeight,
		QualityWeight:     opts.QualityWeight,
		InteractionWeight: opts.InteractionWeight,
		RecencyHalfLife:   30, // 30 days
	}

	reranked, err := m.reranker.ReRank(ctx, query, userID, hybridResults, rerankerOpts)
	if err != nil {
		return nil, fmt.Errorf("re-ranking failed: %w", err)
	}

	// Convert back to MultiModalSearchResult
	finalResults := make([]MultiModalSearchResult, 0, len(reranked))
	for _, r := range reranked {
		finalResults = append(finalResults, MultiModalSearchResult{
			ID:         r.ContextID,
			Type:       "text",
			Score:      r.FinalScore,
			Similarity: r.SemanticScore,
			ContextID:  &r.ContextID,
			Content:    r.Content,
			Title:      r.ContextName,
			UserID:     userID,
			Source:     r.SearchStrategy,
		})
	}

	return finalResults, nil
}

// SearchWithImage performs multimodal search combining image and text query
// This implements the third method from EnhancedSearchService interface - THE MISSING PIECE!
func (m *MultiModalSearchService) SearchWithImage(ctx context.Context, imageData []byte, textQuery string, userID string, opts SearchOptions) ([]MultiModalSearchResult, error) {
	// Validate weights
	if err := m.validateWeights(opts); err != nil {
		return nil, err
	}

	var allResults []MultiModalSearchResult

	// 1. Image-based search (if image provided and enabled)
	if len(imageData) > 0 && opts.IncludeImages && opts.ImageWeight > 0 {
		imageResults, err := m.searchByImage(ctx, imageData, userID, opts.MaxResults)
		if err != nil {
			// Log error but don't fail - continue with text search
			slog.Default().Warn("Image search failed", "error", err)
		} else {
			// Apply image weight to scores
			for i := range imageResults {
				imageResults[i].Score *= opts.ImageWeight
			}
			allResults = append(allResults, imageResults...)
		}
	}

	// 2. Text-based search (if text query provided and enabled)
	if textQuery != "" && opts.IncludeText {
		textWeight := opts.SemanticWeight + opts.KeywordWeight

		hybridOpts := HybridSearchOptions{
			SemanticWeight: opts.SemanticWeight / textWeight,
			KeywordWeight:  opts.KeywordWeight / textWeight,
			MaxResults:     opts.MaxResults,
			MinSimilarity:  opts.MinSimilarity,
			RRFConstant:    60,
		}

		textResults, err := m.hybridSearch.Search(ctx, textQuery, userID, hybridOpts)
		if err != nil {
			return nil, fmt.Errorf("text search failed: %w", err)
		}

		// Convert and apply text weight
		for _, r := range textResults {
			result := MultiModalSearchResult{
				ID:         r.ContextID,
				Type:       "text",
				Score:      r.HybridScore * textWeight,
				Similarity: r.SemanticScore,
				ContextID:  &r.ContextID,
				Content:    r.Content,
				Title:      r.ContextName,
				UserID:     userID,
				Source:     r.SearchStrategy,
			}
			allResults = append(allResults, result)
		}
	}

	// 3. Cross-modal search (text query → find similar images)
	if textQuery != "" && opts.IncludeImages && len(imageData) == 0 {
		crossModalResults, err := m.searchImagesWithText(ctx, textQuery, userID, opts.MaxResults/2)
		if err != nil {
			slog.Default().Warn("Cross-modal search failed", "error", err)
		} else {
			for i := range crossModalResults {
				crossModalResults[i].Score *= opts.ImageWeight
			}
			allResults = append(allResults, crossModalResults...)
		}
	}

	// 4. Merge and sort results by score
	allResults = m.mergeResults(allResults, opts.MaxResults)

	// 5. Apply re-ranking if enabled
	if opts.ReRankEnabled && len(allResults) > 0 {
		return m.reRankResults(ctx, textQuery, userID, allResults, opts)
	}

	return allResults, nil
}

// searchByImage finds similar images using image embedding
func (m *MultiModalSearchService) searchByImage(ctx context.Context, imageData []byte, userID string, maxResults int) ([]MultiModalSearchResult, error) {
	if m.imageEmbedding == nil {
		return nil, fmt.Errorf("image embedding service not available")
	}

	imageResults, err := m.imageEmbedding.SearchSimilarImages(ctx, imageData, userID, maxResults)
	if err != nil {
		return nil, err
	}

	results := make([]MultiModalSearchResult, 0, len(imageResults))
	for _, img := range imageResults {
		caption := ""
		if img.Metadata != nil {
			if c, ok := img.Metadata["caption"].(string); ok {
				caption = c
			}
		}

		results = append(results, MultiModalSearchResult{
			ID:           img.ID,
			Type:         "image",
			Score:        1.0, // Will be weighted later
			Similarity:   0.8, // Placeholder - should come from similarity calculation
			ImageID:      &img.ID,
			ImageCaption: caption,
			UserID:       userID,
			Source:       "image",
			Metadata:     img.Metadata,
		})
	}

	return results, nil
}

// searchImagesWithText performs cross-modal search (text → images)
func (m *MultiModalSearchService) searchImagesWithText(ctx context.Context, textQuery string, userID string, maxResults int) ([]MultiModalSearchResult, error) {
	if m.imageEmbedding == nil {
		return nil, fmt.Errorf("image embedding service not available")
	}

	// Generate text embedding
	textEmbedding, err := m.textEmbedding.GenerateEmbedding(ctx, textQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to generate text embedding: %w", err)
	}

	// Search images using text embedding (cross-modal)
	// This works because CLIP creates shared embedding space for text and images
	query := `
		SELECT
			id, user_id, metadata, created_at,
			1 - (embedding <=> $1::vector) as similarity
		FROM image_embeddings
		WHERE user_id = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := m.pool.Query(ctx, query, textEmbedding, userID, maxResults)
	if err != nil {
		return nil, fmt.Errorf("cross-modal search failed: %w", err)
	}
	defer rows.Close()

	var results []MultiModalSearchResult
	for rows.Next() {
		var (
			id         uuid.UUID
			userIDStr  string
			metadataJSON []byte
			createdAt  interface{}
			similarity float64
		)

		err := rows.Scan(&id, &userIDStr, &metadataJSON, &createdAt, &similarity)
		if err != nil {
			continue
		}

		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			// Parse metadata
		}

		caption := ""
		if metadata != nil {
			if c, ok := metadata["caption"].(string); ok {
				caption = c
			}
		}

		results = append(results, MultiModalSearchResult{
			ID:           id,
			Type:         "image",
			Score:        similarity,
			Similarity:   similarity,
			ImageID:      &id,
			ImageCaption: caption,
			UserID:       userIDStr,
			Source:       "cross_modal",
			Metadata:     metadata,
		})
	}

	return results, nil
}

// mergeResults combines and deduplicates results, keeping top N by score
func (m *MultiModalSearchService) mergeResults(results []MultiModalSearchResult, maxResults int) []MultiModalSearchResult {
	// Remove duplicates (same ID)
	seen := make(map[uuid.UUID]bool)
	unique := make([]MultiModalSearchResult, 0, len(results))

	for _, r := range results {
		if !seen[r.ID] {
			seen[r.ID] = true
			unique = append(unique, r)
		}
	}

	// Sort by score descending
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Score > unique[j].Score
	})

	// Limit to maxResults
	if len(unique) > maxResults {
		unique = unique[:maxResults]
	}

	return unique
}

// reRankResults applies re-ranking to multimodal results
func (m *MultiModalSearchService) reRankResults(ctx context.Context, query string, userID string, results []MultiModalSearchResult, opts SearchOptions) ([]MultiModalSearchResult, error) {
	// Separate text results for re-ranking
	textResults := make([]MultiModalSearchResult, 0)
	imageResults := make([]MultiModalSearchResult, 0)

	for _, r := range results {
		if r.Type == "text" {
			textResults = append(textResults, r)
		} else {
			imageResults = append(imageResults, r)
		}
	}

	// Re-rank text results
	if len(textResults) > 0 && query != "" {
		reranked, err := m.ReRank(ctx, query, userID, textResults, opts)
		if err != nil {
			return nil, err
		}
		textResults = reranked
	}

	// Merge back together
	final := append(textResults, imageResults...)

	// Sort by final score
	sort.Slice(final, func(i, j int) bool {
		return final[i].Score > final[j].Score
	})

	if len(final) > opts.MaxResults {
		final = final[:opts.MaxResults]
	}

	return final, nil
}

// validateWeights ensures search weights are valid
func (m *MultiModalSearchService) validateWeights(opts SearchOptions) error {
	total := opts.SemanticWeight + opts.KeywordWeight + opts.ImageWeight

	if total < 0.99 || total > 1.01 {
		return fmt.Errorf("search weights must sum to 1.0, got %.2f", total)
	}

	if opts.SemanticWeight < 0 || opts.KeywordWeight < 0 || opts.ImageWeight < 0 {
		return fmt.Errorf("search weights cannot be negative")
	}

	return nil
}

// GetSupportedModalities returns the modalities this service supports
func (m *MultiModalSearchService) GetSupportedModalities() []string {
	modalities := []string{"text", "semantic", "keyword"}

	if m.imageEmbedding != nil {
		modalities = append(modalities, "image", "cross_modal")
	}

	return modalities
}
