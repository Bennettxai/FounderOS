package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ModuleActionRegistry defines the interface for registering/unregistering module actions
// This allows dependency injection to avoid import cycles with sorx package
type ModuleActionRegistry interface {
	RegisterModuleAction(actionKey string, handler interface{}) error
	UnregisterModuleAction(actionKey string) error
}

// ModuleActionMetadata contains metadata for a module action
type ModuleActionMetadata struct {
	ModuleID    uuid.UUID              `json:"module_id"`
	ModuleSlug  string                 `json:"module_slug"`
	ActionName  string                 `json:"action_name"`
	ActionKey   string                 `json:"action_key"`
	ActionType  string                 `json:"action_type"` // function, api, workflow
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"` // Full action config from manifest
	Logger      *slog.Logger           `json:"-"`
}

// ModuleInstallationService handles installing and managing module installations
type ModuleInstallationService struct {
	pool           *pgxpool.Pool
	logger         *slog.Logger
	actionRegistry ModuleActionRegistry
}

func NewModuleInstallationService(pool *pgxpool.Pool, logger *slog.Logger) *ModuleInstallationService {
	return &ModuleInstallationService{
		pool:   pool,
		logger: logger,
	}
}

// SetActionRegistry sets the action registry (called after service initialization)
func (s *ModuleInstallationService) SetActionRegistry(registry ModuleActionRegistry) {
	s.actionRegistry = registry
}

// InstallModule installs a module into a workspace
func (s *ModuleInstallationService) InstallModule(
	ctx context.Context,
	moduleID uuid.UUID,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) error {
	// Get module
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.GetModule(ctx, moduleID)
	if err != nil {
		return fmt.Errorf("module not found: %w", err)
	}

	// Check if already installed
	var existingID uuid.UUID
	err = s.pool.QueryRow(ctx, `
		SELECT id FROM module_installations
		WHERE module_id = $1 AND workspace_id = $2
	`, moduleID, workspaceID).Scan(&existingID)

	if err == nil {
		return fmt.Errorf("module already installed in this workspace")
	}

	// Create installation record
	query := `
		INSERT INTO module_installations (
			module_id, workspace_id, installed_by, installed_version,
			config_override, is_enabled, is_auto_update
		) VALUES ($1, $2, $3, $4, $5, TRUE, TRUE)
		RETURNING id
	`

	configOverride := make(map[string]interface{})
	configOverrideJSON, _ := json.Marshal(configOverride)

	var installationID uuid.UUID
	err = s.pool.QueryRow(ctx, query,
		moduleID, workspaceID, userID, module.Version,
		configOverrideJSON,
	).Scan(&installationID)

	if err != nil {
		return fmt.Errorf("failed to create installation record: %w", err)
	}

	// Register actions with sorx engine
	if err := s.RegisterActions(ctx, module); err != nil {
		s.logger.Warn("failed to register actions", "module_id", moduleID, "error", err)
		// Don't fail installation, just log warning
	}

	// Update install count
	_, err = s.pool.Exec(ctx, `
		UPDATE custom_modules SET install_count = install_count + 1
		WHERE id = $1
	`, moduleID)
	if err != nil {
		s.logger.Warn("failed to update install count", "module_id", moduleID, "error", err)
	}

	s.logger.Info("module installed", "module_id", moduleID, "workspace_id", workspaceID, "installation_id", installationID)
	return nil
}

