package services

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ============================================================
// PEDRO-6: Snapshot Diff Tests
// ============================================================

// --- 1. computeUnifiedDiff tests ---

func TestDiffIdenticalContent(t *testing.T) {
	content := "line1\nline2\nline3"
	result := computeUnifiedDiff("test.go", content, content)

	if result.Added != 0 {
		t.Errorf("Expected 0 lines added for identical content, got %d", result.Added)
	}
	if result.Removed != 0 {
		t.Errorf("Expected 0 lines removed for identical content, got %d", result.Removed)
	}
	// All lines should be context (prefixed with space)
	for _, line := range strings.Split(result.Text, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue // header lines
		}
		if !strings.HasPrefix(line, " ") {
			t.Errorf("Expected context line (space prefix) for identical content, got: %q", line)
		}
	}
}

func TestDiffAddedLines(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nline2\nline3\nline4"
	result := computeUnifiedDiff("test.go", old, new)

	if result.Added != 2 {
		t.Errorf("Expected 2 lines added, got %d", result.Added)
	}
	if result.Removed != 0 {
		t.Errorf("Expected 0 lines removed, got %d", result.Removed)
	}
	if !strings.Contains(result.Text, "+line3") {
		t.Error("Diff should contain +line3")
	}
	if !strings.Contains(result.Text, "+line4") {
		t.Error("Diff should contain +line4")
	}
}

func TestDiffRemovedLines(t *testing.T) {
	old := "line1\nline2\nline3\nline4"
	new := "line1\nline2"
	result := computeUnifiedDiff("test.go", old, new)

	if result.Removed != 2 {
		t.Errorf("Expected 2 lines removed, got %d", result.Removed)
	}
	if result.Added != 0 {
		t.Errorf("Expected 0 lines added, got %d", result.Added)
	}
	if !strings.Contains(result.Text, "-line3") {
		t.Error("Diff should contain -line3")
	}
}

func TestDiffModifiedLines(t *testing.T) {
	old := "line1\nold_line\nline3"
	new := "line1\nnew_line\nline3"
	result := computeUnifiedDiff("test.go", old, new)

	if result.Added != 1 {
		t.Errorf("Expected 1 line added, got %d", result.Added)
	}
	if result.Removed != 1 {
		t.Errorf("Expected 1 line removed, got %d", result.Removed)
	}
	if !strings.Contains(result.Text, "-old_line") {
		t.Error("Diff should contain -old_line")
	}
	if !strings.Contains(result.Text, "+new_line") {
		t.Error("Diff should contain +new_line")
	}
}

func TestDiffEmptyOldContent(t *testing.T) {
	result := computeUnifiedDiff("new_file.go", "", "package main\n\nfunc main() {}")
	if result.Added == 0 {
		t.Error("Expected added lines for new file")
	}
	if result.Removed != 0 {
		t.Errorf("Expected 0 removed lines, got %d", result.Removed)
	}
}

func TestDiffEmptyNewContent(t *testing.T) {
	result := computeUnifiedDiff("deleted.go", "package main\n\nfunc main() {}", "")
	if result.Removed == 0 {
		t.Error("Expected removed lines for deleted file")
	}
	if result.Added != 0 {
		t.Errorf("Expected 0 added lines, got %d", result.Added)
	}
}

func TestDiffHeaderFormat(t *testing.T) {
	result := computeUnifiedDiff("src/main.go", "a", "b")
	if !strings.HasPrefix(result.Text, "--- a/src/main.go\n") {
		t.Errorf("Expected diff header with old file path, got: %s", result.Text[:50])
	}
	if !strings.Contains(result.Text, "+++ b/src/main.go\n") {
		t.Error("Expected diff header with new file path")
	}
}

func TestDiffBothEmpty(t *testing.T) {
	result := computeUnifiedDiff("empty.txt", "", "")
	if result.Added != 0 || result.Removed != 0 {
		t.Errorf("Expected no changes for empty→empty, got added=%d removed=%d", result.Added, result.Removed)
	}
}

// --- 2. computeLCS tests ---

func TestLCSIdentical(t *testing.T) {
	lines := []string{"a", "b", "c"}
	lcs := computeLCS(lines, lines)
	if len(lcs) != 3 {
		t.Errorf("Expected LCS length 3 for identical slices, got %d", len(lcs))
	}
}

func TestLCSNoCommon(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"x", "y", "z"}
	lcs := computeLCS(a, b)
	if len(lcs) != 0 {
		t.Errorf("Expected LCS length 0 for disjoint slices, got %d", len(lcs))
	}
}

