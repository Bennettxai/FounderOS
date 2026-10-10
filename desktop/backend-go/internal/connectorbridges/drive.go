// drive.go — Google Drive fetcher for the OptimalEngine connector layer.
//
// Implements adapters.DriveFetcher on top of the existing google.Provider
// (the same OAuth wiring that powers Gmail). Pulls files via Drive v3 API.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/google"
)

// DriveFetcher pulls Drive files via the Drive v3 REST API. The token is
// the same OAuth token Gmail uses — the user just has to grant the
// drive.readonly scope at OAuth time.
type DriveFetcher struct {
	prov   *google.Provider
	client *http.Client
}

// NewDriveFetcher returns a fetcher backed by the given google.Provider.
func NewDriveFetcher(prov *google.Provider) *DriveFetcher {
	return &DriveFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListFiles paginates over the user's accessible files, ordered by
// modifiedTime desc. The cursor is Drive's nextPageToken (opaque).
//
// We deliberately do NOT export Google native types via files.export here —
// the connector pipeline only needs metadata at this stage; downstream
// architecture processors fetch + classify content through their own
// processors when needed. Keeps batches small and predictable.
func (f *DriveFetcher) ListFiles(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tokSrc, err := f.prov.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("drive: token source: %w", err)
	}
	tok, err := tokSrc.Token()
	if err != nil {
		return nil, "", fmt.Errorf("drive: token: %w", err)
	}

	params := url.Values{}
	params.Set("pageSize", fmt.Sprintf("%d", max))
	params.Set("orderBy", "modifiedTime desc")
	params.Set("fields", "nextPageToken, files(id, name, mimeType, modifiedTime, webViewLink)")
	if cursor != "" {
		params.Set("pageToken", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/drive/v3/files?"+params.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("drive: request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("drive: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Files         []map[string]any `json:"files"`
		NextPageToken string           `json:"nextPageToken"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("drive: decode: %w", err)
	}
	return out.Files, out.NextPageToken, nil
}
