// RAG Search Types for BusinessOS Frontend
// Integrates with backend /api/rag endpoints

// Hybrid Search Types
export interface HybridSearchRequest {
	query: string;
	semantic_weight?: number; // Default: 0.7
	keyword_weight?: number; // Default: 0.3
	max_results?: number; // Default: 10
	min_similarity?: number; // Default: 0.3
	project_id?: string;
	workspace_id?: string;
}

export interface HybridSearchResult {
	context_id: string;
	block_id: string;
	block_type: string;
	content: string;
	context_name: string;
	context_type: string;
	semantic_score: number;
	keyword_score: number;
	hybrid_score: number;
	search_strategy: 'semantic' | 'keyword' | 'hybrid';
	created_at?: string;
	updated_at?: string;
}

export interface HybridSearchResponse {
	query: string;
	results: HybridSearchResult[];
	count: number;
	options: {
		semantic_weight: number;
		keyword_weight: number;
		max_results: number;
		min_similarity?: number;
	};
}

// Search Explanation Types
export interface SearchExplanationResponse {
	query: string;
	total_results: number;
	strategy_breakdown: {
		semantic: number;
		keyword: number;
		hybrid: number;
	};
	avg_semantic_score: number;
	avg_keyword_score: number;
	avg_hybrid_score: number;
	options: {
		semantic_weight: number;
		keyword_weight: number;
		rrf_constant: number;
		min_similarity: number;
	};
	top_5_results?: HybridSearchResult[];
}

// Agentic RAG Types
export type QueryIntent =
	| 'factual_lookup'
	| 'conceptual_search'
	| 'procedural'
	| 'comparison'
	| 'recent'
	| 'exhaustive'
	| 'ambiguous';

export type SearchStrategy = 'semantic_only' | 'keyword_only' | 'hybrid' | 'multi_pass';

export interface AgenticRAGRequest {
	query: string;
	max_results?: number; // Default: 10
	min_quality_score?: number; // Default: 0.5
	project_id?: string;
	task_id?: string;
	use_personalization?: boolean; // Default: false
	workspace_id?: string;
}

export interface AgenticRAGResult extends HybridSearchResult {
	// Re-ranking scores
	recency_score: number;
	quality_score: number;
	interaction_score: number;
	context_score: number;
	final_score: number;

	// Ranking information
	original_rank: number;
	reranked_position: number;
	rank_change: number;

	// Score breakdown
	score_breakdown: {
		semantic: number;
		recency: number;
		quality: number;
		interaction: number;
		context: number;
	};
}

export interface AgenticRAGResponse {
	results: AgenticRAGResult[];
	query_intent: QueryIntent;
	strategy_used: SearchStrategy;
	strategy_reasoning: string;
	quality_score: number;
	iteration_count: number;
	personalized: boolean;
	processing_time_ms: number;
	metadata: {
		intent_classification: QueryIntent;
		user_preferences?: {
			preferred_tone?: string;
			preferred_verbosity?: string;
			expertise_areas?: string[];
		};
	};
}

// Re-rank Types
export interface ReRankRequest {
	results: HybridSearchResult[];
	query: string;
	project_id?: string;
	task_id?: string;
}

export interface ReRankResponse {
	results: AgenticRAGResult[];
	original_count: number;
	reranked_count: number;
}

// Memory Types
export type MemoryType = 'pattern' | 'decision' | 'fact' | 'preference' | 'context';

export interface RAGMemory {
	id: string;
	user_id: string;
	title: string;
	summary: string;
	content: string;
	memory_type: MemoryType;
	category: string;
	importance_score: number;
	access_count: number;
	is_pinned: boolean;
	tags: string[];
	created_at: string;
	updated_at: string;
}

export interface CreateMemoryRequest {
	title: string;
	summary: string;
	content: string;
	memory_type: MemoryType;
	category: string;
	source_type?: 'conversation' | 'document' | 'manual';
	source_id?: string;
	project_id?: string;
	node_id?: string;
	importance_score?: number; // Default: 0.5
	tags?: string[];
}

export interface ListMemoriesRequest {
	type?: MemoryType;
	limit?: number; // Default: 50, max: 100
	offset?: number;
}

export interface ListMemoriesResponse {
	memories: RAGMemory[];
	count: number;
	total?: number;
}

// Error Response
export interface RAGErrorResponse {
	error: string;
	details?: string;
	code?: string;
}

// UI Helper Types
export interface SearchFilters {
	workspace_id?: string;
	project_id?: string;
	min_score?: number;
	date_from?: string;
	date_to?: string;
}

export interface SearchWeights {
	semantic: number;
	keyword: number;
}

// Document Preview Types
export interface DocumentPreview {
	context_id: string;
	context_name: string;
	context_type: string;
	full_content?: string;
	highlighted_content?: string;
	metadata?: Record<string, any>;
}
