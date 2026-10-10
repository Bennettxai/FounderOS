package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRoleContext represents the role-based context for an agent
type UserRoleContext struct {
	UserID      string    `json:"user_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`

	// Role Info
	RoleName        string `json:"role_name"`
	RoleDisplayName string `json:"role_display_name"`
	HierarchyLevel  int    `json:"hierarchy_level"`

	// Permissions
	Permissions map[string]map[string]interface{} `json:"permissions"`

	// Project-specific roles
	ProjectRoles map[uuid.UUID]string `json:"project_roles"` // projectID -> project role

	// Profile
	Title          string   `json:"title"`
	Department     string   `json:"department"`
	ExpertiseAreas []string `json:"expertise_areas"`
}

// RoleContextService manages role-based context for agents
type RoleContextService struct {
	pool *pgxpool.Pool
}

// NewRoleContextService creates a new role context service
func NewRoleContextService(pool *pgxpool.Pool) *RoleContextService {
	return &RoleContextService{
		pool: pool,
	}
}

func applyRolePermissions(roleCtx *UserRoleContext, permissions map[string]interface{}) {
	for resource, rawPermissions := range permissions {
		if roleCtx.Permissions[resource] == nil {
			roleCtx.Permissions[resource] = make(map[string]interface{})
		}

		switch perms := rawPermissions.(type) {
		case []interface{}:
			for _, rawPermission := range perms {
				if permission, ok := rawPermission.(string); ok {
					roleCtx.Permissions[resource][permission] = true
				}
			}
		case []string:
			for _, permission := range perms {
				roleCtx.Permissions[resource][permission] = true
			}
		case map[string]interface{}:
			for permission, value := range perms {
				roleCtx.Permissions[resource][permission] = value
			}
		case string:
			roleCtx.Permissions[resource][perms] = true
		case bool:
			if perms {
				roleCtx.Permissions[resource]["*"] = true
			}
		}
	}
}

// GetUserRoleContext retrieves the complete role context for a user
func (s *RoleContextService) GetUserRoleContext(ctx context.Context, userID string, workspaceID uuid.UUID) (*UserRoleContext, error) {
	roleCtx := &UserRoleContext{
		UserID:         userID,
		WorkspaceID:    workspaceID,
		Permissions:    make(map[string]map[string]interface{}),
		ProjectRoles:   make(map[uuid.UUID]string),
		ExpertiseAreas: make([]string, 0),
	}

	// Get user's workspace role
	var roleName, roleDisplayName string
	var hierarchyLevel int
	var rolePermissions map[string]interface{}
	err := s.pool.QueryRow(ctx, `
		SELECT wm.role, wr.display_name, wr.hierarchy_level, wr.permissions
		FROM workspace_members wm
		JOIN workspace_roles wr ON wr.name = wm.role AND wr.workspace_id = wm.workspace_id
		WHERE wm.user_id = $1 AND wm.workspace_id = $2
	`, userID, workspaceID).Scan(&roleName, &roleDisplayName, &hierarchyLevel, &rolePermissions)

	if err != nil {
		return nil, fmt.Errorf("get workspace role: %w", err)
	}

	roleCtx.RoleName = roleName
	roleCtx.RoleDisplayName = roleDisplayName
	roleCtx.HierarchyLevel = hierarchyLevel

	// workspace_roles.permissions is canonical. role_permissions is a legacy
	// expansion table that may be absent or only partially populated.
	applyRolePermissions(roleCtx, rolePermissions)

	// Overlay expanded rows when available without making them a prerequisite
	// for authorization.
	rows, err := s.pool.Query(ctx, `
		SELECT rp.resource, rp.permission, rp.metadata
		FROM role_permissions rp
		WHERE rp.workspace_id = $1 AND rp.role = $2
	`, workspaceID, roleName)
	if err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
			return nil, fmt.Errorf("get permissions: %w", err)
		}
	} else {
		defer rows.Close()

		for rows.Next() {
			var resource, permission string
			var metadata map[string]interface{}

			if err := rows.Scan(&resource, &permission, &metadata); err != nil {
				continue
			}

			if roleCtx.Permissions[resource] == nil {
				roleCtx.Permissions[resource] = make(map[string]interface{})
			}
			roleCtx.Permissions[resource][permission] = metadata
		}
	}

	// Get user's profile info
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(title, ''), COALESCE(department, '')
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&roleCtx.Title, &roleCtx.Department)
	if err != nil {
		// Profile is optional, just log and continue
		roleCtx.Title = ""
		roleCtx.Department = ""
	}

	// Get user's expertise areas from facts/patterns
	expertiseRows, err := s.pool.Query(ctx, `
		SELECT DISTINCT category
		FROM user_facts
		WHERE user_id = $1 AND category != ''
		LIMIT 10
	`, userID)
	if err == nil {
		defer expertiseRows.Close()
		for expertiseRows.Next() {
			var category string
			if err := expertiseRows.Scan(&category); err == nil {
				roleCtx.ExpertiseAreas = append(roleCtx.ExpertiseAreas, category)
			}
		}
	}

	// Get project-specific roles
	projectRows, err := s.pool.Query(ctx, `
		SELECT project_id, role
		FROM project_members
		WHERE user_id = $1 AND workspace_id = $2
	`, userID, workspaceID)
	if err == nil {
		defer projectRows.Close()
		for projectRows.Next() {
			var projectID uuid.UUID
			var projectRole string
			if err := projectRows.Scan(&projectID, &projectRole); err == nil {
				roleCtx.ProjectRoles[projectID] = projectRole
			}
		}
	}

	return roleCtx, nil
}

