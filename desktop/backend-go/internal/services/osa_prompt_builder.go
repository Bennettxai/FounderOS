package services

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"path" // Use path instead of filepath for embed.FS (always uses forward slashes)
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"gopkg.in/yaml.v3"
)

// systemTemplatesFS embeds OSA template files at compile time
//
//go:embed all:templates/osa
var systemTemplatesFS embed.FS

// OSAPromptBuilder handles template loading and rendering for OSA prompts
type OSAPromptBuilder struct {
	pool            *pgxpool.Pool
	queries         *sqlc.Queries
	systemTemplates map[string]*PromptTemplate
	logger          *slog.Logger
}

// PromptTemplate represents a parsed template
type PromptTemplate struct {
	Name           string               `yaml:"name"`
	DisplayName    string               `yaml:"display_name"`
	Description    string               `yaml:"description"`
	Category       string               `yaml:"category"`
	Version        string               `yaml:"version"`
	Tags           []string             `yaml:"tags"`
	Variables      []VariableDefinition `yaml:"variables"`
	TemplateString string               `yaml:"template"`
	compiledTpl    *template.Template
}

// VariableDefinition defines a template variable
type VariableDefinition struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
	Description string      `yaml:"description"`
}

// AppGenerationRequest represents a request to build an app generation prompt
type AppGenerationRequest struct {
	TemplateName string
	Variables    map[string]interface{}
	UserID       *uuid.UUID
	WorkspaceID  *uuid.UUID
}

// PromptBuildResult contains the rendered prompt and metadata
type PromptBuildResult struct {
	Prompt       string
	TemplateID   *uuid.UUID
	TemplateName string
	Variables    map[string]interface{}
	RenderTimeMs int64
}

// NewOSAPromptBuilder creates a new prompt builder instance
func NewOSAPromptBuilder(pool *pgxpool.Pool, logger *slog.Logger) (*OSAPromptBuilder, error) {
	pb := &OSAPromptBuilder{
		pool:            pool,
		queries:         sqlc.New(pool),
		systemTemplates: make(map[string]*PromptTemplate),
		logger:          logger,
	}

	// Load system templates from embedded filesystem
	if err := pb.loadSystemTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load system templates: %w", err)
	}

	pb.logger.Info("OSA prompt builder initialized", "system_templates", len(pb.systemTemplates))

	return pb, nil
}

// loadSystemTemplates reads YAML files from embedded FS
func (pb *OSAPromptBuilder) loadSystemTemplates() error {
	entries, err := systemTemplatesFS.ReadDir("templates/osa")
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Use path.Join() instead of filepath.Join() for embed.FS (always uses forward slashes)
		data, err := systemTemplatesFS.ReadFile(path.Join("templates/osa", entry.Name()))
		if err != nil {
			pb.logger.Warn("failed to read template file", "file", entry.Name(), "error", err)
			continue
		}

		var tpl PromptTemplate
		if err := yaml.Unmarshal(data, &tpl); err != nil {
			pb.logger.Warn("failed to parse template YAML", "file", entry.Name(), "error", err)
			continue
		}

		// Compile the template
		compiledTpl, err := template.New(tpl.Name).Parse(tpl.TemplateString)
		if err != nil {
			pb.logger.Warn("failed to compile template", "name", tpl.Name, "error", err)
			continue
		}
		tpl.compiledTpl = compiledTpl

		pb.systemTemplates[tpl.Name] = &tpl
		pb.logger.Info("loaded system template", "name", tpl.Name, "version", tpl.Version, "category", tpl.Category)
	}

	if len(pb.systemTemplates) == 0 {
		return fmt.Errorf("no system templates loaded")
	}

	return nil
}

