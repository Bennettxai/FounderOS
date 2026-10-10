// Package airtable provides the Airtable integration (Bases, Tables, Records).
package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ============================================================================
// Tables API Methods
// ============================================================================

// GetTables retrieves all tables in a specific base.
// API Reference: https://airtable.com/developers/web/api/get-base-schema
func (p *Provider) GetTables(ctx context.Context, userID, baseID string) ([]Table, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://api.airtable.com/v0/meta/bases/%s/tables", baseID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get tables: status %d", resp.StatusCode)
	}

	var result struct {
		Tables []Table `json:"tables"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Tables, nil
}

// GetTable retrieves a specific table by ID or name from a base.
// tableIDOrName can be either the table ID (tblXXXXXXXXXXXXXX) or table name.
// API Reference: https://airtable.com/developers/web/api/get-base-schema
func (p *Provider) GetTable(ctx context.Context, userID, baseID, tableIDOrName string) (*Table, error) {
	// Get all tables in the base
	tables, err := p.GetTables(ctx, userID, baseID)
	if err != nil {
		return nil, err
	}

	// Search by ID or name
	for _, table := range tables {
		if table.ID == tableIDOrName || table.Name == tableIDOrName {
			return &table, nil
		}
	}

	return nil, fmt.Errorf("table not found: %s", tableIDOrName)
}
