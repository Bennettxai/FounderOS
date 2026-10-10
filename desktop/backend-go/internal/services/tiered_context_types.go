package services

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextTier represents the level of context detail
type ContextTier int

const (
	TierFullContext ContextTier = 1 // Full content, embeddings searched
	TierAwareness   ContextTier = 2 // Titles/summaries only
	TierOnDemand    ContextTier = 3 // Available via tool call
)

// TieredContextRequest contains all context selection parameters
type TieredContextRequest struct {
	UserID      string
	ContextIDs  []uuid.UUID // Selected contexts (Level 1)
	ProjectID   *uuid.UUID  // Selected project (Level 1)
	NodeID      *uuid.UUID  // Business node context
	DocumentIDs []uuid.UUID // Attached document IDs for RAG
}

// TieredContext contains all context organized by tier
type TieredContext struct {
	Level1 *FullContext      `json:"level_1"` // Full context for selected items
	Level2 *AwarenessContext `json:"level_2"` // Awareness of related items
	Level3 *OnDemandRegistry `json:"level_3"` // Registry of fetchable items
}

// FullContext contains detailed information for selected items (Level 1)
type FullContext struct {
	Project      *ProjectFullContext  `json:"project,omitempty"`
	Contexts     []ContextFullContext `json:"contexts,omitempty"`
	Tasks        []TaskFullContext    `json:"tasks,omitempty"`
	LinkedClient *ClientFullContext   `json:"linked_client,omitempty"`
	TeamMembers  []TeamMemberContext  `json:"team_members,omitempty"`
	RelevantRAG  []RelevantBlock      `json:"relevant_rag,omitempty"`
	Documents    []DocumentContext    `json:"documents,omitempty"` // Attached documents for RAG
	Memories     []MemoryFullContext  `json:"memories,omitempty"`  // Personal memories from episodic memory
}

// MemoryFullContext contains memory information for L1 context injection
type MemoryFullContext struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
	MemoryType      string    `json:"memory_type"`
	ImportanceScore float64   `json:"importance_score"`
}

// DocumentContext contains document information for context injection
type DocumentContext struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	DisplayName string    `json:"display_name,omitempty"`
	Content     string    `json:"content,omitempty"`
	ChunkCount  int       `json:"chunk_count"`
	MimeType    string    `json:"mime_type,omitempty"`
}

// ProjectFullContext contains complete project information
type ProjectFullContext struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	ClientName  string    `json:"client_name,omitempty"`
	ProjectType string    `json:"project_type,omitempty"`
}

// ContextFullContext contains complete context/document information
type ContextFullContext struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Content       string    `json:"content,omitempty"`
	SystemPrompt  string    `json:"system_prompt,omitempty"`
	WordCount     int       `json:"word_count"`
	HasEmbeddings bool      `json:"has_embeddings"`
}

// TaskFullContext contains complete task information
type TaskFullContext struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	DueDate      string    `json:"due_date,omitempty"`
	AssigneeName string    `json:"assignee_name,omitempty"`
}

// ClientFullContext contains complete client information
type ClientFullContext struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email,omitempty"`
	Industry     string    `json:"industry,omitempty"`
	Status       string    `json:"status"`
	ContactCount int       `json:"contact_count"`
}

// TeamMemberContext contains team member summary
type TeamMemberContext struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Role   string    `json:"role"`
	Status string    `json:"status"`
}

// AwarenessContext contains summaries for related items (Level 2)
type AwarenessContext struct {
	OtherProjects   []EntitySummary `json:"other_projects,omitempty"`
	SiblingContexts []EntitySummary `json:"sibling_contexts,omitempty"`
	RelatedClients  []EntitySummary `json:"related_clients,omitempty"`
	NodeInfo        *NodeSummary    `json:"node_info,omitempty"`
	ParentNodes     []NodeSummary   `json:"parent_nodes,omitempty"`
	UserFacts       []UserFact      `json:"user_facts,omitempty"`
}

// EntitySummary provides minimal awareness of an entity
type EntitySummary struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type,omitempty"` // For contexts: document, business, etc.
}

// NodeSummary provides business node context
type NodeSummary struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Health        string    `json:"health,omitempty"`
	Purpose       string    `json:"purpose,omitempty"`
	ThisWeekFocus []string  `json:"this_week_focus,omitempty"`
}

// OnDemandRegistry tracks what can be fetched on-demand (Level 3)
type OnDemandRegistry struct {
	AvailableEntities []OnDemandEntity `json:"available_entities"`
}

// OnDemandEntity represents an entity that can be fetched
type OnDemandEntity struct {
	Type string    `json:"type"` // project, context, task, client, team_member
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// nodeAncestryRow is an internal type used during node ancestry queries
type nodeAncestryRow struct {
	ID        uuid.UUID
	ParentID  *uuid.UUID
	ContextID *uuid.UUID
	Name      string
	Type      string
	Health    string
	Purpose   string
	Depth     int
}

// TieredContextService builds tiered context for AI queries
type TieredContextService struct {
	pool             *pgxpool.Pool
	embeddingService *EmbeddingService
	summarizer       *SummarizerService
}

// NewTieredContextService creates a new tiered context service
func NewTieredContextService(pool *pgxpool.Pool, embeddingService *EmbeddingService, summarizer *SummarizerService) *TieredContextService {
	return &TieredContextService{
		pool:             pool,
		embeddingService: embeddingService,
		summarizer:       summarizer,
	}
}
