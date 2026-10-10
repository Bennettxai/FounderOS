package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// restoreSettings restores workspace settings
func (s *WorkspaceVersionService) restoreSettings(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID uuid.UUID,
	settings map[string]interface{},
) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE workspaces
		SET settings = $1, updated_at = NOW()
		WHERE id = $2
	`, settingsJSON, workspaceID)

	return err
}

// restoreRoles restores workspace roles
func (s *WorkspaceVersionService) restoreRoles(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID uuid.UUID,
	roles []RoleSnapshot,
) error {
	// Delete existing non-system roles
	_, err := tx.Exec(ctx, `
		DELETE FROM workspace_roles
		WHERE workspace_id = $1 AND is_system = FALSE
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("delete existing roles: %w", err)
	}

	// Restore roles
	for _, role := range roles {
		if role.IsSystem {
			// Skip system roles - don't restore them
			continue
		}

		permissions, _ := json.Marshal(role.Permissions)

		_, err := tx.Exec(ctx, `
			INSERT INTO workspace_roles (
				id, workspace_id, name, display_name, description, color, icon,
				hierarchy_level, is_system, is_default, permissions
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (workspace_id, name) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description = EXCLUDED.description,
				color = EXCLUDED.color,
				icon = EXCLUDED.icon,
				hierarchy_level = EXCLUDED.hierarchy_level,
				is_default = EXCLUDED.is_default,
				permissions = EXCLUDED.permissions
		`, role.ID, workspaceID, role.Name, role.DisplayName, role.Description,
			role.Color, role.Icon, role.HierarchyLevel, role.IsSystem,
			role.IsDefault, permissions)

		if err != nil {
			return fmt.Errorf("insert role %s: %w", role.Name, err)
		}
	}

	return nil
}

// restoreMembers restores workspace members
func (s *WorkspaceVersionService) restoreMembers(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID uuid.UUID,
	members []MemberSnapshot,
) error {
	// Don't delete existing members - only update/add
	// This prevents accidental removal of current members

	for _, member := range members {
		_, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (
				id, workspace_id, user_id, role_id, role_name, status,
				invited_at, joined_at, invited_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (workspace_id, user_id) DO UPDATE SET
				role_id = EXCLUDED.role_id,
				role_name = EXCLUDED.role_name,
				status = EXCLUDED.status
		`, member.ID, workspaceID, member.UserID, member.RoleID, member.RoleName,
			member.Status, member.InvitedAt, member.JoinedAt, member.InvitedBy)

		if err != nil {
			return fmt.Errorf("restore member %s: %w", member.UserID, err)
		}
	}

	return nil
}

// restoreApps restores user-generated apps
func (s *WorkspaceVersionService) restoreApps(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID uuid.UUID,
	apps []AppSnapshot,
) error {
	// Delete existing apps
	_, err := tx.Exec(ctx, `
		DELETE FROM user_generated_apps WHERE workspace_id = $1
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("delete existing apps: %w", err)
	}

	// Restore apps
	for _, app := range apps {
		customConfig, _ := json.Marshal(app.CustomConfig)

		_, err := tx.Exec(ctx, `
			INSERT INTO user_generated_apps (
				id, workspace_id, template_id, app_name, module_instance_id,
				is_visible, is_pinned, is_favorite, position_index,
				custom_config, custom_icon, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		`, app.ID, workspaceID, app.TemplateID, app.AppName, app.ModuleInstanceID,
			app.IsVisible, app.IsPinned, app.IsFavorite, app.PositionIndex,
			customConfig, app.CustomIcon)

		if err != nil {
			return fmt.Errorf("restore app %s: %w", app.AppName, err)
		}
	}

	return nil
}

// restoreMemories restores workspace memories
func (s *WorkspaceVersionService) restoreMemories(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID uuid.UUID,
	memories []MemorySnapshot,
) error {
	// Delete existing memories (except user-specific ones)
	_, err := tx.Exec(ctx, `
		DELETE FROM workspace_memories
		WHERE workspace_id = $1 AND visibility = 'workspace'
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("delete existing memories: %w", err)
	}

	// Restore memories
	for _, memory := range memories {
		metadata, _ := json.Marshal(memory.Metadata)

		_, err := tx.Exec(ctx, `
			INSERT INTO workspace_memories (
				id, workspace_id, user_id, title, summary, content,
				memory_type, category, scope_type, scope_id, visibility,
				created_by, importance_score, tags, source, metadata,
				is_pinned, is_active, is_archived, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW())
			ON CONFLICT (id) DO UPDATE SET
				title = EXCLUDED.title,
				summary = EXCLUDED.summary,
				content = EXCLUDED.content,
				importance_score = EXCLUDED.importance_score
		`, memory.ID, workspaceID, memory.UserID, memory.Title, memory.Summary,
			memory.Content, memory.MemoryType, memory.Category, memory.ScopeType,
			memory.ScopeID, memory.Visibility, memory.CreatedBy, memory.ImportanceScore,
			memory.Tags, memory.Source, metadata, memory.IsPinned,
			memory.IsActive, memory.IsArchived)

		if err != nil {
			return fmt.Errorf("restore memory %s: %w", memory.ID, err)
		}
	}

	return nil
}
