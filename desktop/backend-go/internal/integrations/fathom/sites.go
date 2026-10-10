package fathom

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Site represents a Fathom site/property.
type Site struct {
	ID          string    `json:"id"`
	Name        string    `json:"object"`
	SharingURL  string    `json:"sharing_url,omitempty"`
	ShareConfig string    `json:"share_config,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SitesResponse is the API response for listing sites.
type SitesResponse struct {
	Data    []Site `json:"data"`
	HasMore bool   `json:"has_more"`
}

// fetchSites retrieves all sites from the Fathom API.
func (p *Provider) fetchSites(ctx context.Context, apiKey string) ([]Site, error) {
	body, err := p.makeRequest(ctx, apiKey, "GET", "/sites", nil)
	if err != nil {
		return nil, err
	}

	var resp SitesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// syncSites synchronizes sites from the Fathom API to the local database.
func (p *Provider) syncSites(ctx context.Context, userID, apiKey string) (*syncStats, error) {
	sites, err := p.fetchSites(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	stats := &syncStats{}

	for _, site := range sites {
		_, err := p.pool.Exec(ctx, `
			INSERT INTO fathom_sites (
				user_id, site_id, name, sharing_url, synced_at
			) VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_id, site_id) DO UPDATE SET
				name = EXCLUDED.name,
				sharing_url = EXCLUDED.sharing_url,
				synced_at = NOW(),
				updated_at = NOW()
		`, userID, site.ID, site.Name, site.SharingURL)

		if err != nil {
			return nil, fmt.Errorf("failed to save site %s: %w", site.ID, err)
		}
		stats.Created++
	}

	return stats, nil
}

// GetSites returns sites from the local database.
func (p *Provider) GetSites(ctx context.Context, userID string) ([]Site, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT site_id, name, sharing_url
		FROM fathom_sites
		WHERE user_id = $1
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site
		var sharingURL *string
		err := rows.Scan(&site.ID, &site.Name, &sharingURL)
		if err != nil {
			return nil, err
		}
		if sharingURL != nil {
			site.SharingURL = *sharingURL
		}
		sites = append(sites, site)
	}

	return sites, nil
}
