package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleSheet represents a synced Google Sheet.
type GoogleSheet struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	SpreadsheetID string       `json:"spreadsheet_id"`
	DriveFileID   string       `json:"drive_file_id,omitempty"`
	Title         string       `json:"title"`
	Locale        string       `json:"locale,omitempty"`
	TimeZone      string       `json:"time_zone,omitempty"`
	SheetCount    int          `json:"sheet_count"`
	Sheets        []SheetInfo  `json:"sheets,omitempty"`
	NamedRanges   []NamedRange `json:"named_ranges,omitempty"`
	CreatedTime   time.Time    `json:"created_time,omitempty"`
	ModifiedTime  time.Time    `json:"modified_time,omitempty"`
	SyncedAt      time.Time    `json:"synced_at"`
}

// SheetInfo represents information about a sheet within a spreadsheet.
type SheetInfo struct {
	SheetID    int64  `json:"sheet_id"`
	Title      string `json:"title"`
	Index      int64  `json:"index"`
	SheetType  string `json:"sheet_type"` // GRID, OBJECT, etc.
	RowCount   int64  `json:"row_count,omitempty"`
	ColumnCount int64 `json:"column_count,omitempty"`
	Hidden     bool   `json:"hidden"`
}

// NamedRange represents a named range in a spreadsheet.
type NamedRange struct {
	NamedRangeID string `json:"named_range_id"`
	Name         string `json:"name"`
	Range        string `json:"range"`
}

// SheetsService handles Google Sheets operations.
type SheetsService struct {
	provider *Provider
}

// NewSheetsService creates a new Sheets service.
func NewSheetsService(provider *Provider) *SheetsService {
	return &SheetsService{provider: provider}
}

// GetSheetsAPI returns a Google Sheets API service for a user.
func (s *SheetsService) GetSheetsAPI(ctx context.Context, userID string) (*sheets.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return srv, nil
}

// SyncSpreadsheet syncs a single Google Sheet by its ID.
func (s *SheetsService) SyncSpreadsheet(ctx context.Context, userID, spreadsheetID string) (*GoogleSheet, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Sheets API: %w", err)
	}

	spreadsheet, err := srv.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	return s.saveSpreadsheet(ctx, userID, spreadsheet)
}

// saveSpreadsheet saves a Google Sheet to the database.
func (s *SheetsService) saveSpreadsheet(ctx context.Context, userID string, spreadsheet *sheets.Spreadsheet) (*GoogleSheet, error) {
	// Extract properties
	title := spreadsheet.Properties.Title
	locale := spreadsheet.Properties.Locale
	timeZone := spreadsheet.Properties.TimeZone

	// Extract sheets info
	sheetInfos := make([]SheetInfo, 0)
	for _, sheet := range spreadsheet.Sheets {
		props := sheet.Properties
		info := SheetInfo{
			SheetID:   props.SheetId,
			Title:     props.Title,
			Index:     props.Index,
			SheetType: props.SheetType,
			Hidden:    props.Hidden,
		}
		if props.GridProperties != nil {
			info.RowCount = props.GridProperties.RowCount
			info.ColumnCount = props.GridProperties.ColumnCount
		}
		sheetInfos = append(sheetInfos, info)
	}

	// Extract named ranges
	namedRanges := make([]NamedRange, 0)
	for _, nr := range spreadsheet.NamedRanges {
		namedRanges = append(namedRanges, NamedRange{
			NamedRangeID: nr.NamedRangeId,
			Name:         nr.Name,
			Range:        formatRange(nr.Range),
		})
	}

	// Insert or update spreadsheet
	var id string
	err := s.provider.Pool().QueryRow(ctx, `
		INSERT INTO google_sheets (
			user_id, spreadsheet_id, title, locale, time_zone, sheet_count, sheets, named_ranges, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id, spreadsheet_id) DO UPDATE SET
			title = EXCLUDED.title,
			locale = EXCLUDED.locale,
			time_zone = EXCLUDED.time_zone,
			sheet_count = EXCLUDED.sheet_count,
			sheets = EXCLUDED.sheets,
			named_ranges = EXCLUDED.named_ranges,
			synced_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, userID, spreadsheet.SpreadsheetId, title, locale, timeZone, len(spreadsheet.Sheets), sheetInfos, namedRanges).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to save spreadsheet: %w", err)
	}

	return &GoogleSheet{
		ID:            id,
		UserID:        userID,
		SpreadsheetID: spreadsheet.SpreadsheetId,
		Title:         title,
		Locale:        locale,
		TimeZone:      timeZone,
		SheetCount:    len(spreadsheet.Sheets),
		Sheets:        sheetInfos,
		NamedRanges:   namedRanges,
		SyncedAt:      time.Now(),
	}, nil
}

// formatRange formats a GridRange to a string like "Sheet1!A1:B10".
func formatRange(r *sheets.GridRange) string {
	if r == nil {
		return ""
	}
	// Note: This is a simplified format. Full A1 notation conversion is complex.
	return fmt.Sprintf("SheetID:%d", r.SheetId)
}

// GetSpreadsheets retrieves Google Sheets for a user.
func (s *SheetsService) GetSpreadsheets(ctx context.Context, userID string, limit, offset int) ([]*GoogleSheet, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, spreadsheet_id, title, locale, time_zone, sheet_count,
			created_time, modified_time, synced_at
		FROM google_sheets
		WHERE user_id = $1
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spreadsheets []*GoogleSheet
	for rows.Next() {
		var sheet GoogleSheet
		var locale, timeZone pgtype.Text
		var createdTime, modifiedTime pgtype.Timestamptz

		err := rows.Scan(
			&sheet.ID, &sheet.UserID, &sheet.SpreadsheetID, &sheet.Title, &locale, &timeZone, &sheet.SheetCount,
			&createdTime, &modifiedTime, &sheet.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		sheet.Locale = locale.String
		sheet.TimeZone = timeZone.String
		if createdTime.Valid {
			sheet.CreatedTime = createdTime.Time
		}
		if modifiedTime.Valid {
			sheet.ModifiedTime = modifiedTime.Time
		}

		spreadsheets = append(spreadsheets, &sheet)
	}

	return spreadsheets, nil
}

// GetSpreadsheet retrieves a single Google Sheet by spreadsheet ID.
func (s *SheetsService) GetSpreadsheet(ctx context.Context, userID, spreadsheetID string) (*GoogleSheet, error) {
	var sheet GoogleSheet
	var locale, timeZone pgtype.Text
	var sheetsData, namedRangesData []byte
	var createdTime, modifiedTime pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, spreadsheet_id, title, locale, time_zone, sheet_count, sheets, named_ranges,
			created_time, modified_time, synced_at
		FROM google_sheets
		WHERE user_id = $1 AND spreadsheet_id = $2
	`, userID, spreadsheetID).Scan(
		&sheet.ID, &sheet.UserID, &sheet.SpreadsheetID, &sheet.Title, &locale, &timeZone, &sheet.SheetCount, &sheetsData, &namedRangesData,
		&createdTime, &modifiedTime, &sheet.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	sheet.Locale = locale.String
	sheet.TimeZone = timeZone.String
	if createdTime.Valid {
		sheet.CreatedTime = createdTime.Time
	}
	if modifiedTime.Valid {
		sheet.ModifiedTime = modifiedTime.Time
	}

	return &sheet, nil
}

// SearchSpreadsheets searches spreadsheets by title.
func (s *SheetsService) SearchSpreadsheets(ctx context.Context, userID, query string, limit int) ([]*GoogleSheet, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, spreadsheet_id, title, sheet_count, synced_at
		FROM google_sheets
		WHERE user_id = $1 AND title ILIKE $2
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spreadsheets []*GoogleSheet
	for rows.Next() {
		var sheet GoogleSheet
		err := rows.Scan(&sheet.ID, &sheet.UserID, &sheet.SpreadsheetID, &sheet.Title, &sheet.SheetCount, &sheet.SyncedAt)
		if err != nil {
			return nil, err
		}
		spreadsheets = append(spreadsheets, &sheet)
	}

	return spreadsheets, nil
}

// CreateSpreadsheet creates a new Google Sheet.
func (s *SheetsService) CreateSpreadsheet(ctx context.Context, userID, title string) (*sheets.Spreadsheet, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
	}

	created, err := srv.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create spreadsheet: %w", err)
	}

	// Save to database
	if _, err := s.saveSpreadsheet(ctx, userID, created); err != nil {
		slog.Info("Failed to save created spreadsheet to database", "error", err)
	}

	return created, nil
}

