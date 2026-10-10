// Package beehiiv ports FounderOS v1's lib/connectors/beehiiv.ts: the newsletter
// audience (subscriber count with provenance) and past newsletters with their
// send analytics. Read-only; Beehiiv has no write path in FounderOS v1.
package beehiiv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "beehiiv", Name: "Beehiiv (Email List)", Kind: connectors.KindSocial}

const (
	defaultBaseURL = "https://api.beehiiv.com/v2"
	TTL            = 60 * time.Second
	statusTimeout  = 6 * time.Second
	postsTimeout   = 8 * time.Second
)

// Metric names which stat field a reading came from. The two are defined
// differently, so a series that silently mixes them is not one series.
type Metric string

const (
	MetricActive Metric = "active_subscriptions"
	MetricTotal  Metric = "total_subscriptions"
)

// Reading is one subscriber observation. Fresh is false when the count was
// served from cache because the API call failed: worth showing, never worth
// recording as a new day's observation (FOS-659).
type Reading struct {
	Subscribers   int    `json:"subscribers"`
	PublicationID string `json:"publicationId"`
	Metric        Metric `json:"metric"`
	Fresh         bool   `json:"fresh"`
}

// Newsletter is one past send with its analytics; rates are percentages.
type Newsletter struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	PublishedAt     string  `json:"publishedAt"`
	WebURL          string  `json:"webUrl,omitempty"` // "" where the TS has null
	Recipients      float64 `json:"recipients"`
	Delivered       float64 `json:"delivered"`
	DeliveryRate    float64 `json:"deliveryRate"`
	Opens           float64 `json:"opens"`
	OpenRate        float64 `json:"openRate"`
	Clicks          float64 `json:"clicks"`
	ClickRate       float64 `json:"clickRate"`
	Unsubscribes    float64 `json:"unsubscribes"`
	UnsubscribeRate float64 `json:"unsubscribeRate"`
	SpamReports     float64 `json:"spamReports"`
	WebViews        float64 `json:"webViews"`
}

type Connector struct {
	res     connectors.Resolver
	baseURL string
	status  *http.Client
	posts   *http.Client
	now     func() time.Time

	mu         sync.Mutex
	reading    *cached[*Reading]
	postsCache *cached[[]Newsletter]
}

type cached[T any] struct {
	at   time.Time
	data T
}

func New(res connectors.Resolver) *Connector {
	return &Connector{
		res:     res,
		baseURL: defaultBaseURL,
		status:  connectors.HTTPClient(statusTimeout),
		posts:   connectors.HTTPClient(postsTimeout),
		now:     time.Now,
	}
}

func (c *Connector) creds() (key, pub string) {
	return c.res.Resolve("BEEHIIV_API_KEY"), c.res.Resolve("BEEHIIV_PUBLICATION_ID")
}

