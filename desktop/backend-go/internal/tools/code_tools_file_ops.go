package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ========================================
// READ FILE TOOL
// ========================================

type ReadFileInput struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type ReadFileOutput struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	TotalLines int    `json:"total_lines"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
}

func (r *CodeToolRegistry) ReadFile(ctx context.Context, input json.RawMessage) (string, error) {
	var params ReadFileInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	absPath, err := r.resolveAndValidatePath(params.Path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", params.Path)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// Handle line range
	startLine := 1
	endLine := totalLines

	if params.StartLine > 0 {
		startLine = params.StartLine
	}
	if params.EndLine > 0 {
		endLine = params.EndLine
	}

	// Clamp values
	if startLine < 1 {
		startLine = 1
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		startLine = endLine
	}

	// Build output with line numbers
	var sb strings.Builder
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		sb.WriteString(fmt.Sprintf("%4d | %s\n", i+1, lines[i]))
	}

	output := ReadFileOutput{
		Path:       params.Path,
		Content:    sb.String(),
		TotalLines: totalLines,
		StartLine:  startLine,
		EndLine:    endLine,
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}
	return string(result), nil
}

// ========================================
// WRITE FILE TOOL
// ========================================

type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteFileOutput struct {
	Path         string `json:"path"`
	BackupPath   string `json:"backup_path,omitempty"`
	BytesWritten int    `json:"bytes_written"`
	LinesWritten int    `json:"lines_written"`
}

func (r *CodeToolRegistry) WriteFile(ctx context.Context, input json.RawMessage) (string, error) {
	var params WriteFileInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	absPath, err := r.resolveAndValidatePath(params.Path)
	if err != nil {
		return "", err
	}

	// Create backup if file exists
	var backupPath string
	if _, err := os.Stat(absPath); err == nil {
		backup, backupErr := r.backupMgr.CreateBackup(params.Path)
		if backupErr == nil {
			backupPath = backup.BackupPath
		}
	}

	// Create directory if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	lines := strings.Count(params.Content, "\n") + 1

	output := WriteFileOutput{
		Path:         params.Path,
		BackupPath:   backupPath,
		BytesWritten: len(params.Content),
		LinesWritten: lines,
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}
	return string(result), nil
}

// ========================================
// EDIT FILE TOOL
// ========================================

type EditFileInput struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	Occurrence int    `json:"occurrence,omitempty"` // 0 = all, 1 = first (default), 2 = second, etc.
}

type EditFileOutput struct {
	Path         string `json:"path"`
	BackupPath   string `json:"backup_path,omitempty"`
	Replacements int    `json:"replacements"`
	Diff         string `json:"diff"`
}

func (r *CodeToolRegistry) EditFile(ctx context.Context, input json.RawMessage) (string, error) {
	var params EditFileInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if params.OldString == "" {
		return "", fmt.Errorf("old_string is required")
	}

	absPath, err := r.resolveAndValidatePath(params.Path)
	if err != nil {
		return "", err
	}

	// Read current content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	// Check if old_string exists
	if !strings.Contains(oldContent, params.OldString) {
		return "", fmt.Errorf("old_string not found in file")
	}

	// Create backup before editing
	var backupPath string
	backup, backupErr := r.backupMgr.CreateBackup(params.Path)
	if backupErr == nil {
		backupPath = backup.BackupPath
	}

	// Perform replacement
	var newContent string
	var replacements int

	if params.Occurrence == 0 {
		// Replace all occurrences
		replacements = strings.Count(oldContent, params.OldString)
		newContent = strings.ReplaceAll(oldContent, params.OldString, params.NewString)
	} else {
		// Replace specific occurrence
		occurrence := params.Occurrence
		if occurrence < 1 {
			occurrence = 1
		}

		parts := strings.SplitN(oldContent, params.OldString, occurrence+1)
		if len(parts) <= occurrence {
			return "", fmt.Errorf("occurrence %d not found (only %d occurrences exist)", occurrence, len(parts)-1)
		}

		// Rebuild with replacement at specific occurrence
		newContent = strings.Join(parts[:occurrence], params.OldString) + params.NewString + parts[occurrence]
		replacements = 1
	}

	// Write new content
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Generate simple diff
	diff := generateSimpleDiff(params.OldString, params.NewString)

	output := EditFileOutput{
		Path:         params.Path,
		BackupPath:   backupPath,
		Replacements: replacements,
		Diff:         diff,
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}
	return string(result), nil
}
