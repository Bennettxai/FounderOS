package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ModuleExportService handles exporting modules as ZIP files
type ModuleExportService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// ModuleManifest represents the module.json manifest structure
type ModuleManifest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Icon        string                 `json:"icon,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
	Manifest    map[string]interface{} `json:"manifest"`
	Config      map[string]interface{} `json:"config"`
	ExportedAt  string                 `json:"exported_at"`
}

func NewModuleExportService(pool *pgxpool.Pool, logger *slog.Logger) *ModuleExportService {
	return &ModuleExportService{
		pool:   pool,
		logger: logger,
	}
}

// ExportModule exports a module as a ZIP file
func (s *ModuleExportService) ExportModule(ctx context.Context, moduleID uuid.UUID) ([]byte, error) {
	// Get module data
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.GetModule(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get module: %w", err)
	}

	// Validate before export
	if err := s.ValidateBeforeExport(module); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generate manifest
	manifest := s.GenerateManifestJSON(module)

	// Create ZIP buffer
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add manifest file
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestFile, err := zipWriter.Create("module.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest file: %w", err)
	}
	if _, err := manifestFile.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Package files from manifest
	if err := s.PackageFiles(zipWriter, module); err != nil {
		return nil, fmt.Errorf("failed to package files: %w", err)
	}

	// Close ZIP
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP: %w", err)
	}

	s.logger.Info("module exported", "module_id", moduleID, "size_bytes", buf.Len())
	return buf.Bytes(), nil
}

// GenerateManifestJSON creates the module.json manifest
func (s *ModuleExportService) GenerateManifestJSON(module *CustomModule) ModuleManifest {
	desc := ""
	if module.Description != nil {
		desc = *module.Description
	}
	icon := ""
	if module.Icon != nil {
		icon = *module.Icon
	}
	return ModuleManifest{
		ID:          module.ID.String(),
		Name:        module.Name,
		Slug:        module.Slug,
		Version:     module.Version,
		Description: desc,
		Category:    module.Category,
		Icon:        icon,
		Tags:        module.Tags,
		Keywords:    module.Keywords,
		Manifest:    module.Manifest,
		Config:      module.Config,
		ExportedAt:  module.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// PackageFiles adds module files to the ZIP
func (s *ModuleExportService) PackageFiles(zipWriter *zip.Writer, module *CustomModule) error {
	// Check if manifest contains files
	if module.Manifest == nil {
		return nil
	}

	files, ok := module.Manifest["files"].([]interface{})
	if !ok || len(files) == 0 {
		s.logger.Info("no files to package", "module_id", module.ID)
		return nil
	}

	// Create files directory
	filesDir := "files/"

	for i, fileItem := range files {
		fileMap, ok := fileItem.(map[string]interface{})
		if !ok {
			s.logger.Warn("invalid file entry", "index", i)
			continue
		}

		path, _ := fileMap["path"].(string)
		contentBase64, _ := fileMap["content"].(string)

		if path == "" || contentBase64 == "" {
			s.logger.Warn("missing path or content", "index", i)
			continue
		}

		// Decode base64 content
		content, err := base64.StdEncoding.DecodeString(contentBase64)
		if err != nil {
			s.logger.Warn("failed to decode file content", "path", path, "error", err)
			continue
		}

		// Add file to ZIP
		filePath := filesDir + path
		fileWriter, err := zipWriter.Create(filePath)
		if err != nil {
			s.logger.Warn("failed to create file in ZIP", "path", filePath, "error", err)
			continue
		}

		if _, err := fileWriter.Write(content); err != nil {
			s.logger.Warn("failed to write file", "path", filePath, "error", err)
			continue
		}

		s.logger.Debug("packaged file", "path", filePath, "size", len(content))
	}

	return nil
}

// ValidateBeforeExport validates module data before export
func (s *ModuleExportService) ValidateBeforeExport(module *CustomModule) error {
	if module.Name == "" {
		return fmt.Errorf("module name is required")
	}

	if module.Slug == "" {
		return fmt.Errorf("module slug is required")
	}

	if module.Version == "" {
		return fmt.Errorf("module version is required")
	}

	if module.Manifest == nil {
		return fmt.Errorf("module manifest is required")
	}

	// Validate manifest has required fields
	if _, ok := module.Manifest["actions"]; !ok {
		return fmt.Errorf("manifest must contain 'actions' field")
	}

	actions, ok := module.Manifest["actions"].([]interface{})
	if !ok {
		return fmt.Errorf("manifest 'actions' must be an array")
	}

	if len(actions) == 0 {
		return fmt.Errorf("manifest must contain at least one action")
	}

	s.logger.Info("module validated for export", "module_id", module.ID, "actions_count", len(actions))
	return nil
}

// GetExportStats returns statistics about exported module
func (s *ModuleExportService) GetExportStats(module *CustomModule) map[string]interface{} {
	stats := map[string]interface{}{
		"module_id": module.ID.String(),
		"name":      module.Name,
		"version":   module.Version,
		"category":  module.Category,
	}

	// Count actions
	if module.Manifest != nil {
		if actions, ok := module.Manifest["actions"].([]interface{}); ok {
			stats["actions_count"] = len(actions)
		}

		// Count files
		if files, ok := module.Manifest["files"].([]interface{}); ok {
			stats["files_count"] = len(files)

			// Calculate total size
			totalSize := 0
			for _, fileItem := range files {
				if fileMap, ok := fileItem.(map[string]interface{}); ok {
					if contentBase64, ok := fileMap["content"].(string); ok {
						content, err := base64.StdEncoding.DecodeString(contentBase64)
						if err == nil {
							totalSize += len(content)
						}
					}
				}
			}
			stats["total_size_bytes"] = totalSize
		}
	}

	return stats
}

// ExportModuleWithMetadata exports module with additional metadata
func (s *ModuleExportService) ExportModuleWithMetadata(ctx context.Context, moduleID uuid.UUID) ([]byte, map[string]interface{}, error) {
	// Export module
	zipData, err := s.ExportModule(ctx, moduleID)
	if err != nil {
		return nil, nil, err
	}

	// Get module for stats
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.GetModule(ctx, moduleID)
	if err != nil {
		return zipData, nil, nil
	}

	// Generate stats
	stats := s.GetExportStats(module)
	stats["export_size_bytes"] = len(zipData)

	return zipData, stats, nil
}

// GenerateReadme generates a README.md for the module
func (s *ModuleExportService) GenerateReadme(module *CustomModule) string {
	readme := fmt.Sprintf("# %s\n\n", module.Name)
	readme += fmt.Sprintf("**Version:** %s\n", module.Version)
	readme += fmt.Sprintf("**Category:** %s\n\n", module.Category)
	if module.Description != nil {
		readme += fmt.Sprintf("## Description\n\n%s\n\n", *module.Description)
	}

	// List actions
	if module.Manifest != nil {
		if actions, ok := module.Manifest["actions"].([]interface{}); ok && len(actions) > 0 {
			readme += "## Actions\n\n"
			for _, actionItem := range actions {
				if actionMap, ok := actionItem.(map[string]interface{}); ok {
					name, _ := actionMap["name"].(string)
					actionType, _ := actionMap["type"].(string)
					description, _ := actionMap["description"].(string)

					readme += fmt.Sprintf("### %s\n", name)
					readme += fmt.Sprintf("- **Type:** %s\n", actionType)
					if description != "" {
						readme += fmt.Sprintf("- **Description:** %s\n", description)
					}
					readme += "\n"
				}
			}
		}
	}

	// List tags
	if len(module.Tags) > 0 {
		readme += "## Tags\n\n"
		for _, tag := range module.Tags {
			readme += fmt.Sprintf("- %s\n", tag)
		}
		readme += "\n"
	}

	return readme
}

// ExportWithReadme exports module with auto-generated README
func (s *ModuleExportService) ExportWithReadme(ctx context.Context, moduleID uuid.UUID) ([]byte, error) {
	// Get module
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.GetModule(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get module: %w", err)
	}

	// Validate
	if err := s.ValidateBeforeExport(module); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generate manifest
	manifest := s.GenerateManifestJSON(module)
	readme := s.GenerateReadme(module)

	// Create ZIP
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add manifest
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	manifestFile, err := zipWriter.Create("module.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest file: %w", err)
	}
	if _, err := manifestFile.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Add README
	readmeFile, err := zipWriter.Create("README.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create README file: %w", err)
	}
	if _, err := readmeFile.Write([]byte(readme)); err != nil {
		return nil, fmt.Errorf("failed to write README: %w", err)
	}

	// Package files
	if err := s.PackageFiles(zipWriter, module); err != nil {
		return nil, fmt.Errorf("failed to package files: %w", err)
	}

	// Close ZIP
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP: %w", err)
	}

	s.logger.Info("module exported with readme", "module_id", moduleID, "size_bytes", buf.Len())
	return buf.Bytes(), nil
}
