// Package slack ports FounderOS v1's lib/connectors/slack.ts: the Slack Web API
// status check, the /comms channel board and message feed, and the reply send
// (guarded by FOUNDEROS_WRITES).
package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "slack", Name: "Slack", Kind: connectors.KindSlack}

// ErrNotConfigured means SLACK_BOT_TOKEN is not set: no source, not "no messages".
var ErrNotConfigured = errors.New("SLACK_BOT_TOKEN is not set")

const (
	defaultBaseURL = "https://slack.com/api/"
	tokenKey       = "SLACK_BOT_TOKEN"

	// History is one API call per channel; see slack.ts for why the scan is
	// capped, run in small concurrent waves, and cached.
	maxHistoryChannels = 12
	historyConcurrency = 6
	CacheTTL           = 20 * time.Minute
)

type Message struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
	TS      string `json:"ts"`
}

type Channel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsMember  bool   `json:"isMember"`
	IsPrivate bool   `json:"isPrivate"`
	Members   int    `json:"members"`
	Topic     string `json:"topic"`
}

type SendResult struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type Connector struct {
	res     connectors.Resolver
	baseURL string
	http    *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache *messageCache
}

type messageCache struct {
	at    time.Time
	limit int
	items []Message
}

func New(res connectors.Resolver) *Connector {
	return &Connector{
		res:     res,
		baseURL: defaultBaseURL,
		http:    connectors.HTTPClient(15 * time.Second),
		now:     time.Now,
	}
}

func (c *Connector) token() string { return c.res.Resolve(tokenKey) }

// InvalidateCache drops the message cache so a send or a test sees fresh state.
func (c *Connector) InvalidateCache() {
	c.mu.Lock()
	c.cache = nil
	c.mu.Unlock()
}

func (c *Connector) status(state connectors.State, detail string, meta map[string]any) connectors.Status {
	return connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind, State: state, Detail: detail, Meta: meta}
}

// Status calls auth.test. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	tok := c.token()
	if tok == "" {
		return c.status(connectors.StateNotConfigured,
			"Set SLACK_BOT_TOKEN (xoxb-…) in ~/.founderos/.env or under API keys. Needs channels:read, channels:history, users:read scopes.", nil)
	}
	var auth struct {
		Team string `json:"team"`
		User string `json:"user"`
	}
	if err := c.get(ctx, tok, "auth.test", nil, &auth); err != nil {
		return c.status(connectors.StateError, "Token set but auth failed: "+err.Error(), nil)
	}
	return c.status(connectors.StateConnected,
		fmt.Sprintf("Connected to %s as %s", auth.Team, auth.User),
		map[string]any{"team": auth.Team, "user": auth.User})
}

type rawChannel struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	IsArchived bool    `json:"is_archived"`
	IsMember   bool    `json:"is_member"`
	IsPrivate  bool    `json:"is_private"`
	NumMembers int     `json:"num_members"`
	Updated    float64 `json:"updated"`
	Topic      struct {
		Value string `json:"value"`
	} `json:"topic"`
}

