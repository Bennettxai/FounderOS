// Package services provides business logic services for the application.
// sandbox_cleanup.go handles cleanup of orphaned containers and old resources.
package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/container"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Cleanup configuration defaults
const (
	defaultCleanupInterval    = 1 * time.Hour
	defaultStoppedGracePeriod = 24 * time.Hour
	defaultEventRetention     = 7 * 24 * time.Hour // 7 days
)

// CleanupConfig holds configuration for the cleanup service.
type CleanupConfig struct {
	CleanupInterval    time.Duration // How often to run cleanup
	StoppedGracePeriod time.Duration // Time to wait before removing stopped containers
	EventRetention     time.Duration // How long to keep sandbox events
	DryRun             bool          // If true, only log what would be cleaned up
}

// DefaultCleanupConfig returns the default cleanup configuration.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		CleanupInterval:    defaultCleanupInterval,
		StoppedGracePeriod: defaultStoppedGracePeriod,
		EventRetention:     defaultEventRetention,
		DryRun:             false,
	}
}

// CleanupResult holds the results of a cleanup operation.
type CleanupResult struct {
	OrphanedContainersRemoved int
	StoppedContainersRemoved  int
	OldEventsDeleted          int
	PortsReleased             int
	Errors                    []error
	StartedAt                 time.Time
	Duration                  time.Duration
}

// SandboxCleanupService handles cleanup of orphaned and stopped containers.
type SandboxCleanupService struct {
	containerManager *container.AppContainerManager
	portAllocator    *SandboxPortAllocator
	pool             *pgxpool.Pool
	queries          *sqlc.Queries
	logger           *slog.Logger
	config           CleanupConfig

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}

	// Stats
	lastCleanup       time.Time
	lastCleanupResult *CleanupResult
}

// NewSandboxCleanupService creates a new cleanup service.
func NewSandboxCleanupService(
	containerManager *container.AppContainerManager,
	portAllocator *SandboxPortAllocator,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	config CleanupConfig,
) *SandboxCleanupService {
	if config.CleanupInterval == 0 {
		config.CleanupInterval = defaultCleanupInterval
	}
	if config.StoppedGracePeriod == 0 {
		config.StoppedGracePeriod = defaultStoppedGracePeriod
	}
	if config.EventRetention == 0 {
		config.EventRetention = defaultEventRetention
	}

	return &SandboxCleanupService{
		containerManager: containerManager,
		portAllocator:    portAllocator,
		pool:             pool,
		queries:          sqlc.New(pool),
		logger:           logger.With("service", "cleanup"),
		config:           config,
	}
}

// Start begins the cleanup loop.
func (s *SandboxCleanupService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.logger.Info("starting cleanup service",
		"interval", s.config.CleanupInterval,
		"grace_period", s.config.StoppedGracePeriod,
		"dry_run", s.config.DryRun)

	go s.cleanupLoop(ctx)
	return nil
}

// Stop halts the cleanup loop.
func (s *SandboxCleanupService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopCh)
	s.logger.Info("cleanup service stopped")
}

// IsRunning returns whether the cleanup service is running.
func (s *SandboxCleanupService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// RunCleanup performs an immediate cleanup.
func (s *SandboxCleanupService) RunCleanup(ctx context.Context) *CleanupResult {
	result := &CleanupResult{
		StartedAt: time.Now().UTC(),
		Errors:    make([]error, 0),
	}

	s.logger.Info("starting cleanup run", "dry_run", s.config.DryRun)

	// 1. Clean up orphaned containers
	orphaned := s.cleanupOrphanedContainers(ctx, result)
	result.OrphanedContainersRemoved = orphaned

	// 2. Clean up stopped containers past grace period
	stopped := s.cleanupStoppedContainers(ctx, result)
	result.StoppedContainersRemoved = stopped

	// 3. Clean up old events
	events := s.cleanupOldEvents(ctx, result)
	result.OldEventsDeleted = events

	// 4. Clean up orphaned port allocations
	ports := s.cleanupOrphanedPorts(ctx, result)
	result.PortsReleased = ports

	result.Duration = time.Since(result.StartedAt)

	s.mu.Lock()
	s.lastCleanup = time.Now()
	s.lastCleanupResult = result
	s.mu.Unlock()

	s.logger.Info("cleanup run completed",
		"duration", result.Duration,
		"orphaned_removed", result.OrphanedContainersRemoved,
		"stopped_removed", result.StoppedContainersRemoved,
		"events_deleted", result.OldEventsDeleted,
		"ports_released", result.PortsReleased,
		"errors", len(result.Errors))

	return result
}

// GetStats returns cleanup service statistics.
func (s *SandboxCleanupService) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := map[string]interface{}{
		"running":          s.running,
		"cleanup_interval": s.config.CleanupInterval.String(),
		"grace_period":     s.config.StoppedGracePeriod.String(),
		"event_retention":  s.config.EventRetention.String(),
		"dry_run":          s.config.DryRun,
	}

	if !s.lastCleanup.IsZero() {
		stats["last_cleanup"] = s.lastCleanup.Format(time.RFC3339)
		stats["since_last_cleanup"] = time.Since(s.lastCleanup).String()
	}

	if s.lastCleanupResult != nil {
		stats["last_orphaned_removed"] = s.lastCleanupResult.OrphanedContainersRemoved
		stats["last_stopped_removed"] = s.lastCleanupResult.StoppedContainersRemoved
		stats["last_events_deleted"] = s.lastCleanupResult.OldEventsDeleted
		stats["last_ports_released"] = s.lastCleanupResult.PortsReleased
		stats["last_duration"] = s.lastCleanupResult.Duration.String()
		stats["last_errors"] = len(s.lastCleanupResult.Errors)
	}

	return stats
}

