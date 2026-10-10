package services

import (
	"context"
	"encoding/json"
	"fmt"
)

// saveMemory saves an extracted memory to the database
func (s *MemoryExtractorService) saveMemory(ctx context.Context, memory *ExtractedMemory) error {
	tagsJSON, _ := json.Marshal(memory.Tags)
	entitiesJSON, _ := json.Marshal(memory.Entities)
	relatedJSON, _ := json.Marshal(memory.RelatedTo)
	metadataJSON, _ := json.Marshal(memory.Metadata)

	// Add entities and related to metadata
	if memory.Metadata == nil {
		memory.Metadata = make(map[string]interface{})
	}
	memory.Metadata["entities"] = memory.Entities
	memory.Metadata["related_to"] = memory.RelatedTo
	metadataJSON, _ = json.Marshal(memory.Metadata)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO memories (id, user_id, memory_type, content, summary, tags, importance_score, source_type, source_id, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (id) DO UPDATE SET
		    content = EXCLUDED.content,
		    summary = EXCLUDED.summary,
		    tags = EXCLUDED.tags,
		    metadata = EXCLUDED.metadata`,
		memory.ID, memory.UserID, string(memory.Type), memory.Content, memory.Summary,
		tagsJSON, memory.Importance, string(memory.Source), memory.SourceID, metadataJSON, memory.ExtractedAt)

	// Store additional details in a separate table if needed
	_ = entitiesJSON
	_ = relatedJSON

	return err
}

// GetExtractedMemories retrieves extracted memories for a user
func (s *MemoryExtractorService) GetExtractedMemories(ctx context.Context, userID string, memoryType string, limit int) ([]ExtractedMemory, error) {
	query := `SELECT id, user_id, memory_type, content, summary, tags, importance_score, source_type, source_id, metadata, created_at
	          FROM memories WHERE user_id = $1`
	args := []interface{}{userID}

	if memoryType != "" {
		query += " AND memory_type = $2"
		args = append(args, memoryType)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := make([]ExtractedMemory, 0)
	for rows.Next() {
		var m ExtractedMemory
		var tagsJSON, metadataJSON []byte
		var source, sourceID, summary *string

		err := rows.Scan(&m.ID, &m.UserID, &m.Type, &m.Content, &summary, &tagsJSON,
			&m.Importance, &source, &sourceID, &metadataJSON, &m.ExtractedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(tagsJSON, &m.Tags)
		json.Unmarshal(metadataJSON, &m.Metadata)
		if source != nil {
			m.Source = MemorySource(*source)
		}
		if sourceID != nil {
			m.SourceID = *sourceID
		}
		if summary != nil {
			m.Summary = *summary
		}

		memories = append(memories, m)
	}

	return memories, nil
}
