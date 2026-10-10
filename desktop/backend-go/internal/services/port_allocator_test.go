package services

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
	"log/slog"
	"os"
)

// testLogger creates a logger for tests
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// testConfig creates a config for tests
func testConfig(minPort, maxPort int) *config.Config {
	return &config.Config{
		SandboxPortMin: minPort,
		SandboxPortMax: maxPort,
	}
}

func TestNewSandboxPortAllocator_ValidConfig(t *testing.T) {
	cfg := testConfig(9000, 9100)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pa == nil {
		t.Fatal("expected port allocator, got nil")
	}
	if pa.minPort != 9000 {
		t.Errorf("expected minPort 9000, got %d", pa.minPort)
	}
	if pa.maxPort != 9100 {
		t.Errorf("expected maxPort 9100, got %d", pa.maxPort)
	}
}

func TestNewSandboxPortAllocator_InvalidConfig(t *testing.T) {
	logger := testLogger()

	tests := []struct {
		name    string
		minPort int
		maxPort int
	}{
		{"negative min port", -1, 9100},
		{"negative max port", 9000, -1},
		{"min equals max", 9000, 9000},
		{"min greater than max", 9100, 9000},
		{"range too small", 9000, 9005},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(tt.minPort, tt.maxPort)
			_, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
			if err == nil {
				t.Error("expected error for invalid config")
			}
		})
	}
}

func TestPortAllocator_Allocate(t *testing.T) {
	cfg := testConfig(9000, 9010)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()
	appID := uuid.New()

	// Allocate a port
	port, err := pa.Allocate(ctx, appID)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	// Port should be in range
	if port < 9000 || port > 9010 {
		t.Errorf("port %d out of range [9000, 9010]", port)
	}

	// Allocating again for same app should return same port
	port2, err := pa.Allocate(ctx, appID)
	if err != nil {
		t.Fatalf("failed to re-allocate port: %v", err)
	}
	if port2 != port {
		t.Errorf("expected same port %d, got %d", port, port2)
	}
}

func TestPortAllocator_AllocateSequential(t *testing.T) {
	cfg := testConfig(9000, 9020)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()
	allocatedPorts := make(map[int]bool)

	// Allocate multiple ports
	for i := 0; i < 10; i++ {
		appID := uuid.New()
		port, err := pa.Allocate(ctx, appID)
		if err != nil {
			t.Fatalf("failed to allocate port %d: %v", i, err)
		}

		if allocatedPorts[port] {
			t.Errorf("port %d allocated twice", port)
		}
		allocatedPorts[port] = true
	}

	if len(allocatedPorts) != 10 {
		t.Errorf("expected 10 unique ports, got %d", len(allocatedPorts))
	}
}

func TestPortAllocator_AllocateConcurrent(t *testing.T) {
	cfg := testConfig(9000, 9200) // Larger range for concurrent test
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ports := make(chan int, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrent allocation
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			appID := uuid.New()
			port, err := pa.Allocate(ctx, appID)
			if err != nil {
				errors <- err
				return
			}
			ports <- port
		}()
	}

	wg.Wait()
	close(ports)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("allocation error: %v", err)
	}

	// Check for duplicate ports
	allocatedPorts := make(map[int]bool)
	for port := range ports {
		if allocatedPorts[port] {
			t.Errorf("port %d allocated to multiple apps", port)
		}
		allocatedPorts[port] = true
	}

	if len(allocatedPorts) != numGoroutines {
		t.Errorf("expected %d unique ports, got %d", numGoroutines, len(allocatedPorts))
	}
}