func TestLCSPartialOverlap(t *testing.T) {
	a := []string{"a", "b", "c", "d"}
	b := []string{"a", "x", "c", "d"}
	lcs := computeLCS(a, b)
	if len(lcs) != 3 {
		t.Errorf("Expected LCS length 3, got %d: %v", len(lcs), lcs)
	}
	expected := []string{"a", "c", "d"}
	for i, v := range expected {
		if i < len(lcs) && lcs[i] != v {
			t.Errorf("LCS[%d] = %q, want %q", i, lcs[i], v)
		}
	}
}

func TestLCSEmptyInput(t *testing.T) {
	lcs := computeLCS(nil, []string{"a"})
	if len(lcs) != 0 {
		t.Errorf("Expected empty LCS for nil input, got %d", len(lcs))
	}

	lcs = computeLCS([]string{"a"}, nil)
	if len(lcs) != 0 {
		t.Errorf("Expected empty LCS for nil second input, got %d", len(lcs))
	}
}

func TestLCSLargeInputFallback(t *testing.T) {
	// Create input that exceeds the 1M cell limit
	a := make([]string, 1001)
	b := make([]string, 1001)
	for i := range a {
		a[i] = "line"
		b[i] = "line"
	}
	// 1001 * 1001 = 1_002_001 > 1_000_000, should fallback to nil
	lcs := computeLCS(a, b)
	if lcs != nil {
		t.Error("Expected nil LCS for large input (fallback), got non-nil")
	}
}

// --- 3. indexFilesByPath tests ---

func TestIndexFilesByPath(t *testing.T) {
	files := []generatedFileInfo{
		{FilePath: "src/main.go", Content: "package main", ContentHash: "abc"},
		{FilePath: "src/handler.go", Content: "package handler", ContentHash: "def"},
	}

	index := indexFilesByPath(files)

	if len(index) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(index))
	}
	if f, ok := index["src/main.go"]; !ok || f.ContentHash != "abc" {
		t.Error("Expected src/main.go with hash abc")
	}
	if f, ok := index["src/handler.go"]; !ok || f.ContentHash != "def" {
		t.Error("Expected src/handler.go with hash def")
	}
}

func TestIndexFilesByPathEmpty(t *testing.T) {
	index := indexFilesByPath(nil)
	if len(index) != 0 {
		t.Errorf("Expected empty index for nil input, got %d", len(index))
	}
}

func TestIndexFilesByPathDuplicatePath(t *testing.T) {
	files := []generatedFileInfo{
		{FilePath: "same.go", ContentHash: "first"},
		{FilePath: "same.go", ContentHash: "second"},
	}
	index := indexFilesByPath(files)
	// Last one wins
	if index["same.go"].ContentHash != "second" {
		t.Error("Expected last file to win for duplicate path")
	}
}

// --- 4. extractModuleInstanceIDs tests ---

func TestExtractModuleInstanceIDs(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	snapshot := &WorkspaceSnapshot{
		Apps: []AppSnapshot{
			{AppName: "app1", ModuleInstanceID: &id1},
			{AppName: "app2", ModuleInstanceID: nil},
			{AppName: "app3", ModuleInstanceID: &id2},
		},
	}

	ids := extractModuleInstanceIDs(snapshot)

	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs (skipping nil), got %d", len(ids))
	}
	if ids[0] != id1 {
		t.Errorf("Expected first ID %s, got %s", id1, ids[0])
	}
	if ids[1] != id2 {
		t.Errorf("Expected second ID %s, got %s", id2, ids[1])
	}
}

func TestExtractModuleInstanceIDsEmpty(t *testing.T) {
	snapshot := &WorkspaceSnapshot{Apps: nil}
	ids := extractModuleInstanceIDs(snapshot)
	if len(ids) != 0 {
		t.Errorf("Expected 0 IDs for empty snapshot, got %d", len(ids))
	}
}

// --- 5. countNewApps tests ---

func TestCountNewApps(t *testing.T) {
	from := &WorkspaceSnapshot{
		Apps: []AppSnapshot{
			{AppName: "existing-app"},
		},
	}
	to := &WorkspaceSnapshot{
		Apps: []AppSnapshot{
			{AppName: "existing-app"},
			{AppName: "new-app"},
		},
	}

	count := countNewApps(from, to)
	if count != 1 {
		t.Errorf("Expected 1 new app, got %d", count)
	}
}

func TestCountNewAppsNone(t *testing.T) {
	s := &WorkspaceSnapshot{
		Apps: []AppSnapshot{{AppName: "app1"}},
	}
	count := countNewApps(s, s)
	if count != 0 {
		t.Errorf("Expected 0 new apps for identical snapshots, got %d", count)
	}
}

func TestCountNewAppsAllNew(t *testing.T) {
	from := &WorkspaceSnapshot{Apps: nil}
	to := &WorkspaceSnapshot{
		Apps: []AppSnapshot{
			{AppName: "a"},
			{AppName: "b"},
		},
	}
	count := countNewApps(from, to)
	if count != 2 {
		t.Errorf("Expected 2 new apps, got %d", count)
	}
}

