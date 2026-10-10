package container

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultMonitorConfig(t *testing.T) {
	config := DefaultMonitorConfig()

	if config.IdleTimeout != 30*time.Minute {
		t.Errorf("Expected IdleTimeout=30m, got %v", config.IdleTimeout)
	}
	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("Expected CleanupInterval=5m, got %v", config.CleanupInterval)
	}
	if config.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected HealthCheckInterval=30s, got %v", config.HealthCheckInterval)
	}
	if config.MaxMemoryBytes != 512*1024*1024 {
		t.Errorf("Expected MaxMemoryBytes=512MB, got %d", config.MaxMemoryBytes)
	}
	if config.MaxCPUPercent != 50.0 {
		t.Errorf("Expected MaxCPUPercent=50.0, got %f", config.MaxCPUPercent)
	}
}

func TestNewContainerMetrics(t *testing.T) {
	metrics := NewContainerMetrics()

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}
	if metrics.MonitorStartTime.IsZero() {
		t.Error("Expected MonitorStartTime to be set")
	}
	if atomic.LoadInt64(&metrics.ActiveContainers) != 0 {
		t.Error("Expected ActiveContainers to be 0")
	}
	if atomic.LoadInt64(&metrics.TotalStarted) != 0 {
		t.Error("Expected TotalStarted to be 0")
	}
}

func TestContainerMetrics_AtomicOperations(t *testing.T) {
	metrics := NewContainerMetrics()

	// Test increment operations
	metrics.IncrementStarted()
	metrics.IncrementStarted()
	metrics.IncrementStarted()

	if atomic.LoadInt64(&metrics.TotalStarted) != 3 {
		t.Errorf("Expected TotalStarted=3, got %d", atomic.LoadInt64(&metrics.TotalStarted))
	}

	metrics.IncrementStopped()
	metrics.IncrementStopped()

	if atomic.LoadInt64(&metrics.TotalStopped) != 2 {
		t.Errorf("Expected TotalStopped=2, got %d", atomic.LoadInt64(&metrics.TotalStopped))
	}

	metrics.IncrementErrors()

	if atomic.LoadInt64(&metrics.TotalErrors) != 1 {
		t.Errorf("Expected TotalErrors=1, got %d", atomic.LoadInt64(&metrics.TotalErrors))
	}
}

func TestContainerMetrics_ToJSON(t *testing.T) {
	metrics := NewContainerMetrics()
	metrics.IncrementStarted()
	metrics.IncrementStopped()
	metrics.IncrementErrors()

	json := metrics.ToJSON()

	if json["total_started"] != int64(1) {
		t.Errorf("Expected total_started=1, got %v", json["total_started"])
	}
	if json["total_stopped"] != int64(1) {
		t.Errorf("Expected total_stopped=1, got %v", json["total_stopped"])
	}
	if json["total_errors"] != int64(1) {
		t.Errorf("Expected total_errors=1, got %v", json["total_errors"])
	}
	if _, exists := json["uptime_seconds"]; !exists {
		t.Error("Expected uptime_seconds in JSON")
	}
}

func TestContainerStats_Creation(t *testing.T) {
	stats := &ContainerStats{
		ContainerID:  "test-container-123",
		UserID:       "test-user-456",
		State:        "running",
		LastActivity: time.Now(),
		IsHealthy:    true,
	}

	if stats.ContainerID != "test-container-123" {
		t.Errorf("Expected ContainerID=test-container-123, got %s", stats.ContainerID)
	}
	if stats.IsHealthy != true {
		t.Error("Expected IsHealthy=true")
	}
	if stats.IsZombie != false {
		t.Error("Expected IsZombie=false by default")
	}
}

func TestContainerMonitor_RegisterUnregister(t *testing.T) {
	// Create a mock-like container monitor without actual Docker connection
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	// Register container
	monitor.RegisterContainer("container-1", "user-1")

	// Verify registration
	monitor.statsMutex.RLock()
	stats, exists := monitor.containerStats["container-1"]
	monitor.statsMutex.RUnlock()

	if !exists {
		t.Error("Expected container to be registered")
	}
	if stats.ContainerID != "container-1" {
		t.Errorf("Expected ContainerID=container-1, got %s", stats.ContainerID)
	}
	if stats.UserID != "user-1" {
		t.Errorf("Expected UserID=user-1, got %s", stats.UserID)
	}
	if atomic.LoadInt64(&monitor.metrics.TotalStarted) != 1 {
		t.Error("Expected TotalStarted to increment")
	}

	// Unregister container
	monitor.UnregisterContainer("container-1")

	monitor.statsMutex.RLock()
	_, exists = monitor.containerStats["container-1"]
	monitor.statsMutex.RUnlock()

	if exists {
		t.Error("Expected container to be unregistered")
	}
	if atomic.LoadInt64(&monitor.metrics.TotalStopped) != 1 {
		t.Error("Expected TotalStopped to increment")
	}
}

