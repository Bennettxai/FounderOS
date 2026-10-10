// Package manychat ports FounderOS v1's lib/connectors/manychat.ts (Instagram DM
// automation) and the inbound External Request webhook of
// app/api/webhooks/manychat/route.ts.
//
// ManyChat blocks EVERY request for 24 hours once its daily cap is hit, so the
// status check makes at most one GET /fb/page/getInfo per 3-hour window per
// key, shared by every Connector in the process, and failures are cached too.
// An HTTP answer is never retried; only a transport failure gets one retry.
//
// ManyChat's API cannot list DMs. The live DM inbox is fed by the webhook
// (webhook.go), not by polling this connector.
package manychat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "manychat", Name: "ManyChat (IG DMs)", Kind: connectors.KindSocial}

const (
	DefaultBaseURL = "https://api.manychat.com"
	keyName        = "MANYCHAT_API_KEY"
	// StatusTTL is the operator's call (2026-08-14): everything else 15 minutes,
	// ManyChat 3 hours.
	StatusTTL = 3 * time.Hour
	// Timeout matches paperclip, docusign and payments.
	Timeout = 8 * time.Second
)

type Connector struct {
	res connectors.Resolver
	// ClaudeJSON is the ~/.claude.json holding the manychat MCP registration,
	// the last fallback for the key.
	ClaudeJSON string
	BaseURL    string
	Now        func() time.Time
	client     *http.Client
}

func New(res connectors.Resolver) *Connector {
	_, _, _, claudeJSON := connectors.CredFiles()
	return &Connector{
		res:        res,
		ClaudeJSON: claudeJSON,
		BaseURL:    DefaultBaseURL,
		Now:        time.Now,
		client:     connectors.HTTPClient(Timeout),
	}
}

// key mirrors lib/creds.ts resolveManychatKey: env.local, then the process
// env, then the manychat MCP registration in ~/.claude.json.
func (c *Connector) key() string {
	if k := c.res.Resolve(keyName); k != "" {
		return k
	}
	return connectors.McpEnvKey(c.ClaudeJSON, "manychat", keyName)
}

// PageInfo is the part of GET /fb/page/getInfo the board surfaces.
type PageInfo struct {
	Name     string
	Username string // "" when ManyChat has none
	IsPro    bool
}

// ParsePageInfo maps a getInfo payload (wrapped in data or bare). Nil when no
// usable name is present.
func ParsePageInfo(raw []byte) *PageInfo {
	var outer map[string]json.RawMessage
	if json.Unmarshal(raw, &outer) != nil || outer == nil {
		return nil
	}
	body := raw
	if d, ok := outer["data"]; ok && string(d) != "null" {
		body = d
	}
	var p struct {
		Name     any `json:"name"`
		Username any `json:"username"`
		IsPro    any `json:"is_pro"`
	}
	if json.Unmarshal(body, &p) != nil {
		return nil
	}
	name, _ := p.Name.(string)
	if name == "" {
		return nil
	}
	username, _ := p.Username.(string)
	isPro, _ := p.IsPro.(bool)
	return &PageInfo{Name: name, Username: username, IsPro: isPro}
}

type cached struct {
	at     time.Time
	state  connectors.State
	detail string
	meta   map[string]any
}

// The cap is per account, so the cache is per process, not per Connector.
var (
	cacheMu sync.Mutex
	cache   = map[string]cached{}
)

func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	key := c.key()
	if key == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = "Set MANYCHAT_API_KEY in ~/.founderos/.env or under API keys (ManyChat → Settings → API)."
		return st
	}

	// Held across the check so concurrent callers share one request.
	cacheMu.Lock()
	defer cacheMu.Unlock()
	ck := c.BaseURL + "\x00" + key
	now := c.Now()
	if hit, ok := cache[ck]; ok && now.Sub(hit.at) < StatusTTL {
		st.State, st.Detail, st.Meta = hit.state, hit.detail, hit.meta
		return st
	}

	page, err := c.pageInfo(ctx, key)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Key set but API check failed: " + reason(err)
	} else {
		handle := page.Name
		if page.Username != "" {
			handle = "@" + page.Username
		}
		st.State = connectors.StateConnected
		st.Detail = handle + " · Instagram"
		if page.IsPro {
			st.Detail += " · Pro"
		}
		st.Meta = map[string]any{"handle": handle}
	}
	cache[ck] = cached{at: now, state: st.State, detail: st.Detail, meta: st.Meta}
	return st
}

func (c *Connector) pageInfo(ctx context.Context, key string) (*PageInfo, error) {
	var (
		res  *http.Response
		last error
	)
	// One retry, on a transport failure only. A 429 is still a 429 the
	// second time, and retrying it is how an account walks into the block.
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/fb/page/getInfo", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Accept", "application/json")
		res, last = c.client.Do(req)
		if last == nil || ctx.Err() != nil {
			break
		}
	}
	if last != nil {
		return nil, last
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	page := ParsePageInfo(buf.Bytes())
	if page == nil {
		return nil, errors.New("no account info in response")
	}
	return page, nil
}

// SendResult is what the /social reply box shows inline.
type SendResult struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// SendText DMs a subscriber through sendContent. It is guarded by
// FOUNDEROS_WRITES, never retried (a resent sendContent is a second DM), and
// never claims success without a real 2xx. err is non-nil whenever nothing
// was sent; errors.Is(err, guard.ErrWritesDisabled) tells a guard refusal.
func (c *Connector) SendText(ctx context.Context, subscriberID, text string) (SendResult, error) {
	key := c.key()
	if key == "" {
		err := errors.New("MANYCHAT_API_KEY not set — connect ManyChat to send DMs.")
		return SendResult{Detail: err.Error()}, err
	}
	err := guard.Outbound("manychat.send_text", func() error {
		payload, _ := json.Marshal(map[string]any{
			"subscriber_id": subscriberID,
			"data": map[string]any{
				"version": "v2",
				"content": map[string]any{
					"type":     "instagram",
					"messages": []map[string]string{{"type": "text", "text": text}},
				},
			},
			"message_tag": "ACCOUNT_UPDATE",
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/fb/sending/sendContent", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		res, err := c.client.Do(req)
		if err != nil {
			return err
		}
		res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return fmt.Errorf("HTTP %d", res.StatusCode)
		}
		return nil
	})
	if err != nil {
		return SendResult{Detail: "ManyChat send failed: " + reason(err)}, err
	}
	return SendResult{OK: true, Detail: "sent"}, nil
}

// reason turns a client error into board wording without the request URL.
func reason(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() {
			return "request timed out"
		}
		return strings.TrimSpace(ue.Err.Error())
	}
	return err.Error()
}
