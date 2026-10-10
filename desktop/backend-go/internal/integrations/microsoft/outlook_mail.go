package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SyncMessagesResult represents the result of a message sync.
type SyncMessagesResult struct {
	TotalMessages  int `json:"total_messages"`
	SyncedMessages int `json:"synced_messages"`
	FailedMessages int `json:"failed_messages"`
}

// Graph API message structure
type graphMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	Subject        string `json:"subject"`
	BodyPreview    string `json:"bodyPreview"`
	Body           *struct {
		ContentType string `json:"contentType"` // "text" | "html"
		Content     string `json:"content"`
	} `json:"body"`
	Importance string `json:"importance"`
	From       *struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"from"`
	ToRecipients []struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"toRecipients"`
	CcRecipients []struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"ccRecipients"`
	IsRead         bool     `json:"isRead"`
	IsDraft        bool     `json:"isDraft"`
	HasAttachments bool     `json:"hasAttachments"`
	ParentFolderID string   `json:"parentFolderId"`
	Categories     []string `json:"categories"`
	Flag           struct {
		FlagStatus string `json:"flagStatus"`
	} `json:"flag"`
	ReceivedDateTime string `json:"receivedDateTime"`
	SentDateTime     string `json:"sentDateTime"`
}

// SyncMessages syncs messages from Outlook.
func (s *OutlookService) SyncMessages(ctx context.Context, userID string, maxResults int) (*SyncMessagesResult, error) {
	slog.Info("Outlook mail sync starting for user : max  messages", "user_id", userID, "max_results", maxResults)

	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	result := &SyncMessagesResult{}
	skip := 0

	for {
		// Build URL — fetch full body so it's available for the unified inbox
		// preview pane and the engine-sync RAG indexing path.
		apiURL := fmt.Sprintf("%s/me/messages?$top=%d&$skip=%d&$orderby=receivedDateTime desc&$select=id,conversationId,subject,bodyPreview,body,importance,from,toRecipients,ccRecipients,isRead,isDraft,hasAttachments,parentFolderId,categories,flag,receivedDateTime,sentDateTime",
			GraphAPIBase, maxResults, skip)

		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get messages: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API error: %s", resp.Status)
		}

		var msgResp struct {
			Value    []graphMessage `json:"value"`
			NextLink string         `json:"@odata.nextLink"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		result.TotalMessages += len(msgResp.Value)

		for _, msg := range msgResp.Value {
			if err := s.saveMessage(ctx, userID, &msg); err != nil {
				slog.Info("Failed to save message", "id", msg.ID, "error", err)
				result.FailedMessages++
			} else {
				result.SyncedMessages++
			}
		}

		// Check if we should continue
		if msgResp.NextLink == "" || result.TotalMessages >= maxResults {
			break
		}
		skip += len(msgResp.Value)
	}

	slog.Info("Outlook mail sync complete for user : synced / messages", "user_id", userID, "synced", result.SyncedMessages, "total", result.TotalMessages)

	return result, nil
}

