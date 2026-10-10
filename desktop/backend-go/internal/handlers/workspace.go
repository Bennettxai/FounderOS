package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// WORKSPACE HANDLER STRUCT
// =====================================================================

// WorkspaceHandler handles all workspace-domain HTTP requests.
// It owns workspace CRUD, member management, invites, audit logs, and
// workspace versioning — all routes that live under /api/workspaces.
type WorkspaceHandler struct {
	pool                    *pgxpool.Pool
	workspaceService        *services.WorkspaceService
	workspaceVersionService *services.WorkspaceVersionService
	roleContextService      *services.RoleContextService
	inviteService           *services.WorkspaceInviteService
	auditService            *services.WorkspaceAuditService
}

// NewWorkspaceHandler constructs a WorkspaceHandler with all required dependencies.
func NewWorkspaceHandler(
	pool *pgxpool.Pool,
	workspaceService *services.WorkspaceService,
	workspaceVersionService *services.WorkspaceVersionService,
	roleContextService *services.RoleContextService,
	inviteService *services.WorkspaceInviteService,
	auditService *services.WorkspaceAuditService,
) *WorkspaceHandler {
	return &WorkspaceHandler{
		pool:                    pool,
		workspaceService:        workspaceService,
		workspaceVersionService: workspaceVersionService,
		roleContextService:      roleContextService,
		inviteService:           inviteService,
		auditService:            auditService,
	}
}

