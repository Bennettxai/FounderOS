package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// OSAAppService handles OSA app generation business logic
type OSAAppService struct {
	pool      *pgxpool.Pool
	queries   *sqlc.Queries
	osaClient *osa.Client
	eventBus  *BuildEventBus
	logger    *slog.Logger
}

// NewOSAAppService creates a new OSA app service
func NewOSAAppService(
	pool *pgxpool.Pool,
	queries *sqlc.Queries,
	osaClient *osa.Client,
	eventBus *BuildEventBus,
) *OSAAppService {
	return &OSAAppService{
		pool:      pool,
		queries:   queries,
		osaClient: osaClient,
		eventBus:  eventBus,
		logger:    slog.Default().With("service", "osa_app"),
	}
}

// GenerateAppRequest represents the input for app generation
type GenerateAppRequest struct {
	UserID       uuid.UUID
	WorkspaceID  *uuid.UUID // Optional - will create default if nil
	Name         string
	Description  string
	TemplateType string                 // Optional - defaults to "full-stack"
	Parameters   map[string]interface{} // Optional custom parameters
}

// GenerateAppResponse represents the result of app generation
type GenerateAppResponse struct {
	AppID       uuid.UUID
	WorkspaceID uuid.UUID
	Status      string
	Message     string
}

// GenerateApp initiates app generation and streams progress via SSE
// Returns a channel of events that can be consumed for SSE streaming
func (s *OSAAppService) GenerateApp(ctx context.Context, req *GenerateAppRequest) (<-chan streaming.StreamEvent, error) {
	s.logger.Info("starting app generation",
		"user_id", req.UserID,
		"name", req.Name,
		"template_type", req.TemplateType,
	)

	// Validate input
	if req.Name == "" {
		return nil, fmt.Errorf("app name is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("app description is required")
	}

	// Set defaults
	if req.TemplateType == "" {
		req.TemplateType = "full-stack"
	}

	// Get or create workspace
	workspaceID, err := s.ensureWorkspace(ctx, req.UserID, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure workspace: %w", err)
	}

	// Create event channel
	eventCh := make(chan streaming.StreamEvent, 10)

	// Launch generation in background
	go s.runGeneration(ctx, req, workspaceID, eventCh)

	return eventCh, nil
}

// runGeneration executes the generation workflow and sends progress events
func (s *OSAAppService) runGeneration(
	ctx context.Context,
	req *GenerateAppRequest,
	workspaceID uuid.UUID,
	eventCh chan<- streaming.StreamEvent,
) {
	defer close(eventCh)

	// Send initial event
	s.sendEvent(eventCh, streaming.StreamEvent{
		Type:    "progress",
		Content: "Initializing app generation...",
		Data: map[string]interface{}{
			"percent": 10,
			"phase":   "initializing",
		},
	})

	// Create database record
	appID, err := s.createAppRecord(ctx, req, workspaceID)
	if err != nil {
		s.logger.Error("failed to create app record", "error", err)
		s.sendEvent(eventCh, streaming.StreamEvent{
			Type:    streaming.EventTypeError,
			Content: fmt.Sprintf("Failed to create app record: %v", err),
		})
		return
	}

	s.logger.Info("created app record", "app_id", appID)

	// Send progress update
	s.sendEvent(eventCh, streaming.StreamEvent{
		Type:    "progress",
		Content: "Preparing generation prompt...",
		Data: map[string]interface{}{
			"percent": 25,
			"phase":   "prompt_generation",
			"app_id":  appID.String(),
		},
	})

	generatedPrompt := req.Description

	// Send progress update
	s.sendEvent(eventCh, streaming.StreamEvent{
		Type:    "progress",
		Content: "Calling OSA API to generate app...",
		Data: map[string]interface{}{
			"percent": 40,
			"phase":   "calling_osa",
		},
	})

	// Call OSA client
	osaReq := &osa.AppGenerationRequest{
		UserID:      req.UserID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: generatedPrompt,
		Type:        req.TemplateType,
		Parameters:  req.Parameters,
	}

	osaResp, err := s.osaClient.GenerateApp(ctx, osaReq)
	if err != nil {
		s.logger.Error("OSA API call failed", "error", err)
		// Update status to failed
		_ = s.updateAppStatus(ctx, appID, "failed")
		s.sendEvent(eventCh, streaming.StreamEvent{
			Type:    streaming.EventTypeError,
			Content: fmt.Sprintf("OSA API call failed: %v", err),
		})
		return
	}

	s.logger.Info("OSA API responded", "osa_response_app_id", osaResp.AppID, "status", osaResp.Status)

	// Send progress update
	s.sendEvent(eventCh, streaming.StreamEvent{
		Type:    "progress",
		Content: "App generation in progress...",
		Data: map[string]interface{}{
			"percent":             60,
			"phase":               "generating",
			"osa_response_app_id": osaResp.AppID,
		},
	})

	// Poll for status until complete (with timeout)
	if err := s.pollGenerationStatus(ctx, osaResp.AppID, req.UserID, appID, eventCh); err != nil {
		s.logger.Error("generation polling failed", "error", err)
		_ = s.updateAppStatus(ctx, appID, "failed")
		s.sendEvent(eventCh, streaming.StreamEvent{
			Type:    streaming.EventTypeError,
			Content: fmt.Sprintf("Generation polling failed: %v", err),
		})
		return
	}

	// Update final status
	if err := s.updateAppStatus(ctx, appID, "active"); err != nil {
		s.logger.Warn("failed to update final status", "error", err)
	}

	// Send completion event
	s.sendEvent(eventCh, streaming.StreamEvent{
		Type:    streaming.EventTypeDone,
		Content: "App generation completed successfully",
		Data: map[string]interface{}{
			"app_id":              appID.String(),
			"workspace_id":        workspaceID.String(),
			"status":              "active",
			"osa_response_app_id": osaResp.AppID,
			"deployment_url":      osaResp.Data["deployment_url"],
		},
	})
}

// pollGenerationStatus polls OSA API for generation progress
func (s *OSAAppService) pollGenerationStatus(
	ctx context.Context,
	osaAppID string,
	userID uuid.UUID,
	appID uuid.UUID,
	eventCh chan<- streaming.StreamEvent,
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("generation timeout after 5 minutes")
		case <-ticker.C:
			status, err := s.osaClient.GetAppStatus(ctx, osaAppID, userID)
			if err != nil {
				s.logger.Warn("failed to get status", "error", err)
				continue
			}

			s.logger.Debug("polling status", "status", status.Status, "progress", status.Progress)

			// Send progress update
			progressPercent := int(status.Progress * 100)
			if progressPercent < 60 {
				progressPercent = 60
			}
			if progressPercent > 95 && status.Status != "completed" {
				progressPercent = 95
			}

			s.sendEvent(eventCh, streaming.StreamEvent{
				Type:    "progress",
				Content: status.CurrentStep,
				Data: map[string]interface{}{
					"percent": progressPercent,
					"phase":   status.Status,
					"step":    status.CurrentStep,
				},
			})

			// Check completion
			if status.Status == "completed" {
				s.logger.Info("generation completed", "app_id", appID)
				return nil
			}

			if status.Status == "failed" {
				return fmt.Errorf("generation failed: %s", status.Error)
			}
		}
	}
}

