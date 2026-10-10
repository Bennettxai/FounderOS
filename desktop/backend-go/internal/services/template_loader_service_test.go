package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestTemplatesDir creates a temporary directory with test templates
func setupTestTemplatesDir(t *testing.T) string {
	tmpDir := t.TempDir()

	// Create a minimal test template
	testTemplate := `name: "test-template"
display_name: "Test Template"
description: "A test template for unit testing"
category: "test"
version: "1.0.0"
tags:
  - "test"
  - "example"

variables:
  - name: "RequiredVar"
    type: "string"
    required: true
    description: "A required variable"

  - name: "OptionalVar"
    type: "string"
    required: false
    default: "default-value"
    description: "An optional variable with default"

  - name: "ArrayVar"
    type: "array"
    required: false
    default: []
    description: "An array variable"

  - name: "ObjectVar"
    type: "object"
    required: false
    description: "An object variable"

template: |
  # Test Template
  Required: {{.RequiredVar}}
  Optional: {{.OptionalVar}}
`

	// Write test template to file
	err := os.WriteFile(filepath.Join(tmpDir, "test-template.yaml"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create another template for listing tests
	anotherTemplate := `name: "another-template"
display_name: "Another Template"
description: "Another test template"
category: "test"
version: "1.0.0"
tags:
  - "test"

variables:
  - name: "Name"
    type: "string"
    required: true
    description: "Name variable"

template: |
  Hello {{.Name}}!
`

	err = os.WriteFile(filepath.Join(tmpDir, "another-template.yaml"), []byte(anotherTemplate), 0644)
	require.NoError(t, err)

	return tmpDir
}

func TestNewTemplateLoaderService(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	assert.NotNil(t, service)
	assert.Equal(t, tmpDir, service.templatesDir)
	assert.NotNil(t, service.cache)
	assert.NotNil(t, service.logger)
}

func TestLoadTemplate_Success(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	tmpl, err := service.LoadTemplate("test-template")

	require.NoError(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, "test-template", tmpl.Name)
	assert.Equal(t, "Test Template", tmpl.DisplayName)
	assert.Equal(t, "A test template for unit testing", tmpl.Description)
	assert.Equal(t, "test", tmpl.Category)
	assert.Equal(t, "1.0.0", tmpl.Version)
	assert.Equal(t, []string{"test", "example"}, tmpl.Tags)
	assert.Len(t, tmpl.Variables, 4)
	assert.Contains(t, tmpl.Template, "Required: {{.RequiredVar}}")
}

func TestLoadTemplate_NotFound(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	tmpl, err := service.LoadTemplate("non-existent")

	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "failed to read template file")
}

func TestLoadTemplate_Caching(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	// First load - should read from file
	tmpl1, err := service.LoadTemplate("test-template")
	require.NoError(t, err)
	assert.NotNil(t, tmpl1)

	// Second load - should use cache
	tmpl2, err := service.LoadTemplate("test-template")
	require.NoError(t, err)
	assert.NotNil(t, tmpl2)

	// Should be the same pointer (from cache)
	assert.Same(t, tmpl1, tmpl2)
}

func TestListTemplates(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	templates, err := service.ListTemplates()

	require.NoError(t, err)
	assert.Len(t, templates, 2)

	// Check that both templates are present
	names := make(map[string]bool)
	for _, tmpl := range templates {
		names[tmpl.Name] = true
	}
	assert.True(t, names["test-template"])
	assert.True(t, names["another-template"])
}

func TestRenderTemplate_Success(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	variables := map[string]interface{}{
		"RequiredVar": "test-value",
	}

	result, err := service.RenderTemplate("test-template", variables)

	require.NoError(t, err)
	assert.Contains(t, result, "Required: test-value")
	assert.Contains(t, result, "Optional: default-value") // Should use default
}

func TestRenderTemplate_WithCustomOptional(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	variables := map[string]interface{}{
		"RequiredVar": "test-value",
		"OptionalVar": "custom-value",
	}

	result, err := service.RenderTemplate("test-template", variables)

	require.NoError(t, err)
	assert.Contains(t, result, "Required: test-value")
	assert.Contains(t, result, "Optional: custom-value")
}

func TestRenderTemplate_MissingRequired(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	variables := map[string]interface{}{
		"OptionalVar": "some-value",
	}

	result, err := service.RenderTemplate("test-template", variables)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "required variable 'RequiredVar' is missing")
}

