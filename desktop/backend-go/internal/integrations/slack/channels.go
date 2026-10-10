package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Channel represents a Slack channel.
type Channel struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SlackID      string    `json:"slack_id"`
	Name         string    `json:"name"`
	IsPrivate    bool      `json:"is_private"`
	IsDM         bool      `json:"is_dm"`
	MemberCount  int       `json:"member_count"`
	Topic        string    `json:"topic,omitempty"`
	Purpose      string    `json:"purpose,omitempty"`
	UnreadCount  int       `json:"unread_count"`
	LastActivity *time.Time `json:"last_activity,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChannelService handles Slack channel operations.
type ChannelService struct {
	provider *Provider
}

// NewChannelService creates a new channel service.
func NewChannelService(provider *Provider) *ChannelService {
	return &ChannelService{provider: provider}
}

// SyncChannelsResult represents the result of a channel sync.
type SyncChannelsResult struct {
	TotalChannels  int `json:"total_channels"`
	SyncedChannels int `json:"synced_channels"`
	FailedChannels int `json:"failed_channels"`
}

// SyncChannels syncs channels from Slack.
func (s *ChannelService) SyncChannels(ctx context.Context, userID string) (*SyncChannelsResult, error) {
	slog.Info("Slack channel sync starting for user", "user_id", userID)

	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	result := &SyncChannelsResult{}

	// Get public channels
	publicChannels, err := s.fetchChannels(ctx, token.AccessToken, false)
	if err != nil {
		slog.Info("Failed to fetch public channels", "error", err)
	} else {
		result.TotalChannels += len(publicChannels)
		for _, ch := range publicChannels {
			if err := s.saveChannel(ctx, userID, ch, false, false); err != nil {
				slog.Info("Failed to save channel", "id", ch.ID, "error", err)
				result.FailedChannels++
			} else {
				result.SyncedChannels++
			}
		}
	}

	// Get private channels
	privateChannels, err := s.fetchChannels(ctx, token.AccessToken, true)
	if err != nil {
		slog.Info("Failed to fetch private channels", "error", err)
	} else {
		result.TotalChannels += len(privateChannels)
		for _, ch := range privateChannels {
			if err := s.saveChannel(ctx, userID, ch, true, false); err != nil {
				slog.Info("Failed to save channel", "id", ch.ID, "error", err)
				result.FailedChannels++
			} else {
				result.SyncedChannels++
			}
		}
	}

	// Get DMs
	dms, err := s.fetchDMs(ctx, token.AccessToken)
	if err != nil {
		slog.Info("Failed to fetch DMs", "error", err)
	} else {
		result.TotalChannels += len(dms)
		for _, dm := range dms {
			if err := s.saveChannel(ctx, userID, dm, false, true); err != nil {
				slog.Info("Failed to save DM", "id", dm.ID, "error", err)
				result.FailedChannels++
			} else {
				result.SyncedChannels++
			}
		}
	}

	slog.Info("Slack channel sync complete for user : synced / channels", "user_id", userID, "synced", result.SyncedChannels, "total", result.TotalChannels)

	return result, nil
}

// fetchChannels fetches channels from Slack API.
func (s *ChannelService) fetchChannels(ctx context.Context, token string, private bool) ([]SlackChannel, error) {
	endpoint := "https://slack.com/api/conversations.list"
	params := url.Values{}
	params.Set("types", "public_channel")
	if private {
		params.Set("types", "private_channel")
	}
	params.Set("limit", "200")

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
		Channels []SlackChannel `json:"channels"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Channels, nil
}