// RegisterWorkspaceRoutes wires all workspace-domain routes onto the provided
// RouterGroup. auth is the authentication middleware shared across the API.
//
// NOTE on route ordering: /versions/compare/:v1/:v2 must be registered before
// /versions/:version because Gin's radix tree treats "compare" as a wildcard match.
// Similarly, /invites/validate and /invites/accept are public endpoints registered
// outside the workspace-scoped group (no role context injection needed).
func RegisterWorkspaceRoutes(api *gin.RouterGroup, h *WorkspaceHandler, auth gin.HandlerFunc) {
	workspaces := api.Group("/workspaces")
	workspaces.Use(auth, middleware.RequireAuth())
	{
		// Workspace CRUD - no role context needed for create/list
		workspaces.POST("", h.CreateWorkspace)
		workspaces.GET("", h.ListWorkspaces)

		// Workspace-scoped routes - inject role context for permission checks
		workspaceScoped := workspaces.Group("/:id")
		workspaceScoped.Use(middleware.InjectRoleContext(h.pool, h.roleContextService))
		{
			// Read operations - any member
			workspaceScoped.GET("", h.GetWorkspace)
			workspaceScoped.GET("/members", h.ListWorkspaceMembers)
			workspaceScoped.GET("/roles", h.ListWorkspaceRoles)
			workspaceScoped.GET("/profile", h.GetWorkspaceProfile)
			workspaceScoped.GET("/role-context", h.GetUserRoleContext)

			// Update user profile
			workspaceScoped.PUT("/profile", h.UpdateWorkspaceProfile)

			// Optimal Engine connection (per-workspace, shared) - members read,
			// managers+ configure.
			workspaceScoped.GET("/engine", h.GetWorkspaceEngineConfig)
			workspaceScoped.PUT("/engine", middleware.RequireWorkspaceManager(), h.UpdateWorkspaceEngineConfig)
			workspaceScoped.POST("/engine/test", h.TestWorkspaceEngineConnection)

			// Per-module sync policies (local-first; sync a whole module or not).
			// Members read; managers+ change.
			workspaceScoped.GET("/sync-policies", h.GetWorkspaceSyncPolicies)
			workspaceScoped.PUT("/sync-policies", middleware.RequireWorkspaceManager(), h.SetWorkspaceSyncPolicies)

			// Update workspace - requires admin or owner
			workspaceScoped.PUT("", middleware.RequireWorkspaceAdmin(), h.UpdateWorkspace)

			// Delete workspace - requires owner only
			workspaceScoped.DELETE("", middleware.RequireWorkspaceOwner(h.pool), h.DeleteWorkspace)

			// Invite members - requires manager, admin, or owner
			workspaceScoped.POST("/members/invite", middleware.RequireWorkspaceManager(), h.AddWorkspaceMember)

			// Update/remove members - requires admin or owner
			workspaceScoped.PUT("/members/:userId", middleware.RequireWorkspaceAdmin(), h.UpdateWorkspaceMemberRole)
			workspaceScoped.DELETE("/members/:userId", middleware.RequireWorkspaceAdmin(), h.RemoveWorkspaceMember)

			// Workspace invitations - manager+ can invite, admin+ can list/revoke
			workspaceScoped.POST("/invites", middleware.RequireWorkspaceManager(), h.CreateWorkspaceInvite)
			workspaceScoped.GET("/invites", middleware.RequireWorkspaceAdmin(), h.ListWorkspaceInvites)
			workspaceScoped.DELETE("/invites/:inviteId", middleware.RequireWorkspaceAdmin(), h.RevokeWorkspaceInvite)

			// Audit logs - admin+ can view
			workspaceScoped.GET("/audit-logs", middleware.RequireWorkspaceAdmin(), h.ListAuditLogs)
			workspaceScoped.GET("/audit-logs/:logId", middleware.RequireWorkspaceAdmin(), h.GetAuditLog)
			workspaceScoped.GET("/audit-logs/user/:userId", middleware.RequireWorkspaceAdmin(), h.GetUserActivity)
			workspaceScoped.GET("/audit-logs/resource/:resourceType/:resourceId", middleware.RequireWorkspaceAdmin(), h.GetResourceHistory)
			workspaceScoped.GET("/audit-logs/stats/actions", middleware.RequireWorkspaceAdmin(), h.GetActionStats)
			workspaceScoped.GET("/audit-logs/stats/active-users", middleware.RequireWorkspaceAdmin(), h.GetMostActiveUsers)

			// Workspace versions - /versions/compare must precede /versions/:version
			workspaceScoped.GET("/versions", h.ListWorkspaceVersions)
			workspaceScoped.POST("/versions", middleware.RequireWorkspaceAdmin(), h.CreateWorkspaceVersion)
			workspaceScoped.GET("/versions/compare/:v1/:v2", h.CompareWorkspaceVersions)
			workspaceScoped.GET("/versions/:version", h.GetWorkspaceVersion)
			workspaceScoped.POST("/restore/:version", middleware.RequireWorkspaceAdmin(), h.RestoreWorkspaceVersion)
		}

		// Public invite endpoints - no workspace context required
		workspaces.POST("/invites/validate", h.ValidateWorkspaceInvite)
		workspaces.POST("/invites/accept", h.AcceptWorkspaceInvite)
	}
}

// =====================================================================
// WORKSPACE CRUD HANDLERS
// =====================================================================

// CreateWorkspace creates a new workspace.
// POST /api/workspaces
func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req services.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	workspace, err := h.workspaceService.CreateWorkspace(c.Request.Context(), req, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workspace)
}

// ListWorkspaces lists all workspaces the user is a member of.
// GET /api/workspaces
func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaces, err := h.workspaceService.ListUserWorkspaces(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspaces": workspaces})
}

// GetWorkspace gets a workspace by ID.
// GET /api/workspaces/:id
func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	workspace, err := h.workspaceService.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

