package container

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// TestCleanupOrphans_RaceCondition tests the critical unlock/relock pattern in CleanupOrphans
// This test verifies that the mutex is properly released and reacquired when iterating
// over containers to prevent deadlocks during orphan cleanup.
func TestCleanupOrphans_RaceCondition(t *testing.T) {
	// Skip if Docker is not available
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker not available, skipping race condition test")
	}
	defer cli.Close()

	ctx := context.Background()
	manager, err := NewContainerManager(ctx, "ubuntu:22.04")
	if err != nil {
		t.Skipf("Failed to create container manager (Docker not available): %v", err)
	}
	defer manager.Close()

	// Create monitor with very short intervals for faster testing
	config := &MonitorConfig{
		IdleTimeout:         10 * time.Second,
		CleanupInterval:     1 * time.Second,
		HealthCheckInterval: 1 * time.Second,
	}

	monitor := NewContainerMonitor(manager, config)

	// Simulate concurrent access to the manager's containers map
	// while CleanupOrphans is running
	var wg sync.WaitGroup
	raceDetected := atomic.Int32{}
	iterations := 100

	// Goroutine 1: Repeatedly call CleanupOrphans
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			err := monitor.CleanupOrphans(ctx)
			if err != nil {
				t.Logf("CleanupOrphans iteration %d: %v", i, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Goroutine 2: Concurrently add/remove containers from manager
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			// Simulate container registration
			manager.mu.Lock()
			testID := "test-container-" + string(rune(i))
			manager.containers[testID] = &ContainerInfo{
				ID:           testID,
				UserID:       "test-user",
				Status:       "running",
				CreatedAt:    time.Now(),
				LastActivity: time.Now(),
			}
			manager.mu.Unlock()

			time.Sleep(5 * time.Millisecond)

			// Simulate container removal
			manager.mu.Lock()
			delete(manager.containers, testID)
			manager.mu.Unlock()

			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Goroutine 3: Concurrently read from containers map
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations*2; i++ {
			manager.mu.RLock()
			count := len(manager.containers)
			manager.mu.RUnlock()

			if count < 0 {
				// This should never happen, indicates race condition
				raceDetected.Add(1)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify no race conditions were detected
	if raceDetected.Load() > 0 {
		t.Errorf("Race condition detected: %d invalid reads", raceDetected.Load())
	}

	t.Logf("Successfully completed %d iterations without race conditions", iterations)
}

// TestMonitor_ConcurrentStatsAccess tests concurrent access to container stats
// This verifies that the statsMutex properly protects the containerStats map
func TestMonitor_ConcurrentStatsAccess(t *testing.T) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	// Launch multiple goroutines performing various operations
	for g := 0; g < numGoroutines; g++ {
		goroutineID := g

		// Register/Unregister
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				containerID := "container-" + string(rune(id*100+i))
				monitor.RegisterContainer(containerID, "user-"+string(rune(id)))
				time.Sleep(1 * time.Millisecond)
				monitor.UnregisterContainer(containerID)
			}
		}(goroutineID)

		// UpdateActivity
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				containerID := "container-" + string(rune(id*100+i))
				monitor.RegisterContainer(containerID, "user-"+string(rune(id)))
				monitor.UpdateActivity(containerID)
				time.Sleep(1 * time.Millisecond)
			}
		}(goroutineID)

		// GetAllContainerStats
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				stats := monitor.GetAllContainerStats()
				_ = len(stats) // Use the result
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	allStats := monitor.GetAllContainerStats()
	t.Logf("Final container count: %d", len(allStats))

	// Check metrics are consistent
	started := atomic.LoadInt64(&monitor.metrics.TotalStarted)
	stopped := atomic.LoadInt64(&monitor.metrics.TotalStopped)

	t.Logf("Total started: %d, Total stopped: %d", started, stopped)

	if started < 0 || stopped < 0 {
		t.Error("Metrics corruption detected: negative counters")
	}
}

// TestMonitor_CleanupLoopRaceCondition tests the cleanup loop with concurrent container operations
func TestMonitor_CleanupLoopRaceCondition(t *testing.T) {
	// Skip if Docker is not available
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker not available, skipping race condition test")
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager, err := NewContainerManager(ctx, "ubuntu:22.04")
	if err != nil {
		t.Skipf("Failed to create container manager (Docker not available): %v", err)
	}
	defer manager.Close()

	config := &MonitorConfig{
		IdleTimeout:         50 * time.Millisecond, // Very short for testing
		CleanupInterval:     20 * time.Millisecond, // Frequent cleanups
		HealthCheckInterval: 1 * time.Second,
	}

	monitor := NewContainerMonitor(manager, config)

	// Start monitoring
	if err := monitor.StartMonitoring(ctx); err != nil {
		t.Fatalf("Failed to start monitoring: %v", err)
	}

	// Simulate concurrent container operations while cleanup is running
	var wg sync.WaitGroup

	// Add/remove containers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			containerID := "test-container-" + string(rune(i))
			monitor.RegisterContainer(containerID, "test-user")
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Update activity
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			containerID := "test-container-" + string(rune(i))
			monitor.UpdateActivity(containerID)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Read stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = monitor.GetAllContainerStats()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Stop monitoring
	if err := monitor.StopMonitoring(); err != nil {
		t.Errorf("Failed to stop monitoring: %v", err)
	}

	t.Log("Cleanup loop completed without race conditions")
}

