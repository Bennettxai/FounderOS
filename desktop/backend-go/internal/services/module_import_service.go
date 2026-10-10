package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ModuleImportService handles importing modules from ZIP files
type ModuleImportService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// ImportedModule represents the parsed module from ZIP
type ImportedModule struct {
	Manifest ModuleManifest
	Files    map[string][]byte
	Readme   string
}

func NewModuleImportService(pool *pgxpool.Pool, logger *slog.Logger) *ModuleImportService {
	return &ModuleImportService{
		pool:   pool,
		logger: logger,
	}
}

// ImportModule imports a module from ZIP data
func (s *ModuleImportService) ImportModule(
	ctx context.Context,
	zipData []byte,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (*CustomModule, error) {
	// Parse ZIP
	imported, err := s.ParseZIP(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ZIP: %w", err)
	}

	// Validate manifest
	if err := s.ValidateManifest(&imported.Manifest); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve dependencies
	if err := s.ResolveDependencies(ctx, &imported.Manifest); err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Create module
	module, err := s.CreateModule(ctx, workspaceID, userID, imported)
	if err != nil {
		return nil, fmt.Errorf("failed to create module: %w", err)
	}

	s.logger.Info("module imported successfully", "module_id", module.ID, "name", module.Name)
	return module, nil
}

// ParseZIP extracts and parses the ZIP file
func (s *ModuleImportService) ParseZIP(zipData []byte) (*ImportedModule, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}

	imported := &ImportedModule{
		Files: make(map[string][]byte),
	}

	var manifestFound bool

	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Open file
		rc, err := file.Open()
		if err != nil {
			s.logger.Warn("failed to open file in ZIP", "path", file.Name, "error", err)
			continue
		}

		// Read content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			s.logger.Warn("failed to read file", "path", file.Name, "error", err)
			continue
		}

		// Process based on filename
		switch file.Name {
		case "module.json":
			// Parse manifest
			if err := json.Unmarshal(content, &imported.Manifest); err != nil {
				return nil, fmt.Errorf("failed to parse module.json: %w", err)
			}
			manifestFound = true

		case "README.md":
			imported.Readme = string(content)

		default:
			// Store other files
			if strings.HasPrefix(file.Name, "files/") {
				// Strip "files/" prefix
				path := strings.TrimPrefix(file.Name, "files/")
				imported.Files[path] = content
				s.logger.Debug("extracted file", "path", path, "size", len(content))
			}
		}
	}

	if !manifestFound {
		return nil, fmt.Errorf("module.json not found in ZIP")
	}

	// Encode files as base64 in manifest
	if len(imported.Files) > 0 {
		files := make([]interface{}, 0, len(imported.Files))
		for path, content := range imported.Files {
			files = append(files, map[string]interface{}{
				"path":    path,
				"content": base64.StdEncoding.EncodeToString(content),
				"size":    len(content),
			})
		}

		if imported.Manifest.Manifest == nil {
			imported.Manifest.Manifest = make(map[string]interface{})
		}
		imported.Manifest.Manifest["files"] = files
	}

	s.logger.Info("ZIP parsed", "files_count", len(imported.Files))
	return imported, nil
}

// ValidateManifest validates the module manifest
func (s *ModuleImportService) ValidateManifest(manifest *ModuleManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("module name is required")
	}

	if manifest.Slug == "" {
		return fmt.Errorf("module slug is required")
	}

	if manifest.Version == "" {
		return fmt.Errorf("module version is required")
	}

	if manifest.Manifest == nil {
		return fmt.Errorf("manifest data is required")
	}

	// Validate actions
	actions, ok := manifest.Manifest["actions"]
	if !ok {
		return fmt.Errorf("manifest must contain 'actions' field")
	}

	actionsList, ok := actions.([]interface{})
	if !ok {
		return fmt.Errorf("manifest 'actions' must be an array")
	}

	if len(actionsList) == 0 {
		return fmt.Errorf("manifest must contain at least one action")
	}

	// Validate each action
	for i, actionItem := range actionsList {
		actionMap, ok := actionItem.(map[string]interface{})
		if !ok {
			return fmt.Errorf("action %d must be an object", i)
		}

		if _, ok := actionMap["name"]; !ok {
			return fmt.Errorf("action %d must have a 'name' field", i)
		}

		if _, ok := actionMap["type"]; !ok {
			return fmt.Errorf("action %d must have a 'type' field", i)
		}
	}

	s.logger.Info("manifest validated", "name", manifest.Name, "actions_count", len(actionsList))
	return nil
}

