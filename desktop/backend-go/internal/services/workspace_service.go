package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =====================================================================
// TYPES
// =====================================================================

type WorkspaceService struct {
	pool *pgxpool.Pool
}

type Workspace struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  *string                `json:"description,omitempty"`
	LogoURL      *string                `json:"logo_url,omitempty"`
	PlanType     string                 `json:"plan_type"`
	MaxMembers   int                    `json:"max_members"`
	MaxProjects  int                    `json:"max_projects"`
	MaxStorageGB int                    `json:"max_storage_gb"`
	Settings     map[string]interface{} `json:"settings"`
	OwnerID      string                 `json:"owner_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type WorkspaceMember struct {
	ID                uuid.UUID              `json:"id"`
	WorkspaceID       uuid.UUID              `json:"workspace_id"`
	UserID            string                 `json:"user_id"`
	Role              string                 `json:"role"`
	Status            string                 `json:"status"`
	InvitedBy         *string                `json:"invited_by,omitempty"`
	InvitedAt         *time.Time             `json:"invited_at,omitempty"`
	JoinedAt          *time.Time             `json:"joined_at,omitempty"`
	CustomPermissions map[string]interface{} `json:"custom_permissions,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	// Enriched from the "user" table for display in admin UIs.
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Image *string `json:"image,omitempty"`
}

type WorkspaceRole struct {
	ID             uuid.UUID              `json:"id"`
	WorkspaceID    uuid.UUID              `json:"workspace_id"`
	Name           string                 `json:"name"`
	DisplayName    string                 `json:"display_name"`
	Description    *string                `json:"description,omitempty"`
	Color          *string                `json:"color,omitempty"`
	Icon           *string                `json:"icon,omitempty"`
	Permissions    map[string]interface{} `json:"permissions"`
	IsSystem       bool                   `json:"is_system"`
	IsDefault      bool                   `json:"is_default"`
	HierarchyLevel int                    `json:"hierarchy_level"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type CreateWorkspaceRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug,omitempty"` // Optional, auto-generated if not provided
	Description *string                `json:"description,omitempty"`
	LogoURL     *string                `json:"logo_url,omitempty"`
	PlanType    string                 `json:"plan_type,omitempty"` // Default: "free"
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	LogoURL     *string                `json:"logo_url,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role,omitempty"` // Default: "member"
}

// =====================================================================
// CONSTRUCTOR
// =====================================================================

func NewWorkspaceService(pool *pgxpool.Pool) *WorkspaceService {
	return &WorkspaceService{pool: pool}
}

