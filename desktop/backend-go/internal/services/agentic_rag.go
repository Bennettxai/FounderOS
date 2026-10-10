package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AgenticRAGService implements intelligent, self-adaptive RAG
// Implements SORX 2.0 Agentic RAG SKILL
type AgenticRAGService struct {
	pool              *pgxpool.Pool
	hybridSearch      *HybridSearchService
	reranker          *ReRankerService
	embeddingSvc      *EmbeddingService
	learningSvc       *LearningService
	cache             *RAGCacheService        // Optional cache
	queryExpansion    *QueryExpansionService  // Optional query expansion
}

// QueryIntent represents the classified intent of a user query
type QueryIntent string

const (
	IntentFactualLookup    QueryIntent = "factual_lookup"     // Looking for specific facts
	IntentConceptualSearch QueryIntent = "conceptual_search"  // Exploring concepts/ideas
	IntentProcedural       QueryIntent = "procedural"         // How-to questions
	IntentComparison       QueryIntent = "comparison"         // Comparing things
	IntentRecent           QueryIntent = "recent"             // Looking for recent content
	IntentExhaustive       QueryIntent = "exhaustive"         // Comprehensive search
	IntentAmbiguous        QueryIntent = "ambiguous"          // Unclear intent
)

// SearchStrategy represents the chosen search approach
type SearchStrategy string

const (
	StrategySemanticOnly SearchStrategy = "semantic_only"
	StrategyKeywordOnly  SearchStrategy = "keyword_only"
	StrategyHybrid       SearchStrategy = "hybrid"
	StrategyMultiPass    SearchStrategy = "multi_pass"
)

// AgenticRAGRequest represents a request to the agentic RAG system
type AgenticRAGRequest struct {
	Query              string
	UserID             string
	MaxResults         int
	MinQualityScore    float64 // Minimum acceptable quality (0-1)
	ProjectContext     *uuid.UUID
	TaskContext        *uuid.UUID
	UsePersonalization bool
	QueryIntent        QueryIntent // Optional: can be pre-classified
}

