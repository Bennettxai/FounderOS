package google

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

// GoogleDoc represents a synced Google Doc.
type GoogleDoc struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	DocumentID   string       `json:"document_id"`
	DriveFileID  string       `json:"drive_file_id,omitempty"`
	Title        string       `json:"title"`
	BodyText     string       `json:"body_text,omitempty"`
	WordCount    int          `json:"word_count"`
	Headers      []DocHeader  `json:"headers,omitempty"`
	Locale       string       `json:"locale,omitempty"`
	CreatedTime  time.Time    `json:"created_time,omitempty"`
	ModifiedTime time.Time    `json:"modified_time,omitempty"`
	SyncedAt     time.Time    `json:"synced_at"`
}

// DocHeader represents a header/heading in a document.
type DocHeader struct {
	Level int    `json:"level"` // 1-6 for H1-H6
	Text  string `json:"text"`
}

// DocsService handles Google Docs operations.
type DocsService struct {
	provider *Provider
}

// NewDocsService creates a new Docs service.
func NewDocsService(provider *Provider) *DocsService {
	return &DocsService{provider: provider}
}

// GetDocsAPI returns a Google Docs API service for a user.
func (s *DocsService) GetDocsAPI(ctx context.Context, userID string) (*docs.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := docs.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	return srv, nil
}

// SyncDocument syncs a single Google Doc by its ID.
func (s *DocsService) SyncDocument(ctx context.Context, userID, documentID string) (*GoogleDoc, error) {
	srv, err := s.GetDocsAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docs API: %w", err)
	}

	doc, err := srv.Documents.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return s.saveDocument(ctx, userID, doc)
}

