package services

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// analyzeComponents analyzes UI components
func (s *AppProfilerService) analyzeComponents(rootPath string, stack TechStack, opts *ProfileOptions) []ComponentInfo {
	components := make([]ComponentInfo, 0)

	// Determine component patterns based on stack
	var patterns []string
	var componentRegex *regexp.Regexp

	if containsAny(stack.Frontend, "Svelte", "SvelteKit") {
		patterns = []string{"*.svelte"}
		componentRegex = regexp.MustCompile(`export\s+let\s+(\w+)`)
	} else if containsAny(stack.Frontend, "React", "Next.js") {
		patterns = []string{"*.tsx", "*.jsx"}
		componentRegex = regexp.MustCompile(`(?:interface|type)\s+\w*Props`)
	} else if containsAny(stack.Frontend, "Vue") {
		patterns = []string{"*.vue"}
	}

	for _, pattern := range patterns {
		filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			// Check exclusions
			for _, exclude := range opts.ExcludePatterns {
				if strings.Contains(path, exclude) {
					return nil
				}
			}

			matched, _ := filepath.Match(pattern, d.Name())
			if !matched {
				return nil
			}

			content, _ := os.ReadFile(path)
			contentStr := string(content)

			comp := ComponentInfo{
				Name:     strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
				FilePath: path,
				Lines:    len(strings.Split(contentStr, "\n")),
				Props:    make([]string, 0),
				Events:   make([]string, 0),
				UsedIn:   make([]string, 0),
			}

			// Determine component type
			relPath, _ := filepath.Rel(rootPath, path)
			if strings.Contains(relPath, "page") || strings.Contains(relPath, "routes") {
				comp.Type = "page"
			} else if strings.Contains(relPath, "layout") {
				comp.Type = "layout"
			} else {
				comp.Type = "component"
			}

			// Extract props
			if componentRegex != nil {
				matches := componentRegex.FindAllStringSubmatch(contentStr, -1)
				for _, match := range matches {
					if len(match) > 1 {
						comp.Props = append(comp.Props, match[1])
					}
				}
			}

			components = append(components, comp)
			return nil
		})
	}

	return components
}

// analyzeModules analyzes code modules
func (s *AppProfilerService) analyzeModules(rootPath string, languages []LanguageInfo, opts *ProfileOptions) []ModuleInfo {
	modules := make([]ModuleInfo, 0)

	// Determine primary language
	primaryLang := ""
	if len(languages) > 0 {
		primaryLang = languages[0].Name
	}

	var patterns []string
	switch primaryLang {
	case "Go":
		patterns = []string{"*.go"}
	case "TypeScript", "JavaScript":
		patterns = []string{"*.ts", "*.js"}
	case "Python":
		patterns = []string{"*.py"}
	}

	for _, pattern := range patterns {
		filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			// Check exclusions
			for _, exclude := range opts.ExcludePatterns {
				if strings.Contains(path, exclude) {
					return nil
				}
			}

			matched, _ := filepath.Match(pattern, d.Name())
			if !matched {
				return nil
			}

			content, _ := os.ReadFile(path)
			contentStr := string(content)

			mod := ModuleInfo{
				Name:    strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
				Path:    path,
				Lines:   len(strings.Split(contentStr, "\n")),
				Exports: make([]string, 0),
				Imports: make([]string, 0),
			}

			// Determine module type
			relPath, _ := filepath.Rel(rootPath, path)
			relPathLower := strings.ToLower(relPath)
			switch {
			case strings.Contains(relPathLower, "handler"):
				mod.Type = "handler"
			case strings.Contains(relPathLower, "service"):
				mod.Type = "service"
			case strings.Contains(relPathLower, "repository") || strings.Contains(relPathLower, "repo"):
				mod.Type = "repository"
			case strings.Contains(relPathLower, "util") || strings.Contains(relPathLower, "helper"):
				mod.Type = "utility"
			case strings.Contains(relPathLower, "model") || strings.Contains(relPathLower, "entity"):
				mod.Type = "model"
			case strings.Contains(relPathLower, "middleware"):
				mod.Type = "middleware"
			default:
				mod.Type = "module"
			}

			// Extract exports (simplified)
			if primaryLang == "Go" {
				exportRegex := regexp.MustCompile(`func\s+([A-Z]\w+)`)
				matches := exportRegex.FindAllStringSubmatch(contentStr, -1)
				for _, match := range matches {
					if len(match) > 1 {
						mod.Exports = append(mod.Exports, match[1])
					}
				}
			}

			modules = append(modules, mod)
			return nil
		})
	}

	return modules
}

// analyzeEndpoints analyzes API endpoints
func (s *AppProfilerService) analyzeEndpoints(rootPath string, stack TechStack, opts *ProfileOptions) []APIEndpointInfo {
	endpoints := make([]APIEndpointInfo, 0)

	// Go patterns
	goEndpointPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:Get|Post|Put|Delete|Patch|Handle)\s*\(\s*"([^"]+)"`),
		regexp.MustCompile(`r\.(?:Get|Post|Put|Delete|Patch)\s*\(\s*"([^"]+)"`),
		regexp.MustCompile(`\.(?:GET|POST|PUT|DELETE|PATCH)\s*\(\s*"([^"]+)"`),
		regexp.MustCompile(`HandleFunc\s*\(\s*"([^"]+)"`),
	}

	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Check exclusions
		for _, exclude := range opts.ExcludePatterns {
			if strings.Contains(path, exclude) {
				return nil
			}
		}

		ext := filepath.Ext(d.Name())
		if ext != ".go" && ext != ".ts" && ext != ".js" {
			return nil
		}

		content, _ := os.ReadFile(path)
		contentStr := string(content)

		for _, pattern := range goEndpointPatterns {
			matches := pattern.FindAllStringSubmatch(contentStr, -1)
			for _, match := range matches {
				if len(match) > 1 {
					endpointPath := match[1]
					method := "GET"

					// Infer method from pattern
					fullMatch := match[0]
					if strings.Contains(strings.ToLower(fullMatch), "post") {
						method = "POST"
					} else if strings.Contains(strings.ToLower(fullMatch), "put") {
						method = "PUT"
					} else if strings.Contains(strings.ToLower(fullMatch), "delete") {
						method = "DELETE"
					} else if strings.Contains(strings.ToLower(fullMatch), "patch") {
						method = "PATCH"
					}

					endpoints = append(endpoints, APIEndpointInfo{
						Method:      method,
						Path:        endpointPath,
						HandlerPath: path,
					})
				}
			}
		}

		return nil
	})

	return endpoints
}

