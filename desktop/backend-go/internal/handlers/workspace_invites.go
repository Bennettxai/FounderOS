package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services/templates"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// WORKSPACE INVITE HANDLERS
// =====================================================================

// CreateInviteRequest represents the request to create an invitation.
type CreateInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=owner admin manager member viewer guest"`
}

// CreateWorkspaceInvite creates a new workspace invitation.
// POST /api/workspaces/:id/invites
// Required permission: invite_members (manager+)
func (h *WorkspaceHandler) CreateWorkspaceInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Create invitation
	invite, err := h.inviteService.CreateInvite(
		c.Request.Context(),
		workspaceID,
		req.Email,
		req.Role,
		user.ID,
	)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create invitation", err)
		return
	}

	// Send the invitation email (best-effort; never fails the request).
	go func(to, role, token, inviterName, inviterEmail string, wsID uuid.UUID) {
		emailSvc := templates.NewEmailTemplateService()
		if !emailSvc.IsEnabled() {
			return
		}
		appURL := os.Getenv("APP_URL")
		if appURL == "" {
			appURL = "https://app.businessos.dev"
		}
		var wsName string
		_ = h.pool.QueryRow(context.Background(),
			`SELECT name FROM workspaces WHERE id=$1`, wsID).Scan(&wsName)
		if wsName == "" {
			wsName = "your workspace"
		}
		if mailErr := emailSvc.SendWorkspaceInvitation(context.Background(), to, templates.WorkspaceInvitationData{
			InviterName:    inviterName,
			InviterEmail:   inviterEmail,
			WorkspaceName:  wsName,
			Role:           role,
			InvitationLink: appURL + "/invite/" + token,
			ExpiresIn:      "7 days",
		}); mailErr != nil {
			slog.Warn("[Workspace Invites] failed to send invite email", "error", mailErr, "to", to)
		}
	}(req.Email, req.Role, invite.Token, user.Name, user.Email, workspaceID)

	// Log the action
	if h.auditService != nil {
		if _, err := h.auditService.LogAction(
			c.Request.Context(),
			workspaceID,
			user.ID,
			"invite_member",
			"invite",
			nil,
			map[string]interface{}{
				"invite_id": invite.ID.String(),
				"email":     req.Email,
				"role":      req.Role,
			},
			getIPAddress(c),
			getUserAgent(c),
		); err != nil {
			// Log audit failure but don't fail the request
			slog.Warn("[Workspace Invites] Failed to log audit action", "error", err)
		}
	}

	c.JSON(http.StatusCreated, invite)
}

// ListWorkspaceInvites lists all invitations for a workspace.
// GET /api/workspaces/:id/invites
// Required permission: manage_members (admin+)
func (h *WorkspaceHandler) ListWorkspaceInvites(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	invites, err := h.inviteService.ListWorkspaceInvites(c.Request.Context(), workspaceID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list workspace invites", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invites": invites})
}

// AcceptInviteRequest represents the request to accept an invitation.
type AcceptInviteRequest struct {
	Token string `json:"token" binding:"required"`
}

// ValidateInviteRequest represents the request to validate an invitation.
type ValidateInviteRequest struct {
	Token string `json:"token" binding:"required"`
}

// ValidateInviteResponse represents the response when validating an invitation.
type ValidateInviteResponse struct {
	Valid         bool   `json:"valid"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	Role          string `json:"role,omitempty"`
	Email         string `json:"email,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ValidateWorkspaceInvite validates an invitation token without accepting it.
// POST /api/workspaces/invites/validate
// Returns workspace info if valid, allowing the user to preview before accepting.
func (h *WorkspaceHandler) ValidateWorkspaceInvite(c *gin.Context) {
	var req ValidateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ValidateInviteResponse{
			Valid: false,
			Error: "Token is required",
		})
		return
	}

	// Get invite by token
	invite, err := h.inviteService.GetInviteByToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusOK, ValidateInviteResponse{
			Valid: false,
			Error: "Invalid or expired invitation code",
		})
		return
	}

	// Check if still pending
	if invite.Status != "pending" {
		c.JSON(http.StatusOK, ValidateInviteResponse{
			Valid: false,
			Error: "This invitation has already been " + invite.Status,
		})
		return
	}

	// Check if expired
	if time.Now().After(invite.ExpiresAt) {
		c.JSON(http.StatusOK, ValidateInviteResponse{
			Valid: false,
			Error: "This invitation has expired",
		})
		return
	}

	// Get workspace name
	workspaceName := ""
	err = h.pool.QueryRow(c.Request.Context(),
		"SELECT name FROM workspaces WHERE id = $1",
		invite.WorkspaceID,
	).Scan(&workspaceName)
	if err != nil {
		workspaceName = "Unknown Workspace"
	}

	c.JSON(http.StatusOK, ValidateInviteResponse{
		Valid:         true,
		WorkspaceName: workspaceName,
		WorkspaceID:   invite.WorkspaceID.String(),
		Role:          invite.Role,
		Email:         invite.Email,
		ExpiresAt:     invite.ExpiresAt.Format("2006-01-02"),
	})
}

