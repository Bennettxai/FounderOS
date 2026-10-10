package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

// GoogleSlide represents a synced Google Slides presentation.
type GoogleSlide struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	PresentationID string      `json:"presentation_id"`
	DriveFileID    string      `json:"drive_file_id,omitempty"`
	Title          string      `json:"title"`
	Locale         string      `json:"locale,omitempty"`
	SlideCount     int         `json:"slide_count"`
	Slides         []SlideInfo `json:"slides,omitempty"`
	PageWidth      float64     `json:"page_width,omitempty"`
	PageHeight     float64     `json:"page_height,omitempty"`
	CreatedTime    time.Time   `json:"created_time,omitempty"`
	ModifiedTime   time.Time   `json:"modified_time,omitempty"`
	SyncedAt       time.Time   `json:"synced_at"`
}

// SlideInfo represents information about a slide in a presentation.
type SlideInfo struct {
	ObjectID    string   `json:"object_id"`
	SlideIndex  int      `json:"slide_index"`
	Title       string   `json:"title,omitempty"`
	TextContent string   `json:"text_content,omitempty"`
	HasImages   bool     `json:"has_images"`
	HasTables   bool     `json:"has_tables"`
	HasCharts   bool     `json:"has_charts"`
	LayoutID    string   `json:"layout_id,omitempty"`
	MasterID    string   `json:"master_id,omitempty"`
}

// SlidesService handles Google Slides operations.
type SlidesService struct {
	provider *Provider
}

// NewSlidesService creates a new Slides service.
func NewSlidesService(provider *Provider) *SlidesService {
	return &SlidesService{provider: provider}
}

// GetSlidesAPI returns a Google Slides API service for a user.
func (s *SlidesService) GetSlidesAPI(ctx context.Context, userID string) (*slides.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := slides.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create slides service: %w", err)
	}

	return srv, nil
}

// SyncPresentation syncs a single Google Slides presentation by its ID.
func (s *SlidesService) SyncPresentation(ctx context.Context, userID, presentationID string) (*GoogleSlide, error) {
	srv, err := s.GetSlidesAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slides API: %w", err)
	}

	presentation, err := srv.Presentations.Get(presentationID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get presentation: %w", err)
	}

	return s.savePresentation(ctx, userID, presentation)
}

