package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// captureWorkspaceState captures current workspace state
func (s *WorkspaceVersionService) captureWorkspaceState(
	ctx context.Context,
	workspaceID uuid.UUID,
) json.RawMessage {
	snapshot := WorkspaceSnapshot{
		Timestamp: time.Now(),
		Apps:      s.captureApps(ctx, workspaceID),
		Members:   s.captureMembers(ctx, workspaceID),
		Roles:     s.captureRoles(ctx, workspaceID),
		Settings:  s.captureSettings(ctx, workspaceID),
		Memories:  s.captureMemories(ctx, workspaceID),
	}

	// Calculate metadata
	snapshot.Metadata = SnapshotMetadata{
		AppCount:    len(snapshot.Apps),
		MemberCount: len(snapshot.Members),
		RoleCount:   len(snapshot.Roles),
		MemoryCount: len(snapshot.Memories),
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		s.logger.Error("failed to marshal snapshot", "error", err)
		return json.RawMessage("{}")
	}

	return data
}

// captureApps captures all user-generated apps
func (s *WorkspaceVersionService) captureApps(ctx context.Context, workspaceID uuid.UUID) []AppSnapshot {
	rows, err := s.pool.Query(ctx, `
		SELECT id, app_name, template_id, module_instance_id, is_visible, is_pinned, is_favorite,
		       position_index, custom_config, custom_icon
		FROM user_generated_apps
		WHERE workspace_id = $1
		ORDER BY position_index
	`, workspaceID)

	if err != nil {
		s.logger.Error("failed to capture apps", "error", err)
		return []AppSnapshot{}
	}
	defer rows.Close()

	var apps []AppSnapshot
	for rows.Next() {
		var app AppSnapshot
		var customConfig json.RawMessage

		err := rows.Scan(
			&app.ID, &app.AppName, &app.TemplateID, &app.ModuleInstanceID,
			&app.IsVisible, &app.IsPinned, &app.IsFavorite,
			&app.PositionIndex, &customConfig, &app.CustomIcon,
		)
		if err != nil {
			s.logger.Error("failed to scan app", "error", err)
			continue
		}

		// Unmarshal custom config
		if len(customConfig) > 0 {
			json.Unmarshal(customConfig, &app.CustomConfig)
		}

		apps = append(apps, app)
	}

	return apps
}

// captureMembers captures all workspace members
func (s *WorkspaceVersionService) captureMembers(ctx context.Context, workspaceID uuid.UUID) []MemberSnapshot {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, role_id, role_name, status, invited_at, joined_at, invited_by
		FROM workspace_members
		WHERE workspace_id = $1
		ORDER BY joined_at
	`, workspaceID)

	if err != nil {
		s.logger.Error("failed to capture members", "error", err)
		return []MemberSnapshot{}
	}
	defer rows.Close()

	var members []MemberSnapshot
	for rows.Next() {
		var member MemberSnapshot
		err := rows.Scan(
			&member.ID, &member.UserID, &member.RoleID, &member.RoleName,
			&member.Status, &member.InvitedAt, &member.JoinedAt, &member.InvitedBy,
		)
		if err != nil {
			s.logger.Error("failed to scan member", "error", err)
			continue
		}
		members = append(members, member)
	}

	return members
}

// captureRoles captures all workspace roles
func (s *WorkspaceVersionService) captureRoles(ctx context.Context, workspaceID uuid.UUID) []RoleSnapshot {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, display_name, description, color, icon,
		       hierarchy_level, is_system, is_default, permissions
		FROM workspace_roles
		WHERE workspace_id = $1
		ORDER BY hierarchy_level DESC
	`, workspaceID)

	if err != nil {
		s.logger.Error("failed to capture roles", "error", err)
		return []RoleSnapshot{}
	}
	defer rows.Close()

	var roles []RoleSnapshot
	for rows.Next() {
		var role RoleSnapshot
		var permissions json.RawMessage

		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.Color, &role.Icon, &role.HierarchyLevel,
			&role.IsSystem, &role.IsDefault, &permissions,
		)
		if err != nil {
			s.logger.Error("failed to scan role", "error", err)
			continue
		}

		// Unmarshal permissions
		if len(permissions) > 0 {
			json.Unmarshal(permissions, &role.Permissions)
		}

		roles = append(roles, role)
	}

	return roles
}

// captureSettings captures workspace settings
func (s *WorkspaceVersionService) captureSettings(ctx context.Context, workspaceID uuid.UUID) map[string]interface{} {
	var settings json.RawMessage
	err := s.pool.QueryRow(ctx, `
		SELECT settings FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&settings)

	if err != nil {
		s.logger.Error("failed to capture settings", "error", err)
		return map[string]interface{}{}
	}

	var result map[string]interface{}
	if len(settings) > 0 {
		json.Unmarshal(settings, &result)
	}

	return result
}

// captureMemories captures workspace memories
func (s *WorkspaceVersionService) captureMemories(ctx context.Context, workspaceID uuid.UUID) []MemorySnapshot {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       scope_type, scope_id, visibility, created_by, importance_score,
		       tags, source, metadata, is_pinned, is_active, is_archived
		FROM workspace_memories
		WHERE workspace_id = $1 AND is_active = TRUE AND is_archived = FALSE
		ORDER BY importance_score DESC
	`, workspaceID)

	if err != nil {
		s.logger.Error("failed to capture memories", "error", err)
		return []MemorySnapshot{}
	}
	defer rows.Close()

	var memories []MemorySnapshot
	for rows.Next() {
		var memory MemorySnapshot
		var metadata json.RawMessage

		err := rows.Scan(
			&memory.ID, &memory.UserID, &memory.Title, &memory.Summary,
			&memory.Content, &memory.MemoryType, &memory.Category,
			&memory.ScopeType, &memory.ScopeID, &memory.Visibility,
			&memory.CreatedBy, &memory.ImportanceScore, &memory.Tags,
			&memory.Source, &metadata, &memory.IsPinned,
			&memory.IsActive, &memory.IsArchived,
		)
		if err != nil {
			s.logger.Error("failed to scan memory", "error", err)
			continue
		}

		// Unmarshal metadata
		if len(metadata) > 0 {
			json.Unmarshal(metadata, &memory.Metadata)
		}

		memories = append(memories, memory)
	}

	return memories
}
