package fathom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Aggregation represents analytics aggregation data.
type Aggregation struct {
	Visits      int     `json:"visits"`
	Uniques     int     `json:"uniques"`
	Pageviews   int     `json:"pageviews"`
	AvgDuration float64 `json:"avg_duration"`
	BounceRate  float64 `json:"bounce_rate"`
	Date        string  `json:"date,omitempty"`
	Pathname    string  `json:"pathname,omitempty"`
	Hostname    string  `json:"hostname,omitempty"`
	Referrer    string  `json:"referrer,omitempty"`
	Country     string  `json:"country_code,omitempty"`
	Device      string  `json:"device_type,omitempty"`
	Browser     string  `json:"browser,omitempty"`
}

// AggregationsResponse is the API response for aggregations.
type AggregationsResponse []Aggregation

// AggregationsInput contains parameters for fetching aggregations.
type AggregationsInput struct {
	Entity        string // "pageview" or "event"
	EntityID      string // site_id or event_id
	Aggregates    string // comma-separated: visits,uniques,pageviews,avg_duration,bounce_rate
	DateFrom      string // YYYY-MM-DD
	DateTo        string // YYYY-MM-DD
	DateGrouping  string // day, month, year
	FieldGrouping string // pathname, hostname, referrer, etc.
	SortBy        string // field to sort by
	Limit         int    // max results
}

// syncAggregations synchronizes aggregation data from the Fathom API to the local database.
func (p *Provider) syncAggregations(ctx context.Context, userID, apiKey string) (*syncStats, error) {
	// Get all sites first
	sites, err := p.fetchSites(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	stats := &syncStats{}

	// Sync aggregations for each site (last 30 days)
	for _, site := range sites {
		params := url.Values{}
		params.Set("entity", "pageview")
		params.Set("entity_id", site.ID)
		params.Set("aggregates", "visits,uniques,pageviews,avg_duration,bounce_rate")
		params.Set("date_grouping", "day")
		params.Set("date_from", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
		params.Set("date_to", time.Now().Format("2006-01-02"))

		body, err := p.makeRequest(ctx, apiKey, "GET", "/aggregations", params)
		if err != nil {
			continue // Skip this site on error
		}

		var aggregations AggregationsResponse
		if err := json.Unmarshal(body, &aggregations); err != nil {
			continue
		}

		for _, agg := range aggregations {
			_, err := p.pool.Exec(ctx, `
				INSERT INTO fathom_aggregations (
					user_id, site_id, date, visits, uniques, pageviews,
					avg_duration, bounce_rate, synced_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
				ON CONFLICT (user_id, site_id, date) DO UPDATE SET
					visits = EXCLUDED.visits,
					uniques = EXCLUDED.uniques,
					pageviews = EXCLUDED.pageviews,
					avg_duration = EXCLUDED.avg_duration,
					bounce_rate = EXCLUDED.bounce_rate,
					synced_at = NOW()
			`, userID, site.ID, agg.Date, agg.Visits, agg.Uniques,
				agg.Pageviews, agg.AvgDuration, agg.BounceRate)

			if err == nil {
				stats.Created++
			}
		}
	}

	return stats, nil
}

// GetAggregations returns aggregated analytics data for a site.
func (p *Provider) GetAggregations(ctx context.Context, userID string, input AggregationsInput) ([]Aggregation, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("entity", input.Entity)
	params.Set("entity_id", input.EntityID)
	params.Set("aggregates", input.Aggregates)

	if input.DateFrom != "" {
		params.Set("date_from", input.DateFrom)
	}
	if input.DateTo != "" {
		params.Set("date_to", input.DateTo)
	}
	if input.DateGrouping != "" {
		params.Set("date_grouping", input.DateGrouping)
	}
	if input.FieldGrouping != "" {
		params.Set("field_grouping", input.FieldGrouping)
	}
	if input.SortBy != "" {
		params.Set("sort_by", input.SortBy)
	}
	if input.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", input.Limit))
	}

	body, err := p.makeRequest(ctx, token.AccessToken, "GET", "/aggregations", params)
	if err != nil {
		return nil, err
	}

	var aggregations AggregationsResponse
	if err := json.Unmarshal(body, &aggregations); err != nil {
		return nil, err
	}

	return aggregations, nil
}