func TestContainerMonitor_UpdateActivity(t *testing.T) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	// Register container
	initialTime := time.Now().Add(-time.Hour)
	monitor.containerStats["container-1"] = &ContainerStats{
		ContainerID:  "container-1",
		LastActivity: initialTime,
	}

	// Update activity
	monitor.UpdateActivity("container-1")

	// Verify update
	monitor.statsMutex.RLock()
	stats := monitor.containerStats["container-1"]
	monitor.statsMutex.RUnlock()

	if stats.LastActivity.Before(initialTime) || stats.LastActivity.Equal(initialTime) {
		t.Error("Expected LastActivity to be updated")
	}
}

func TestContainerMonitor_GetAllContainerStats(t *testing.T) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	// Register multiple containers
	monitor.RegisterContainer("container-1", "user-1")
	monitor.RegisterContainer("container-2", "user-2")
	monitor.RegisterContainer("container-3", "user-1")

	// Get all stats
	allStats := monitor.GetAllContainerStats()

	if len(allStats) != 3 {
		t.Errorf("Expected 3 containers, got %d", len(allStats))
	}

	// Verify it's a copy (modify original and check copy is unchanged)
	monitor.statsMutex.Lock()
	monitor.containerStats["container-1"].State = "modified"
	monitor.statsMutex.Unlock()

	if allStats["container-1"].State == "modified" {
		t.Error("Expected GetAllContainerStats to return a copy, not reference")
	}
}

func TestContainerMonitor_GetContainerStats(t *testing.T) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	// Test non-existent container
	_, err := monitor.GetContainerStats("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}

	// Register and retrieve
	monitor.RegisterContainer("container-1", "user-1")
	stats, err := monitor.GetContainerStats("container-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if stats.ContainerID != "container-1" {
		t.Errorf("Expected ContainerID=container-1, got %s", stats.ContainerID)
	}
}

func TestContainerMonitor_IdleDetection(t *testing.T) {
	config := &MonitorConfig{
		IdleTimeout:         100 * time.Millisecond,
		CleanupInterval:     time.Minute,
		HealthCheckInterval: time.Minute,
	}

	monitor := &ContainerMonitor{
		config:         config,
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	// Add container with old activity
	monitor.containerStats["old-container"] = &ContainerStats{
		ContainerID:  "old-container",
		LastActivity: time.Now().Add(-time.Hour),
	}

	// Add container with recent activity
	monitor.containerStats["new-container"] = &ContainerStats{
		ContainerID:  "new-container",
		LastActivity: time.Now(),
	}

	// Check idle detection logic
	monitor.statsMutex.RLock()
	oldStats := monitor.containerStats["old-container"]
	newStats := monitor.containerStats["new-container"]
	monitor.statsMutex.RUnlock()

	if time.Since(oldStats.LastActivity) < config.IdleTimeout {
		t.Error("Expected old container to be detected as idle")
	}
	if time.Since(newStats.LastActivity) > config.IdleTimeout {
		t.Error("Expected new container to not be detected as idle")
	}
}

// Benchmark tests for metrics operations

func BenchmarkMetricsIncrement(b *testing.B) {
	metrics := NewContainerMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementStarted()
	}
}

func BenchmarkMetricsToJSON(b *testing.B) {
	metrics := NewContainerMetrics()
	metrics.IncrementStarted()
	metrics.IncrementStopped()
	metrics.IncrementErrors()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ToJSON()
	}
}

func BenchmarkRegisterUnregister(b *testing.B) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RegisterContainer("container", "user")
		monitor.UnregisterContainer("container")
	}
}

func BenchmarkUpdateActivity(b *testing.B) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}
	monitor.RegisterContainer("container", "user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.UpdateActivity("container")
	}
}