// EnsureDefaultWorkspace synchronously guarantees that a user has one usable
// workspace with canonical roles and an owner membership. It is safe to call
// after every authentication because the advisory lock and membership recheck
// make concurrent email/OAuth callbacks idempotent.
func (s *WorkspaceService) EnsureDefaultWorkspace(ctx context.Context, ownerID string) (*Workspace, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, fmt.Errorf("owner ID is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin default workspace transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, ownerID); err != nil {
		return nil, fmt.Errorf("lock default workspace provisioning: %w", err)
	}

	created := false
	workspace, err := scanWorkspace(tx.QueryRow(ctx, `
		SELECT w.id, w.name, w.slug, w.description, w.logo_url, w.plan_type,
		       w.max_members, w.max_projects, w.max_storage_gb, w.settings,
		       w.owner_id, w.created_at, w.updated_at
		FROM workspace_members wm
		JOIN workspaces w ON w.id = wm.workspace_id
		WHERE wm.user_id = $1 AND wm.status = 'active'
		ORDER BY CASE WHEN wm.role = 'owner' THEN 0 ELSE 1 END, wm.joined_at NULLS LAST
		LIMIT 1
	`, ownerID))
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("find default workspace: %w", err)
	}

	if err == pgx.ErrNoRows {
		created = true
		digest := sha256.Sum256([]byte(ownerID))
		slug := fmt.Sprintf("workspace-%x", digest[:8])
		workspace, err = scanWorkspace(tx.QueryRow(ctx, `
			INSERT INTO workspaces (
				name, slug, description, plan_type, max_members, max_projects,
				max_storage_gb, settings, owner_id
			)
			VALUES (
				'My Workspace', $1, 'A clean workspace ready to become your business operating system',
				'free', 5, 10, 5, '{"module_profile":"primitive"}'::jsonb, $2
			)
			ON CONFLICT (slug) DO UPDATE
			SET owner_id = EXCLUDED.owner_id, updated_at = NOW()
			RETURNING id, name, slug, description, logo_url, plan_type,
			          max_members, max_projects, max_storage_gb, settings,
			          owner_id, created_at, updated_at
		`, slug, ownerID))
		if err != nil {
			return nil, fmt.Errorf("create default workspace: %w", err)
		}

		if _, err := tx.Exec(ctx, `SELECT seed_default_workspace_roles($1)`, workspace.ID); err != nil {
			return nil, fmt.Errorf("seed default workspace roles: %w", err)
		}
	}

	if created {
		if _, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (
				workspace_id, user_id, role_id, role_name, role, status, joined_at
			)
			SELECT $1, $2, wr.id, 'owner', 'owner', 'active', NOW()
			FROM workspace_roles wr
			WHERE wr.workspace_id = $1 AND wr.name = 'owner'
			ON CONFLICT (workspace_id, user_id) DO NOTHING
		`, workspace.ID, ownerID); err != nil {
			return nil, fmt.Errorf("create default workspace owner: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE workspace_members wm
			SET role_id = wr.id,
			    role_name = wr.name,
			    role = wr.name,
			    updated_at = NOW()
			FROM workspace_roles wr
			WHERE wm.workspace_id = $1
			  AND wm.user_id = $2
			  AND wr.workspace_id = wm.workspace_id
			  AND wr.name = COALESCE(NULLIF(wm.role_name, ''), NULLIF(wm.role, ''), 'member')
		`, workspace.ID, ownerID); err != nil {
			return nil, fmt.Errorf("repair workspace membership role: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO sync_policies (workspace_id, module, sync_mode, is_published)
		SELECT $1, module, 'workspace', TRUE
		FROM (VALUES ('knowledge'), ('contexts'), ('projects'), ('tasks')) AS modules(module)
		ON CONFLICT (workspace_id, module) DO NOTHING
	`, workspace.ID); err != nil {
		return nil, fmt.Errorf("seed default workspace sync policies: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit default workspace transaction: %w", err)
	}
	return workspace, nil
}

func scanWorkspace(row pgx.Row) (*Workspace, error) {
	var workspace Workspace
	err := row.Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description,
		&workspace.LogoURL, &workspace.PlanType, &workspace.MaxMembers,
		&workspace.MaxProjects, &workspace.MaxStorageGB, &workspace.Settings,
		&workspace.OwnerID, &workspace.CreatedAt, &workspace.UpdatedAt,
	)
	return &workspace, err
}

// =====================================================================
// WORKSPACE CRUD
// =====================================================================

// CreateWorkspace creates a new workspace with default roles and adds the owner
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest, ownerID string) (*Workspace, error) {
	// Validate name
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	} else {
		// Validate provided slug
		if !isValidSlug(slug) {
			return nil, fmt.Errorf("invalid slug: must be lowercase letters, numbers, and hyphens only")
		}
	}

	// SECURITY (HIGH-15): Never trust plan_type from user input — it must be
	// determined server-side via the billing flow. Always default to "free".
	// Upgrades are handled exclusively through the billing/subscription service.
	planType := "free"

	// Set limits based on plan type
	maxMembers, maxProjects, maxStorageGB := getPlanLimits(planType)

	// Default settings
	settings := req.Settings
	if settings == nil {
		settings = make(map[string]interface{})
	}
	if _, configured := settings["module_profile"]; !configured {
		settings["module_profile"] = "primitive"
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create workspace
	var workspace Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, description, logo_url, plan_type, max_members, max_projects, max_storage_gb, settings, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, slug, description, logo_url, plan_type, max_members, max_projects, max_storage_gb, settings, owner_id, created_at, updated_at
	`, req.Name, slug, req.Description, req.LogoURL, planType, maxMembers, maxProjects, maxStorageGB, settings, ownerID).Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.LogoURL,
		&workspace.PlanType, &workspace.MaxMembers, &workspace.MaxProjects, &workspace.MaxStorageGB,
		&workspace.Settings, &workspace.OwnerID, &workspace.CreatedAt, &workspace.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Seed default roles
	_, err = tx.Exec(ctx, "SELECT seed_default_workspace_roles($1)", workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("seed default roles: %w", err)
	}

	// BusinessOSSync: seed canonical module sync policies so shared modules sync
	// to the cloud out of the box (mirrors migration 101's backfill for new
	// workspaces). Without this, write-side workspace_id stamping silently no-ops.
	_, err = tx.Exec(ctx, `
		INSERT INTO sync_policies (workspace_id, module, sync_mode, is_published)
		SELECT $1, m.module, 'workspace', TRUE
		FROM (VALUES ('knowledge'), ('contexts'), ('projects'), ('tasks')) AS m(module)
		ON CONFLICT (workspace_id, module) DO NOTHING
	`, workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("seed sync policies: %w", err)
	}

	// Add owner as first member
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role_id, role_name, role, status, joined_at)
		SELECT $1, $2, wr.id, 'owner', 'owner', 'active', NOW()
		FROM workspace_roles wr
		WHERE wr.workspace_id = $1 AND wr.name = 'owner'
	`, workspace.ID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("add owner as member: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &workspace, nil
}

// GetWorkspace retrieves a workspace by ID
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Workspace, error) {
	var workspace Workspace
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, logo_url, plan_type, max_members, max_projects, max_storage_gb, settings, owner_id, created_at, updated_at
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.LogoURL,
		&workspace.PlanType, &workspace.MaxMembers, &workspace.MaxProjects, &workspace.MaxStorageGB,
		&workspace.Settings, &workspace.OwnerID, &workspace.CreatedAt, &workspace.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("workspace not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return &workspace, nil
}

// GetWorkspaceBySlug retrieves a workspace by slug
func (s *WorkspaceService) GetWorkspaceBySlug(ctx context.Context, slug string) (*Workspace, error) {
	var workspace Workspace
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, logo_url, plan_type, max_members, max_projects, max_storage_gb, settings, owner_id, created_at, updated_at
		FROM workspaces
		WHERE slug = $1
	`, slug).Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.LogoURL,
		&workspace.PlanType, &workspace.MaxMembers, &workspace.MaxProjects, &workspace.MaxStorageGB,
		&workspace.Settings, &workspace.OwnerID, &workspace.CreatedAt, &workspace.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("workspace not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return &workspace, nil
}

