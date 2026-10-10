// Package services provides business logic services for the application.
// sandbox_integration.go integrates OSA app generation with sandbox deployment.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/utils"
)

// Sandbox integration errors
var (
	ErrIntegrationAppNotFound = errors.New("app not found")
	ErrWorkspacePathEmpty     = errors.New("workspace path is empty")
	ErrInvalidAppStatus       = errors.New("app status is not ready for deployment")
	ErrDeploymentDisabled     = errors.New("sandbox deployment is disabled")
	ErrNoDockerImage          = errors.New("could not determine docker image for app type")
)

// App status constants for OSA generated apps
const (
	AppStatusGenerated = "generated"
	AppStatusBuilding  = "building"
	AppStatusBuilt     = "built"
	AppStatusDeploying = "deploying"
	AppStatusDeployed  = "deployed"
	AppStatusFailed    = "failed"
	AppStatusError     = "error"
)

// AppConfig represents the configuration extracted from app metadata.
type AppConfig struct {
	AppType       string            `json:"app_type"`       // e.g., "react", "node", "python", "go"
	Framework     string            `json:"framework"`      // e.g., "nextjs", "vite", "express"
	Port          int               `json:"port"`           // Container port
	StartCommand  []string          `json:"start_command"`  // Command to start app
	BuildCommand  []string          `json:"build_command"`  // Command to build app
	Environment   map[string]string `json:"environment"`    // Environment variables
	WorkingDir    string            `json:"working_dir"`    // Working directory in container
	Dependencies  []string          `json:"dependencies"`   // List of dependencies
	RequiresBuild bool              `json:"requires_build"` // Whether app needs build step
}

// SandboxIntegrationService integrates OSA app generation with sandbox deployment.
type SandboxIntegrationService struct {
	sandboxService *SandboxDeploymentService
	pool           *pgxpool.Pool
	queries        *sqlc.Queries
	config         *config.Config
	logger         *slog.Logger

	// Configuration
	autoDeployEnabled bool
	workspaceBasePath string // Base path where OSA stores generated files
}

// NewSandboxIntegrationService creates a new sandbox integration service.
func NewSandboxIntegrationService(
	sandboxService *SandboxDeploymentService,
	pool *pgxpool.Pool,
	cfg *config.Config,
	logger *slog.Logger,
) *SandboxIntegrationService {
	return &SandboxIntegrationService{
		sandboxService:    sandboxService,
		pool:              pool,
		queries:           sqlc.New(pool),
		config:            cfg,
		logger:            logger.With("service", "sandbox_integration"),
		autoDeployEnabled: cfg.OSAEnabled,        // Auto-deploy if OSA is enabled
		workspaceBasePath: "/tmp/osa_workspaces", // Default path, should be configurable
	}
}

