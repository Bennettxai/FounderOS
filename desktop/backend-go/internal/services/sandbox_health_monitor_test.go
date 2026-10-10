package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// testHealthLogger creates a logger for tests
func testHealthLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestDefaultHealthMonitorConfig(t *testing.T) {
	config := DefaultHealthMonitorConfig()

	if config.CheckInterval != defaultCheckInterval {
		t.Errorf("CheckInterval = %v, expected %v", config.CheckInterval, defaultCheckInterval)
	}
	if config.UnhealthyTimeout != defaultUnhealthyTimeout {
		t.Errorf("UnhealthyTimeout = %v, expected %v", config.UnhealthyTimeout, defaultUnhealthyTimeout)
	}
	if config.AutoRestart != false {
		t.Errorf("AutoRestart = %v, expected false", config.AutoRestart)
	}
	if config.MaxRetries != defaultMaxRetries {
		t.Errorf("MaxRetries = %d, expected %d", config.MaxRetries, defaultMaxRetries)
	}
}

func TestNewSandboxHealthMonitor(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)
	if monitor == nil {
		t.Fatal("expected monitor, got nil")
	}
	if monitor.logger == nil {
		t.Fatal("expected logger to be set")
	}
	if monitor.config.CheckInterval != defaultCheckInterval {
		t.Errorf("CheckInterval = %v, expected %v", monitor.config.CheckInterval, defaultCheckInterval)
	}
}

func TestNewSandboxHealthMonitor_ZeroConfig(t *testing.T) {
	logger := testHealthLogger()
	config := HealthMonitorConfig{} // All zeros

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)
	if monitor == nil {
		t.Fatal("expected monitor, got nil")
	}

	// Should use defaults
	if monitor.config.CheckInterval != defaultCheckInterval {
		t.Errorf("CheckInterval = %v, expected default %v", monitor.config.CheckInterval, defaultCheckInterval)
	}
	if monitor.config.UnhealthyTimeout != defaultUnhealthyTimeout {
		t.Errorf("UnhealthyTimeout = %v, expected default %v", monitor.config.UnhealthyTimeout, defaultUnhealthyTimeout)
	}
	if monitor.config.MaxRetries != defaultMaxRetries {
		t.Errorf("MaxRetries = %d, expected default %d", monitor.config.MaxRetries, defaultMaxRetries)
	}
}

func TestHealthStatus_Constants(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatusStarting, "starting"},
		{HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("HealthStatus = %s, expected %s", tt.status, tt.expected)
		}
	}
}

func TestHealthCheckResult_Fields(t *testing.T) {
	appID := uuid.New()
	now := time.Now().UTC()

	result := HealthCheckResult{
		AppID:        appID,
		ContainerID:  "abc123",
		Status:       HealthStatusHealthy,
		ResponseTime: 50 * time.Millisecond,
		CheckedAt:    now,
		Message:      "container is healthy",
	}

	if result.AppID != appID {
		t.Error("AppID mismatch")
	}
	if result.ContainerID != "abc123" {
		t.Errorf("ContainerID = %s, expected abc123", result.ContainerID)
	}
	if result.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, expected healthy", result.Status)
	}
	if result.ResponseTime != 50*time.Millisecond {
		t.Errorf("ResponseTime = %v, expected 50ms", result.ResponseTime)
	}
	if result.Message != "container is healthy" {
		t.Errorf("Message = %s, expected 'container is healthy'", result.Message)
	}
}

func TestHealthMonitor_IsRunning_InitialState(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)
	if monitor.IsRunning() {
		t.Error("monitor should not be running initially")
	}
}

func TestStartStop(t *testing.T) {
	logger := testHealthLogger()
	config := HealthMonitorConfig{
		CheckInterval: 100 * time.Millisecond,
	}

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	ctx := context.Background()

	// Start
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start monitor: %v", err)
	}

	if !monitor.IsRunning() {
		t.Error("monitor should be running after Start")
	}

	// Start again should be idempotent
	err = monitor.Start(ctx)
	if err != nil {
		t.Fatalf("second start should not error: %v", err)
	}

	// Stop
	monitor.Stop()
	time.Sleep(50 * time.Millisecond) // Give time for goroutine to stop

	if monitor.IsRunning() {
		t.Error("monitor should not be running after Stop")
	}

	// Stop again should be safe
	monitor.Stop()
}

func TestCheckContainer_NoManager(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	ctx := context.Background()
	appID := uuid.New()

	result, err := monitor.CheckContainer(ctx, appID, "container123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != HealthStatusUnknown {
		t.Errorf("Status = %s, expected unknown", result.Status)
	}
	if result.Message != "container manager not available" {
		t.Errorf("Message = %s, expected 'container manager not available'", result.Message)
	}
}

