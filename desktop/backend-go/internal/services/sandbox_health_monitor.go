// Package services provides business logic services for the application.
// sandbox_health_monitor.go provides periodic health monitoring for sandbox containers.
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

// HealthStatus represents the health state of a container.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusStarting  HealthStatus = "starting"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// Health check configuration
const (
	defaultCheckInterval    = 30 * time.Second
	defaultUnhealthyTimeout = 3 * time.Minute
	defaultMaxRetries       = 3
)

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	AppID        uuid.UUID
	ContainerID  string
	Status       HealthStatus
	ResponseTime time.Duration
	CheckedAt    time.Time
	Message      string
}

// HealthMonitorConfig holds configuration for the health monitor.
type HealthMonitorConfig struct {
	CheckInterval    time.Duration
	UnhealthyTimeout time.Duration
	AutoRestart      bool
	MaxRetries       int
}

// DefaultHealthMonitorConfig returns the default health monitor configuration.
func DefaultHealthMonitorConfig() HealthMonitorConfig {
	return HealthMonitorConfig{
		CheckInterval:    defaultCheckInterval,
		UnhealthyTimeout: defaultUnhealthyTimeout,
		AutoRestart:      false,
		MaxRetries:       defaultMaxRetries,
	}
}

// SandboxHealthMonitor monitors the health of sandbox containers.
type SandboxHealthMonitor struct {
	containerManager *container.AppContainerManager
	pool             *pgxpool.Pool
	queries          *sqlc.Queries
	logger           *slog.Logger
	config           HealthMonitorConfig

	// State tracking
	mu              sync.RWMutex
	running         bool
	stopCh          chan struct{}
	unhealthyCounts map[uuid.UUID]int // Track consecutive unhealthy checks
	lastCheck       map[uuid.UUID]time.Time

	// Callbacks
	onHealthChange func(result HealthCheckResult)
}

// NewSandboxHealthMonitor creates a new health monitor.
func NewSandboxHealthMonitor(
	containerManager *container.AppContainerManager,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	config HealthMonitorConfig,
) *SandboxHealthMonitor {
	if config.CheckInterval == 0 {
		config.CheckInterval = defaultCheckInterval
	}
	if config.UnhealthyTimeout == 0 {
		config.UnhealthyTimeout = defaultUnhealthyTimeout
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}

	return &SandboxHealthMonitor{
		containerManager: containerManager,
		pool:             pool,
		queries:          sqlc.New(pool),
		logger:           logger.With("service", "health_monitor"),
		config:           config,
		unhealthyCounts:  make(map[uuid.UUID]int),
		lastCheck:        make(map[uuid.UUID]time.Time),
	}
}

// SetHealthChangeCallback sets a callback to be called when health status changes.
func (m *SandboxHealthMonitor) SetHealthChangeCallback(callback func(HealthCheckResult)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onHealthChange = callback
}

// Start begins the health monitoring loop.
func (m *SandboxHealthMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	m.logger.Info("starting health monitor",
		"interval", m.config.CheckInterval,
		"auto_restart", m.config.AutoRestart)

	go m.monitorLoop(ctx)
	return nil
}

// Stop halts the health monitoring loop.
func (m *SandboxHealthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopCh)
	m.logger.Info("health monitor stopped")
}

// IsRunning returns whether the monitor is currently running.
func (m *SandboxHealthMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// CheckContainer performs an immediate health check on a specific container.
func (m *SandboxHealthMonitor) CheckContainer(ctx context.Context, appID uuid.UUID, containerID string) (*HealthCheckResult, error) {
	if m.containerManager == nil {
		return &HealthCheckResult{
			AppID:     appID,
			Status:    HealthStatusUnknown,
			CheckedAt: time.Now().UTC(),
			Message:   "container manager not available",
		}, nil
	}

	start := time.Now()

	// Get container info from Docker
	info, err := m.containerManager.GetAppContainerInfo(ctx, containerID)
	responseTime := time.Since(start)

	result := &HealthCheckResult{
		AppID:        appID,
		ContainerID:  containerID,
		ResponseTime: responseTime,
		CheckedAt:    time.Now().UTC(),
	}

	if err != nil {
		result.Status = HealthStatusUnknown
		result.Message = err.Error()
		return result, nil
	}

	// Map Docker health status
	switch info.HealthStatus {
	case "healthy":
		result.Status = HealthStatusHealthy
		result.Message = "container is healthy"
	case "unhealthy":
		result.Status = HealthStatusUnhealthy
		result.Message = "container health check failed"
	case "starting":
		result.Status = HealthStatusStarting
		result.Message = "container is starting"
	default:
		// Fall back to container state
		if info.Status == "running" {
			result.Status = HealthStatusHealthy
			result.Message = "container is running"
		} else {
			result.Status = HealthStatusUnhealthy
			result.Message = "container is not running: " + info.Status
		}
	}

	return result, nil
}

// GetHealthStats returns health monitoring statistics.
func (m *SandboxHealthMonitor) GetHealthStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unhealthyApps := make([]string, 0)
	for appID, count := range m.unhealthyCounts {
		if count > 0 {
			unhealthyApps = append(unhealthyApps, appID.String())
		}
	}

	return map[string]interface{}{
		"running":         m.running,
		"check_interval":  m.config.CheckInterval.String(),
		"auto_restart":    m.config.AutoRestart,
		"tracked_apps":    len(m.lastCheck),
		"unhealthy_apps":  unhealthyApps,
		"unhealthy_count": len(unhealthyApps),
	}
}

