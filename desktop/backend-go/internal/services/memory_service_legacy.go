package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ========== LEGACY METHODS (for backward compatibility) ==========

// CreateMemory creates a new memory in the database (legacy table)
func (m *MemoryService) CreateMemory(ctx context.Context, memory *Memory) error {
	// Generate embedding if service is available
	var embeddingVec interface{}
	var embeddingModel *string
	if m.embeddingSvc != nil {
		textToEmbed := memory.Title + " " + memory.Summary + " " + memory.Content
		embedding, err := m.embeddingSvc.GenerateEmbedding(ctx, textToEmbed)
		if err == nil && len(embedding) > 0 {
			// Convert to pgvector format
			embeddingVec = pgvector.NewVector(embedding)
			model := "nomic-embed-text"
			embeddingModel = &model
		}
	}

	// Prepare tags - pgx can handle []string directly
	tags := memory.Tags
	if tags == nil {
		tags = []string{} // Empty slice instead of nil
	}

	// Insert query
	query := `
		INSERT INTO memories (
			user_id, title, summary, content, memory_type, category,
			source_type, source_id, project_id, node_id,
			importance_score, tags, embedding, embedding_model
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	err := m.pool.QueryRow(ctx, query,
		memory.UserID,
		memory.Title,
		memory.Summary,
		memory.Content,
		memory.MemoryType,
		memory.Category,
		memory.SourceType,
		memory.SourceID,
		memory.ProjectID,
		memory.NodeID,
		memory.ImportanceScore,
		tags, // Pass slice directly instead of JSON
		embeddingVec,
		embeddingModel,
	).Scan(&memory.ID, &memory.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}

	return nil
}

// GetMemory retrieves a memory by ID (legacy table)
func (m *MemoryService) GetMemory(ctx context.Context, userID string, memoryID uuid.UUID) (*Memory, error) {
	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, project_id, node_id, importance_score,
		       access_count, is_pinned, tags, created_at, updated_at
		FROM memories
		WHERE id = $1 AND user_id = $2 AND is_active = TRUE
	`

	var memory Memory
	var tagsJSON []byte

	err := m.pool.QueryRow(ctx, query, memoryID, userID).Scan(
		&memory.ID,
		&memory.UserID,
		&memory.Title,
		&memory.Summary,
		&memory.Content,
		&memory.MemoryType,
		&memory.Category,
		&memory.SourceType,
		&memory.SourceID,
		&memory.ProjectID,
		&memory.NodeID,
		&memory.ImportanceScore,
		&memory.AccessCount,
		&memory.IsPinned,
		&tagsJSON,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	// Deserialize tags
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &memory.Tags); err != nil {
			// Log error but continue - tags are not critical
			slog.Warn("failed to unmarshal tags for memory", "memory_id", memoryID, "error", err)
		}
	}

	return &memory, nil
}

// ListMemories retrieves memories for a user (legacy table)
func (m *MemoryService) ListMemories(ctx context.Context, userID string, memoryType *string, limit int) ([]Memory, error) {
	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, project_id, node_id, importance_score,
		       access_count, is_pinned, tags, created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND is_active = TRUE
	`

	args := []interface{}{userID}
	if memoryType != nil && *memoryType != "" {
		query += " AND memory_type = $2"
		args = append(args, *memoryType)
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprint(len(args)+1)
	args = append(args, limit)

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var memory Memory
		var tagsJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.Title,
			&memory.Summary,
			&memory.Content,
			&memory.MemoryType,
			&memory.Category,
			&memory.SourceType,
			&memory.SourceID,
			&memory.ProjectID,
			&memory.NodeID,
			&memory.ImportanceScore,
			&memory.AccessCount,
			&memory.IsPinned,
			&tagsJSON,
			&memory.CreatedAt,
			&memory.UpdatedAt,
		)

		if err != nil {
			continue
		}

		// Deserialize tags
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &memory.Tags); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal tags for memory", "memory_id", memory.ID, "error", err)
			}
		}

		memories = append(memories, memory)
	}

	return memories, nil
}

// SearchByEmbedding performs semantic search on memories using pgvector similarity
func (m *MemoryService) SearchByEmbedding(ctx context.Context, userID string, embedding []float32, limit int) ([]Memory, error) {
	if len(embedding) == 0 {
		return nil, fmt.Errorf("empty embedding provided")
	}

	if limit <= 0 {
		limit = 5
	}

	// Convert embedding to pgvector format
	vec := pgvector.NewVector(embedding)

	// Use pgvector cosine distance operator (<=>) for semantic search
	// Note: Lower distance = higher similarity; <=> uses cosine distance
	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, project_id, node_id, importance_score,
		       access_count, is_pinned, tags, created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND is_active = TRUE AND embedding IS NOT NULL
		ORDER BY embedding <=> $2
		LIMIT $3
	`

	rows, err := m.pool.Query(ctx, query, userID, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories by embedding: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var memory Memory
		var tagsJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.Title,
			&memory.Summary,
			&memory.Content,
			&memory.MemoryType,
			&memory.Category,
			&memory.SourceType,
			&memory.SourceID,
			&memory.ProjectID,
			&memory.NodeID,
			&memory.ImportanceScore,
			&memory.AccessCount,
			&memory.IsPinned,
			&tagsJSON,
			&memory.CreatedAt,
			&memory.UpdatedAt,
		)

		if err != nil {
			continue
		}

		// Deserialize tags
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &memory.Tags); err != nil {
				// Log error but continue
				slog.Warn("failed to unmarshal tags for memory", "memory_id", memory.ID, "error", err)
			}
		}

		memories = append(memories, memory)
	}

	return memories, nil
}