// --- 6. VersionDiffResult structure tests ---

func TestVersionDiffResultStructure(t *testing.T) {
	result := &VersionDiffResult{
		FromVersion: "0.0.1",
		ToVersion:   "0.0.2",
		Summary: VersionDiffSummary{
			FilesAdded:        3,
			FilesRemoved:      1,
			FilesModified:     2,
			FilesUnchanged:    5,
			TotalLinesAdded:   100,
			TotalLinesRemoved: 20,
			AppsAdded:         1,
			AppsRemoved:       0,
		},
		Files: []FileDiff{
			{
				FilePath:   "src/new.go",
				ChangeType: "added",
				Language:   "go",
				LinesAdded: 50,
			},
			{
				FilePath:     "src/old.go",
				ChangeType:   "removed",
				Language:     "go",
				LinesRemoved: 20,
			},
			{
				FilePath:     "src/main.go",
				ChangeType:   "modified",
				Language:     "go",
				LinesAdded:   10,
				LinesRemoved: 5,
			},
		},
	}

	if result.FromVersion != "0.0.1" {
		t.Errorf("Expected FromVersion 0.0.1, got %s", result.FromVersion)
	}
	if result.ToVersion != "0.0.2" {
		t.Errorf("Expected ToVersion 0.0.2, got %s", result.ToVersion)
	}
	if result.Summary.FilesAdded != 3 {
		t.Errorf("Expected 3 files added, got %d", result.Summary.FilesAdded)
	}
	if len(result.Files) != 3 {
		t.Errorf("Expected 3 file diffs, got %d", len(result.Files))
	}

	// Verify change types
	changeTypes := map[string]int{}
	for _, f := range result.Files {
		changeTypes[f.ChangeType]++
	}
	if changeTypes["added"] != 1 {
		t.Errorf("Expected 1 added file, got %d", changeTypes["added"])
	}
	if changeTypes["removed"] != 1 {
		t.Errorf("Expected 1 removed file, got %d", changeTypes["removed"])
	}
	if changeTypes["modified"] != 1 {
		t.Errorf("Expected 1 modified file, got %d", changeTypes["modified"])
	}
}

// --- 7. FileDiff change type validation ---

func TestFileDiffChangeTypes(t *testing.T) {
	validTypes := []string{"added", "removed", "modified", "unchanged"}

	for _, ct := range validTypes {
		fd := FileDiff{
			FilePath:   "test.go",
			ChangeType: ct,
		}
		if fd.ChangeType != ct {
			t.Errorf("Expected change type %q, got %q", ct, fd.ChangeType)
		}
	}
}

// --- 8. End-to-end diff simulation ---