func TestPortAllocator_Release(t *testing.T) {
	cfg := testConfig(9000, 9010)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()
	appID := uuid.New()

	// Allocate a port
	port, err := pa.Allocate(ctx, appID)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}

	// Release the port
	err = pa.Release(ctx, appID)
	if err != nil {
		t.Fatalf("failed to release port: %v", err)
	}

	// Port should be available again
	if !pa.IsAvailable(ctx, port) {
		t.Error("port should be available after release")
	}

	// Allocating a new app should be able to get the released port
	appID2 := uuid.New()
	port2, err := pa.Allocate(ctx, appID2)
	if err != nil {
		t.Fatalf("failed to allocate after release: %v", err)
	}

	// Note: port2 might be the same as port since it's the first available
	if port2 < 9000 || port2 > 9010 {
		t.Errorf("port %d out of range", port2)
	}
}

func TestPortAllocator_Exhaustion(t *testing.T) {
	cfg := testConfig(9000, 9010) // 11 ports available (9000-9010 inclusive)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()

	// Allocate all 11 ports
	for i := 0; i < 11; i++ {
		appID := uuid.New()
		_, err := pa.Allocate(ctx, appID)
		if err != nil {
			t.Fatalf("failed to allocate port %d: %v", i, err)
		}
	}

	// Next allocation should fail
	appID := uuid.New()
	_, err = pa.Allocate(ctx, appID)
	if err != ErrPortsExhausted {
		t.Errorf("expected ErrPortsExhausted, got %v", err)
	}
}

func TestPortAllocator_IsAvailable(t *testing.T) {
	cfg := testConfig(9000, 9010)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()

	// Port in range should be available
	if !pa.IsAvailable(ctx, 9005) {
		t.Error("port 9005 should be available")
	}

	// Port outside range should not be available
	if pa.IsAvailable(ctx, 8000) {
		t.Error("port 8000 should not be available (out of range)")
	}
	if pa.IsAvailable(ctx, 10000) {
		t.Error("port 10000 should not be available (out of range)")
	}

	// Allocate a port
	appID := uuid.New()
	port, err := pa.Allocate(ctx, appID)
	if err != nil {
		t.Fatalf("failed to allocate: %v", err)
	}

	// Allocated port should not be available
	if pa.IsAvailable(ctx, port) {
		t.Errorf("port %d should not be available after allocation", port)
	}
}

func TestPortAllocator_GetPortForApp(t *testing.T) {
	cfg := testConfig(9000, 9010)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()
	appID := uuid.New()

	// Get port for non-existent app
	_, err = pa.GetPortForApp(ctx, appID)
	if err != ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}

	// Allocate a port
	port, err := pa.Allocate(ctx, appID)
	if err != nil {
		t.Fatalf("failed to allocate: %v", err)
	}

	// Get port for existing app
	gotPort, err := pa.GetPortForApp(ctx, appID)
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}
	if gotPort != port {
		t.Errorf("expected port %d, got %d", port, gotPort)
	}
}

func TestPortAllocator_GetStats(t *testing.T) {
	cfg := testConfig(9000, 9010)
	logger := testLogger()

	pa, err := NewSandboxPortAllocator(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	ctx := context.Background()

	// Initial stats
	stats := pa.GetStats()
	if stats["min_port"] != 9000 {
		t.Errorf("expected min_port 9000, got %v", stats["min_port"])
	}
	if stats["max_port"] != 9010 {
		t.Errorf("expected max_port 9010, got %v", stats["max_port"])
	}
	if stats["total_ports"] != 11 {
		t.Errorf("expected total_ports 11, got %v", stats["total_ports"])
	}
	if stats["allocated"] != 0 {
		t.Errorf("expected allocated 0, got %v", stats["allocated"])
	}

	// Allocate some ports
	for i := 0; i < 3; i++ {
		appID := uuid.New()
		_, err := pa.Allocate(ctx, appID)
		if err != nil {
			t.Fatalf("failed to allocate: %v", err)
		}
	}

	stats = pa.GetStats()
	if stats["allocated"] != 3 {
		t.Errorf("expected allocated 3, got %v", stats["allocated"])
	}
	if stats["available"] != 8 {
		t.Errorf("expected available 8, got %v", stats["available"])
	}
}