// AgenticRAGResponse represents the response from agentic RAG
type AgenticRAGResponse struct {
	Results           []ReRankedResult `json:"results"`
	QueryIntent       QueryIntent      `json:"query_intent"`
	StrategyUsed      SearchStrategy   `json:"strategy_used"`
	StrategyReasoning string           `json:"strategy_reasoning"`
	QualityScore      float64          `json:"quality_score"`
	IterationCount    int              `json:"iteration_count"`
	Personalized      bool             `json:"personalized"`
	ProcessingTimeMs  int64            `json:"processing_time_ms"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// NewAgenticRAGService creates a new agentic RAG service
func NewAgenticRAGService(
	pool *pgxpool.Pool,
	hybridSearch *HybridSearchService,
	reranker *ReRankerService,
	embeddingSvc *EmbeddingService,
	learningSvc *LearningService,
) *AgenticRAGService {
	return &AgenticRAGService{
		pool:         pool,
		hybridSearch: hybridSearch,
		reranker:     reranker,
		embeddingSvc: embeddingSvc,
		learningSvc:  learningSvc,
	}
}

// SetCache sets the cache service
func (a *AgenticRAGService) SetCache(cache *RAGCacheService) {
	a.cache = cache
}

// SetQueryExpansion sets the query expansion service
func (a *AgenticRAGService) SetQueryExpansion(qe *QueryExpansionService) {
	a.queryExpansion = qe
}

// Retrieve performs intelligent, adaptive retrieval
func (a *AgenticRAGService) Retrieve(ctx context.Context, req AgenticRAGRequest) (*AgenticRAGResponse, error) {
	startTime := time.Now()

	// Default parameters
	if req.MaxResults == 0 {
		req.MaxResults = 10
	}
	if req.MinQualityScore == 0 {
		req.MinQualityScore = 0.5
	}

	// Check cache first
	if a.cache != nil {
		if cached, err := a.cache.GetAgenticRAGResponse(ctx, req); err == nil && cached != nil {
			cached.Response.Metadata["cache_hit"] = true
			cached.Response.Metadata["cached_at"] = cached.CachedAt
			return cached.Response, nil
		}
	}

	response := &AgenticRAGResponse{
		Metadata: make(map[string]interface{}),
	}

	// Expand query if expansion service available
	if a.queryExpansion != nil {
		expanded, _ := a.queryExpansion.Expand(ctx, req.Query, false) // Don't use LLM rewriting for now
		if expanded != nil && len(expanded.Expanded) > 0 {
			response.Metadata["query_expansion"] = map[string]interface{}{
				"original": expanded.Original,
				"expanded": expanded.Expanded,
			}
		}
	}

	// Step 1: Query Understanding
	intent := a.classifyQueryIntent(req.Query)
	response.QueryIntent = intent
	response.Metadata["intent_classification"] = intent

	// Step 2: Strategy Selection
	strategy, reasoning := a.selectStrategy(intent, req)
	response.StrategyUsed = strategy
	response.StrategyReasoning = reasoning

	// Step 3: Personalization (if enabled)
	var userPrefs *PersonalizationProfile
	if req.UsePersonalization && a.learningSvc != nil {
		userPrefs, _ = a.learningSvc.GetPersonalizationProfile(ctx, req.UserID)
		if userPrefs != nil {
			response.Personalized = true
			response.Metadata["user_preferences"] = map[string]interface{}{
				"preferred_tone":      userPrefs.PreferredTone,
				"preferred_verbosity": userPrefs.PreferredVerbosity,
				"expertise_areas":     userPrefs.ExpertiseAreas,
			}
		}
	}

	// Step 4: Execute Search Strategy (with retries if needed)
	var results []ReRankedResult
	var err error
	maxIterations := 3
	iteration := 0

	for iteration < maxIterations {
		iteration++

		// Execute strategy
		results, err = a.executeStrategy(ctx, strategy, req, userPrefs)
		if err != nil {
			return nil, fmt.Errorf("execute strategy (iteration %d): %w", iteration, err)
		}

		// Step 5: Self-Critique - evaluate result quality
		qualityScore := a.evaluateResultQuality(results, req)
		response.QualityScore = qualityScore

		// If quality is acceptable, we're done
		if qualityScore >= req.MinQualityScore || iteration >= maxIterations {
			response.Results = results
			response.IterationCount = iteration
			break
		}

		// Quality too low, try alternative strategy
		strategy = a.selectFallbackStrategy(strategy, qualityScore)
		response.Metadata[fmt.Sprintf("iteration_%d_quality", iteration)] = qualityScore
		response.Metadata[fmt.Sprintf("iteration_%d_strategy", iteration)] = strategy
	}

	// Record processing time
	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	return response, nil
}

// classifyQueryIntent analyzes the query to understand user intent
func (a *AgenticRAGService) classifyQueryIntent(query string) QueryIntent {
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	// Pattern matching for intent classification
	// In production, this could use a small LLM or ML classifier

	// Factual lookup indicators
	factualKeywords := []string{"what is", "define", "meaning of", "who is", "where is", "when did"}
	for _, keyword := range factualKeywords {
		if strings.Contains(queryLower, keyword) {
			return IntentFactualLookup
		}
	}

	// Procedural indicators
	proceduralKeywords := []string{"how to", "how do i", "steps to", "guide for", "tutorial"}
	for _, keyword := range proceduralKeywords {
		if strings.Contains(queryLower, keyword) {
			return IntentProcedural
		}
	}

	// Comparison indicators
	comparisonKeywords := []string{"vs", "versus", "compare", "difference between", "better than"}
	for _, keyword := range comparisonKeywords {
		if strings.Contains(queryLower, keyword) {
			return IntentComparison
		}
	}

	// Recent content indicators
	recentKeywords := []string{"recent", "latest", "new", "today", "this week", "yesterday"}
	for _, keyword := range recentKeywords {
		if strings.Contains(queryLower, keyword) {
			return IntentRecent
		}
	}

	// Exhaustive indicators
	exhaustiveKeywords := []string{"all", "everything about", "comprehensive", "complete"}
	for _, keyword := range exhaustiveKeywords {
		if strings.Contains(queryLower, keyword) {
			return IntentExhaustive
		}
	}

	// Very short or single-word queries are often ambiguous
	if len(words) <= 2 {
		return IntentAmbiguous
	}

	// Default to conceptual search
	return IntentConceptualSearch
}

// selectStrategy chooses the best search strategy based on intent
func (a *AgenticRAGService) selectStrategy(intent QueryIntent, req AgenticRAGRequest) (SearchStrategy, string) {
	switch intent {
	case IntentFactualLookup:
		return StrategyKeywordOnly, "Factual queries benefit from exact keyword matching"

	case IntentConceptualSearch:
		return StrategySemanticOnly, "Conceptual exploration works best with semantic understanding"

	case IntentProcedural:
		return StrategyHybrid, "How-to queries need both semantic understanding and keyword precision"

	case IntentComparison:
		return StrategyHybrid, "Comparisons require finding related concepts (semantic) and specific terms (keyword)"

	case IntentRecent:
		return StrategyHybrid, "Recent content search benefits from hybrid approach with recency boost"

	case IntentExhaustive:
		return StrategyMultiPass, "Comprehensive search requires multiple strategies to ensure coverage"

	case IntentAmbiguous:
		return StrategyHybrid, "Ambiguous queries use hybrid search to cast a wider net"

	default:
		return StrategyHybrid, "Default to hybrid search for balanced results"
	}
}

// executeStrategy runs the chosen search strategy
func (a *AgenticRAGService) executeStrategy(
	ctx context.Context,
	strategy SearchStrategy,
	req AgenticRAGRequest,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	switch strategy {
	case StrategySemanticOnly:
		return a.executeSemanticSearch(ctx, req, userPrefs)

	case StrategyKeywordOnly:
		return a.executeKeywordSearch(ctx, req, userPrefs)

	case StrategyHybrid:
		return a.executeHybridSearch(ctx, req, userPrefs)

	case StrategyMultiPass:
		return a.executeMultiPassSearch(ctx, req, userPrefs)

	default:
		return a.executeHybridSearch(ctx, req, userPrefs)
	}
}

// executeSemanticSearch performs semantic-only search
func (a *AgenticRAGService) executeSemanticSearch(
	ctx context.Context,
	req AgenticRAGRequest,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	opts := DefaultHybridSearchOptions()
	opts.SemanticWeight = 1.0
	opts.KeywordWeight = 0.0
	opts.MaxResults = req.MaxResults * 2 // Get more for re-ranking

	results, err := a.hybridSearch.Search(ctx, req.Query, req.UserID, opts)
	if err != nil {
		return nil, err
	}

	// Re-rank
	return a.rerank(ctx, req, results, userPrefs)
}

// executeKeywordSearch performs keyword-only search
func (a *AgenticRAGService) executeKeywordSearch(
	ctx context.Context,
	req AgenticRAGRequest,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	opts := DefaultHybridSearchOptions()
	opts.SemanticWeight = 0.0
	opts.KeywordWeight = 1.0
	opts.MaxResults = req.MaxResults * 2

	results, err := a.hybridSearch.Search(ctx, req.Query, req.UserID, opts)
	if err != nil {
		return nil, err
	}

	return a.rerank(ctx, req, results, userPrefs)
}

// executeHybridSearch performs balanced hybrid search
func (a *AgenticRAGService) executeHybridSearch(
	ctx context.Context,
	req AgenticRAGRequest,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	opts := DefaultHybridSearchOptions()
	opts.MaxResults = req.MaxResults * 2

	// Adjust weights based on query intent
	if req.QueryIntent == IntentRecent {
		opts.SemanticWeight = 0.5
		opts.KeywordWeight = 0.5
	}

	results, err := a.hybridSearch.Search(ctx, req.Query, req.UserID, opts)
	if err != nil {
		return nil, err
	}

	return a.rerank(ctx, req, results, userPrefs)
}

// executeMultiPassSearch performs multiple search passes and combines results
func (a *AgenticRAGService) executeMultiPassSearch(
	ctx context.Context,
	req AgenticRAGRequest,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	// Pass 1: Semantic search
	semanticResults, err := a.executeSemanticSearch(ctx, req, userPrefs)
	if err != nil {
		return nil, err
	}

	// Pass 2: Keyword search
	keywordResults, err := a.executeKeywordSearch(ctx, req, userPrefs)
	if err != nil {
		return nil, err
	}

	// Combine and deduplicate
	seen := make(map[string]bool)
	var combined []ReRankedResult

	// Add semantic results first (prioritize semantic understanding)
	for _, r := range semanticResults {
		key := r.ContextID.String() + ":" + r.BlockID
		if !seen[key] {
			combined = append(combined, r)
			seen[key] = true
		}
	}

	// Add keyword results
	for _, r := range keywordResults {
		key := r.ContextID.String() + ":" + r.BlockID
		if !seen[key] {
			combined = append(combined, r)
			seen[key] = true
		}
	}

	// Limit to requested max
	if len(combined) > req.MaxResults {
		combined = combined[:req.MaxResults]
	}

	return combined, nil
}

// rerank applies re-ranking to results
func (a *AgenticRAGService) rerank(
	ctx context.Context,
	req AgenticRAGRequest,
	results []HybridSearchResult,
	userPrefs *PersonalizationProfile,
) ([]ReRankedResult, error) {

	opts := DefaultReRankingOptions()

	// Adjust re-ranking based on intent
	switch req.QueryIntent {
	case IntentRecent:
		opts.RecencyWeight = 0.4 // Boost recency
		opts.SemanticWeight = 0.3
	case IntentFactualLookup:
		opts.QualityWeight = 0.4 // Boost quality for facts
		opts.SemanticWeight = 0.3
	}

	// Add context if available
	opts.CurrentProjectID = req.ProjectContext

	// Personalize weights if user preferences available
	if userPrefs != nil {
		// Users who prefer concise content might prefer quality over quantity
		if userPrefs.PreferredVerbosity == "concise" {
			opts.QualityWeight += 0.1
			opts.SemanticWeight -= 0.1
		}
	}

	reranked, err := a.reranker.ReRank(ctx, req.Query, req.UserID, results, opts)
	if err != nil {
		return nil, err
	}

	// Limit to requested max
	if len(reranked) > req.MaxResults {
		reranked = reranked[:req.MaxResults]
	}

	return reranked, nil
}

// evaluateResultQuality assesses the quality of retrieved results
func (a *AgenticRAGService) evaluateResultQuality(results []ReRankedResult, req AgenticRAGRequest) float64 {
	if len(results) == 0 {
		return 0.0
	}

	// Quality metrics:
	// 1. Average final score of top results
	// 2. Result count (did we get enough results?)
	// 3. Score distribution (are results consistently good?)

	var totalScore float64
	minScore := 1.0
	maxScore := 0.0

	topN := min(5, len(results))
	for i := 0; i < topN; i++ {
		score := results[i].FinalScore
		totalScore += score
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	avgScore := totalScore / float64(topN)

	// Penalty if we didn't get enough results
	resultPenalty := 1.0
	if len(results) < req.MaxResults/2 {
		resultPenalty = float64(len(results)) / float64(req.MaxResults/2)
	}

	// Consistency bonus (higher if scores are consistently good)
	consistency := 1.0 - (maxScore - minScore)

	// Combined quality score
	quality := (avgScore * 0.6) + (resultPenalty * 0.2) + (consistency * 0.2)

	return quality
}

// selectFallbackStrategy chooses a fallback strategy when quality is low
func (a *AgenticRAGService) selectFallbackStrategy(currentStrategy SearchStrategy, qualityScore float64) SearchStrategy {
	// If current strategy failed, try alternatives
	switch currentStrategy {
	case StrategySemanticOnly:
		return StrategyHybrid // Broaden to hybrid
	case StrategyKeywordOnly:
		return StrategyHybrid // Broaden to hybrid
	case StrategyHybrid:
		return StrategyMultiPass // Try comprehensive search
	case StrategyMultiPass:
		return StrategySemanticOnly // Fall back to semantic
	default:
		return StrategyHybrid
	}
}
