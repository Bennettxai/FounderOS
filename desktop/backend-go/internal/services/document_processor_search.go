package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ============================================================================
// Semantic Search
// ============================================================================

// SearchDocuments performs semantic search on documents
func (p *DocumentProcessor) SearchDocuments(ctx context.Context, userID, query string, limit int, projectID, nodeID *uuid.UUID) ([]DocumentSearchResult, error) {
	if p.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not available")
	}

	if limit <= 0 {
		limit = 10
	}

	// Generate query embedding
	queryEmbedding, err := p.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	vec := pgvector.NewVector(queryEmbedding)

	// Search chunks
	queryStr := `
		SELECT dc.id, dc.document_id, dc.content, dc.page_number, dc.section_title,
		       ud.display_name, ud.document_type,
		       1 - (dc.embedding <=> $1) as similarity
		FROM document_chunks dc
		JOIN uploaded_documents ud ON dc.document_id = ud.id
		WHERE ud.user_id = $2 AND dc.embedding IS NOT NULL
	`
	args := []interface{}{vec, userID}
	argIdx := 3

	if projectID != nil {
		queryStr += fmt.Sprintf(" AND ud.project_id = $%d", argIdx)
		args = append(args, *projectID)
		argIdx++
	}

	if nodeID != nil {
		queryStr += fmt.Sprintf(" AND ud.node_id = $%d", argIdx)
		args = append(args, *nodeID)
		argIdx++
	}

	queryStr += fmt.Sprintf(" ORDER BY dc.embedding <=> $1 LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := p.pool.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	var results []DocumentSearchResult
	for rows.Next() {
		var r DocumentSearchResult
		var displayName, docType *string

		err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.ChunkContent, &r.PageNumber,
			&r.SectionTitle, &displayName, &docType, &r.RelevanceScore)
		if err != nil {
			continue
		}

		if displayName != nil {
			r.DocumentTitle = *displayName
		}
		if docType != nil {
			r.DocumentType = *docType
		}

		results = append(results, r)
	}

	return results, nil
}

// GetRelevantChunks retrieves chunks most relevant to a context
func (p *DocumentProcessor) GetRelevantChunks(ctx context.Context, userID, contextText string, limit int) ([]DocumentChunk, error) {
	if p.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not available")
	}

	if limit <= 0 {
		limit = 5
	}

	// Generate embedding
	embedding, err := p.embeddingService.GenerateEmbedding(ctx, contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	vec := pgvector.NewVector(embedding)

	rows, err := p.pool.Query(ctx, `
		SELECT dc.id, dc.document_id, dc.chunk_index, dc.content, dc.token_count,
		       dc.page_number, dc.start_char, dc.end_char, dc.section_title, dc.chunk_type
		FROM document_chunks dc
		JOIN uploaded_documents ud ON dc.document_id = ud.id
		WHERE ud.user_id = $1 AND dc.embedding IS NOT NULL
		ORDER BY dc.embedding <=> $2
		LIMIT $3
	`, userID, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var chunks []DocumentChunk
	for rows.Next() {
		var c DocumentChunk
		err := rows.Scan(&c.ID, &c.DocumentID, &c.ChunkIndex, &c.Content, &c.TokenCount,
			&c.PageNumber, &c.StartChar, &c.EndChar, &c.SectionTitle, &c.ChunkType)
		if err != nil {
			continue
		}
		chunks = append(chunks, c)
	}

	return chunks, nil
}