// analyzeDatabaseSchema analyzes database schema from migrations
func (s *AppProfilerService) analyzeDatabaseSchema(rootPath string) *DatabaseSchemaInfo {
	schema := &DatabaseSchemaInfo{
		Tables:     make([]TableInfo, 0),
		Migrations: make([]MigrationInfo, 0),
	}

	// Look for migrations directory
	migrationPaths := []string{
		filepath.Join(rootPath, "migrations"),
		filepath.Join(rootPath, "db", "migrations"),
		filepath.Join(rootPath, "internal", "database", "migrations"),
	}

	for _, migPath := range migrationPaths {
		if _, err := os.Stat(migPath); err == nil {
			files, _ := os.ReadDir(migPath)
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".sql") {
					schema.Migrations = append(schema.Migrations, MigrationInfo{
						Name: f.Name(),
						Path: filepath.Join(migPath, f.Name()),
					})

					// Parse SQL for tables
					content, _ := os.ReadFile(filepath.Join(migPath, f.Name()))
					tables := s.parseTablesFromSQL(string(content))
					schema.Tables = append(schema.Tables, tables...)
				}
			}
			break
		}
	}

	schema.TotalTables = len(schema.Tables)
	return schema
}

// parseTablesFromSQL extracts table information from SQL
func (s *AppProfilerService) parseTablesFromSQL(sql string) []TableInfo {
	tables := make([]TableInfo, 0)

	tableRegex := regexp.MustCompile(`CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(\w+)`)
	matches := tableRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, TableInfo{
				Name:    match[1],
				Columns: make([]ColumnInfo, 0),
			})
		}
	}

	return tables
}

// extractReadmeSummary extracts summary from README
func (s *AppProfilerService) extractReadmeSummary(rootPath string) string {
	readmeFiles := []string{"README.md", "readme.md", "README", "README.txt"}

	for _, readme := range readmeFiles {
		path := filepath.Join(rootPath, readme)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Extract first paragraph or section
		lines := strings.Split(string(content), "\n")
		var summary strings.Builder
		inContent := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip title
			if strings.HasPrefix(trimmed, "#") && !inContent {
				continue
			}

			// Skip empty lines at start
			if trimmed == "" && !inContent {
				continue
			}

			inContent = true

			// Stop at next heading or after enough content
			if strings.HasPrefix(trimmed, "#") || summary.Len() > 500 {
				break
			}

			summary.WriteString(trimmed)
			summary.WriteString(" ")
		}

		return strings.TrimSpace(summary.String())
	}

	return ""
}

// detectIntegrations detects external integrations
func (s *AppProfilerService) detectIntegrations(rootPath string) []IntegrationPoint {
	integrations := make([]IntegrationPoint, 0)

	// Check for common integration patterns
	envFile := filepath.Join(rootPath, ".env.example")
	content, err := os.ReadFile(envFile)
	if err != nil {
		envFile = filepath.Join(rootPath, ".env.sample")
		content, err = os.ReadFile(envFile)
	}

	if err == nil {
		contentStr := string(content)

		// Database URLs
		if strings.Contains(contentStr, "DATABASE_URL") {
			integrations = append(integrations, IntegrationPoint{
				Name: "Database",
				Type: "database",
			})
		}

		// Redis
		if strings.Contains(contentStr, "REDIS") {
			integrations = append(integrations, IntegrationPoint{
				Name: "Redis",
				Type: "database",
			})
		}

		// OpenAI/Anthropic
		if strings.Contains(contentStr, "OPENAI") || strings.Contains(contentStr, "ANTHROPIC") {
			integrations = append(integrations, IntegrationPoint{
				Name: "AI Provider",
				Type: "api",
			})
		}

		// Stripe
		if strings.Contains(contentStr, "STRIPE") {
			integrations = append(integrations, IntegrationPoint{
				Name: "Stripe",
				Type: "api",
			})
		}
	}

	return integrations
}

// generateDescription generates a description for the profile
func (s *AppProfilerService) generateDescription(profile *ApplicationProfile) string {
	var desc strings.Builder

	desc.WriteString(fmt.Sprintf("A %s application", profile.AppType))

	if len(profile.Languages) > 0 {
		langs := make([]string, 0)
		for i, l := range profile.Languages {
			if i >= 3 {
				break
			}
			langs = append(langs, l.Name)
		}
		desc.WriteString(fmt.Sprintf(" built with %s", strings.Join(langs, ", ")))
	}

	if len(profile.Frameworks) > 0 {
		desc.WriteString(fmt.Sprintf(" using %s", strings.Join(profile.Frameworks, ", ")))
	}

	desc.WriteString(fmt.Sprintf(". Contains %d files with %d lines of code.", profile.FileCount, profile.LinesOfCode))

	return desc.String()
}
