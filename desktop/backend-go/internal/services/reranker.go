package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReRankerService implements intelligent re-ranking of search results
// Implements SORX 2.0 Re-Ranking SKILL
type ReRankerService struct {
	pool         *pgxpool.Pool
	embeddingSvc *EmbeddingService
}

// ReRankingOptions configures re-ranking behavior
type ReRankingOptions struct {
	// Scoring weights (should sum to 1.0)
	SemanticWeight      float64 // Weight for semantic similarity
	RecencyWeight       float64 // Weight for content recency
	QualityWeight       float64 // Weight for content quality signals
	InteractionWeight   float64 // Weight for user interaction history
	ContextRelevance    float64 // Weight for contextual relevance

	// Recency parameters
	RecencyDecayDays    int     // Days after which recency score starts decaying
	RecencyHalfLife     int     // Days for recency score to decay to 50%

	// Quality signals
	MinContentLength    int     // Minimum content length for quality
	PreferredBlockTypes []string // Preferred block types (e.g., "heading", "paragraph")

	// Context
	CurrentProjectID    *uuid.UUID // Current project context
	CurrentTaskID       *uuid.UUID // Current task context
}

// ReRankedResult represents a result after re-ranking
type ReRankedResult struct {
	HybridSearchResult              // Embed original result

	// Re-ranking scores
	RecencyScore      float64 `json:"recency_score"`
	QualityScore      float64 `json:"quality_score"`
	InteractionScore  float64 `json:"interaction_score"`
	ContextScore      float64 `json:"context_score"`
	FinalScore        float64 `json:"final_score"`

	// Metadata
	OriginalRank      int     `json:"original_rank"`
	ReRankedPosition  int     `json:"reranked_position"`
	RankChange        int     `json:"rank_change"` // Positive = moved up

	// Explanation
	ScoreBreakdown    map[string]float64 `json:"score_breakdown"`
}

// DefaultReRankingOptions returns sensible defaults
func DefaultReRankingOptions() ReRankingOptions {
	return ReRankingOptions{
		SemanticWeight:    0.4,  // Semantic similarity is important
		RecencyWeight:     0.2,  // Recent content is more relevant
		QualityWeight:     0.2,  // Quality matters
		InteractionWeight: 0.1,  // User history provides signals
		ContextRelevance:  0.1,  // Context awareness

		RecencyDecayDays: 30,
		RecencyHalfLife:  90,

		MinContentLength:    50,
		PreferredBlockTypes: []string{"heading", "paragraph", "list"},
	}
}

// NewReRankerService creates a new re-ranking service
func NewReRankerService(pool *pgxpool.Pool, embeddingSvc *EmbeddingService) *ReRankerService {
	return &ReRankerService{
		pool:         pool,
		embeddingSvc: embeddingSvc,
	}
}

