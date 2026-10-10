package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// recordMemoryAccess logs a memory access event asynchronously.
// Must be called as a goroutine; uses background context to avoid
// binding to the request lifecycle.
func (h *MemoryHandler) recordMemoryAccess(memoryID uuid.UUID, userID, accessType string, conversationID *string, relevanceScore *float64) {
	query := `
		INSERT INTO memory_access_log (memory_id, user_id, access_type, conversation_id, relevance_score)
		VALUES ($1, $2, $3, $4, $5)
	`
	var convID *uuid.UUID
	if conversationID != nil {
		if parsed, err := uuid.Parse(*conversationID); err == nil {
			convID = &parsed
		}
	}
	h.pool.Exec(context.Background(), query, memoryID, userID, accessType, convID, relevanceScore)
}

// scanMemoryRow scans a pgx.Rows cursor into a MemoryResponse.
// Used in multi-row queries (ListMemories, GetProjectMemories, etc.).
func scanMemoryRow(rows pgx.Rows) (MemoryResponse, error) {
	var m MemoryResponse
	var id, sourceID, projectID, nodeID uuid.UUID
	var sourceIDValid, projectIDValid, nodeIDValid bool
	var lastAccessed, expires *time.Time
	var tags, metadata []byte

	err := rows.Scan(
		&id, &m.UserID, &m.Title, &m.Summary, &m.Content, &m.MemoryType, &m.Category,
		&m.SourceType, &sourceID, &m.SourceContext, &projectID, &nodeID,
		&m.ImportanceScore, &m.AccessCount, &lastAccessed,
		&m.IsActive, &m.IsPinned, &expires, &tags, &metadata,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return m, err
	}

	m.ID = id.String()
	if sourceIDValid {
		s := sourceID.String()
		m.SourceID = &s
	}
	if projectIDValid {
		s := projectID.String()
		m.ProjectID = &s
	}
	if nodeIDValid {
		s := nodeID.String()
		m.NodeID = &s
	}
	if lastAccessed != nil {
		s := lastAccessed.Format(time.RFC3339)
		m.LastAccessedAt = &s
	}
	if expires != nil {
		s := expires.Format(time.RFC3339)
		m.ExpiresAt = &s
	}

	json.Unmarshal(tags, &m.Tags)
	if m.Tags == nil {
		m.Tags = []string{}
	}
	json.Unmarshal(metadata, &m.Metadata)
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	return m, nil
}

// scanMemoryRowSingle scans a single pgx.Row (from QueryRow) into a MemoryResponse.
// Used in single-row queries: CreateMemory, GetMemory, UpdateMemory, PinMemory.
func scanMemoryRowSingle(row pgx.Row) (MemoryResponse, error) {
	var m MemoryResponse
	var id uuid.UUID
	var sourceID, projectID, nodeID *uuid.UUID
	var lastAccessed, expires *time.Time
	var tags, metadata []byte
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&id, &m.UserID, &m.Title, &m.Summary, &m.Content, &m.MemoryType, &m.Category,
		&m.SourceType, &sourceID, &m.SourceContext, &projectID, &nodeID,
		&m.ImportanceScore, &m.AccessCount, &lastAccessed,
		&m.IsActive, &m.IsPinned, &expires, &tags, &metadata,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return m, err
	}

	m.ID = id.String()
	m.CreatedAt = createdAt.Format(time.RFC3339)
	m.UpdatedAt = updatedAt.Format(time.RFC3339)

	if sourceID != nil {
		s := sourceID.String()
		m.SourceID = &s
	}
	if projectID != nil {
		s := projectID.String()
		m.ProjectID = &s
	}
	if nodeID != nil {
		s := nodeID.String()
		m.NodeID = &s
	}
	if lastAccessed != nil {
		s := lastAccessed.Format(time.RFC3339)
		m.LastAccessedAt = &s
	}
	if expires != nil {
		s := expires.Format(time.RFC3339)
		m.ExpiresAt = &s
	}

	json.Unmarshal(tags, &m.Tags)
	if m.Tags == nil {
		m.Tags = []string{}
	}
	json.Unmarshal(metadata, &m.Metadata)
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	return m, nil
}