// monitorLoop runs the periodic health check loop.
func (m *SandboxHealthMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	// Run initial check
	m.runHealthChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("health monitor context cancelled")
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runHealthChecks(ctx)
		}
	}
}

// runHealthChecks checks all running sandboxes.
func (m *SandboxHealthMonitor) runHealthChecks(ctx context.Context) {
	if m.pool == nil {
		return
	}

	// Get all running sandboxes from database
	sandboxes, err := m.queries.ListRunningSandboxes(ctx)
	if err != nil {
		m.logger.Warn("failed to list running sandboxes", "error", err)
		return
	}

	m.logger.Debug("running health checks", "sandbox_count", len(sandboxes))

	for _, sandbox := range sandboxes {
		if !sandbox.ID.Valid || sandbox.ContainerID == nil {
			continue
		}

		appID := uuid.UUID(sandbox.ID.Bytes)
		containerID := *sandbox.ContainerID

		result, err := m.CheckContainer(ctx, appID, containerID)
		if err != nil {
			m.logger.Warn("health check failed",
				"app_id", appID,
				"error", err)
			continue
		}

		m.processHealthResult(ctx, result)
	}
}

// processHealthResult handles a health check result.
func (m *SandboxHealthMonitor) processHealthResult(ctx context.Context, result *HealthCheckResult) {
	m.mu.Lock()
	previousCount := m.unhealthyCounts[result.AppID]
	m.lastCheck[result.AppID] = result.CheckedAt

	var statusChanged bool
	switch result.Status {
	case HealthStatusHealthy:
		if previousCount > 0 {
			statusChanged = true
			m.logger.Info("sandbox recovered",
				"app_id", result.AppID,
				"previous_unhealthy_count", previousCount)
		}
		m.unhealthyCounts[result.AppID] = 0

	case HealthStatusUnhealthy:
		m.unhealthyCounts[result.AppID]++
		newCount := m.unhealthyCounts[result.AppID]
		if previousCount == 0 {
			statusChanged = true
		}
		m.logger.Warn("sandbox unhealthy",
			"app_id", result.AppID,
			"consecutive_failures", newCount,
			"message", result.Message)

	case HealthStatusStarting:
		// Don't count starting as unhealthy
		m.logger.Debug("sandbox starting", "app_id", result.AppID)
	}
	m.mu.Unlock()

	// Update database
	m.updateHealthInDB(ctx, result)

	// Notify callback if status changed
	if statusChanged && m.onHealthChange != nil {
		m.onHealthChange(*result)
	}

	// Check if auto-restart is needed
	m.mu.RLock()
	unhealthyCount := m.unhealthyCounts[result.AppID]
	m.mu.RUnlock()

	if m.config.AutoRestart && unhealthyCount >= m.config.MaxRetries {
		m.logger.Info("triggering auto-restart due to unhealthy status",
			"app_id", result.AppID,
			"unhealthy_count", unhealthyCount)
		m.triggerRestart(ctx, result.AppID, result.ContainerID)
	}
}

// updateHealthInDB updates the health status in the database.
func (m *SandboxHealthMonitor) updateHealthInDB(ctx context.Context, result *HealthCheckResult) {
	if m.pool == nil {
		return
	}

	pgAppID := pgtype.UUID{Bytes: result.AppID, Valid: true}
	statusStr := string(result.Status)

	_, err := m.queries.UpdateAppHealthStatus(ctx, sqlc.UpdateAppHealthStatusParams{
		ID:           pgAppID,
		HealthStatus: &statusStr,
	})
	if err != nil {
		m.logger.Warn("failed to update health status in database",
			"app_id", result.AppID,
			"error", err)
	}
}

// triggerRestart triggers a container restart.
func (m *SandboxHealthMonitor) triggerRestart(ctx context.Context, appID uuid.UUID, containerID string) {
	if m.containerManager == nil {
		return
	}

	// Stop the container
	timeout := 10
	if err := m.containerManager.StopAppContainer(ctx, containerID, &timeout); err != nil {
		m.logger.Warn("failed to stop unhealthy container",
			"app_id", appID,
			"container_id", containerID,
			"error", err)
	}

	// Start the container
	if err := m.containerManager.StartAppContainer(ctx, containerID); err != nil {
		m.logger.Error("failed to restart unhealthy container",
			"app_id", appID,
			"container_id", containerID,
			"error", err)
		return
	}

	// Reset unhealthy count
	m.mu.Lock()
	m.unhealthyCounts[appID] = 0
	m.mu.Unlock()

	m.logger.Info("successfully restarted unhealthy container",
		"app_id", appID,
		"container_id", containerID)

	// Log event
	m.logHealthEvent(ctx, appID, "auto_restarted", "container restarted due to health check failures")
}

// logHealthEvent logs a health-related event.
func (m *SandboxHealthMonitor) logHealthEvent(ctx context.Context, appID uuid.UUID, eventType string, message string) {
	if m.pool == nil {
		return
	}

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	details, _ := encodeJSON(map[string]interface{}{
		"message": message,
	})

	_, err := m.queries.InsertSandboxEvent(ctx, sqlc.InsertSandboxEventParams{
		ModuleInstanceID: pgAppID,
		EventType:        eventType,
		Details:          details,
	})
	if err != nil {
		m.logger.Warn("failed to log health event",
			"app_id", appID,
			"event_type", eventType,
			"error", err)
	}
}

// ClearAppTracking removes an app from health tracking.
func (m *SandboxHealthMonitor) ClearAppTracking(appID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.unhealthyCounts, appID)
	delete(m.lastCheck, appID)
}
