package templates

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/resend/resend-go/v2"
)

//go:embed email/*.html
var emailTemplates embed.FS

// EmailTemplateService handles email template rendering and sending
type EmailTemplateService struct {
	client       *resend.Client
	fromEmail    string
	fromName     string
	appURL       string
	logoURL      string
	enabled      bool
	templates    *template.Template
	supportEmail string
}

// NewEmailTemplateService creates a new email template service
func NewEmailTemplateService() *EmailTemplateService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		slog.Info("Email template service disabled (RESEND_API_KEY not set)")
		return &EmailTemplateService{enabled: false}
	}

	fromEmail := os.Getenv("RESEND_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "noreply@osa.dev"
	}

	fromName := os.Getenv("RESEND_FROM_NAME")
	if fromName == "" {
		fromName = "BusinessOS"
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}

	logoURL := os.Getenv("EMAIL_LOGO_URL")
	if logoURL == "" {
		logoURL = appURL + "/osa-logo.png"
	}

	supportEmail := os.Getenv("SUPPORT_EMAIL")
	if supportEmail == "" {
		supportEmail = "support@osa.dev"
	}

	// Parse all templates
	tmpl, err := template.ParseFS(emailTemplates, "email/*.html")
	if err != nil {
		slog.Warn("WARNING: Failed to parse email templates", "error", err)
		return &EmailTemplateService{enabled: false}
	}

	slog.Info("EmailTemplateService initialized", "from_name", fromName, "from_email", fromEmail, "app_url", appURL)

	return &EmailTemplateService{
		client:       resend.NewClient(apiKey),
		fromEmail:    fromEmail,
		fromName:     fromName,
		appURL:       appURL,
		logoURL:      logoURL,
		supportEmail: supportEmail,
		enabled:      true,
		templates:    tmpl,
	}
}

// IsEnabled returns whether the email service is enabled
func (s *EmailTemplateService) IsEnabled() bool {
	return s.enabled
}

// BaseEmailData contains common fields for all email templates
type BaseEmailData struct {
	AppURL       string
	LogoURL      string
	SupportEmail string
	Year         int
	Subject      string
}

func (s *EmailTemplateService) newBaseData(subject string) BaseEmailData {
	return BaseEmailData{
		AppURL:       s.appURL,
		LogoURL:      s.logoURL,
		SupportEmail: s.supportEmail,
		Year:         time.Now().Year(),
		Subject:      subject,
	}
}

// ===========================================
// Authentication Email Templates
// ===========================================

// EmailVerificationData for email verification template
type EmailVerificationData struct {
	BaseEmailData
	UserName         string
	VerificationLink string
	VerificationCode string
	ExpiresIn        string
}

// SendEmailVerification sends the email verification email
func (s *EmailTemplateService) SendEmailVerification(ctx context.Context, to, userName, verificationLink, verificationCode, expiresIn string) error {
	subject := "Verify your BusinessOS account"
	data := EmailVerificationData{
		BaseEmailData:    s.newBaseData(subject),
		UserName:         userName,
		VerificationLink: verificationLink,
		VerificationCode: verificationCode,
		ExpiresIn:        expiresIn,
	}
	return s.sendTemplate(ctx, to, subject, "email_verification.html", data)
}

// PasswordResetData for password reset template
type PasswordResetData struct {
	BaseEmailData
	UserName    string
	ResetLink   string
	ExpiresIn   string
	IPAddress   string
	RequestTime string
	Device      string
}

// SendPasswordReset sends the password reset email
func (s *EmailTemplateService) SendPasswordReset(ctx context.Context, to, userName, resetLink, expiresIn, ipAddress, device string) error {
	subject := "Reset your BusinessOS password"
	data := PasswordResetData{
		BaseEmailData: s.newBaseData(subject),
		UserName:      userName,
		ResetLink:     resetLink,
		ExpiresIn:     expiresIn,
		IPAddress:     ipAddress,
		RequestTime:   time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		Device:        device,
	}
	return s.sendTemplate(ctx, to, subject, "password_reset.html", data)
}

// PasswordChangedData for password changed notification
type PasswordChangedData struct {
	BaseEmailData
	UserName   string
	ChangeTime string
	Device     string
	IPAddress  string
	SecureLink string
}