// ParseStats maps a publications/{id}?expand[]=stats payload to a subscriber
// count, preferring active subscriptions and naming the field it used.
func ParseStats(body []byte) (int, Metric, bool) {
	var doc struct {
		Data  *struct{ Stats map[string]any } `json:"data"`
		Stats map[string]any                  `json:"stats"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return 0, "", false
	}
	stats := doc.Stats
	if doc.Data != nil && doc.Data.Stats != nil {
		stats = doc.Data.Stats
	}
	for _, m := range []Metric{MetricActive, MetricTotal} {
		if n, ok := stats[string(m)].(float64); ok && !math.IsInf(n, 0) && !math.IsNaN(n) && n >= 0 {
			return int(n), m, true
		}
	}
	return 0, "", false
}

func (c *Connector) fetch(ctx context.Context, client *http.Client, key, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

func (c *Connector) statsURL(pub string) string {
	return c.baseURL + "/publications/" + pub + "?expand[]=stats"
}

func (c *Connector) st(state connectors.State, detail string, meta map[string]any) connectors.Status {
	return connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind, State: state, Detail: detail, Meta: meta}
}

// Status checks the publication's stats live. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	key, pub := c.creds()
	if key == "" || pub == "" {
		return c.st(connectors.StateNotConfigured, connectors.SetKeys("BEEHIIV_API_KEY", "BEEHIIV_PUBLICATION_ID"), nil)
	}
	body, err := c.fetch(ctx, c.status, key, c.statsURL(pub))
	if err != nil {
		return c.st(connectors.StateError, "Key set but API check failed: "+err.Error(), nil)
	}
	n, _, ok := ParseStats(body)
	if !ok {
		return c.st(connectors.StateError, "Key set but API check failed: no subscriber stats in response", nil)
	}
	return c.st(connectors.StateConnected, thousands(n)+" subscribers", map[string]any{"subscribers": n})
}

// Reading is the live subscriber reading, cached for 60s; nil when
// unconfigured or when no count has ever been fetched. On a failed call it
// returns the last good count marked Fresh=false; callers that persist
// history must drop those.
func (c *Connector) Reading(ctx context.Context) *Reading {
	c.mu.Lock()
	if rc := c.reading; rc != nil && c.now().Sub(rc.at) < TTL {
		c.mu.Unlock()
		return copyReading(rc.data)
	}
	c.mu.Unlock()
	key, pub := c.creds()
	if key == "" || pub == "" {
		return nil
	}
	at := c.now()
	body, err := c.fetch(ctx, c.status, key, c.statsURL(pub))
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if c.reading != nil && c.reading.data != nil {
			stale := *c.reading.data
			stale.Fresh = false
			return &stale
		}
		return nil
	}
	var r *Reading
	if n, m, ok := ParseStats(body); ok {
		r = &Reading{Subscribers: n, PublicationID: pub, Metric: m, Fresh: true}
	}
	c.reading = &cached[*Reading]{at: at, data: r}
	return copyReading(r)
}

func copyReading(r *Reading) *Reading {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}

// Subscribers is the count only, for display paths that do not persist
// history. ok is false when the count is unknown.
func (c *Connector) Subscribers(ctx context.Context) (int, bool) {
	if r := c.Reading(ctx); r != nil {
		return r.Subscribers, true
	}
	return 0, false
}

// Posts returns past newsletters (latest 50 by publish date), cached for 60s;
// nil when unconfigured, on an error with nothing cached, or when there are
// none, so callers fall back to the seed.
func (c *Connector) Posts(ctx context.Context) []Newsletter {
	c.mu.Lock()
	if pc := c.postsCache; pc != nil && c.now().Sub(pc.at) < TTL {
		c.mu.Unlock()
		return append([]Newsletter(nil), pc.data...)
	}
	c.mu.Unlock()
	key, pub := c.creds()
	if key == "" || pub == "" {
		return nil
	}
	at := c.now()
	url := c.baseURL + "/publications/" + pub + "/posts?expand[]=stats&limit=50&order_by=publish_date&direction=desc"
	body, err := c.fetch(ctx, c.posts, key, url)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if c.postsCache != nil {
			return append([]Newsletter(nil), c.postsCache.data...)
		}
		return nil
	}
	list := ParsePosts(body)
	if len(list) == 0 {
		list = nil
	}
	c.postsCache = &cached[[]Newsletter]{at: at, data: list}
	return append([]Newsletter(nil), list...)
}

// ParsePosts maps a posts+stats payload to Newsletters, keeping only sent
// posts. It handles the REST field names with aliases for the MCP shape and
// never fails: bad input yields an empty list.
func ParsePosts(body []byte) []Newsletter {
	var doc struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return []Newsletter{}
	}
	out := []Newsletter{}
	for _, raw := range doc.Data {
		var p map[string]any
		if json.Unmarshal(raw, &p) != nil || p == nil {
			continue
		}
		id := jsString(p["id"])
		if id == "" || (jsString(p["status"]) != "confirmed" && jsString(p["status"]) != "published") {
			continue
		}
		stats, _ := p["stats"].(map[string]any)
		e, _ := stats["email"].(map[string]any)
		web, _ := stats["web"].(map[string]any)
		recipients := first(e, "recipients", "total_sent")
		delivered := first(e, "delivered", "total_delivered")
		unsubscribes := first(e, "unsubscribes", "total_unsubscribes")
		n := Newsletter{
			ID:           id,
			Title:        "Untitled",
			PublishedAt:  publishedAt(p),
			Recipients:   recipients,
			Delivered:    delivered,
			Opens:        first(e, "unique_opens", "total_unique_opened"),
			OpenRate:     round2(first(e, "open_rate")),
			Clicks:       first(e, "unique_clicks", "total_unique_email_clicked_verified", "total_unique_email_clicked_raw"),
			ClickRate:    round2(first(e, "click_rate")),
			Unsubscribes: unsubscribes,
			SpamReports:  first(e, "spam_reports", "total_spam_reported"),
			WebViews:     first(web, "views", "total_web_viewed"),
		}
		if t, ok := p["title"]; ok && t != nil {
			n.Title = jsString(t)
		}
		if u, ok := p["web_url"].(string); ok {
			n.WebURL = u
		}
		if recipients > 0 {
			n.DeliveryRate = round2(delivered / recipients * 100)
		}
		if delivered > 0 {
			n.UnsubscribeRate = round2(unsubscribes / delivered * 100)
		}
		out = append(out, n)
	}
	return out
}

// first mirrors the TS `num(a ?? b ?? c)`: the first present (non-null) key
// decides, and a non-number there reads 0.
func first(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if f, ok := v.(float64); ok && !math.IsInf(f, 0) && !math.IsNaN(f) {
			return f
		}
		return 0
	}
	return 0
}

func round2(n float64) float64 { return math.Floor(n*100+0.5) / 100 }

const isoMillis = "2006-01-02T15:04:05.000Z"

// publishedAt: publish_date (unix seconds) or displayed_date, then created.
// Beehiiv's created is also unix seconds; the TS passes it to new Date() as
// milliseconds, which lands in 1970, so here it is read as seconds.
func publishedAt(p map[string]any) string {
	ts := p["publish_date"]
	if ts == nil {
		ts = p["displayed_date"]
	}
	if ts == nil {
		ts = p["created"]
	}
	switch v := ts.(type) {
	case float64:
		return time.Unix(0, int64(v*1e9)).UTC().Format(isoMillis)
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC().Format(isoMillis)
		}
	}
	return time.Now().UTC().Format(isoMillis)
}

func jsString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(s)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

// thousands formats like toLocaleString('en-US').
func thousands(n int) string {
	s := strconv.Itoa(n)
	neg := n < 0
	if neg {
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "-" + s
	}
	return s
}
