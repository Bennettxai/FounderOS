//go:build ignore

// Package main provides an automation tool for migrating error handling patterns
// to use the centralized error response system (CUS-91).
//
// Usage:
//   go run scripts/migrate_errors.go [flags] <files...>
//
// Flags:
//   --dry-run    Preview changes without modifying files
//   --verbose    Show detailed operation logs
//   --backup     Create .bak files before modifying (default: true)
//
// Examples:
//   # Dry run on single file
//   go run scripts/migrate_errors.go --dry-run internal/handlers/chat.go
//
//   # Apply migration to multiple files
//   go run scripts/migrate_errors.go internal/handlers/*.go
//
//   # Apply migration to entire handlers directory
//   go run scripts/migrate_errors.go internal/handlers/*.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// Error pattern definitions based on CUS-91_MIGRATION_PROGRESS.md
type errorPattern struct {
	statusCode    string
	oldPattern    *regexp.Regexp
	newFunc       string
	requiresField bool // If true, needs to extract field/resource name
	requiresErr   bool // If true, needs to find err variable
}

var errorPatterns = []errorPattern{
	// 401 Unauthorized - most common
	{
		statusCode: "http.StatusUnauthorized",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusUnauthorized,\s*gin\.H\{"error":\s*"[^"]*"\}\)`),
		newFunc:    "utils.RespondUnauthorized(c, slog.Default())",
	},
	// 400 Bad Request - invalid ID
	{
		statusCode:    "http.StatusBadRequest",
		oldPattern:    regexp.MustCompile(`c\.JSON\(http\.StatusBadRequest,\s*gin\.H\{"error":\s*"Invalid\s+([^"]+)"\}\)`),
		newFunc:       "utils.RespondInvalidID(c, slog.Default(), \"%s\")",
		requiresField: true,
	},
	// 400 Bad Request - general
	{
		statusCode: "http.StatusBadRequest",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusBadRequest,\s*gin\.H\{"error":\s*"([^"]+)"\}\)`),
		newFunc:    "utils.RespondBadRequest(c, slog.Default(), \"%s\")",
	},
	// 400 Bad Request - with err.Error()
	{
		statusCode:  "http.StatusBadRequest",
		oldPattern:  regexp.MustCompile(`c\.JSON\(http\.StatusBadRequest,\s*gin\.H\{"error":\s*err\.Error\(\)\}\)`),
		newFunc:     "utils.RespondInvalidRequest(c, slog.Default(), err)",
		requiresErr: true,
	},
	// 404 Not Found
	{
		statusCode:    "http.StatusNotFound",
		oldPattern:    regexp.MustCompile(`c\.JSON\(http\.StatusNotFound,\s*gin\.H\{"error":\s*"([^"]+)\s+not found"\}\)`),
		newFunc:       "utils.RespondNotFound(c, slog.Default(), \"%s\")",
		requiresField: true,
	},
	// 404 Not Found - generic
	{
		statusCode: "http.StatusNotFound",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusNotFound,\s*gin\.H\{"error":\s*"[^"]*"\}\)`),
		newFunc:    "utils.RespondNotFound(c, slog.Default(), \"Resource\")",
	},
	// 500 Internal Server Error - with err
	{
		statusCode:  "http.StatusInternalServerError",
		oldPattern:  regexp.MustCompile(`c\.JSON\(http\.StatusInternalServerError,\s*gin\.H\{"error":\s*"Failed to ([^"]+)"\}\)`),
		newFunc:     "utils.RespondInternalError(c, slog.Default(), \"%s\", err)",
		requiresErr: true,
	},
	// 500 Internal Server Error - generic
	{
		statusCode: "http.StatusInternalServerError",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusInternalServerError,\s*gin\.H\{"error":\s*"[^"]*"\}\)`),
		newFunc:    "utils.RespondInternalError(c, slog.Default(), \"operation\", nil)",
	},
	// 403 Forbidden
	{
		statusCode: "http.StatusForbidden",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusForbidden,\s*gin\.H\{"error":\s*"([^"]+)"\}\)`),
		newFunc:    "utils.RespondForbidden(c, slog.Default(), \"%s\")",
	},
	// 409 Conflict
	{
		statusCode: "http.StatusConflict",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusConflict,\s*gin\.H\{"error":\s*"([^"]+)"\}\)`),
		newFunc:    "utils.RespondConflict(c, slog.Default(), \"%s\")",
	},
	// 503 Service Unavailable
	{
		statusCode: "http.StatusServiceUnavailable",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusServiceUnavailable,\s*gin\.H\{"error":\s*"([^"]+)\s+is temporarily unavailable"\}\)`),
		newFunc:    "utils.RespondServiceUnavailable(c, slog.Default(), \"%s\")",
	},
	// 501 Not Implemented
	{
		statusCode: "http.StatusNotImplemented",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusNotImplemented,\s*gin\.H\{"error":\s*"([^"]+)\s+is not implemented"\}\)`),
		newFunc:    "utils.RespondNotImplemented(c, slog.Default(), \"%s\")",
	},
	// 429 Too Many Requests
	{
		statusCode: "http.StatusTooManyRequests",
		oldPattern: regexp.MustCompile(`c\.JSON\(http\.StatusTooManyRequests,\s*gin\.H\{"error":\s*"Too many requests to ([^"]+)"\}\)`),
		newFunc:    "utils.RespondTooManyRequests(c, slog.Default(), \"%s\")",
	},
}

