package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// =====================================================================
// ROLE CONTEXT MIDDLEWARE
// =====================================================================

// InjectRoleContext injects the user's role context into the request
// This middleware should be used on workspace-scoped routes
func InjectRoleContext(pool *pgxpool.Pool, roleContextService *services.RoleContextService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		// Get workspace ID from route parameter
		workspaceIDStr := c.Param("id")
		if workspaceIDStr == "" {
			// Try to get from workspaceId param
			workspaceIDStr = c.Param("workspaceId")
		}

		if workspaceIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace ID required"})
			c.Abort()
			return
		}

		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
			c.Abort()
			return
		}

		// Get role context
		roleCtx, err := roleContextService.GetUserRoleContext(c.Request.Context(), user.ID, workspaceID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
			c.Abort()
			return
		}

		// Inject into context
		c.Set("role_context", roleCtx)
		c.Set("workspace_id", workspaceID)

		c.Next()
	}
}

// GetRoleContext retrieves the role context from the request context
func GetRoleContext(c *gin.Context) *services.UserRoleContext {
	roleCtx, exists := c.Get("role_context")
	if !exists {
		return nil
	}
	return roleCtx.(*services.UserRoleContext)
}

// GetWorkspaceID retrieves the workspace ID from the request context
func GetWorkspaceID(c *gin.Context) *uuid.UUID {
	workspaceID, exists := c.Get("workspace_id")
	if !exists {
		return nil
	}
	id := workspaceID.(uuid.UUID)
	return &id
}

// =====================================================================
// PERMISSION CHECK MIDDLEWARES
// =====================================================================

// RequirePermission checks if the user has a specific permission on a resource
func RequirePermission(resource, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		if !roleCtx.HasPermission(resource, permission) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "Permission denied",
				"resource":   resource,
				"permission": permission,
				"your_role":  roleCtx.RoleName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has at least one of the specified permissions
func RequireAnyPermission(checks []PermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		for _, check := range checks {
			if roleCtx.HasPermission(check.Resource, check.Permission) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":     "Permission denied",
			"your_role": roleCtx.RoleName,
		})
		c.Abort()
	}
}

// PermissionCheck represents a resource-permission pair
type PermissionCheck struct {
	Resource   string
	Permission string
}

// RequireAllPermissions checks if the user has all of the specified permissions
func RequireAllPermissions(checks []PermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		for _, check := range checks {
			if !roleCtx.HasPermission(check.Resource, check.Permission) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":      "Permission denied",
					"resource":   check.Resource,
					"permission": check.Permission,
					"your_role":  roleCtx.RoleName,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// =====================================================================
// HIERARCHY LEVEL MIDDLEWARES
// =====================================================================

// RequireHierarchyLevel checks if the user's hierarchy level is at or above the required level
// Lower numbers = higher hierarchy (1 = owner, 2 = admin, etc.)
func RequireHierarchyLevel(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		if !roleCtx.IsAtLeastLevel(minLevel) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient hierarchy level",
				"required_level": minLevel,
				"your_level":     roleCtx.HierarchyLevel,
				"your_role":      roleCtx.RoleName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWorkspaceOwner checks if the user is the workspace owner
func RequireWorkspaceOwner(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		workspaceID := GetWorkspaceID(c)
		if workspaceID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace ID not available"})
			c.Abort()
			return
		}

		// Check if user is owner
		var ownerID string
		err := pool.QueryRow(c.Request.Context(), "SELECT owner_id FROM workspaces WHERE id = $1", workspaceID).Scan(&ownerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
			c.Abort()
			return
		}

		if ownerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only workspace owner can perform this action"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWorkspaceAdmin checks if the user is owner or admin
func RequireWorkspaceAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		// Owner (level 1) or Admin (level 2)
		if !roleCtx.IsAtLeastLevel(2) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":     "Only owners and admins can perform this action",
				"your_role": roleCtx.RoleName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWorkspaceManager checks if the user is owner, admin, or manager
func RequireWorkspaceManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		// Owner (1), Admin (2), or Manager (3)
		if !roleCtx.IsAtLeastLevel(3) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":     "Only owners, admins, and managers can perform this action",
				"your_role": roleCtx.RoleName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// =====================================================================
// ROLE-BASED MIDDLEWARES
// =====================================================================

// RequireRole checks if the user has one of the specified roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCtx := GetRoleContext(c)
		if roleCtx == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role context not available"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if roleCtx.RoleName == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":         "Role not allowed",
			"allowed_roles": allowedRoles,
			"your_role":     roleCtx.RoleName,
		})
		c.Abort()
	}
}

// =====================================================================
// WORKSPACE MEMBERSHIP MIDDLEWARE
// =====================================================================

// RequireWorkspaceMember checks if the user is a member of the workspace
// This is lighter than InjectRoleContext as it just checks membership
func RequireWorkspaceMember(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		workspaceIDStr := c.Param("id")
		if workspaceIDStr == "" {
			workspaceIDStr = c.Param("workspaceId")
		}

		if workspaceIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace ID required"})
			c.Abort()
			return
		}

		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
			c.Abort()
			return
		}

		// Check membership
		var exists bool
		err = pool.QueryRow(c.Request.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM workspace_members
				WHERE workspace_id = $1 AND user_id = $2 AND status = 'active'
			)
		`, workspaceID, user.ID).Scan(&exists)

		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
			c.Abort()
			return
		}

		c.Set("workspace_id", workspaceID)
		c.Next()
	}
}

// =====================================================================
// HELPER FUNCTIONS
// =====================================================================

// CheckPermission checks if a user has a specific permission (non-middleware version)
func CheckPermission(c *gin.Context, resource, permission string) bool {
	roleCtx := GetRoleContext(c)
	if roleCtx == nil {
		return false
	}
	return roleCtx.HasPermission(resource, permission)
}

// CheckHierarchyLevel checks if a user meets the minimum hierarchy level (non-middleware version)
func CheckHierarchyLevel(c *gin.Context, minLevel int) bool {
	roleCtx := GetRoleContext(c)
	if roleCtx == nil {
		return false
	}
	return roleCtx.IsAtLeastLevel(minLevel)
}

// IsWorkspaceOwner checks if the current user is the workspace owner (non-middleware version)
func IsWorkspaceOwner(c *gin.Context, pool *pgxpool.Pool) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}

	workspaceID := GetWorkspaceID(c)
	if workspaceID == nil {
		return false
	}

	var ownerID string
	err := pool.QueryRow(c.Request.Context(), "SELECT owner_id FROM workspaces WHERE id = $1", workspaceID).Scan(&ownerID)
	if err != nil {
		return false
	}

	return ownerID == user.ID
}
