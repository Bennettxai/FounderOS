package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// testCleanupLogger creates a logger for tests
func testCleanupLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestDefaultCleanupConfig(t *testing.T) {
	config := DefaultCleanupConfig()

	if config.CleanupInterval != defaultCleanupInterval {
		t.Errorf("CleanupInterval = %v, expected %v", config.CleanupInterval, defaultCleanupInterval)
	}
	if config.StoppedGracePeriod != defaultStoppedGracePeriod {
		t.Errorf("StoppedGracePeriod = %v, expected %v", config.StoppedGracePeriod, defaultStoppedGracePeriod)
	}
	if config.EventRetention != defaultEventRetention {
		t.Errorf("EventRetention = %v, expected %v", config.EventRetention, defaultEventRetention)
	}
	if config.DryRun != false {
		t.Errorf("DryRun = %v, expected false", config.DryRun)
	}
}

func TestNewSandboxCleanupService(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if service.logger == nil {
		t.Fatal("expected logger to be set")
	}
}

func TestNewSandboxCleanupService_ZeroConfig(t *testing.T) {
	logger := testCleanupLogger()
	config := CleanupConfig{} // All zeros

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)
	if service == nil {
		t.Fatal("expected service, got nil")
	}

	// Should use defaults
	if service.config.CleanupInterval != defaultCleanupInterval {
		t.Errorf("CleanupInterval = %v, expected default %v", service.config.CleanupInterval, defaultCleanupInterval)
	}
	if service.config.StoppedGracePeriod != defaultStoppedGracePeriod {
		t.Errorf("StoppedGracePeriod = %v, expected default %v", service.config.StoppedGracePeriod, defaultStoppedGracePeriod)
	}
	if service.config.EventRetention != defaultEventRetention {
		t.Errorf("EventRetention = %v, expected default %v", service.config.EventRetention, defaultEventRetention)
	}
}

func TestCleanupConfig_Fields(t *testing.T) {
	config := CleanupConfig{
		CleanupInterval:    2 * time.Hour,
		StoppedGracePeriod: 48 * time.Hour,
		EventRetention:     14 * 24 * time.Hour,
		DryRun:             true,
	}

	if config.CleanupInterval != 2*time.Hour {
		t.Errorf("CleanupInterval = %v, expected 2h", config.CleanupInterval)
	}
	if config.StoppedGracePeriod != 48*time.Hour {
		t.Errorf("StoppedGracePeriod = %v, expected 48h", config.StoppedGracePeriod)
	}
	if config.EventRetention != 14*24*time.Hour {
		t.Errorf("EventRetention = %v, expected 14d", config.EventRetention)
	}
	if config.DryRun != true {
		t.Error("DryRun should be true")
	}
}

func TestCleanupResult_Fields(t *testing.T) {
	now := time.Now().UTC()

	result := CleanupResult{
		OrphanedContainersRemoved: 3,
		StoppedContainersRemoved:  2,
		OldEventsDeleted:          100,
		PortsReleased:             1,
		Errors:                    make([]error, 0),
		StartedAt:                 now,
		Duration:                  5 * time.Second,
	}

	if result.OrphanedContainersRemoved != 3 {
		t.Errorf("OrphanedContainersRemoved = %d, expected 3", result.OrphanedContainersRemoved)
	}
	if result.StoppedContainersRemoved != 2 {
		t.Errorf("StoppedContainersRemoved = %d, expected 2", result.StoppedContainersRemoved)
	}
	if result.OldEventsDeleted != 100 {
		t.Errorf("OldEventsDeleted = %d, expected 100", result.OldEventsDeleted)
	}
	if result.PortsReleased != 1 {
		t.Errorf("PortsReleased = %d, expected 1", result.PortsReleased)
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, expected 5s", result.Duration)
	}
}

func TestIsRunning_InitialState(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)
	if service.IsRunning() {
		t.Error("service should not be running initially")
	}
}

