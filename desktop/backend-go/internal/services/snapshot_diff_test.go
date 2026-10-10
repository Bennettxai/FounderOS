package services

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSnapshotQueries provides mock implementation for SnapshotDiffQueries
type MockSnapshotQueries struct {
	mock.Mock
}

func (m *MockSnapshotQueries) ListFilesByApp(ctx context.Context, appID pgtype.UUID) ([]sqlc.OsaModuleFile, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]sqlc.OsaModuleFile), args.Error(1)
}

func (m *MockSnapshotQueries) GetFileByPath(ctx context.Context, params sqlc.GetFileByPathParams) (sqlc.OsaModuleFile, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(sqlc.OsaModuleFile), args.Error(1)
}

// Helper to create OsaModuleFile for testing
func createTestFile(path string, size int32, hash string, content string) sqlc.OsaModuleFile {
	return sqlc.OsaModuleFile{
		FilePath:      path,
		FileSizeBytes: size,
		ContentHash:   hash,
		Content:       content,
	}
}

func TestSnapshotDiffService_ComputeDiff_EmptySnapshots(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return([]sqlc.OsaModuleFile{}, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return([]sqlc.OsaModuleFile{}, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Diff.Added))
	assert.Equal(t, 0, len(result.Diff.Removed))
	assert.Equal(t, 0, len(result.Diff.Modified))
	assert.Equal(t, 0, result.Diff.Unchanged)
	assert.Equal(t, 0, result.Summary.TotalChanges)
	assert.Equal(t, snapshot1ID, result.Snapshot1.ID)
	assert.Equal(t, snapshot2ID, result.Snapshot2.ID)
	assert.Equal(t, 0, result.Snapshot1.FileCount)
	assert.Equal(t, 0, result.Snapshot2.FileCount)

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_IdenticalSnapshots(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files := []sqlc.OsaModuleFile{
		createTestFile("main.go", 100, "abc123", ""),
		createTestFile("utils.go", 200, "def456", ""),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Diff.Added))
	assert.Equal(t, 0, len(result.Diff.Removed))
	assert.Equal(t, 0, len(result.Diff.Modified))
	assert.Equal(t, 2, result.Diff.Unchanged)
	assert.Equal(t, 0, result.Summary.TotalChanges)
	assert.Equal(t, 2, result.Snapshot1.FileCount)
	assert.Equal(t, 2, result.Snapshot2.FileCount)

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_AllAdded(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files2 := []sqlc.OsaModuleFile{
		createTestFile("new1.go", 100, "abc123", ""),
		createTestFile("new2.go", 200, "def456", ""),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return([]sqlc.OsaModuleFile{}, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files2, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Diff.Added))
	assert.Equal(t, 0, len(result.Diff.Removed))
	assert.Equal(t, 0, len(result.Diff.Modified))
	assert.Equal(t, 0, result.Diff.Unchanged)
	assert.Equal(t, 2, result.Summary.TotalChanges)

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_AllRemoved(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files1 := []sqlc.OsaModuleFile{
		createTestFile("old1.go", 100, "abc123", ""),
		createTestFile("old2.go", 200, "def456", ""),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files1, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return([]sqlc.OsaModuleFile{}, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Diff.Added))
	assert.Equal(t, 2, len(result.Diff.Removed))
	assert.Equal(t, 0, len(result.Diff.Modified))
	assert.Equal(t, 0, result.Diff.Unchanged)
	assert.Equal(t, 2, result.Summary.TotalChanges)

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_MixedChanges(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files1 := []sqlc.OsaModuleFile{
		createTestFile("unchanged.go", 100, "same123", ""),
		createTestFile("modified.go", 200, "old456", ""),
		createTestFile("removed.go", 150, "gone789", ""),
	}

	files2 := []sqlc.OsaModuleFile{
		createTestFile("unchanged.go", 100, "same123", ""),
		createTestFile("modified.go", 250, "new456", ""),
		createTestFile("added.go", 300, "new999", ""),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files1, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files2, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Diff.Added), "Should have 1 added file")
	assert.Equal(t, 1, len(result.Diff.Removed), "Should have 1 removed file")
	assert.Equal(t, 1, len(result.Diff.Modified), "Should have 1 modified file")
	assert.Equal(t, 1, result.Diff.Unchanged, "Should have 1 unchanged file")
	assert.Equal(t, 3, result.Summary.TotalChanges, "Total changes should be 3")

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_WithUnifiedDiff(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files1 := []sqlc.OsaModuleFile{
		createTestFile("test.go", 30, "old123", "line1\nline2\nline3"),
	}
	files2 := []sqlc.OsaModuleFile{
		createTestFile("test.go", 35, "new456", "line1\nline2 modified\nline3"),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files1, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files2, nil)

	// GetFileByPath returns the full OsaModuleFile for diff generation
	mockQueries.On("GetFileByPath", mock.Anything, sqlc.GetFileByPathParams{
		WorkflowID: pgtype.UUID{Bytes: snapshot1ID, Valid: true},
		FilePath:   "test.go",
	}).Return(sqlc.OsaModuleFile{Content: "line1\nline2\nline3"}, nil)
	mockQueries.On("GetFileByPath", mock.Anything, sqlc.GetFileByPathParams{
		WorkflowID: pgtype.UUID{Bytes: snapshot2ID, Valid: true},
		FilePath:   "test.go",
	}).Return(sqlc.OsaModuleFile{Content: "line1\nline2 modified\nline3"}, nil)

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, true, 0)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Diff.Modified))
	assert.NotEmpty(t, result.Diff.Modified[0].Diff, "Diff should be generated when includeDiff=true")
	assert.Contains(t, result.Diff.Modified[0].Diff, "--- a/old")
	assert.Contains(t, result.Diff.Modified[0].Diff, "+++ b/old")
	assert.Equal(t, 1, result.Summary.LinesAdded)
	assert.Equal(t, 1, result.Summary.LinesRemoved)

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_DiffTruncation(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files1 := []sqlc.OsaModuleFile{
		createTestFile("large.go", 1000, "old123", "old line content"),
	}
	files2 := []sqlc.OsaModuleFile{
		createTestFile("large.go", 1100, "new456", "new line content"),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files1, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files2, nil)
	mockQueries.On("GetFileByPath", mock.Anything, mock.Anything).
		Return(sqlc.OsaModuleFile{Content: "old line content"}, nil).Once()
	mockQueries.On("GetFileByPath", mock.Anything, mock.Anything).
		Return(sqlc.OsaModuleFile{Content: "new line content"}, nil).Once()

	maxDiffSize := 50
	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, true, maxDiffSize)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Diff.Modified))
	assert.Contains(t, result.Diff.Modified[0].Diff, "... (diff truncated)", "Diff should contain truncation message")

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_ErrorFetchingSnapshot1(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(nil, errors.New("database error"))

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch snapshot1 files")

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_ErrorFetchingSnapshot2(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return([]sqlc.OsaModuleFile{}, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(nil, errors.New("database error"))

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, false, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch snapshot2 files")

	mockQueries.AssertExpectations(t)
}

func TestSnapshotDiffService_ComputeDiff_ErrorFetchingFileContent(t *testing.T) {
	mockQueries := new(MockSnapshotQueries)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewSnapshotDiffService(mockQueries, logger)

	snapshot1ID := uuid.New()
	snapshot2ID := uuid.New()

	files1 := []sqlc.OsaModuleFile{
		createTestFile("test.go", 100, "old123", ""),
	}
	files2 := []sqlc.OsaModuleFile{
		createTestFile("test.go", 150, "new456", ""),
	}

	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot1ID, Valid: true}).
		Return(files1, nil)
	mockQueries.On("ListFilesByApp", mock.Anything, pgtype.UUID{Bytes: snapshot2ID, Valid: true}).
		Return(files2, nil)
	mockQueries.On("GetFileByPath", mock.Anything, mock.Anything).
		Return(sqlc.OsaModuleFile{}, errors.New("file not found"))

	result, err := service.ComputeDiff(context.Background(), snapshot1ID, snapshot2ID, true, 0)

	require.NoError(t, err, "Should not error on diff generation failure")
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Diff.Modified))
	assert.Contains(t, result.Diff.Modified[0].Diff, "Error generating diff")

	mockQueries.AssertExpectations(t)
}

