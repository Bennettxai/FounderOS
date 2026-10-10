package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentProcessor handles document processing, chunking, and semantic search
type DocumentProcessor struct {
	pool             *pgxpool.Pool
	embeddingService *EmbeddingService
	logger           *slog.Logger
	storagePath      string
}

// NewDocumentProcessor creates a new document processor
func NewDocumentProcessor(pool *pgxpool.Pool, embeddingService *EmbeddingService, storagePath string) *DocumentProcessor {
	if storagePath == "" {
		storagePath = "./uploads"
	}
	return &DocumentProcessor{
		pool:             pool,
		embeddingService: embeddingService,
		logger:           slog.Default().With("service", "document_processor"),
		storagePath:      storagePath,
	}
}

// ============================================================================
// Types
// ============================================================================

// DocumentUpload represents an uploaded document
type DocumentUpload struct {
	ID               uuid.UUID  `json:"id"`
	UserID           string     `json:"user_id"`
	Filename         string     `json:"filename"`
	OriginalFilename string     `json:"original_filename"`
	DisplayName      string     `json:"display_name,omitempty"`
	Description      string     `json:"description,omitempty"`
	FileType         string     `json:"file_type"`
	MimeType         string     `json:"mime_type"`
	FileSizeBytes    int64      `json:"file_size_bytes"`
	StoragePath      string     `json:"storage_path"`
	ExtractedText    string     `json:"extracted_text,omitempty"`
	PageCount        int        `json:"page_count,omitempty"`
	WordCount        int        `json:"word_count,omitempty"`
	DocumentType     string     `json:"document_type,omitempty"`
	Category         string     `json:"category,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty"`
	NodeID           *uuid.UUID `json:"node_id,omitempty"`
	ProcessingStatus string     `json:"processing_status"`
	ProcessingError  string     `json:"processing_error,omitempty"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// DocumentChunk represents a chunk of a document
type DocumentChunk struct {
	ID           uuid.UUID `json:"id"`
	DocumentID   uuid.UUID `json:"document_id"`
	ChunkIndex   int       `json:"chunk_index"`
	Content      string    `json:"content"`
	TokenCount   int       `json:"token_count"`
	PageNumber   *int      `json:"page_number,omitempty"`
	StartChar    int       `json:"start_char"`
	EndChar      int       `json:"end_char"`
	SectionTitle string    `json:"section_title,omitempty"`
	ChunkType    string    `json:"chunk_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProcessDocumentInput contains parameters for processing a document
type ProcessDocumentInput struct {
	UserID           string
	Filename         string
	OriginalFilename string
	DisplayName      string
	Description      string
	MimeType         string
	Content          []byte
	ProjectID        *uuid.UUID
	NodeID           *uuid.UUID
	DocumentType     string
	Category         string
	Tags             []string
}

// ChunkingOptions configures how documents are chunked
type ChunkingOptions struct {
	MaxChunkSize    int  // Maximum characters per chunk
	ChunkOverlap    int  // Character overlap between chunks
	PreserveHeaders bool // Try to keep headers with their content
	SplitOnHeaders  bool // Split at markdown headers
}

// DefaultChunkingOptions returns sensible defaults
func DefaultChunkingOptions() ChunkingOptions {
	return ChunkingOptions{
		MaxChunkSize:    1500,
		ChunkOverlap:    200,
		PreserveHeaders: true,
		SplitOnHeaders:  true,
	}
}

// DocumentSearchResult represents a search result
type DocumentSearchResult struct {
	DocumentID     uuid.UUID `json:"document_id"`
	ChunkID        uuid.UUID `json:"chunk_id"`
	DocumentTitle  string    `json:"document_title"`
	ChunkContent   string    `json:"chunk_content"`
	RelevanceScore float64   `json:"relevance_score"`
	PageNumber     *int      `json:"page_number,omitempty"`
	SectionTitle   string    `json:"section_title,omitempty"`
	DocumentType   string    `json:"document_type,omitempty"`
}
