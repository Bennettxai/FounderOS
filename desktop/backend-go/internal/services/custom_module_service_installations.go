package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateInstallation creates a module installation
func (s *CustomModuleService) CreateInstallation(
	ctx context.Context,
	moduleID uuid.UUID,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	version string,
) (*ModuleInstallation, error) {
	query := `
		INSERT INTO module_installations (
			module_id, workspace_id, installed_by, installed_version
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (module_id, workspace_id) DO UPDATE
		SET installed_version = EXCLUDED.installed_version,
		    updated_at = NOW()
		RETURNING
			id, module_id, workspace_id, installed_by, installed_version,
			config_override, is_enabled, is_auto_update,
			installed_at, updated_at, last_used_at
	`

	installation := &ModuleInstallation{}
	err := s.pool.QueryRow(ctx, query, moduleID, workspaceID, userID, version).Scan(
		&installation.ID,
		&installation.ModuleID,
		&installation.WorkspaceID,
		&installation.InstalledBy,
		&installation.InstalledVersion,
		&installation.ConfigOverride,
		&installation.IsEnabled,
		&installation.IsAutoUpdate,
		&installation.InstalledAt,
		&installation.UpdatedAt,
		&installation.LastUsedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create installation", "error", err)
		return nil, fmt.Errorf("failed to create installation: %w", err)
	}

	// Increment install count
	_, _ = s.pool.Exec(ctx, "UPDATE custom_modules SET install_count = install_count + 1 WHERE id = $1", moduleID)

	s.logger.Info("Module installed", "module_id", moduleID, "workspace_id", workspaceID)
	return installation, nil
}

// ListInstallations lists all module installations for a workspace
func (s *CustomModuleService) ListInstallations(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]ModuleInstallation, error) {
	query := `
		SELECT
			id, module_id, workspace_id, installed_by, installed_version,
			config_override, is_enabled, is_auto_update,
			installed_at, updated_at, last_used_at
		FROM module_installations
		WHERE workspace_id = $1
		ORDER BY installed_at DESC
	`

	rows, err := s.pool.Query(ctx, query, workspaceID)
	if err != nil {
		s.logger.Error("Failed to list installations", "error", err)
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	defer rows.Close()

	var installations []ModuleInstallation
	for rows.Next() {
		var inst ModuleInstallation
		err := rows.Scan(
			&inst.ID,
			&inst.ModuleID,
			&inst.WorkspaceID,
			&inst.InstalledBy,
			&inst.InstalledVersion,
			&inst.ConfigOverride,
			&inst.IsEnabled,
			&inst.IsAutoUpdate,
			&inst.InstalledAt,
			&inst.UpdatedAt,
			&inst.LastUsedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan installation", "error", err)
			continue
		}
		installations = append(installations, inst)
	}

	return installations, nil
}