func (s *OutlookService) saveMessage(ctx context.Context, userID string, msg *graphMessage) error {
	// Parse recipients
	toRecipients := make([]EmailRecipient, 0)
	for _, r := range msg.ToRecipients {
		toRecipients = append(toRecipients, EmailRecipient{
			Email: r.EmailAddress.Address,
			Name:  r.EmailAddress.Name,
		})
	}

	ccRecipients := make([]EmailRecipient, 0)
	for _, r := range msg.CcRecipients {
		ccRecipients = append(ccRecipients, EmailRecipient{
			Email: r.EmailAddress.Address,
			Name:  r.EmailAddress.Name,
		})
	}

	// Parse dates
	var receivedDateTime, sentDateTime *time.Time
	if msg.ReceivedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, msg.ReceivedDateTime)
		receivedDateTime = &t
	}
	if msg.SentDateTime != "" {
		t, _ := time.Parse(time.RFC3339, msg.SentDateTime)
		sentDateTime = &t
	}

	// Extract from info
	var fromEmail, fromName string
	if msg.From != nil {
		fromEmail = msg.From.EmailAddress.Address
		fromName = msg.From.EmailAddress.Name
	}

	// Extract body content + content type
	var bodyContent, bodyContentType string
	if msg.Body != nil {
		bodyContent = msg.Body.Content
		bodyContentType = msg.Body.ContentType
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_mail_messages (
			user_id, message_id, conversation_id, subject, body_preview, body_content, body_content_type, importance,
			from_email, from_name, to_recipients, cc_recipients,
			is_read, is_draft, has_attachments, folder_id, categories, flag_status,
			received_datetime, sent_datetime, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW())
		ON CONFLICT (user_id, message_id) DO UPDATE SET
			conversation_id = EXCLUDED.conversation_id,
			subject = EXCLUDED.subject,
			body_preview = EXCLUDED.body_preview,
			body_content = EXCLUDED.body_content,
			body_content_type = EXCLUDED.body_content_type,
			importance = EXCLUDED.importance,
			from_email = EXCLUDED.from_email,
			from_name = EXCLUDED.from_name,
			to_recipients = EXCLUDED.to_recipients,
			cc_recipients = EXCLUDED.cc_recipients,
			is_read = EXCLUDED.is_read,
			is_draft = EXCLUDED.is_draft,
			has_attachments = EXCLUDED.has_attachments,
			folder_id = EXCLUDED.folder_id,
			categories = EXCLUDED.categories,
			flag_status = EXCLUDED.flag_status,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, msg.ID, msg.ConversationID, msg.Subject, msg.BodyPreview, bodyContent, bodyContentType, msg.Importance,
		fromEmail, fromName, toRecipients, ccRecipients,
		msg.IsRead, msg.IsDraft, msg.HasAttachments, msg.ParentFolderID, msg.Categories, msg.Flag.FlagStatus,
		receivedDateTime, sentDateTime)

	if err != nil {
		return err
	}

	// Fire the engine-sync hook so the email lands in the OptimalEngine
	// knowledge graph. Refetch by (user_id, message_id) so the engine
	// Signal carries the canonical DB UUID — re-syncs hit the same MD
	// file. Mirrors the Gmail pattern in google/gmail_read.go::saveEmail.
	if s.OnEmailSaved != nil {
		if saved, ferr := s.getMessageByMessageID(ctx, userID, msg.ID); ferr == nil && saved != nil {
			s.OnEmailSaved(ctx, saved, userID)
		}
	}

	return nil
}

// FetchAndSaveMessage fetches a single message from Graph by its id,
// upserts it into microsoft_mail_messages, and fires OnEmailSaved.
// Used by the Graph webhook handler — every notification gives us only
// {resource, resourceData.id, changeType}, never the full body, so we
// have to GET the resource ourselves before persisting.
func (s *OutlookService) FetchAndSaveMessage(ctx context.Context, userID, messageID string) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get HTTP client: %w", err)
	}

	apiURL := fmt.Sprintf("%s/me/messages/%s?$select=id,conversationId,subject,bodyPreview,body,importance,from,toRecipients,ccRecipients,isRead,isDraft,hasAttachments,parentFolderId,categories,flag,receivedDateTime,sentDateTime",
		GraphAPIBase, messageID)

	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("graph fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graph fetch: %s", resp.Status)
	}

	var msg graphMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	return s.saveMessage(ctx, userID, &msg)
}

