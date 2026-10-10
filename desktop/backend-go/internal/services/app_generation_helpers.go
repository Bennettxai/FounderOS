package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/appgen"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// parseQueueItemID converts queue item ID string to UUID
func parseQueueItemID(queueItemID string) (uuid.UUID, error) {
	return uuid.Parse(queueItemID)
}

// inferFileCategory determines which subdirectory a file belongs to
func inferFileCategory(filename string) string {
	filename = strings.ToLower(filename)

	// Frontend patterns
	if strings.Contains(filename, ".svelte") ||
		strings.Contains(filename, ".tsx") ||
		strings.Contains(filename, ".jsx") ||
		strings.Contains(filename, "component") ||
		strings.Contains(filename, "frontend") ||
		strings.HasPrefix(filename, "src/") {
		return "frontend"
	}

	// Backend patterns
	if strings.HasSuffix(filename, ".go") ||
		strings.Contains(filename, "handler") ||
		strings.Contains(filename, "service") ||
		strings.Contains(filename, "repository") ||
		strings.Contains(filename, "backend") ||
		strings.Contains(filename, "internal/") {
		return "backend"
	}

	// Database patterns
	if strings.HasSuffix(filename, ".sql") ||
		strings.Contains(filename, "migration") ||
		strings.Contains(filename, "schema") ||
		strings.Contains(filename, "database") {
		return "database"
	}

	// Test patterns
	if strings.HasSuffix(filename, "_test.go") ||
		strings.HasSuffix(filename, ".test.ts") ||
		strings.HasSuffix(filename, ".spec.ts") ||
		strings.Contains(filename, "test") {
		return "tests"
	}

	// Default to root if unsure
	return ""
}

// saveFileToDatabase persists a generated file to the osa_module_files table
func (o *AppGenerationOrchestrator) saveFileToDatabase(ctx context.Context, appID uuid.UUID, filePath, content string) error {
	if o.queries == nil {
		return fmt.Errorf("queries not initialized")
	}

	// Calculate content hash
	hash := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hash[:])

	// Extract file info
	fileName := filepath.Base(filePath)
	fileType := inferFileType(filePath)
	language := inferLanguage(filePath)
	lineCount := int32(strings.Count(content, "\n") + 1)
	encoding := "utf-8"

	// Create pgtype.UUID for app_id
	pgAppID := pgtype.UUID{
		Bytes: appID,
		Valid: true,
	}

	// Create file record
	// Note: Using nil for dependencies and metadata to avoid Supabase simple_protocol JSON parsing issues
	// WorkflowID is now nullable (constraint dropped), so we use nil since we don't have a proper workflow context
	_, err := o.queries.CreateGeneratedFile(ctx, sqlc.CreateGeneratedFileParams{
		WorkflowID:       pgtype.UUID{Valid: false}, // Nullable - we don't have a workflow context from queue
		ModuleInstanceID: pgAppID,
		FilePath:         filePath,
		FileName:         fileName,
		FileType:         fileType,
		Language:         &language,
		Content:          content,
		ContentHash:      contentHash,
		FileSizeBytes:    int32(len(content)),
		LineCount:        &lineCount,
		Encoding:         &encoding,
		Purpose:          nil,
		Dependencies:     nil,
		Metadata:         nil,
	})

	if err != nil {
		return fmt.Errorf("create generated file: %w", err)
	}

	return nil
}

// inferFileType determines the file type from extension
func inferFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "source"
	case ".ts", ".tsx", ".js", ".jsx":
		return "source"
	case ".svelte", ".vue":
		return "component"
	case ".sql":
		return "database"
	case ".json", ".yaml", ".yml", ".toml":
		return "config"
	case ".md", ".txt":
		return "documentation"
	case ".html", ".css", ".scss":
		return "frontend"
	case ".sh", ".bash":
		return "script"
	default:
		return "other"
	}
}

// inferLanguage determines programming language from extension
func inferLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".svelte":
		return "svelte"
	case ".vue":
		return "vue"
	case ".sql":
		return "sql"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".html":
		return "html"
	case ".css", ".scss":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "bash"
	default:
		return "text"
	}
}

// enrichTaskDescriptions replaces the hardcoded generic agent task descriptions
// (from orchestrator.go) with context-specific descriptions derived from the
// user's actual request. Called after CreatePlan, before Execute.
func enrichTaskDescriptions(plan *appgen.Plan, req MultiAgentAppRequest) {
	// Build base context from request
	base := req.AppName
	if req.Description != "" {
		base += " — " + req.Description
	}

	featureList := ""
	if len(req.Features) > 0 {
		featureList = " Features required: " + strings.Join(req.Features, ", ") + "."
	}

	// Map agent types to context-rich description templates
	descFor := map[appgen.AgentType]string{
		appgen.AgentFrontend: fmt.Sprintf(
			"Build the Svelte/SvelteKit frontend for %s.%s Use TypeScript, Tailwind CSS, and Svelte stores. All pages must be responsive and connected to the Go backend API.",
			base, featureList,
		),
		appgen.AgentBackend: fmt.Sprintf(
			"Build the Go backend API for %s.%s Use Gin framework with PostgreSQL via sqlc. Include authentication middleware, proper error handling, and structured slog logging.",
			base, featureList,
		),
		appgen.AgentDatabase: fmt.Sprintf(
			"Create PostgreSQL migrations for %s.%s Write idempotent SQL migration files with proper indexes, foreign keys, and constraints. Do not invent seed/sample rows unless the user explicitly asks.",
			base, featureList,
		),
		appgen.AgentTest: fmt.Sprintf(
			"Write comprehensive tests for %s.%s Cover all API endpoints with integration tests, add unit tests for business logic, and ensure at least 80%% coverage.",
			base, featureList,
		),
	}

	for i := range plan.Tasks {
		if desc, ok := descFor[plan.Tasks[i].Type]; ok {
			plan.Tasks[i].Description = desc
		}
	}
}
