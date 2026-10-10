// Package services provides business logic services for the application.
// sandbox_deployment.go orchestrates sandbox container deployment.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/container"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Sandbox deployment errors
var (
	ErrSandboxNotFound       = errors.New("sandbox not found")
	ErrSandboxAlreadyRunning = errors.New("sandbox is already running")
	ErrSandboxNotRunning     = errors.New("sandbox is not running")
	ErrMaxSandboxesReached   = errors.New("maximum number of sandboxes reached for user")
	ErrDeploymentFailed      = errors.New("sandbox deployment failed")
	ErrInvalidAppID          = errors.New("invalid app ID")
)

// SandboxStatus represents the current status of a sandbox.
type SandboxStatus string

const (
	SandboxStatusPending  SandboxStatus = "pending"
	SandboxStatusBuilding SandboxStatus = "building"
	SandboxStatusRunning  SandboxStatus = "running"
	SandboxStatusStopped  SandboxStatus = "stopped"
	SandboxStatusError    SandboxStatus = "error"
)

// SandboxInfo holds information about a deployed sandbox.
type SandboxInfo struct {
	AppID        uuid.UUID
	AppName      string
	UserID       uuid.UUID
	ContainerID  string
	Status       SandboxStatus
	Port         int
	URL          string
	Image        string
	AppType      string // App type (svelte, react, etc.) for image auto-detection
	CreatedAt    time.Time
	StartedAt    *time.Time
	HealthStatus string
	ErrorMessage string
}

// SandboxDeploymentRequest contains the configuration for deploying a sandbox.
type SandboxDeploymentRequest struct {
	AppID         uuid.UUID         `json:"app_id"`
	AppName       string            `json:"app_name"`
	UserID        uuid.UUID         `json:"user_id,omitempty"`
	Image         string            `json:"image"`          // Docker image to use
	ContainerPort int               `json:"container_port"` // Port inside container (e.g., 3000)
	WorkspacePath string            `json:"workspace_path"` // Path to workspace on host
	Environment   map[string]string `json:"environment"`    // Environment variables
	StartCommand  []string          `json:"start_command"`  // Command to start the app
	WorkingDir    string            `json:"working_dir"`    // Working directory inside container
	MemoryLimit   int64             `json:"memory_limit"`   // Optional memory limit
	CPUQuota      int64             `json:"cpu_quota"`      // Optional CPU quota
}

// SandboxDeploymentService orchestrates sandbox container deployment.
type SandboxDeploymentService struct {
	portAllocator    *SandboxPortAllocator
	containerManager *container.AppContainerManager
	pool             *pgxpool.Pool
	queries          *sqlc.Queries
	config           *config.Config
	logger           *slog.Logger
	mu               sync.Mutex

	// Track in-progress deployments to prevent duplicates
	inProgress map[uuid.UUID]bool
}

// NewSandboxDeploymentService creates a new sandbox deployment service.
func NewSandboxDeploymentService(
	pool *pgxpool.Pool,
	dockerClient *client.Client,
	cfg *config.Config,
	logger *slog.Logger,
) (*SandboxDeploymentService, error) {
	// Create port allocator
	portAllocator, err := NewSandboxPortAllocator(nil, pool, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create port allocator: %w", err)
	}

	// Create app container manager
	containerManager := container.NewAppContainerManager(dockerClient, logger, "")

	service := &SandboxDeploymentService{
		portAllocator:    portAllocator,
		containerManager: containerManager,
		pool:             pool,
		queries:          sqlc.New(pool),
		config:           cfg,
		logger:           logger.With("service", "sandbox_deployment"),
		inProgress:       make(map[uuid.UUID]bool),
	}

	return service, nil
}

