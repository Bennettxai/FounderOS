package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// CompareVersions compares two workspace snapshots and returns file-level diffs
func (s *WorkspaceVersionService) CompareVersions(
	ctx context.Context,
	workspaceID uuid.UUID,
	fromVersion string,
	toVersion string,
	filterFile string,
) (*VersionDiffResult, error) {
	// Fetch both snapshots
	fromSnapshot, err := s.getSnapshotData(ctx, workspaceID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("fetch version %s: %w", fromVersion, err)
	}

	toSnapshot, err := s.getSnapshotData(ctx, workspaceID, toVersion)
	if err != nil {
		return nil, fmt.Errorf("fetch version %s: %w", toVersion, err)
	}

	// Build maps of module instance IDs from each snapshot
	fromAppIDs := extractModuleInstanceIDs(fromSnapshot)
	toAppIDs := extractModuleInstanceIDs(toSnapshot)

	// Fetch generated files for each set of app IDs
	fromFiles, err := s.fetchGeneratedFiles(ctx, fromAppIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch files for version %s: %w", fromVersion, err)
	}

	toFiles, err := s.fetchGeneratedFiles(ctx, toAppIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch files for version %s: %w", toVersion, err)
	}

	// Index files by path
	fromFileMap := indexFilesByPath(fromFiles)
	toFileMap := indexFilesByPath(toFiles)

	// Compute diffs
	var fileDiffs []FileDiff

	// Files in toVersion (added or modified)
	for path, toFile := range toFileMap {
		if filterFile != "" && path != filterFile {
			continue
		}
		if fromFile, exists := fromFileMap[path]; exists {
			// File exists in both versions
			if fromFile.ContentHash == toFile.ContentHash {
				fileDiffs = append(fileDiffs, FileDiff{
					FilePath:   path,
					ChangeType: "unchanged",
					Language:   toFile.Language,
					FileType:   toFile.FileType,
				})
			} else {
				diff := computeUnifiedDiff(path, fromFile.Content, toFile.Content)
				fileDiffs = append(fileDiffs, FileDiff{
					FilePath:     path,
					ChangeType:   "modified",
					Language:     toFile.Language,
					FileType:     toFile.FileType,
					OldContent:   fromFile.Content,
					NewContent:   toFile.Content,
					UnifiedDiff:  diff.Text,
					LinesAdded:   diff.Added,
					LinesRemoved: diff.Removed,
				})
			}
		} else {
			// File only in toVersion → added
			lineCount := strings.Count(toFile.Content, "\n") + 1
			fileDiffs = append(fileDiffs, FileDiff{
				FilePath:   path,
				ChangeType: "added",
				Language:   toFile.Language,
				FileType:   toFile.FileType,
				NewContent: toFile.Content,
				LinesAdded: lineCount,
			})
		}
	}

	// Files only in fromVersion → removed
	for path, fromFile := range fromFileMap {
		if filterFile != "" && path != filterFile {
			continue
		}
		if _, exists := toFileMap[path]; !exists {
			lineCount := strings.Count(fromFile.Content, "\n") + 1
			fileDiffs = append(fileDiffs, FileDiff{
				FilePath:     path,
				ChangeType:   "removed",
				Language:     fromFile.Language,
				FileType:     fromFile.FileType,
				OldContent:   fromFile.Content,
				LinesRemoved: lineCount,
			})
		}
	}

	// Build summary
	summary := VersionDiffSummary{
		AppsAdded:   countNewApps(fromSnapshot, toSnapshot),
		AppsRemoved: countNewApps(toSnapshot, fromSnapshot),
	}
	for _, fd := range fileDiffs {
		switch fd.ChangeType {
		case "added":
			summary.FilesAdded++
		case "removed":
			summary.FilesRemoved++
		case "modified":
			summary.FilesModified++
		case "unchanged":
			summary.FilesUnchanged++
		}
		summary.TotalLinesAdded += fd.LinesAdded
		summary.TotalLinesRemoved += fd.LinesRemoved
	}

	return &VersionDiffResult{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Summary:     summary,
		Files:       fileDiffs,
	}, nil
}