// GetRoleContextPrompt generates a formatted prompt for the agent about user's role
func (r *UserRoleContext) GetRoleContextPrompt() string {
	return fmt.Sprintf(`═══════════════════════════════════════════════════════════════════════════════
🔐 CRITICAL: USER ROLE & PERMISSIONS CONTEXT
═══════════════════════════════════════════════════════════════════════════════

You are assisting a user with the following role and permissions. This information is
CRITICAL and MUST be acknowledged in your responses when relevant.

**User:** %s
**Workspace Role:** %s (%s)
**Authority Level:** %d (lower = higher authority)
**Title:** %s
**Department:** %s

╔═══════════════════════════════════════════════════════════════════════════════╗
║ PERMISSIONS GRANTED TO THIS USER                                             ║
╚═══════════════════════════════════════════════════════════════════════════════╝
%s

╔═══════════════════════════════════════════════════════════════════════════════╗
║ ACTIONS RESTRICTED FROM THIS USER                                            ║
╚═══════════════════════════════════════════════════════════════════════════════╝
%s

═══════════════════════════════════════════════════════════════════════════════
🎯 MANDATORY BEHAVIOR:
═══════════════════════════════════════════════════════════════════════════════
1. When the user asks "what can I do?" or similar questions, IMMEDIATELY reference
   their role (%s) and explain their specific permissions listed above.

2. ALWAYS acknowledge their role when providing workspace-related guidance.
   Example: "As the %s of this workspace, you have..."

3. ONLY suggest actions that are within their permission set listed above.

4. If they request something outside their permissions, politely explain:
   "I see you'd like to [action], but this requires [permission/role].
    Your current role (%s) doesn't include this permission."

5. Tailor technical depth and business context to their title (%s) and
   department (%s).

═══════════════════════════════════════════════════════════════════════════════

`,
		r.UserID,
		r.RoleDisplayName,
		r.RoleName,
		r.HierarchyLevel,
		r.Title,
		r.Department,
		r.formatCanDo(),
		r.formatCannotDo(),
		r.RoleDisplayName,
		r.RoleDisplayName,
		r.RoleDisplayName,
		r.Title,
		r.Department,
	)
}

// formatCanDo formats the list of permissions the user has
func (r *UserRoleContext) formatCanDo() string {
	if len(r.Permissions) == 0 {
		return "- No specific permissions configured"
	}

	var lines []string
	for resource, perms := range r.Permissions {
		var permList []string
		for perm := range perms {
			permList = append(permList, perm)
		}
		if len(permList) > 0 {
			lines = append(lines, fmt.Sprintf("- **%s**: %s", resource, strings.Join(permList, ", ")))
		}
	}

	if len(lines) == 0 {
		return "- No specific permissions configured"
	}

	return strings.Join(lines, "\n")
}

// formatCannotDo formats the list of common actions the user cannot do
func (r *UserRoleContext) formatCannotDo() string {
	// Define common restricted actions by role
	restricted := make([]string, 0)

	// Role-based restrictions
	switch strings.ToLower(r.RoleName) {
	case "viewer", "guest":
		restricted = append(restricted,
			"- Create or modify projects",
			"- Delete any resources",
			"- Manage workspace settings",
			"- Invite or remove members",
			"- Modify role permissions",
		)
	case "member", "developer":
		restricted = append(restricted,
			"- Delete workspace",
			"- Manage workspace billing",
			"- Modify role permissions",
			"- Remove workspace owners",
		)
	case "admin", "manager":
		restricted = append(restricted,
			"- Delete workspace (only owner can)",
		)
	case "owner":
		// Owners have full access
		return "- None (full workspace access)"
	}

	// Add restrictions based on missing permissions
	commonResources := []string{"projects", "members", "settings", "billing", "roles"}
	for _, resource := range commonResources {
		if _, hasResource := r.Permissions[resource]; !hasResource {
			restricted = append(restricted, fmt.Sprintf("- Access %s resource", resource))
		}
	}

	if len(restricted) == 0 {
		return "- Not explicitly restricted (depends on role permissions)"
	}

	// Deduplicate and return
	seen := make(map[string]bool)
	var unique []string
	for _, item := range restricted {
		if !seen[item] {
			seen[item] = true
			unique = append(unique, item)
		}
	}

	return strings.Join(unique, "\n")
}

// HasPermission checks if the user has a specific permission on a resource
func (r *UserRoleContext) HasPermission(resource, permission string) bool {
	if resPerms, ok := r.Permissions[resource]; ok {
		permValue, hasPerm := resPerms[permission]
		if !hasPerm {
			return false
		}
		// Check if the permission value is a boolean and true
		if boolVal, ok := permValue.(bool); ok {
			return boolVal
		}
		// If not a boolean, assume true if the key exists
		return true
	}
	return false
}

// GetProjectRole returns the user's role for a specific project
func (r *UserRoleContext) GetProjectRole(projectID uuid.UUID) (string, bool) {
	role, ok := r.ProjectRoles[projectID]
	return role, ok
}

// IsAtLeastLevel checks if user's hierarchy level is at or above the specified level
// Lower numbers = higher hierarchy (e.g., 1 = owner, 2 = admin, 3 = member, 4 = viewer)
func (r *UserRoleContext) IsAtLeastLevel(level int) bool {
	return r.HierarchyLevel <= level
}

// GetExpertiseContext returns a formatted string of user's expertise areas
func (r *UserRoleContext) GetExpertiseContext() string {
	if len(r.ExpertiseAreas) == 0 {
		return "No specific expertise areas identified"
	}

	return fmt.Sprintf("Expertise areas: %s", strings.Join(r.ExpertiseAreas, ", "))
}
