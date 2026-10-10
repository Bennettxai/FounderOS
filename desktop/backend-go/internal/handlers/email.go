package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/services/templates"
	"github.com/rhl/businessos-backend/internal/utils"
)

// EmailHandler handles email-related endpoints
type EmailHandler struct {
	emailTemplateService *templates.EmailTemplateService
}

// NewEmailHandler creates a new email handler
func NewEmailHandler() *EmailHandler {
	return &EmailHandler{
		emailTemplateService: templates.NewEmailTemplateService(),
	}
}

// TestEmailRequest represents a test email request
type TestEmailRequest struct {
	To       string `json:"to"`
	Template string `json:"template,omitempty"` // Optional: specific template to test
}

// HandleTestEmail sends a test email to verify configuration
func (h *EmailHandler) HandleTestEmail(c *gin.Context) {
	var req TestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.To == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	if !h.emailTemplateService.IsEnabled() {
		utils.RespondInternalError(c, slog.Default(), "email service", nil)
		return
	}

	// Send a test email
	err := h.emailTemplateService.SendTestEmail(c.Request.Context(), req.To)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "send test email", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test email sent successfully",
		"to":      req.To,
	})
}

// HandleSendVerificationEmail sends a verification email
func (h *EmailHandler) HandleSendVerificationEmail(c *gin.Context) {
	var req struct {
		To               string `json:"to"`
		UserName         string `json:"user_name"`
		VerificationLink string `json:"verification_link"`
		VerificationCode string `json:"verification_code"`
		ExpiresIn        string `json:"expires_in"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.To == "" || req.VerificationLink == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	if req.ExpiresIn == "" {
		req.ExpiresIn = "24 hours"
	}

	err := h.emailTemplateService.SendEmailVerification(
		c.Request.Context(),
		req.To,
		req.UserName,
		req.VerificationLink,
		req.VerificationCode,
		req.ExpiresIn,
	)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "send verification email", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent",
		"to":      req.To,
	})
}

// HandleSendPasswordResetEmail sends a password reset email
func (h *EmailHandler) HandleSendPasswordResetEmail(c *gin.Context) {
	var req struct {
		To        string `json:"to"`
		UserName  string `json:"user_name"`
		ResetLink string `json:"reset_link"`
		ExpiresIn string `json:"expires_in"`
		IPAddress string `json:"ip_address"`
		Device    string `json:"device"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.To == "" || req.ResetLink == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	if req.ExpiresIn == "" {
		req.ExpiresIn = "1 hour"
	}

	err := h.emailTemplateService.SendPasswordReset(
		c.Request.Context(),
		req.To,
		req.UserName,
		req.ResetLink,
		req.ExpiresIn,
		req.IPAddress,
		req.Device,
	)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "send password reset email", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset email sent",
		"to":      req.To,
	})
}

// HandleSendWelcomeEmail sends a welcome email
func (h *EmailHandler) HandleSendWelcomeEmail(c *gin.Context) {
	var req struct {
		To                string `json:"to"`
		UserName          string `json:"user_name"`
		WorkspaceName     string `json:"workspace_name"`
		BusinessType      string `json:"business_type"`
		GettingStartedURL string `json:"getting_started_url"`
		HelpCenterURL     string `json:"help_center_url"`
		CommunityURL      string `json:"community_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.To == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	data := templates.WelcomeData{
		UserName:          req.UserName,
		WorkspaceName:     req.WorkspaceName,
		BusinessType:      req.BusinessType,
		GettingStartedURL: req.GettingStartedURL,
		HelpCenterURL:     req.HelpCenterURL,
		CommunityURL:      req.CommunityURL,
	}

	err := h.emailTemplateService.SendWelcome(c.Request.Context(), req.To, data)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "send welcome email", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome email sent",
		"to":      req.To,
	})
}

// GetEmailStatus returns the email service configuration status
func (h *EmailHandler) GetEmailStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled": h.emailTemplateService.IsEnabled(),
		"templates": []string{
			"email_verification",
			"password_reset",
			"password_changed",
			"magic_link",
			"workspace_invitation",
			"role_changed",
			"workspace_removal",
			"task_assigned",
			"task_due_reminder",
			"task_overdue",
			"mention",
			"comment_reply",
			"welcome",
			"weekly_digest",
		},
	})
}