// BuildAppGenerationPrompt builds a prompt for app generation
func (pb *OSAPromptBuilder) BuildAppGenerationPrompt(
	ctx context.Context,
	req AppGenerationRequest,
) (*PromptBuildResult, error) {
	startTime := time.Now()

	// Resolve template (user > workspace > system)
	tpl, templateID, err := pb.resolveTemplate(ctx, req.TemplateName, req.UserID, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve template: %w", err)
	}

	// Validate required variables
	if err := pb.validateVariables(tpl, req.Variables); err != nil {
		return nil, fmt.Errorf("variable validation failed: %w", err)
	}

	// Apply defaults for missing optional variables
	variables := pb.applyDefaults(tpl, req.Variables)

	// Render template
	var buf bytes.Buffer
	if err := tpl.compiledTpl.Execute(&buf, variables); err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	renderTime := time.Since(startTime).Milliseconds()

	result := &PromptBuildResult{
		Prompt:       buf.String(),
		TemplateID:   templateID,
		TemplateName: tpl.Name,
		Variables:    variables,
		RenderTimeMs: renderTime,
	}

	pb.logger.Info("prompt rendered successfully",
		"template", tpl.Name,
		"render_time_ms", renderTime,
		"prompt_length", len(result.Prompt),
	)

	return result, nil
}

// resolveTemplate finds the appropriate template (user > workspace > system)
func (pb *OSAPromptBuilder) resolveTemplate(
	ctx context.Context,
	name string,
	userID *uuid.UUID,
	workspaceID *uuid.UUID,
) (*PromptTemplate, *uuid.UUID, error) {
	// Try user-level template first
	if userID != nil {
		tpl, id, err := pb.getDBTemplate(ctx, name, "user", userID, nil)
		if err == nil && tpl != nil {
			pb.logger.Debug("resolved user-level template", "name", name, "user_id", userID)
			return tpl, id, nil
		}
	}

	// Try workspace-level template
	if workspaceID != nil {
		tpl, id, err := pb.getDBTemplate(ctx, name, "workspace", nil, workspaceID)
		if err == nil && tpl != nil {
			pb.logger.Debug("resolved workspace-level template", "name", name, "workspace_id", workspaceID)
			return tpl, id, nil
		}
	}

	// Fall back to system template
	if tpl, ok := pb.systemTemplates[name]; ok {
		pb.logger.Debug("resolved system-level template", "name", name)
		return tpl, nil, nil
	}

	return nil, nil, fmt.Errorf("template not found: %s", name)
}

// getDBTemplate retrieves a template from the database.
// Database templates are not yet implemented; callers fall through to system templates.
func (pb *OSAPromptBuilder) getDBTemplate(
	ctx context.Context,
	name string,
	scope string,
	userID *uuid.UUID,
	workspaceID *uuid.UUID,
) (*PromptTemplate, *uuid.UUID, error) {
	// Database templates not yet implemented — fall through to system templates
	pb.logger.Debug("database templates not yet implemented, falling through to system",
		"name", name,
		"scope", scope,
	)
	return nil, nil, nil
}