// AcceptWorkspaceInvite accepts a workspace invitation.
// POST /api/workspaces/invites/accept
// No workspace permission required (public endpoint for invited users).
func (h *WorkspaceHandler) AcceptWorkspaceInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Get invite to validate and log workspace ID
	invite, err := h.inviteService.GetInviteByToken(c.Request.Context(), req.Token)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Invitation")
		return
	}

	// SECURITY: Validate that the authenticated user's email matches the invite email.
	// This prevents users from accepting invitations meant for other email addresses.
	if !strings.EqualFold(user.Email, invite.Email) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "This invitation was sent to a different email address. Please sign in with the email address that received the invitation.",
			"code":  "EMAIL_MISMATCH",
		})
		return
	}

	// Check if invite is still pending
	if invite.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "This invitation has already been " + invite.Status,
			"code":  "INVITE_NOT_PENDING",
		})
		return
	}

	// Check if invite has expired
	if time.Now().After(invite.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "This invitation has expired. Please request a new invitation.",
			"code":  "INVITE_EXPIRED",
		})
		return
	}

	// Accept invitation
	err = h.inviteService.AcceptInvite(c.Request.Context(), req.Token, user.ID)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	// Log the action
	if h.auditService != nil {
		if _, err := h.auditService.LogAction(
			c.Request.Context(),
			invite.WorkspaceID,
			user.ID,
			"accept_invite",
			"invite",
			nil,
			map[string]interface{}{
				"invite_id": invite.ID.String(),
				"email":     invite.Email,
				"role":      invite.Role,
			},
			getIPAddress(c),
			getUserAgent(c),
		); err != nil {
			// Log audit failure but don't fail the request
			slog.Warn("[Workspace Invites] Failed to log audit action", "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Invitation accepted successfully",
		"workspace_id": invite.WorkspaceID.String(),
		"role":         invite.Role,
	})
}

// RevokeWorkspaceInvite revokes a pending invitation.
// DELETE /api/workspaces/:id/invites/:inviteId
// Required permission: manage_members (admin+)
func (h *WorkspaceHandler) RevokeWorkspaceInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "invite ID")
		return
	}

	// SECURITY (CRIT-12): Pass workspaceID to scope the revoke query — prevents an
	// admin of one workspace from revoking invites belonging to a different workspace.
	err = h.inviteService.RevokeInvite(c.Request.Context(), inviteID, workspaceID)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	// Log the action
	if h.auditService != nil {
		if _, err := h.auditService.LogAction(
			c.Request.Context(),
			workspaceID,
			user.ID,
			"revoke_invite",
			"invite",
			nil,
			map[string]interface{}{
				"invite_id": inviteID.String(),
			},
			getIPAddress(c),
			getUserAgent(c),
		); err != nil {
			// Log audit failure but don't fail the request
			slog.Warn("[Workspace Invites] Failed to log audit action", "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked successfully"})
}

// =====================================================================
// INVITE HELPER FUNCTIONS
// =====================================================================

func getIPAddress(c *gin.Context) *string {
	ip := c.ClientIP()
	return &ip
}

func getUserAgent(c *gin.Context) *string {
	ua := c.GetHeader("User-Agent")
	if ua == "" {
		return nil
	}
	return &ua
}