func TestDiffSimulationAddedFile(t *testing.T) {
	fromFiles := []generatedFileInfo{}
	toFiles := []generatedFileInfo{
		{FilePath: "new_file.go", Content: "package new\nfunc Hello() {}", ContentHash: "abc", Language: "go", FileType: "source"},
	}

	fromMap := indexFilesByPath(fromFiles)
	toMap := indexFilesByPath(toFiles)

	var diffs []FileDiff
	for path, toFile := range toMap {
		if _, exists := fromMap[path]; !exists {
			diffs = append(diffs, FileDiff{
				FilePath:   path,
				ChangeType: "added",
				Language:   toFile.Language,
				NewContent: toFile.Content,
				LinesAdded: strings.Count(toFile.Content, "\n") + 1,
			})
		}
	}

	if len(diffs) != 1 {
		t.Fatalf("Expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].ChangeType != "added" {
		t.Errorf("Expected change type 'added', got %q", diffs[0].ChangeType)
	}
	if diffs[0].LinesAdded != 2 {
		t.Errorf("Expected 2 lines added, got %d", diffs[0].LinesAdded)
	}
}

func TestDiffSimulationRemovedFile(t *testing.T) {
	fromFiles := []generatedFileInfo{
		{FilePath: "old_file.go", Content: "package old", ContentHash: "xyz", Language: "go"},
	}
	toFiles := []generatedFileInfo{}

	fromMap := indexFilesByPath(fromFiles)
	toMap := indexFilesByPath(toFiles)

	var diffs []FileDiff
	for path, fromFile := range fromMap {
		if _, exists := toMap[path]; !exists {
			diffs = append(diffs, FileDiff{
				FilePath:     path,
				ChangeType:   "removed",
				OldContent:   fromFile.Content,
				LinesRemoved: strings.Count(fromFile.Content, "\n") + 1,
			})
		}
	}

	if len(diffs) != 1 {
		t.Fatalf("Expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].ChangeType != "removed" {
		t.Errorf("Expected 'removed', got %q", diffs[0].ChangeType)
	}
}

func TestDiffSimulationModifiedFile(t *testing.T) {
	fromFiles := []generatedFileInfo{
		{FilePath: "main.go", Content: "package main\nfunc old() {}", ContentHash: "aaa"},
	}
	toFiles := []generatedFileInfo{
		{FilePath: "main.go", Content: "package main\nfunc new() {}", ContentHash: "bbb"},
	}

	fromMap := indexFilesByPath(fromFiles)
	toMap := indexFilesByPath(toFiles)

	for path, toFile := range toMap {
		fromFile := fromMap[path]
		if fromFile.ContentHash != toFile.ContentHash {
			diff := computeUnifiedDiff(path, fromFile.Content, toFile.Content)
			if diff.Added == 0 && diff.Removed == 0 {
				t.Error("Expected changes in modified file diff")
			}
			if !strings.Contains(diff.Text, "-func old()") {
				t.Error("Expected diff to show removed old function")
			}
			if !strings.Contains(diff.Text, "+func new()") {
				t.Error("Expected diff to show added new function")
			}
		}
	}
}

func TestDiffSimulationUnchangedFile(t *testing.T) {
	hash := "same_hash"
	fromFiles := []generatedFileInfo{
		{FilePath: "unchanged.go", Content: "package same", ContentHash: hash},
	}
	toFiles := []generatedFileInfo{
		{FilePath: "unchanged.go", Content: "package same", ContentHash: hash},
	}

	fromMap := indexFilesByPath(fromFiles)
	toMap := indexFilesByPath(toFiles)

	for path, toFile := range toMap {
		fromFile := fromMap[path]
		if fromFile.ContentHash == toFile.ContentHash {
			// File is unchanged - correct
			return
		}
	}
	t.Error("Should have detected unchanged file")
}

// --- 9. Summary computation ---

func TestDiffSummaryComputation(t *testing.T) {
	diffs := []FileDiff{
		{ChangeType: "added", LinesAdded: 10},
		{ChangeType: "added", LinesAdded: 20},
		{ChangeType: "removed", LinesRemoved: 5},
		{ChangeType: "modified", LinesAdded: 8, LinesRemoved: 3},
		{ChangeType: "unchanged"},
	}

	summary := VersionDiffSummary{}
	for _, fd := range diffs {
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

	if summary.FilesAdded != 2 {
		t.Errorf("Expected 2 added, got %d", summary.FilesAdded)
	}
	if summary.FilesRemoved != 1 {
		t.Errorf("Expected 1 removed, got %d", summary.FilesRemoved)
	}
	if summary.FilesModified != 1 {
		t.Errorf("Expected 1 modified, got %d", summary.FilesModified)
	}
	if summary.FilesUnchanged != 1 {
		t.Errorf("Expected 1 unchanged, got %d", summary.FilesUnchanged)
	}
	if summary.TotalLinesAdded != 38 {
		t.Errorf("Expected 38 total lines added, got %d", summary.TotalLinesAdded)
	}
	if summary.TotalLinesRemoved != 8 {
		t.Errorf("Expected 8 total lines removed, got %d", summary.TotalLinesRemoved)
	}
}

// --- 10. Diff with complex content ---

func TestDiffMultiLineGoFile(t *testing.T) {
	old := `package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func GetUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}`

	new := `package handlers

import (
	"log/slog"
	"net/http"
	"github.com/gin-gonic/gin"
)

func GetUser(c *gin.Context) {
	id := c.Param("id")
	slog.Info("fetching user", "id", id)
	user, err := fetchUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}`

	result := computeUnifiedDiff("internal/handlers/user.go", old, new)

	if result.Added == 0 {
		t.Error("Expected added lines in modified Go file")
	}
	if result.Removed == 0 {
		t.Error("Expected removed lines in modified Go file")
	}

	// Should contain the slog import addition
	if !strings.Contains(result.Text, "+\t\"log/slog\"") {
		t.Error("Diff should show added slog import")
	}
	// Should contain the new slog.Info line
	if !strings.Contains(result.Text, "+\tslog.Info") {
		t.Error("Diff should show added slog.Info line")
	}

	// Header should reference the file path
	if !strings.Contains(result.Text, "--- a/internal/handlers/user.go") {
		t.Error("Diff header should reference old file path")
	}
}

// --- Benchmarks ---

func BenchmarkComputeUnifiedDiff(b *testing.B) {
	old := strings.Repeat("line content here\n", 100)
	new := old + "added line\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeUnifiedDiff("bench.go", old, new)
	}
}

func BenchmarkComputeLCS(b *testing.B) {
	a := make([]string, 200)
	bSlice := make([]string, 200)
	for i := range a {
		a[i] = "common line"
		bSlice[i] = "common line"
	}
	// Modify some lines
	bSlice[50] = "different"
	bSlice[100] = "different"
	bSlice[150] = "different"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeLCS(a, bSlice)
	}
}
