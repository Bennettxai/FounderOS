package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// MemoryService handles memory persistence operations
type MemoryService struct {
	pool         *pgxpool.Pool
	embeddingSvc *EmbeddingService
}

// NewMemoryService creates a new memory service
func NewMemoryService(pool *pgxpool.Pool, embeddingSvc *EmbeddingService) *MemoryService {
	return &MemoryService{
		pool:         pool,
		embeddingSvc: embeddingSvc,
	}
}

// WorkspaceMemoryRequest represents a request to create a workspace memory
type WorkspaceMemoryRequest struct {
	WorkspaceID     uuid.UUID
	UserID          string
	Title           string
	Summary         string
	Content         string
	MemoryType      string // 'general', 'decision', 'pattern', 'context', 'learning', 'preference'
	Category        string
	Visibility      string // 'workspace', 'private', 'shared'
	Tags            []string
	Metadata        map[string]interface{}
	ImportanceScore float64
	ScopeType       *string    // 'workspace', 'project', 'node'
	ScopeID         *uuid.UUID // project or node ID
}

// WorkspaceMemory represents a memory in the workspace_memories table
type WorkspaceMemory struct {
	ID              uuid.UUID              `json:"id"`
	WorkspaceID     uuid.UUID              `json:"workspace_id"`
	Title           string                 `json:"title"`
	Summary         string                 `json:"summary"`
	Content         string                 `json:"content"`
	MemoryType      string                 `json:"memory_type"`
	Category        string                 `json:"category"`
	Visibility      string                 `json:"visibility"`
	OwnerUserID     *string                `json:"owner_user_id,omitempty"`
	SharedWith      []string               `json:"shared_with,omitempty"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	ImportanceScore float64                `json:"importance_score"`
	AccessCount     int                    `json:"access_count"`
	ScopeType       string                 `json:"scope_type"`
	ScopeID         *uuid.UUID             `json:"scope_id,omitempty"`
	IsPinned        bool                   `json:"is_pinned"`
	IsActive        bool                   `json:"is_active"`
	CreatedBy       string                 `json:"created_by"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	LastAccessedAt  *time.Time             `json:"last_accessed_at,omitempty"`
}

// MemoryQueryOptions represents query options for retrieving memories
type MemoryQueryOptions struct {
	MemoryType *string
	Category   *string
	Tags       []string
	Limit      int
}

// CreateWorkspaceMemory creates a workspace-level memory (accessible to all workspace members)
func (m *MemoryService) CreateWorkspaceMemory(ctx context.Context, req WorkspaceMemoryRequest) (*WorkspaceMemory, error) {
	// Generate embedding
	var embeddingVec interface{}
	if m.embeddingSvc != nil {
		textToEmbed := req.Title + " " + req.Summary + " " + req.Content
		embedding, err := m.embeddingSvc.GenerateEmbedding(ctx, textToEmbed)
		if err == nil && len(embedding) > 0 {
			// Convert to pgvector format
			embeddingVec = pgvector.NewVector(embedding)
		}
	}

	// Prepare tags - pgx can handle []string directly
	tags := req.Tags
	if tags == nil {
		tags = []string{} // Empty slice instead of nil
	}

	// Serialize metadata to JSON
	metadataJSON := "{}"
	if req.Metadata != nil {
		metaBytes, _ := json.Marshal(req.Metadata)
		metadataJSON = string(metaBytes)
	}

	// Visibility must be 'workspace' for workspace memories
	visibility := "workspace"
	if req.Visibility != "" {
		visibility = req.Visibility
	}

	// Set owner_user_id only for private/shared memories
	var ownerUserID *string
	if visibility != "workspace" {
		ownerUserID = &req.UserID
	}

	scopeType := "workspace"
	if req.ScopeType != nil {
		scopeType = *req.ScopeType
	}

	importanceScore := req.ImportanceScore
	if importanceScore == 0 {
		importanceScore = 0.5
	}

	query := `
		INSERT INTO workspace_memories (
			workspace_id, title, summary, content, memory_type, category,
			visibility, owner_user_id, tags, metadata, importance_score,
			scope_type, scope_id, embedding, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	var memory WorkspaceMemory
	err := m.pool.QueryRow(ctx, query,
		req.WorkspaceID,
		req.Title,
		req.Summary,
		req.Content,
		req.MemoryType,
		req.Category,
		visibility,
		ownerUserID,
		tags, // Pass slice directly
		metadataJSON,
		importanceScore,
		scopeType,
		req.ScopeID,
		embeddingVec,
		req.UserID,
	).Scan(&memory.ID, &memory.CreatedAt, &memory.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create workspace memory: %w", err)
	}

	// Populate return object
	memory.WorkspaceID = req.WorkspaceID
	memory.Title = req.Title
	memory.Summary = req.Summary
	memory.Content = req.Content
	memory.MemoryType = req.MemoryType
	memory.Category = req.Category
	memory.Visibility = visibility
	memory.OwnerUserID = ownerUserID
	memory.Tags = req.Tags
	memory.Metadata = req.Metadata
	memory.ImportanceScore = importanceScore
	memory.ScopeType = scopeType
	memory.ScopeID = req.ScopeID
	memory.CreatedBy = req.UserID
	memory.IsActive = true
	memory.IsPinned = false
	memory.AccessCount = 0

	return &memory, nil
}

// CreateUserMemory creates a private user memory (only owner can access)
func (m *MemoryService) CreateUserMemory(ctx context.Context, workspaceID uuid.UUID, userID string, title, summary, content, memoryType string, tags []string, metadata map[string]interface{}) (*WorkspaceMemory, error) {
	req := WorkspaceMemoryRequest{
		WorkspaceID:     workspaceID,
		UserID:          userID,
		Title:           title,
		Summary:         summary,
		Content:         content,
		MemoryType:      memoryType,
		Visibility:      "private",
		Tags:            tags,
		Metadata:        metadata,
		ImportanceScore: 0.5,
	}
	return m.CreateWorkspaceMemory(ctx, req)
}
