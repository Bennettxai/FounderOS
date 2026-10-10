package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ============================================================================
// Document Processing
// ============================================================================

// ProcessDocument processes an uploaded document
func (p *DocumentProcessor) ProcessDocument(ctx context.Context, input ProcessDocumentInput) (*DocumentUpload, error) {
	p.logger.Info("processing document", "filename", input.OriginalFilename, "size", len(input.Content))

	// Determine file type from mime type
	fileType := p.getFileType(input.MimeType)
	if fileType == "" {
		return nil, fmt.Errorf("unsupported file type: %s", input.MimeType)
	}

	// Create document record
	doc := &DocumentUpload{
		ID:               uuid.New(),
		UserID:           input.UserID,
		Filename:         input.Filename,
		OriginalFilename: input.OriginalFilename,
		DisplayName:      input.DisplayName,
		Description:      input.Description,
		FileType:         fileType,
		MimeType:         input.MimeType,
		FileSizeBytes:    int64(len(input.Content)),
		ProjectID:        input.ProjectID,
		NodeID:           input.NodeID,
		DocumentType:     input.DocumentType,
		Category:         input.Category,
		Tags:             input.Tags,
		ProcessingStatus: "processing",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save file to storage
	storagePath, err := p.saveFile(input.UserID, doc.ID.String(), input.Filename, input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	doc.StoragePath = storagePath

	// Insert document record
	if err := p.insertDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Process document asynchronously
	go p.processDocumentAsync(doc.ID, input.Content)

	return doc, nil
}

// processDocumentAsync processes document content in background
func (p *DocumentProcessor) processDocumentAsync(docID uuid.UUID, content []byte) {
	ctx := context.Background()

	// Extract text
	extractedText, pageCount, err := p.extractText(content)
	if err != nil {
		p.updateProcessingStatus(ctx, docID, "failed", err.Error())
		return
	}

	// Count words
	wordCount := p.countWords(extractedText)

	// Update document with extracted text
	_, err = p.pool.Exec(ctx, `
		UPDATE uploaded_documents
		SET extracted_text = $1, page_count = $2, word_count = $3, updated_at = NOW()
		WHERE id = $4
	`, extractedText, pageCount, wordCount, docID)
	if err != nil {
		p.logger.Error("failed to update extracted text", "error", err, "doc_id", docID)
	}

	// Generate embedding for the whole document
	if p.embeddingService != nil && len(extractedText) > 0 {
		// Use summary or first 8000 chars for document-level embedding
		textForEmbedding := extractedText
		if len(textForEmbedding) > 8000 {
			textForEmbedding = textForEmbedding[:8000]
		}

		embedding, err := p.embeddingService.GenerateEmbedding(ctx, textForEmbedding)
		if err == nil && len(embedding) > 0 {
			vec := pgvector.NewVector(embedding)
			if _, execErr := p.pool.Exec(ctx, `UPDATE uploaded_documents SET embedding = $1 WHERE id = $2`, vec, docID); execErr != nil {
				p.logger.Warn("failed to store document embedding", "error", execErr, "doc_id", docID)
			}
		}
	}

	// Chunk the document
	chunks := p.chunkDocument(extractedText, DefaultChunkingOptions())

	// Insert chunks and generate embeddings
	for i, chunk := range chunks {
		chunkID := uuid.New()

		// Insert chunk
		_, err := p.pool.Exec(ctx, `
			INSERT INTO document_chunks (id, document_id, chunk_index, content, token_count, start_char, end_char, section_title, chunk_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, chunkID, docID, i, chunk.Content, chunk.TokenCount, chunk.StartChar, chunk.EndChar, chunk.SectionTitle, chunk.ChunkType)

		if err != nil {
			p.logger.Error("failed to insert chunk", "error", err, "doc_id", docID, "chunk", i)
			continue
		}

		// Generate chunk embedding
		if p.embeddingService != nil {
			embedding, err := p.embeddingService.GenerateEmbedding(ctx, chunk.Content)
			if err == nil && len(embedding) > 0 {
				vec := pgvector.NewVector(embedding)
				if _, execErr := p.pool.Exec(ctx, `UPDATE document_chunks SET embedding = $1 WHERE id = $2`, vec, chunkID); execErr != nil {
					p.logger.Warn("failed to store chunk embedding", "error", execErr, "doc_id", docID, "chunk_id", chunkID, "chunk", i)
				}
			}
		}
	}

	// Mark as completed
	now := time.Now()
	p.pool.Exec(ctx, `
		UPDATE uploaded_documents SET processing_status = 'completed', processed_at = $1, updated_at = NOW() WHERE id = $2
	`, now, docID)

	p.logger.Info("document processing completed", "doc_id", docID, "chunks", len(chunks), "words", wordCount)
}

func (p *DocumentProcessor) insertDocument(ctx context.Context, doc *DocumentUpload) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO uploaded_documents (
			id, user_id, filename, original_filename, display_name, description,
			file_type, mime_type, file_size_bytes, storage_path,
			project_id, node_id, document_type, category, tags,
			processing_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, doc.ID, doc.UserID, doc.Filename, doc.OriginalFilename, doc.DisplayName, doc.Description,
		doc.FileType, doc.MimeType, doc.FileSizeBytes, doc.StoragePath,
		doc.ProjectID, doc.NodeID, doc.DocumentType, doc.Category, doc.Tags,
		doc.ProcessingStatus, doc.CreatedAt, doc.UpdatedAt)

	return err
}

func (p *DocumentProcessor) updateProcessingStatus(ctx context.Context, docID uuid.UUID, status, errorMsg string) {
	p.pool.Exec(ctx, `
		UPDATE uploaded_documents SET processing_status = $1, processing_error = $2, updated_at = NOW() WHERE id = $3
	`, status, errorMsg, docID)
}
