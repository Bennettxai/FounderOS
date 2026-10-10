package services

import (
	"context"
	"fmt"
)

// CreateShare creates a module share
func (s *CustomModuleService) CreateShare(
	ctx context.Context,
	req CreateShareRequest,
) (*ModuleShare, error) {
	query := `
		INSERT INTO module_shares (
			module_id, shared_with_user_id, shared_with_workspace_id, shared_with_email,
			can_view, can_install, can_modify, can_reshare, shared_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id, module_id, shared_with_user_id, shared_with_workspace_id, shared_with_email,
			can_view, can_install, can_modify, can_reshare, shared_by, shared_at, expires_at
	`

	share := &ModuleShare{}
	err := s.pool.QueryRow(ctx, query,
		req.ModuleID,
		req.SharedWithUserID,
		req.SharedWithWorkspaceID,
		req.SharedWithEmail,
		req.CanView,
		req.CanInstall,
		req.CanModify,
		req.CanReshare,
		req.SharedBy,
	).Scan(
		&share.ID,
		&share.ModuleID,
		&share.SharedWithUserID,
		&share.SharedWithWorkspaceID,
		&share.SharedWithEmail,
		&share.CanView,
		&share.CanInstall,
		&share.CanModify,
		&share.CanReshare,
		&share.SharedBy,
		&share.SharedAt,
		&share.ExpiresAt,
	)

	if err != nil {
		s.logger.Error("Failed to create share", "error", err)
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	s.logger.Info("Module shared", "module_id", req.ModuleID)
	return share, nil
}