// cleanupLoop runs periodic cleanup.
func (s *SandboxCleanupService) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("cleanup loop context cancelled")
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.RunCleanup(ctx)
		}
	}
}

// cleanupOrphanedContainers removes containers without matching DB entries.
func (s *SandboxCleanupService) cleanupOrphanedContainers(ctx context.Context, result *CleanupResult) int {
	if s.containerManager == nil {
		return 0
	}

	// List all BusinessOS app containers
	containers, err := s.containerManager.ListAppContainers(ctx)
	if err != nil {
		s.logger.Warn("failed to list containers for cleanup", "error", err)
		result.Errors = append(result.Errors, err)
		return 0
	}

	removed := 0
	for _, c := range containers {
		// Check if container has matching DB entry
		if s.pool != nil && c.AppID != uuid.Nil {
			pgAppID := pgtype.UUID{Bytes: c.AppID, Valid: true}
			_, err := s.queries.GetAppSandboxInfo(ctx, pgAppID)
			if err == nil {
				// Container has DB entry, skip
				continue
			}
		}

		s.logger.Info("found orphaned container",
			"container_id", c.ContainerID[:12],
			"app_id", c.AppID,
			"status", c.Status)

		if s.config.DryRun {
			s.logger.Info("[DRY RUN] would remove orphaned container",
				"container_id", c.ContainerID[:12])
			continue
		}

		// Remove the container
		if err := s.containerManager.RemoveAppContainer(ctx, c.ContainerID, true); err != nil {
			s.logger.Warn("failed to remove orphaned container",
				"container_id", c.ContainerID[:12],
				"error", err)
			result.Errors = append(result.Errors, err)
			continue
		}

		s.logger.Info("removed orphaned container",
			"container_id", c.ContainerID[:12])
		removed++
	}

	return removed
}

// cleanupStoppedContainers removes stopped containers past the grace period.
func (s *SandboxCleanupService) cleanupStoppedContainers(ctx context.Context, result *CleanupResult) int {
	if s.pool == nil {
		return 0
	}

	// Get stopped sandboxes from DB
	sandboxes, err := s.queries.ListStoppedSandboxes(ctx)
	if err != nil {
		s.logger.Warn("failed to list stopped sandboxes", "error", err)
		result.Errors = append(result.Errors, err)
		return 0
	}

	removed := 0
	gracePeriodCutoff := time.Now().UTC().Add(-s.config.StoppedGracePeriod)

	for _, sandbox := range sandboxes {
		if !sandbox.ID.Valid || sandbox.ContainerID == nil {
			continue
		}

		// Check if past grace period
		if sandbox.UpdatedAt.Valid && sandbox.UpdatedAt.Time.After(gracePeriodCutoff) {
			// Not yet past grace period
			continue
		}

		appID := uuid.UUID(sandbox.ID.Bytes)
		containerID := *sandbox.ContainerID

		s.logger.Info("found stopped container past grace period",
			"app_id", appID,
			"container_id", containerID[:12])

		if s.config.DryRun {
			s.logger.Info("[DRY RUN] would remove stopped container",
				"app_id", appID,
				"container_id", containerID[:12])
			continue
		}

		// Remove the container if it exists
		if s.containerManager != nil {
			if err := s.containerManager.RemoveAppContainer(ctx, containerID, true); err != nil {
				s.logger.Warn("failed to remove stopped container",
					"container_id", containerID[:12],
					"error", err)
				// Continue to clear DB entry even if container removal fails
			}
		}

		// Clear sandbox info in DB
		pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
		if err := s.queries.ClearSandboxInfo(ctx, pgAppID); err != nil {
			s.logger.Warn("failed to clear sandbox info",
				"app_id", appID,
				"error", err)
			result.Errors = append(result.Errors, err)
			continue
		}

		// Release port
		if s.portAllocator != nil {
			if err := s.portAllocator.Release(ctx, appID); err != nil {
				s.logger.Warn("failed to release port for stopped container",
					"app_id", appID,
					"error", err)
			}
		}

		s.logger.Info("removed stopped container",
			"app_id", appID)
		removed++
	}

	return removed
}