// getSnapshotData fetches and parses a specific workspace version
func (s *WorkspaceVersionService) getSnapshotData(
	ctx context.Context,
	workspaceID uuid.UUID,
	versionNumber string,
) (*WorkspaceSnapshot, error) {
	var snapshotData json.RawMessage
	err := s.pool.QueryRow(ctx, `
		SELECT snapshot_data
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, versionNumber).Scan(&snapshotData)

	if err != nil {
		return nil, err
	}

	var snapshot WorkspaceSnapshot
	if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
		return nil, fmt.Errorf("invalid snapshot data: %w", err)
	}

	return &snapshot, nil
}

// fetchGeneratedFiles fetches all generated files for a set of app IDs
func (s *WorkspaceVersionService) fetchGeneratedFiles(
	ctx context.Context,
	appIDs []uuid.UUID,
) ([]generatedFileInfo, error) {
	if len(appIDs) == 0 {
		return nil, nil
	}

	// Build query with app IDs
	placeholders := make([]string, len(appIDs))
	args := make([]interface{}, len(appIDs))
	for i, id := range appIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT file_path, content, content_hash, COALESCE(language, ''), COALESCE(file_type, '')
		FROM osa_module_files
		WHERE module_instance_id IN (%s) AND is_latest = true
		ORDER BY file_path
	`, strings.Join(placeholders, ","))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query generated files: %w", err)
	}
	defer rows.Close()

	var files []generatedFileInfo
	for rows.Next() {
		var f generatedFileInfo
		if err := rows.Scan(&f.FilePath, &f.Content, &f.ContentHash, &f.Language, &f.FileType); err != nil {
			s.logger.Error("failed to scan generated file", "error", err)
			continue
		}
		files = append(files, f)
	}

	return files, nil
}

// extractModuleInstanceIDs extracts module instance IDs from a workspace snapshot.
func extractModuleInstanceIDs(snapshot *WorkspaceSnapshot) []uuid.UUID {
	var ids []uuid.UUID
	for _, app := range snapshot.Apps {
		if app.ModuleInstanceID != nil {
			ids = append(ids, *app.ModuleInstanceID)
		}
	}
	return ids
}

// indexFilesByPath creates a map of file path → file info
func indexFilesByPath(files []generatedFileInfo) map[string]generatedFileInfo {
	m := make(map[string]generatedFileInfo, len(files))
	for _, f := range files {
		m[f.FilePath] = f
	}
	return m
}

// countNewApps counts apps in 'to' that are not in 'from' (by app name)
func countNewApps(from, to *WorkspaceSnapshot) int {
	fromNames := make(map[string]bool, len(from.Apps))
	for _, app := range from.Apps {
		fromNames[app.AppName] = true
	}
	count := 0
	for _, app := range to.Apps {
		if !fromNames[app.AppName] {
			count++
		}
	}
	return count
}

// computeUnifiedDiff computes a unified diff between two strings
func computeUnifiedDiff(filePath, oldContent, newContent string) diffResult {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple line-by-line diff using LCS
	lcs := computeLCS(oldLines, newLines)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	added, removed := 0, 0
	oldIdx, newIdx, lcsIdx := 0, 0, 0

	for lcsIdx < len(lcs) {
		// Output removed lines (in old but not matching LCS)
		for oldIdx < len(oldLines) && oldLines[oldIdx] != lcs[lcsIdx] {
			sb.WriteString(fmt.Sprintf("-%s\n", oldLines[oldIdx]))
			removed++
			oldIdx++
		}
		// Output added lines (in new but not matching LCS)
		for newIdx < len(newLines) && newLines[newIdx] != lcs[lcsIdx] {
			sb.WriteString(fmt.Sprintf("+%s\n", newLines[newIdx]))
			added++
			newIdx++
		}
		// Output context line
		sb.WriteString(fmt.Sprintf(" %s\n", lcs[lcsIdx]))
		oldIdx++
		newIdx++
		lcsIdx++
	}

	// Remaining old lines (removed)
	for oldIdx < len(oldLines) {
		sb.WriteString(fmt.Sprintf("-%s\n", oldLines[oldIdx]))
		removed++
		oldIdx++
	}
	// Remaining new lines (added)
	for newIdx < len(newLines) {
		sb.WriteString(fmt.Sprintf("+%s\n", newLines[newIdx]))
		added++
		newIdx++
	}

	return diffResult{Text: sb.String(), Added: added, Removed: removed}
}

// computeLCS computes the Longest Common Subsequence of two string slices
func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)

	// For large files, limit LCS computation to avoid O(m*n) memory/CPU exhaustion
	// 100K cells ≈ ~800KB memory, ~10-50ms compute — safe for concurrent requests
	if m*n > 100_000 {
		// Fallback: treat entire content as changed
		return nil
	}

	// Standard DP LCS
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find LCS
	result := make([]string, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			result = append(result, a[i-1])
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Reverse
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	return result
}

// incrementVersion increments semantic version
func incrementVersion(current *string) string {
	if current == nil || *current == "" {
		return "0.0.1"
	}

	// Parse semantic version: "0.0.1" -> [0, 0, 1]
	parts := strings.Split(*current, ".")
	if len(parts) != 3 {
		return "0.0.1"
	}

	// Convert patch version
	var patch int
	_, err := fmt.Sscanf(parts[2], "%d", &patch)
	if err != nil {
		return "0.0.1"
	}

	patch++
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch)
}
