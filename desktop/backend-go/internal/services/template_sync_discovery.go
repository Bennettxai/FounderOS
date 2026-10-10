package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SyncTemplates syncs all YAML templates to the database.
// Strategy: YAML files are the source of truth.
func (s *TemplateSyncService) SyncTemplates(ctx context.Context) (*SyncResult, error) {
	s.logger.Info("starting template sync", "templates_dir", s.templatesDir)

	result := &SyncResult{
		Errors: []string{},
	}

	// Find all YAML files in the templates directory
	yamlFiles, err := s.findYAMLFiles(s.templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find YAML files: %w", err)
	}

	s.logger.Info("found YAML templates", "count", len(yamlFiles))

	// Process each YAML file
	for _, yamlFile := range yamlFiles {
		if err := s.syncTemplate(ctx, yamlFile, result); err != nil {
			errMsg := fmt.Sprintf("%s: %v", yamlFile, err)
			result.Errors = append(result.Errors, errMsg)
			s.logger.Error("failed to sync template", "file", yamlFile, "error", err)
		}
	}

	s.logger.Info("template sync completed",
		"inserted", result.Inserted,
		"updated", result.Updated,
		"skipped", result.Skipped,
		"errors", len(result.Errors),
	)

	return result, nil
}

// syncTemplate syncs a single YAML template to the database.
func (s *TemplateSyncService) syncTemplate(ctx context.Context, yamlPath string, result *SyncResult) error {
	// Load YAML template
	tmpl, err := s.loadYAMLTemplate(yamlPath)
	if err != nil {
		return fmt.Errorf("load YAML: %w", err)
	}

	s.logger.Debug("loaded YAML template", "name", tmpl.Name, "category", tmpl.Category)

	// Map YAML to DB structure
	dbTemplate := s.MapYAMLToDB(tmpl)

	// Check if template exists in DB
	exists, err := s.templateExists(ctx, tmpl.Name)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}

	if exists {
		// Update existing template
		if err := s.updateTemplate(ctx, dbTemplate); err != nil {
			return fmt.Errorf("update: %w", err)
		}
		result.Updated++
		s.logger.Info("updated template", "name", tmpl.Name)
	} else {
		// Insert new template
		if err := s.insertTemplate(ctx, dbTemplate); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
		result.Inserted++
		s.logger.Info("inserted template", "name", tmpl.Name)
	}

	return nil
}

// findYAMLFiles finds all YAML files in a directory recursively.
func (s *TemplateSyncService) findYAMLFiles(dir string) ([]string, error) {
	var yamlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" {
				yamlFiles = append(yamlFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return yamlFiles, nil
}

// loadYAMLTemplate loads a template from a YAML file.
func (s *TemplateSyncService) loadYAMLTemplate(yamlPath string) (*TemplateDefinition, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var tmpl TemplateDefinition
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("unmarshal YAML: %w", err)
	}

	return &tmpl, nil
}
