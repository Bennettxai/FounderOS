package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// hubSpotDeal represents a HubSpot deal from the API.
type hubSpotDeal struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// hubSpotDealsResponse represents the paginated API response.
type hubSpotDealsResponse struct {
	Results []hubSpotDeal `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

// syncDeals syncs deals from HubSpot to the database.
func (p *Provider) syncDeals(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	queries := sqlc.New(p.pool)
	stats := &syncStats{}

	// Pagination loop
	var after string
	for {
		// Build request URL
		url := fmt.Sprintf("%s/crm/v3/objects/deals?limit=100", BaseAPIURL)
		if after != "" {
			url += fmt.Sprintf("&after=%s", after)
		}

		// Add properties we care about
		url += "&properties=dealname,amount,pipeline,dealstage,closedate,hubspot_owner_id"
		url += "&associations=companies,contacts"

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return stats, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return stats, fmt.Errorf("failed to fetch deals: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return stats, fmt.Errorf("HubSpot API error: %d", resp.StatusCode)
		}

		var dealsResp hubSpotDealsResponse
		if err := json.NewDecoder(resp.Body).Decode(&dealsResp); err != nil {
			return stats, fmt.Errorf("failed to decode response: %w", err)
		}

		// Process each deal
		for _, deal := range dealsResp.Results {
			// Marshal properties to JSON for storage
			propertiesJSON, err := json.Marshal(deal.Properties)
			if err != nil {
				continue
			}

			// Helper to get string properties
			getProp := func(key string) *string {
				if val, ok := deal.Properties[key].(string); ok && val != "" {
					return &val
				}
				return nil
			}

			// Helper to get numeric properties
			getNumeric := func(key string) pgtype.Numeric {
				num := pgtype.Numeric{}
				if val, ok := deal.Properties[key].(float64); ok {
					_ = num.Scan(val)
				} else if val, ok := deal.Properties[key].(string); ok && val != "" {
					_ = num.Scan(val)
				}
				return num
			}

			// Helper to parse date
			getDate := func(key string) pgtype.Date {
				date := pgtype.Date{}
				if val, ok := deal.Properties[key].(string); ok && val != "" {
					if t, err := time.Parse("2006-01-02", val); err == nil {
						_ = date.Scan(t)
					}
				}
				return date
			}

			// Convert timestamps
			createdAt := pgtype.Timestamptz{}
			_ = createdAt.Scan(deal.CreatedAt)

			updatedAt := pgtype.Timestamptz{}
			_ = updatedAt.Scan(deal.UpdatedAt)

			// For now, we'll leave associations as empty arrays
			// A more complete implementation would parse the associations from the response
			emptyArray := []string{}
			emptyArrayJSON, _ := json.Marshal(emptyArray)

			// Upsert deal
			result, err := queries.UpsertHubSpotDeal(ctx, sqlc.UpsertHubSpotDealParams{
				UserID:                userID,
				HubspotID:             deal.ID,
				DealName:              getProp("dealname"),
				Amount:                getNumeric("amount"),
				Pipeline:              getProp("pipeline"),
				DealStage:             getProp("dealstage"),
				CloseDate:             getDate("closedate"),
				OwnerID:               getProp("hubspot_owner_id"),
				AssociatedCompanyIds:  emptyArrayJSON,
				AssociatedContactIds:  emptyArrayJSON,
				Properties:            propertiesJSON,
				CreatedAtHubspot:      createdAt,
				UpdatedAtHubspot:      updatedAt,
			})

			if err != nil {
				continue // Skip errors for individual deals
			}

			// Check if it was an insert or update
			if result.CreatedAt.Time.Equal(result.UpdatedAt.Time) {
				stats.Created++
			} else {
				stats.Updated++
			}
		}

		// Check if there are more pages
		if dealsResp.Paging.Next.After == "" {
			break
		}
		after = dealsResp.Paging.Next.After
	}

	return stats, nil
}

// GetDeals retrieves deals for a user from the database.
func (p *Provider) GetDeals(ctx context.Context, userID string, limit, offset int32) ([]sqlc.HubspotDeal, error) {
	queries := sqlc.New(p.pool)
	return queries.GetHubSpotDealsByUser(ctx, sqlc.GetHubSpotDealsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}
