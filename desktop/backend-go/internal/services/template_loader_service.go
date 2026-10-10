package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"gopkg.in/yaml.v3"
)

// TemplateLoaderService handles loading and rendering YAML templates
type TemplateLoaderService struct {
	templatesDir string
	cache        map[string]*TemplateDefinition
	mu           sync.RWMutex
	logger       *slog.Logger
}

// NewTemplateLoaderService creates a new template loader service
func NewTemplateLoaderService(templatesDir string) *TemplateLoaderService {
	return &TemplateLoaderService{
		templatesDir: templatesDir,
		cache:        make(map[string]*TemplateDefinition),
		logger:       slog.Default().With("service", "template_loader"),
	}
}

// LoadTemplate loads and parses a YAML template by name
func (s *TemplateLoaderService) LoadTemplate(name string) (*TemplateDefinition, error) {
	// Check cache first (read lock)
	s.mu.RLock()
	if cached, ok := s.cache[name]; ok {
		s.mu.RUnlock()
		s.logger.Debug("template loaded from cache", "name", name)
		return cached, nil
	}
	s.mu.RUnlock()

	// Not in cache, load from file (write lock)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache after acquiring write lock
	if cached, ok := s.cache[name]; ok {
		s.logger.Debug("template loaded from cache after lock", "name", name)
		return cached, nil
	}

	// Read YAML file
	filename := filepath.Join(s.templatesDir, name+".yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		s.logger.Error("failed to read template file", "name", name, "error", err)
		return nil, fmt.Errorf("failed to read template file %s: %w", filename, err)
	}

	// Parse YAML
	var tmpl TemplateDefinition
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		s.logger.Error("failed to parse template YAML", "name", name, "error", err)
		return nil, fmt.Errorf("failed to parse template YAML %s: %w", filename, err)
	}

	// Store in cache
	s.cache[name] = &tmpl
	s.logger.Info("template loaded successfully", "name", name)

	return &tmpl, nil
}

// ListTemplates returns all available templates
func (s *TemplateLoaderService) ListTemplates() ([]*TemplateDefinition, error) {
	// Read directory
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		s.logger.Error("failed to read templates directory", "error", err)
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []*TemplateDefinition

	// Load each YAML file
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		// Get template name (without .yaml extension)
		name := entry.Name()[:len(entry.Name())-5]

		// Load template
		tmpl, err := s.LoadTemplate(name)
		if err != nil {
			s.logger.Warn("failed to load template, skipping", "name", name, "error", err)
			continue
		}

		templates = append(templates, tmpl)
	}

	s.logger.Info("listed templates", "count", len(templates))
	return templates, nil
}

// RenderTemplate renders a template with the provided variables
func (s *TemplateLoaderService) RenderTemplate(name string, variables map[string]interface{}) (string, error) {
	// Load template definition
	tmplDef, err := s.LoadTemplate(name)
	if err != nil {
		return "", fmt.Errorf("failed to load template: %w", err)
	}

	// Validate variables
	if err := s.ValidateVariables(tmplDef, variables); err != nil {
		return "", fmt.Errorf("variable validation failed: %w", err)
	}

	// Apply defaults for missing optional variables
	variablesWithDefaults := s.applyDefaults(tmplDef, variables)

	// Parse Go template
	tmpl, err := template.New(name).Parse(tmplDef.Template)
	if err != nil {
		s.logger.Error("failed to parse template", "name", name, "error", err)
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var result []byte
	buf := &bytesBuffer{buf: &result}
	if err := tmpl.Execute(buf, variablesWithDefaults); err != nil {
		s.logger.Error("failed to execute template", "name", name, "error", err)
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	s.logger.Info("template rendered successfully", "name", name, "output_size", len(result))
	return string(result), nil
}

// ValidateVariables validates that all required variables are present and have correct types
func (s *TemplateLoaderService) ValidateVariables(tmplDef *TemplateDefinition, variables map[string]interface{}) error {
	for _, varDef := range tmplDef.Variables {
		value, exists := variables[varDef.Name]

		// Check required variables
		if varDef.Required && !exists {
			return fmt.Errorf("required variable '%s' is missing", varDef.Name)
		}

		// Skip validation if variable not provided and not required
		if !exists {
			continue
		}

		// Validate type
		if err := ValidateVariableType(value, varDef.Type); err != nil {
			return fmt.Errorf("variable '%s': %w", varDef.Name, err)
		}
	}

	return nil
}

// applyDefaults applies default values for missing optional variables
func (s *TemplateLoaderService) applyDefaults(tmplDef *TemplateDefinition, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy provided variables
	for k, v := range variables {
		result[k] = v
	}

	// Apply defaults for missing optional variables
	for _, varDef := range tmplDef.Variables {
		if _, exists := result[varDef.Name]; !exists && varDef.Default != nil {
			result[varDef.Name] = varDef.Default
		}
	}

	return result
}

// ClearCache clears the template cache
func (s *TemplateLoaderService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]*TemplateDefinition)
	s.logger.Info("template cache cleared")
}

// bytesBuffer is a helper to capture template execution output
type bytesBuffer struct {
	buf *[]byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	*b.buf = append(*b.buf, p...)
	return len(p), nil
}
