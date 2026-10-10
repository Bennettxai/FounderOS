package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// SyncUser synchronizes a user's data to OSA
// This operation is idempotent and can be safely retried
func (s *OSASyncService) SyncUser(ctx context.Context, userID uuid.UUID) error {
	s.logger.InfoContext(ctx, "syncing user to OSA", "user_id", userID)

	// Start transaction for atomic sync status update
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Get or create sync status
	syncStatus, err := s.getOrCreateSyncStatus(ctx, qtx, "user", userID)
	if err != nil {
		return fmt.Errorf("get sync status: %w", err)
	}

	// Skip if already synced recently (within last 5 minutes)
	if syncStatus.SyncStatus != nil && *syncStatus.SyncStatus == "synced" && syncStatus.LastSyncAt.Valid {
		lastSync := syncStatus.LastSyncAt.Time
		if time.Since(lastSync) < 5*time.Minute {
			s.logger.DebugContext(ctx, "user recently synced, skipping", "user_id", userID, "last_sync", lastSync)
			return nil
		}
	}

	// Fetch user data
	userData, err := s.fetchUserData(ctx, tx, userID)
	if err != nil {
		return s.handleSyncError(ctx, qtx, "user", userID, err)
	}

	// Create outbox event for reliable publishing
	if err := s.createOutboxEvent(ctx, qtx, "user", userID, "user.sync", userData); err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}

	// Update sync status
	syncedStatus := "synced"
	_, err = qtx.UpdateOSASyncStatus(ctx, sqlc.UpdateOSASyncStatusParams{
		EntityType: "user",
		SyncStatus: &syncedStatus,
		LastError:  nil,
		EntityID:   pgTypeUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("update sync status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "user synced successfully", "user_id", userID)
	return nil
}

// SyncWorkspace synchronizes a workspace and its associated data to OSA
// This includes workspace metadata, members, and active modules
func (s *OSASyncService) SyncWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	s.logger.InfoContext(ctx, "syncing workspace to OSA", "workspace_id", workspaceID)

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Get or create sync status
	syncStatus, err := s.getOrCreateSyncStatus(ctx, qtx, "workspace", workspaceID)
	if err != nil {
		return fmt.Errorf("get sync status: %w", err)
	}

	// Skip if already synced recently (within last 10 minutes)
	if syncStatus.SyncStatus != nil && *syncStatus.SyncStatus == "synced" && syncStatus.LastSyncAt.Valid {
		lastSync := syncStatus.LastSyncAt.Time
		if time.Since(lastSync) < 10*time.Minute {
			s.logger.DebugContext(ctx, "workspace recently synced, skipping",
				"workspace_id", workspaceID, "last_sync", lastSync)
			return nil
		}
	}

	// Fetch workspace data including members and settings
	workspaceData, err := s.fetchWorkspaceData(ctx, tx, workspaceID)
	if err != nil {
		return s.handleSyncError(ctx, qtx, "workspace", workspaceID, err)
	}

	// Sync workspace to OSA via OSA client
	if err := s.syncWorkspaceToOSA(ctx, workspaceID, workspaceData); err != nil {
		return s.handleSyncError(ctx, qtx, "workspace", workspaceID, err)
	}

	// Create outbox event for downstream processing
	if err := s.createOutboxEvent(ctx, qtx, "workspace", workspaceID, "workspace.sync", workspaceData); err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}

	// Update sync status
	syncedStatus := "synced"
	_, err = qtx.UpdateOSASyncStatus(ctx, sqlc.UpdateOSASyncStatusParams{
		EntityType: "workspace",
		SyncStatus: &syncedStatus,
		LastError:  nil,
		EntityID:   pgTypeUUID(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("update sync status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "workspace synced successfully", "workspace_id", workspaceID)
	return nil
}

func (s *OSASyncService) fetchUserData(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (map[string]interface{}, error) {
	var userData struct {
		ID        uuid.UUID
		Email     string
		Name      *string
		AvatarURL *string
		CreatedAt time.Time
	}

	err := tx.QueryRow(ctx, `
		SELECT id, email, name, avatar_url, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&userData.ID, &userData.Email, &userData.Name, &userData.AvatarURL, &userData.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	return map[string]interface{}{
		"id":         userData.ID,
		"email":      userData.Email,
		"name":       userData.Name,
		"avatar_url": userData.AvatarURL,
		"created_at": userData.CreatedAt,
	}, nil
}

func (s *OSASyncService) fetchWorkspaceData(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID) (map[string]interface{}, error) {
	var workspace struct {
		ID          uuid.UUID
		Name        string
		Slug        string
		Description *string
		OwnerID     string
		CreatedAt   time.Time
	}

	err := tx.QueryRow(ctx, `
		SELECT id, name, slug, description, owner_id, created_at
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(&workspace.ID, &workspace.Name, &workspace.Slug,
		&workspace.Description, &workspace.OwnerID, &workspace.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("fetch workspace: %w", err)
	}

	// Fetch member count
	var memberCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND status = 'active'
	`, workspaceID).Scan(&memberCount)

	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch member count", "error", err)
		memberCount = 0
	}

	return map[string]interface{}{
		"id":           workspace.ID,
		"name":         workspace.Name,
		"slug":         workspace.Slug,
		"description":  workspace.Description,
		"owner_id":     workspace.OwnerID,
		"member_count": memberCount,
		"created_at":   workspace.CreatedAt,
	}, nil
}

func (s *OSASyncService) syncWorkspaceToOSA(ctx context.Context, workspaceID uuid.UUID, data map[string]interface{}) error {
	// Check if workspace already exists in OSA
	userIDStr, ok := data["owner_id"].(string)
	if !ok {
		return fmt.Errorf("invalid owner_id type")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("parse owner_id: %w", err)
	}

	// Try to get existing workspaces
	workspaces, err := s.osaClient.GetWorkspaces(ctx, userID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get OSA workspaces", "error", err)
		// Continue anyway - workspace might not exist yet
	}

	// Check if workspace already synced
	if workspaces != nil {
		for _, ws := range workspaces.Workspaces {
			if ws.ID == workspaceID.String() {
				s.logger.DebugContext(ctx, "workspace already exists in OSA", "workspace_id", workspaceID)
				return nil
			}
		}
	}

	// Note: OSA client doesn't have a direct CreateWorkspace method yet
	// For now, we log this and rely on the outbox pattern
	s.logger.InfoContext(ctx, "workspace queued for OSA sync", "workspace_id", workspaceID)
	return nil
}
