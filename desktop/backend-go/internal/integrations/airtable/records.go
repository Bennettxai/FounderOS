// Package airtable provides the Airtable integration (Bases, Tables, Records).
package airtable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ============================================================================
// Records API Methods (CRUD Operations)
// ============================================================================

// GetRecords retrieves records from a specific table.
// tableIDOrName can be either the table ID (tblXXXXXXXXXXXXXX) or table name.
// API Reference: https://airtable.com/developers/web/api/list-records
func (p *Provider) GetRecords(ctx context.Context, userID, baseID, tableIDOrName string, options *RecordQueryOptions) (*RecordList, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Build URL with query parameters
	baseURL := fmt.Sprintf("%s/%s/%s", APIURL, baseID, url.PathEscape(tableIDOrName))
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters if options provided
	if options != nil {
		query := reqURL.Query()
		if options.MaxRecords > 0 {
			query.Set("maxRecords", fmt.Sprintf("%d", options.MaxRecords))
		}
		if options.PageSize > 0 {
			query.Set("pageSize", fmt.Sprintf("%d", options.PageSize))
		}
		if options.Offset != "" {
			query.Set("offset", options.Offset)
		}
		if options.View != "" {
			query.Set("view", options.View)
		}
		if options.FilterByFormula != "" {
			query.Set("filterByFormula", options.FilterByFormula)
		}
		if options.Sort != "" {
			query.Set("sort", options.Sort)
		}
		reqURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get records: status %d", resp.StatusCode)
	}

	var recordList RecordList
	if err := json.NewDecoder(resp.Body).Decode(&recordList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &recordList, nil
}

// CreateRecord creates a new record in a table.
// API Reference: https://airtable.com/developers/web/api/create-records
func (p *Provider) CreateRecord(ctx context.Context, userID, baseID, tableIDOrName string, fields map[string]interface{}) (*Record, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL := fmt.Sprintf("%s/%s/%s", APIURL, baseID, url.PathEscape(tableIDOrName))

	// Build request body
	requestBody := map[string]interface{}{
		"fields": fields,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create record: status %d", resp.StatusCode)
	}

	var record Record
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &record, nil
}

// UpdateRecord updates an existing record in a table.
// API Reference: https://airtable.com/developers/web/api/update-record
func (p *Provider) UpdateRecord(ctx context.Context, userID, baseID, tableIDOrName, recordID string, fields map[string]interface{}) (*Record, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL := fmt.Sprintf("%s/%s/%s/%s", APIURL, baseID, url.PathEscape(tableIDOrName), recordID)

	// Build request body
	requestBody := map[string]interface{}{
		"fields": fields,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update record: status %d", resp.StatusCode)
	}

	var record Record
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &record, nil
}

// DeleteRecord deletes a record from a table.
// API Reference: https://airtable.com/developers/web/api/delete-record
func (p *Provider) DeleteRecord(ctx context.Context, userID, baseID, tableIDOrName, recordID string) error {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	requestURL := fmt.Sprintf("%s/%s/%s/%s", APIURL, baseID, url.PathEscape(tableIDOrName), recordID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete record: status %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// Query Options
// ============================================================================

// RecordQueryOptions defines options for querying records.
type RecordQueryOptions struct {
	MaxRecords      int    // Maximum total number of records to retrieve
	PageSize        int    // Number of records per page (max 100)
	Offset          string // Pagination offset from previous response
	View            string // View name or ID to use
	FilterByFormula string // Airtable formula to filter records
	Sort            string // Sort specification (e.g., "field1 asc, field2 desc")
}

// SyncRecords fetches records from Airtable API and persists them to the database.
func (p *Provider) SyncRecords(ctx context.Context, userID, baseID, tableID string, limit int) (int, error) {
	options := &RecordQueryOptions{
		MaxRecords: limit,
	}

	recordList, err := p.GetRecords(ctx, userID, baseID, tableID, options)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch records: %w", err)
	}

	queries := sqlc.New(p.pool)
	synced := 0

	for _, record := range recordList.Records {
		// Marshal fields to JSON
		fieldsJSON, err := json.Marshal(record.Fields)
		if err != nil {
			slog.Warn("Failed to marshal fields for record", "record_id", record.ID, "error", err)
			continue
		}

		// Parse created time
		var createdTime pgtype.Timestamptz
		if record.CreatedTime != "" {
			// Airtable returns ISO 8601 format
			// We'll let PostgreSQL parse it
			if err := createdTime.Scan(record.CreatedTime); err != nil {
				slog.Warn("Failed to parse created time for record", "record_id", record.ID, "error", err)
			}
		}

		_, err = queries.UpsertAirtableRecord(ctx, sqlc.UpsertAirtableRecordParams{
			UserID:              userID,
			RecordID:            record.ID,
			TableID:             tableID,
			BaseID:              baseID,
			Fields:              fieldsJSON,
			CreatedTimeAirtable: createdTime,
		})
		if err != nil {
			slog.Warn("Failed to upsert record", "record_id", record.ID, "error", err)
			continue
		}
		synced++
	}

	return synced, nil
}