type listPage struct {
	Channels []rawChannel `json:"channels"`
	Meta     struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

// ShapeChannels is the pure half of ListChannels: raw conversations.list
// entries to a clean, name-sorted board without archived channels.
func ShapeChannels(raw []rawChannel) []Channel {
	out := []Channel{}
	for _, c := range raw {
		if c.ID == "" || c.Name == "" || c.IsArchived {
			continue
		}
		out = append(out, Channel{ID: c.ID, Name: c.Name, IsMember: c.IsMember, IsPrivate: c.IsPrivate, Members: c.NumMembers, Topic: c.Topic.Value})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if a != b {
			return a < b
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ListChannels returns every current (non-archived) channel, walking every
// page. Unlike the TS, which returns [] on failure, an unreachable Slack comes
// back as an error so no caller can mistake it for an empty workspace.
func (c *Connector) ListChannels(ctx context.Context) ([]Channel, error) {
	tok := c.token()
	if tok == "" {
		return nil, ErrNotConfigured
	}
	var all []rawChannel
	cursor := ""
	for {
		q := url.Values{"types": {"public_channel,private_channel"}, "exclude_archived": {"true"}, "limit": {"200"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page listPage
		if err := c.get(ctx, tok, "conversations.list", q, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Channels...)
		cursor = page.Meta.NextCursor
		if cursor == "" {
			return ShapeChannels(all), nil
		}
	}
}

// RecentMessages is the /comms feed: the newest messages across the 12 most
// recently active joined channels, newest first. Cached for 20 minutes; a
// cached scan at a larger limit answers a smaller one. Only non-empty results
// are cached, so an outage retries instead of pinning.
func (c *Connector) RecentMessages(ctx context.Context, limit int) ([]Message, error) {
	c.mu.Lock()
	if mc := c.cache; mc != nil && mc.limit >= limit && c.now().Sub(mc.at) < CacheTTL {
		out := append([]Message(nil), mc.items[:min(limit, len(mc.items))]...)
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	tok := c.token()
	if tok == "" {
		return nil, ErrNotConfigured
	}
	// One page, as in the TS: the listing is the whole workspace at limit 200.
	var page listPage
	q := url.Values{"types": {"public_channel,private_channel"}, "exclude_archived": {"true"}, "limit": {"200"}}
	if err := c.get(ctx, tok, "conversations.list", q, &page); err != nil {
		return nil, err
	}
	var joined []rawChannel
	for _, ch := range page.Channels {
		if ch.ID != "" && ch.IsMember {
			joined = append(joined, ch)
		}
	}
	sort.SliceStable(joined, func(i, j int) bool { return joined[i].Updated > joined[j].Updated })
	if len(joined) > maxHistoryChannels {
		joined = joined[:maxHistoryChannels]
	}

	var messages []Message
	for i := 0; i < len(joined); i += historyConcurrency {
		wave := joined[i:min(i+historyConcurrency, len(joined))]
		results := make([][]Message, len(wave))
		var wg sync.WaitGroup
		for k, ch := range wave {
			wg.Add(1)
			go func(k int, ch rawChannel) {
				defer wg.Done()
				results[k] = c.history(ctx, tok, ch)
			}(k, ch)
		}
		wg.Wait()
		for _, r := range results {
			messages = append(messages, r...)
		}
	}
	sort.SliceStable(messages, func(i, j int) bool { return tsNum(messages[i].TS) > tsNum(messages[j].TS) })
	if len(messages) > limit {
		messages = messages[:limit]
	}
	if len(messages) > 0 {
		c.mu.Lock()
		c.cache = &messageCache{at: c.now(), limit: limit, items: append([]Message(nil), messages...)}
		c.mu.Unlock()
	}
	if messages == nil {
		messages = []Message{}
	}
	return messages, nil
}

// history reads one channel; one unreadable channel must not blank the feed.
func (c *Connector) history(ctx context.Context, tok string, ch rawChannel) []Message {
	var h struct {
		Messages []struct {
			User string `json:"user"`
			Text string `json:"text"`
			TS   string `json:"ts"`
		} `json:"messages"`
	}
	if err := c.get(ctx, tok, "conversations.history", url.Values{"channel": {ch.ID}, "limit": {"10"}}, &h); err != nil {
		return nil
	}
	name := ch.Name
	if name == "" {
		name = ch.ID
	}
	out := make([]Message, 0, len(h.Messages))
	for _, m := range h.Messages {
		user := m.User
		if user == "" {
			user = "unknown"
		}
		out = append(out, Message{Channel: name, User: user, Text: m.Text, TS: m.TS})
	}
	return out
}

func tsNum(ts string) float64 {
	f, _ := strconv.ParseFloat(ts, 64)
	return f
}

// SendMessage posts text into a channel by name. It is an outbound write:
// while FOUNDEROS_WRITES is off it returns guard.ErrWritesDisabled and touches
// nothing, not even the channel lookup.
func (c *Connector) SendMessage(ctx context.Context, channelName, text string) (SendResult, error) {
	var res SendResult
	err := guard.Outbound("slack.chat.postMessage", func() error {
		tok := c.token()
		if tok == "" {
			res = SendResult{Detail: "SLACK_BOT_TOKEN not set — add it under Connections → API keys"}
			return nil
		}
		var page listPage
		q := url.Values{"types": {"public_channel,private_channel"}, "limit": {"200"}}
		if err := c.get(ctx, tok, "conversations.list", q, &page); err != nil {
			res = SendResult{Detail: err.Error()}
			return nil
		}
		want := strings.TrimPrefix(channelName, "#")
		id := ""
		for _, ch := range page.Channels {
			if ch.Name == want {
				id = ch.ID
				break
			}
		}
		if id == "" {
			res = SendResult{Detail: fmt.Sprintf("channel #%s not found or bot not invited", channelName)}
			return nil
		}
		if err := c.post(ctx, tok, "chat.postMessage", map[string]string{"channel": id, "text": text}); err != nil {
			res = SendResult{Detail: err.Error()}
			return nil
		}
		res = SendResult{OK: true, Detail: fmt.Sprintf("sent to #%s", channelName)}
		return nil
	})
	if err != nil {
		return SendResult{Detail: err.Error()}, err
	}
	return res, nil
}

// ---- Web API plumbing -----------------------------------------------------

// get calls a read method. Slack accepts GET with query arguments for every
// read method used here, so reads never need a readPOSTs allowance.
func (c *Connector) get(ctx context.Context, tok, method string, q url.Values, out any) error {
	u := c.baseURL + method
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return c.do(req, tok, out)
}

func (c *Connector) post(ctx context.Context, tok, method string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+method, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	return c.do(req, tok, nil)
}

// do sends the request and applies Slack's envelope: HTTP 200 with ok:false is
// an API error, worded like @slack/web-api's "An API error occurred: <code>".
func (c *Connector) do(req *http.Request, tok string, out any) error {
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (retry after %ss)", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var env struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unreadable Slack response: %w", err)
	}
	if !env.OK {
		return fmt.Errorf("An API error occurred: %s", env.Error)
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}