// validateVariables ensures required variables are provided
func (pb *OSAPromptBuilder) validateVariables(tpl *PromptTemplate, vars map[string]interface{}) error {
	var missing []string

	for _, varDef := range tpl.Variables {
		if varDef.Required {
			if _, ok := vars[varDef.Name]; !ok {
				missing = append(missing, varDef.Name)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required variables missing: %s", strings.Join(missing, ", "))
	}

	return nil
}

// applyDefaults fills in default values for missing optional variables
func (pb *OSAPromptBuilder) applyDefaults(tpl *PromptTemplate, vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy provided variables
	for k, v := range vars {
		result[k] = v
	}

	// Apply defaults for missing optional variables
	for _, varDef := range tpl.Variables {
		if !varDef.Required {
			if _, ok := result[varDef.Name]; !ok && varDef.Default != nil {
				result[varDef.Name] = varDef.Default
			}
		}
	}

	return result
}

// GetTemplateInfo returns information about a template (for UI/API)
func (pb *OSAPromptBuilder) GetTemplateInfo(ctx context.Context, templateName string) (*PromptTemplate, error) {
	// Try system templates first
	if tpl, ok := pb.systemTemplates[templateName]; ok {
		return tpl, nil
	}

	return nil, fmt.Errorf("template not found: %s", templateName)
}

// ListAvailableTemplates returns all available templates
func (pb *OSAPromptBuilder) ListAvailableTemplates(ctx context.Context, category string) ([]*PromptTemplate, error) {
	var templates []*PromptTemplate

	// Add system templates
	for _, tpl := range pb.systemTemplates {
		if category == "" || tpl.Category == category {
			templates = append(templates, tpl)
		}
	}

	pb.logger.Debug("listed available templates", "count", len(templates), "category", category)

	return templates, nil
}

// LogTemplateUsage logs template usage for analytics (async)
func (pb *OSAPromptBuilder) LogTemplateUsage(
	ctx context.Context,
	result *PromptBuildResult,
	userID uuid.UUID,
	workspaceID *uuid.UUID,
	status string,
	errorMsg string,
) {
	// Run in background to avoid blocking
	go func() {
		// Create a new context with timeout for the background operation
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		variablesJSON, err := json.Marshal(result.Variables)
		if err != nil {
			pb.logger.Warn("failed to marshal variables for logging", "error", err)
			return
		}

		pb.logger.Debug("template usage persistence is not configured",
			"template", result.TemplateName,
			"user_id", userID,
			"workspace_id", workspaceID,
			"status", status,
			"render_time_ms", result.RenderTimeMs,
		)

		_ = bgCtx
		_ = variablesJSON
		_ = errorMsg
	}()
}

// ValidateTemplateVariables validates a set of variables against a template
// without rendering (useful for API validation)
func (pb *OSAPromptBuilder) ValidateTemplateVariables(
	ctx context.Context,
	templateName string,
	variables map[string]interface{},
) error {
	tpl, err := pb.GetTemplateInfo(ctx, templateName)
	if err != nil {
		return err
	}

	return pb.validateVariables(tpl, variables)
}

// GetTemplateVariableSchema returns the variable schema for a template (for UI generation)
func (pb *OSAPromptBuilder) GetTemplateVariableSchema(
	ctx context.Context,
	templateName string,
) ([]VariableDefinition, error) {
	tpl, err := pb.GetTemplateInfo(ctx, templateName)
	if err != nil {
		return nil, err
	}

	return tpl.Variables, nil
}

// renderTemplate executes a compiled PromptTemplate with the provided variables.
func (pb *OSAPromptBuilder) renderTemplate(tpl *PromptTemplate, vars map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tpl.compiledTpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}
	return buf.String(), nil
}

// BuildStandardVariables creates the standard variable map for app generation templates.
func (pb *OSAPromptBuilder) BuildStandardVariables(
	appName string,
	description string,
	features []string,
	complexity string,
) map[string]interface{} {
	vars := map[string]interface{}{
		"app_name":    appName,
		"description": description,
		"features":    features,
		"timestamp":   time.Now().Format(time.RFC3339),
	}
	if complexity != "" {
		vars["complexity"] = complexity
	}
	if len(features) > 0 {
		vars["features_list"] = strings.Join(features, ", ")
	} else {
		vars["features_list"] = "none specified"
	}
	return vars
}

// AgentPromptSet contains rendered prompts for each agent type in the generation pipeline.
type AgentPromptSet struct {
	Frontend string
	Backend  string
	Database string
	Test     string
	Metadata map[string]interface{}
}

// BuildAgentPrompts generates prompt context for each agent in the generation pipeline.
// The BACKEND agent calls this from app_generation_orchestrator.go to get enriched
// task descriptions instead of the current hardcoded strings.
func (pb *OSAPromptBuilder) BuildAgentPrompts(
	ctx context.Context,
	appName string,
	description string,
	features []string,
) (*AgentPromptSet, error) {
	vars := pb.BuildStandardVariables(appName, description, features, "")

	// Try to render a base context from templates
	baseContext := ""
	tpl, _, err := pb.resolveTemplate(ctx, "app-generation", nil, nil)
	if err == nil && tpl != nil {
		rendered, renderErr := pb.renderTemplate(tpl, vars)
		if renderErr == nil {
			baseContext = rendered
		}
	}

	featuresStr := ""
	if len(features) > 0 {
		featuresStr = " Features: " + strings.Join(features, ", ") + "."
	}

	base := appName + ": " + description + featuresStr
	if baseContext != "" {
		base = base + "\n\n" + baseContext
	}

	return &AgentPromptSet{
		Frontend: "Create Svelte frontend for " + base,
		Backend:  "Create Go backend for " + base,
		Database: "Create PostgreSQL migrations for " + base,
		Test:     "Create tests for " + base,
		Metadata: vars,
	}, nil
}