// ListUserWorkspaces lists all workspaces a user is a member of
func (s *WorkspaceService) ListUserWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT w.id, w.name, w.slug, w.description, w.logo_url, w.plan_type,
		       w.max_members, w.max_projects, w.max_storage_gb, w.settings, w.owner_id,
		       w.created_at, w.updated_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE wm.user_id = $1 AND wm.status = 'active'
		ORDER BY w.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		var w Workspace
		err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.Description, &w.LogoURL, &w.PlanType,
			&w.MaxMembers, &w.MaxProjects, &w.MaxStorageGB, &w.Settings, &w.OwnerID,
			&w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		workspaces = append(workspaces, w)
	}

	return workspaces, nil
}

// UpdateWorkspace updates a workspace
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, workspaceID uuid.UUID, req UpdateWorkspaceRequest) (*Workspace, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{workspaceID}
	argIdx := 2

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.LogoURL != nil {
		updates = append(updates, fmt.Sprintf("logo_url = $%d", argIdx))
		args = append(args, *req.LogoURL)
		argIdx++
	}
	if req.Settings != nil {
		updates = append(updates, fmt.Sprintf("settings = $%d", argIdx))
		args = append(args, req.Settings)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetWorkspace(ctx, workspaceID)
	}

	query := fmt.Sprintf(`
		UPDATE workspaces
		SET %s, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, description, logo_url, plan_type, max_members, max_projects, max_storage_gb, settings, owner_id, created_at, updated_at
	`, strings.Join(updates, ", "))

	var workspace Workspace
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.LogoURL,
		&workspace.PlanType, &workspace.MaxMembers, &workspace.MaxProjects, &workspace.MaxStorageGB,
		&workspace.Settings, &workspace.OwnerID, &workspace.CreatedAt, &workspace.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update workspace: %w", err)
	}

	return &workspace, nil
}

