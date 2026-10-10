package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Helper Methods
// ============================================================================

func (p *DocumentProcessor) getFileType(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "pdf"):
		return "pdf"
	case strings.Contains(mimeType, "markdown") || strings.HasSuffix(mimeType, "md"):
		return "markdown"
	case strings.Contains(mimeType, "word") || strings.Contains(mimeType, "docx"):
		return "docx"
	case strings.Contains(mimeType, "text/plain"):
		return "txt"
	case strings.Contains(mimeType, "image"):
		return "image"
	default:
		return ""
	}
}

func (p *DocumentProcessor) saveFile(userID, docID, filename string, content []byte) (string, error) {
	// Create user directory
	userDir := filepath.Join(p.storagePath, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", err
	}

	// Generate unique filename
	ext := filepath.Ext(filename)
	storageName := fmt.Sprintf("%s%s", docID, ext)
	storagePath := filepath.Join(userDir, storageName)

	// Write file
	if err := os.WriteFile(storagePath, content, 0644); err != nil {
		return "", err
	}

	return storagePath, nil
}

func (p *DocumentProcessor) countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func (p *DocumentProcessor) estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English
	return len(text) / 4
}

// ============================================================================
// Document CRUD
// ============================================================================

// GetDocument retrieves a document by ID
func (p *DocumentProcessor) GetDocument(ctx context.Context, userID string, docID uuid.UUID) (*DocumentUpload, error) {
	var doc DocumentUpload
	var processedAt *time.Time

	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, filename, original_filename, display_name, description,
		       file_type, mime_type, file_size_bytes, storage_path, extracted_text,
		       page_count, word_count, project_id, node_id, document_type, category, tags,
		       processing_status, processing_error, processed_at, created_at, updated_at
		FROM uploaded_documents
		WHERE id = $1 AND user_id = $2
	`, docID, userID).Scan(
		&doc.ID, &doc.UserID, &doc.Filename, &doc.OriginalFilename, &doc.DisplayName, &doc.Description,
		&doc.FileType, &doc.MimeType, &doc.FileSizeBytes, &doc.StoragePath, &doc.ExtractedText,
		&doc.PageCount, &doc.WordCount, &doc.ProjectID, &doc.NodeID, &doc.DocumentType, &doc.Category, &doc.Tags,
		&doc.ProcessingStatus, &doc.ProcessingError, &processedAt, &doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	doc.ProcessedAt = processedAt
	return &doc, nil
}

// GetDocumentContent reads the file content
func (p *DocumentProcessor) GetDocumentContent(ctx context.Context, userID string, docID uuid.UUID) ([]byte, error) {
	doc, err := p.GetDocument(ctx, userID, docID)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(doc.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// DeleteDocument deletes a document and its chunks
func (p *DocumentProcessor) DeleteDocument(ctx context.Context, userID string, docID uuid.UUID) error {
	// Get document for file path
	doc, err := p.GetDocument(ctx, userID, docID)
	if err != nil {
		return err
	}

	// Delete from database (chunks deleted via cascade)
	_, err = p.pool.Exec(ctx, `DELETE FROM uploaded_documents WHERE id = $1 AND user_id = $2`, docID, userID)
	if err != nil {
		return err
	}

	// Delete file
	if doc.StoragePath != "" {
		os.Remove(doc.StoragePath)
	}

	return nil
}

// ReprocessDocument reprocesses an existing document
func (p *DocumentProcessor) ReprocessDocument(ctx context.Context, userID string, docID uuid.UUID) error {
	// Get document
	doc, err := p.GetDocument(ctx, userID, docID)
	if err != nil {
		return err
	}

	// Read content
	content, err := os.ReadFile(doc.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Delete existing chunks
	p.pool.Exec(ctx, `DELETE FROM document_chunks WHERE document_id = $1`, docID)

	// Update status
	p.pool.Exec(ctx, `UPDATE uploaded_documents SET processing_status = 'processing', updated_at = NOW() WHERE id = $1`, docID)

	// Reprocess
	go p.processDocumentAsync(docID, content)

	return nil
}