// fetchDMs fetches direct messages from Slack API.
func (s *ChannelService) fetchDMs(ctx context.Context, token string) ([]SlackChannel, error) {
	endpoint := "https://slack.com/api/conversations.list"
	params := url.Values{}
	params.Set("types", "im,mpim")
	params.Set("limit", "200")

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
		Channels []SlackChannel `json:"channels"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Channels, nil
}

// SlackChannel represents a channel from the Slack API.
type SlackChannel struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	IsPrivate      bool   `json:"is_private"`
	IsIM           bool   `json:"is_im"`
	IsMpIM         bool   `json:"is_mpim"`
	NumMembers     int    `json:"num_members"`
	Topic          Topic  `json:"topic"`
	Purpose        Topic  `json:"purpose"`
	UnreadCount    int    `json:"unread_count"`
	LastRead       string `json:"last_read"`
	User           string `json:"user"` // For DMs
}

// Topic represents a channel topic or purpose.
type Topic struct {
	Value string `json:"value"`
}

// saveChannel saves a Slack channel to the database.
func (s *ChannelService) saveChannel(ctx context.Context, userID string, ch SlackChannel, isPrivate, isDM bool) error {
	name := ch.Name
	if isDM && name == "" {
		name = "Direct Message"
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO slack_channels (
			user_id, slack_id, name, is_private, is_dm,
			member_count, topic, purpose, unread_count, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_id, slack_id) DO UPDATE SET
			name = EXCLUDED.name,
			is_private = EXCLUDED.is_private,
			is_dm = EXCLUDED.is_dm,
			member_count = EXCLUDED.member_count,
			topic = EXCLUDED.topic,
			purpose = EXCLUDED.purpose,
			unread_count = EXCLUDED.unread_count,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, ch.ID, name, isPrivate || ch.IsPrivate, isDM || ch.IsIM || ch.IsMpIM,
		ch.NumMembers, ch.Topic.Value, ch.Purpose.Value, ch.UnreadCount)

	return err
}

// ============================================
// MCP-Compatible Methods (Slack API Direct)
// These methods make direct API calls for MCP tools.
// ============================================

// ListChannels returns channels directly from Slack API for MCP tools.
func (s *ChannelService) ListChannels(ctx context.Context, userID string, limit int) ([]SlackChannel, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	channels, err := s.fetchChannels(ctx, token.AccessToken, false)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(channels) > limit {
		channels = channels[:limit]
	}

	return channels, nil
}

// SendMessage sends a message to a Slack channel and returns the timestamp.
func (s *ChannelService) SendMessage(ctx context.Context, userID, channelID, text, threadTS string) (string, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("text", text)
	if threadTS != "" {
		params.Set("thread_ts", threadTS)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error"`
		Timestamp string `json:"ts"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if !result.OK {
		return "", fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Timestamp, nil
}

// MCPSlackMessage represents a message from Slack API (for MCP tools).
type MCPSlackMessage struct {
	Timestamp       string `json:"ts"`
	User            string `json:"user"`
	Text            string `json:"text"`
	ThreadTimestamp string `json:"thread_ts,omitempty"`
	ReplyCount      int    `json:"reply_count,omitempty"`
}

// GetChannelHistory returns recent messages from a channel via Slack API.
func (s *ChannelService) GetChannelHistory(ctx context.Context, userID, channelID string, limit int) ([]MCPSlackMessage, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/api/conversations.history?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK       bool              `json:"ok"`
		Error    string            `json:"error"`
		Messages []MCPSlackMessage `json:"messages"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Messages, nil
}

// SlackSearchResult represents a search result from Slack API.
type SlackSearchResult struct {
	Total   int                  `json:"total"`
	Matches []SlackSearchMessage `json:"matches"`
}

// SlackSearchMessage represents a message in search results.
type SlackSearchMessage struct {
	Timestamp string `json:"ts"`
	User      string `json:"user"`
	Username  string `json:"username"`
	Text      string `json:"text"`
}

// SearchMessages searches for messages in Slack workspace.
func (s *ChannelService) SearchMessages(ctx context.Context, userID, query string, count int) (*SlackSearchResult, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/api/search.messages?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK       bool `json:"ok"`
		Error    string `json:"error"`
		Messages struct {
			Total   int `json:"total"`
			Matches []SlackSearchMessage `json:"matches"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return &SlackSearchResult{
		Total:   result.Messages.Total,
		Matches: result.Messages.Matches,
	}, nil
}

// SlackUser represents a user from Slack API.
type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	IsAdmin  bool   `json:"is_admin"`
	Profile  SlackUserProfile `json:"profile"`
}

// SlackUserProfile represents user profile info.
type SlackUserProfile struct {
	Email      string `json:"email"`
	Title      string `json:"title"`
	StatusText string `json:"status_text"`
}

// ListUsers returns users from the Slack workspace.
func (s *ChannelService) ListUsers(ctx context.Context, userID string, limit int) ([]SlackUser, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/api/users.list?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Members []SlackUser `json:"members"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.Members, nil
}

// GetUserInfo returns info for a specific Slack user.
func (s *ChannelService) GetUserInfo(ctx context.Context, userID, slackUserID string) (*SlackUser, error) {
	token, err := s.provider.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	params := url.Values{}
	params.Set("user", slackUserID)

	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/api/users.info?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK    bool       `json:"ok"`
		Error string     `json:"error"`
		User  *SlackUser `json:"user"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.User, nil
}

// GetChannels retrieves channels for a user.
func (s *ChannelService) GetChannels(ctx context.Context, userID string, includePrivate, includeDMs bool) ([]*Channel, error) {
	query := `
		SELECT id, user_id, slack_id, name, is_private, is_dm,
			member_count, topic, purpose, unread_count, last_activity,
			created_at, updated_at
		FROM slack_channels
		WHERE user_id = $1
	`

	if !includePrivate {
		query += " AND is_private = false"
	}
	if !includeDMs {
		query += " AND is_dm = false"
	}

	query += " ORDER BY last_activity DESC NULLS LAST, name"

	rows, err := s.provider.Pool().Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		var ch Channel
		err := rows.Scan(
			&ch.ID, &ch.UserID, &ch.SlackID, &ch.Name, &ch.IsPrivate, &ch.IsDM,
			&ch.MemberCount, &ch.Topic, &ch.Purpose, &ch.UnreadCount, &ch.LastActivity,
			&ch.CreatedAt, &ch.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, &ch)
	}

	return channels, nil
}