// cleanupOldEvents deletes sandbox events older than retention period.
func (s *SandboxCleanupService) cleanupOldEvents(ctx context.Context, result *CleanupResult) int {
	if s.pool == nil {
		return 0
	}

	if s.config.DryRun {
		s.logger.Info("[DRY RUN] would delete old sandbox events",
			"older_than", s.config.EventRetention)
		return 0
	}

	// Convert duration to interval for PostgreSQL
	interval := pgtype.Interval{
		Microseconds: s.config.EventRetention.Microseconds(),
		Valid:        true,
	}

	if err := s.queries.DeleteOldSandboxEvents(ctx, interval); err != nil {
		s.logger.Warn("failed to delete old sandbox events", "error", err)
		result.Errors = append(result.Errors, err)
		return 0
	}

	// Note: We can't easily get the count of deleted rows without a custom query
	s.logger.Info("deleted old sandbox events",
		"older_than", s.config.EventRetention)
	return 1 // Indicate cleanup was performed
}

// cleanupOrphanedPorts releases ports that are allocated but have no matching container.
func (s *SandboxCleanupService) cleanupOrphanedPorts(ctx context.Context, result *CleanupResult) int {
	if s.portAllocator == nil || s.pool == nil {
		return 0
	}

	// Get port allocator stats to find allocated apps
	stats := s.portAllocator.GetStats()
	allocatedCount, ok := stats["allocated"].(int)
	if !ok || allocatedCount == 0 {
		return 0
	}

	// This is a simplified implementation
	// A more complete implementation would iterate through allocated ports
	// and verify each has a valid running container

	// For now, we rely on the container and stopped cleanup to handle this
	return 0
}

// CleanupFailedDeployment cleans up resources from a failed deployment.
func (s *SandboxCleanupService) CleanupFailedDeployment(ctx context.Context, appID uuid.UUID, containerID string) error {
	s.logger.Info("cleaning up failed deployment",
		"app_id", appID,
		"container_id", containerID)

	// Remove container if exists
	if s.containerManager != nil && containerID != "" {
		if err := s.containerManager.RemoveAppContainer(ctx, containerID, true); err != nil {
			s.logger.Warn("failed to remove container from failed deployment",
				"container_id", containerID,
				"error", err)
		}
	}

	// Release port
	if s.portAllocator != nil {
		if err := s.portAllocator.Release(ctx, appID); err != nil {
			s.logger.Warn("failed to release port from failed deployment",
				"app_id", appID,
				"error", err)
		}
	}

	// Update DB status
	if s.pool != nil {
		pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
		if err := s.queries.ClearSandboxInfo(ctx, pgAppID); err != nil {
			s.logger.Warn("failed to clear sandbox info after failed deployment",
				"app_id", appID,
				"error", err)
			return err
		}
	}

	// Log cleanup event
	s.logCleanupEvent(ctx, appID, "failed_deployment_cleanup", "cleaned up resources from failed deployment")

	return nil
}

// logCleanupEvent logs a cleanup-related event.
func (s *SandboxCleanupService) logCleanupEvent(ctx context.Context, appID uuid.UUID, eventType string, message string) {
	if s.pool == nil {
		return
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	details, _ := encodeJSON(map[string]interface{}{
		"message": message,
	})

	_, err := s.queries.InsertSandboxEvent(ctx, sqlc.InsertSandboxEventParams{
		ModuleInstanceID: pgAppID,
		EventType:        eventType,
		Details:          details,
	})
	if err != nil {
		s.logger.Warn("failed to log cleanup event",
			"app_id", appID,
			"event_type", eventType,
			"error", err)
	}
}
