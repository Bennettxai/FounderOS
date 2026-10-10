package google

import (
	"context"

	"google.golang.org/api/gmail/v1"
)

// MarkAsRead marks an email as read.
func (s *GmailService) MarkAsRead(ctx context.Context, userID, emailID string) error {
	_, err := s.provider.Pool().Exec(ctx, `
		UPDATE emails SET is_read = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, emailID, userID)
	if err != nil {
		return err
	}

	email, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return nil
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return nil
	}

	_, err = srv.Users.Messages.Modify("me", email.ExternalID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Do()

	return err
}

// ArchiveEmail archives an email by removing it from inbox.
func (s *GmailService) ArchiveEmail(ctx context.Context, userID, emailID string) error {
	_, err := s.provider.Pool().Exec(ctx, `
		UPDATE emails SET is_archived = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, emailID, userID)
	if err != nil {
		return err
	}

	email, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return nil
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return nil
	}

	_, err = srv.Users.Messages.Modify("me", email.ExternalID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Do()

	return err
}

// DeleteEmail moves an email to trash.
func (s *GmailService) DeleteEmail(ctx context.Context, userID, emailID string) error {
	email, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return err
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return err
	}

	_, err = srv.Users.Messages.Trash("me", email.ExternalID).Do()
	if err != nil {
		return err
	}

	_, err = s.provider.Pool().Exec(ctx, `
		UPDATE emails SET is_trash = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, emailID, userID)

	return err
}