func TestCleanupStartStop(t *testing.T) {
	logger := testCleanupLogger()
	config := CleanupConfig{
		CleanupInterval: 1 * time.Hour, // Long interval to avoid actual cleanup
	}

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()

	// Start
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Error("service should be running after Start")
	}

	// Start again should be idempotent
	err = service.Start(ctx)
	if err != nil {
		t.Fatalf("second start should not error: %v", err)
	}

	// Stop
	service.Stop()
	time.Sleep(50 * time.Millisecond) // Give time for goroutine to stop

	if service.IsRunning() {
		t.Error("service should not be running after Stop")
	}

	// Stop again should be safe
	service.Stop()
}

func TestRunCleanup_NoResources(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := service.RunCleanup(ctx)

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
	// Duration might be 0 if cleanup is very fast
	if result.Duration < 0 {
		t.Error("Duration should be >= 0")
	}
	// With no container manager or pool, counts should be 0
	if result.OrphanedContainersRemoved != 0 {
		t.Errorf("OrphanedContainersRemoved = %d, expected 0", result.OrphanedContainersRemoved)
	}
	if result.StoppedContainersRemoved != 0 {
		t.Errorf("StoppedContainersRemoved = %d, expected 0", result.StoppedContainersRemoved)
	}
}

func TestRunCleanup_DryRun(t *testing.T) {
	logger := testCleanupLogger()
	config := CleanupConfig{
		CleanupInterval: defaultCleanupInterval,
		DryRun:          true,
	}

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := service.RunCleanup(ctx)

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	// Dry run should not report any actual cleanups
	if result.OrphanedContainersRemoved != 0 {
		t.Errorf("OrphanedContainersRemoved = %d, expected 0 in dry run", result.OrphanedContainersRemoved)
	}
}

func TestGetStats_Empty(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if stats["running"].(bool) != false {
		t.Error("running should be false")
	}
	if stats["cleanup_interval"].(string) != config.CleanupInterval.String() {
		t.Errorf("cleanup_interval = %s, expected %s", stats["cleanup_interval"], config.CleanupInterval.String())
	}
	if stats["dry_run"].(bool) != false {
		t.Error("dry_run should be false")
	}
}

func TestGetStats_AfterCleanup(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	service.RunCleanup(ctx)

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	// Should have last_cleanup stats after running cleanup
	if _, ok := stats["last_cleanup"]; !ok {
		t.Error("expected last_cleanup in stats")
	}
	if _, ok := stats["last_duration"]; !ok {
		t.Error("expected last_duration in stats")
	}
}

func TestCleanupFailedDeployment_NoResources(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	appID := uuid.New()

	// Should not error even with no resources
	err := service.CleanupFailedDeployment(ctx, appID, "container123")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCleanupOrphanedContainers_NoManager(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := &CleanupResult{Errors: make([]error, 0)}

	count := service.cleanupOrphanedContainers(ctx, result)
	if count != 0 {
		t.Errorf("count = %d, expected 0 with no container manager", count)
	}
}

func TestCleanupStoppedContainers_NoPool(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := &CleanupResult{Errors: make([]error, 0)}

	count := service.cleanupStoppedContainers(ctx, result)
	if count != 0 {
		t.Errorf("count = %d, expected 0 with no pool", count)
	}
}

func TestCleanupOldEvents_NoPool(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := &CleanupResult{Errors: make([]error, 0)}

	count := service.cleanupOldEvents(ctx, result)
	if count != 0 {
		t.Errorf("count = %d, expected 0 with no pool", count)
	}
}

func TestCleanupOrphanedPorts_NoAllocator(t *testing.T) {
	logger := testCleanupLogger()
	config := DefaultCleanupConfig()

	service := NewSandboxCleanupService(nil, nil, nil, logger, config)

	ctx := context.Background()
	result := &CleanupResult{Errors: make([]error, 0)}

	count := service.cleanupOrphanedPorts(ctx, result)
	if count != 0 {
		t.Errorf("count = %d, expected 0 with no port allocator", count)
	}
}
