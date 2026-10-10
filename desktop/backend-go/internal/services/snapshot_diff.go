package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// SnapshotDiffQueries defines the query interface needed by SnapshotDiffService
type SnapshotDiffQueries interface {
	ListFilesByApp(ctx context.Context, appID pgtype.UUID) ([]sqlc.OsaModuleFile, error)
	GetFileByPath(ctx context.Context, arg sqlc.GetFileByPathParams) (sqlc.OsaModuleFile, error)
}

// SnapshotDiffService provides snapshot comparison functionality
type SnapshotDiffService struct {
	queries SnapshotDiffQueries
	logger  *slog.Logger
}

// SnapshotInfo contains basic snapshot metadata
type SnapshotInfo struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt string    `json:"created_at"`
	FileCount int       `json:"file_count"`
}

// AddedFile represents a file that exists in snapshot2 but not in snapshot1
type AddedFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// RemovedFile represents a file that exists in snapshot1 but not in snapshot2
type RemovedFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// ModifiedFile represents a file that exists in both snapshots but has different content
type ModifiedFile struct {
	Path      string `json:"path"`
	OldSize   int64  `json:"old_size"`
	NewSize   int64  `json:"new_size"`
	OldSHA256 string `json:"old_sha256"`
	NewSHA256 string `json:"new_sha256"`
	Diff      string `json:"diff,omitempty"`
}

// DiffResult contains all categorized file changes
type DiffResult struct {
	Added     []AddedFile    `json:"added"`
	Removed   []RemovedFile  `json:"removed"`
	Modified  []ModifiedFile `json:"modified"`
	Unchanged int            `json:"unchanged"`
}

// DiffSummary provides aggregate statistics about the diff
type DiffSummary struct {
	TotalChanges int `json:"total_changes"`
	LinesAdded   int `json:"lines_added"`
	LinesRemoved int `json:"lines_removed"`
}

// SnapshotDiffResponse is the complete diff response
type SnapshotDiffResponse struct {
	Snapshot1 SnapshotInfo `json:"snapshot1"`
	Snapshot2 SnapshotInfo `json:"snapshot2"`
	Diff      DiffResult   `json:"diff"`
	Summary   DiffSummary  `json:"summary"`
}

// NewSnapshotDiffService creates a new snapshot diff service
func NewSnapshotDiffService(queries SnapshotDiffQueries, logger *slog.Logger) *SnapshotDiffService {
	return &SnapshotDiffService{
		queries: queries,
		logger:  logger,
	}
}

