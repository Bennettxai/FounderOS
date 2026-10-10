package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/resend/resend-go/v2"
)

// EmailService handles sending emails via Resend
type EmailService struct {
	client    *resend.Client
	fromEmail string
	fromName  string
	appURL    string
	enabled   bool
}

// NewEmailService creates a new EmailService instance
func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		slog.Info("Email service disabled (RESEND_API_KEY not set)")
		return &EmailService{enabled: false}
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

	slog.Info("EmailService initialized", "from_email", fromEmail, "app_url", appURL)

	return &EmailService{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
		appURL:    appURL,
		enabled:   true,
	}
}

// IsEnabled returns whether the email service is configured and enabled
func (s *EmailService) IsEnabled() bool {
	return s.enabled
}

// InvitationEmailData contains data for workspace invitation emails
type InvitationEmailData struct {
	RecipientEmail string
	WorkspaceName  string
	WorkspaceLogo  string
	InviterName    string
	RoleName       string
	Token          string
	ExpiresIn      string
}

// SendInvitationEmail sends a workspace invitation magic link email
func (s *EmailService) SendInvitationEmail(ctx context.Context, data InvitationEmailData) error {
	if !s.enabled {
		slog.Info("Email disabled - would send invitation",
			"recipient", data.RecipientEmail,
			"token", data.Token)
		return nil
	}

	magicLink := fmt.Sprintf("%s/invite/%s", s.appURL, data.Token)

	html := s.buildInvitationHTML(data, magicLink)
	text := s.buildInvitationText(data, magicLink)

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{data.RecipientEmail},
		Subject: fmt.Sprintf("%s invited you to join %s", data.InviterName, data.WorkspaceName),
		Html:    html,
		Text:    text,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		slog.Info("Failed to send invitation email to", "id", data.RecipientEmail, "error", err)
		return fmt.Errorf("failed to send invitation email: %w", err)
	}

	slog.Info("Invitation email sent to , ID", "id", data.RecipientEmail, "id", sent.Id)
	return nil
}

// NotificationEmailData contains data for notification emails
type NotificationEmailData struct {
	RecipientEmail string
	RecipientName  string
	Subject        string
	Title          string
	Body           string
	ActionURL      string
	ActionText     string
}

// SendNotificationEmail sends a notification email
func (s *EmailService) SendNotificationEmail(ctx context.Context, data NotificationEmailData) error {
	if !s.enabled {
		slog.Info("Email disabled - would send notification to", "value0", data.RecipientEmail, "value1", data.Subject)
		return nil
	}

	html := s.buildNotificationHTML(data)
	text := s.buildNotificationText(data)

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{data.RecipientEmail},
		Subject: data.Subject,
		Html:    html,
		Text:    text,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		slog.Info("Failed to send notification email to", "id", data.RecipientEmail, "error", err)
		return fmt.Errorf("failed to send notification email: %w", err)
	}

	slog.Info("Notification email sent to , ID", "id", data.RecipientEmail, "id", sent.Id)
	return nil
}