// DeleteWorkspace deletes a workspace (only owner can do this)
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID, userID string) error {
	// Verify user is owner
	var ownerID string
	err := s.pool.QueryRow(ctx, "SELECT owner_id FROM workspaces WHERE id = $1", workspaceID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("workspace not found")
	}
	if err != nil {
		return fmt.Errorf("check ownership: %w", err)
	}

	if ownerID != userID {
		return fmt.Errorf("only workspace owner can delete workspace")
	}

	// Delete workspace (cascade will delete members, roles, etc.)
	_, err = s.pool.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", workspaceID)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}

	return nil
}

// =====================================================================
// MEMBER MANAGEMENT
// =====================================================================

// AddMember adds a user to a workspace
func (s *WorkspaceService) AddMember(ctx context.Context, workspaceID uuid.UUID, req AddMemberRequest, invitedBy string) (*WorkspaceMember, error) {
	role := req.Role
	if role == "" {
		role = "member" // Default role
	}

	// Verify role exists
	var roleExists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_roles WHERE workspace_id = $1 AND name = $2)", workspaceID, role).Scan(&roleExists)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !roleExists {
		return nil, fmt.Errorf("role '%s' does not exist in this workspace", role)
	}

	// Check member limit
	var currentMembers int
	var maxMembers int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*), w.max_members
		FROM workspace_members wm
		JOIN workspaces w ON w.id = wm.workspace_id
		WHERE wm.workspace_id = $1 AND wm.status = 'active'
		GROUP BY w.max_members
	`, workspaceID).Scan(&currentMembers, &maxMembers)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("check member limit: %w", err)
	}

	if currentMembers >= maxMembers {
		return nil, fmt.Errorf("workspace has reached maximum member limit (%d)", maxMembers)
	}

	// Add member
	var member WorkspaceMember
	now := time.Now()
	err = s.pool.QueryRow(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, status, invited_by, invited_at, joined_at)
		VALUES ($1, $2, $3, 'active', $4, $5, $5)
		RETURNING id, workspace_id, user_id, role, status, invited_by, invited_at, joined_at, custom_permissions, created_at, updated_at
	`, workspaceID, req.UserID, role, invitedBy, now).Scan(
		&member.ID, &member.WorkspaceID, &member.UserID, &member.Role, &member.Status,
		&member.InvitedBy, &member.InvitedAt, &member.JoinedAt, &member.CustomPermissions,
		&member.CreatedAt, &member.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return &member, nil
}

