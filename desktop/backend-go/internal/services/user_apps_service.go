package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserAppsService handles user-generated apps management
type UserAppsService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// UserGeneratedApp represents a user's generated app
type UserGeneratedApp struct {
	ID               uuid.UUID              `json:"id"`
	WorkspaceID      uuid.UUID              `json:"workspace_id"`
	TemplateID       *uuid.UUID             `json:"template_id"`
	AppName          string                 `json:"app_name"`
	ModuleInstanceID *uuid.UUID             `json:"module_instance_id"`
	IsVisible        bool                   `json:"is_visible"`
	IsPinned         bool                   `json:"is_pinned"`
	IsFavorite       bool                   `json:"is_favorite"`
	PositionIndex    *int                   `json:"position_index"`
	CustomConfig     map[string]interface{} `json:"custom_config"`
	CustomIcon       *string                `json:"custom_icon"`
	GeneratedAt      string                 `json:"generated_at"`
	LastAccessedAt   *string                `json:"last_accessed_at"`
	AccessCount      int                    `json:"access_count"`
}

func NewUserAppsService(pool *pgxpool.Pool, logger *slog.Logger) *UserAppsService {
	return &UserAppsService{
		pool:   pool,
		logger: logger,
	}
}

// ListUserApps returns all visible user apps for a workspace
func (s *UserAppsService) ListUserApps(ctx context.Context, workspaceID uuid.UUID) ([]UserGeneratedApp, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			id,
			workspace_id,
			template_id,
			app_name,
			module_instance_id,
			is_visible,
			is_pinned,
			is_favorite,
			position_index,
			COALESCE(custom_config, '{}'::jsonb),
			custom_icon,
			generated_at,
			last_accessed_at,
			access_count
		FROM user_generated_apps
		WHERE workspace_id = $1 AND is_visible = true
		ORDER BY
			is_pinned DESC,
			position_index ASC NULLS LAST,
			generated_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("query user apps: %w", err)
	}
	defer rows.Close()

	var apps []UserGeneratedApp
	for rows.Next() {
		var app UserGeneratedApp
		var configJSON []byte

		err := rows.Scan(
			&app.ID,
			&app.WorkspaceID,
			&app.TemplateID,
			&app.AppName,
			&app.ModuleInstanceID,
			&app.IsVisible,
			&app.IsPinned,
			&app.IsFavorite,
			&app.PositionIndex,
			&configJSON,
			&app.CustomIcon,
			&app.GeneratedAt,
			&app.LastAccessedAt,
			&app.AccessCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user app: %w", err)
		}

		// Parse custom config
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &app.CustomConfig)
		}

		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return apps, nil
}

// GetUserApp returns a single user app by ID
func (s *UserAppsService) GetUserApp(ctx context.Context, appID uuid.UUID, workspaceID uuid.UUID) (*UserGeneratedApp, error) {
	var app UserGeneratedApp
	var configJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT
			id,
			workspace_id,
			template_id,
			app_name,
			module_instance_id,
			is_visible,
			is_pinned,
			is_favorite,
			position_index,
			COALESCE(custom_config, '{}'::jsonb),
			custom_icon,
			generated_at,
			last_accessed_at,
			access_count
		FROM user_generated_apps
		WHERE id = $1 AND workspace_id = $2
	`, appID, workspaceID).Scan(
		&app.ID,
		&app.WorkspaceID,
		&app.TemplateID,
		&app.AppName,
		&app.ModuleInstanceID,
		&app.IsVisible,
		&app.IsPinned,
		&app.IsFavorite,
		&app.PositionIndex,
		&configJSON,
		&app.CustomIcon,
		&app.GeneratedAt,
		&app.LastAccessedAt,
		&app.AccessCount,
	)

	if err != nil {
		return nil, fmt.Errorf("query user app: %w", err)
	}

	// Parse custom config
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &app.CustomConfig)
	}

	return &app, nil
}

// IncrementAccessCount increments the access count for an app
func (s *UserAppsService) IncrementAccessCount(ctx context.Context, appID uuid.UUID, workspaceID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_generated_apps
		SET
			access_count = access_count + 1,
			last_accessed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2
	`, appID, workspaceID)

	if err != nil {
		return fmt.Errorf("increment access count: %w", err)
	}

	return nil
}

