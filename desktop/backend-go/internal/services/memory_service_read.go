package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetWorkspaceMemories retrieves workspace-level memories (accessible to all workspace members)
func (m *MemoryService) GetWorkspaceMemories(ctx context.Context, workspaceID uuid.UUID, userID string, opts MemoryQueryOptions) ([]WorkspaceMemory, error) {
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	var memoryType *string
	if opts.MemoryType != nil {
		memoryType = opts.MemoryType
	}

	rows, err := m.pool.Query(ctx, "SELECT * FROM get_workspace_memories($1, $2, $3, $4)",
		workspaceID, userID, memoryType, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace memories: %w", err)
	}
	defer rows.Close()

	var memories []WorkspaceMemory
	for rows.Next() {
		var mem WorkspaceMemory
		var tagsJSON []byte
		var metadataJSON []byte

		err := rows.Scan(
			&mem.ID,
			&mem.Content,
			&mem.MemoryType,
			&mem.ImportanceScore,
			&tagsJSON,
			&metadataJSON,
			&mem.AccessCount,
			&mem.CreatedAt,
		)
		if err != nil {
			continue
		}

		// Deserialize
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &mem.Tags); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal tags for memory", "memory_id", mem.ID, "error", err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal metadata for memory", "memory_id", mem.ID, "error", err)
			}
		}

		mem.WorkspaceID = workspaceID
		mem.Visibility = "workspace"
		memories = append(memories, mem)
	}

	return memories, nil
}

// GetUserMemories retrieves user's private and shared memories
func (m *MemoryService) GetUserMemories(ctx context.Context, workspaceID uuid.UUID, userID string, opts MemoryQueryOptions) ([]WorkspaceMemory, error) {
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	var memoryType *string
	if opts.MemoryType != nil {
		memoryType = opts.MemoryType
	}

	rows, err := m.pool.Query(ctx, "SELECT * FROM get_user_memories($1, $2, $3, $4)",
		workspaceID, userID, memoryType, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user memories: %w", err)
	}
	defer rows.Close()

	var memories []WorkspaceMemory
	for rows.Next() {
		var mem WorkspaceMemory
		var tagsJSON []byte
		var metadataJSON []byte
		var sharedWithJSON []byte

		err := rows.Scan(
			&mem.ID,
			&mem.Content,
			&mem.MemoryType,
			&mem.ImportanceScore,
			&tagsJSON,
			&metadataJSON,
			&mem.Visibility,
			&sharedWithJSON,
			&mem.AccessCount,
			&mem.CreatedAt,
		)
		if err != nil {
			continue
		}

		// Deserialize
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &mem.Tags); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal tags for memory", "memory_id", mem.ID, "error", err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal metadata for memory", "memory_id", mem.ID, "error", err)
			}
		}
		if len(sharedWithJSON) > 0 {
			if err := json.Unmarshal(sharedWithJSON, &mem.SharedWith); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal sharedWith for memory", "memory_id", mem.ID, "error", err)
			}
		}

		mem.WorkspaceID = workspaceID
		mem.OwnerUserID = &userID
		memories = append(memories, mem)
	}

	return memories, nil
}

// GetAccessibleMemories retrieves all memories accessible to the user (workspace + private + shared)
func (m *MemoryService) GetAccessibleMemories(ctx context.Context, workspaceID uuid.UUID, userID string, opts MemoryQueryOptions) ([]WorkspaceMemory, error) {
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	var memoryType *string
	if opts.MemoryType != nil {
		memoryType = opts.MemoryType
	}

	rows, err := m.pool.Query(ctx, "SELECT * FROM get_accessible_memories($1, $2, $3, $4)",
		workspaceID, userID, memoryType, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible memories: %w", err)
	}
	defer rows.Close()

	var memories []WorkspaceMemory
	for rows.Next() {
		var mem WorkspaceMemory
		var tagsJSON []byte
		var metadataJSON []byte
		var isOwner bool

		err := rows.Scan(
			&mem.ID,
			&mem.Content,
			&mem.MemoryType,
			&mem.Visibility,
			&mem.ImportanceScore,
			&tagsJSON,
			&metadataJSON,
			&isOwner,
			&mem.AccessCount,
			&mem.CreatedAt,
		)
		if err != nil {
			continue
		}

		// Deserialize
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &mem.Tags); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal tags for memory", "memory_id", mem.ID, "error", err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal metadata for memory", "memory_id", mem.ID, "error", err)
			}
		}

		mem.WorkspaceID = workspaceID
		if isOwner && mem.Visibility != "workspace" {
			mem.OwnerUserID = &userID
		}
		memories = append(memories, mem)
	}

	return memories, nil
}

// GetWorkspaceMemoryByID retrieves a specific workspace memory by ID
func (m *MemoryService) GetWorkspaceMemoryByID(ctx context.Context, workspaceID uuid.UUID, memoryID uuid.UUID, userID string) (*WorkspaceMemory, error) {
	// First check if user can access this memory
	var canAccess bool
	err := m.pool.QueryRow(ctx, "SELECT can_access_memory($1, $2)", userID, memoryID).Scan(&canAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to check memory access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("user does not have access to this memory")
	}

	query := `
		SELECT id, workspace_id, title, summary, content, memory_type, category,
		       visibility, owner_user_id, shared_with, tags, metadata,
		       importance_score, access_count, scope_type, scope_id,
		       is_pinned, is_active, created_by, created_at, updated_at, last_accessed_at
		FROM workspace_memories
		WHERE id = $1 AND workspace_id = $2
	`

	var mem WorkspaceMemory
	var tagsJSON []byte
	var metadataJSON []byte
	var sharedWithJSON []byte

	err = m.pool.QueryRow(ctx, query, memoryID, workspaceID).Scan(
		&mem.ID,
		&mem.WorkspaceID,
		&mem.Title,
		&mem.Summary,
		&mem.Content,
		&mem.MemoryType,
		&mem.Category,
		&mem.Visibility,
		&mem.OwnerUserID,
		&sharedWithJSON,
		&tagsJSON,
		&metadataJSON,
		&mem.ImportanceScore,
		&mem.AccessCount,
		&mem.ScopeType,
		&mem.ScopeID,
		&mem.IsPinned,
		&mem.IsActive,
		&mem.CreatedBy,
		&mem.CreatedAt,
		&mem.UpdatedAt,
		&mem.LastAccessedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("memory not found")
		}
		return nil, fmt.Errorf("failed to get workspace memory: %w", err)
	}

	// Deserialize JSON fields
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &mem.Tags); err != nil {
			// Log error but continue - tags are not critical
			slog.Warn("failed to unmarshal tags for memory", "memory_id", memoryID, "error", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
			// Log error but continue - metadata is not critical
			slog.Warn("failed to unmarshal metadata for memory", "memory_id", memoryID, "error", err)
		}
	}
	if len(sharedWithJSON) > 0 {
		if err := json.Unmarshal(sharedWithJSON, &mem.SharedWith); err != nil {
			// Log error but continue - sharedWith is not critical
			slog.Warn("failed to unmarshal sharedWith for memory", "memory_id", memoryID, "error", err)
		}
	}

	// Track access synchronously - no goroutine leak
	// Use a short timeout context to avoid blocking
	trackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.TrackAccess(trackCtx, memoryID); err != nil {
		// Log error but don't fail the request
		slog.Warn("failed to track access for memory", "memory_id", memoryID, "error", err)
	}

	return &mem, nil
}