// SendPasswordChanged sends the password changed notification
func (s *EmailTemplateService) SendPasswordChanged(ctx context.Context, to, userName, device, ipAddress string) error {
	subject := "Your BusinessOS password was changed"
	data := PasswordChangedData{
		BaseEmailData: s.newBaseData(subject),
		UserName:      userName,
		ChangeTime:    time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		Device:        device,
		IPAddress:     ipAddress,
		SecureLink:    s.appURL + "/settings/security",
	}
	return s.sendTemplate(ctx, to, subject, "password_changed.html", data)
}

// MagicLinkData for magic link login
type MagicLinkData struct {
	BaseEmailData
	UserName  string
	MagicLink string
	ExpiresIn string
}

// SendMagicLink sends a passwordless login link
func (s *EmailTemplateService) SendMagicLink(ctx context.Context, to, userName, magicLink, expiresIn string) error {
	subject := "Your BusinessOS login link"
	data := MagicLinkData{
		BaseEmailData: s.newBaseData(subject),
		UserName:      userName,
		MagicLink:     magicLink,
		ExpiresIn:     expiresIn,
	}
	return s.sendTemplate(ctx, to, subject, "magic_link.html", data)
}

// ===========================================
// Workspace & Team Email Templates
// ===========================================

// WorkspaceInvitationData for workspace invitation template
type WorkspaceInvitationData struct {
	BaseEmailData
	InviterName     string
	InviterEmail    string
	WorkspaceName   string
	Role            string
	InvitationLink  string
	ExpiresIn       string
	PersonalMessage string
}

// SendWorkspaceInvitation sends a workspace invitation email
func (s *EmailTemplateService) SendWorkspaceInvitation(ctx context.Context, to string, data WorkspaceInvitationData) error {
	subject := fmt.Sprintf("%s invited you to %s", data.InviterName, data.WorkspaceName)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "workspace_invitation.html", data)
}

// RoleChangedData for role change notification
type RoleChangedData struct {
	BaseEmailData
	UserName      string
	WorkspaceName string
	OldRole       string
	NewRole       string
	ChangedBy     string
	WorkspaceLink string
}

// SendRoleChanged sends a role change notification
func (s *EmailTemplateService) SendRoleChanged(ctx context.Context, to string, data RoleChangedData) error {
	subject := fmt.Sprintf("Your role in %s has changed", data.WorkspaceName)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "role_changed.html", data)
}

// WorkspaceRemovalData for workspace removal notification
type WorkspaceRemovalData struct {
	BaseEmailData
	UserName      string
	WorkspaceName string
	RemovedBy     string
	RemovalTime   string
	Reason        string
}

// SendWorkspaceRemoval sends a workspace removal notification
func (s *EmailTemplateService) SendWorkspaceRemoval(ctx context.Context, to string, data WorkspaceRemovalData) error {
	subject := fmt.Sprintf("You've been removed from %s", data.WorkspaceName)
	data.BaseEmailData = s.newBaseData(subject)
	if data.RemovalTime == "" {
		data.RemovalTime = time.Now().Format("January 2, 2006 at 3:04 PM MST")
	}
	return s.sendTemplate(ctx, to, subject, "workspace_removal.html", data)
}

// OrganizationInvitationData for organization invitation template
type OrganizationInvitationData struct {
	BaseEmailData
	InviterName      string
	InviterEmail     string
	OrganizationName string
	Role             string
	InvitationLink   string
	ExpiresIn        string
	PersonalMessage  string
}

// SendOrganizationInvitation sends an invite to join an organization (account).
func (s *EmailTemplateService) SendOrganizationInvitation(ctx context.Context, to string, data OrganizationInvitationData) error {
	subject := fmt.Sprintf("%s invited you to %s", data.InviterName, data.OrganizationName)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "organization_invitation.html", data)
}

// TeamAssignmentData for team assignment notification
type TeamAssignmentData struct {
	BaseEmailData
	UserName      string
	TeamName      string
	WorkspaceName string
	AssignedBy    string
	WorkspaceLink string
}

// SendTeamAssignment notifies a user they were added to a team.
func (s *EmailTemplateService) SendTeamAssignment(ctx context.Context, to string, data TeamAssignmentData) error {
	subject := fmt.Sprintf("You've been added to %s", data.TeamName)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "team_assignment.html", data)
}

// ===========================================
// Task Email Templates
// ===========================================

// TaskAssignedData for task assignment notification
type TaskAssignedData struct {
	BaseEmailData
	UserName        string
	AssignerName    string
	TaskTitle       string
	TaskDescription string
	ProjectName     string
	Priority        string
	DueDate         string
	TaskLink        string
}