// OnAppGenerationComplete handles the completion of app generation and triggers sandbox deployment.
// This is the main integration point called when OSA finishes generating an app.
func (s *SandboxIntegrationService) OnAppGenerationComplete(
	ctx context.Context,
	appID uuid.UUID,
	workspacePath string,
	appConfig *AppConfig,
) error {
	if !s.autoDeployEnabled {
		s.logger.Info("sandbox auto-deploy disabled, skipping", "app_id", appID)
		return ErrDeploymentDisabled
	}

	s.logger.Info("handling app generation completion",
		"app_id", appID,
		"workspace_path", workspacePath,
		"app_type", appConfig.AppType)

	// Validate inputs
	if appID == uuid.Nil {
		return ErrInvalidAppID
	}
	if workspacePath == "" {
		return ErrWorkspacePathEmpty
	}

	// Fetch app details from database
	app, err := s.getAppDetails(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to get app details: %w", err)
	}

	// Validate app status
	if app.Status == nil || !s.isDeployableStatus(*app.Status) {
		s.logger.Warn("app status not ready for deployment",
			"app_id", appID,
			"status", app.Status)
		return fmt.Errorf("%w: %s", ErrInvalidAppStatus, derefString(app.Status))
	}

	// Update app status to deploying
	if err := s.updateAppStatus(ctx, appID, AppStatusDeploying, nil); err != nil {
		s.logger.Warn("failed to update app status to deploying", "app_id", appID, "error", err)
	}

	// Determine Docker image based on app type
	dockerImage, err := s.DetermineDockerImage(appConfig.AppType)
	if err != nil {
		s.updateAppStatus(ctx, appID, AppStatusError, utils.StringPtr(err.Error()))
		return err
	}

	// Build deployment request
	deployReq, err := s.BuildDeploymentRequest(app, workspacePath, appConfig, dockerImage)
	if err != nil {
		s.updateAppStatus(ctx, appID, AppStatusError, utils.StringPtr(err.Error()))
		return fmt.Errorf("failed to build deployment request: %w", err)
	}

	// Deploy the sandbox
	sandboxInfo, err := s.sandboxService.Deploy(ctx, deployReq)
	if err != nil {
		s.logger.Error("sandbox deployment failed",
			"app_id", appID,
			"error", err)
		s.updateAppStatus(ctx, appID, AppStatusFailed, utils.StringPtr(fmt.Sprintf("Deployment failed: %v", err)))
		return fmt.Errorf("sandbox deployment failed: %w", err)
	}

	// Update app status to deployed
	deploymentURL := sandboxInfo.URL
	if err := s.updateAppStatus(ctx, appID, AppStatusDeployed, &deploymentURL); err != nil {
		s.logger.Warn("failed to update app status to deployed", "app_id", appID, "error", err)
	}

	s.logger.Info("app successfully deployed to sandbox",
		"app_id", appID,
		"sandbox_url", sandboxInfo.URL,
		"container_id", sandboxInfo.ContainerID)

	return nil
}

// DetermineDockerImage determines the appropriate Docker image based on app type.
func (s *SandboxIntegrationService) DetermineDockerImage(appType string) (string, error) {
	// Normalize app type
	appType = strings.ToLower(strings.TrimSpace(appType))

	// Map app types to Docker images
	imageMap := map[string]string{
		// JavaScript/TypeScript
		"react":      "node:20-alpine",
		"nextjs":     "node:20-alpine",
		"vue":        "node:20-alpine",
		"svelte":     "node:20-alpine",
		"angular":    "node:20-alpine",
		"node":       "node:20-alpine",
		"nodejs":     "node:20-alpine",
		"express":    "node:20-alpine",
		"typescript": "node:20-alpine",

		// Python
		"python":  "python:3.11-slim",
		"flask":   "python:3.11-slim",
		"django":  "python:3.11-slim",
		"fastapi": "python:3.11-slim",

		// Go
		"go":     "golang:1.22-alpine",
		"golang": "golang:1.22-alpine",

		// Static sites
		"html":   "nginx:alpine",
		"static": "nginx:alpine",

		// Default fallback
		"default": "node:20-alpine",
	}

	image, ok := imageMap[appType]
	if !ok {
		s.logger.Warn("unknown app type, using default image",
			"app_type", appType,
			"default_image", imageMap["default"])
		return imageMap["default"], nil
	}

	return image, nil
}

