package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ========================================
// SEARCH CODE TOOL
// ========================================

type SearchCodeInput struct {
	Query       string `json:"query"`
	Path        string `json:"path,omitempty"`
	FilePattern string `json:"file_pattern,omitempty"`
	MaxResults  int    `json:"max_results,omitempty"`
	Regex       bool   `json:"regex,omitempty"`
}

type SearchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type SearchCodeOutput struct {
	Query   string        `json:"query"`
	Matches []SearchMatch `json:"matches"`
	Total   int           `json:"total"`
}

func (r *CodeToolRegistry) SearchCode(ctx context.Context, input json.RawMessage) (string, error) {
	var params SearchCodeInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	searchPath := r.workspaceRoot
	if params.Path != "" {
		absPath, err := r.resolveAndValidatePath(params.Path)
		if err != nil {
			return "", err
		}
		searchPath = absPath
	}

	maxResults := 50
	if params.MaxResults > 0 {
		maxResults = params.MaxResults
	}

	var matches []SearchMatch
	var searchRegex *regexp.Regexp

	if params.Regex {
		var err error
		searchRegex, err = regexp.Compile(params.Query)
		if err != nil {
			return "", fmt.Errorf("invalid regex: %w", err)
		}
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories and hidden files
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip backup directory
		if strings.Contains(path, BackupDirName) {
			return nil
		}

		// Apply file pattern filter
		if params.FilePattern != "" {
			matched, _ := filepath.Match(params.FilePattern, info.Name())
			if !matched {
				return nil
			}
		}

		// Skip binary files (simple heuristic)
		if isBinaryFile(info.Name()) {
			return nil
		}

		// Search in file
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		relPath, _ := filepath.Rel(r.workspaceRoot, path)
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			var found bool
			if params.Regex && searchRegex != nil {
				found = searchRegex.MatchString(line)
			} else {
				found = strings.Contains(line, params.Query)
			}

			if found {
				matches = append(matches, SearchMatch{
					File:    relPath,
					Line:    lineNum,
					Content: truncateString(line, 200),
				})

				if len(matches) >= maxResults {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("search failed: %w", err)
	}

	output := SearchCodeOutput{
		Query:   params.Query,
		Matches: matches,
		Total:   len(matches),
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}
	return string(result), nil
}

// ========================================
// LIST FILES TOOL
// ========================================

type ListFilesInput struct {
	Path      string `json:"path,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	MaxDepth  int    `json:"max_depth,omitempty"`
}

type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

type ListFilesOutput struct {
	Path  string      `json:"path"`
	Files []FileEntry `json:"files"`
	Total int         `json:"total"`
}

func (r *CodeToolRegistry) ListFiles(ctx context.Context, input json.RawMessage) (string, error) {
	var params ListFilesInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	searchPath := r.workspaceRoot
	if params.Path != "" {
		absPath, err := r.resolveAndValidatePath(params.Path)
		if err != nil {
			return "", err
		}
		searchPath = absPath
	}

	maxDepth := 1
	if params.Recursive {
		maxDepth = 10
	}
	if params.MaxDepth > 0 {
		maxDepth = params.MaxDepth
	}

	var files []FileEntry
	baseDepth := strings.Count(searchPath, string(filepath.Separator))

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Calculate depth
		currentDepth := strings.Count(path, string(filepath.Separator)) - baseDepth

		// Skip if too deep
		if currentDepth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip root
		if path == searchPath {
			return nil
		}

		// Skip hidden files/dirs
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply pattern filter
		if params.Pattern != "" {
			matched, _ := filepath.Match(params.Pattern, info.Name())
			if !matched && !info.IsDir() {
				return nil
			}
		}

		relPath, _ := filepath.Rel(r.workspaceRoot, path)

		entry := FileEntry{
			Name:     info.Name(),
			Path:     relPath,
			IsDir:    info.IsDir(),
			Modified: info.ModTime(),
		}

		if !info.IsDir() {
			entry.Size = info.Size()
		}

		files = append(files, entry)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	relPath, _ := filepath.Rel(r.workspaceRoot, searchPath)
	if relPath == "." {
		relPath = "/"
	}

	output := ListFilesOutput{
		Path:  relPath,
		Files: files,
		Total: len(files),
	}

	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}
	return string(result), nil
}