// SendTaskAssigned sends a task assignment notification
func (s *EmailTemplateService) SendTaskAssigned(ctx context.Context, to string, data TaskAssignedData) error {
	subject := fmt.Sprintf("%s assigned you: %s", data.AssignerName, data.TaskTitle)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "task_assigned.html", data)
}

// TaskDueReminderData for task due reminder
type TaskDueReminderData struct {
	BaseEmailData
	UserName       string
	TaskTitle      string
	ProjectName    string
	DueDate        string
	DueRelative    string
	HoursRemaining string
	TaskLink       string
}

// SendTaskDueReminder sends a task due reminder
func (s *EmailTemplateService) SendTaskDueReminder(ctx context.Context, to string, data TaskDueReminderData) error {
	subject := fmt.Sprintf("⏰ Reminder: %s is due %s", data.TaskTitle, data.DueRelative)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "task_due_reminder.html", data)
}

// TaskOverdueData for task overdue notification
type TaskOverdueData struct {
	BaseEmailData
	UserName    string
	TaskTitle   string
	ProjectName string
	DueDate     string
	DaysOverdue string
	TaskLink    string
}

// SendTaskOverdue sends a task overdue notification
func (s *EmailTemplateService) SendTaskOverdue(ctx context.Context, to string, data TaskOverdueData) error {
	subject := fmt.Sprintf("🚨 Overdue: %s", data.TaskTitle)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "task_overdue.html", data)
}

// ===========================================
// Comments & Mentions Email Templates
// ===========================================

// MentionData for @mention notification
type MentionData struct {
	BaseEmailData
	UserName       string
	MentionerName  string
	EntityType     string
	EntityTitle    string
	CommentSnippet string
	CommentLink    string
}

// SendMention sends a mention notification
func (s *EmailTemplateService) SendMention(ctx context.Context, to string, data MentionData) error {
	subject := fmt.Sprintf("%s mentioned you in %s", data.MentionerName, data.EntityTitle)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "mention.html", data)
}

// CommentReplyData for comment reply notification
type CommentReplyData struct {
	BaseEmailData
	UserName        string
	ReplierName     string
	EntityTitle     string
	OriginalSnippet string
	ReplySnippet    string
	CommentLink     string
}

// SendCommentReply sends a comment reply notification
func (s *EmailTemplateService) SendCommentReply(ctx context.Context, to string, data CommentReplyData) error {
	subject := fmt.Sprintf("%s replied to your comment", data.ReplierName)
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "comment_reply.html", data)
}

// ===========================================
// Onboarding & Welcome Email Templates
// ===========================================

// WelcomeData for welcome email
type WelcomeData struct {
	BaseEmailData
	UserName          string
	WorkspaceName     string
	BusinessType      string
	GettingStartedURL string
	HelpCenterURL     string
	CommunityURL      string
}

// SendWelcome sends a welcome email
func (s *EmailTemplateService) SendWelcome(ctx context.Context, to string, data WelcomeData) error {
	subject := fmt.Sprintf("Welcome to BusinessOS, %s! 🎉", data.UserName)
	data.BaseEmailData = s.newBaseData(subject)
	if data.GettingStartedURL == "" {
		data.GettingStartedURL = s.appURL + "/dashboard"
	}
	if data.HelpCenterURL == "" {
		data.HelpCenterURL = s.appURL + "/help"
	}
	return s.sendTemplate(ctx, to, subject, "welcome.html", data)
}

// ===========================================
// Digest Email Templates
// ===========================================

// WeeklyDigestTask represents a task in the weekly digest
type WeeklyDigestTask struct {
	Title   string
	DueDate string
}

// WeeklyDigestProject represents a project in the weekly digest
type WeeklyDigestProject struct {
	Name      string
	TaskCount int
}

// WeeklyDigestData for weekly digest email
type WeeklyDigestData struct {
	BaseEmailData
	UserName          string
	WeekRange         string
	TasksCompleted    int
	TasksCreated      int
	TasksOverdue      int
	CommentsReceived  int
	MentionsCount     int
	UpcomingDeadlines []WeeklyDigestTask
	TopProjects       []WeeklyDigestProject
	DashboardLink     string
}