// GetValues reads values from a range in a spreadsheet.
func (s *SheetsService) GetValues(ctx context.Context, userID, spreadsheetID, rangeStr string) (*sheets.ValueRange, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	return srv.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Do()
}

// UpdateValues writes values to a range in a spreadsheet.
func (s *SheetsService) UpdateValues(ctx context.Context, userID, spreadsheetID, rangeStr string, values [][]interface{}) (*sheets.UpdateValuesResponse, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	return srv.Spreadsheets.Values.Update(spreadsheetID, rangeStr, valueRange).
		ValueInputOption("USER_ENTERED").
		Do()
}

// AppendValues appends values to a spreadsheet.
func (s *SheetsService) AppendValues(ctx context.Context, userID, spreadsheetID, rangeStr string, values [][]interface{}) (*sheets.AppendValuesResponse, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	return srv.Spreadsheets.Values.Append(spreadsheetID, rangeStr, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
}

// ClearValues clears values from a range in a spreadsheet.
func (s *SheetsService) ClearValues(ctx context.Context, userID, spreadsheetID, rangeStr string) error {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return err
	}

	_, err = srv.Spreadsheets.Values.Clear(spreadsheetID, rangeStr, &sheets.ClearValuesRequest{}).Do()
	return err
}

// AddSheet adds a new sheet to an existing spreadsheet.
func (s *SheetsService) AddSheet(ctx context.Context, userID, spreadsheetID, sheetTitle string) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	request := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetTitle,
					},
				},
			},
		},
	}

	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheetID, request).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add sheet: %w", err)
	}

	// Re-sync the spreadsheet
	if _, err := s.SyncSpreadsheet(ctx, userID, spreadsheetID); err != nil {
		slog.Info("Failed to re-sync spreadsheet after adding sheet", "error", err)
	}

	return resp, nil
}

// DeleteSheet removes a sheet from a spreadsheet.
func (s *SheetsService) DeleteSheet(ctx context.Context, userID, spreadsheetID string, sheetID int64) error {
	srv, err := s.GetSheetsAPI(ctx, userID)
	if err != nil {
		return err
	}

	request := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: sheetID,
				},
			},
		},
	}

	_, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, request).Do()
	if err != nil {
		return fmt.Errorf("failed to delete sheet: %w", err)
	}

	// Re-sync the spreadsheet
	if _, err := s.SyncSpreadsheet(ctx, userID, spreadsheetID); err != nil {
		slog.Info("Failed to re-sync spreadsheet after deleting sheet", "error", err)
	}

	return nil
}

// IsConnected checks if Google Sheets is connected for a user.
func (s *SheetsService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsSheetsScope(scope) {
			return true
		}
	}
	return false
}

func containsSheetsScope(scope string) bool {
	sheetsScopes := []string{
		"https://www.googleapis.com/auth/spreadsheets",
		"spreadsheets",
		"spreadsheets.readonly",
	}
	for _, s := range sheetsScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