// savePresentation saves a Google Slides presentation to the database.
func (s *SlidesService) savePresentation(ctx context.Context, userID string, presentation *slides.Presentation) (*GoogleSlide, error) {
	// Extract properties
	title := presentation.Title
	locale := presentation.Locale

	// Extract page size
	var pageWidth, pageHeight float64
	if presentation.PageSize != nil {
		if presentation.PageSize.Width != nil {
			pageWidth = presentation.PageSize.Width.Magnitude
		}
		if presentation.PageSize.Height != nil {
			pageHeight = presentation.PageSize.Height.Magnitude
		}
	}

	// Extract slides info
	slideInfos := make([]SlideInfo, 0)
	for i, slide := range presentation.Slides {
		info := SlideInfo{
			ObjectID:   slide.ObjectId,
			SlideIndex: i,
		}

		// Check for various element types
		for _, element := range slide.PageElements {
			if element.Shape != nil && element.Shape.Text != nil {
				// Extract text content
				for _, te := range element.Shape.Text.TextElements {
					if te.TextRun != nil {
						info.TextContent += te.TextRun.Content
					}
				}
			}
			if element.Image != nil {
				info.HasImages = true
			}
			if element.Table != nil {
				info.HasTables = true
			}
			if element.SheetsChart != nil {
				info.HasCharts = true
			}
		}

		// Get layout and master IDs
		if slide.SlideProperties != nil {
			info.LayoutID = slide.SlideProperties.LayoutObjectId
			info.MasterID = slide.SlideProperties.MasterObjectId
		}

		slideInfos = append(slideInfos, info)
	}

	// Insert or update presentation
	var id string
	err := s.provider.Pool().QueryRow(ctx, `
		INSERT INTO google_slides (
			user_id, presentation_id, title, locale, slide_count, slides, page_width, page_height, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id, presentation_id) DO UPDATE SET
			title = EXCLUDED.title,
			locale = EXCLUDED.locale,
			slide_count = EXCLUDED.slide_count,
			slides = EXCLUDED.slides,
			page_width = EXCLUDED.page_width,
			page_height = EXCLUDED.page_height,
			synced_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, userID, presentation.PresentationId, title, locale, len(presentation.Slides), slideInfos, pageWidth, pageHeight).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to save presentation: %w", err)
	}

	return &GoogleSlide{
		ID:             id,
		UserID:         userID,
		PresentationID: presentation.PresentationId,
		Title:          title,
		Locale:         locale,
		SlideCount:     len(presentation.Slides),
		Slides:         slideInfos,
		PageWidth:      pageWidth,
		PageHeight:     pageHeight,
		SyncedAt:       time.Now(),
	}, nil
}

// GetPresentations retrieves Google Slides presentations for a user.
func (s *SlidesService) GetPresentations(ctx context.Context, userID string, limit, offset int) ([]*GoogleSlide, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, presentation_id, title, locale, slide_count, page_width, page_height,
			created_time, modified_time, synced_at
		FROM google_slides
		WHERE user_id = $1
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presentations []*GoogleSlide
	for rows.Next() {
		var p GoogleSlide
		var locale pgtype.Text
		var pageWidth, pageHeight pgtype.Float8
		var createdTime, modifiedTime pgtype.Timestamptz

		err := rows.Scan(
			&p.ID, &p.UserID, &p.PresentationID, &p.Title, &locale, &p.SlideCount, &pageWidth, &pageHeight,
			&createdTime, &modifiedTime, &p.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		p.Locale = locale.String
		if pageWidth.Valid {
			p.PageWidth = pageWidth.Float64
		}
		if pageHeight.Valid {
			p.PageHeight = pageHeight.Float64
		}
		if createdTime.Valid {
			p.CreatedTime = createdTime.Time
		}
		if modifiedTime.Valid {
			p.ModifiedTime = modifiedTime.Time
		}

		presentations = append(presentations, &p)
	}

	return presentations, nil
}

// GetPresentation retrieves a single Google Slides presentation by presentation ID.
func (s *SlidesService) GetPresentation(ctx context.Context, userID, presentationID string) (*GoogleSlide, error) {
	var p GoogleSlide
	var locale pgtype.Text
	var slidesData []byte
	var pageWidth, pageHeight pgtype.Float8
	var createdTime, modifiedTime pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, presentation_id, title, locale, slide_count, slides, page_width, page_height,
			created_time, modified_time, synced_at
		FROM google_slides
		WHERE user_id = $1 AND presentation_id = $2
	`, userID, presentationID).Scan(
		&p.ID, &p.UserID, &p.PresentationID, &p.Title, &locale, &p.SlideCount, &slidesData, &pageWidth, &pageHeight,
		&createdTime, &modifiedTime, &p.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Locale = locale.String
	if pageWidth.Valid {
		p.PageWidth = pageWidth.Float64
	}
	if pageHeight.Valid {
		p.PageHeight = pageHeight.Float64
	}
	if createdTime.Valid {
		p.CreatedTime = createdTime.Time
	}
	if modifiedTime.Valid {
		p.ModifiedTime = modifiedTime.Time
	}

	return &p, nil
}

// SearchPresentations searches presentations by title.
func (s *SlidesService) SearchPresentations(ctx context.Context, userID, query string, limit int) ([]*GoogleSlide, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, presentation_id, title, slide_count, synced_at
		FROM google_slides
		WHERE user_id = $1 AND title ILIKE $2
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presentations []*GoogleSlide
	for rows.Next() {
		var p GoogleSlide
		err := rows.Scan(&p.ID, &p.UserID, &p.PresentationID, &p.Title, &p.SlideCount, &p.SyncedAt)
		if err != nil {
			return nil, err
		}
		presentations = append(presentations, &p)
	}

	return presentations, nil
}

// CreatePresentation creates a new Google Slides presentation.
func (s *SlidesService) CreatePresentation(ctx context.Context, userID, title string) (*slides.Presentation, error) {
	srv, err := s.GetSlidesAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	presentation := &slides.Presentation{
		Title: title,
	}

	created, err := srv.Presentations.Create(presentation).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create presentation: %w", err)
	}

	// Save to database
	if _, err := s.savePresentation(ctx, userID, created); err != nil {
		slog.Info("Failed to save created presentation to database", "error", err)
	}

	return created, nil
}