// scanMemoryRowWithExtra scans a pgx.Rows cursor into a MemoryResponse plus one
// extra float64 column (similarity or relevance_score) appended at the end of
// the SELECT list.
func scanMemoryRowWithExtra(rows pgx.Rows, extra *float64) (MemoryResponse, error) {
	var m MemoryResponse
	var id uuid.UUID
	var sourceID, projectID, nodeID *uuid.UUID
	var lastAccessed, expires *time.Time
	var tags, metadata []byte
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&id, &m.UserID, &m.Title, &m.Summary, &m.Content, &m.MemoryType, &m.Category,
		&m.SourceType, &sourceID, &m.SourceContext, &projectID, &nodeID,
		&m.ImportanceScore, &m.AccessCount, &lastAccessed,
		&m.IsActive, &m.IsPinned, &expires, &tags, &metadata,
		&createdAt, &updatedAt, extra,
	)
	if err != nil {
		return m, err
	}

	m.ID = id.String()
	m.CreatedAt = createdAt.Format(time.RFC3339)
	m.UpdatedAt = updatedAt.Format(time.RFC3339)

	if sourceID != nil {
		s := sourceID.String()
		m.SourceID = &s
	}
	if projectID != nil {
		s := projectID.String()
		m.ProjectID = &s
	}
	if nodeID != nil {
		s := nodeID.String()
		m.NodeID = &s
	}
	if lastAccessed != nil {
		s := lastAccessed.Format(time.RFC3339)
		m.LastAccessedAt = &s
	}
	if expires != nil {
		s := expires.Format(time.RFC3339)
		m.ExpiresAt = &s
	}

	json.Unmarshal(tags, &m.Tags)
	if m.Tags == nil {
		m.Tags = []string{}
	}
	json.Unmarshal(metadata, &m.Metadata)
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	return m, nil
}

// scanUserFactRow scans a pgx.Rows cursor into a UserFactResponse.
func scanUserFactRow(rows pgx.Rows) (UserFactResponse, error) {
	var f UserFactResponse
	var id uuid.UUID
	var sourceMemoryID *uuid.UUID
	var lastConfirmed *time.Time
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&id, &f.UserID, &f.FactKey, &f.FactValue, &f.FactType,
		&sourceMemoryID, &f.ConfidenceScore, &f.IsActive,
		&lastConfirmed, &createdAt, &updatedAt,
	)
	if err != nil {
		return f, err
	}

	f.ID = id.String()
	f.CreatedAt = createdAt.Format(time.RFC3339)
	f.UpdatedAt = updatedAt.Format(time.RFC3339)

	if sourceMemoryID != nil {
		s := sourceMemoryID.String()
		f.SourceMemoryID = &s
	}
	if lastConfirmed != nil {
		s := lastConfirmed.Format(time.RFC3339)
		f.LastConfirmedAt = &s
	}

	return f, nil
}

// scanUserFactRowSingle scans a single pgx.Row into a UserFactResponse.
func scanUserFactRowSingle(row pgx.Row) (UserFactResponse, error) {
	var f UserFactResponse
	var id uuid.UUID
	var sourceMemoryID *uuid.UUID
	var lastConfirmed *time.Time
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&id, &f.UserID, &f.FactKey, &f.FactValue, &f.FactType,
		&sourceMemoryID, &f.ConfidenceScore, &f.IsActive,
		&lastConfirmed, &createdAt, &updatedAt,
	)
	if err != nil {
		return f, err
	}

	f.ID = id.String()
	f.CreatedAt = createdAt.Format(time.RFC3339)
	f.UpdatedAt = updatedAt.Format(time.RFC3339)

	if sourceMemoryID != nil {
		s := sourceMemoryID.String()
		f.SourceMemoryID = &s
	}
	if lastConfirmed != nil {
		s := lastConfirmed.Format(time.RFC3339)
		f.LastConfirmedAt = &s
	}

	return f, nil
}
