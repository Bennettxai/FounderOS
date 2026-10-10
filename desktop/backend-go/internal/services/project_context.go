package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectContextService handles loading context for projects and nodes
type ProjectContextService struct {
	pool           *pgxpool.Pool
	contextService *ContextService
	logger         *slog.Logger
}

// NewProjectContextService creates a new project context service
func NewProjectContextService(pool *pgxpool.Pool, contextService *ContextService) *ProjectContextService {
	return &ProjectContextService{
		pool:           pool,
		contextService: contextService,
		logger:         slog.Default().With("service", "project_context"),
	}
}

// ============================================================================
// Types
// ============================================================================

// Project represents a project entity
type Project struct {
	ID          uuid.UUID  `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority,omitempty"`
	ClientID    *uuid.UUID `json:"client_id,omitempty"`
	ClientName  string     `json:"client_name,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Node represents a business node entity
type Node struct {
	ID          uuid.UUID  `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Path        string     `json:"path,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Memory represents a memory entity
type Memory struct {
	ID              uuid.UUID  `json:"id"`
	UserID          string     `json:"user_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	Content         string     `json:"content"`
	MemoryType      string     `json:"memory_type"`
	Category        string     `json:"category,omitempty"`
	SourceType      string     `json:"source_type"`
	SourceID        *uuid.UUID `json:"source_id,omitempty"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	NodeID          *uuid.UUID `json:"node_id,omitempty"`
	ImportanceScore float64    `json:"importance_score"`
	AccessCount     int        `json:"access_count"`
	IsPinned        bool       `json:"is_pinned"`
	Tags            []string   `json:"tags,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// VoiceNote represents a voice note entity
type VoiceNote struct {
	ID         uuid.UUID  `json:"id"`
	UserID     string     `json:"user_id"`
	Title      string     `json:"title,omitempty"`
	Transcript string     `json:"transcript,omitempty"`
	Duration   int        `json:"duration"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
	NodeID     *uuid.UUID `json:"node_id,omitempty"`
	KeyTopics  []string   `json:"key_topics,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Artifact represents an artifact entity
type Artifact struct {
	ID           uuid.UUID  `json:"id"`
	UserID       string     `json:"user_id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	ArtifactType string     `json:"artifact_type"`
	ProjectID    *uuid.UUID `json:"project_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Document represents an uploaded document
type Document struct {
	ID            uuid.UUID `json:"id"`
	UserID        string    `json:"user_id"`
	Filename      string    `json:"filename"`
	DisplayName   string    `json:"display_name,omitempty"`
	Description   string    `json:"description,omitempty"`
	FileType      string    `json:"file_type"`
	DocumentType  string    `json:"document_type,omitempty"`
	ExtractedText string    `json:"extracted_text,omitempty"`
	WordCount     int       `json:"word_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// Context represents a KB context entity
type Context struct {
	ID           uuid.UUID `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Content      string    `json:"content,omitempty"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProjectContext contains all context for a project
type ProjectContext struct {
	Project               *Project              `json:"project"`
	Profile               *ContextProfile       `json:"profile,omitempty"`
	Memories              []Memory              `json:"memories"`
	Documents             []Document            `json:"documents"`
	Artifacts             []Artifact            `json:"artifacts"`
	VoiceNotes            []VoiceNote           `json:"voice_notes"`
	Conversations         []ConversationSummary `json:"conversations"`
	KnowledgeBaseContexts []Context             `json:"kb_contexts"`
	UserFacts             []UserFact            `json:"user_facts"`
	TotalTokenEstimate    int                   `json:"total_token_estimate"`
}

// NodeContext contains all context for a node
type NodeContext struct {
	Node          *Node           `json:"node"`
	Ancestors     []*Node         `json:"ancestors,omitempty"`
	Profile       *ContextProfile `json:"profile,omitempty"`
	Memories      []Memory        `json:"memories"`
	Projects      []Project       `json:"projects"`
	ParentContext *ProjectContext `json:"parent_context,omitempty"`
}

// InjectedContext represents context ready to be injected into a conversation
type InjectedContext struct {
	SystemPromptAddition string         `json:"system_prompt_addition"`
	LoadedItems          []ContextItem  `json:"loaded_items"`
	TotalTokens          int            `json:"total_tokens"`
	TokenBreakdown       map[string]int `json:"token_breakdown"`
}
