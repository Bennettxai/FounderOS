package fathom

import (
	"context"
	"encoding/json"
	"net/url"
)

// Event represents a Fathom custom event.
type Event struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EventsResponse is the API response for events.
type EventsResponse struct {
	Data    []Event `json:"data"`
	HasMore bool    `json:"has_more"`
}

// GetEvents returns custom events for a site.
func (p *Provider) GetEvents(ctx context.Context, userID, siteID string) ([]Event, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("site_id", siteID)

	body, err := p.makeRequest(ctx, token.AccessToken, "GET", "/events", params)
	if err != nil {
		return nil, err
	}

	var resp EventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}