// Deploy creates and starts a sandbox container for an app.
func (s *SandboxDeploymentService) Deploy(ctx context.Context, req SandboxDeploymentRequest) (*SandboxInfo, error) {
	if req.AppID == uuid.Nil {
		return nil, ErrInvalidAppID
	}

	s.mu.Lock()
	// Check if deployment is already in progress
	if s.inProgress[req.AppID] {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: deployment already in progress", ErrSandboxAlreadyRunning)
	}
	s.inProgress[req.AppID] = true
	s.mu.Unlock()

	// Clean up in-progress flag when done
	defer func() {
		s.mu.Lock()
		delete(s.inProgress, req.AppID)
		s.mu.Unlock()
	}()

	s.logger.Info("starting sandbox deployment",
		"app_id", req.AppID,
		"app_name", req.AppName,
		"user_id", req.UserID,
		"image", req.Image)

	// Check user quota
	if err := s.checkUserQuota(ctx, req.UserID); err != nil {
		return nil, err
	}

	// Check if sandbox already exists
	existing, err := s.GetSandboxInfo(ctx, req.AppID)
	if err == nil && existing.Status == SandboxStatusRunning {
		return nil, ErrSandboxAlreadyRunning
	}

	// Allocate a port
	port, err := s.portAllocator.Allocate(ctx, req.AppID)
	if err != nil {
		s.logger.Error("failed to allocate port", "app_id", req.AppID, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrDeploymentFailed, err)
	}

	// Update database with pending status
	if err := s.updateSandboxStatus(ctx, req.AppID, SandboxStatusBuilding, port, "", ""); err != nil {
		s.logger.Warn("failed to update sandbox status", "app_id", req.AppID, "error", err)
	}

	// Create container configuration
	containerConfig := container.AppContainerConfig{
		AppID:         req.AppID,
		AppName:       req.AppName,
		UserID:        req.UserID,
		Image:         req.Image,
		WorkspacePath: req.WorkspacePath,
		ContainerPort: req.ContainerPort,
		HostPort:      port,
		Environment:   req.Environment,
		StartCommand:  req.StartCommand,
		WorkingDir:    req.WorkingDir,
		MemoryLimit:   req.MemoryLimit,
		CPUQuota:      req.CPUQuota,
	}

	// Set default container port if not specified
	if containerConfig.ContainerPort == 0 {
		containerConfig.ContainerPort = 3000
	}

	// Create the container
	containerInfo, err := s.containerManager.CreateAppContainer(ctx, containerConfig)
	if err != nil {
		s.logger.Error("failed to create container", "app_id", req.AppID, "error", err)
		s.portAllocator.Release(ctx, req.AppID)
		s.updateSandboxStatus(ctx, req.AppID, SandboxStatusError, 0, "", err.Error())
		return nil, fmt.Errorf("%w: failed to create container: %v", ErrDeploymentFailed, err)
	}

	// Start the container
	if err := s.containerManager.StartAppContainer(ctx, containerInfo.ContainerID); err != nil {
		s.logger.Error("failed to start container", "app_id", req.AppID, "container_id", containerInfo.ContainerID, "error", err)
		// Cleanup: remove the created container
		s.containerManager.RemoveAppContainer(ctx, containerInfo.ContainerID, true)
		s.portAllocator.Release(ctx, req.AppID)
		s.updateSandboxStatus(ctx, req.AppID, SandboxStatusError, 0, "", err.Error())
		return nil, fmt.Errorf("%w: failed to start container: %v", ErrDeploymentFailed, err)
	}

	// Build sandbox URL
	sandboxURL := fmt.Sprintf("http://localhost:%d", port)

	// Update database with running status
	if err := s.updateSandboxStatus(ctx, req.AppID, SandboxStatusRunning, port, containerInfo.ContainerID, sandboxURL); err != nil {
		s.logger.Warn("failed to update sandbox status in database", "app_id", req.AppID, "error", err)
	}

	// Log deployment event
	s.logDeploymentEvent(ctx, req.AppID, "deployed", map[string]interface{}{
		"container_id": containerInfo.ContainerID,
		"port":         port,
		"image":        req.Image,
	})

	now := time.Now().UTC()
	return &SandboxInfo{
		AppID:       req.AppID,
		AppName:     req.AppName,
		UserID:      req.UserID,
		ContainerID: containerInfo.ContainerID,
		Status:      SandboxStatusRunning,
		Port:        port,
		URL:         sandboxURL,
		Image:       req.Image,
		CreatedAt:   containerInfo.CreatedAt,
		StartedAt:   &now,
	}, nil
}

// Stop stops a running sandbox container.
func (s *SandboxDeploymentService) Stop(ctx context.Context, appID uuid.UUID) error {
	info, err := s.GetSandboxInfo(ctx, appID)
	if err != nil {
		return err
	}

	if info.ContainerID == "" {
		return ErrSandboxNotRunning
	}

	s.logger.Info("stopping sandbox", "app_id", appID, "container_id", info.ContainerID)

	// Stop the container (10 second timeout)
	timeout := 10
	if err := s.containerManager.StopAppContainer(ctx, info.ContainerID, &timeout); err != nil {
		s.logger.Error("failed to stop container", "app_id", appID, "error", err)
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}

	// Update database
	if err := s.updateSandboxStatus(ctx, appID, SandboxStatusStopped, info.Port, info.ContainerID, info.URL); err != nil {
		s.logger.Warn("failed to update sandbox status", "app_id", appID, "error", err)
	}

	// Log event
	s.logDeploymentEvent(ctx, appID, "stopped", nil)

	return nil
}

