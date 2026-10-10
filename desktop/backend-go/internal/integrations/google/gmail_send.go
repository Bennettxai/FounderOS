package google

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// SendEmail sends an email via Gmail.
func (s *GmailService) SendEmail(ctx context.Context, userID string, email *ComposeEmail) error {
	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return err
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	if len(email.Cc) > 0 {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.Cc, ", ")))
	}
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	if email.IsHTML {
		msgBuilder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msgBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(email.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	_, err = srv.Users.Messages.Send("me", &gmail.Message{
		Raw: raw,
	}).Do()

	return err
}