func TestSetHealthChangeCallback(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	callback := func(result HealthCheckResult) {
		// Callback for testing
		_ = result
	}

	monitor.SetHealthChangeCallback(callback)

	if monitor.onHealthChange == nil {
		t.Error("callback should be set")
	}
}

func TestGetHealthStats_Empty(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	stats := monitor.GetHealthStats()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if stats["running"].(bool) != false {
		t.Error("running should be false")
	}
	if stats["tracked_apps"].(int) != 0 {
		t.Errorf("tracked_apps = %d, expected 0", stats["tracked_apps"].(int))
	}
	if stats["unhealthy_count"].(int) != 0 {
		t.Errorf("unhealthy_count = %d, expected 0", stats["unhealthy_count"].(int))
	}
}

func TestClearAppTracking(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	appID := uuid.New()

	// Add some tracking data
	monitor.mu.Lock()
	monitor.unhealthyCounts[appID] = 3
	monitor.lastCheck[appID] = time.Now()
	monitor.mu.Unlock()

	// Verify data exists
	monitor.mu.RLock()
	if _, ok := monitor.unhealthyCounts[appID]; !ok {
		t.Error("expected unhealthyCounts entry to exist")
	}
	if _, ok := monitor.lastCheck[appID]; !ok {
		t.Error("expected lastCheck entry to exist")
	}
	monitor.mu.RUnlock()

	// Clear tracking
	monitor.ClearAppTracking(appID)

	// Verify data removed
	monitor.mu.RLock()
	if _, ok := monitor.unhealthyCounts[appID]; ok {
		t.Error("unhealthyCounts entry should be removed")
	}
	if _, ok := monitor.lastCheck[appID]; ok {
		t.Error("lastCheck entry should be removed")
	}
	monitor.mu.RUnlock()
}

func TestProcessHealthResult_Healthy(t *testing.T) {
	logger := testHealthLogger()
	config := DefaultHealthMonitorConfig()

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	appID := uuid.New()

	// Set previous unhealthy count
	monitor.mu.Lock()
	monitor.unhealthyCounts[appID] = 2
	monitor.mu.Unlock()

	result := &HealthCheckResult{
		AppID:     appID,
		Status:    HealthStatusHealthy,
		CheckedAt: time.Now(),
	}

	ctx := context.Background()
	monitor.processHealthResult(ctx, result)

	// Should reset unhealthy count
	monitor.mu.RLock()
	count := monitor.unhealthyCounts[appID]
	monitor.mu.RUnlock()

	if count != 0 {
		t.Errorf("unhealthyCounts = %d, expected 0 after healthy result", count)
	}
}

func TestProcessHealthResult_Unhealthy(t *testing.T) {
	logger := testHealthLogger()
	config := HealthMonitorConfig{
		CheckInterval: defaultCheckInterval,
		MaxRetries:    5, // Won't trigger restart
		AutoRestart:   false,
	}

	monitor := NewSandboxHealthMonitor(nil, nil, logger, config)

	appID := uuid.New()

	result := &HealthCheckResult{
		AppID:     appID,
		Status:    HealthStatusUnhealthy,
		CheckedAt: time.Now(),
		Message:   "test failure",
	}

	ctx := context.Background()
	monitor.processHealthResult(ctx, result)

	// Should increment unhealthy count
	monitor.mu.RLock()
	count := monitor.unhealthyCounts[appID]
	monitor.mu.RUnlock()

	if count != 1 {
		t.Errorf("unhealthyCounts = %d, expected 1 after first unhealthy result", count)
	}

	// Process again
	monitor.processHealthResult(ctx, result)

	monitor.mu.RLock()
	count = monitor.unhealthyCounts[appID]
	monitor.mu.RUnlock()

	if count != 2 {
		t.Errorf("unhealthyCounts = %d, expected 2 after second unhealthy result", count)
	}
}

func TestHealthMonitorConfig_Fields(t *testing.T) {
	config := HealthMonitorConfig{
		CheckInterval:    1 * time.Minute,
		UnhealthyTimeout: 5 * time.Minute,
		AutoRestart:      true,
		MaxRetries:       5,
	}

	if config.CheckInterval != 1*time.Minute {
		t.Errorf("CheckInterval = %v, expected 1m", config.CheckInterval)
	}
	if config.UnhealthyTimeout != 5*time.Minute {
		t.Errorf("UnhealthyTimeout = %v, expected 5m", config.UnhealthyTimeout)
	}
	if config.AutoRestart != true {
		t.Error("AutoRestart should be true")
	}
	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, expected 5", config.MaxRetries)
	}
}