// ReRank improves the ranking of search results using multiple signals
func (r *ReRankerService) ReRank(ctx context.Context, query string, userID string, results []HybridSearchResult, opts ReRankingOptions) ([]ReRankedResult, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Normalize weights
	totalWeight := opts.SemanticWeight + opts.RecencyWeight + opts.QualityWeight + opts.InteractionWeight + opts.ContextRelevance
	if totalWeight == 0 {
		return nil, fmt.Errorf("at least one re-ranking weight must be > 0")
	}
	opts.SemanticWeight /= totalWeight
	opts.RecencyWeight /= totalWeight
	opts.QualityWeight /= totalWeight
	opts.InteractionWeight /= totalWeight
	opts.ContextRelevance /= totalWeight

	// Gather metadata for re-ranking
	metadata, err := r.gatherMetadata(ctx, results, userID)
	if err != nil {
		return nil, fmt.Errorf("gather metadata: %w", err)
	}

	// Calculate scores for each result
	reranked := make([]ReRankedResult, len(results))
	for i, result := range results {
		rr := ReRankedResult{
			HybridSearchResult: result,
			OriginalRank:       i + 1,
			ScoreBreakdown:     make(map[string]float64),
		}

		// 1. Semantic score (already have from hybrid search)
		semanticScore := result.HybridScore
		rr.ScoreBreakdown["semantic"] = semanticScore

		// 2. Recency score
		recencyScore := r.calculateRecencyScore(metadata[result.ContextID], opts)
		rr.RecencyScore = recencyScore
		rr.ScoreBreakdown["recency"] = recencyScore

		// 3. Quality score
		qualityScore := r.calculateQualityScore(result, opts)
		rr.QualityScore = qualityScore
		rr.ScoreBreakdown["quality"] = qualityScore

		// 4. Interaction score
		interactionScore := r.calculateInteractionScore(metadata[result.ContextID])
		rr.InteractionScore = interactionScore
		rr.ScoreBreakdown["interaction"] = interactionScore

		// 5. Context relevance score
		contextScore := r.calculateContextScore(metadata[result.ContextID], opts)
		rr.ContextScore = contextScore
		rr.ScoreBreakdown["context"] = contextScore

		// Calculate weighted final score
		rr.FinalScore = (semanticScore * opts.SemanticWeight) +
			(recencyScore * opts.RecencyWeight) +
			(qualityScore * opts.QualityWeight) +
			(interactionScore * opts.InteractionWeight) +
			(contextScore * opts.ContextRelevance)

		reranked[i] = rr
	}

	// Sort by final score (descending)
	for i := 0; i < len(reranked)-1; i++ {
		for j := i + 1; j < len(reranked); j++ {
			if reranked[j].FinalScore > reranked[i].FinalScore {
				reranked[i], reranked[j] = reranked[j], reranked[i]
			}
		}
	}

	// Update positions and rank changes
	for i := range reranked {
		reranked[i].ReRankedPosition = i + 1
		reranked[i].RankChange = reranked[i].OriginalRank - reranked[i].ReRankedPosition
	}

	return reranked, nil
}

// contextMetadata holds metadata about a context for re-ranking
type contextMetadata struct {
	ContextID      uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AccessCount    int
	LastAccessedAt *time.Time
	ProjectID      *uuid.UUID
	ParentID       *uuid.UUID
}

// gatherMetadata fetches metadata needed for re-ranking
func (r *ReRankerService) gatherMetadata(ctx context.Context, results []HybridSearchResult, userID string) (map[uuid.UUID]contextMetadata, error) {
	// Extract unique context IDs
	contextIDs := make([]uuid.UUID, 0, len(results))
	seen := make(map[uuid.UUID]bool)
	for _, result := range results {
		if !seen[result.ContextID] {
			contextIDs = append(contextIDs, result.ContextID)
			seen[result.ContextID] = true
		}
	}

	if len(contextIDs) == 0 {
		return make(map[uuid.UUID]contextMetadata), nil
	}

	// Query metadata
	rows, err := r.pool.Query(ctx, `
		SELECT id, created_at, updated_at, access_count, last_accessed_at, project_id, parent_id
		FROM contexts
		WHERE id = ANY($1) AND user_id = $2
	`, contextIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("query metadata: %w", err)
	}
	defer rows.Close()

	metadata := make(map[uuid.UUID]contextMetadata)
	for rows.Next() {
		var m contextMetadata
		var lastAccessed *time.Time
		var projectID, parentID uuid.NullUUID

		err := rows.Scan(
			&m.ContextID,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.AccessCount,
			&lastAccessed,
			&projectID,
			&parentID,
		)
		if err != nil {
			continue
		}

		m.LastAccessedAt = lastAccessed
		if projectID.Valid {
			m.ProjectID = &projectID.UUID
		}
		if parentID.Valid {
			m.ParentID = &parentID.UUID
		}

		metadata[m.ContextID] = m
	}

	return metadata, rows.Err()
}

// calculateRecencyScore scores based on content freshness
// Uses exponential decay: score = e^(-λt) where λ = ln(2)/halfLife
func (r *ReRankerService) calculateRecencyScore(meta contextMetadata, opts ReRankingOptions) float64 {
	// Use updated_at as the reference time
	age := time.Since(meta.UpdatedAt)
	ageDays := age.Hours() / 24

	// If within decay threshold, score is 1.0
	if ageDays <= float64(opts.RecencyDecayDays) {
		return 1.0
	}

	// Apply exponential decay
	decayConstant := math.Log(2) / float64(opts.RecencyHalfLife)
	adjustedAge := ageDays - float64(opts.RecencyDecayDays)
	score := math.Exp(-decayConstant * adjustedAge)

	return math.Max(0.1, score) // Floor at 0.1 to avoid completely discarding old content
}

