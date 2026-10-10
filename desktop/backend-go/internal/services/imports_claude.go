package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
)

// =============================================================================
// CLAUDE IMPORT
// =============================================================================

// ImportClaudeConversations imports conversations from a Claude export file
func (s *ImportService) ImportClaudeConversations(ctx context.Context, userID string, reader io.Reader, filename string) (*ImportResult, error) {
	// Parse the export file
	var export ClaudeExport
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&export); err != nil {
		return nil, fmt.Errorf("failed to parse Claude export: %w", err)
	}

	// Create import job
	job, err := s.CreateImportJob(ctx, CreateImportJobInput{
		UserID:           userID,
		SourceType:       sqlc.ImportSourceTypeClaudeExport,
		OriginalFilename: filename,
		TargetModule:     "conversations",
		ImportOptions:    map[string]any{"format": "claude", "version": "1.0"},
	})
	if err != nil {
		return nil, err
	}

	jobID, _ := uuid.FromBytes(job.ID.Bytes[:])
	totalRecords := len(export.Conversations)

	// Update total records
	s.queries.UpdateImportJobTotalRecords(ctx, sqlc.UpdateImportJobTotalRecordsParams{
		ID:           job.ID,
		UserID:       userID,
		TotalRecords: ptr(int32(totalRecords)),
	})

	// Update status to processing
	s.queries.UpdateImportJobStatus(ctx, sqlc.UpdateImportJobStatusParams{
		ID:              job.ID,
		UserID:          userID,
		Status:          sqlc.NullImportStatus{ImportStatus: sqlc.ImportStatusProcessing, Valid: true},
		ProgressPercent: ptr(int32(0)),
	})

	result := &ImportResult{
		JobID:  jobID,
		Status: sqlc.ImportStatusProcessing,
	}
	result.TotalRecords = totalRecords
	var errors []ImportError

	// Process each conversation
	for i, conv := range export.Conversations {
		// Check for duplicate
		exists, _ := s.queries.CheckExternalRecordExists(ctx, sqlc.CheckExternalRecordExistsParams{
			UserID:     userID,
			SourceType: sqlc.ImportSourceTypeClaudeExport,
			ExternalID: conv.UUID,
		})
		if exists {
			result.SkippedRecords++
			continue
		}

		// Convert Claude messages to normalized format
		messages := s.parseClaudeMessages(conv)
		messagesJSON, _ := json.Marshal(messages)

		// Build search content
		searchContent := s.buildSearchContent(conv.Name, messages)

		// Calculate data hash for deduplication
		dataHash := s.hashData(messagesJSON)

		// Parse timestamps
		createdAt := s.parseISO8601(conv.CreatedAt)
		updatedAt := s.parseISO8601(conv.UpdatedAt)

		// Build metadata
		metadata := map[string]any{}
		if conv.Project != nil {
			metadata["project_uuid"] = conv.Project.UUID
			metadata["project_name"] = conv.Project.Name
		}
		metadataJSON, _ := json.Marshal(metadata)

		// Create imported conversation
		imported, err := s.queries.CreateImportedConversation(ctx, sqlc.CreateImportedConversationParams{
			UserID:                 userID,
			ImportJobID:            job.ID,
			SourceType:             sqlc.ImportSourceTypeClaudeExport,
			ExternalConversationID: ptrIfNotEmpty(conv.UUID),
			Title:                  ptrIfNotEmpty(conv.Name),
			Model:                  ptrIfNotEmpty(conv.Model),
			Messages:               messagesJSON,
			MessageCount:           ptr(int32(len(messages))),
			OriginalCreatedAt:      createdAt,
			OriginalUpdatedAt:      updatedAt,
			Metadata:               metadataJSON,
			SearchContent:          ptrIfNotEmpty(searchContent),
			Tags:                   []string{},
		})
		if err != nil {
			errors = append(errors, ImportError{
				RecordIndex: i,
				ExternalID:  conv.UUID,
				Error:       err.Error(),
			})
			result.FailedRecords++
			continue
		}

		// Track the imported record for deduplication
		s.queries.CreateImportedRecord(ctx, sqlc.CreateImportedRecordParams{
			UserID:           userID,
			ImportJobID:      job.ID,
			SourceType:       sqlc.ImportSourceTypeClaudeExport,
			ExternalID:       conv.UUID,
			TargetModule:     "conversations",
			TargetRecordID:   imported.ID,
			ExternalDataHash: ptrIfNotEmpty(dataHash),
		})

		result.ImportedRecords++

		// Update progress every 10 records
		if (i+1)%10 == 0 || i == totalRecords-1 {
			progress := int((float64(i+1) / float64(totalRecords)) * 100)
			s.UpdateImportProgress(ctx, userID, jobID, ImportProgress{
				TotalRecords:     totalRecords,
				ProcessedRecords: i + 1,
				ImportedRecords:  result.ImportedRecords,
				SkippedRecords:   result.SkippedRecords,
				FailedRecords:    result.FailedRecords,
				ProgressPercent:  progress,
			})
		}
	}

	// Complete the job
	result.Errors = errors
	if result.FailedRecords > 0 && result.ImportedRecords == 0 {
		result.Status = sqlc.ImportStatusFailed
		s.FailImportJob(ctx, userID, jobID, "All records failed to import", map[string]any{
			"errors": errors,
		})
	} else {
		result.Status = sqlc.ImportStatusCompleted
		s.CompleteImportJob(ctx, userID, jobID, map[string]any{
			"imported":    result.ImportedRecords,
			"skipped":     result.SkippedRecords,
			"failed":      result.FailedRecords,
			"error_count": len(errors),
		})
	}

	return result, nil
}

// parseClaudeMessages converts Claude messages to normalized format
func (s *ImportService) parseClaudeMessages(conv ClaudeConversation) []NormalizedMessage {
	var messages []NormalizedMessage

	for _, msg := range conv.ChatMessages {
		// Map Claude sender to standard role
		role := "user"
		if msg.Sender == "assistant" {
			role = "assistant"
		}

		normalized := NormalizedMessage{
			Role:    role,
			Content: msg.Text,
		}

		// Parse timestamp if available
		if msg.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, msg.CreatedAt); err == nil {
				normalized.Timestamp = &t
			}
		}

		// Include file information in metadata
		if len(msg.Files) > 0 {
			files := make([]map[string]any, len(msg.Files))
			for i, f := range msg.Files {
				files[i] = map[string]any{
					"file_name": f.FileName,
					"file_type": f.FileType,
					"file_size": f.FileSize,
				}
			}
			normalized.Metadata = map[string]any{"files": files}
		}

		messages = append(messages, normalized)
	}

	return messages
}
