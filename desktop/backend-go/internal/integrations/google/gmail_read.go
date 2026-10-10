package google

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GetGmailAPI returns a Gmail API service for a user.
func (s *GmailService) GetGmailAPI(ctx context.Context, userID string) (*gmail.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return srv, nil
}

// SyncEmails syncs emails from Gmail.
// Requires gmail.readonly scope to be granted during OAuth.
func (s *GmailService) SyncEmails(ctx context.Context, userID string, maxResults int64) (*SyncEmailsResult, error) {
	slog.Info("Gmail sync starting for user : max  emails", "user_id", userID, "max_results", maxResults)

	if !s.IsConnected(ctx, userID) {
		return nil, fmt.Errorf("Gmail access not authorized. User must grant gmail.readonly scope during OAuth login")
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail API: %w", err)
	}

	req := srv.Users.Messages.List("me").
		MaxResults(maxResults).
		Q("in:inbox OR in:sent")

	messages, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	result := &SyncEmailsResult{
		TotalEmails: len(messages.Messages),
	}

	for _, msg := range messages.Messages {
		fullMsg, err := srv.Users.Messages.Get("me", msg.Id).Format("full").Do()
		if err != nil {
			slog.Info("Failed to get message", "id", msg.Id, "error", err)
			result.FailedEmails++
			continue
		}

		if err := s.saveEmail(ctx, userID, fullMsg); err != nil {
			slog.Info("Failed to save message", "id", msg.Id, "error", err)
			result.FailedEmails++
		} else {
			result.SyncedEmails++
		}
	}

	slog.Info("Gmail sync complete for user : synced / emails", "user_id", userID, "synced", result.SyncedEmails, "total", result.TotalEmails)

	return result, nil
}

// saveEmail saves an email to the database.
func (s *GmailService) saveEmail(ctx context.Context, userID string, msg *gmail.Message) error {
	var subject, from, to, cc, replyTo, date string
	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			subject = header.Value
		case "from":
			from = header.Value
		case "to":
			to = header.Value
		case "cc":
			cc = header.Value
		case "reply-to":
			replyTo = header.Value
		case "date":
			date = header.Value
		}
	}

	fromName, fromEmail := parseEmailAddress(from)
	toAddrs := parseEmailAddresses(to)
	ccAddrs := parseEmailAddresses(cc)
	bodyText, bodyHTML := extractBody(msg.Payload)

	labels := msg.LabelIds
	isRead := !containsLabel(labels, "UNREAD")
	isStarred := containsLabel(labels, "STARRED")
	isImportant := containsLabel(labels, "IMPORTANT")
	isDraft := containsLabel(labels, "DRAFT")
	isSent := containsLabel(labels, "SENT")
	isArchived := !containsLabel(labels, "INBOX") && !containsLabel(labels, "TRASH") && !containsLabel(labels, "SPAM")
	isTrash := containsLabel(labels, "TRASH")

	attachments := extractAttachments(msg.Payload)

	var emailDate time.Time
	if date != "" {
		if parsed, err := parseEmailDate(date); err == nil {
			emailDate = parsed
		}
	}
	if emailDate.IsZero() && msg.InternalDate != 0 {
		emailDate = time.UnixMilli(msg.InternalDate)
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO emails (
			user_id, provider, external_id, thread_id,
			subject, snippet, from_email, from_name,
			to_emails, cc_emails, reply_to,
			body_text, body_html, attachments,
			is_read, is_starred, is_important, is_draft, is_sent, is_archived, is_trash,
			labels, date, received_at, synced_at
		) VALUES (
			$1, 'gmail', $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, NOW()
		)
		ON CONFLICT (user_id, provider, external_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			snippet = EXCLUDED.snippet,
			is_read = EXCLUDED.is_read,
			is_starred = EXCLUDED.is_starred,
			is_important = EXCLUDED.is_important,
			is_archived = EXCLUDED.is_archived,
			is_trash = EXCLUDED.is_trash,
			labels = EXCLUDED.labels,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, msg.Id, msg.ThreadId,
		subject, msg.Snippet, fromEmail, fromName,
		toAddrs, ccAddrs, replyTo,
		bodyText, bodyHTML, attachments,
		isRead, isStarred, isImportant, isDraft, isSent, isArchived, isTrash,
		labels, emailDate, emailDate)

	if err != nil {
		return err
	}

	// Fire the engine-sync hook so the email lands in the OptimalEngine
	// knowledge graph. We refetch by external_id to get the canonical
	// DB UUID (the engine uses it as the Signal ID, so re-syncs hit the
	// same MD file).
	if s.OnEmailSaved != nil {
		if email, ferr := s.getEmailByExternalID(ctx, userID, msg.Id); ferr == nil && email != nil {
			s.OnEmailSaved(ctx, email, userID)
		}
	}

	return nil
}