// RemoveMember removes a user from a workspace
func (s *WorkspaceService) RemoveMember(ctx context.Context, workspaceID uuid.UUID, userID string) error {
	// Don't allow removing the owner
	var ownerID string
	err := s.pool.QueryRow(ctx, "SELECT owner_id FROM workspaces WHERE id = $1", workspaceID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("check owner: %w", err)
	}
	if ownerID == userID {
		return fmt.Errorf("cannot remove workspace owner")
	}

	result, err := s.pool.Exec(ctx, "DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2", workspaceID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// UpdateMemberRole updates a member's role
func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, workspaceID uuid.UUID, userID string, newRole string) (*WorkspaceMember, error) {
	// Don't allow changing owner role
	var ownerID string
	err := s.pool.QueryRow(ctx, "SELECT owner_id FROM workspaces WHERE id = $1", workspaceID).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("check owner: %w", err)
	}
	if ownerID == userID {
		return nil, fmt.Errorf("cannot change owner role")
	}

	// Verify new role exists
	var roleExists bool
	err = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_roles WHERE workspace_id = $1 AND name = $2)", workspaceID, newRole).Scan(&roleExists)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !roleExists {
		return nil, fmt.Errorf("role '%s' does not exist", newRole)
	}

	// Update role
	var member WorkspaceMember
	err = s.pool.QueryRow(ctx, `
		UPDATE workspace_members
		SET role = $1, updated_at = NOW()
		WHERE workspace_id = $2 AND user_id = $3
		RETURNING id, workspace_id, user_id, role, status, invited_by, invited_at, joined_at, custom_permissions, created_at, updated_at
	`, newRole, workspaceID, userID).Scan(
		&member.ID, &member.WorkspaceID, &member.UserID, &member.Role, &member.Status,
		&member.InvitedBy, &member.InvitedAt, &member.JoinedAt, &member.CustomPermissions,
		&member.CreatedAt, &member.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	return &member, nil
}

// ListMembers lists all members of a workspace
func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT wm.id, wm.workspace_id, wm.user_id, wm.role, wm.status, wm.invited_by,
		       wm.invited_at, wm.joined_at, wm.custom_permissions, wm.created_at, wm.updated_at,
		       u.name, u.email, u.image
		FROM workspace_members wm
		LEFT JOIN "user" u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
		ORDER BY wm.joined_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []WorkspaceMember
	for rows.Next() {
		var m WorkspaceMember
		err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.Status,
			&m.InvitedBy, &m.InvitedAt, &m.JoinedAt, &m.CustomPermissions,
			&m.CreatedAt, &m.UpdatedAt, &m.Name, &m.Email, &m.Image)
		if err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}

	return members, nil
}

// GetUserRole gets a user's role in a workspace
func (s *WorkspaceService) GetUserRole(ctx context.Context, workspaceID uuid.UUID, userID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
	`, workspaceID, userID).Scan(&role)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("user is not a member of this workspace")
	}
	if err != nil {
		return "", fmt.Errorf("get user role: %w", err)
	}
	return role, nil
}

// =====================================================================
// ROLE MANAGEMENT
// =====================================================================

// ListRoles lists all roles in a workspace
func (s *WorkspaceService) ListRoles(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceRole, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, name, display_name, description, color, icon, permissions, is_system, is_default, hierarchy_level, created_at, updated_at
		FROM workspace_roles
		WHERE workspace_id = $1
		ORDER BY hierarchy_level ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []WorkspaceRole
	for rows.Next() {
		var r WorkspaceRole
		err := rows.Scan(&r.ID, &r.WorkspaceID, &r.Name, &r.DisplayName, &r.Description,
			&r.Color, &r.Icon, &r.Permissions, &r.IsSystem, &r.IsDefault,
			&r.HierarchyLevel, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, r)
	}

	return roles, nil
}

// =====================================================================
// HELPER FUNCTIONS
// =====================================================================

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")
	// Remove consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	// Trim hyphens from edges
	slug = strings.Trim(slug, "-")
	return slug
}

// isValidSlug checks if a slug is valid
func isValidSlug(slug string) bool {
	match, _ := regexp.MatchString("^[a-z0-9-]+$", slug)
	return match
}

// getPlanLimits returns the limits for a plan type
func getPlanLimits(planType string) (maxMembers, maxProjects, maxStorageGB int) {
	switch planType {
	case "free":
		return 5, 10, 5
	case "starter":
		return 15, 50, 50
	case "professional":
		return 50, 200, 200
	case "enterprise":
		return 999, 9999, 1000
	default:
		return 5, 10, 5
	}
}