func TestValidateVariables_AllValid(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	tmpl, err := service.LoadTemplate("test-template")
	require.NoError(t, err)

	variables := map[string]interface{}{
		"RequiredVar": "test",
		"OptionalVar": "optional",
		"ArrayVar":    []string{"a", "b"},
		"ObjectVar":   map[string]interface{}{"key": "value"},
	}

	err = service.ValidateVariables(tmpl, variables)
	assert.NoError(t, err)
}

func TestValidateVariables_InvalidType(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	tmpl, err := service.LoadTemplate("test-template")
	require.NoError(t, err)

	variables := map[string]interface{}{
		"RequiredVar": "test",
		"ArrayVar":    "not-an-array", // Should be array
	}

	err = service.ValidateVariables(tmpl, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ArrayVar")
	assert.Contains(t, err.Error(), "expected type 'array'")
}

func TestTemplateLoaderValidateVariables_MissingRequired(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	tmpl, err := service.LoadTemplate("test-template")
	require.NoError(t, err)

	variables := map[string]interface{}{
		"OptionalVar": "test",
	}

	err = service.ValidateVariables(tmpl, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required variable 'RequiredVar' is missing")
}

func TestTemplateLoaderClearCache(t *testing.T) {
	tmpDir := setupTestTemplatesDir(t)
	service := NewTemplateLoaderService(tmpDir)

	// Load template (populates cache)
	tmpl1, err := service.LoadTemplate("test-template")
	require.NoError(t, err)
	assert.NotNil(t, tmpl1)

	// Clear cache
	service.ClearCache()

	// Load again - should read from file (different pointer)
	tmpl2, err := service.LoadTemplate("test-template")
	require.NoError(t, err)
	assert.NotNil(t, tmpl2)

	// Should NOT be the same pointer
	assert.NotSame(t, tmpl1, tmpl2)
}

// Integration test with real OSA templates
func TestLoadRealOSATemplates(t *testing.T) {
	// Skip if OSA templates directory doesn't exist
	templatesDir := GetTemplatesDirectory()
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("OSA templates directory not found, skipping integration test")
	}

	service := NewTemplateLoaderService(templatesDir)

	// Test loading each known template
	templates := []string{
		"bug-fix",
		"crm-app-generation",
		"dashboard-creation",
		"data-pipeline-creation",
		"feature-addition",
	}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl, err := service.LoadTemplate(name)
			require.NoError(t, err, "Failed to load template: %s", name)
			assert.Equal(t, name, tmpl.Name)
			assert.NotEmpty(t, tmpl.DisplayName)
			assert.NotEmpty(t, tmpl.Description)
			assert.NotEmpty(t, tmpl.Template)
		})
	}
}

// Integration test: List all real OSA templates
func TestListRealOSATemplates(t *testing.T) {
	templatesDir := GetTemplatesDirectory()
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("OSA templates directory not found, skipping integration test")
	}

	service := NewTemplateLoaderService(templatesDir)

	templates, err := service.ListTemplates()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(templates), 5, "Should have at least 5 OSA templates")

	// Verify each template has required fields
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.Name)
		assert.NotEmpty(t, tmpl.DisplayName)
		assert.NotEmpty(t, tmpl.Description)
		assert.NotEmpty(t, tmpl.Category)
		assert.NotEmpty(t, tmpl.Version)
	}
}
