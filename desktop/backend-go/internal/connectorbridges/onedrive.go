// onedrive.go — OneDrive fetcher for the OptimalEngine connector layer.
//
// Implements adapters.OnedriveFetcher via Microsoft Graph /me/drive/root/
// children. Reuses the existing microsoft.Provider for OAuth.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
)

const onedriveItemsURL = "https://graph.microsoft.com/v1.0/me/drive/root/children"

// OnedriveFetcher pulls OneDrive items. Token comes from microsoft.Provider.
// Cursor format: a Graph @odata.nextLink (full URL, opaque to the engine).
type OnedriveFetcher struct {
	prov   *microsoft.Provider
	client *http.Client
}

// NewOnedriveFetcher returns a fetcher wired to the Microsoft provider.
func NewOnedriveFetcher(prov *microsoft.Provider) *OnedriveFetcher {
	return &OnedriveFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListItems pulls one batch of children at the drive root, ordered by
// lastModifiedDateTime desc. We don't recurse into subfolders here —
// keeps the batch bounded and predictable. A future fetcher can walk
// folder children explicitly when full-tree indexing is needed.
func (f *OnedriveFetcher) ListItems(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("onedrive: token: %w", err)
	}

	target := cursor
	if target == "" {
		target = fmt.Sprintf("%s?$top=%d&$orderby=lastModifiedDateTime%%20desc", onedriveItemsURL, max)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("onedrive: request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("onedrive: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Value    []map[string]any `json:"value"`
		NextLink string           `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("onedrive: decode: %w", err)
	}
	return out.Value, out.NextLink, nil
}