func TestGenerateUnifiedDiff_SimpleDiff(t *testing.T) {
	oldContent := "line1\nline2\nline3"
	newContent := "line1\nline2 changed\nline3"

	diff, linesAdded, linesRemoved := generateUnifiedDiff(oldContent, newContent, 0)

	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "--- a/old")
	assert.Contains(t, diff, "+++ b/old")
	assert.Contains(t, diff, "-line2")
	assert.Contains(t, diff, "+line2 changed")
	assert.Equal(t, 1, linesAdded)
	assert.Equal(t, 1, linesRemoved)
}

func TestGenerateUnifiedDiff_NoChanges(t *testing.T) {
	content := "line1\nline2\nline3"

	diff, linesAdded, linesRemoved := generateUnifiedDiff(content, content, 0)

	assert.Contains(t, diff, "--- a/old")
	assert.Contains(t, diff, "+++ b/old")
	assert.Equal(t, 0, linesAdded)
	assert.Equal(t, 0, linesRemoved)
}

func TestGenerateUnifiedDiff_OnlyAdditions(t *testing.T) {
	oldContent := "line1\nline2"
	newContent := "line1\nline2\nline3\nline4"

	diff, linesAdded, linesRemoved := generateUnifiedDiff(oldContent, newContent, 0)

	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "+line3")
	assert.Contains(t, diff, "+line4")
	assert.Equal(t, 2, linesAdded)
	assert.Equal(t, 0, linesRemoved)
}

func TestGenerateUnifiedDiff_OnlyRemovals(t *testing.T) {
	oldContent := "line1\nline2\nline3\nline4"
	newContent := "line1\nline2"

	diff, linesAdded, linesRemoved := generateUnifiedDiff(oldContent, newContent, 0)

	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "-line3")
	assert.Contains(t, diff, "-line4")
	assert.Equal(t, 0, linesAdded)
	assert.Equal(t, 2, linesRemoved)
}

func TestGenerateUnifiedDiff_EmptyOldContent(t *testing.T) {
	oldContent := ""
	newContent := "line1\nline2"

	diff, linesAdded, linesRemoved := generateUnifiedDiff(oldContent, newContent, 0)

	assert.NotEmpty(t, diff)
	// strings.Split("", "\n") returns [""] (1 element), so empty string counts as a line removed
	assert.Equal(t, 2, linesAdded)
	assert.Equal(t, 1, linesRemoved)
}

func TestGenerateUnifiedDiff_EmptyNewContent(t *testing.T) {
	oldContent := "line1\nline2"
	newContent := ""

	diff, linesAdded, linesRemoved := generateUnifiedDiff(oldContent, newContent, 0)

	assert.NotEmpty(t, diff)
	// strings.Split("", "\n") returns [""] (1 element), so empty string counts as a line added
	assert.Equal(t, 1, linesAdded)
	assert.Equal(t, 2, linesRemoved)
}