// saveDocument saves a Google Doc to the database.
func (s *DocsService) saveDocument(ctx context.Context, userID string, doc *docs.Document) (*GoogleDoc, error) {
	// Extract body text
	bodyText := extractDocumentText(doc)
	wordCount := len(strings.Fields(bodyText))

	// Extract headers
	headers := extractDocumentHeaders(doc)

	// Get locale
	locale := ""
	if doc.DocumentStyle != nil {
		// Note: Locale might not be directly available; using default
	}

	// Insert or update document
	var id string
	err := s.provider.Pool().QueryRow(ctx, `
		INSERT INTO google_docs (
			user_id, document_id, title, body_text, word_count, headers, locale, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, document_id) DO UPDATE SET
			title = EXCLUDED.title,
			body_text = EXCLUDED.body_text,
			word_count = EXCLUDED.word_count,
			headers = EXCLUDED.headers,
			locale = EXCLUDED.locale,
			synced_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, userID, doc.DocumentId, doc.Title, bodyText, wordCount, headers, locale).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &GoogleDoc{
		ID:         id,
		UserID:     userID,
		DocumentID: doc.DocumentId,
		Title:      doc.Title,
		BodyText:   bodyText,
		WordCount:  wordCount,
		Headers:    headers,
		Locale:     locale,
		SyncedAt:   time.Now(),
	}, nil
}

// extractDocumentText extracts plain text from a Google Doc.
func extractDocumentText(doc *docs.Document) string {
	var text strings.Builder

	if doc.Body == nil || doc.Body.Content == nil {
		return ""
	}

	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			for _, e := range element.Paragraph.Elements {
				if e.TextRun != nil {
					text.WriteString(e.TextRun.Content)
				}
			}
		}
		if element.Table != nil {
			for _, row := range element.Table.TableRows {
				for _, cell := range row.TableCells {
					for _, content := range cell.Content {
						if content.Paragraph != nil {
							for _, e := range content.Paragraph.Elements {
								if e.TextRun != nil {
									text.WriteString(e.TextRun.Content)
									text.WriteString(" ")
								}
							}
						}
					}
				}
			}
		}
	}

	return text.String()
}

// extractDocumentHeaders extracts headers/headings from a Google Doc.
func extractDocumentHeaders(doc *docs.Document) []DocHeader {
	var headers []DocHeader

	if doc.Body == nil || doc.Body.Content == nil {
		return headers
	}

	for _, element := range doc.Body.Content {
		if element.Paragraph != nil && element.Paragraph.ParagraphStyle != nil {
			style := element.Paragraph.ParagraphStyle.NamedStyleType
			level := 0

			switch style {
			case "HEADING_1":
				level = 1
			case "HEADING_2":
				level = 2
			case "HEADING_3":
				level = 3
			case "HEADING_4":
				level = 4
			case "HEADING_5":
				level = 5
			case "HEADING_6":
				level = 6
			}

			if level > 0 {
				var text strings.Builder
				for _, e := range element.Paragraph.Elements {
					if e.TextRun != nil {
						text.WriteString(e.TextRun.Content)
					}
				}
				headers = append(headers, DocHeader{
					Level: level,
					Text:  strings.TrimSpace(text.String()),
				})
			}
		}
	}

	return headers
}

// GetDocuments retrieves Google Docs for a user.
func (s *DocsService) GetDocuments(ctx context.Context, userID string, limit, offset int) ([]*GoogleDoc, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, document_id, title, body_text, word_count, headers, locale,
			created_time, modified_time, synced_at
		FROM google_docs
		WHERE user_id = $1
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []*GoogleDoc
	for rows.Next() {
		var d GoogleDoc
		var bodyText, locale pgtype.Text
		var headers []byte
		var createdTime, modifiedTime pgtype.Timestamptz

		err := rows.Scan(
			&d.ID, &d.UserID, &d.DocumentID, &d.Title, &bodyText, &d.WordCount, &headers, &locale,
			&createdTime, &modifiedTime, &d.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		d.BodyText = bodyText.String
		d.Locale = locale.String
		if createdTime.Valid {
			d.CreatedTime = createdTime.Time
		}
		if modifiedTime.Valid {
			d.ModifiedTime = modifiedTime.Time
		}

		documents = append(documents, &d)
	}

	return documents, nil
}

// GetDocument retrieves a single Google Doc by document ID.
func (s *DocsService) GetDocument(ctx context.Context, userID, documentID string) (*GoogleDoc, error) {
	var d GoogleDoc
	var bodyText, locale pgtype.Text
	var headers []byte
	var createdTime, modifiedTime pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, document_id, title, body_text, word_count, headers, locale,
			created_time, modified_time, synced_at
		FROM google_docs
		WHERE user_id = $1 AND document_id = $2
	`, userID, documentID).Scan(
		&d.ID, &d.UserID, &d.DocumentID, &d.Title, &bodyText, &d.WordCount, &headers, &locale,
		&createdTime, &modifiedTime, &d.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	d.BodyText = bodyText.String
	d.Locale = locale.String
	if createdTime.Valid {
		d.CreatedTime = createdTime.Time
	}
	if modifiedTime.Valid {
		d.ModifiedTime = modifiedTime.Time
	}

	return &d, nil
}

// SearchDocuments searches documents by title or content.
func (s *DocsService) SearchDocuments(ctx context.Context, userID, query string, limit int) ([]*GoogleDoc, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, document_id, title, word_count, synced_at
		FROM google_docs
		WHERE user_id = $1
			AND (title ILIKE $2 OR body_text ILIKE $2)
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []*GoogleDoc
	for rows.Next() {
		var d GoogleDoc
		err := rows.Scan(&d.ID, &d.UserID, &d.DocumentID, &d.Title, &d.WordCount, &d.SyncedAt)
		if err != nil {
			return nil, err
		}
		documents = append(documents, &d)
	}

	return documents, nil
}

// CreateDocument creates a new Google Doc.
func (s *DocsService) CreateDocument(ctx context.Context, userID, title string) (*docs.Document, error) {
	srv, err := s.GetDocsAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	doc := &docs.Document{
		Title: title,
	}

	created, err := srv.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Save to database
	if _, err := s.saveDocument(ctx, userID, created); err != nil {
		slog.Info("Failed to save created document to database", "error", err)
	}

	return created, nil
}

// UpdateDocumentTitle updates a document's title.
func (s *DocsService) UpdateDocumentTitle(ctx context.Context, userID, documentID, newTitle string) error {
	// Note: Google Docs API doesn't have a direct way to update the title
	// You need to use Drive API to rename the file
	// For now, just update the local database
	_, err := s.provider.Pool().Exec(ctx, `
		UPDATE google_docs SET title = $1, updated_at = NOW()
		WHERE user_id = $2 AND document_id = $3
	`, newTitle, userID, documentID)

	return err
}

// AppendText appends text to a Google Doc.
func (s *DocsService) AppendText(ctx context.Context, userID, documentID, text string) error {
	srv, err := s.GetDocsAPI(ctx, userID)
	if err != nil {
		return err
	}

	// Get current document to find the end index
	doc, err := srv.Documents.Get(documentID).Do()
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	// Find the end index
	endIndex := int64(1)
	if doc.Body != nil && len(doc.Body.Content) > 0 {
		lastElement := doc.Body.Content[len(doc.Body.Content)-1]
		endIndex = lastElement.EndIndex - 1
	}

	// Create the insert request
	request := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Text: text,
					Location: &docs.Location{
						Index: endIndex,
					},
				},
			},
		},
	}

	_, err = srv.Documents.BatchUpdate(documentID, request).Do()
	if err != nil {
		return fmt.Errorf("failed to append text: %w", err)
	}

	// Re-sync the document
	if _, err := s.SyncDocument(ctx, userID, documentID); err != nil {
		slog.Info("Failed to re-sync document after append", "error", err)
	}

	return nil
}

// IsConnected checks if Google Docs is connected for a user.
func (s *DocsService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsDocsScope(scope) {
			return true
		}
	}
	return false
}

func containsDocsScope(scope string) bool {
	docsScopes := []string{
		"https://www.googleapis.com/auth/documents",
		"documents",
		"documents.readonly",
	}
	for _, s := range docsScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
