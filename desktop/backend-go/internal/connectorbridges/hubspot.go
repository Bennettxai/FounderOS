// hubspot_connector.go — HubSpot fetcher for the OptimalEngine connector layer.
//
// Phase 1 surfaces the user's CRM contacts. Companies + deals follow the
// same shape (the adapter's Transform routes by `attributes.type`) and
// can be fanned out by the fetcher in a follow-up — keeping the wire
// narrow for now means the contract is easy to reason about.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/hubspot"
)

const hubspotContactsEndpoint = "https://api.hubapi.com/crm/v3/objects/contacts"

// HubspotFetcher implements adapters.HubspotFetcher using the
// CRM v3 REST API. Token comes from the existing hubspot.Provider.
type HubspotFetcher struct {
	prov   *hubspot.Provider
	client *http.Client
}

func NewHubspotFetcher(prov *hubspot.Provider) *HubspotFetcher {
	return &HubspotFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListRecords pulls the next page of contacts. The fetcher always tags
// each record with attributes.type=Contact so the adapter's Transform
// emits a "contact" genre. Future fan-out (deals, companies, tickets)
// would alternate the type per page.
func (f *HubspotFetcher) ListRecords(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("hubspot: token: %w", err)
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", max))
	params.Set("properties", "firstname,lastname,email,company,jobtitle")
	if cursor != "" {
		params.Set("after", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		hubspotContactsEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("hubspot: request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("hubspot: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Results []map[string]any `json:"results"`
		Paging  struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("hubspot: decode: %w", err)
	}

	// Tag each record so the adapter's Transform — which is shape-agnostic
	// across HubSpot record types — can route by genre.
	for i := range out.Results {
		if attrs, ok := out.Results[i]["attributes"].(map[string]any); ok {
			if _, exists := attrs["type"]; !exists {
				attrs["type"] = "Contact"
			}
		} else {
			out.Results[i]["attributes"] = map[string]any{"type": "Contact"}
		}
	}
	return out.Results, out.Paging.Next.After, nil
}