// UninstallModule uninstalls a module from a workspace
func (s *ModuleInstallationService) UninstallModule(
	ctx context.Context,
	moduleID uuid.UUID,
	workspaceID uuid.UUID,
) error {
	// Get module before uninstalling (for action cleanup)
	moduleService := NewCustomModuleService(s.pool, s.logger)
	module, err := moduleService.GetModule(ctx, moduleID)
	if err != nil {
		s.logger.Warn("failed to get module for uninstall", "module_id", moduleID, "error", err)
	}

	// Delete installation record
	result, err := s.pool.Exec(ctx, `
		DELETE FROM module_installations
		WHERE module_id = $1 AND workspace_id = $2
	`, moduleID, workspaceID)

	if err != nil {
		return fmt.Errorf("failed to delete installation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("module not installed in this workspace")
	}

	// Unregister actions
	if module != nil {
		if err := s.UnregisterActions(module); err != nil {
			s.logger.Warn("failed to unregister actions", "module_id", moduleID, "error", err)
		}
	}

	// Update install count
	_, err = s.pool.Exec(ctx, `
		UPDATE custom_modules SET install_count = GREATEST(install_count - 1, 0)
		WHERE id = $1
	`, moduleID)
	if err != nil {
		s.logger.Warn("failed to update install count", "module_id", moduleID, "error", err)
	}

	s.logger.Info("module uninstalled", "module_id", moduleID, "workspace_id", workspaceID)
	return nil
}

// RegisterActions registers module actions with sorx engine
func (s *ModuleInstallationService) RegisterActions(ctx context.Context, module *CustomModule) error {
	// Skip if no action registry configured (e.g., during testing)
	if s.actionRegistry == nil {
		s.logger.Warn("action registry not configured, skipping action registration",
			"module", module.Slug)
		return nil
	}

	if module.Manifest == nil {
		return fmt.Errorf("module has no manifest")
	}

	actions, ok := module.Manifest["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		return fmt.Errorf("no actions found in manifest")
	}

	registeredCount := 0
	registrationErrors := []string{}

	for i, actionItem := range actions {
		actionMap, ok := actionItem.(map[string]interface{})
		if !ok {
			s.logger.Warn("invalid action format", "index", i)
			continue
		}

		actionName, _ := actionMap["name"].(string)
		actionType, _ := actionMap["type"].(string)

		if actionName == "" || actionType == "" {
			s.logger.Warn("action missing required fields", "index", i)
			continue
		}

		// Build action key with module prefix
		actionKey := fmt.Sprintf("%s.%s", module.Slug, actionName)

		// Create action metadata for registration
		actionMeta := ModuleActionMetadata{
			ModuleID:    module.ID,
			ModuleSlug:  module.Slug,
			ActionName:  actionName,
			ActionKey:   actionKey,
			ActionType:  actionType,
			Description: getStringField(actionMap, "description"),
			Config:      actionMap,
			Logger:      s.logger,
		}

		// Register with sorx engine via registry
		if err := s.actionRegistry.RegisterModuleAction(actionKey, actionMeta); err != nil {
			errMsg := fmt.Sprintf("%s: %v", actionKey, err)
			registrationErrors = append(registrationErrors, errMsg)
			s.logger.Warn("failed to register action",
				"action", actionKey,
				"type", actionType,
				"error", err)
			continue
		}

		registeredCount++
		s.logger.Info("action registered",
			"action", actionKey,
			"type", actionType,
			"module", module.Slug)
	}

	if registeredCount == 0 && len(registrationErrors) > 0 {
		return fmt.Errorf("failed to register any actions: %v", registrationErrors)
	}

	s.logger.Info("actions registered",
		"module", module.Slug,
		"count", registeredCount,
		"total", len(actions),
		"errors", len(registrationErrors))
	return nil
}

// GetModuleActions returns the actions defined in a module's manifest
// This can be used by the sorx package to register actions without import cycles
func (s *ModuleInstallationService) GetModuleActions(module *CustomModule) ([]map[string]interface{}, error) {
	if module.Manifest == nil {
		return nil, fmt.Errorf("module has no manifest")
	}

	actions, ok := module.Manifest["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		return nil, fmt.Errorf("no actions found in manifest")
	}

	result := make([]map[string]interface{}, 0, len(actions))

	for _, actionItem := range actions {
		actionMap, ok := actionItem.(map[string]interface{})
		if !ok {
			continue
		}

		actionName, _ := actionMap["name"].(string)
		if actionName == "" {
			continue
		}

		// Add module prefix
		enriched := make(map[string]interface{})
		for k, v := range actionMap {
			enriched[k] = v
		}
		enriched["_module_slug"] = module.Slug
		enriched["_action_key"] = fmt.Sprintf("%s.%s", module.Slug, actionName)

		result = append(result, enriched)
	}

	return result, nil
}

// UnregisterActions unregisters module actions from sorx engine
func (s *ModuleInstallationService) UnregisterActions(module *CustomModule) error {
	// Skip if no action registry configured
	if s.actionRegistry == nil {
		s.logger.Warn("action registry not configured, skipping action unregistration",
			"module", module.Slug)
		return nil
	}

	if module.Manifest == nil {
		return nil
	}

	actions, ok := module.Manifest["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		return nil
	}

	unregisteredCount := 0
	for i, actionItem := range actions {
		actionMap, ok := actionItem.(map[string]interface{})
		if !ok {
			continue
		}

		actionName, _ := actionMap["name"].(string)
		if actionName == "" {
			continue
		}

		// Build action key with module prefix
		actionKey := fmt.Sprintf("%s.%s", module.Slug, actionName)

		// Unregister from sorx engine via registry
		if err := s.actionRegistry.UnregisterModuleAction(actionKey); err != nil {
			s.logger.Warn("failed to unregister action",
				"action", actionKey,
				"error", err)
			continue
		}

		unregisteredCount++
		s.logger.Info("action unregistered",
			"action", actionKey,
			"module", module.Slug,
			"index", i)
	}

	s.logger.Info("actions unregistered",
		"module", module.Slug,
		"count", unregisteredCount)
	return nil
}

// LoadInstalledModules loads all installed modules on server startup
func (s *ModuleInstallationService) LoadInstalledModules(ctx context.Context) error {
	s.logger.Info("loading installed modules")

	// Get all enabled installations
	query := `
		SELECT mi.module_id, mi.workspace_id, cm.slug, cm.name, cm.manifest
		FROM module_installations mi
		JOIN custom_modules cm ON cm.id = mi.module_id
		WHERE mi.is_enabled = TRUE
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query installations: %w", err)
	}
	defer rows.Close()

	loadedCount := 0

	for rows.Next() {
		var moduleID, workspaceID uuid.UUID
		var slug, name string
		var manifestJSON []byte

		err := rows.Scan(&moduleID, &workspaceID, &slug, &name, &manifestJSON)
		if err != nil {
			s.logger.Warn("failed to scan installation", "error", err)
			continue
		}

		// Parse manifest
		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			s.logger.Warn("failed to parse manifest", "module_id", moduleID, "error", err)
			continue
		}

		// Create module object
		module := &CustomModule{
			ID:       moduleID,
			Slug:     slug,
			Name:     name,
			Manifest: manifest,
		}

		// Register actions
		if err := s.RegisterActions(ctx, module); err != nil {
			s.logger.Warn("failed to register actions", "module_id", moduleID, "error", err)
			continue
		}

		loadedCount++
		s.logger.Info("module loaded", "module_id", moduleID, "slug", slug, "workspace_id", workspaceID)
	}

	s.logger.Info("installed modules loaded", "count", loadedCount)
	return nil
}

// ListInstalledModules lists all installed modules for a workspace
func (s *ModuleInstallationService) ListInstalledModules(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]ModuleInstallation, error) {
	query := `
		SELECT
			id, module_id, workspace_id, installed_by, installed_version,
			config_override, is_enabled, is_auto_update, installed_at, updated_at, last_used_at
		FROM module_installations
		WHERE workspace_id = $1
		ORDER BY installed_at DESC
	`

	rows, err := s.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query installations: %w", err)
	}
	defer rows.Close()

	installations := make([]ModuleInstallation, 0)

	for rows.Next() {
		var inst ModuleInstallation
		var configJSON []byte

		err := rows.Scan(
			&inst.ID, &inst.ModuleID, &inst.WorkspaceID, &inst.InstalledBy, &inst.InstalledVersion,
			&configJSON, &inst.IsEnabled, &inst.IsAutoUpdate, &inst.InstalledAt, &inst.UpdatedAt,
			&inst.LastUsedAt,
		)

		if err != nil {
			s.logger.Warn("failed to scan installation", "error", err)
			continue
		}

		// Parse config
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &inst.ConfigOverride)
		}

		installations = append(installations, inst)
	}

	return installations, nil
}

// UpdateInstallation updates an installation's settings
func (s *ModuleInstallationService) UpdateInstallation(
	ctx context.Context,
	installationID uuid.UUID,
	configOverride map[string]interface{},
	isEnabled bool,
) error {
	configJSON, err := json.Marshal(configOverride)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE module_installations
		SET config_override = $1, is_enabled = $2, updated_at = NOW()
		WHERE id = $3
	`, configJSON, isEnabled, installationID)

	if err != nil {
		return fmt.Errorf("failed to update installation: %w", err)
	}

	s.logger.Info("installation updated", "installation_id", installationID)
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// getStringField safely extracts a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