// BuildDeploymentRequest builds a sandbox deployment request from app details.
func (s *SandboxIntegrationService) BuildDeploymentRequest(
	app *sqlc.OsaModuleInstance,
	workspacePath string,
	appConfig *AppConfig,
	dockerImage string,
) (SandboxDeploymentRequest, error) {
	// Extract user ID from workspace
	userID, err := s.getUserIDFromWorkspace(context.Background(), app.WorkspaceID)
	if err != nil {
		return SandboxDeploymentRequest{}, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Determine container port (default to 3000 if not specified)
	containerPort := appConfig.Port
	if containerPort == 0 {
		containerPort = 3000
	}

	// Build environment variables
	environment := make(map[string]string)
	if appConfig.Environment != nil {
		environment = appConfig.Environment
	}
	// Add default environment variables
	environment["NODE_ENV"] = "development"
	environment["PORT"] = fmt.Sprintf("%d", containerPort)

	// Determine start command
	startCommand := appConfig.StartCommand
	if len(startCommand) == 0 {
		// Infer start command based on app type
		startCommand = s.inferStartCommand(appConfig.AppType)
	}

	// Determine working directory
	workingDir := appConfig.WorkingDir
	if workingDir == "" {
		workingDir = "/workspace"
	}

	// Build absolute workspace path
	absoluteWorkspacePath := workspacePath
	if !filepath.IsAbs(workspacePath) {
		absoluteWorkspacePath = filepath.Join(s.workspaceBasePath, workspacePath)
	}

	req := SandboxDeploymentRequest{
		AppID:         uuid.UUID(app.ID.Bytes),
		AppName:       app.Name,
		UserID:        userID,
		Image:         dockerImage,
		ContainerPort: containerPort,
		WorkspacePath: absoluteWorkspacePath,
		Environment:   environment,
		StartCommand:  startCommand,
		WorkingDir:    workingDir,
		MemoryLimit:   512 * 1024 * 1024, // 512MB default
		CPUQuota:      50000,             // 50% CPU default
	}

	return req, nil
}

// inferStartCommand infers the start command based on app type.
func (s *SandboxIntegrationService) inferStartCommand(appType string) []string {
	appType = strings.ToLower(strings.TrimSpace(appType))

	commandMap := map[string][]string{
		"react":   {"npm", "start"},
		"nextjs":  {"npm", "run", "dev"},
		"vue":     {"npm", "run", "dev"},
		"svelte":  {"npm", "run", "dev"},
		"node":    {"node", "index.js"},
		"express": {"node", "server.js"},
		"python":  {"python", "app.py"},
		"flask":   {"python", "app.py"},
		"fastapi": {"uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"},
		"go":      {"go", "run", "main.go"},
		"static":  {"nginx", "-g", "daemon off;"},
	}

	if cmd, ok := commandMap[appType]; ok {
		return cmd
	}

	// Default
	return []string{"npm", "start"}
}

// getAppDetails fetches app details from the database.
func (s *SandboxIntegrationService) getAppDetails(ctx context.Context, appID uuid.UUID) (*sqlc.OsaModuleInstance, error) {
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	app, err := s.queries.GetOSAModuleInstance(ctx, pgAppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIntegrationAppNotFound, err)
	}
	return &app, nil
}

// getUserIDFromWorkspace retrieves the user ID associated with a workspace.
func (s *SandboxIntegrationService) getUserIDFromWorkspace(ctx context.Context, workspaceID pgtype.UUID) (uuid.UUID, error) {
	workspace, err := s.queries.GetOSAWorkspace(ctx, workspaceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workspace not found: %w", err)
	}
	return uuid.UUID(workspace.UserID.Bytes), nil
}

// updateAppStatus updates the app status in the database.
func (s *SandboxIntegrationService) updateAppStatus(ctx context.Context, appID uuid.UUID, status string, deploymentURL *string) error {
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}

	// Prepare update parameters
	params := sqlc.UpdateOSAModuleInstanceStatusParams{
		ID:            pgAppID,
		Status:        &status,
		ErrorMessage:  nil, // Error message set separately when status is error/failed
		DeploymentUrl: deploymentURL,
	}

	_, err := s.queries.UpdateOSAModuleInstanceStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	return nil
}

// isDeployableStatus checks if the app status is ready for deployment.
func (s *SandboxIntegrationService) isDeployableStatus(status string) bool {
	deployableStatuses := []string{
		AppStatusGenerated,
		AppStatusBuilt,
	}

	for _, ds := range deployableStatuses {
		if status == ds {
			return true
		}
	}
	return false
}

