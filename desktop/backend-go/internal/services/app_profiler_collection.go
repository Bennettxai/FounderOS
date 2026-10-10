package services

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// buildDirectoryTree builds the directory structure tree
func (s *AppProfilerService) buildDirectoryTree(rootPath string, opts *ProfileOptions, depth int) *DirectoryTree {
	if depth > opts.MaxDepth {
		return nil
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return nil
	}

	name := filepath.Base(rootPath)

	// Check exclusions
	for _, pattern := range opts.ExcludePatterns {
		if name == pattern || strings.Contains(rootPath, pattern) {
			return nil
		}
	}

	// Skip hidden files/dirs
	if !opts.IncludeHidden && strings.HasPrefix(name, ".") && name != "." {
		return nil
	}

	tree := &DirectoryTree{
		Name: name,
		Path: rootPath,
		Size: info.Size(),
	}

	if info.IsDir() {
		tree.Type = "directory"
		entries, err := os.ReadDir(rootPath)
		if err != nil {
			return tree
		}

		tree.Children = make([]*DirectoryTree, 0)
		for _, entry := range entries {
			childPath := filepath.Join(rootPath, entry.Name())
			child := s.buildDirectoryTree(childPath, opts, depth+1)
			if child != nil {
				tree.Children = append(tree.Children, child)
			}
		}
	} else {
		tree.Type = "file"
		tree.FileType = strings.TrimPrefix(filepath.Ext(name), ".")
	}

	return tree
}

// analyzeLanguages analyzes programming languages used
func (s *AppProfilerService) analyzeLanguages(rootPath string, opts *ProfileOptions) ([]LanguageInfo, int, int) {
	languageExtensions := map[string]string{
		".go":     "Go",
		".js":     "JavaScript",
		".ts":     "TypeScript",
		".jsx":    "JavaScript (React)",
		".tsx":    "TypeScript (React)",
		".svelte": "Svelte",
		".vue":    "Vue",
		".py":     "Python",
		".rb":     "Ruby",
		".java":   "Java",
		".kt":     "Kotlin",
		".swift":  "Swift",
		".rs":     "Rust",
		".c":      "C",
		".cpp":    "C++",
		".cs":     "C#",
		".php":    "PHP",
		".sql":    "SQL",
		".html":   "HTML",
		".css":    "CSS",
		".scss":   "SCSS",
		".json":   "JSON",
		".yaml":   "YAML",
		".yml":    "YAML",
		".md":     "Markdown",
		".sh":     "Shell",
		".bat":    "Batch",
	}

	langStats := make(map[string]struct{ files, lines int })
	totalLines := 0
	totalFiles := 0

	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Check exclusions
		name := d.Name()
		for _, pattern := range opts.ExcludePatterns {
			if name == pattern {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		if lang, ok := languageExtensions[ext]; ok {
			lines := s.countLines(path)
			stats := langStats[lang]
			stats.files++
			stats.lines += lines
			langStats[lang] = stats
			totalLines += lines
			totalFiles++
		}

		return nil
	})

	// Convert to slice and calculate percentages
	languages := make([]LanguageInfo, 0, len(langStats))
	for lang, stats := range langStats {
		percentage := 0.0
		if totalLines > 0 {
			percentage = float64(stats.lines) / float64(totalLines) * 100
		}
		languages = append(languages, LanguageInfo{
			Name:       lang,
			Files:      stats.files,
			Lines:      stats.lines,
			Percentage: percentage,
		})
	}

	// Sort by lines (descending)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Lines > languages[j].Lines
	})

	return languages, totalLines, totalFiles
}

// countLines counts lines in a file
func (s *AppProfilerService) countLines(path string) int {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.Split(string(content), "\n"))
}

