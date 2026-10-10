package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Message represents a Slack message.
type Message struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ChannelID   string    `json:"channel_id"`
	SlackTS     string    `json:"slack_ts"`
	SenderID    string    `json:"sender_id"`
	SenderName  string    `json:"sender_name"`
	Content     string    `json:"content"`
	ThreadTS    string    `json:"thread_ts,omitempty"`
	ReplyCount  int       `json:"reply_count"`
	Attachments []any     `json:"attachments,omitempty"`
	IsEdited    bool      `json:"is_edited"`
	SentAt      time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MessageSavedHook is called after a Slack message is saved to the
// database. Used by handlers to mirror the message into the OptimalEngine
// knowledge graph. nil = no-op.
type MessageSavedHook func(ctx context.Context, msg *Message, userID string)

// MessageService handles Slack message operations.
type MessageService struct {
	provider *Provider

	// OnMessageSaved is fired after a message is upserted into the
	// `slack_messages` table. Set externally — nil when the engine
	// isn't wired, in which case sync continues without indexing.
	OnMessageSaved MessageSavedHook
}

// NewMessageService creates a new message service.
func NewMessageService(provider *Provider) *MessageService {
	return &MessageService{provider: provider}
}

// SyncMessagesResult represents the result of a message sync.
type SyncMessagesResult struct {
	TotalMessages  int `json:"total_messages"`
	SyncedMessages int `json:"synced_messages"`
	FailedMessages int `json:"failed_messages"`
}

// SyncMessages syncs messages for a channel from Slack.
func (s *MessageService) SyncMessages(ctx context.Context, userID, channelID string, limit int) (*SyncMessagesResult, error) {
	slog.Info("Slack message sync starting for user , channel", "value0", userID, "value1", channelID)

	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Get the Slack channel ID
	var slackChannelID string
	err = s.provider.Pool().QueryRow(ctx, `
		SELECT slack_id FROM slack_channels WHERE id = $1 AND user_id = $2
	`, channelID, userID).Scan(&slackChannelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	messages, err := s.fetchMessages(ctx, token.AccessToken, slackChannelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	result := &SyncMessagesResult{
		TotalMessages: len(messages),
	}

	for _, msg := range messages {
		if err := s.SaveMessage(ctx, userID, channelID, msg); err != nil {
			slog.Info("Failed to save message", "id", msg.TS, "error", err)
			result.FailedMessages++
		} else {
			result.SyncedMessages++
		}
	}

	slog.Info("Slack message sync complete: synced / messages", "value0", result.SyncedMessages, "value1", result.TotalMessages)

	return result, nil
}

// SlackMessage represents a message from the Slack API.
type SlackMessage struct {
	TS          string `json:"ts"`
	User        string `json:"user"`
	Text        string `json:"text"`
	ThreadTS    string `json:"thread_ts"`
	ReplyCount  int    `json:"reply_count"`
	Attachments []any  `json:"attachments"`
	Edited      *struct {
		User string `json:"user"`
		TS   string `json:"ts"`
	} `json:"edited"`
}

// fetchMessages fetches messages from Slack API.
func (s *MessageService) fetchMessages(ctx context.Context, token, channelID string, limit int) ([]SlackMessage, error) {
	endpoint := "https://slack.com/api/conversations.history"
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK       bool           `json:"ok"`
		Error    string         `json:"error"`
		Messages []SlackMessage `json:"messages"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Messages, nil
}

// SaveMessage upserts a Slack message into slack_messages and fires the
// OnMessageSaved hook (engine sync). Exported so webhook handlers
// (internal/handlers/comms_webhooks_slack.go) can persist a single
// message from an Events API payload — sync code paths use the same
// upsert so realtime + manual sync stay byte-identical.
//
// channelID is the BO uuid of the parent slack_channels row.
func (s *MessageService) SaveMessage(ctx context.Context, userID, channelID string, msg SlackMessage) error {
	// Parse timestamp to time
	sentAt := parseSlackTS(msg.TS)
	isEdited := msg.Edited != nil

	// Get sender name (would need to look up from users, using ID for now)
	senderName := msg.User

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO slack_messages (
			user_id, channel_id, slack_ts, sender_id, sender_name,
			content, thread_ts, reply_count, is_edited, sent_at, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (user_id, channel_id, slack_ts) DO UPDATE SET
			content = EXCLUDED.content,
			reply_count = EXCLUDED.reply_count,
			is_edited = EXCLUDED.is_edited,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, channelID, msg.TS, msg.User, senderName,
		msg.Text, msg.ThreadTS, msg.ReplyCount, isEdited, sentAt)

	if err != nil {
		return err
	}

	// Fire the engine-sync hook so the message lands in the OptimalEngine
	// knowledge graph. Refetch to get the canonical DB UUID.
	if s.OnMessageSaved != nil {
		if saved, ferr := s.getMessageByTS(ctx, userID, channelID, msg.TS); ferr == nil && saved != nil {
			s.OnMessageSaved(ctx, saved, userID)
		}
	}

	return nil
}

// getMessageByTS fetches a single message row by its (user, channel,
// slack_ts) tuple. Used by the engine-sync hook to find the canonical
// DB UUID after an upsert.
func (s *MessageService) getMessageByTS(ctx context.Context, userID, channelID, slackTS string) (*Message, error) {
	var msg Message
	var threadTS pgtype.Text

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, channel_id, slack_ts, sender_id, sender_name,
			content, thread_ts, reply_count, is_edited, sent_at,
			created_at, updated_at
		FROM slack_messages
		WHERE user_id = $1 AND channel_id = $2 AND slack_ts = $3
	`, userID, channelID, slackTS).Scan(
		&msg.ID, &msg.UserID, &msg.ChannelID, &msg.SlackTS, &msg.SenderID, &msg.SenderName,
		&msg.Content, &threadTS, &msg.ReplyCount, &msg.IsEdited, &msg.SentAt,
		&msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.ThreadTS = threadTS.String
	return &msg, nil
}

// parseSlackTS parses a Slack timestamp to time.Time.
func parseSlackTS(ts string) time.Time {
	// Slack timestamps are Unix timestamps with microseconds: "1234567890.123456"
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return time.Now()
	}

	var sec, nsec int64
	fmt.Sscanf(parts[0], "%d", &sec)
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &nsec)
		nsec *= 1000 // Convert microseconds to nanoseconds
	}

	return time.Unix(sec, nsec)
}

