package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceVersionService handles version snapshots for workspaces
type WorkspaceVersionService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewWorkspaceVersionService creates a new version service
func NewWorkspaceVersionService(pool *pgxpool.Pool, logger *slog.Logger) *WorkspaceVersionService {
	return &WorkspaceVersionService{
		pool:   pool,
		logger: logger,
	}
}

// CreateSnapshot captures the current workspace state as a new version
func (s *WorkspaceVersionService) CreateSnapshot(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID string,
) (string, error) {
	// Get next version number
	var lastVersion *string
	err := s.pool.QueryRow(ctx, `
		SELECT version_number
		FROM workspace_versions
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, workspaceID).Scan(&lastVersion)

	if err != nil && err != pgx.ErrNoRows {
		return "", fmt.Errorf("fetch last version: %w", err)
	}

	nextVersion := incrementVersion(lastVersion)

	// Capture snapshot data
	snapshotData := s.captureWorkspaceState(ctx, workspaceID)

	// Parse to extract metadata
	var snapshot WorkspaceSnapshot
	json.Unmarshal(snapshotData, &snapshot)

	// Create metadata JSON
	metadataJSON, _ := json.Marshal(snapshot.Metadata)

	// Save snapshot
	_, err = s.pool.Exec(ctx, `
		INSERT INTO workspace_versions (workspace_id, version_number, snapshot_data, snapshot_metadata, created_by)
		VALUES ($1, $2, $3, $4, $5)
	`, workspaceID, nextVersion, snapshotData, metadataJSON, userID)

	if err != nil {
		return "", fmt.Errorf("save snapshot: %w", err)
	}

	s.logger.Info("workspace snapshot created",
		"workspace_id", workspaceID,
		"version", nextVersion)
	return nextVersion, nil
}

// RestoreSnapshot restores a workspace to a specific version
func (s *WorkspaceVersionService) RestoreSnapshot(
	ctx context.Context,
	workspaceID uuid.UUID,
	versionNumber string,
	userID string,
) error {
	// Fetch snapshot
	var snapshotData json.RawMessage
	err := s.pool.QueryRow(ctx, `
		SELECT snapshot_data
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, versionNumber).Scan(&snapshotData)

	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}

	// Parse snapshot
	var snapshot WorkspaceSnapshot
	if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
		return fmt.Errorf("invalid snapshot data: %w", err)
	}

	// Create backup of current state BEFORE restoring
	backupVersion, err := s.CreateSnapshot(ctx, workspaceID, userID)
	if err != nil {
		s.logger.Error("failed to create backup before restore", "error", err)
		// Continue anyway - restore is more important than backup failure
	} else {
		s.logger.Info("backup created before restore", "backup_version", backupVersion)
	}

	// Start transaction for atomic restore
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Restore workspace settings
	if err := s.restoreSettings(ctx, tx, workspaceID, snapshot.Settings); err != nil {
		return fmt.Errorf("restore settings: %w", err)
	}

	// Restore roles (must be before members as members reference roles)
	if err := s.restoreRoles(ctx, tx, workspaceID, snapshot.Roles); err != nil {
		return fmt.Errorf("restore roles: %w", err)
	}

	// Restore members
	if err := s.restoreMembers(ctx, tx, workspaceID, snapshot.Members); err != nil {
		return fmt.Errorf("restore members: %w", err)
	}

	// Restore apps
	if err := s.restoreApps(ctx, tx, workspaceID, snapshot.Apps); err != nil {
		return fmt.Errorf("restore apps: %w", err)
	}

	// Restore memories
	if err := s.restoreMemories(ctx, tx, workspaceID, snapshot.Memories); err != nil {
		return fmt.Errorf("restore memories: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("workspace restored successfully",
		"workspace_id", workspaceID,
		"version", versionNumber,
		"apps_restored", len(snapshot.Apps),
		"members_restored", len(snapshot.Members),
		"backup_version", backupVersion)

	return nil
}

// PreviewRestore previews changes that would be made by restoring a version (dry run)
func (s *WorkspaceVersionService) PreviewRestore(
	ctx context.Context,
	workspaceID uuid.UUID,
	versionNumber string,
) (map[string]interface{}, error) {
	// Fetch snapshot
	var snapshotData json.RawMessage
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT snapshot_data, created_at
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, versionNumber).Scan(&snapshotData, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Parse snapshot
	var snapshot WorkspaceSnapshot
	if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
		return nil, fmt.Errorf("invalid snapshot data: %w", err)
	}

	// Get current workspace state for comparison
	var currentName, currentSlug string
	var currentSettings json.RawMessage
	err = s.pool.QueryRow(ctx, `
		SELECT name, slug, settings
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(&currentName, &currentSlug, &currentSettings)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch current workspace: %w", err)
	}

	// Count current entities
	var currentRoleCount, currentMemberCount, currentAppCount int
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspace_roles WHERE workspace_id = $1", workspaceID).Scan(&currentRoleCount)
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1", workspaceID).Scan(&currentMemberCount)
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM osa_module_instances WHERE workspace_id = $1", workspaceID).Scan(&currentAppCount)

	// Build preview response
	preview := map[string]interface{}{
		"snapshot_info": map[string]interface{}{
			"version":    versionNumber,
			"created_at": createdAt,
		},
		"current_state": map[string]interface{}{
			"name":         currentName,
			"slug":         currentSlug,
			"role_count":   currentRoleCount,
			"member_count": currentMemberCount,
			"app_count":    currentAppCount,
		},
		"snapshot_state": map[string]interface{}{
			"role_count":   snapshot.Metadata.RoleCount,
			"member_count": snapshot.Metadata.MemberCount,
			"app_count":    snapshot.Metadata.AppCount,
			"memory_count": snapshot.Metadata.MemoryCount,
		},
		"changes": map[string]interface{}{
			"roles_diff":   snapshot.Metadata.RoleCount - currentRoleCount,
			"members_diff": snapshot.Metadata.MemberCount - currentMemberCount,
			"apps_diff":    snapshot.Metadata.AppCount - currentAppCount,
		},
		"details": map[string]interface{}{
			"roles_to_restore":    len(snapshot.Roles),
			"members_to_restore":  len(snapshot.Members),
			"apps_to_restore":     len(snapshot.Apps),
			"memories_to_restore": len(snapshot.Memories),
		},
		"warnings": []string{
			"This operation will create a backup of the current state",
			"All current workspace data will be replaced with snapshot data",
			"This action cannot be undone except by restoring another version",
		},
	}

	return preview, nil
}

// ListVersions returns all versions for a workspace
func (s *WorkspaceVersionService) ListVersions(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, version_number, created_by, created_at, snapshot_metadata
		FROM workspace_versions
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`, workspaceID)

	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var versionNumber string
		var createdBy *string
		var createdAt time.Time
		var metadata *json.RawMessage

		err := rows.Scan(&id, &versionNumber, &createdBy, &createdAt, &metadata)
		if err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}

		version := map[string]interface{}{
			"id":                id,
			"version_number":    versionNumber,
			"created_by":        createdBy,
			"created_at":        createdAt,
			"snapshot_metadata": metadata,
		}
		versions = append(versions, version)
	}

	return versions, nil
}