// CreateUserApp creates a new user app from a template
func (s *UserAppsService) CreateUserApp(ctx context.Context, workspaceID uuid.UUID, templateID uuid.UUID, appName string, config map[string]interface{}) (*UserGeneratedApp, error) {
	var configJSON []byte
	var err error
	if config != nil {
		configJSON, err = json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("marshal config: %w", err)
		}
	}

	var app UserGeneratedApp
	var returnedConfigJSON []byte

	err = s.pool.QueryRow(ctx, `
		INSERT INTO user_generated_apps (
			workspace_id,
			template_id,
			app_name,
			is_visible,
			is_pinned,
			is_favorite,
			position_index,
			custom_config
		) VALUES (
			$1, $2, $3, true, false, false, NULL, $4
		) RETURNING
			id,
			workspace_id,
			template_id,
			app_name,
			module_instance_id,
			is_visible,
			is_pinned,
			is_favorite,
			position_index,
			COALESCE(custom_config, '{}'::jsonb),
			custom_icon,
			generated_at,
			last_accessed_at,
			access_count
	`, workspaceID, templateID, appName, configJSON).Scan(
		&app.ID,
		&app.WorkspaceID,
		&app.TemplateID,
		&app.AppName,
		&app.ModuleInstanceID,
		&app.IsVisible,
		&app.IsPinned,
		&app.IsFavorite,
		&app.PositionIndex,
		&returnedConfigJSON,
		&app.CustomIcon,
		&app.GeneratedAt,
		&app.LastAccessedAt,
		&app.AccessCount,
	)

	if err != nil {
		s.logger.Error("failed to create user app", "workspace_id", workspaceID, "template_id", templateID, "error", err)
		return nil, fmt.Errorf("create user app: %w", err)
	}

	// Parse custom config
	if len(returnedConfigJSON) > 0 {
		json.Unmarshal(returnedConfigJSON, &app.CustomConfig)
	}

	s.logger.Info("user app created", "app_id", app.ID, "workspace_id", workspaceID, "template_id", templateID)

	return &app, nil
}

// DeleteUserApp soft deletes a user app
func (s *UserAppsService) DeleteUserApp(ctx context.Context, appID uuid.UUID, workspaceID uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM user_generated_apps
		WHERE id = $1 AND workspace_id = $2
	`, appID, workspaceID)

	if err != nil {
		s.logger.Error("failed to delete user app", "app_id", appID, "workspace_id", workspaceID, "error", err)
		return fmt.Errorf("delete user app: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("app not found")
	}

	s.logger.Info("user app deleted", "app_id", appID, "workspace_id", workspaceID)
	return nil
}

// UpdateUserApp updates user app settings (visibility, pinning, etc.)
func (s *UserAppsService) UpdateUserApp(ctx context.Context, appID uuid.UUID, workspaceID uuid.UUID, updates map[string]interface{}) error {
	// Build dynamic update query based on provided fields
	query := "UPDATE user_generated_apps SET updated_at = NOW()"
	args := []interface{}{appID, workspaceID}
	argPos := 3

	if isVisible, ok := updates["is_visible"].(bool); ok {
		query += fmt.Sprintf(", is_visible = $%d", argPos)
		args = append(args, isVisible)
		argPos++
	}

	if isPinned, ok := updates["is_pinned"].(bool); ok {
		query += fmt.Sprintf(", is_pinned = $%d", argPos)
		args = append(args, isPinned)
		argPos++
	}

	if isFavorite, ok := updates["is_favorite"].(bool); ok {
		query += fmt.Sprintf(", is_favorite = $%d", argPos)
		args = append(args, isFavorite)
		argPos++
	}

	if positionIndex, ok := updates["position_index"].(int); ok {
		query += fmt.Sprintf(", position_index = $%d", argPos)
		args = append(args, positionIndex)
		argPos++
	}

	if customIcon, ok := updates["custom_icon"].(string); ok {
		query += fmt.Sprintf(", custom_icon = $%d", argPos)
		args = append(args, customIcon)
		argPos++
	}

	query += " WHERE id = $1 AND workspace_id = $2"

	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user app: %w", err)
	}

	return nil
}