// GetMessages retrieves messages for a channel.
func (s *MessageService) GetMessages(ctx context.Context, userID, channelID string, limit, offset int) ([]*Message, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, channel_id, slack_ts, sender_id, sender_name,
			content, thread_ts, reply_count, is_edited, sent_at,
			created_at, updated_at
		FROM slack_messages
		WHERE user_id = $1 AND channel_id = $2
		ORDER BY sent_at DESC
		LIMIT $3 OFFSET $4
	`, userID, channelID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var threadTS pgtype.Text

		err := rows.Scan(
			&msg.ID, &msg.UserID, &msg.ChannelID, &msg.SlackTS, &msg.SenderID, &msg.SenderName,
			&msg.Content, &threadTS, &msg.ReplyCount, &msg.IsEdited, &msg.SentAt,
			&msg.CreatedAt, &msg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		msg.ThreadTS = threadTS.String
		messages = append(messages, &msg)
	}

	return messages, nil
}

// SendMessage sends a message to a Slack channel.
func (s *MessageService) SendMessage(ctx context.Context, userID, channelID, text string) error {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Get the Slack channel ID
	var slackChannelID string
	err = s.provider.Pool().QueryRow(ctx, `
		SELECT slack_id FROM slack_channels WHERE id = $1 AND user_id = $2
	`, channelID, userID).Scan(&slackChannelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	// Send message via Slack API
	data := url.Values{}
	data.Set("channel", slackChannelID)
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage",
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		TS    string `json:"ts"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("slack api error: %s", result.Error)
	}

	slog.Info("Sent message to channel , ts", "value0", slackChannelID, "value1", result.TS)
	return nil
}
