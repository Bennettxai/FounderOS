package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/utils"
)

// OSAWorkspaceInitService handles automatic workspace creation for new users
type OSAWorkspaceInitService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewOSAWorkspaceInitService creates a new workspace initialization service
func NewOSAWorkspaceInitService(pool *pgxpool.Pool, logger *slog.Logger) *OSAWorkspaceInitService {
	return &OSAWorkspaceInitService{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}


// DefaultWorkspaceLayout returns the default 2D layout configuration
func (s *OSAWorkspaceInitService) DefaultWorkspaceLayout() map[string]interface{} {
	return map[string]interface{}{
		"version": "1.0",
		"panels": []map[string]interface{}{
			{
				"id":       "terminal",
				"type":     "terminal",
				"position": map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 12},
				"visible":  true,
			},
			{
				"id":       "file-explorer",
				"type":     "file-explorer",
				"position": map[string]interface{}{"x": 6, "y": 0, "w": 6, "h": 12},
				"visible":  true,
			},
			{
				"id":       "module-builder",
				"type":     "module-builder",
				"position": map[string]interface{}{"x": 0, "y": 12, "w": 12, "h": 12},
				"visible":  true,
			},
		},
		"connections": []interface{}{},
	}
}

// DefaultWorkspaceSettings returns the default workspace settings
func (s *OSAWorkspaceInitService) DefaultWorkspaceSettings() map[string]interface{} {
	return map[string]interface{}{
		"theme":            "dark",
		"autoSave":         true,
		"autoSaveInterval": 30,
		"notifications": map[string]interface{}{
			"buildComplete": true,
			"errors":        true,
			"warnings":      true,
		},
		"terminal": map[string]interface{}{
			"fontSize":   14,
			"fontFamily": "Monaco, monospace",
			"cursorBlink": true,
		},
		"fileExplorer": map[string]interface{}{
			"showHidden":    false,
			"sortBy":        "name",
			"sortDirection": "asc",
		},
	}
}

// CreateDefaultWorkspace creates a default workspace for a new user
func (s *OSAWorkspaceInitService) CreateDefaultWorkspace(ctx context.Context, userID uuid.UUID) (*sqlc.OsaWorkspace, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	s.logger.Info("Creating default OSA workspace",
		slog.String("user_id", userID.String()),
	)

	// Convert layout and settings to JSONB
	layoutJSON, err := json.Marshal(s.DefaultWorkspaceLayout())
	if err != nil {
		s.logger.Error("Failed to marshal layout", slog.Any("error", err))
		return nil, err
	}

	settingsJSON, err := json.Marshal(s.DefaultWorkspaceSettings())
	if err != nil {
		s.logger.Error("Failed to marshal settings", slog.Any("error", err))
		return nil, err
	}

	// Create the workspace
	workspace, err := s.queries.CreateOSAWorkspace(ctx, sqlc.CreateOSAWorkspaceParams{
		UserID:        uuidToPgtype(userID),
		Name:          "My Workspace",
		Mode:          utils.StringPtr("2d"),
		Layout:        layoutJSON,
		ActiveModules: []pgtype.UUID{}, // Empty initially
		TemplateType:  utils.StringPtr("business_os"),
		Settings:      settingsJSON,
	})

	if err != nil {
		s.logger.Error("Failed to create workspace",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	s.logger.Info("Default workspace created successfully",
		slog.String("user_id", userID.String()),
		slog.String("workspace_id", workspace.ID.String()),
	)

	return &workspace, nil
}

// EnsureUserHasWorkspace ensures a user has at least one workspace, creating default if needed
func (s *OSAWorkspaceInitService) EnsureUserHasWorkspace(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Check if user already has workspaces
	workspaces, err := s.queries.ListOSAWorkspacesByUser(ctx, uuidToPgtype(userID))
	if err != nil {
		s.logger.Error("Failed to list user workspaces",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return err
	}

	// If user has no workspaces, create default
	if len(workspaces) == 0 {
		_, err := s.CreateDefaultWorkspace(ctx, userID)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateWorkspaceFromTemplate creates a workspace from a predefined template
func (s *OSAWorkspaceInitService) CreateWorkspaceFromTemplate(
	ctx context.Context,
	userID uuid.UUID,
	templateType string,
	workspaceName string,
) (*sqlc.OsaWorkspace, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	s.logger.Info("Creating workspace from template",
		slog.String("user_id", userID.String()),
		slog.String("template", templateType),
		slog.String("name", workspaceName),
	)

	// Get template-specific layout and modules
	layout := s.getLayoutForTemplate(templateType)
	settings := s.DefaultWorkspaceSettings()

	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return nil, err
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	workspace, err := s.queries.CreateOSAWorkspace(ctx, sqlc.CreateOSAWorkspaceParams{
		UserID:        uuidToPgtype(userID),
		Name:          workspaceName,
		Mode:          utils.StringPtr("2d"),
		Layout:        layoutJSON,
		ActiveModules: []pgtype.UUID{},
		TemplateType:  utils.StringPtr(templateType),
		Settings:      settingsJSON,
	})

	if err != nil {
		s.logger.Error("Failed to create workspace from template",
			slog.String("template", templateType),
			slog.Any("error", err),
		)
		return nil, err
	}

	return &workspace, nil
}

// getLayoutForTemplate returns layout configuration for different template types
func (s *OSAWorkspaceInitService) getLayoutForTemplate(templateType string) map[string]interface{} {
	switch templateType {
	case "agency_os":
		return map[string]interface{}{
			"version": "1.0",
			"panels": []map[string]interface{}{
				{
					"id":       "client-dashboard",
					"type":     "dashboard",
					"position": map[string]interface{}{"x": 0, "y": 0, "w": 8, "h": 12},
					"visible":  true,
				},
				{
					"id":       "project-timeline",
					"type":     "timeline",
					"position": map[string]interface{}{"x": 8, "y": 0, "w": 4, "h": 12},
					"visible":  true,
				},
			},
		}
	case "content_os":
		return map[string]interface{}{
			"version": "1.0",
			"panels": []map[string]interface{}{
				{
					"id":       "content-calendar",
					"type":     "calendar",
					"position": map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 12},
					"visible":  true,
				},
				{
					"id":       "media-library",
					"type":     "media",
					"position": map[string]interface{}{"x": 6, "y": 0, "w": 6, "h": 12},
					"visible":  true,
				},
			},
		}
	case "sales_os":
		return map[string]interface{}{
			"version": "1.0",
			"panels": []map[string]interface{}{
				{
					"id":       "pipeline",
					"type":     "pipeline",
					"position": map[string]interface{}{"x": 0, "y": 0, "w": 8, "h": 12},
					"visible":  true,
				},
				{
					"id":       "contacts",
					"type":     "contacts",
					"position": map[string]interface{}{"x": 8, "y": 0, "w": 4, "h": 12},
					"visible":  true,
				},
			},
		}
	default: // business_os or custom_os
		return s.DefaultWorkspaceLayout()
	}
}