// calculateQualityScore scores based on content quality signals
func (r *ReRankerService) calculateQualityScore(result HybridSearchResult, opts ReRankingOptions) float64 {
	score := 0.5 // Baseline

	// 1. Content length (reasonable length is better)
	contentLength := len(result.Content)
	if contentLength >= opts.MinContentLength {
		// Optimal length is around 200-500 chars for a block
		if contentLength >= 200 && contentLength <= 500 {
			score += 0.3
		} else if contentLength >= 100 && contentLength < 200 {
			score += 0.2
		} else if contentLength > 500 && contentLength <= 1000 {
			score += 0.15
		} else if contentLength > 1000 {
			score += 0.1 // Very long blocks might be less focused
		} else {
			score += 0.1 // Short but above minimum
		}
	}

	// 2. Block type preference
	for _, preferredType := range opts.PreferredBlockTypes {
		if result.BlockType == preferredType {
			score += 0.2
			break
		}
	}

	return math.Min(1.0, score)
}

// calculateInteractionScore scores based on user interaction history
func (r *ReRankerService) calculateInteractionScore(meta contextMetadata) float64 {
	score := 0.0

	// Access count (logarithmic scale to prevent dominance)
	if meta.AccessCount > 0 {
		score += math.Log10(float64(meta.AccessCount)+1) / 3.0 // Normalized to ~0.0-0.5
	}

	// Recent access bonus
	if meta.LastAccessedAt != nil {
		daysSinceAccess := time.Since(*meta.LastAccessedAt).Hours() / 24
		if daysSinceAccess <= 7 {
			score += 0.3
		} else if daysSinceAccess <= 30 {
			score += 0.2
		} else if daysSinceAccess <= 90 {
			score += 0.1
		}
	}

	return math.Min(1.0, score)
}

// calculateContextScore scores based on contextual relevance
func (r *ReRankerService) calculateContextScore(meta contextMetadata, opts ReRankingOptions) float64 {
	score := 0.5 // Baseline

	// If result is from current project context
	if opts.CurrentProjectID != nil && meta.ProjectID != nil {
		if *opts.CurrentProjectID == *meta.ProjectID {
			score += 0.5
		}
	}

	// Future: Could expand with task context, workspace context, etc.

	return math.Min(1.0, score)
}

// ExplainReRanking provides detailed explanation of re-ranking decisions
func (r *ReRankerService) ExplainReRanking(ctx context.Context, query string, userID string, results []HybridSearchResult, opts ReRankingOptions) (map[string]interface{}, error) {
	reranked, err := r.ReRank(ctx, query, userID, results, opts)
	if err != nil {
		return nil, err
	}

	// Analyze rank changes
	var movedUp, movedDown, unchanged int
	var avgRankChange float64
	maxPositiveChange := 0
	maxNegativeChange := 0

	for _, r := range reranked {
		if r.RankChange > 0 {
			movedUp++
			if r.RankChange > maxPositiveChange {
				maxPositiveChange = r.RankChange
			}
		} else if r.RankChange < 0 {
			movedDown++
			if r.RankChange < maxNegativeChange {
				maxNegativeChange = r.RankChange
			}
		} else {
			unchanged++
		}
		avgRankChange += float64(r.RankChange)
	}

	if len(reranked) > 0 {
		avgRankChange /= float64(len(reranked))
	}

	return map[string]interface{}{
		"query":         query,
		"total_results": len(reranked),
		"rank_changes": map[string]interface{}{
			"moved_up":            movedUp,
			"moved_down":          movedDown,
			"unchanged":           unchanged,
			"avg_rank_change":     math.Round(avgRankChange*100) / 100,
			"max_positive_change": maxPositiveChange,
			"max_negative_change": maxNegativeChange,
		},
		"options": map[string]interface{}{
			"semantic_weight":    opts.SemanticWeight,
			"recency_weight":     opts.RecencyWeight,
			"quality_weight":     opts.QualityWeight,
			"interaction_weight": opts.InteractionWeight,
			"context_weight":     opts.ContextRelevance,
		},
		"top_5_reranked": reranked[:min(5, len(reranked))],
	}, nil
}