// getEmailByExternalID fetches a single email row by its provider-side
// external ID (Gmail message ID). Used by the engine-sync hook to find
// the canonical DB UUID after an upsert.
func (s *GmailService) getEmailByExternalID(ctx context.Context, userID, externalID string) (*Email, error) {
	var e Email
	var toEmails, ccEmails, attachments, labels []byte
	var bodyText, bodyHTML, replyTo pgtype.Text
	var date, receivedAt pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, provider, external_id, thread_id,
			subject, snippet, from_email, from_name,
			to_emails, cc_emails, reply_to,
			body_text, body_html, attachments,
			is_read, is_starred, is_important, is_draft, is_sent, is_archived, is_trash,
			labels, date, received_at
		FROM emails
		WHERE user_id = $1 AND provider = 'gmail' AND external_id = $2
	`, userID, externalID).Scan(
		&e.ID, &e.UserID, &e.Provider, &e.ExternalID, &e.ThreadID,
		&e.Subject, &e.Snippet, &e.FromEmail, &e.FromName,
		&toEmails, &ccEmails, &replyTo,
		&bodyText, &bodyHTML, &attachments,
		&e.IsRead, &e.IsStarred, &e.IsImportant, &e.IsDraft, &e.IsSent, &e.IsArchived, &e.IsTrash,
		&labels, &date, &receivedAt,
	)
	if err != nil {
		return nil, err
	}

	e.BodyText = bodyText.String
	e.BodyHTML = bodyHTML.String
	e.ReplyTo = replyTo.String
	if date.Valid {
		e.Date = date.Time
	}
	if receivedAt.Valid {
		e.ReceivedAt = receivedAt.Time
	}
	return &e, nil
}

// GetEmails retrieves emails for a user.
func (s *GmailService) GetEmails(ctx context.Context, userID string, folder EmailFolder, limit, offset int) ([]*Email, error) {
	query := `
		SELECT id, user_id, provider, external_id, thread_id,
			subject, snippet, from_email, from_name,
			to_emails, cc_emails, reply_to,
			body_text, body_html, attachments,
			is_read, is_starred, is_important, is_draft, is_sent, is_archived, is_trash,
			labels, date, received_at
		FROM emails
		WHERE user_id = $1
	`

	switch folder {
	case FolderInbox:
		query += " AND is_archived = false AND is_trash = false AND is_draft = false"
	case FolderSent:
		query += " AND is_sent = true"
	case FolderDrafts:
		query += " AND is_draft = true"
	case FolderStarred:
		query += " AND is_starred = true"
	case FolderArchive:
		query += " AND is_archived = true"
	case FolderTrash:
		query += " AND is_trash = true"
	}

	query += " ORDER BY date DESC LIMIT $2 OFFSET $3"

	rows, err := s.provider.Pool().Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []*Email
	for rows.Next() {
		var e Email
		var toEmails, ccEmails, attachments, labels []byte
		var bodyText, bodyHTML, replyTo pgtype.Text
		var date, receivedAt pgtype.Timestamptz

		err := rows.Scan(
			&e.ID, &e.UserID, &e.Provider, &e.ExternalID, &e.ThreadID,
			&e.Subject, &e.Snippet, &e.FromEmail, &e.FromName,
			&toEmails, &ccEmails, &replyTo,
			&bodyText, &bodyHTML, &attachments,
			&e.IsRead, &e.IsStarred, &e.IsImportant, &e.IsDraft, &e.IsSent, &e.IsArchived, &e.IsTrash,
			&labels, &date, &receivedAt,
		)
		if err != nil {
			return nil, err
		}

		e.BodyText = bodyText.String
		e.BodyHTML = bodyHTML.String
		e.ReplyTo = replyTo.String
		if date.Valid {
			e.Date = date.Time
		}
		if receivedAt.Valid {
			e.ReceivedAt = receivedAt.Time
		}

		emails = append(emails, &e)
	}

	return emails, nil
}

// GetEmail retrieves a single email by ID.
func (s *GmailService) GetEmail(ctx context.Context, userID, emailID string) (*Email, error) {
	var e Email
	var toEmails, ccEmails, attachments, labels []byte
	var bodyText, bodyHTML, replyTo pgtype.Text
	var date, receivedAt pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, provider, external_id, thread_id,
			subject, snippet, from_email, from_name,
			to_emails, cc_emails, reply_to,
			body_text, body_html, attachments,
			is_read, is_starred, is_important, is_draft, is_sent, is_archived, is_trash,
			labels, date, received_at
		FROM emails
		WHERE id = $1 AND user_id = $2
	`, emailID, userID).Scan(
		&e.ID, &e.UserID, &e.Provider, &e.ExternalID, &e.ThreadID,
		&e.Subject, &e.Snippet, &e.FromEmail, &e.FromName,
		&toEmails, &ccEmails, &replyTo,
		&bodyText, &bodyHTML, &attachments,
		&e.IsRead, &e.IsStarred, &e.IsImportant, &e.IsDraft, &e.IsSent, &e.IsArchived, &e.IsTrash,
		&labels, &date, &receivedAt,
	)
	if err != nil {
		return nil, err
	}

	e.BodyText = bodyText.String
	e.BodyHTML = bodyHTML.String
	e.ReplyTo = replyTo.String
	if date.Valid {
		e.Date = date.Time
	}
	if receivedAt.Valid {
		e.ReceivedAt = receivedAt.Time
	}

	return &e, nil
}

// GetEmailByID retrieves a single email by its database ID.
// Alias for GetEmail for handler compatibility.
func (s *GmailService) GetEmailByID(ctx context.Context, userID, emailID string) (*Email, error) {
	return s.GetEmail(ctx, userID, emailID)
}

// IsConnected checks if Gmail is connected for a user.
func (s *GmailService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if strings.Contains(scope, "gmail") {
			return true
		}
	}
	return false
}