// ResolveDependencies checks if module dependencies are available
func (s *ModuleImportService) ResolveDependencies(ctx context.Context, manifest *ModuleManifest) error {
	// Check if manifest has dependencies
	if manifest.Manifest == nil {
		return nil
	}

	dependencies, ok := manifest.Manifest["dependencies"]
	if !ok {
		return nil
	}

	depList, ok := dependencies.([]interface{})
	if !ok || len(depList) == 0 {
		return nil
	}

	s.logger.Info("checking dependencies", "count", len(depList))

	// Check each dependency
	for i, depItem := range depList {
		depMap, ok := depItem.(map[string]interface{})
		if !ok {
			s.logger.Warn("invalid dependency format", "index", i)
			continue
		}

		depName, _ := depMap["name"].(string)
		depVersion, _ := depMap["version"].(string)
		required, _ := depMap["required"].(bool)

		if depName == "" {
			continue
		}

		// Check if dependency exists in database
		var exists bool
		err := s.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM custom_modules
				WHERE slug = $1 AND version = $2 AND is_published = TRUE
			)
		`, depName, depVersion).Scan(&exists)

		if err != nil {
			s.logger.Warn("failed to check dependency", "name", depName, "error", err)
			continue
		}

		if !exists {
			if required {
				return fmt.Errorf("required dependency not found: %s@%s", depName, depVersion)
			}
			s.logger.Warn("optional dependency not found", "name", depName, "version", depVersion)
		} else {
			s.logger.Info("dependency found", "name", depName, "version", depVersion)
		}
	}

	return nil
}

// CreateModule creates the module in database
func (s *ModuleImportService) CreateModule(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	imported *ImportedModule,
) (*CustomModule, error) {
	// Prepare module data
	req := CreateModuleRequest{
		Name:        imported.Manifest.Name,
		Description: imported.Manifest.Description,
		Category:    imported.Manifest.Category,
		Manifest:    imported.Manifest.Manifest,
		Config:      imported.Manifest.Config,
		Icon:        imported.Manifest.Icon,
		Tags:        imported.Manifest.Tags,
		Keywords:    imported.Manifest.Keywords,
	}

	// Create module via service
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.CreateModule(ctx, workspaceID, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create module: %w", err)
	}

	// Update version if different from default
	if imported.Manifest.Version != "" && imported.Manifest.Version != "0.0.1" {
		_, err = s.pool.Exec(ctx, `
			UPDATE custom_modules SET version = $1 WHERE id = $2
		`, imported.Manifest.Version, module.ID)
		if err != nil {
			s.logger.Warn("failed to update module version", "module_id", module.ID, "error", err)
		} else {
			module.Version = imported.Manifest.Version
		}
	}

	s.logger.Info("module created from import", "module_id", module.ID, "name", module.Name)
	return module, nil
}

// GetImportStats returns statistics about imported module
func (s *ModuleImportService) GetImportStats(imported *ImportedModule) map[string]interface{} {
	stats := map[string]interface{}{
		"name":        imported.Manifest.Name,
		"version":     imported.Manifest.Version,
		"category":    imported.Manifest.Category,
		"files_count": len(imported.Files),
	}

	// Count actions
	if imported.Manifest.Manifest != nil {
		if actions, ok := imported.Manifest.Manifest["actions"].([]interface{}); ok {
			stats["actions_count"] = len(actions)
		}

		// Count dependencies
		if deps, ok := imported.Manifest.Manifest["dependencies"].([]interface{}); ok {
			stats["dependencies_count"] = len(deps)
		}
	}

	// Calculate total size
	totalSize := 0
	for _, content := range imported.Files {
		totalSize += len(content)
	}
	stats["total_size_bytes"] = totalSize

	return stats
}

// ValidateZIPStructure validates the ZIP structure before parsing
func (s *ModuleImportService) ValidateZIPStructure(zipData []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("invalid ZIP file: %w", err)
	}

	// Check for required files
	var hasManifest bool

	for _, file := range reader.File {
		if file.Name == "module.json" {
			hasManifest = true
			break
		}
	}

	if !hasManifest {
		return fmt.Errorf("module.json not found in ZIP")
	}

	// Check ZIP size
	if len(zipData) > 100*1024*1024 { // 100MB limit
		return fmt.Errorf("ZIP file too large (max 100MB)")
	}

	// Check file count
	if len(reader.File) > 1000 {
		return fmt.Errorf("too many files in ZIP (max 1000)")
	}

	return nil
}

// ImportWithValidation imports with full validation
func (s *ModuleImportService) ImportWithValidation(
	ctx context.Context,
	zipData []byte,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (*CustomModule, error) {
	// Validate ZIP structure
	if err := s.ValidateZIPStructure(zipData); err != nil {
		return nil, fmt.Errorf("ZIP validation failed: %w", err)
	}

	// Parse ZIP
	imported, err := s.ParseZIP(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ZIP: %w", err)
	}

	// Validate manifest
	if err := s.ValidateManifest(&imported.Manifest); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	// Check for naming conflicts
	exists, err := s.CheckModuleExists(ctx, imported.Manifest.Slug, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing modules: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("module with slug '%s' already exists in workspace", imported.Manifest.Slug)
	}

	// Resolve dependencies
	if err := s.ResolveDependencies(ctx, &imported.Manifest); err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Create module
	module, err := s.CreateModule(ctx, workspaceID, userID, imported)
	if err != nil {
		return nil, fmt.Errorf("failed to create module: %w", err)
	}

	s.logger.Info("module imported with validation", "module_id", module.ID, "name", module.Name)
	return module, nil
}

// CheckModuleExists checks if a module with the same slug exists
func (s *ModuleImportService) CheckModuleExists(ctx context.Context, slug string, workspaceID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM custom_modules
			WHERE slug = $1 AND workspace_id = $2
		)
	`, slug, workspaceID).Scan(&exists)
	return exists, err
}