// SendGenericEmail sends a custom email
func (s *EmailService) SendGenericEmail(ctx context.Context, to string, subject string, html string, text string) error {
	if !s.enabled {
		slog.Info("Email disabled - would send email to", "value0", to, "value1", subject)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{to},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		slog.Info("Failed to send email to", "id", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Email sent to , ID", "id", to, "id", sent.Id)
	return nil
}

func (s *EmailService) buildInvitationHTML(data InvitationEmailData, magicLink string) string {
	logoURL := s.appURL + "/logo.png"
	if data.WorkspaceLogo != "" {
		logoURL = data.WorkspaceLogo
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #1f2937;
            background-color: #f3f4f6;
            margin: 0;
            padding: 0;
        }
        .wrapper {
            max-width: 600px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            padding: 40px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo img {
            height: 40px;
        }
        h1 {
            color: #111827;
            font-size: 24px;
            font-weight: 600;
            margin: 0 0 20px 0;
            text-align: center;
        }
        .workspace-name {
            color: #2563eb;
        }
        .invite-box {
            background: #f9fafb;
            border-radius: 8px;
            padding: 20px;
            margin: 24px 0;
            text-align: center;
        }
        .invite-box p {
            margin: 0;
            color: #6b7280;
        }
        .invite-box .role {
            font-size: 18px;
            font-weight: 600;
            color: #111827;
            margin-top: 8px;
        }
        .button {
            display: inline-block;
            background: #2563eb;
            color: white !important;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            margin: 24px 0;
        }
        .button:hover {
            background: #1d4ed8;
        }
        .button-wrapper {
            text-align: center;
        }
        .link-fallback {
            margin-top: 24px;
            padding-top: 24px;
            border-top: 1px solid #e5e7eb;
        }
        .link-fallback p {
            color: #6b7280;
            font-size: 14px;
            margin: 0 0 8px 0;
        }
        .link-fallback a {
            color: #2563eb;
            word-break: break-all;
            font-size: 14px;
        }
        .footer {
            text-align: center;
            margin-top: 32px;
            color: #9ca3af;
            font-size: 13px;
        }
        .footer p {
            margin: 4px 0;
        }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="card">
            <div class="logo">
                <img src="%s" alt="BusinessOS" />
            </div>
            
            <h1>Join <span class="workspace-name">%s</span></h1>
            
            <p><strong>%s</strong> has invited you to join their workspace on BusinessOS.</p>
            
            <div class="invite-box">
                <p>You've been invited as</p>
                <p class="role">%s</p>
            </div>
            
            <div class="button-wrapper">
                <a href="%s" class="button">Accept Invitation</a>
            </div>
            
            <div class="link-fallback">
                <p>Or copy and paste this link into your browser:</p>
                <a href="%s">%s</a>
            </div>
        </div>
        
        <div class="footer">
            <p>This invitation expires in %s.</p>
            <p>If you weren't expecting this invitation, you can safely ignore this email.</p>
        </div>
    </div>
</body>
</html>
`, logoURL, data.WorkspaceName, data.InviterName, data.RoleName, magicLink, magicLink, magicLink, data.ExpiresIn)
}

func (s *EmailService) buildInvitationText(data InvitationEmailData, magicLink string) string {
	return fmt.Sprintf(`You've been invited to join %s on BusinessOS

%s has invited you to join their workspace as a %s.

Accept the invitation by clicking the link below:
%s

This invitation expires in %s.

If you weren't expecting this invitation, you can safely ignore this email.
`, data.WorkspaceName, data.InviterName, data.RoleName, magicLink, data.ExpiresIn)
}

func (s *EmailService) buildNotificationHTML(data NotificationEmailData) string {
	actionButton := ""
	if data.ActionURL != "" && data.ActionText != "" {
		actionButton = fmt.Sprintf(`
            <div class="button-wrapper">
                <a href="%s" class="button">%s</a>
            </div>
`, data.ActionURL, data.ActionText)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #1f2937;
            background-color: #f3f4f6;
            margin: 0;
            padding: 0;
        }
        .wrapper {
            max-width: 600px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        .card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            padding: 40px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo img {
            height: 40px;
        }
        h1 {
            color: #111827;
            font-size: 24px;
            font-weight: 600;
            margin: 0 0 20px 0;
        }
        .body-text {
            color: #4b5563;
            margin: 16px 0;
        }
        .button {
            display: inline-block;
            background: #2563eb;
            color: white !important;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 14px;
            margin: 16px 0;
        }
        .button:hover {
            background: #1d4ed8;
        }
        .button-wrapper {
            margin-top: 24px;
        }
        .footer {
            text-align: center;
            margin-top: 32px;
            color: #9ca3af;
            font-size: 13px;
        }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="card">
            <div class="logo">
                <img src="%s/logo.png" alt="BusinessOS" />
            </div>
            
            <h1>%s</h1>
            
            <div class="body-text">%s</div>
            %s
        </div>
        
        <div class="footer">
            <p>You received this email because you have notifications enabled.</p>
        </div>
    </div>
</body>
</html>
`, s.appURL, data.Title, data.Body, actionButton)
}

func (s *EmailService) buildNotificationText(data NotificationEmailData) string {
	actionText := ""
	if data.ActionURL != "" {
		actionText = fmt.Sprintf("\n\n%s: %s", data.ActionText, data.ActionURL)
	}

	return fmt.Sprintf(`%s

%s%s
`, data.Title, data.Body, actionText)
}