// Restart restarts a sandbox container.
func (s *SandboxDeploymentService) Restart(ctx context.Context, appID uuid.UUID) (*SandboxInfo, error) {
	info, err := s.GetSandboxInfo(ctx, appID)
	if err != nil {
		return nil, err
	}

	if info.ContainerID == "" {
		return nil, ErrSandboxNotFound
	}

	s.logger.Info("restarting sandbox", "app_id", appID, "container_id", info.ContainerID)

	// Stop if running
	if info.Status == SandboxStatusRunning {
		timeout := 10
		s.containerManager.StopAppContainer(ctx, info.ContainerID, &timeout)
	}

	// Start the container
	if err := s.containerManager.StartAppContainer(ctx, info.ContainerID); err != nil {
		s.logger.Error("failed to restart container", "app_id", appID, "error", err)
		s.updateSandboxStatus(ctx, appID, SandboxStatusError, info.Port, info.ContainerID, err.Error())
		return nil, fmt.Errorf("failed to restart sandbox: %w", err)
	}

	// Update status
	s.updateSandboxStatus(ctx, appID, SandboxStatusRunning, info.Port, info.ContainerID, info.URL)

	// Log event
	s.logDeploymentEvent(ctx, appID, "restarted", nil)

	info.Status = SandboxStatusRunning
	now := time.Now().UTC()
	info.StartedAt = &now

	return info, nil
}

// Remove removes a sandbox container and releases resources.
func (s *SandboxDeploymentService) Remove(ctx context.Context, appID uuid.UUID) error {
	info, err := s.GetSandboxInfo(ctx, appID)
	if err != nil && !errors.Is(err, ErrSandboxNotFound) {
		return err
	}

	s.logger.Info("removing sandbox", "app_id", appID)

	// Remove container if exists
	if info != nil && info.ContainerID != "" {
		if err := s.containerManager.RemoveAppContainer(ctx, info.ContainerID, true); err != nil {
			s.logger.Warn("failed to remove container", "app_id", appID, "error", err)
		}
	}

	// Release port
	if err := s.portAllocator.Release(ctx, appID); err != nil {
		s.logger.Warn("failed to release port", "app_id", appID, "error", err)
	}

	// Clear sandbox info in database
	if err := s.clearSandboxInfo(ctx, appID); err != nil {
		s.logger.Warn("failed to clear sandbox info", "app_id", appID, "error", err)
	}

	// Log event
	s.logDeploymentEvent(ctx, appID, "removed", nil)

	return nil
}

