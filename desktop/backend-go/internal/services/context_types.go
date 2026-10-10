package services

import (
	"time"

	"github.com/google/uuid"
)

// ContextProfile represents a context profile entity
type ContextProfile struct {
	ID                 uuid.UUID      `json:"id"`
	UserID             string         `json:"user_id"`
	EntityType         string         `json:"entity_type"`
	EntityID           uuid.UUID      `json:"entity_id"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	ContextTree        map[string]any `json:"context_tree"`
	Summary            string         `json:"summary,omitempty"`
	KeyFacts           []string       `json:"key_facts,omitempty"`
	DocumentTypes      []string       `json:"document_types,omitempty"`
	TotalDocuments     int            `json:"total_documents"`
	TotalFileSizeBytes int64          `json:"total_file_size_bytes"`
	TotalContexts      int            `json:"total_contexts"`
	TotalMemories      int            `json:"total_memories"`
	TotalArtifacts     int            `json:"total_artifacts"`
	TotalTasks         int            `json:"total_tasks"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ContextLoadingRule defines what context to load automatically
type ContextLoadingRule struct {
	ID                      uuid.UUID `json:"id"`
	UserID                  string    `json:"user_id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description,omitempty"`
	TriggerType             string    `json:"trigger_type"`
	TriggerValue            string    `json:"trigger_value,omitempty"`
	LoadMemories            bool      `json:"load_memories"`
	MemoryTypes             []string  `json:"memory_types,omitempty"`
	MemoryLimit             int       `json:"memory_limit"`
	LoadContexts            bool      `json:"load_contexts"`
	ContextCategories       []string  `json:"context_categories,omitempty"`
	ContextLimit            int       `json:"context_limit"`
	LoadArtifacts           bool      `json:"load_artifacts"`
	ArtifactTypes           []string  `json:"artifact_types,omitempty"`
	ArtifactLimit           int       `json:"artifact_limit"`
	LoadRecentConversations bool      `json:"load_recent_conversations"`
	ConversationLimit       int       `json:"conversation_limit"`
	Priority                int       `json:"priority"`
	IsActive                bool      `json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// AgentContextSession tracks context for an agent session
type AgentContextSession struct {
	ID                      uuid.UUID   `json:"id"`
	UserID                  string      `json:"user_id"`
	ConversationID          uuid.UUID   `json:"conversation_id"`
	AgentType               string      `json:"agent_type"`
	AgentID                 *uuid.UUID  `json:"agent_id,omitempty"`
	MaxContextTokens        int         `json:"max_context_tokens"`
	UsedContextTokens       int         `json:"used_context_tokens"`
	AvailableTokens         int         `json:"available_tokens"`
	LoadedMemories          []uuid.UUID `json:"loaded_memories"`
	LoadedContexts          []uuid.UUID `json:"loaded_contexts"`
	LoadedArtifacts         []uuid.UUID `json:"loaded_artifacts"`
	LoadedDocuments         []uuid.UUID `json:"loaded_documents"`
	BaseSystemPrompt        string      `json:"base_system_prompt,omitempty"`
	InjectedContext         string      `json:"injected_context,omitempty"`
	TotalSystemPromptTokens int         `json:"total_system_prompt_tokens"`
	ProjectID               *uuid.UUID  `json:"project_id,omitempty"`
	NodeID                  *uuid.UUID  `json:"node_id,omitempty"`
	FocusMode               string      `json:"focus_mode,omitempty"`
	StartedAt               time.Time   `json:"started_at"`
	LastActivityAt          time.Time   `json:"last_activity_at"`
	EndedAt                 *time.Time  `json:"ended_at,omitempty"`
}

// TreeSearchParams contains parameters for tree search
type TreeSearchParams struct {
	Query        string   `json:"query"`
	SearchType   string   `json:"search_type"`  // 'title', 'content', 'semantic', 'browse'
	EntityTypes  []string `json:"entity_types"` // 'memories', 'contexts', 'artifacts', 'documents'
	ProjectScope *string  `json:"project_scope"`
	NodeScope    *string  `json:"node_scope"`
	MaxResults   int      `json:"max_results"`
}

// TreeSearchResult represents a search result from the context tree
type TreeSearchResult struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	Summary        string    `json:"summary,omitempty"`
	RelevanceScore float64   `json:"relevance_score"`
	TreePath       []string  `json:"tree_path"`
	TokenEstimate  int       `json:"token_estimate"`
}

// ContextItem represents a loaded context item
type ContextItem struct {
	ID         uuid.UUID      `json:"id"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	TokenCount int            `json:"token_count"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ContextTree represents the hierarchical context structure
type ContextTree struct {
	RootNode    *ContextTreeNode `json:"root_node"`
	TotalItems  int              `json:"total_items"`
	LastUpdated time.Time        `json:"last_updated"`
}

// ContextTreeNode represents a node in the context tree
type ContextTreeNode struct {
	ID          uuid.UUID          `json:"id"`
	Type        string             `json:"type"` // 'root', 'project', 'node', 'category', 'item'
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Icon        string             `json:"icon,omitempty"`
	ItemCount   int                `json:"item_count"`
	TokenCount  int                `json:"token_count"`
	Children    []*ContextTreeNode `json:"children,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
}

// TreeStatistics contains statistics about the context tree
type TreeStatistics struct {
	TotalProjects   int            `json:"total_projects"`
	TotalNodes      int            `json:"total_nodes"`
	TotalMemories   int            `json:"total_memories"`
	TotalContexts   int            `json:"total_contexts"`
	TotalArtifacts  int            `json:"total_artifacts"`
	TotalDocuments  int            `json:"total_documents"`
	TotalVoiceNotes int            `json:"total_voice_notes"`
	ByType          map[string]int `json:"by_type"`
	TotalTokens     int            `json:"total_tokens"`
}

// AgentContext contains the built context for an agent
type AgentContext struct {
	SystemPromptAddition string                `json:"system_prompt_addition"`
	LoadedMemories       []ContextItem         `json:"loaded_memories"`
	LoadedContexts       []ContextItem         `json:"loaded_contexts"`
	LoadedArtifacts      []ContextItem         `json:"loaded_artifacts"`
	LoadedDocuments      []ContextItem         `json:"loaded_documents"`
	RecentConversations  []ConversationSummary `json:"recent_conversations"`
	UserFacts            []UserFact            `json:"user_facts"`
	TotalTokens          int                   `json:"total_tokens"`
	TokenBreakdown       map[string]int        `json:"token_breakdown"`
}

// ConversationSummary represents a summarized conversation
type ConversationSummary struct {
	ID            uuid.UUID `json:"id"`
	Summary       string    `json:"summary"`
	KeyPoints     []string  `json:"key_points,omitempty"`
	DecisionsMade []string  `json:"decisions_made,omitempty"`
	Topics        []string  `json:"topics,omitempty"`
	MessageCount  int       `json:"message_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserFact represents a user fact
type UserFact struct {
	ID              uuid.UUID `json:"id"`
	UserID          string    `json:"user_id"`
	FactKey         string    `json:"fact_key"`
	FactValue       string    `json:"fact_value"`
	FactType        string    `json:"fact_type"`
	ConfidenceScore float64   `json:"confidence_score"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

// ContextBuildInput contains parameters for building agent context
type ContextBuildInput struct {
	UserID         string
	ProjectID      *uuid.UUID
	NodeID         *uuid.UUID
	ConversationID *uuid.UUID
	AgentType      string
	FocusMode      string
	CurrentQuery   string
	MaxTokens      int
}
