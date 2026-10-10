package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
)

// =============================================================================
// CHATGPT IMPORT
// =============================================================================

// ImportChatGPTConversations imports conversations from a ChatGPT export file
func (s *ImportService) ImportChatGPTConversations(ctx context.Context, userID string, reader io.Reader, filename string) (*ImportResult, error) {
	// Parse the export file
	var export ChatGPTExport
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&export); err != nil {
		return nil, fmt.Errorf("failed to parse ChatGPT export: %w", err)
	}

	// Create import job
	job, err := s.CreateImportJob(ctx, CreateImportJobInput{
		UserID:           userID,
		SourceType:       sqlc.ImportSourceTypeChatgptExport,
		OriginalFilename: filename,
		TargetModule:     "conversations",
		ImportOptions:    map[string]any{"format": "chatgpt", "version": "1.0"},
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
			SourceType: sqlc.ImportSourceTypeChatgptExport,
			ExternalID: conv.ID,
		})
		if exists {
			result.SkippedRecords++
			continue
		}

		// Convert ChatGPT messages to normalized format
		messages, model := s.parseChatGPTMessages(conv)
		messagesJSON, _ := json.Marshal(messages)

		// Build search content
		searchContent := s.buildSearchContent(conv.Title, messages)

		// Calculate data hash for deduplication
		dataHash := s.hashData(messagesJSON)

		// Create imported conversation
		imported, err := s.queries.CreateImportedConversation(ctx, sqlc.CreateImportedConversationParams{
			UserID:                 userID,
			ImportJobID:            job.ID,
			SourceType:             sqlc.ImportSourceTypeChatgptExport,
			ExternalConversationID: ptrIfNotEmpty(conv.ID),
			Title:                  ptrIfNotEmpty(conv.Title),
			Model:                  ptrIfNotEmpty(model),
			Messages:               messagesJSON,
			MessageCount:           ptr(int32(len(messages))),
			OriginalCreatedAt:      s.unixToTimestamp(conv.CreateTime),
			OriginalUpdatedAt:      s.unixToTimestamp(conv.UpdateTime),
			Metadata:               nil,
			SearchContent:          ptrIfNotEmpty(searchContent),
			Tags:                   []string{},
		})
		if err != nil {
			errors = append(errors, ImportError{
				RecordIndex: i,
				ExternalID:  conv.ID,
				Error:       err.Error(),
			})
			result.FailedRecords++
			continue
		}

		// Track the imported record for deduplication
		s.queries.CreateImportedRecord(ctx, sqlc.CreateImportedRecordParams{
			UserID:           userID,
			ImportJobID:      job.ID,
			SourceType:       sqlc.ImportSourceTypeChatgptExport,
			ExternalID:       conv.ID,
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

// parseChatGPTMessages extracts messages from ChatGPT's node mapping
func (s *ImportService) parseChatGPTMessages(conv ChatGPTConversation) ([]NormalizedMessage, string) {
	var messages []NormalizedMessage
	var model string

	// Find the root node and traverse the conversation
	// ChatGPT stores messages in a tree structure via parent/children references
	visited := make(map[string]bool)

	// Build ordered message list by traversing from root
	var traverseNode func(nodeID string)
	traverseNode = func(nodeID string) {
		if visited[nodeID] {
			return
		}
		visited[nodeID] = true

		node, ok := conv.Mapping[nodeID]
		if !ok || node.Message == nil {
			// Continue to children even if this node has no message
			for _, childID := range node.Children {
				traverseNode(childID)
			}
			return
		}

		msg := node.Message

		// Skip system messages and tool messages typically
		if msg.Author.Role == "user" || msg.Author.Role == "assistant" {
			content := s.extractChatGPTContent(msg.Content)
			if content != "" {
				normalized := NormalizedMessage{
					Role:    msg.Author.Role,
					Content: content,
				}

				if msg.CreateTime != nil {
					t := time.Unix(int64(*msg.CreateTime), 0)
					normalized.Timestamp = &t
				}

				// Extract model from metadata if available
				if msg.Metadata != nil {
					if m, ok := msg.Metadata["model_slug"].(string); ok && model == "" {
						model = m
					}
				}

				messages = append(messages, normalized)
			}
		}

		// Process children in order
		for _, childID := range node.Children {
			traverseNode(childID)
		}
	}

	// Find root node (node with no parent)
	for nodeID, node := range conv.Mapping {
		if node.Parent == nil {
			traverseNode(nodeID)
			break
		}
	}

	return messages, model
}

// extractChatGPTContent extracts text content from ChatGPT's content structure
func (s *ImportService) extractChatGPTContent(content ChatGPTContent) string {
	var parts []string
	for _, part := range content.Parts {
		switch v := part.(type) {
		case string:
			if v != "" {
				parts = append(parts, v)
			}
		case map[string]any:
			// Handle structured content (e.g., images, code blocks)
			if text, ok := v["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}
