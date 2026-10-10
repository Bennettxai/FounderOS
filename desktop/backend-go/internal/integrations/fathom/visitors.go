package fathom

import (
	"context"
	"encoding/json"
	"net/url"
)

// CurrentVisitors represents real-time visitor count.
type CurrentVisitors struct {
	Total   int `json:"total"`
	Content []struct {
		Pathname string `json:"pathname"`
		Total    int    `json:"total"`
	} `json:"content,omitempty"`
}

// GetCurrentVisitors returns real-time visitor count for a site.
func (p *Provider) GetCurrentVisitors(ctx context.Context, userID, siteID string, detailed bool) (*CurrentVisitors, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("site_id", siteID)
	if detailed {
		params.Set("detailed", "true")
	}

	body, err := p.makeRequest(ctx, token.AccessToken, "GET", "/current_visitors", params)
	if err != nil {
		return nil, err
	}

	var visitors CurrentVisitors
	if err := json.Unmarshal(body, &visitors); err != nil {
		return nil, err
	}

	return &visitors, nil
}