// ComputeDiff compares two snapshots and returns the differences
func (s *SnapshotDiffService) ComputeDiff(
	ctx context.Context,
	snapshot1ID uuid.UUID,
	snapshot2ID uuid.UUID,
	includeDiff bool,
	maxDiffSize int,
) (*SnapshotDiffResponse, error) {
	// Fetch file metadata for snapshot1 (using app_id)
	files1, err := s.queries.ListFilesByApp(ctx, pgtype.UUID{Bytes: snapshot1ID, Valid: true})
	if err != nil {
		s.logger.Error("failed to fetch snapshot1 files", "snapshot_id", snapshot1ID, "error", err)
		return nil, fmt.Errorf("failed to fetch snapshot1 files: %w", err)
	}

	// Fetch file metadata for snapshot2 (using app_id)
	files2, err := s.queries.ListFilesByApp(ctx, pgtype.UUID{Bytes: snapshot2ID, Valid: true})
	if err != nil {
		s.logger.Error("failed to fetch snapshot2 files", "snapshot_id", snapshot2ID, "error", err)
		return nil, fmt.Errorf("failed to fetch snapshot2 files: %w", err)
	}

	// Build maps keyed by file_path for fast lookup
	map1 := make(map[string]sqlc.OsaModuleFile)
	for _, f := range files1 {
		map1[f.FilePath] = f
	}

	map2 := make(map[string]sqlc.OsaModuleFile)
	for _, f := range files2 {
		map2[f.FilePath] = f
	}

	// Initialize result structures
	result := DiffResult{
		Added:    []AddedFile{},
		Removed:  []RemovedFile{},
		Modified: []ModifiedFile{},
	}
	summary := DiffSummary{}

	// Categorize files
	// 1. Check snapshot2 files (for Added and Modified)
	for path, file2 := range map2 {
		file1, existsInSnapshot1 := map1[path]

		if !existsInSnapshot1 {
			// File only in snapshot2 → Added
			result.Added = append(result.Added, AddedFile{
				Path:   file2.FilePath,
				Size:   int64(file2.FileSizeBytes),
				SHA256: file2.ContentHash,
			})
		} else if file1.ContentHash != file2.ContentHash {
			// File in both but different SHA256 → Modified
			modified := ModifiedFile{
				Path:      file2.FilePath,
				OldSize:   int64(file1.FileSizeBytes),
				NewSize:   int64(file2.FileSizeBytes),
				OldSHA256: file1.ContentHash,
				NewSHA256: file2.ContentHash,
			}

			// Generate diff if requested
			if includeDiff {
				diff, linesAdded, linesRemoved, err := s.fetchAndDiff(ctx, snapshot1ID, snapshot2ID, path, maxDiffSize)
				if err != nil {
					s.logger.Warn("failed to generate diff", "path", path, "error", err)
					modified.Diff = fmt.Sprintf("Error generating diff: %v", err)
				} else {
					modified.Diff = diff
					summary.LinesAdded += linesAdded
					summary.LinesRemoved += linesRemoved
				}
			}

			result.Modified = append(result.Modified, modified)
		} else {
			// Same SHA256 → Unchanged
			result.Unchanged++
		}
	}

	// 2. Check snapshot1 files for Removed (files not in snapshot2)
	for path, file1 := range map1 {
		if _, existsInSnapshot2 := map2[path]; !existsInSnapshot2 {
			// File only in snapshot1 → Removed
			result.Removed = append(result.Removed, RemovedFile{
				Path:   file1.FilePath,
				Size:   int64(file1.FileSizeBytes),
				SHA256: file1.ContentHash,
			})
		}
	}

	// Calculate summary
	summary.TotalChanges = len(result.Added) + len(result.Removed) + len(result.Modified)

	// Build snapshot info
	snapshot1Info := SnapshotInfo{
		ID:        snapshot1ID,
		CreatedAt: "", // Will be populated from actual snapshot record if needed
		FileCount: len(files1),
	}

	snapshot2Info := SnapshotInfo{
		ID:        snapshot2ID,
		CreatedAt: "", // Will be populated from actual snapshot record if needed
		FileCount: len(files2),
	}

	return &SnapshotDiffResponse{
		Snapshot1: snapshot1Info,
		Snapshot2: snapshot2Info,
		Diff:      result,
		Summary:   summary,
	}, nil
}

// fetchAndDiff fetches file contents and generates a unified diff
func (s *SnapshotDiffService) fetchAndDiff(
	ctx context.Context,
	snapshot1ID uuid.UUID,
	snapshot2ID uuid.UUID,
	filePath string,
	maxDiffSize int,
) (diff string, linesAdded int, linesRemoved int, err error) {
	// Fetch old file record (snapshot1)
	oldFile, err := s.queries.GetFileByPath(ctx, sqlc.GetFileByPathParams{
		WorkflowID: pgtype.UUID{Bytes: snapshot1ID, Valid: true},
		FilePath:   filePath,
	})
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to fetch old file: %w", err)
	}

	// Fetch new file record (snapshot2)
	newFile, err := s.queries.GetFileByPath(ctx, sqlc.GetFileByPathParams{
		WorkflowID: pgtype.UUID{Bytes: snapshot2ID, Valid: true},
		FilePath:   filePath,
	})
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to fetch new file: %w", err)
	}

	// Generate unified diff
	diff, linesAdded, linesRemoved = generateUnifiedDiff(oldFile.Content, newFile.Content, maxDiffSize)
	return diff, linesAdded, linesRemoved, nil
}

// generateUnifiedDiff creates a unified diff by delegating to the shared
// computeUnifiedDiff (workspace_version_service.go) and adding truncation support.
func generateUnifiedDiff(oldContent, newContent string, maxSize int) (diff string, linesAdded int, linesRemoved int) {
	result := computeUnifiedDiff("old", oldContent, newContent)

	diffText := result.Text
	if maxSize > 0 && len(diffText) > maxSize {
		diffText = diffText[:maxSize] + "\n... (diff truncated)"
	}

	return diffText, result.Added, result.Removed
}