// ParseAppConfig parses app configuration from app metadata JSON.
func ParseAppConfig(metadataJSON []byte) (*AppConfig, error) {
	if len(metadataJSON) == 0 {
		// Return default config
		return &AppConfig{
			AppType:      "node",
			Port:         3000,
			StartCommand: []string{"npm", "start"},
			Environment:  make(map[string]string),
		}, nil
	}

	var config AppConfig
	if err := json.Unmarshal(metadataJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}

	return &config, nil
}

// StopAppSandbox stops the sandbox for an app.
func (s *SandboxIntegrationService) StopAppSandbox(ctx context.Context, appID uuid.UUID) error {
	s.logger.Info("stopping app sandbox", "app_id", appID)

	if err := s.sandboxService.Stop(ctx, appID); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}

	// Update app status
	if err := s.updateAppStatus(ctx, appID, AppStatusBuilt, nil); err != nil {
		s.logger.Warn("failed to update app status after stop", "app_id", appID, "error", err)
	}

	return nil
}

// RestartAppSandbox restarts the sandbox for an app.
func (s *SandboxIntegrationService) RestartAppSandbox(ctx context.Context, appID uuid.UUID) (*SandboxInfo, error) {
	s.logger.Info("restarting app sandbox", "app_id", appID)

	sandboxInfo, err := s.sandboxService.Restart(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to restart sandbox: %w", err)
	}

	// Update app status
	deploymentURL := sandboxInfo.URL
	if err := s.updateAppStatus(ctx, appID, AppStatusDeployed, &deploymentURL); err != nil {
		s.logger.Warn("failed to update app status after restart", "app_id", appID, "error", err)
	}

	return sandboxInfo, nil
}

// RemoveAppSandbox completely removes the sandbox for an app.
func (s *SandboxIntegrationService) RemoveAppSandbox(ctx context.Context, appID uuid.UUID) error {
	s.logger.Info("removing app sandbox", "app_id", appID)

	if err := s.sandboxService.Remove(ctx, appID); err != nil {
		return fmt.Errorf("failed to remove sandbox: %w", err)
	}

	// Update app status
	if err := s.updateAppStatus(ctx, appID, AppStatusGenerated, nil); err != nil {
		s.logger.Warn("failed to update app status after removal", "app_id", appID, "error", err)
	}

	return nil
}

// GetAppSandboxInfo retrieves sandbox information for an app.
func (s *SandboxIntegrationService) GetAppSandboxInfo(ctx context.Context, appID uuid.UUID) (*SandboxInfo, error) {
	return s.sandboxService.GetSandboxInfo(ctx, appID)
}

// ListWorkspaceAppSandboxes lists all sandboxes for apps in a workspace.
func (s *SandboxIntegrationService) ListWorkspaceAppSandboxes(ctx context.Context, workspaceID uuid.UUID) ([]*SandboxInfo, error) {
	// Get user ID from workspace
	pgWorkspaceID := pgtype.UUID{Bytes: workspaceID, Valid: true}
	workspace, err := s.queries.GetOSAWorkspace(ctx, pgWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	userID := uuid.UUID(workspace.UserID.Bytes)
	return s.sandboxService.ListUserSandboxes(ctx, userID)
}

// SetAutoDeployEnabled enables or disables automatic sandbox deployment.
func (s *SandboxIntegrationService) SetAutoDeployEnabled(enabled bool) {
	s.autoDeployEnabled = enabled
	s.logger.Info("auto-deploy setting changed", "enabled", enabled)
}

// SetWorkspaceBasePath sets the base path for OSA workspace files.
func (s *SandboxIntegrationService) SetWorkspaceBasePath(path string) {
	s.workspaceBasePath = path
	s.logger.Info("workspace base path updated", "path", path)
}

// derefString gets string value from pointer safely
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