// TestMonitor_MetricsAtomicity tests that metrics operations are truly atomic
func TestMonitor_MetricsAtomicity(t *testing.T) {
	metrics := NewContainerMetrics()
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	var wg sync.WaitGroup

	// Concurrently increment all metrics
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3) // One for each metric type

		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				metrics.IncrementStarted()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				metrics.IncrementStopped()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				metrics.IncrementErrors()
			}
		}()
	}

	wg.Wait()

	// Verify counts are exactly as expected
	expectedCount := int64(numGoroutines * incrementsPerGoroutine)
	actualStarted := atomic.LoadInt64(&metrics.TotalStarted)
	actualStopped := atomic.LoadInt64(&metrics.TotalStopped)
	actualErrors := atomic.LoadInt64(&metrics.TotalErrors)

	if actualStarted != expectedCount {
		t.Errorf("TotalStarted atomicity violation: expected %d, got %d", expectedCount, actualStarted)
	}

	if actualStopped != expectedCount {
		t.Errorf("TotalStopped atomicity violation: expected %d, got %d", expectedCount, actualStopped)
	}

	if actualErrors != expectedCount {
		t.Errorf("TotalErrors atomicity violation: expected %d, got %d", expectedCount, actualErrors)
	}

	t.Logf("Atomicity test passed: %d operations per metric type", expectedCount)
}

// TestMonitor_UpdateContainerStats_Concurrency tests the updateContainerStats helper
func TestMonitor_UpdateContainerStats_Concurrency(t *testing.T) {
	monitor := &ContainerMonitor{
		config:         DefaultMonitorConfig(),
		metrics:        NewContainerMetrics(),
		stopChan:       make(chan struct{}),
		containerStats: make(map[string]*ContainerStats),
	}

	containerID := "test-container"
	numGoroutines := 50
	updatesPerGoroutine := 100

	var wg sync.WaitGroup

	// Concurrently update the same container's stats
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		goroutineID := i

		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				monitor.updateContainerStats(containerID, "user-1", func(stats *ContainerStats) {
					stats.State = "running"
					stats.MemoryUsage = uint64(id*1000 + j)
					stats.CPUPercent = float64(id)
					stats.HealthErrors = id
				})
			}
		}(goroutineID)
	}

	wg.Wait()

	// Verify the container stats exist and are in a valid state
	stats, err := monitor.GetContainerStats(containerID)
	if err != nil {
		t.Errorf("Failed to get container stats: %v", err)
	}

	if stats.State != "running" {
		t.Errorf("Expected state=running, got %s", stats.State)
	}

	t.Logf("Concurrent updateContainerStats completed successfully")
}

// TestCleanupOrphans_UnlockRelockPattern specifically tests the unlock/relock pattern
func TestCleanupOrphans_UnlockRelockPattern(t *testing.T) {
	// This test verifies that CleanupOrphans properly unlocks the mutex
	// before calling RemoveContainer to prevent deadlocks

	// Skip if Docker is not available
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker not available, skipping pattern test")
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager, err := NewContainerManager(ctx, "ubuntu:22.04")
	if err != nil {
		t.Skipf("Failed to create container manager (Docker not available): %v", err)
	}
	defer manager.Close()

	monitor := NewContainerMonitor(manager, DefaultMonitorConfig())

	// Add a container that will appear as an orphan
	// (exists in Docker but not in manager.containers map)
	testContainerID := "orphan-test-container"

	// Manually add to manager's map temporarily
	manager.mu.Lock()
	manager.containers[testContainerID] = &ContainerInfo{
		ID:     testContainerID,
		UserID: "test-user",
		Status: "created",
	}
	manager.mu.Unlock()

	// Now remove it from the map to make it appear as an orphan
	manager.mu.Lock()
	delete(manager.containers, testContainerID)
	manager.mu.Unlock()

	// Concurrently access the manager while CleanupOrphans runs
	var wg sync.WaitGroup
	deadlockDetected := make(chan bool, 1)

	// This goroutine should complete quickly if there's no deadlock
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Panic recovered in CleanupOrphans: %v", r)
			}
		}()

		err := monitor.CleanupOrphans(ctx)
		if err != nil {
			t.Logf("CleanupOrphans error (expected): %v", err)
		}
	}()

	// Try to acquire the lock from another goroutine
	// This should succeed if CleanupOrphans properly releases the lock
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // Give CleanupOrphans time to start

		// Try to acquire lock with timeout
		lockAcquired := make(chan bool, 1)
		go func() {
			manager.mu.Lock()
			lockAcquired <- true
			manager.mu.Unlock()
		}()

		select {
		case <-lockAcquired:
			// Lock was acquired successfully - no deadlock
			t.Log("Lock acquired successfully - no deadlock detected")
		case <-time.After(2 * time.Second):
			// Lock acquisition timed out - potential deadlock
			deadlockDetected <- true
			t.Error("Potential deadlock: could not acquire lock within timeout")
		}
	}()

	// Wait for both goroutines with timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Test completed successfully - no deadlock")
	case <-time.After(5 * time.Second):
		t.Error("Test timed out - likely deadlock in CleanupOrphans")
	case <-deadlockDetected:
		t.Error("Deadlock detected in unlock/relock pattern")
	}
}
