package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MemoryHierarchyService handles workspace vs user memory isolation
type MemoryHierarchyService struct {
	pool *pgxpool.Pool
}

// WorkspaceMemoryItem represents a memory entry (workspace or private)
type WorkspaceMemoryItem struct {
	ID          uuid.UUID              `json:"id"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	MemoryType  string                 `json:"memory_type"`
	Visibility  string                 `json:"visibility"` // workspace, private, shared
	Importance  float64                `json:"importance"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	OwnerUserID *string                `json:"owner_user_id,omitempty"`
	SharedWith  []string               `json:"shared_with,omitempty"`
	AccessCount int                    `json:"access_count"`
	IsOwner     bool                   `json:"is_owner"`
	CreatedAt   time.Time              `json:"created_at"`
}

// NewMemoryHierarchyService creates a new memory hierarchy service
func NewMemoryHierarchyService(pool *pgxpool.Pool) *MemoryHierarchyService {
	return &MemoryHierarchyService{pool: pool}
}

// GetWorkspaceMemories retrieves shared workspace memories
func (s *MemoryHierarchyService) GetWorkspaceMemories(ctx context.Context, workspaceID uuid.UUID, userID string, memoryType *string, limit int) ([]WorkspaceMemoryItem, error) {
	var typeParam *string
	if memoryType != nil && *memoryType != "" {
		typeParam = memoryType
	}

	rows, err := s.pool.Query(ctx, "SELECT * FROM get_workspace_memories($1, $2, $3, $4)", workspaceID, userID, typeParam, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []WorkspaceMemoryItem
	for rows.Next() {
		var m WorkspaceMemoryItem
		err := rows.Scan(&m.ID, &m.Title, &m.Content, &m.MemoryType, &m.Importance, &m.Tags, &m.Metadata, &m.AccessCount, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		m.Visibility = "workspace"
		memories = append(memories, m)
	}

	return memories, nil
}

// GetUserMemories retrieves user's private and shared memories
func (s *MemoryHierarchyService) GetUserMemories(ctx context.Context, workspaceID uuid.UUID, userID string, memoryType *string, limit int) ([]WorkspaceMemoryItem, error) {
	var typeParam *string
	if memoryType != nil && *memoryType != "" {
		typeParam = memoryType
	}

	rows, err := s.pool.Query(ctx, "SELECT * FROM get_user_memories($1, $2, $3, $4)", workspaceID, userID, typeParam, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []WorkspaceMemoryItem
	for rows.Next() {
		var m WorkspaceMemoryItem
		err := rows.Scan(&m.ID, &m.Title, &m.Content, &m.MemoryType, &m.Importance, &m.Tags, &m.Metadata, &m.Visibility, &m.SharedWith, &m.AccessCount, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		m.OwnerUserID = &userID
		memories = append(memories, m)
	}

	return memories, nil
}

// GetAccessibleMemories retrieves all memories accessible to user (workspace + private + shared)
func (s *MemoryHierarchyService) GetAccessibleMemories(ctx context.Context, workspaceID uuid.UUID, userID string, memoryType *string, limit int) ([]WorkspaceMemoryItem, error) {
	// NOTE: Using raw SQL query instead of PostgreSQL function because function has issues
	// The raw query works correctly and returns expected results

	rows, err := s.pool.Query(ctx, `
		SELECT
			wm.id,
			wm.title,
			wm.content,
			wm.memory_type,
			wm.visibility,
			wm.importance_score as importance,
			wm.tags,
			wm.metadata,
			(wm.owner_user_id = $2 OR wm.owner_user_id IS NULL) as is_owner,
			wm.access_count,
			wm.created_at
		FROM workspace_memories wm
		WHERE wm.workspace_id = $1
		AND wm.is_active = true
		AND (
			wm.visibility = 'workspace' OR wm.visibility IS NULL
			OR
			(wm.visibility = 'private' AND wm.owner_user_id = $2)
			OR
			(wm.visibility = 'shared' AND (wm.owner_user_id = $2 OR $2 = ANY(COALESCE(wm.shared_with, ARRAY[]::TEXT[]))))
		)
		AND ($3::text IS NULL OR wm.memory_type = $3::text)
		ORDER BY wm.importance_score DESC NULLS LAST, wm.created_at DESC
		LIMIT $4
	`, workspaceID, userID, memoryType, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []WorkspaceMemoryItem
	for rows.Next() {
		var m WorkspaceMemoryItem
		err := rows.Scan(&m.ID, &m.Title, &m.Content, &m.MemoryType, &m.Visibility, &m.Importance, &m.Tags, &m.Metadata, &m.IsOwner, &m.AccessCount, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		memories = append(memories, m)
	}

	return memories, nil
}

// CanAccessMemory checks if user can access a specific memory
func (s *MemoryHierarchyService) CanAccessMemory(ctx context.Context, userID string, memoryID uuid.UUID) (bool, error) {
	var canAccess bool
	err := s.pool.QueryRow(ctx, "SELECT can_access_memory($1, $2)", userID, memoryID).Scan(&canAccess)
	return canAccess, err
}

// ShareMemory shares a private memory with specific users
func (s *MemoryHierarchyService) ShareMemory(ctx context.Context, memoryID uuid.UUID, ownerID string, shareWithUserIDs []string) error {
	var success bool
	err := s.pool.QueryRow(ctx, "SELECT share_memory($1, $2, $3)", memoryID, ownerID, shareWithUserIDs).Scan(&success)
	if err != nil {
		return fmt.Errorf("share memory: %w", err)
	}
	if !success {
		return fmt.Errorf("failed to share memory")
	}
	return nil
}

// UnshareMemory makes a shared memory private again
func (s *MemoryHierarchyService) UnshareMemory(ctx context.Context, memoryID uuid.UUID, ownerID string) error {
	var success bool
	err := s.pool.QueryRow(ctx, "SELECT unshare_memory($1, $2)", memoryID, ownerID).Scan(&success)
	if err != nil {
		return fmt.Errorf("unshare memory: %w", err)
	}
	if !success {
		return fmt.Errorf("failed to unshare memory")
	}
	return nil
}

// TrackAccess increments access counter for a memory
func (s *MemoryHierarchyService) TrackAccess(ctx context.Context, memoryID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "SELECT track_memory_access($1)", memoryID)
	return err
}

// CreateWorkspaceMemory creates a workspace-level memory (accessible to all members)
func (s *MemoryHierarchyService) CreateWorkspaceMemory(ctx context.Context, workspaceID uuid.UUID, title, content, memoryType, createdBy string, tags []string, metadata map[string]interface{}) (*WorkspaceMemoryItem, error) {
	memory := &WorkspaceMemoryItem{}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_memories (workspace_id, title, content, memory_type, visibility, created_by, tags, metadata)
		VALUES ($1, $2, $3, $4, 'workspace', $5, $6, $7)
		RETURNING id, title, content, memory_type, visibility, tags, metadata, access_count, created_at
	`, workspaceID, title, content, memoryType, createdBy, tags, metadata).Scan(
		&memory.ID, &memory.Title, &memory.Content, &memory.MemoryType, &memory.Visibility, &memory.Tags, &memory.Metadata, &memory.AccessCount, &memory.CreatedAt,
	)

	return memory, err
}

// CreatePrivateMemory creates a user's private memory
func (s *MemoryHierarchyService) CreatePrivateMemory(ctx context.Context, workspaceID uuid.UUID, userID, title, content, memoryType string, tags []string, metadata map[string]interface{}) (*WorkspaceMemoryItem, error) {
	memory := &WorkspaceMemoryItem{}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_memories (workspace_id, title, content, memory_type, visibility, owner_user_id, created_by, tags, metadata)
		VALUES ($1, $2, $3, $4, 'private', $5, $5, $6, $7)
		RETURNING id, title, content, memory_type, visibility, owner_user_id, tags, metadata, access_count, created_at
	`, workspaceID, title, content, memoryType, userID, tags, metadata).Scan(
		&memory.ID, &memory.Title, &memory.Content, &memory.MemoryType, &memory.Visibility, &memory.OwnerUserID, &memory.Tags, &memory.Metadata, &memory.AccessCount, &memory.CreatedAt,
	)

	return memory, err
}
