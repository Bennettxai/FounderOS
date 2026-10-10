package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// EmbeddingService handles vector embeddings for semantic search
type EmbeddingService struct {
	pool              *pgxpool.Pool
	ollamaURL         string
	model             string
	httpClient        *http.Client
	dimensions        int
	cache             *RAGCacheService            // Optional cache for embeddings (legacy)
	embeddingCache    *EmbeddingCacheAdapter      // New embedding cache with Redis backend
}

// EmbeddingBlock represents a document block for embedding
type EmbeddingBlock struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// OllamaEmbedRequest represents a request to Ollama embedding API
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse represents a response from Ollama embedding API
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// SearchResult represents a semantic search result
type SearchResult struct {
	ContextID   uuid.UUID  `json:"context_id"`
	BlockID     string     `json:"block_id"`
	BlockType   string     `json:"block_type"`
	Content     string     `json:"content"`
	ContextName string     `json:"context_name"`
	ContextType string     `json:"context_type"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Similarity  float64    `json:"similarity"`
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(pool *pgxpool.Pool, ollamaURL string) *EmbeddingService {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	return &EmbeddingService{
		pool:      pool,
		ollamaURL: ollamaURL,
		model:     "nomic-embed-text", // 768 dimensions
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		dimensions: 768,
	}
}

// NewEmbeddingServiceWithCache creates a new embedding service with cache
func NewEmbeddingServiceWithCache(pool *pgxpool.Pool, ollamaURL string, embeddingCache *EmbeddingCacheAdapter) *EmbeddingService {
	service := NewEmbeddingService(pool, ollamaURL)
	service.embeddingCache = embeddingCache
	return service
}

// SetCache sets the cache service for embedding caching (legacy)
func (s *EmbeddingService) SetCache(cache *RAGCacheService) {
	s.cache = cache
}

// SetEmbeddingCache sets the new embedding cache service
func (s *EmbeddingService) SetEmbeddingCache(cache *EmbeddingCacheAdapter) {
	s.embeddingCache = cache
}

// GenerateEmbedding generates an embedding for text using Ollama
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Clean and truncate text if too long (nomic-embed-text has 8192 token limit)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text for embedding")
	}

	// Simple truncation - in production might want smarter chunking
	if len(text) > 8000 {
		text = text[:8000]
	}

	// Check new embedding cache first (higher priority)
	if s.embeddingCache != nil {
		if cached, found, err := s.embeddingCache.GetEmbedding(ctx, text, "text"); err == nil && found {
			return cached, nil
		}
	}

	// Fallback to legacy cache
	if s.cache != nil {
		if cached, err := s.cache.GetEmbedding(ctx, text); err == nil && cached != nil {
			return cached, nil
		}
	}

	reqBody, err := json.Marshal(OllamaEmbedRequest{
		Model:  s.model,
		Prompt: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var result OllamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	// Cache the embedding for future use (new cache with 24h TTL for text)
	if s.embeddingCache != nil {
		_ = s.embeddingCache.SetEmbedding(ctx, text, result.Embedding, "text", 24*time.Hour)
	}

	// Also cache in legacy cache for backward compatibility
	if s.cache != nil {
		_ = s.cache.SetEmbedding(ctx, text, result.Embedding) // Ignore cache errors
	}

	return result.Embedding, nil
}

// IndexDocument indexes all blocks of a document
func (s *EmbeddingService) IndexDocument(ctx context.Context, contextID uuid.UUID, blocks []Block) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing embeddings for this document
	_, err = tx.Exec(ctx, `DELETE FROM context_embeddings WHERE context_id = $1`, contextID)
	if err != nil {
		return fmt.Errorf("delete existing embeddings: %w", err)
	}

	// Track successful embeddings
	embeddedCount := 0

	// Insert new embeddings for each block with content
	for _, block := range blocks {
		content := strings.TrimSpace(block.Content)
		if content == "" {
			continue
		}

		// Generate embedding for this block
		embedding, err := s.GenerateEmbedding(ctx, content)
		if err != nil {
			// Log but continue with other blocks
			slog.Warn("failed to embed block", "block_id", block.ID, "error", err)
			continue
		}

		// Insert using pgvector
		vec := pgvector.NewVector(embedding)
		_, err = tx.Exec(ctx, `
			INSERT INTO context_embeddings (context_id, block_id, block_type, content, embedding)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (context_id, block_id)
			DO UPDATE SET content = $4, embedding = $5, updated_at = NOW()
		`, contextID, block.ID, block.Type, content, vec)
		if err != nil {
			return fmt.Errorf("insert embedding for block %s: %w", block.ID, err)
		}
		embeddedCount++
	}

	// Update context embedding status
	_, err = tx.Exec(ctx, `
		UPDATE contexts
		SET embedding_status = 'indexed',
		    last_embedded_at = NOW(),
		    embedding_count = $2
		WHERE id = $1
	`, contextID, embeddedCount)
	if err != nil {
		return fmt.Errorf("update context status: %w", err)
	}

	return tx.Commit(ctx)
}

// SimilaritySearch finds similar content blocks
func (s *EmbeddingService) SimilaritySearch(ctx context.Context, query string, limit int, userID string) ([]SearchResult, error) {
	queryEmbedding, err := s.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}

	vec := pgvector.NewVector(queryEmbedding)

	rows, err := s.pool.Query(ctx, `
		SELECT
			ce.context_id,
			ce.block_id,
			ce.block_type,
			ce.content,
			c.name as context_name,
			c.type::text as context_type,
			c.parent_id,
			1 - (ce.embedding <=> $1) as similarity
		FROM context_embeddings ce
		JOIN contexts c ON c.id = ce.context_id
		WHERE c.user_id = $2
		  AND c.is_archived = false
		ORDER BY ce.embedding <=> $1
		LIMIT $3
	`, vec, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query similar: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var parentID pgtype.UUID

		err := rows.Scan(
			&r.ContextID,
			&r.BlockID,
			&r.BlockType,
			&r.Content,
			&r.ContextName,
			&r.ContextType,
			&parentID,
			&r.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		if parentID.Valid {
			id := uuid.UUID(parentID.Bytes)
			r.ParentID = &id
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// ScopedSimilaritySearch finds similar content blocks ONLY within specified context IDs
func (s *EmbeddingService) ScopedSimilaritySearch(ctx context.Context, query string, contextIDs []uuid.UUID, userID string, limit int) ([]RelevantBlock, error) {
	if len(contextIDs) == 0 {
		return nil, nil // No contexts to search
	}

	queryEmbedding, err := s.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	if limit <= 0 {
		limit = 5
	}

	vec := pgvector.NewVector(queryEmbedding)

	rows, err := s.pool.Query(ctx, `
		SELECT
			ce.context_id,
			c.name as document_name,
			ce.content as block_content,
			ce.block_type,
			1 - (ce.embedding <=> $1) as similarity
		FROM context_embeddings ce
		JOIN contexts c ON c.id = ce.context_id
		WHERE c.user_id = $2
		  AND c.is_archived = false
		  AND c.id = ANY($3)
		ORDER BY ce.embedding <=> $1
		LIMIT $4
	`, vec, userID, contextIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("scoped query similar: %w", err)
	}
	defer rows.Close()

	var results []RelevantBlock
	for rows.Next() {
		var r RelevantBlock

		err := rows.Scan(
			&r.ContextID,
			&r.DocumentName,
			&r.BlockContent,
			&r.BlockType,
			&r.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetEmbeddingStats returns statistics about embeddings for a user
func (s *EmbeddingService) GetEmbeddingStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	var totalDocs, indexedDocs, totalBlocks int

	row := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_docs,
			COUNT(*) FILTER (WHERE embedding_status = 'indexed') as indexed_docs
		FROM contexts
		WHERE user_id = $1 AND is_archived = false
	`, userID)

	if err := row.Scan(&totalDocs, &indexedDocs); err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	row = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM context_embeddings ce
		JOIN contexts c ON c.id = ce.context_id
		WHERE c.user_id = $1
	`, userID)

	if err := row.Scan(&totalBlocks); err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	return map[string]interface{}{
		"total_documents":   totalDocs,
		"indexed_documents": indexedDocs,
		"total_blocks":      totalBlocks,
		"model":             s.model,
		"dimensions":        s.dimensions,
	}, nil
}

// HealthCheck verifies Ollama embedding endpoint is working
func (s *EmbeddingService) HealthCheck(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to generate a simple embedding
	_, err := s.GenerateEmbedding(ctx, "test")
	return err == nil
}
