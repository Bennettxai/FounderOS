package services

import (
	"regexp"
	"strings"
)

// BundledFile represents an extracted file from a bundle
type BundledFile struct {
	Path    string
	Content string
}

// ParseFileBundle parses OSA-5 multi-file bundle format
// Format:
// === FILE: path/to/file.ext ===
// file content here
// === END FILE ===
func ParseFileBundle(bundle string) []BundledFile {
	files := []BundledFile{}

	// Regex to match file markers
	fileStartRegex := regexp.MustCompile(`=== FILE: (.+?) ===`)
	fileEndMarker := "=== END FILE ==="

	// Split by file start marker
	parts := fileStartRegex.Split(bundle, -1)
	matches := fileStartRegex.FindAllStringSubmatch(bundle, -1)

	// Skip first part (before first file marker)
	for i := 1; i < len(parts); i++ {
		if i-1 >= len(matches) {
			break
		}

		filePath := strings.TrimSpace(matches[i-1][1])
		content := parts[i]

		// Remove END FILE marker and trailing content
		if endIdx := strings.Index(content, fileEndMarker); endIdx != -1 {
			content = content[:endIdx]
		}

		// Trim leading/trailing whitespace
		content = strings.TrimSpace(content)

		if filePath != "" {
			files = append(files, BundledFile{
				Path:    filePath,
				Content: content,
			})
		}
	}

	return files
}

// GetFileExtension returns file extension from path
func GetFileExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// GetLanguageFromExtension returns language for syntax highlighting
func GetLanguageFromExtension(ext string) string {
	langMap := map[string]string{
		"js":         "javascript",
		"jsx":        "javascript",
		"ts":         "typescript",
		"tsx":        "typescript",
		"py":         "python",
		"go":         "go",
		"rs":         "rust",
		"java":       "java",
		"cpp":        "cpp",
		"c":          "c",
		"cs":         "csharp",
		"rb":         "ruby",
		"php":        "php",
		"sh":         "bash",
		"yaml":       "yaml",
		"yml":        "yaml",
		"json":       "json",
		"xml":        "xml",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"md":         "markdown",
		"sql":        "sql",
		"dockerfile": "dockerfile",
		"env":        "bash",
		"gitignore":  "text",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return "plaintext"
}

// CategorizeFileType categorizes files for frontend display
func CategorizeFileType(path string) string {
	ext := GetFileExtension(path)
	lowerPath := strings.ToLower(path)

	// Check path patterns first
	if strings.Contains(lowerPath, "test") || strings.HasSuffix(lowerPath, ".test.js") ||
		strings.HasSuffix(lowerPath, ".spec.js") || strings.HasSuffix(lowerPath, "_test.go") {
		return "test"
	}

	if strings.Contains(lowerPath, "config") || lowerPath == ".gitignore" ||
		lowerPath == ".env" || lowerPath == "package.json" ||
		strings.Contains(lowerPath, ".config.") {
		return "config"
	}

	if strings.Contains(lowerPath, "docker") || lowerPath == "dockerfile" {
		return "deployment"
	}

	if ext == "md" || ext == "txt" || strings.Contains(lowerPath, "readme") {
		return "documentation"
	}

	if ext == "sql" || strings.Contains(lowerPath, "schema") ||
		strings.Contains(lowerPath, "migration") {
		return "schema"
	}

	// Code files (default)
	codeExts := map[string]bool{
		"js": true, "jsx": true, "ts": true, "tsx": true,
		"go": true, "py": true, "rb": true, "java": true,
		"cpp": true, "c": true, "rs": true, "cs": true,
		"php": true, "html": true, "css": true, "scss": true,
	}

	if codeExts[ext] {
		return "code"
	}

	return "documentation"
}