// MigrationStats tracks the migration statistics
type MigrationStats struct {
	FilesProcessed  int
	FilesModified   int
	ErrorsReplaced  int
	ImportsAdded    int
	SpecialCases    []string
	Errors          []string
}

// Config holds the command-line configuration
type Config struct {
	DryRun  bool
	Verbose bool
	Backup  bool
	Files   []string
}

func main() {
	config := parseFlags()

	if len(config.Files) == 0 {
		fmt.Println("Usage: go run scripts/migrate_errors.go [flags] <files...>")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	stats := &MigrationStats{}

	fmt.Printf("CUS-91 Error Centralization Migration Tool\n")
	fmt.Printf("==========================================\n\n")

	if config.DryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be modified\n")
	}

	for _, file := range config.Files {
		if err := processFile(file, config, stats); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", file, err))
			if config.Verbose {
				fmt.Printf("❌ ERROR processing %s: %v\n", file, err)
			}
		}
	}

	printStats(stats, config)

	if len(stats.Errors) > 0 {
		os.Exit(1)
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview changes without modifying files")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show detailed operation logs")
	flag.BoolVar(&config.Backup, "backup", true, "Create .bak files before modifying")

	flag.Parse()
	config.Files = flag.Args()

	return config
}

func processFile(filePath string, config *Config, stats *MigrationStats) error {
	stats.FilesProcessed++

	if config.Verbose {
		fmt.Printf("\n📄 Processing: %s\n", filePath)
	}

	// Read the file
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the Go source file to validate it's valid Go
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Check if file needs migration
	originalSrc := string(src)
	needsMigration := strings.Contains(originalSrc, `gin.H{"error":`)

	if !needsMigration {
		if config.Verbose {
			fmt.Printf("  ✓ No error patterns found - skipping\n")
		}
		return nil
	}

	// Track if we made changes
	modified := false
	replacements := 0

	// Apply text-based replacements (simpler than AST manipulation for this case)
	newSrc := originalSrc

	// Detect special cases first
	specialCases := detectSpecialCases(originalSrc)
	if len(specialCases) > 0 {
		stats.SpecialCases = append(stats.SpecialCases, fmt.Sprintf("%s: %v", filePath, specialCases))
	}

	// Apply each error pattern
	for _, pattern := range errorPatterns {
		matches := pattern.oldPattern.FindAllStringSubmatch(newSrc, -1)
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			oldCode := match[0]
			var newCode string

			if pattern.requiresField && len(match) > 1 {
				// Extract field name from capture group
				field := match[1]
				newCode = fmt.Sprintf(pattern.newFunc, field)
			} else if strings.Contains(pattern.newFunc, "%s") && len(match) > 1 {
				// Extract message from capture group
				message := match[1]
				newCode = fmt.Sprintf(pattern.newFunc, message)
			} else {
				newCode = pattern.newFunc
			}

			// Replace the old pattern with new centralized call
			newSrc = strings.Replace(newSrc, oldCode, newCode, 1)
			replacements++
			modified = true

			if config.Verbose {
				fmt.Printf("  ✓ Replaced: %s\n", oldCode[:min(len(oldCode), 60)]+"...")
				fmt.Printf("    With:     %s\n", newCode)
			}
		}
	}

	if !modified {
		if config.Verbose {
			fmt.Printf("  ⚠️  File contains gin.H errors but no patterns matched - may need manual review\n")
		}
		return nil
	}

	// Add required imports if missing
	needsSlog := !strings.Contains(originalSrc, `"log/slog"`)
	needsUtils := !strings.Contains(originalSrc, `"github.com/rhl/businessos-backend/internal/utils"`)

	if needsSlog || needsUtils {
		// Parse the modified source to add imports
		fset2 := token.NewFileSet()
		file2, err := parser.ParseFile(fset2, filePath, newSrc, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse modified file: %w", err)
		}

		if needsSlog {
			astutil.AddImport(fset2, file2, "log/slog")
			stats.ImportsAdded++
			if config.Verbose {
				fmt.Printf("  ✓ Added import: log/slog\n")
			}
		}

		if needsUtils {
			astutil.AddImport(fset2, file2, "github.com/rhl/businessos-backend/internal/utils")
			stats.ImportsAdded++
			if config.Verbose {
				fmt.Printf("  ✓ Added import: internal/utils\n")
			}
		}

		// Format the code with new imports
		var buf bytes.Buffer
		if err := format.Node(&buf, fset2, file2); err != nil {
			return fmt.Errorf("failed to format code: %w", err)
		}
		newSrc = buf.String()
	} else {
		// Just format the modified source
		formatted, err := format.Source([]byte(newSrc))
		if err != nil {
			return fmt.Errorf("failed to format source: %w", err)
		}
		newSrc = string(formatted)
	}

	// Update statistics
	stats.FilesModified++
	stats.ErrorsReplaced += replacements

	if config.DryRun {
		if config.Verbose {
			fmt.Printf("\n  📋 PREVIEW OF CHANGES:\n")
			fmt.Printf("  %s\n", strings.Repeat("=", 60))
			showDiff(originalSrc, newSrc)
		}
		return nil
	}

	// Create backup if requested
	if config.Backup {
		backupPath := filePath + ".bak"
		if err := os.WriteFile(backupPath, src, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		if config.Verbose {
			fmt.Printf("  ✓ Created backup: %s\n", backupPath)
		}
	}

	// Write the modified file
	if err := os.WriteFile(filePath, []byte(newSrc), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if config.Verbose {
		fmt.Printf("  ✅ Successfully migrated (%d replacements)\n", replacements)
	}

	return nil
}

func detectSpecialCases(src string) []string {
	var cases []string

	// SSE streaming errors
	if strings.Contains(src, "SSE") || strings.Contains(src, "streaming.StreamEvent") {
		cases = append(cases, "SSE streaming - manual review needed")
	}

	// WebSocket errors
	if strings.Contains(src, "websocket") || strings.Contains(src, "gorilla/websocket") {
		cases = append(cases, "WebSocket - manual review needed")
	}

	// Custom error structures
	if regexp.MustCompile(`gin\.H\{[^}]*"error":[^}]*"[^"]*"[^}]*,`).MatchString(src) {
		cases = append(cases, "Complex error structures - manual review needed")
	}

	return cases
}

func showDiff(old, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	maxLines := 20 // Show first 20 changed lines
	shown := 0

	for i := 0; i < min(len(oldLines), len(newLines)) && shown < maxLines; i++ {
		if oldLines[i] != newLines[i] {
			fmt.Printf("  - %s\n", oldLines[i])
			fmt.Printf("  + %s\n", newLines[i])
			shown++
		}
	}

	if shown >= maxLines {
		fmt.Printf("  ... (more changes not shown)\n")
	}
}

func printStats(stats *MigrationStats, config *Config) {
	fmt.Printf("\n\n")
	fmt.Printf("📊 MIGRATION STATISTICS\n")
	fmt.Printf("======================\n\n")
	fmt.Printf("Files processed:  %d\n", stats.FilesProcessed)
	fmt.Printf("Files modified:   %d\n", stats.FilesModified)
	fmt.Printf("Errors replaced:  %d\n", stats.ErrorsReplaced)
	fmt.Printf("Imports added:    %d\n", stats.ImportsAdded)

	if len(stats.SpecialCases) > 0 {
		fmt.Printf("\n⚠️  SPECIAL CASES (manual review needed):\n")
		for _, sc := range stats.SpecialCases {
			fmt.Printf("  • %s\n", sc)
		}
	}

	if len(stats.Errors) > 0 {
		fmt.Printf("\n❌ ERRORS:\n")
		for _, err := range stats.Errors {
			fmt.Printf("  • %s\n", err)
		}
	}

	if config.DryRun {
		fmt.Printf("\n💡 Run without --dry-run to apply changes\n")
	} else {
		fmt.Printf("\n✅ Migration complete!\n")
		if config.Backup {
			fmt.Printf("💾 Backup files created with .bak extension\n")
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
