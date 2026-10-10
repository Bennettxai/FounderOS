// Package loom ports FounderOS v1's lib/connectors/loom.ts, deliberately the
// honest, limited connector: Loom publishes no account API, so there is no
// key to set and no way to list a library. What exists is the public oEmbed
// endpoint, which resolves any Loom share link to title, author, thumbnail
// and duration with no auth. Status reports that lane (limited by design).
package loom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "loom", Name: "Loom", Kind: connectors.KindCreative}

const (
	DefaultOEmbedURL = "https://www.loom.com/v1/oembed"
	// ProbeURL is a public Loom link used only as a reachability probe.
	ProbeURL = "https://www.loom.com/share/e5b8c0b2a1a24a3b9d2f0c1e2a3b4c5d"
	Timeout  = 6 * time.Second
)

var ErrNotLoomURL = errors.New("loom: not an http(s) loom.com link")

type Connector struct {
	OEmbedURL string
	client    *http.Client
}

// New takes a Resolver for symmetry with the other connectors; Loom has no
// credentials to resolve.
func New(connectors.Resolver) *Connector {
	return &Connector{OEmbedURL: DefaultOEmbedURL, client: connectors.HTTPClient(Timeout)}
}

// IsLoomURL is true for a real http(s) loom.com link. Parsed, not regexed,
// so javascript: and lookalike hosts cannot reach a fetch.
func IsLoomURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "loom.com" || host == "www.loom.com"
}

func (c *Connector) get(ctx context.Context, link string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.OEmbedURL+"?url="+url.QueryEscape(link), nil)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	// Any answer at all (a 404 for a dead probe id included) proves the lane
	// is up; only a transport failure means Loom is unreachable.
	res, err := c.get(ctx, ProbeURL)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Loom oEmbed unreachable: " + reason(err)
		return st
	}
	res.Body.Close()
	st.State = connectors.StateConnected
	st.Detail = "oEmbed reachable: any Loom link resolves to title, thumbnail and duration. Loom publishes no account API, so there is no key to set and no way to list your library."
	st.Meta = map[string]any{"scope": "link-level", "auth": "none required"}
	return st
}

// VideoMeta is one Loom link's public metadata. Optional fields are nil
// when oEmbed omits them.
type VideoMeta struct {
	Title           string   `json:"title"`
	Author          *string  `json:"author"`
	ThumbnailURL    *string  `json:"thumbnailUrl"`
	DurationSeconds *float64 `json:"durationSeconds"`
}

// VideoMeta resolves a Loom link. A non-Loom link, a private or removed
// video, or any failure is an error; callers render "unavailable".
func (c *Connector) VideoMeta(ctx context.Context, link string) (*VideoMeta, error) {
	if !IsLoomURL(link) {
		return nil, ErrNotLoomURL
	}
	res, err := c.get(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("loom oembed: %s", reason(err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("loom oembed: HTTP %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, errors.New("loom oembed: unreadable response")
	}
	title, ok := body["title"].(string)
	if !ok {
		return nil, errors.New("loom oembed: no title (private or removed video)")
	}
	m := &VideoMeta{Title: title}
	if s, ok := body["author_name"].(string); ok {
		m.Author = &s
	}
	if s, ok := body["thumbnail_url"].(string); ok {
		m.ThumbnailURL = &s
	}
	if f, ok := body["duration"].(float64); ok {
		m.DurationSeconds = &f
	}
	return m, nil
}

func reason(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() {
			return "request timed out"
		}
		return ue.Err.Error()
	}
	return err.Error()
}
