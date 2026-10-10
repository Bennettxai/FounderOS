package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TemplateGenerationService handles generating apps from templates
type TemplateGenerationService struct {
	pool            *pgxpool.Pool
	logger          *slog.Logger
	templateService *AppTemplateService
	userAppsService *UserAppsService
}

// GenerateFromTemplateRequest represents a request to generate an app from a template
type GenerateFromTemplateRequest struct {
	TemplateID  uuid.UUID              `json:"template_id"`
	WorkspaceID uuid.UUID              `json:"workspace_id"`
	AppName     string                 `json:"app_name"`
	Config      map[string]interface{} `json:"config"`
}

// GeneratedFile represents a single generated file
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int    `json:"size"`
}

// GenerationResult represents the result of generating an app from a template
type GenerationResult struct {
	AppID          uuid.UUID       `json:"app_id"`
	AppName        string          `json:"app_name"`
	TemplateID     uuid.UUID       `json:"template_id"`
	TemplateName   string          `json:"template_name"`
	WorkspaceID    uuid.UUID       `json:"workspace_id"`
	Files          []GeneratedFile `json:"files"`
	TotalFiles     int             `json:"total_files"`
	Status         string          `json:"status"`
	VersionNumber  string          `json:"version_number"`
	GeneratedAt    time.Time       `json:"generated_at"`
}

// NewTemplateGenerationService creates a new template generation service
func NewTemplateGenerationService(pool *pgxpool.Pool, logger *slog.Logger) *TemplateGenerationService {
	return &TemplateGenerationService{
		pool:            pool,
		logger:          logger,
		templateService: NewAppTemplateService(pool, logger),
		userAppsService: NewUserAppsService(pool, logger),
	}
}

// GenerateFromTemplate generates an app from a template with user configuration
func (s *TemplateGenerationService) GenerateFromTemplate(ctx context.Context, userID uuid.UUID, req GenerateFromTemplateRequest) (*GenerationResult, error) {
	s.logger.Info("generating app from template",
		"template_id", req.TemplateID,
		"workspace_id", req.WorkspaceID,
		"app_name", req.AppName,
		"user_id", userID,
	)

	// 1. Get the template
	template, err := s.templateService.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// 2. Get the built-in template definition
	builtinTemplate, exists := GetBuiltInTemplate(template.TemplateName)
	if !exists {
		return nil, fmt.Errorf("built-in template not found: %s", template.TemplateName)
	}

	// 3. Generate files from template
	config := mergeConfig(req.Config, template.TemplateConfig)
	config["app_name"] = req.AppName
	config["workspace_id"] = req.WorkspaceID.String()
	config["generated_at"] = time.Now().Format(time.RFC3339)

	files := generateFiles(builtinTemplate, config)

	// 4. Create user app in database
	app, err := s.userAppsService.CreateUserApp(ctx, req.WorkspaceID, req.TemplateID, req.AppName, config)
	if err != nil {
		return nil, fmt.Errorf("create user app: %w", err)
	}

	// 5. Create initial version snapshot
	snapshotData := map[string]interface{}{
		"files":     files,
		"config":    config,
		"template":  template.TemplateName,
		"generated": true,
	}
	snapshotJSON, err := json.Marshal(snapshotData)
	if err != nil {
		s.logger.Error("failed to marshal snapshot data", "error", err)
	} else {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO app_versions (app_id, version_number, snapshot_data, snapshot_metadata, change_summary, created_by)
			VALUES ($1, '1.0.0', $2, '{"source": "template_generation"}'::jsonb, $3, $4)
		`, app.ID, snapshotJSON, fmt.Sprintf("Initial generation from template: %s", template.DisplayName), userID)
		if err != nil {
			s.logger.Error("failed to create initial version", "app_id", app.ID, "error", err)
		}
	}

	s.logger.Info("app generated successfully",
		"app_id", app.ID,
		"template", template.DisplayName,
		"files_count", len(files),
	)

	return &GenerationResult{
		AppID:         app.ID,
		AppName:       req.AppName,
		TemplateID:    req.TemplateID,
		TemplateName:  template.DisplayName,
		WorkspaceID:   req.WorkspaceID,
		Files:         files,
		TotalFiles:    len(files),
		Status:        "completed",
		VersionNumber: "1.0.0",
		GeneratedAt:   time.Now(),
	}, nil
}

// generateFiles generates files from a built-in template with config substitution
func generateFiles(template *BuiltInTemplate, config map[string]interface{}) []GeneratedFile {
	var files []GeneratedFile
	for path, content := range template.FilesTemplate {
		rendered := substituteConfig(content, config)
		files = append(files, GeneratedFile{
			Path:    path,
			Content: rendered,
			Size:    len(rendered),
		})
	}
	return files
}

// substituteConfig replaces {{key}} placeholders with config values
func substituteConfig(content string, config map[string]interface{}) string {
	result := content
	for key, value := range config {
		placeholder := fmt.Sprintf("{{%s}}", key)
		strValue := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, strValue)
	}
	return result
}

// mergeConfig merges user config with template defaults (user config takes precedence)
func mergeConfig(userConfig, templateConfig map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range templateConfig {
		merged[k] = v
	}
	for k, v := range userConfig {
		merged[k] = v
	}
	return merged
}