// getMessageByMessageID fetches a single message row by its Graph
// message ID, used by saveMessage to deliver the canonical DB UUID
// to the OnEmailSaved hook.
func (s *OutlookService) getMessageByMessageID(ctx context.Context, userID, messageID string) (*OutlookMessage, error) {
	var (
		m               OutlookMessage
		conversationID  *string
		bodyPreview     *string
		bodyContent     *string
		bodyContentType *string
		importance      *string
		folderID        *string
		flagStatus      *string
		categoriesRaw   []byte
		receivedAt      *time.Time
		sentAt          *time.Time
	)
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, message_id, conversation_id,
			subject, body_preview, body_content, body_content_type, importance,
			from_email, from_name,
			is_read, is_draft, has_attachments, folder_id, categories, flag_status,
			received_datetime, sent_datetime, synced_at
		FROM microsoft_mail_messages
		WHERE user_id = $1 AND message_id = $2
	`, userID, messageID).Scan(
		&m.ID, &m.UserID, &m.MessageID, &conversationID,
		&m.Subject, &bodyPreview, &bodyContent, &bodyContentType, &importance,
		&m.FromEmail, &m.FromName,
		&m.IsRead, &m.IsDraft, &m.HasAttachments, &folderID, &categoriesRaw, &flagStatus,
		&receivedAt, &sentAt, &m.SyncedAt,
	)
	if err != nil {
		return nil, err
	}
	if conversationID != nil {
		m.ConversationID = *conversationID
	}
	if bodyPreview != nil {
		m.BodyPreview = *bodyPreview
	}
	if bodyContent != nil {
		m.BodyContent = *bodyContent
	}
	if bodyContentType != nil {
		m.BodyContentType = *bodyContentType
	}
	if importance != nil {
		m.Importance = *importance
	}
	if folderID != nil {
		m.FolderID = *folderID
	}
	if flagStatus != nil {
		m.FlagStatus = *flagStatus
	}
	if len(categoriesRaw) > 0 {
		_ = json.Unmarshal(categoriesRaw, &m.Categories)
	}
	if receivedAt != nil {
		m.ReceivedDateTime = *receivedAt
	}
	if sentAt != nil {
		m.SentDateTime = *sentAt
	}
	return &m, nil
}

// GetMessages retrieves messages for a user.
func (s *OutlookService) GetMessages(ctx context.Context, userID string, folderID string, limit, offset int) ([]*OutlookMessage, error) {
	query := `
		SELECT id, user_id, message_id, conversation_id, subject, body_preview, body_content, body_content_type, importance,
			from_email, from_name, is_read, is_draft, has_attachments, folder_id, categories, flag_status,
			received_datetime, sent_datetime, synced_at
		FROM microsoft_mail_messages
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if folderID != "" {
		query += " AND folder_id = $2 ORDER BY received_datetime DESC NULLS LAST LIMIT $3 OFFSET $4"
		args = append(args, folderID, limit, offset)
	} else {
		query += " ORDER BY received_datetime DESC NULLS LAST LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := s.provider.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*OutlookMessage
	for rows.Next() {
		var m OutlookMessage
		var conversationID, bodyPreview, bodyContent, bodyContentType, importance, folderID, flagStatus *string
		var categories []string
		var receivedDateTime, sentDateTime *time.Time

		err := rows.Scan(
			&m.ID, &m.UserID, &m.MessageID, &conversationID, &m.Subject, &bodyPreview, &bodyContent, &bodyContentType, &importance,
			&m.FromEmail, &m.FromName, &m.IsRead, &m.IsDraft, &m.HasAttachments, &folderID, &categories, &flagStatus,
			&receivedDateTime, &sentDateTime, &m.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		if conversationID != nil {
			m.ConversationID = *conversationID
		}
		if bodyPreview != nil {
			m.BodyPreview = *bodyPreview
		}
		if bodyContent != nil {
			m.BodyContent = *bodyContent
		}
		if bodyContentType != nil {
			m.BodyContentType = *bodyContentType
		}
		if importance != nil {
			m.Importance = *importance
		}
		if folderID != nil {
			m.FolderID = *folderID
		}
		if flagStatus != nil {
			m.FlagStatus = *flagStatus
		}
		m.Categories = categories
		if receivedDateTime != nil {
			m.ReceivedDateTime = *receivedDateTime
		}
		if sentDateTime != nil {
			m.SentDateTime = *sentDateTime
		}

		messages = append(messages, &m)
	}

	return messages, nil
}

// SendMessage sends an email via Outlook.
func (s *OutlookService) SendMessage(ctx context.Context, userID string, to []string, subject, body string, isHTML bool) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return err
	}

	// Build recipients
	toRecipients := make([]map[string]interface{}, 0)
	for _, email := range to {
		toRecipients = append(toRecipients, map[string]interface{}{
			"emailAddress": map[string]string{
				"address": email,
			},
		})
	}

	contentType := "text"
	if isHTML {
		contentType = "html"
	}

	message := map[string]interface{}{
		"message": map[string]interface{}{
			"subject": subject,
			"body": map[string]string{
				"contentType": contentType,
				"content":     body,
			},
			"toRecipients": toRecipients,
		},
		"saveToSentItems": true,
	}

	jsonBody, _ := json.Marshal(message)

	resp, err := client.Post(GraphAPIBase+"/me/sendMail", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	return nil
}
