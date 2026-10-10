package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// PROJECT ACCESS CONTROL HANDLERS (Role-based access)
// =====================================================================
// These handlers use the ProjectAccessService for workspace-level
// project access control with roles: lead, contributor, reviewer, viewer

// ListProjectMembers lists all members of a project.
// GET /api/projects/:id/members
func (h *ProjectHandler) ListProjectMembers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project ID")
		return
	}

	// Check if ProjectAccessService is available
	if h.projectAccessService == nil {
		utils.RespondNotImplemented(c, slog.Default(), "Project access service")
		return
	}

	// Check if user has access to the project
	hasAccess, err := h.projectAccessService.HasAccess(c.Request.Context(), user.ID, projectID)
	if err != nil || !hasAccess {
		utils.RespondForbidden(c, slog.Default(), "You do not have access to this project")
		return
	}

	// List all members
	members, err := h.projectAccessService.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list project members", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddProjectMember adds a member to a project.
// POST /api/projects/:id/members
func (h *ProjectHandler) AddProjectMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project ID")
		return
	}

	// Check if ProjectAccessService is available
	if h.projectAccessService == nil {
		utils.RespondNotImplemented(c, slog.Default(), "Project access service")
		return
	}

	// Check if user has permission to add members (lead or admin)
	canEdit, canDelete, canInvite, role, err := h.projectAccessService.GetPermissions(c.Request.Context(), user.ID, projectID)
	if err != nil {
		utils.RespondForbidden(c, slog.Default(), "You do not have access to this project")
		return
	}

	// Only leads can invite, or users with explicit can_invite permission
	if role != "lead" && !canInvite {
		utils.RespondForbidden(c, slog.Default(), "Only project leads can add members")
		return
	}

	// Suppress unused variable warnings (we may use these in future)
	_, _ = canEdit, canDelete

	var req struct {
		UserID      string    `json:"user_id" binding:"required"`
		WorkspaceID uuid.UUID `json:"workspace_id" binding:"required"`
		Role        string    `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate role
	validRoles := map[string]bool{
		"lead":        true,
		"contributor": true,
		"reviewer":    true,
		"viewer":      true,
	}
	if !validRoles[req.Role] {
		utils.RespondBadRequest(c, slog.Default(), "Invalid role. Must be one of: lead, contributor, reviewer, viewer")
		return
	}

	// Add member to project
	member, err := h.projectAccessService.AddMember(c.Request.Context(), projectID, req.UserID, req.Role, user.ID, req.WorkspaceID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "add project member", err)
		return
	}

	c.JSON(http.StatusCreated, member)
}

// UpdateProjectMemberRole updates a member's role in a project.
// PUT /api/projects/:id/members/:memberId/role
func (h *ProjectHandler) UpdateProjectMemberRole(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project ID")
		return
	}

	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "member ID")
		return
	}

	// Check if ProjectAccessService is available
	if h.projectAccessService == nil {
		utils.RespondNotImplemented(c, slog.Default(), "Project access service")
		return
	}

	// Check if user has permission to update roles (lead only)
	canEdit, canDelete, canInvite, role, err := h.projectAccessService.GetPermissions(c.Request.Context(), user.ID, projectID)
	if err != nil {
		utils.RespondForbidden(c, slog.Default(), "You do not have access to this project")
		return
	}

	// Only leads can update roles
	if role != "lead" {
		utils.RespondForbidden(c, slog.Default(), "Only project leads can update member roles")
		return
	}

	// Suppress unused variable warnings
	_, _, _ = canEdit, canDelete, canInvite

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate role
	validRoles := map[string]bool{
		"lead":        true,
		"contributor": true,
		"reviewer":    true,
		"viewer":      true,
	}
	if !validRoles[req.Role] {
		utils.RespondBadRequest(c, slog.Default(), "Invalid role. Must be one of: lead, contributor, reviewer, viewer")
		return
	}

	// Update member role
	err = h.projectAccessService.UpdateRole(c.Request.Context(), memberID, req.Role)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update member role", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member role updated successfully"})
}

// RemoveProjectMember removes a member from a project.
// DELETE /api/projects/:id/members/:memberId
func (h *ProjectHandler) RemoveProjectMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project ID")
		return
	}

	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "member ID")
		return
	}

	// Check if ProjectAccessService is available
	if h.projectAccessService == nil {
		utils.RespondNotImplemented(c, slog.Default(), "Project access service")
		return
	}

	// Check if user has permission to remove members (lead only)
	canEdit, canDelete, canInvite, role, err := h.projectAccessService.GetPermissions(c.Request.Context(), user.ID, projectID)
	if err != nil {
		utils.RespondForbidden(c, slog.Default(), "You do not have access to this project")
		return
	}

	// Only leads can remove members
	if role != "lead" {
		utils.RespondForbidden(c, slog.Default(), "Only project leads can remove members")
		return
	}

	// Suppress unused variable warnings
	_, _, _ = canEdit, canDelete, canInvite

	// Remove member from project
	err = h.projectAccessService.RemoveMember(c.Request.Context(), memberID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "remove project member", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// CheckProjectAccess checks if a user has access to a project.
// GET /api/projects/:id/access/:userId
func (h *ProjectHandler) CheckProjectAccess(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project ID")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.RespondBadRequest(c, slog.Default(), "User ID is required")
		return
	}

	// Check if ProjectAccessService is available
	if h.projectAccessService == nil {
		utils.RespondNotImplemented(c, slog.Default(), "Project access service")
		return
	}

	// Users can only check their own access unless they're a project lead
	if targetUserID != user.ID {
		canEdit, canDelete, canInvite, role, err := h.projectAccessService.GetPermissions(c.Request.Context(), user.ID, projectID)
		if err != nil || role != "lead" {
			utils.RespondForbidden(c, slog.Default(), "You can only check your own access")
			return
		}
		// Suppress unused variable warnings
		_, _, _ = canEdit, canDelete, canInvite
	}

	// Check access
	hasAccess, err := h.projectAccessService.HasAccess(c.Request.Context(), targetUserID, projectID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "check project access", err)
		return
	}

	// Get permissions if user has access
	var response gin.H
	if hasAccess {
		canEdit, canDelete, canInvite, role, err := h.projectAccessService.GetPermissions(c.Request.Context(), targetUserID, projectID)
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "get project permissions", err)
			return
		}

		response = gin.H{
			"has_access": true,
			"role":       role,
			"permissions": gin.H{
				"can_edit":   canEdit,
				"can_delete": canDelete,
				"can_invite": canInvite,
			},
		}
	} else {
		response = gin.H{
			"has_access": false,
		}
	}

	c.JSON(http.StatusOK, response)
}
