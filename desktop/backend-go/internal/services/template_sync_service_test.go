package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSyncService(t *testing.T) (*TemplateSyncService, *pgxpool.Pool, func()) {
	// Get test database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Connect to test database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Find templates directory
	templatesDir := findTestTemplatesDir(t)

	// Create service
	service := NewTemplateSyncService(pool, logger, templatesDir)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		_, _ = pool.Exec(ctx, "DELETE FROM app_templates WHERE yaml_template_name IS NOT NULL")
		pool.Close()
	}

	return service, pool, cleanup
}

func findTestTemplatesDir(t *testing.T) string {
	// Try to find templates directory relative to test file
	possiblePaths := []string{
		"../prompts/templates/osa",
		"../../internal/prompts/templates/osa",
		"./test_templates", // For mocked tests
	}

	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath
		}
	}

	// If real templates not found, create a temporary test directory
	tempDir := t.TempDir()
	testTemplateDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(testTemplateDir, 0755)
	require.NoError(t, err)

	// Create a simple test template
	testYAML := `name: "test-template"
display_name: "Test Template"
description: "A test template for unit testing"
category: "testing"
version: "1.0.0"
tags:
  - "test"
  - "unit-test"
variables:
  - name: "TestVar"
    type: "string"
    required: true
    description: "A test variable"
template: |
  # Test Template
  This is a test template for {{.TestVar}}.
`

	testFile := filepath.Join(testTemplateDir, "test-template.yaml")
	err = os.WriteFile(testFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	return testTemplateDir
}

func TestTemplateSyncService_MapYAMLToDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		name     string
		yaml     *TemplateDefinition
		validate func(t *testing.T, result *DBTemplate)
	}{
		{
			name: "CRM app generation",
			yaml: &TemplateDefinition{
				Name:        "crm-app-generation",
				DisplayName: "CRM Application Generation",
				Description: "Generate a full-stack CRM application",
				Category:    "app-generation",
				Version:     "1.0.0",
				Tags:        []string{"crm", "full-stack", "business"},
				Variables: []TemplateVariable{
					{
						Name:        "AppType",
						Type:        "string",
						Required:    true,
						Description: "Type of application",
					},
				},
				Template: "# CRM Generation\n\nGenerate CRM with Go + Gin backend and SvelteKit frontend.",
			},
			validate: func(t *testing.T, result *DBTemplate) {
				assert.Equal(t, "crm-app-generation", result.TemplateName)
				assert.Equal(t, "CRM Application Generation", result.DisplayName)
				assert.Equal(t, "app-generation", result.Category)
				assert.Equal(t, "users", result.IconType)
				assert.Equal(t, "full-stack", result.ScaffoldType)
				assert.Greater(t, result.PriorityScore, 80)
				assert.Contains(t, result.TargetBusinessTypes, "saas")
				assert.Contains(t, result.TargetBusinessTypes, "enterprise")
				assert.NotEmpty(t, result.TemplateVariables)
			},
		},
		{
			name: "Dashboard creation",
			yaml: &TemplateDefinition{
				Name:        "dashboard-creation",
				DisplayName: "Analytics Dashboard",
				Description: "Create interactive dashboards",
				Category:    "data-visualization",
				Version:     "1.0.0",
				Tags:        []string{"dashboard", "analytics", "charts"},
				Variables: []TemplateVariable{
					{
						Name:        "DashboardPurpose",
						Type:        "string",
						Required:    true,
						Description: "Purpose of the dashboard",
					},
					{
						Name:        "RefreshInterval",
						Type:        "string",
						Required:    false,
						Default:     "5 minutes",
						Description: "Data refresh interval",
					},
				},
				Template: "# Dashboard\n\nCreate dashboard with Chart.js",
			},
			validate: func(t *testing.T, result *DBTemplate) {
				assert.Equal(t, "dashboard-creation", result.TemplateName)
				assert.Equal(t, "data-visualization", result.Category)
				assert.Equal(t, "chart", result.IconType)
				assert.Equal(t, "svelte", result.ScaffoldType)
				assert.Greater(t, result.PriorityScore, 85)
				assert.Contains(t, result.TargetChallenges, "analytics")
				assert.Contains(t, result.OptionalFeatures, "real_time_updates")
			},
		},
		{
			name: "Bug fix template",
			yaml: &TemplateDefinition{
				Name:        "bug-fix",
				DisplayName: "Bug Fix",
				Description: "Fix bugs with root cause analysis",
				Category:    "maintenance",
				Version:     "1.0.0",
				Tags:        []string{"bug", "debugging", "maintenance"},
				Variables: []TemplateVariable{
					{
						Name:        "BugDescription",
						Type:        "string",
						Required:    true,
						Description: "Description of the bug",
					},
				},
				Template: "# Bug Fix\n\nFollow systematic debugging",
			},
			validate: func(t *testing.T, result *DBTemplate) {
				assert.Equal(t, "bug-fix", result.TemplateName)
				assert.Equal(t, "maintenance", result.Category)
				assert.Equal(t, "wrench", result.IconType)
				assert.Equal(t, "go", result.ScaffoldType)
				assert.Contains(t, result.TargetChallenges, "bug_fixing")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.MapYAMLToDB(tt.yaml)
			require.NotNil(t, result)
			tt.validate(t, result)
		})
	}
}

