package services

import (
	"context"
	"fmt"
)

// SmartChunkingService implements intelligent document chunking for RAG optimization
// Implements SORX 2.0 Smart Chunking SKILL
type SmartChunkingService struct {
	// Token estimator (approximation: ~4 chars per token for English)
	charsPerToken float64
}

// Chunk represents a document chunk with metadata
type Chunk struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	TokenCount  int                    `json:"token_count"`
	Position    int                    `json:"position"`
	Metadata    map[string]interface{} `json:"metadata"`
	ParentDocID string                 `json:"parent_doc_id"`
}

// ChunkOptions configures chunking behavior
type ChunkOptions struct {
	ChunkSize    int     // Target chunk size in tokens (default: 512)
	OverlapRatio float64 // Overlap ratio between chunks (default: 0.2 = 20%)
	PreserveCode bool    // Whether to preserve code block boundaries (default: true)
	MinChunkSize int     // Minimum chunk size in tokens (default: 128)
}

// DocumentType represents the type of document being chunked
type DocumentType string

const (
	DocTypeMarkdown  DocumentType = "markdown"
	DocTypeCode      DocumentType = "code"
	DocTypePlainText DocumentType = "plaintext"
	DocTypeJSON      DocumentType = "json"
	DocTypeXML       DocumentType = "xml"
)

// NewSmartChunkingService creates a new smart chunking service
func NewSmartChunkingService() *SmartChunkingService {
	return &SmartChunkingService{
		charsPerToken: 4.0, // Approximate average for English text
	}
}

// DefaultChunkOptions returns sensible defaults
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:    512,
		OverlapRatio: 0.2,
		PreserveCode: true,
		MinChunkSize: 128,
	}
}

// ChunkDocument intelligently chunks a document based on its type
func (s *SmartChunkingService) ChunkDocument(ctx context.Context, content string, docType DocumentType, parentDocID string, options ChunkOptions) ([]Chunk, error) {
	if content == "" {
		return nil, fmt.Errorf("empty content for chunking")
	}

	// Apply defaults
	if options.ChunkSize == 0 {
		options.ChunkSize = 512
	}
	if options.OverlapRatio == 0 {
		options.OverlapRatio = 0.2
	}
	if options.MinChunkSize == 0 {
		options.MinChunkSize = 128
	}

	// Validate options
	if options.ChunkSize < options.MinChunkSize {
		return nil, fmt.Errorf("chunk_size (%d) must be >= min_chunk_size (%d)", options.ChunkSize, options.MinChunkSize)
	}
	if options.OverlapRatio < 0 || options.OverlapRatio >= 1 {
		return nil, fmt.Errorf("overlap_ratio must be between 0 and 1")
	}

	// Route to appropriate chunking strategy
	switch docType {
	case DocTypeMarkdown:
		return s.ChunkMarkdown(ctx, content, parentDocID, options)
	case DocTypeCode:
		return s.ChunkCode(ctx, content, "", parentDocID, options) // Language detection happens inside
	case DocTypePlainText:
		return s.ChunkPlainText(ctx, content, parentDocID, options)
	case DocTypeJSON, DocTypeXML:
		return s.ChunkStructured(ctx, content, string(docType), parentDocID, options)
	default:
		// Default to plain text chunking
		return s.ChunkPlainText(ctx, content, parentDocID, options)
	}
}
