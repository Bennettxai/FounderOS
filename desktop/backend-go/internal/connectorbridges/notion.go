// notion_connector.go — Notion fetcher for the OptimalEngine connector layer.
package connectorbridges

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/notion"
)

// NotionFetcher pulls pages via Notion's /v1/search endpoint
// filtered to page objects, ordered by most-recently-edited.
type NotionFetcher struct {
	prov   *notion.Provider
	client *http.Client
}

func NewNotionFetcher(prov *notion.Provider) *NotionFetcher {
	return &NotionFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListPages calls /v1/search with cursor for incremental sync. The
// returned objects are page records; downstream Transform maps them to
// Signals (title via properties.Name.title, blocks not fetched here —
// that's a per-page round-trip we defer to a follow-up).
func (f *NotionFetcher) ListPages(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("notion: token: %w", err)
	}

	body := map[string]any{
		"filter":    map[string]any{"value": "page", "property": "object"},
		"sort":      map[string]any{"direction": "descending", "timestamp": "last_edited_time"},
		"page_size": max,
	}
	if cursor != "" {
		body["start_cursor"] = cursor
	}
	encoded, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.notion.com/v1/search", bytes.NewReader(encoded))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Notion-Version", "2022-06-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("notion: search: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("notion: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Results    []map[string]any `json:"results"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("notion: decode: %w", err)
	}
	next := ""
	if out.HasMore {
		next = out.NextCursor
	}
	return out.Results, next, nil
}