// AddSlide adds a new slide to a presentation.
func (s *SlidesService) AddSlide(ctx context.Context, userID, presentationID string, layoutID string) (*slides.BatchUpdatePresentationResponse, error) {
	srv, err := s.GetSlidesAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	request := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateSlide: &slides.CreateSlideRequest{
					SlideLayoutReference: &slides.LayoutReference{
						LayoutId: layoutID,
					},
				},
			},
		},
	}

	resp, err := srv.Presentations.BatchUpdate(presentationID, request).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add slide: %w", err)
	}

	// Re-sync the presentation
	if _, err := s.SyncPresentation(ctx, userID, presentationID); err != nil {
		slog.Info("Failed to re-sync presentation after adding slide", "id", err)
	}

	return resp, nil
}

// DeleteSlide removes a slide from a presentation.
func (s *SlidesService) DeleteSlide(ctx context.Context, userID, presentationID, slideObjectID string) error {
	srv, err := s.GetSlidesAPI(ctx, userID)
	if err != nil {
		return err
	}

	request := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: slideObjectID,
				},
			},
		},
	}

	_, err = srv.Presentations.BatchUpdate(presentationID, request).Do()
	if err != nil {
		return fmt.Errorf("failed to delete slide: %w", err)
	}

	// Re-sync the presentation
	if _, err := s.SyncPresentation(ctx, userID, presentationID); err != nil {
		slog.Info("Failed to re-sync presentation after deleting slide", "id", err)
	}

	return nil
}

// AddTextToSlide adds a text box with text to a slide.
func (s *SlidesService) AddTextToSlide(ctx context.Context, userID, presentationID, slideObjectID, text string) error {
	srv, err := s.GetSlidesAPI(ctx, userID)
	if err != nil {
		return err
	}

	// Generate a unique object ID for the text box
	textBoxID := fmt.Sprintf("textbox_%d", time.Now().UnixNano())

	request := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateShape: &slides.CreateShapeRequest{
					ObjectId:  textBoxID,
					ShapeType: "TEXT_BOX",
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: slideObjectID,
						Size: &slides.Size{
							Width:  &slides.Dimension{Magnitude: 300, Unit: "PT"},
							Height: &slides.Dimension{Magnitude: 50, Unit: "PT"},
						},
						Transform: &slides.AffineTransform{
							ScaleX:     1,
							ScaleY:     1,
							TranslateX: 100,
							TranslateY: 100,
							Unit:       "PT",
						},
					},
				},
			},
			{
				InsertText: &slides.InsertTextRequest{
					ObjectId: textBoxID,
					Text:     text,
				},
			},
		},
	}

	_, err = srv.Presentations.BatchUpdate(presentationID, request).Do()
	if err != nil {
		return fmt.Errorf("failed to add text to slide: %w", err)
	}

	return nil
}

// IsConnected checks if Google Slides is connected for a user.
func (s *SlidesService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsSlidesScope(scope) {
			return true
		}
	}
	return false
}

func containsSlidesScope(scope string) bool {
	slidesScopes := []string{
		"https://www.googleapis.com/auth/presentations",
		"presentations",
		"presentations.readonly",
	}
	for _, s := range slidesScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
