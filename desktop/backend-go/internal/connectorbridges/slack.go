// slack_connector.go — Slack fetcher for the OptimalEngine connector layer.
//
// Implements adapters.SlackFetcher on top of the existing slack.Provider
// (used by BusinessOS for OAuth + token storage). We hit Slack's public
// HTTP API directly rather than going through MessageService because that
// service writes to BusinessOS-specific tables; the connector pipeline
// just needs the raw conversations.history payload to feed Transform.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/slack"
)

// SlackFetcher pulls recent messages across the user's accessible
// channels via the Slack Web API. Cursor format: opaque next_cursor from
// conversations.history; empty = start from newest.
type SlackFetcher struct {
	prov   *slack.Provider
	client *http.Client
}

func NewSlackFetcher(prov *slack.Provider) *SlackFetcher {
	return &SlackFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListMessages pulls the most recent batch of messages from any channel
// the user has joined. Phase 1 strategy: list channels, walk the first one
// with new messages. The cursor encodes (channel_id|next_ts) so subsequent
// calls page through the same channel before moving on. Multi-channel fan-
// out lands in a follow-up — keeps the contract narrow for now.
func (f *SlackFetcher) ListMessages(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("slack: token: %w", err)
	}
	channelID, pageCursor := splitSlackCursor(cursor)
	if channelID == "" {
		channelID, err = f.firstJoinedChannel(ctx, tok.AccessToken)
		if err != nil {
			return nil, "", err
		}
		if channelID == "" {
			return nil, "", nil // no channels joined → caught up
		}
	}

	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("limit", fmt.Sprintf("%d", max))
	if pageCursor != "" {
		params.Set("cursor", pageCursor)
	}
	body, err := f.do(ctx, tok.AccessToken, "https://slack.com/api/conversations.history?"+params.Encode())
	if err != nil {
		return nil, "", err
	}
	var resp struct {
		OK           bool             `json:"ok"`
		Error        string           `json:"error"`
		Messages     []map[string]any `json:"messages"`
		ResponseMeta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"response_metadata"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("slack: decode: %w", err)
	}
	if !resp.OK {
		return nil, "", fmt.Errorf("slack: %s", resp.Error)
	}
	next := ""
	if resp.ResponseMeta.NextCursor != "" {
		next = channelID + "|" + resp.ResponseMeta.NextCursor
	}
	return resp.Messages, next, nil
}

func (f *SlackFetcher) firstJoinedChannel(ctx context.Context, token string) (string, error) {
	body, err := f.do(ctx, token,
		"https://slack.com/api/conversations.list?types=public_channel,private_channel&exclude_archived=true&limit=20")
	if err != nil {
		return "", err
	}
	var resp struct {
		OK       bool `json:"ok"`
		Channels []struct {
			ID       string `json:"id"`
			IsMember bool   `json:"is_member"`
		} `json:"channels"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("slack: decode channels: %w", err)
	}
	if !resp.OK {
		return "", fmt.Errorf("slack: %s", resp.Error)
	}
	for _, c := range resp.Channels {
		if c.IsMember {
			return c.ID, nil
		}
	}
	return "", nil
}

func (f *SlackFetcher) do(ctx context.Context, token, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// splitSlackCursor decodes the "channel|cursor" form. Empty input or
// missing pipe is fine — we just start with no per-channel cursor.
func splitSlackCursor(s string) (channel, cursor string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