func TestTemplateSyncService_CategoryToIcon(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		category     string
		expectedIcon string
	}{
		{"app-generation", "users"},
		{"data-visualization", "chart"},
		{"maintenance", "wrench"},
		{"feature", "plus"},
		{"operations", "server"},
		{"marketing", "globe"},
		{"unknown", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			icon := service.categoryToIcon(tt.category)
			assert.Equal(t, tt.expectedIcon, icon)
		})
	}
}

func TestTemplateSyncService_CalculatePriorityScore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		name          string
		category      string
		tags          []string
		minScore      int
		maxScore      int
	}{
		{
			name:     "App generation with full-stack",
			category: "app-generation",
			tags:     []string{"full-stack", "crm"},
			minScore: 85,
			maxScore: 100,
		},
		{
			name:     "Data visualization",
			category: "data-visualization",
			tags:     []string{"dashboard"},
			minScore: 90,
			maxScore: 100,
		},
		{
			name:     "Bug fix (lower priority)",
			category: "maintenance",
			tags:     []string{"bug"},
			minScore: 60,
			maxScore: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculatePriorityScore(tt.category, tt.tags)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestTemplateSyncService_SyncTemplates_Integration(t *testing.T) {
	service, pool, cleanup := setupTestSyncService(t)
	defer cleanup()

	ctx := context.Background()

	// Run sync
	result, err := service.SyncTemplates(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify results
	assert.GreaterOrEqual(t, result.Inserted+result.Updated, 1, "Should sync at least one template")
	assert.LessOrEqual(t, len(result.Errors), 0, "Should have no errors")

	// Verify templates exist in database
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_templates WHERE yaml_template_name IS NOT NULL").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "Should have inserted templates")
}

func TestTemplateSyncService_GetTemplateByName_Integration(t *testing.T) {
	service, pool, cleanup := setupTestSyncService(t)
	defer cleanup()

	ctx := context.Background()

	// First sync templates
	_, err := service.SyncTemplates(ctx)
	require.NoError(t, err)

	// Get a template by name
	// Note: Actual template name depends on what's in test directory
	var templateName string
	err = pool.QueryRow(ctx, "SELECT template_name FROM app_templates WHERE yaml_template_name IS NOT NULL LIMIT 1").Scan(&templateName)
	if err != nil {
		t.Skip("No templates found, skipping retrieval test")
	}

	template, err := service.GetTemplateByName(ctx, templateName)
	require.NoError(t, err)
	require.NotNil(t, template)

	// Verify template fields
	assert.NotEmpty(t, template.TemplateName)
	assert.NotEmpty(t, template.DisplayName)
	assert.NotEmpty(t, template.Category)
	assert.NotEmpty(t, template.YAMLTemplateName)
}

func TestTemplateSyncService_IdempotentSync_Integration(t *testing.T) {
	service, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	ctx := context.Background()

	// First sync
	_, err := service.SyncTemplates(ctx)
	require.NoError(t, err)

	// Second sync (should update, not insert duplicates)
	result2, err := service.SyncTemplates(ctx)
	require.NoError(t, err)

	// Second sync should update existing templates
	assert.Equal(t, 0, result2.Inserted, "Second sync should not insert new templates")
	assert.Greater(t, result2.Updated, 0, "Second sync should update existing templates")
}

func TestTemplateSyncService_ExtractBusinessTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		name             string
		tags             []string
		category         string
		expectedContains []string
	}{
		{
			name:             "CRM app",
			tags:             []string{"crm", "business"},
			category:         "app-generation",
			expectedContains: []string{"saas", "startup", "small_business"},
		},
		{
			name:             "Dashboard",
			tags:             []string{"dashboard"},
			category:         "data-visualization",
			expectedContains: []string{"saas", "enterprise"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractBusinessTypes(tt.tags, tt.category)
			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestTemplateSyncService_ExtractChallenges(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		name             string
		tags             []string
		category         string
		expectedContains []string
	}{
		{
			name:             "App generation",
			tags:             []string{"full-stack"},
			category:         "app-generation",
			expectedContains: []string{"rapid_prototyping", "scalability"},
		},
		{
			name:             "Bug fix",
			tags:             []string{"bug", "debugging"},
			category:         "maintenance",
			expectedContains: []string{"bug_fixing"},
		},
		{
			name:             "Analytics",
			tags:             []string{"analytics", "charts"},
			category:         "data-visualization",
			expectedContains: []string{"analytics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractChallenges(tt.tags, tt.category)
			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestTemplateSyncService_DetermineScaffoldType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	service := NewTemplateSyncService(nil, logger, "")

	tests := []struct {
		name         string
		tags         []string
		category     string
		expectedType string
	}{
		{
			name:         "Full-stack from tags",
			tags:         []string{"full-stack", "crm"},
			category:     "app-generation",
			expectedType: "full-stack",
		},
		{
			name:         "App generation category",
			tags:         []string{"crm"},
			category:     "app-generation",
			expectedType: "full-stack",
		},
		{
			name:         "Dashboard",
			tags:         []string{"dashboard"},
			category:     "data-visualization",
			expectedType: "svelte",
		},
		{
			name:         "Maintenance",
			tags:         []string{"bug"},
			category:     "maintenance",
			expectedType: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineScaffoldType(tt.tags, tt.category)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}