// SendWeeklyDigest sends a weekly digest email
func (s *EmailTemplateService) SendWeeklyDigest(ctx context.Context, to string, data WeeklyDigestData) error {
	subject := fmt.Sprintf("Your BusinessOS weekly summary - %s", data.WeekRange)
	data.BaseEmailData = s.newBaseData(subject)
	if data.DashboardLink == "" {
		data.DashboardLink = s.appURL + "/dashboard"
	}
	return s.sendTemplate(ctx, to, subject, "weekly_digest.html", data)
}

// ===========================================
// Core Template Sending
// ===========================================

// sendTemplate renders and sends an email using a template
func (s *EmailTemplateService) sendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error {
	if !s.enabled {
		slog.Info("Email disabled - would send  to", "value0", templateName, "value1", to)
		return nil
	}

	// Parse base + THIS email's content into a fresh set. Every content file
	// defines its own "content"/"footer" blocks, so parsing them all at once (the
	// global s.templates) collapses them to whichever file parsed last — every
	// email would render the same wrong body. Parsing per-send isolates the
	// correct content/footer for this template.
	tmpl, err := template.ParseFS(emailTemplates, "email/base.html", "email/"+templateName)
	if err != nil {
		slog.Info("Failed to parse email template", "id", templateName, "error", err)
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Render the full email (base pulls in this template's content + footer).
	var htmlBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&htmlBuf, "base.html", data); err != nil {
		slog.Info("Failed to render base template", "error", err)
		return fmt.Errorf("failed to render base email template: %w", err)
	}

	// Render the content block alone for the plain-text fallback.
	var contentBuf bytes.Buffer
	_ = tmpl.ExecuteTemplate(&contentBuf, "content", data)

	// Generate plain text version
	plainText := s.htmlToPlainText(contentBuf.String())

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{to},
		Subject: subject,
		Html:    htmlBuf.String(),
		Text:    plainText,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		slog.Info("Failed to send email to", "id", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Email sent:,,", "template", templateName, "template", to, "template", sent.Id)
	return nil
}

// htmlToPlainText converts HTML to plain text (basic implementation)
func (s *EmailTemplateService) htmlToPlainText(html string) string {
	// Remove HTML tags
	text := html

	// Replace common HTML elements
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "</div>", "\n")
	text = strings.ReplaceAll(text, "</h1>", "\n\n")
	text = strings.ReplaceAll(text, "</h2>", "\n\n")
	text = strings.ReplaceAll(text, "</li>", "\n")
	text = strings.ReplaceAll(text, "<li>", "• ")

	// Remove remaining tags
	for strings.Contains(text, "<") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if start != -1 && end != -1 && end > start {
			text = text[:start] + text[end+1:]
		} else {
			break
		}
	}

	// Clean up whitespace
	text = strings.TrimSpace(text)

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	return text
}

// ===========================================
// Platform Admin Email Templates
// ===========================================

// EngineSetupReminderData for the Optimal Engine setup reminder email.
type EngineSetupReminderData struct {
	BaseEmailData
	UserName      string
	WorkspaceName string
	SetupLink     string
}

// SendEngineSetupReminder sends a reminder to a workspace owner to connect their Optimal Engine.
func (s *EmailTemplateService) SendEngineSetupReminder(ctx context.Context, to string, data EngineSetupReminderData) error {
	subject := "Connect your Optimal Engine"
	data.BaseEmailData = s.newBaseData(subject)
	return s.sendTemplate(ctx, to, subject, "engine_setup_reminder.html", data)
}

// SendTestEmail sends a test email to verify configuration
func (s *EmailTemplateService) SendTestEmail(ctx context.Context, to string) error {
	if !s.enabled {
		return fmt.Errorf("email service is disabled")
	}

	subject := "BusinessOS Email Test"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: sans-serif; padding: 20px; }
        .success { color: #10b981; font-size: 24px; }
    </style>
</head>
<body>
    <h1 class="success">✓ Email Configuration Working!</h1>
    <p>This test email was sent from your BusinessOS instance.</p>
    <p><strong>From:</strong> %s &lt;%s&gt;</p>
    <p><strong>Time:</strong> %s</p>
</body>
</html>
`, s.fromName, s.fromEmail, time.Now().Format(time.RFC1123))

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{to},
		Subject: subject,
		Html:    html,
		Text:    fmt.Sprintf("Email Configuration Working!\n\nFrom: %s <%s>\nTime: %s", s.fromName, s.fromEmail, time.Now().Format(time.RFC1123)),
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send test email: %w", err)
	}

	slog.Info("Test email sent to , ID", "id", to, "id", sent.Id)
	return nil
}
