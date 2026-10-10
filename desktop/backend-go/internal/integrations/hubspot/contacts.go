package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// hubSpotContact represents a HubSpot contact from the API.
type hubSpotContact struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// hubSpotContactsResponse represents the paginated API response.
type hubSpotContactsResponse struct {
	Results []hubSpotContact `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

// syncContacts syncs contacts from HubSpot to the database.
func (p *Provider) syncContacts(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	queries := sqlc.New(p.pool)
	stats := &syncStats{}

	// Pagination loop
	var after string
	for {
		// Build request URL
		url := fmt.Sprintf("%s/crm/v3/objects/contacts?limit=100", BaseAPIURL)
		if after != "" {
			url += fmt.Sprintf("&after=%s", after)
		}

		// Add properties we care about
		url += "&properties=email,firstname,lastname,phone,company,jobtitle,lifecyclestage,hs_lead_status,hubspot_owner_id"

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return stats, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return stats, fmt.Errorf("failed to fetch contacts: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return stats, fmt.Errorf("HubSpot API error: %d", resp.StatusCode)
		}

		var contactsResp hubSpotContactsResponse
		if err := json.NewDecoder(resp.Body).Decode(&contactsResp); err != nil {
			return stats, fmt.Errorf("failed to decode response: %w", err)
		}

		// Process each contact
		for _, contact := range contactsResp.Results {
			// Marshal properties to JSON for storage
			propertiesJSON, err := json.Marshal(contact.Properties)
			if err != nil {
				continue
			}

			// Helper to get string properties
			getProp := func(key string) *string {
				if val, ok := contact.Properties[key].(string); ok && val != "" {
					return &val
				}
				return nil
			}

			// Convert timestamps
			createdAt := pgtype.Timestamptz{}
			_ = createdAt.Scan(contact.CreatedAt)

			updatedAt := pgtype.Timestamptz{}
			_ = updatedAt.Scan(contact.UpdatedAt)

			// Upsert contact
			result, err := queries.UpsertHubSpotContact(ctx, sqlc.UpsertHubSpotContactParams{
				UserID:           userID,
				HubspotID:        contact.ID,
				Email:            getProp("email"),
				FirstName:        getProp("firstname"),
				LastName:         getProp("lastname"),
				Phone:            getProp("phone"),
				Company:          getProp("company"),
				JobTitle:         getProp("jobtitle"),
				LifecycleStage:   getProp("lifecyclestage"),
				LeadStatus:       getProp("hs_lead_status"),
				OwnerID:          getProp("hubspot_owner_id"),
				Properties:       propertiesJSON,
				CreatedAtHubspot: createdAt,
				UpdatedAtHubspot: updatedAt,
			})

			if err != nil {
				continue // Skip errors for individual contacts
			}

			// Check if it was an insert or update based on created_at == updated_at
			if result.CreatedAt.Time.Equal(result.UpdatedAt.Time) {
				stats.Created++
			} else {
				stats.Updated++
			}
		}

		// Check if there are more pages
		if contactsResp.Paging.Next.After == "" {
			break
		}
		after = contactsResp.Paging.Next.After
	}

	return stats, nil
}

// GetContacts retrieves contacts for a user from the database.
func (p *Provider) GetContacts(ctx context.Context, userID string, limit, offset int32) ([]sqlc.HubspotContact, error) {
	queries := sqlc.New(p.pool)
	return queries.GetHubSpotContactsByUser(ctx, sqlc.GetHubSpotContactsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// CreateContact creates a new contact in HubSpot.
func (p *Provider) CreateContact(ctx context.Context, userID string, email, firstName, lastName string) error {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return err
	}

	// Build request body
	reqBody := map[string]interface{}{
		"properties": map[string]string{
			"email":     email,
			"firstname": firstName,
			"lastname":  lastName,
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", BaseAPIURL+"/crm/v3/objects/contacts", bytes.NewReader(reqJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HubSpot API error: %d", resp.StatusCode)
	}

	return nil
}