// GetSandboxInfo retrieves information about a sandbox.
func (s *SandboxDeploymentService) GetSandboxInfo(ctx context.Context, appID uuid.UUID) (*SandboxInfo, error) {
	if s.pool == nil {
		return nil, ErrSandboxNotFound
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	dbInfo, err := s.queries.GetAppSandboxInfo(ctx, pgAppID)
	if err != nil {
		return nil, ErrSandboxNotFound
	}

	info := &SandboxInfo{
		AppID:   appID,
		AppName: dbInfo.Name, // Name is string, not *string
	}

	// Parse nullable database fields
	if dbInfo.ContainerID != nil {
		info.ContainerID = *dbInfo.ContainerID
	}
	if dbInfo.SandboxStatus != nil {
		info.Status = SandboxStatus(*dbInfo.SandboxStatus)
	}
	if dbInfo.SandboxPort != nil {
		info.Port = int(*dbInfo.SandboxPort)
	}
	if dbInfo.SandboxUrl != nil {
		info.URL = *dbInfo.SandboxUrl
	}
	if dbInfo.ContainerImage != nil {
		info.Image = *dbInfo.ContainerImage
	}
	if dbInfo.AppType != nil {
		info.AppType = *dbInfo.AppType
	}
	if dbInfo.HealthStatus != nil {
		info.HealthStatus = *dbInfo.HealthStatus
	}

	// Get live container info if container exists
	if info.ContainerID != "" {
		liveInfo, err := s.containerManager.GetAppContainerInfo(ctx, info.ContainerID)
		if err == nil {
			info.Status = s.mapContainerStatus(liveInfo.Status)
			info.HealthStatus = liveInfo.HealthStatus
			info.CreatedAt = liveInfo.CreatedAt
			if !liveInfo.StartedAt.IsZero() {
				info.StartedAt = &liveInfo.StartedAt
			}
		}
	}

	return info, nil
}

// GetSandboxLogs retrieves logs from a sandbox container.
func (s *SandboxDeploymentService) GetSandboxLogs(ctx context.Context, appID uuid.UUID, tail string, since string) (string, error) {
	info, err := s.GetSandboxInfo(ctx, appID)
	if err != nil {
		return "", err
	}

	if info.ContainerID == "" {
		return "", ErrSandboxNotRunning
	}

	return s.containerManager.GetAppContainerLogs(ctx, info.ContainerID, tail, since)
}

// ListUserSandboxes lists all sandboxes for a user.
func (s *SandboxDeploymentService) ListUserSandboxes(ctx context.Context, userID uuid.UUID) ([]*SandboxInfo, error) {
	if s.pool == nil {
		return []*SandboxInfo{}, nil
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	dbSandboxes, err := s.queries.ListUserSandboxes(ctx, pgUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sandboxes: %w", err)
	}

	sandboxes := make([]*SandboxInfo, 0, len(dbSandboxes))
	for _, db := range dbSandboxes {
		if !db.ID.Valid {
			continue
		}
		info := &SandboxInfo{
			AppID:   uuid.UUID(db.ID.Bytes),
			UserID:  userID,
			AppName: db.Name, // Name is string, not *string
		}
		if db.ContainerID != nil {
			info.ContainerID = *db.ContainerID
		}
		if db.SandboxStatus != nil {
			info.Status = SandboxStatus(*db.SandboxStatus)
		}
		if db.SandboxPort != nil {
			info.Port = int(*db.SandboxPort)
		}
		if db.SandboxUrl != nil {
			info.URL = *db.SandboxUrl
		}
		sandboxes = append(sandboxes, info)
	}

	return sandboxes, nil
}

// GetStats returns deployment service statistics.
func (s *SandboxDeploymentService) GetStats() map[string]interface{} {
	s.mu.Lock()
	inProgressCount := len(s.inProgress)
	s.mu.Unlock()

	stats := map[string]interface{}{
		"in_progress_deployments": inProgressCount,
	}

	// Add port allocator stats
	portStats := s.portAllocator.GetStats()
	for k, v := range portStats {
		stats["port_"+k] = v
	}

	return stats
}

// checkUserQuota verifies the user hasn't exceeded their sandbox limit.
func (s *SandboxDeploymentService) checkUserQuota(ctx context.Context, userID uuid.UUID) error {
	if s.pool == nil || s.config.SandboxMaxPerUser == 0 {
		return nil
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	count, err := s.queries.CountUserRunningSandboxes(ctx, pgUserID)
	if err != nil {
		s.logger.Warn("failed to count user sandboxes", "user_id", userID, "error", err)
		return nil // Don't block on quota check failure
	}

	if int(count) >= s.config.SandboxMaxPerUser {
		return ErrMaxSandboxesReached
	}

	return nil
}

// updateSandboxStatus updates the sandbox status in the database.
func (s *SandboxDeploymentService) updateSandboxStatus(ctx context.Context, appID uuid.UUID, status SandboxStatus, port int, containerID string, urlOrError string) error {
	if s.pool == nil {
		return nil
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	statusStr := string(status)

	var portPtr *int32
	if port > 0 {
		port32 := int32(port)
		portPtr = &port32
	}

	var containerIDPtr *string
	if containerID != "" {
		containerIDPtr = &containerID
	}

	var urlPtr *string
	if urlOrError != "" && status != SandboxStatusError {
		urlPtr = &urlOrError
	}

	_, err := s.queries.UpdateAppSandboxInfo(ctx, sqlc.UpdateAppSandboxInfoParams{
		ID:            pgAppID,
		ContainerID:   containerIDPtr,
		SandboxPort:   portPtr,
		SandboxUrl:    urlPtr,
		SandboxStatus: &statusStr,
	})
	return err
}

// clearSandboxInfo clears sandbox information from the database.
func (s *SandboxDeploymentService) clearSandboxInfo(ctx context.Context, appID uuid.UUID) error {
	if s.pool == nil {
		return nil
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	return s.queries.ClearSandboxInfo(ctx, pgAppID)
}

// logDeploymentEvent logs a deployment event to the database.
func (s *SandboxDeploymentService) logDeploymentEvent(ctx context.Context, appID uuid.UUID, eventType string, details map[string]interface{}) {
	if s.pool == nil {
		return
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	detailsJSON, _ := encodeJSON(details)

	_, err := s.queries.InsertSandboxEvent(ctx, sqlc.InsertSandboxEventParams{
		ModuleInstanceID: pgAppID,
		EventType:        eventType,
		Details:          detailsJSON,
	})
	if err != nil {
		s.logger.Warn("failed to log deployment event",
			"app_id", appID,
			"event_type", eventType,
			"error", err)
	}
}

// mapContainerStatus maps Docker container status to sandbox status.
func (s *SandboxDeploymentService) mapContainerStatus(dockerStatus string) SandboxStatus {
	switch dockerStatus {
	case "running":
		return SandboxStatusRunning
	case "exited", "dead":
		return SandboxStatusStopped
	case "created", "restarting":
		return SandboxStatusBuilding
	default:
		return SandboxStatusPending
	}
}

// RecoverSandboxes recovers sandbox state from the database on startup.
func (s *SandboxDeploymentService) RecoverSandboxes(ctx context.Context) error {
	// Recover port allocations
	if err := s.portAllocator.RecoverFromDB(ctx); err != nil {
		return fmt.Errorf("failed to recover port allocations: %w", err)
	}

	s.logger.Info("sandbox deployment service recovered")
	return nil
}
