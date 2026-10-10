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

// hubSpotCompany represents a HubSpot company from the API.
type hubSpotCompany struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// hubSpotCompaniesResponse represents the paginated API response.
type hubSpotCompaniesResponse struct {
	Results []hubSpotCompany `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

// syncCompanies syncs companies from HubSpot to the database.
func (p *Provider) syncCompanies(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	queries := sqlc.New(p.pool)
	stats := &syncStats{}

	// Pagination loop
	var after string
	for {
		// Build request URL
		url := fmt.Sprintf("%s/crm/v3/objects/companies?limit=100", BaseAPIURL)
		if after != "" {
			url += fmt.Sprintf("&after=%s", after)
		}

		// Add properties we care about
		url += "&properties=name,domain,industry,numberofemployees,annualrevenue,city,state,country,hubspot_owner_id"

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return stats, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return stats, fmt.Errorf("failed to fetch companies: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return stats, fmt.Errorf("HubSpot API error: %d", resp.StatusCode)
		}

		var companiesResp hubSpotCompaniesResponse
		if err := json.NewDecoder(resp.Body).Decode(&companiesResp); err != nil {
			return stats, fmt.Errorf("failed to decode response: %w", err)
		}

		// Process each company
		for _, company := range companiesResp.Results {
			// Marshal properties to JSON for storage
			propertiesJSON, err := json.Marshal(company.Properties)
			if err != nil {
				continue
			}

			// Helper to get string properties
			getProp := func(key string) *string {
				if val, ok := company.Properties[key].(string); ok && val != "" {
					return &val
				}
				return nil
			}

			// Helper to get int32 properties
			getInt32 := func(key string) *int32 {
				if val, ok := company.Properties[key].(float64); ok {
					i := int32(val)
					return &i
				}
				if val, ok := company.Properties[key].(string); ok && val != "" {
					var i int32
					if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
						return &i
					}
				}
				return nil
			}

			// Helper to get numeric properties
			getNumeric := func(key string) pgtype.Numeric {
				num := pgtype.Numeric{}
				if val, ok := company.Properties[key].(float64); ok {
					_ = num.Scan(val)
				} else if val, ok := company.Properties[key].(string); ok && val != "" {
					_ = num.Scan(val)
				}
				return num
			}

			// Convert timestamps
			createdAt := pgtype.Timestamptz{}
			_ = createdAt.Scan(company.CreatedAt)

			updatedAt := pgtype.Timestamptz{}
			_ = updatedAt.Scan(company.UpdatedAt)

			// Upsert company
			result, err := queries.UpsertHubSpotCompany(ctx, sqlc.UpsertHubSpotCompanyParams{
				UserID:            userID,
				HubspotID:         company.ID,
				Name:              getProp("name"),
				Domain:            getProp("domain"),
				Industry:          getProp("industry"),
				NumberOfEmployees: getInt32("numberofemployees"),
				AnnualRevenue:     getNumeric("annualrevenue"),
				City:              getProp("city"),
				State:             getProp("state"),
				Country:           getProp("country"),
				OwnerID:           getProp("hubspot_owner_id"),
				Properties:        propertiesJSON,
				CreatedAtHubspot:  createdAt,
				UpdatedAtHubspot:  updatedAt,
			})

			if err != nil {
				continue // Skip errors for individual companies
			}

			// Check if it was an insert or update
			if result.CreatedAt.Time.Equal(result.UpdatedAt.Time) {
				stats.Created++
			} else {
				stats.Updated++
			}
		}

		// Check if there are more pages
		if companiesResp.Paging.Next.After == "" {
			break
		}
		after = companiesResp.Paging.Next.After
	}

	return stats, nil
}

// GetCompanies retrieves companies for a user from the database.
func (p *Provider) GetCompanies(ctx context.Context, userID string, limit, offset int32) ([]sqlc.HubspotCompany, error) {
	queries := sqlc.New(p.pool)
	return queries.GetHubSpotCompaniesByUser(ctx, sqlc.GetHubSpotCompaniesByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}