// ensureWorkspace gets existing workspace or creates a default one
func (s *OSAAppService) ensureWorkspace(ctx context.Context, userID uuid.UUID, workspaceID *uuid.UUID) (uuid.UUID, error) {
	if workspaceID != nil {
		// Verify workspace exists and belongs to user
		pgWorkspaceID := pgtype.UUID{Bytes: *workspaceID, Valid: true}
		ws, err := s.queries.GetOSAWorkspace(ctx, pgWorkspaceID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("workspace not found: %w", err)
		}
		// Compare UUIDs - ws.UserID is pgtype.UUID
		if ws.UserID.Bytes != userID {
			return uuid.Nil, fmt.Errorf("workspace does not belong to user")
		}
		return *workspaceID, nil
	}

	// Create default workspace
	mode := "2d"
	templateType := "default"
	ws, err := s.queries.CreateOSAWorkspace(ctx, sqlc.CreateOSAWorkspaceParams{
		UserID:       pgtype.UUID{Bytes: userID, Valid: true},
		Name:         "Default Workspace",
		Mode:         &mode,
		Layout:       nil,
		TemplateType: &templateType,
		Settings:     nil,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	s.logger.Info("created default workspace", "workspace_id", ws.ID, "user_id", userID)
	// Convert pgtype.UUID back to uuid.UUID
	return ws.ID.Bytes, nil
}

// createAppRecord creates the database record for the app
func (s *OSAAppService) createAppRecord(ctx context.Context, req *GenerateAppRequest, workspaceID uuid.UUID) (uuid.UUID, error) {
	description := req.Description
	status := "generating"
	app, err := s.queries.CreateOSAModuleInstance(ctx, sqlc.CreateOSAModuleInstanceParams{
		WorkspaceID: pgtype.UUID{Bytes: workspaceID, Valid: true},
		Name:        req.Name,
		DisplayName: req.Name,
		Description: &description,
		Status:      &status,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create app record: %w", err)
	}

	return app.ID.Bytes, nil
}

// updateAppStatus updates the status of an app
func (s *OSAAppService) updateAppStatus(ctx context.Context, appID uuid.UUID, status string) error {
	_, err := s.queries.UpdateOSAModuleInstanceStatus(ctx, sqlc.UpdateOSAModuleInstanceStatusParams{
		ID:     pgtype.UUID{Bytes: appID, Valid: true},
		Status: &status,
	})
	return err
}

// sendEvent safely sends an event to the channel
func (s *OSAAppService) sendEvent(eventCh chan<- streaming.StreamEvent, event streaming.StreamEvent) {
	defer func() {
		if r := recover(); r != nil {
			if s.logger != nil {
				s.logger.Warn("recovered from panic while sending event", "panic", r)
			}
		}
	}()

	select {
	case eventCh <- event:
	default:
		if s.logger != nil {
			s.logger.Warn("event channel full, dropping event")
		}
	}
}

// GetAppStatus retrieves the current status of an app
func (s *OSAAppService) GetAppStatus(ctx context.Context, appID uuid.UUID) (*sqlc.OsaModuleInstance, error) {
	app, err := s.queries.GetOSAModuleInstance(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}
	return &app, nil
}

// ListUserApps lists all apps for a user
func (s *OSAAppService) ListUserApps(ctx context.Context, userID uuid.UUID) ([]sqlc.ListOSAModuleInstancesByUserRow, error) {
	apps, err := s.queries.ListOSAModuleInstancesByUser(ctx, sqlc.ListOSAModuleInstancesByUserParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Column2: pgtype.UUID{}, // No workspace filter
		Column3: "",            // No status filter
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}
	return apps, nil
}
