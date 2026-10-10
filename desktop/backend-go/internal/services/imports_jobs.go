package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// IMPORT JOB METHODS
// =============================================================================

// CreateImportJob creates a new import job and returns its ID
func (s *ImportService) CreateImportJob(ctx context.Context, input CreateImportJobInput) (*sqlc.ImportJob, error) {
	optionsJSON, err := json.Marshal(input.ImportOptions)
	if err != nil {
		optionsJSON = nil
	}

	var fileSizePtr *int64
	if input.FileSizeBytes > 0 {
		fileSizePtr = &input.FileSizeBytes
	}

	job, err := s.queries.CreateImportJob(ctx, sqlc.CreateImportJobParams{
		UserID:           input.UserID,
		SourceType:       input.SourceType,
		OriginalFilename: ptrIfNotEmpty(input.OriginalFilename),
		FileSizeBytes:    fileSizePtr,
		ContentType:      ptrIfNotEmpty(input.ContentType),
		TargetModule:     input.TargetModule,
		TargetEntity:     nil,
		FieldMapping:     nil,
		TransformRules:   nil,
		ImportOptions:    optionsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	return &job, nil
}

// GetImportJob retrieves an import job by ID
func (s *ImportService) GetImportJob(ctx context.Context, userID string, jobID uuid.UUID) (*sqlc.ImportJob, error) {
	job, err := s.queries.GetImportJob(ctx, sqlc.GetImportJobParams{
		ID:     pgtype.UUID{Bytes: jobID, Valid: true},
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetUserImportJobs retrieves import jobs for a user with pagination
func (s *ImportService) GetUserImportJobs(ctx context.Context, userID string, limit, offset int32) ([]sqlc.ImportJob, error) {
	return s.queries.GetImportJobsByUser(ctx, sqlc.GetImportJobsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// UpdateImportProgress updates the progress of an import job
func (s *ImportService) UpdateImportProgress(ctx context.Context, userID string, jobID uuid.UUID, progress ImportProgress) error {
	return s.queries.UpdateImportJobProgress(ctx, sqlc.UpdateImportJobProgressParams{
		ID:               pgtype.UUID{Bytes: jobID, Valid: true},
		UserID:           userID,
		ProgressPercent:  ptr(int32(progress.ProgressPercent)),
		ProcessedRecords: ptr(int32(progress.ProcessedRecords)),
		ImportedRecords:  ptr(int32(progress.ImportedRecords)),
		SkippedRecords:   ptr(int32(progress.SkippedRecords)),
		FailedRecords:    ptr(int32(progress.FailedRecords)),
	})
}

// FailImportJob marks an import job as failed
func (s *ImportService) FailImportJob(ctx context.Context, userID string, jobID uuid.UUID, errMsg string, details map[string]any) error {
	detailsJSON, _ := json.Marshal(details)
	return s.queries.FailImportJob(ctx, sqlc.FailImportJobParams{
		ID:           pgtype.UUID{Bytes: jobID, Valid: true},
		UserID:       userID,
		ErrorMessage: ptrIfNotEmpty(errMsg),
		ErrorDetails: detailsJSON,
	})
}

// CompleteImportJob marks an import job as completed
func (s *ImportService) CompleteImportJob(ctx context.Context, userID string, jobID uuid.UUID, summary map[string]any) error {
	summaryJSON, _ := json.Marshal(summary)
	return s.queries.CompleteImportJob(ctx, sqlc.CompleteImportJobParams{
		ID:            pgtype.UUID{Bytes: jobID, Valid: true},
		UserID:        userID,
		ResultSummary: summaryJSON,
	})
}
