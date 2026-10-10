package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceInviteService handles workspace invitations via email
type WorkspaceInviteService struct {
	pool *pgxpool.Pool
}

// WorkspaceInvite represents an invitation to join a workspace
type WorkspaceInvite struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	InvitedBy   string     `json:"invited_by"`
	Token       string     `json:"token"`
	Status      string     `json:"status"` // pending, accepted, expired, revoked
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewWorkspaceInviteService creates a new workspace invite service
func NewWorkspaceInviteService(pool *pgxpool.Pool) *WorkspaceInviteService {
	return &WorkspaceInviteService{pool: pool}
}

// CreateInvite creates a new workspace invitation
func (s *WorkspaceInviteService) CreateInvite(
	ctx context.Context,
	workspaceID uuid.UUID,
	email string,
	role string,
	invitedBy string,
) (*WorkspaceInvite, error) {
	// Generate secure token
	token := uuid.New().String()

	// Set expiration to 7 days from now
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	invite := &WorkspaceInvite{}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_invites (workspace_id, email, role, invited_by, token, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		RETURNING id, workspace_id, email, role, invited_by, token, status, expires_at, accepted_at, created_at
	`, workspaceID, email, role, invitedBy, token, expiresAt).Scan(
		&invite.ID,
		&invite.WorkspaceID,
		&invite.Email,
		&invite.Role,
		&invite.InvitedBy,
		&invite.Token,
		&invite.Status,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	return invite, nil
}

// GetInviteByToken retrieves an invitation by token
func (s *WorkspaceInviteService) GetInviteByToken(ctx context.Context, token string) (*WorkspaceInvite, error) {
	invite := &WorkspaceInvite{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, email, role, invited_by, token, status, expires_at, accepted_at, created_at
		FROM workspace_invites
		WHERE token = $1
	`, token).Scan(
		&invite.ID,
		&invite.WorkspaceID,
		&invite.Email,
		&invite.Role,
		&invite.InvitedBy,
		&invite.Token,
		&invite.Status,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}

	return invite, nil
}

// AcceptInvite accepts an invitation and adds user to workspace
func (s *WorkspaceInviteService) AcceptInvite(ctx context.Context, token string, userID string) error {
	// Get invite
	invite, err := s.GetInviteByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invite not found: %w", err)
	}

	// Validate invite
	if invite.Status != "pending" {
		return fmt.Errorf("invite already %s", invite.Status)
	}

	if time.Now().After(invite.ExpiresAt) {
		// Mark as expired
		s.pool.Exec(ctx, "UPDATE workspace_invites SET status = 'expired' WHERE id = $1", invite.ID)
		return fmt.Errorf("invite has expired")
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Add user to workspace. ON CONFLICT DO UPDATE: a previously-removed member
	// re-invited with a different role must get the new role, not silently keep
	// the old one. We also mark joined_at fresh so membership timestamps are right.
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, status, invited_by, invited_at, joined_at)
		VALUES ($1, $2, $3, 'active', $4, $5, NOW())
		ON CONFLICT (workspace_id, user_id) DO UPDATE
		  SET role = EXCLUDED.role,
		      status = 'active',
		      joined_at = NOW()
	`, invite.WorkspaceID, userID, invite.Role, invite.InvitedBy, invite.CreatedAt)

	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	// Mark invite as accepted
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE workspace_invites
		SET status = 'accepted', accepted_at = $1
		WHERE id = $2
	`, now, invite.ID)

	if err != nil {
		return fmt.Errorf("update invite: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RevokeInvite revokes a pending invitation.
// SECURITY (CRIT-12): workspaceID is required and scoped in the query to prevent
// an admin of one workspace from revoking invites belonging to another workspace.
func (s *WorkspaceInviteService) RevokeInvite(ctx context.Context, inviteID uuid.UUID, workspaceID uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE workspace_invites
		SET status = 'revoked'
		WHERE id = $1 AND workspace_id = $2 AND status = 'pending'
	`, inviteID, workspaceID)

	if err != nil {
		return fmt.Errorf("revoke invite: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invite not found or already processed")
	}

	return nil
}

// ListWorkspaceInvites lists all invitations for a workspace
func (s *WorkspaceInviteService) ListWorkspaceInvites(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceInvite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, email, role, invited_by, token, status, expires_at, accepted_at, created_at
		FROM workspace_invites
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`, workspaceID)

	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var invites []WorkspaceInvite
	for rows.Next() {
		var invite WorkspaceInvite
		err := rows.Scan(
			&invite.ID,
			&invite.WorkspaceID,
			&invite.Email,
			&invite.Role,
			&invite.InvitedBy,
			&invite.Token,
			&invite.Status,
			&invite.ExpiresAt,
			&invite.AcceptedAt,
			&invite.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		invites = append(invites, invite)
	}

	return invites, nil
}

// CleanupExpiredInvites marks expired invitations
func (s *WorkspaceInviteService) CleanupExpiredInvites(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx, `
		UPDATE workspace_invites
		SET status = 'expired'
		WHERE status = 'pending' AND expires_at < NOW()
	`)

	if err != nil {
		return 0, fmt.Errorf("cleanup expired: %w", err)
	}

	return result.RowsAffected(), nil
}
