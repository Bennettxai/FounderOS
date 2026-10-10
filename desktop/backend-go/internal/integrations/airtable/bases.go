// Package airtable provides the Airtable integration (Bases, Tables, Records).
package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ============================================================================
// Bases API Methods
// ============================================================================

// GetBases retrieves all bases accessible to the authenticated user.
// API Reference: https://airtable.com/developers/web/api/list-bases
func (p *Provider) GetBases(ctx context.Context, userID string) ([]Base, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.airtable.com/v0/meta/bases", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get bases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bases: status %d", resp.StatusCode)
	}

	var result struct {
		Bases  []Base `json:"bases"`
		Offset string `json:"offset,omitempty"` // For pagination if needed
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Bases, nil
}

// GetBase retrieves a specific base by ID.
// API Reference: https://airtable.com/developers/web/api/get-base-schema
func (p *Provider) GetBase(ctx context.Context, userID, baseID string) (*Base, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Get base metadata
	url := fmt.Sprintf("https://api.airtable.com/v0/meta/bases/%s/tables", baseID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get base: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get base: status %d", resp.StatusCode)
	}

	// Note: The /meta/bases/{baseId}/tables endpoint returns table schema
	// To get base info, we need to call /meta/bases and filter
	bases, err := p.GetBases(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, base := range bases {
		if base.ID == baseID {
			return &base, nil
		}
	}

	return nil, fmt.Errorf("base not found: %s", baseID)
}

// SyncBases fetches bases from Airtable API and persists them to the database.
func (p *Provider) SyncBases(ctx context.Context, userID string) (int, error) {
	bases, err := p.GetBases(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch bases: %w", err)
	}

	queries := sqlc.New(p.pool)
	synced := 0

	for _, base := range bases {
		var permLevel *string
		if base.PermissionLevel != "" {
			permLevel = &base.PermissionLevel
		}

		_, err := queries.UpsertAirtableBase(ctx, sqlc.UpsertAirtableBaseParams{
			UserID:          userID,
			BaseID:          base.ID,
			Name:            base.Name,
			PermissionLevel: permLevel,
		})
		if err != nil {
			slog.Warn("Failed to upsert base", "base_id", base.ID, "error", err)
			continue
		}
		synced++
	}

	return synced, nil
}
