package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectAccessService handles project-level access control
type ProjectAccessService struct {
	pool *pgxpool.Pool
}

// ProjectMember represents a project member with role and permissions
type ProjectMember struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	UserID      string    `json:"user_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Role        string    `json:"role"` // lead, contributor, reviewer, viewer
	CanEdit     bool      `json:"can_edit"`
	CanDelete   bool      `json:"can_delete"`
	CanInvite   bool      `json:"can_invite"`
	AssignedBy  string    `json:"assigned_by"`
	Status      string    `json:"status"` // active, inactive, removed
}

// NewProjectAccessService creates a new project access service
func NewProjectAccessService(pool *pgxpool.Pool) *ProjectAccessService {
	return &ProjectAccessService{pool: pool}
}

// AddMember adds a member to a project
func (s *ProjectAccessService) AddMember(ctx context.Context, projectID uuid.UUID, userID, role, assignedBy string, workspaceID uuid.UUID) (*ProjectMember, error) {
	member := &ProjectMember{}

	// Get default permissions for role
	var canEdit, canDelete, canInvite bool
	err := s.pool.QueryRow(ctx, `
		SELECT default_can_edit, default_can_delete, default_can_invite
		FROM project_role_definitions
		WHERE role = $1
	`, role).Scan(&canEdit, &canDelete, &canInvite)

	if err != nil {
		return nil, fmt.Errorf("invalid role: %w", err)
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO project_members (project_id, user_id, workspace_id, role, can_edit, can_delete, can_invite, assigned_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, project_id, user_id, workspace_id, role, can_edit, can_delete, can_invite, assigned_by, status
	`, projectID, userID, workspaceID, role, canEdit, canDelete, canInvite, assignedBy).Scan(
		&member.ID, &member.ProjectID, &member.UserID, &member.WorkspaceID,
		&member.Role, &member.CanEdit, &member.CanDelete, &member.CanInvite,
		&member.AssignedBy, &member.Status,
	)

	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return member, nil
}

// HasAccess checks if user has access to project
func (s *ProjectAccessService) HasAccess(ctx context.Context, userID string, projectID uuid.UUID) (bool, error) {
	var hasAccess bool
	err := s.pool.QueryRow(ctx, "SELECT has_project_access($1, $2)", userID, projectID).Scan(&hasAccess)
	return hasAccess, err
}

// GetRole gets user's role in project
func (s *ProjectAccessService) GetRole(ctx context.Context, userID string, projectID uuid.UUID) (string, error) {
	var role *string
	err := s.pool.QueryRow(ctx, "SELECT get_project_role($1, $2)", userID, projectID).Scan(&role)
	if role == nil {
		return "", fmt.Errorf("user not in project")
	}
	return *role, err
}

// GetPermissions gets user's permissions in project
func (s *ProjectAccessService) GetPermissions(ctx context.Context, userID string, projectID uuid.UUID) (canEdit, canDelete, canInvite bool, role string, err error) {
	err = s.pool.QueryRow(ctx, "SELECT * FROM get_project_permissions($1, $2)", userID, projectID).Scan(&canEdit, &canDelete, &canInvite, &role)
	return
}

// ListMembers lists all members of a project
func (s *ProjectAccessService) ListMembers(ctx context.Context, projectID uuid.UUID) ([]ProjectMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, user_id, workspace_id, role, can_edit, can_delete, can_invite, assigned_by, status
		FROM project_members
		WHERE project_id = $1 AND status = 'active'
		ORDER BY role, user_id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ProjectMember
	for rows.Next() {
		var m ProjectMember
		err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.WorkspaceID, &m.Role, &m.CanEdit, &m.CanDelete, &m.CanInvite, &m.AssignedBy, &m.Status)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// UpdateRole updates a member's role
func (s *ProjectAccessService) UpdateRole(ctx context.Context, memberID uuid.UUID, newRole string) error {
	// Get default permissions for new role
	var canEdit, canDelete, canInvite bool
	err := s.pool.QueryRow(ctx, `
		SELECT default_can_edit, default_can_delete, default_can_invite
		FROM project_role_definitions
		WHERE role = $1
	`, newRole).Scan(&canEdit, &canDelete, &canInvite)

	if err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE project_members
		SET role = $1, can_edit = $2, can_delete = $3, can_invite = $4
		WHERE id = $5
	`, newRole, canEdit, canDelete, canInvite, memberID)

	return err
}

// RemoveMember removes a member from project
func (s *ProjectAccessService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE project_members
		SET status = 'removed', removed_at = NOW()
		WHERE id = $1
	`, memberID)
	return err
}
