// Package vidalytics ports FounderOS v1's lib/connectors/vidalytics.ts: the VSL
// video host.
//
// Real Public API (Postman collection behind api-docs.vidalytics.com,
// verified 2026-09-23):
//
//	base  https://api.vidalytics.com/public/v1
//	auth  the key sent verbatim as X-API-Key (no Bearer prefix)
//	GET /video  → { content: { data: [{ id, title, status, date_created,
//	              last_published, folder_id, embedGuid }] } }
//
// The stats endpoints (/stats/...) count against a small monthly quota with
// an hourly throttle, so this package reads the library only and has no code
// path that can reach them.
package vidalytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "vidalytics", Name: "Vidalytics", Kind: connectors.KindCreative}

const (
	keyName = "VIDALYTICS_API_KEY"
	// API is the public v1 base URL.
	API     = "https://api.vidalytics.com/public/v1"
	timeout = 6 * time.Second
)

// ErrNotConfigured is returned by reads when no key resolves.
var ErrNotConfigured = errors.New("vidalytics: VIDALYTICS_API_KEY not configured")

// Video is one row of the library.
type Video struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Status        *string `json:"status"`
	CreatedAt     *string `json:"createdAt"`
	LastPublished *string `json:"lastPublished"`
	FolderID      *string `json:"folderId"`
}

type Connector struct {
	res connectors.Resolver
	// BaseURL overrides API (tests).
	BaseURL string
	// CredFiles are the fallback env files, in order. nil means FounderOS v1's
	// order for this key: clue-agent/.env.agents, then ~/.social-media/.env.
	CredFiles []string
	client    *http.Client
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, BaseURL: API, client: connectors.HTTPClient(timeout)}
}

func (c *Connector) key() string {
	files := c.CredFiles
	if files == nil {
		social, clue, _, _ := connectors.CredFiles()
		files = []string{clue, social}
	}
	return c.res.Resolve(keyName, files...)
}

func nonBlank(v any) *string {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// ParseVideos maps content.data to rows. ok is false for a shape it does not
// recognise, so a caller can say "unreadable" instead of "0 videos".
func ParseVideos(body []byte) ([]Video, bool) {
	var doc struct {
		Content *struct {
			Data json.RawMessage `json:"data"`
		} `json:"content"`
	}
	if json.Unmarshal(body, &doc) != nil || doc.Content == nil {
		return nil, false
	}
	var rows []map[string]any
	if json.Unmarshal(doc.Content.Data, &rows) != nil || rows == nil {
		return nil, false
	}
	out := []Video{}
	for _, v := range rows {
		id := nonBlank(v["id"])
		if id == nil {
			continue
		}
		title := "Untitled video"
		if t := nonBlank(v["title"]); t != nil {
			title = *t
		}
		out = append(out, Video{
			ID:            *id,
			Title:         title,
			Status:        nonBlank(v["status"]),
			CreatedAt:     nonBlank(v["date_created"]),
			LastPublished: nonBlank(v["last_published"]),
			FolderID:      nonBlank(v["folder_id"]),
		})
	}
	return out, true
}

func (c *Connector) getVideos(ctx context.Context, key string) ([]Video, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/video", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	videos, ok := ParseVideos(body)
	if !ok {
		return nil, errors.New("unrecognised /video payload")
	}
	return videos, nil
}

// ListVideos returns the video library. Unlike the TS (which reads [] on any
// failure), a failure is an error here so no caller can mistake an outage for
// an empty library.
func (c *Connector) ListVideos(ctx context.Context) ([]Video, error) {
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	return c.getVideos(ctx, key)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	key := c.key()
	if key == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = "Set VIDALYTICS_API_KEY to read the VSL library. Generate it in Vidalytics under Account Settings → Global Settings → API (Premium plan and above)."
		return s
	}
	videos, err := c.getVideos(ctx, key)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "VIDALYTICS_API_KEY is set but the call failed: " + err.Error()
		return s
	}
	plural := "s"
	if len(videos) == 1 {
		plural = ""
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("Vidalytics reachable · %d video%s in the library (stats are quota-metered and not polled)", len(videos), plural)
	s.Meta = map[string]any{"videos": len(videos)}
	return s
}
