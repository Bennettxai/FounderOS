package handlers

// ================================================
// RESPONSE TYPES
// ================================================

// MemoryResponse represents a memory in API responses
type MemoryResponse struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Title           string                 `json:"title"`
	Summary         string                 `json:"summary"`
	Content         string                 `json:"content"`
	MemoryType      string                 `json:"memory_type"`
	Category        *string                `json:"category"`
	SourceType      string                 `json:"source_type"`
	SourceID        *string                `json:"source_id"`
	SourceContext   *string                `json:"source_context"`
	ProjectID       *string                `json:"project_id"`
	NodeID          *string                `json:"node_id"`
	ImportanceScore float64                `json:"importance_score"`
	AccessCount     int                    `json:"access_count"`
	LastAccessedAt  *string                `json:"last_accessed_at"`
	IsActive        bool                   `json:"is_active"`
	IsPinned        bool                   `json:"is_pinned"`
	ExpiresAt       *string                `json:"expires_at"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// UserFactResponse represents a user fact in API responses
type UserFactResponse struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	FactKey         string  `json:"fact_key"`
	FactValue       string  `json:"fact_value"`
	FactType        string  `json:"fact_type"`
	SourceMemoryID  *string `json:"source_memory_id"`
	ConfidenceScore float64 `json:"confidence_score"`
	IsActive        bool    `json:"is_active"`
	LastConfirmedAt *string `json:"last_confirmed_at"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// MemoryStatsResponse contains memory statistics
type MemoryStatsResponse struct {
	TotalMemories     int            `json:"total_memories"`
	ActiveMemories    int            `json:"active_memories"`
	PinnedMemories    int            `json:"pinned_memories"`
	ByType            map[string]int `json:"by_type"`
	ByCategory        map[string]int `json:"by_category"`
	TotalFacts        int            `json:"total_facts"`
	RecentAccessCount int            `json:"recent_access_count"`
}

// MemorySearchResult embeds MemoryResponse with a similarity score
type MemorySearchResult struct {
	MemoryResponse
	Similarity float64 `json:"similarity"`
}

// MemoryRelevantResult embeds MemoryResponse with a hybrid relevance score
type MemoryRelevantResult struct {
	MemoryResponse
	RelevanceScore float64 `json:"relevance_score"`
}

// ================================================
// REQUEST TYPES
// ================================================

// CreateMemoryRequest represents a request to create a memory
type CreateMemoryRequest struct {
	Title         string                 `json:"title" binding:"required"`
	Summary       string                 `json:"summary" binding:"required"`
	Content       string                 `json:"content" binding:"required"`
	MemoryType    string                 `json:"memory_type" binding:"required"` // fact, preference, decision, pattern, insight, interaction, learning
	Category      *string                `json:"category"`
	SourceType    string                 `json:"source_type" binding:"required"` // conversation, voice_note, document, task, project, manual, inferred
	SourceID      *string                `json:"source_id"`
	SourceContext *string                `json:"source_context"`
	ProjectID     *string                `json:"project_id"`
	NodeID        *string                `json:"node_id"`
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// UpdateMemoryRequest represents a request to update a memory
type UpdateMemoryRequest struct {
	Title           *string                `json:"title"`
	Summary         *string                `json:"summary"`
	Content         *string                `json:"content"`
	MemoryType      *string                `json:"memory_type"`
	Category        *string                `json:"category"`
	ImportanceScore *float64               `json:"importance_score"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	IsActive        *bool                  `json:"is_active"`
}

// MemorySearchRequest represents a semantic search request
type MemorySearchRequest struct {
	Query      string   `json:"query" binding:"required"`
	MemoryType *string  `json:"memory_type"`
	ProjectID  *string  `json:"project_id"`
	NodeID     *string  `json:"node_id"`
	Limit      int      `json:"limit"`
	Tags       []string `json:"tags"`
}

// RelevantMemoriesRequest represents a request for relevant memories
type RelevantMemoriesRequest struct {
	Context        string   `json:"context" binding:"required"`
	ConversationID *string  `json:"conversation_id"`
	ProjectID      *string  `json:"project_id"`
	NodeID         *string  `json:"node_id"`
	Limit          int      `json:"limit"`
	MemoryTypes    []string `json:"memory_types"`
}

// UpdateFactRequest represents a request to update a user fact
type UpdateFactRequest struct {
	FactValue       string   `json:"fact_value" binding:"required"`
	FactType        *string  `json:"fact_type"`
	ConfidenceScore *float64 `json:"confidence_score"`
}