// UpdateWorkspace updates a workspace.
// PUT /api/workspaces/:id
func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Check if user has permission (admin or owner)
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can update workspace"})
		return
	}

	var req services.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	workspace, err := h.workspaceService.UpdateWorkspace(c.Request.Context(), workspaceID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

// DeleteWorkspace deletes a workspace.
// DELETE /api/workspaces/:id
func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	err = h.workspaceService.DeleteWorkspace(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workspace deleted successfully"})
}

// =====================================================================
// MEMBER MANAGEMENT HANDLERS
// =====================================================================

// ListWorkspaceMembers lists all members of a workspace.
// GET /api/workspaces/:id/members
func (h *WorkspaceHandler) ListWorkspaceMembers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	members, err := h.workspaceService.ListMembers(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddWorkspaceMember adds a member to the workspace.
// POST /api/workspaces/:id/members/invite
func (h *WorkspaceHandler) AddWorkspaceMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Check if user has permission to invite (manager, admin, or owner)
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners, admins, and managers can invite members"})
		return
	}

	var req services.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	member, err := h.workspaceService.AddMember(c.Request.Context(), workspaceID, req, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// UpdateWorkspaceMemberRole updates a member's role.
// PUT /api/workspaces/:id/members/:userId
func (h *WorkspaceHandler) UpdateWorkspaceMemberRole(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	targetUserID := c.Param("userId")

	// Check if user has permission (admin or owner)
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can update member roles"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	member, err := h.workspaceService.UpdateMemberRole(c.Request.Context(), workspaceID, targetUserID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}

// RemoveWorkspaceMember removes a member from the workspace.
// DELETE /api/workspaces/:id/members/:userId
func (h *WorkspaceHandler) RemoveWorkspaceMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	targetUserID := c.Param("userId")

	// Check if user has permission (admin or owner)
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can remove members"})
		return
	}

	err = h.workspaceService.RemoveMember(c.Request.Context(), workspaceID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// =====================================================================
// ROLE MANAGEMENT HANDLERS
// =====================================================================

// ListWorkspaceRoles lists all roles in a workspace.
// GET /api/workspaces/:id/roles
func (h *WorkspaceHandler) ListWorkspaceRoles(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	roles, err := h.workspaceService.ListRoles(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// =====================================================================
// USER PROFILE & ROLE CONTEXT HANDLERS
// =====================================================================

// GetWorkspaceProfile gets the current user's profile in the workspace.
// GET /api/workspaces/:id/profile
func (h *WorkspaceHandler) GetWorkspaceProfile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Get user's membership in this workspace
	members, err := h.workspaceService.ListMembers(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find current user's membership
	var userMember *services.WorkspaceMember
	for i := range members {
		if members[i].UserID == user.ID {
			userMember = &members[i]
			break
		}
	}

	if userMember == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Return profile (using member info)
	profile := gin.H{
		"user_id":      userMember.UserID,
		"workspace_id": userMember.WorkspaceID.String(),
		"role":         userMember.Role,
		"status":       userMember.Status,
		"joined_at":    userMember.JoinedAt,
		"created_at":   userMember.CreatedAt,
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateWorkspaceProfile updates the current user's profile in the workspace.
// PUT /api/workspaces/:id/profile
func (h *WorkspaceHandler) UpdateWorkspaceProfile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Profile updates not yet implemented.
	// This would typically update user_workspace_profiles table.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Profile updates not yet implemented"})
}

// GetUserRoleContext gets the current user's role context (permissions) in the workspace.
// GET /api/workspaces/:id/role-context
func (h *WorkspaceHandler) GetUserRoleContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Use roleContextService when available
	if h.roleContextService != nil {
		roleCtx, err := h.roleContextService.GetUserRoleContext(c.Request.Context(), user.ID, workspaceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, roleCtx)
		return
	}

	// Fallback: build role context manually from workspace service
	roleName, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a member of this workspace"})
		return
	}

	workspace, err := h.workspaceService.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	roles, err := h.workspaceService.ListRoles(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var userRole *services.WorkspaceRole
	for i := range roles {
		if roles[i].Name == roleName {
			userRole = &roles[i]
			break
		}
	}

	if userRole == nil {
		utils.RespondNotFound(c, slog.Default(), "Role")
		return
	}

	roleContext := gin.H{
		"user_id":           user.ID,
		"workspace_id":      workspaceID.String(),
		"workspace_name":    workspace.Name,
		"role_name":         userRole.Name,
		"role_display_name": userRole.DisplayName,
		"hierarchy_level":   userRole.HierarchyLevel,
		"permissions":       userRole.Permissions,
	}

	c.JSON(http.StatusOK, roleContext)
}