// detectTechStack detects the technology stack
func (s *AppProfilerService) detectTechStack(rootPath string, languages []LanguageInfo) (AppType, TechStack) {
	stack := TechStack{
		Frontend:  make([]string, 0),
		Backend:   make([]string, 0),
		Database:  make([]string, 0),
		DevOps:    make([]string, 0),
		Testing:   make([]string, 0),
		BuildTool: make([]string, 0),
	}

	appType := AppTypeWeb

	// Check for common config files
	checks := map[string]func(){
		"package.json": func() {
			content, _ := os.ReadFile(filepath.Join(rootPath, "package.json"))
			contentStr := string(content)
			if strings.Contains(contentStr, "react") {
				stack.Frontend = append(stack.Frontend, "React")
			}
			if strings.Contains(contentStr, "svelte") {
				stack.Frontend = append(stack.Frontend, "Svelte")
			}
			if strings.Contains(contentStr, "vue") {
				stack.Frontend = append(stack.Frontend, "Vue")
			}
			if strings.Contains(contentStr, "next") {
				stack.Frontend = append(stack.Frontend, "Next.js")
				appType = AppTypeFullStack
			}
			if strings.Contains(contentStr, "express") {
				stack.Backend = append(stack.Backend, "Express")
			}
			if strings.Contains(contentStr, "tailwind") {
				stack.Frontend = append(stack.Frontend, "Tailwind CSS")
			}
			if strings.Contains(contentStr, "jest") || strings.Contains(contentStr, "vitest") {
				stack.Testing = append(stack.Testing, "Jest/Vitest")
			}
		},
		"go.mod": func() {
			stack.Backend = append(stack.Backend, "Go")
			appType = AppTypeAPI
		},
		"requirements.txt": func() {
			stack.Backend = append(stack.Backend, "Python")
		},
		"Gemfile": func() {
			stack.Backend = append(stack.Backend, "Ruby")
		},
		"docker-compose.yml": func() {
			stack.DevOps = append(stack.DevOps, "Docker Compose")
		},
		"Dockerfile": func() {
			stack.DevOps = append(stack.DevOps, "Docker")
		},
		".github/workflows": func() {
			stack.DevOps = append(stack.DevOps, "GitHub Actions")
		},
		"prisma": func() {
			stack.Database = append(stack.Database, "Prisma")
		},
	}

	for file, check := range checks {
		if _, err := os.Stat(filepath.Join(rootPath, file)); err == nil {
			check()
		}
	}

	// Infer database from languages
	for _, lang := range languages {
		if lang.Name == "SQL" {
			stack.Database = append(stack.Database, "SQL")
		}
	}

	// Detect if it's a fullstack app
	if len(stack.Frontend) > 0 && len(stack.Backend) > 0 {
		appType = AppTypeFullStack
	}

	return appType, stack
}

// detectFrameworks detects frameworks used
func (s *AppProfilerService) detectFrameworks(rootPath string) []string {
	frameworks := make([]string, 0)

	// SvelteKit
	if _, err := os.Stat(filepath.Join(rootPath, "svelte.config.js")); err == nil {
		frameworks = append(frameworks, "SvelteKit")
	}

	// Next.js
	if _, err := os.Stat(filepath.Join(rootPath, "next.config.js")); err == nil {
		frameworks = append(frameworks, "Next.js")
	}

	// Go frameworks
	goMod := filepath.Join(rootPath, "go.mod")
	if content, err := os.ReadFile(goMod); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "gin-gonic/gin") {
			frameworks = append(frameworks, "Gin")
		}
		if strings.Contains(contentStr, "labstack/echo") {
			frameworks = append(frameworks, "Echo")
		}
		if strings.Contains(contentStr, "go-chi/chi") {
			frameworks = append(frameworks, "Chi")
		}
		if strings.Contains(contentStr, "gorilla/mux") {
			frameworks = append(frameworks, "Gorilla Mux")
		}
	}

	return frameworks
}

// detectConventions detects coding conventions
func (s *AppProfilerService) detectConventions(rootPath string, opts *ProfileOptions) CodeConventions {
	conv := CodeConventions{
		CommonPatterns: make([]string, 0),
	}

	// Sample a few files to detect conventions
	var sampleContent string
	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if ext == ".go" || ext == ".ts" || ext == ".js" || ext == ".svelte" {
			content, _ := os.ReadFile(path)
			if len(sampleContent) < 10000 {
				sampleContent += string(content)
			}
		}
		return nil
	})

	// Detect naming style
	if regexp.MustCompile(`[a-z]+_[a-z]+`).MatchString(sampleContent) {
		conv.NamingStyle = "snake_case"
	} else if regexp.MustCompile(`[a-z]+[A-Z][a-z]+`).MatchString(sampleContent) {
		conv.NamingStyle = "camelCase"
	}

	// Detect indent style
	if strings.Contains(sampleContent, "\t") {
		conv.IndentStyle = "tabs"
	} else {
		conv.IndentStyle = "spaces"
		// Detect indent size
		if regexp.MustCompile(`\n {4}[^ ]`).MatchString(sampleContent) {
			conv.IndentSize = 4
		} else if regexp.MustCompile(`\n {2}[^ ]`).MatchString(sampleContent) {
			conv.IndentSize = 2
		}
	}

	// Detect quote style
	singleQuotes := strings.Count(sampleContent, "'")
	doubleQuotes := strings.Count(sampleContent, "\"")
	if singleQuotes > doubleQuotes {
		conv.QuoteStyle = "single"
	} else {
		conv.QuoteStyle = "double"
	}

	// Detect semicolons
	conv.Semicolons = strings.Contains(sampleContent, ";")

	// Detect common patterns
	if strings.Contains(sampleContent, "interface") {
		conv.CommonPatterns = append(conv.CommonPatterns, "interfaces")
	}
	if strings.Contains(sampleContent, "async") {
		conv.CommonPatterns = append(conv.CommonPatterns, "async/await")
	}
	if strings.Contains(sampleContent, "struct") {
		conv.CommonPatterns = append(conv.CommonPatterns, "structs")
	}

	return conv
}
