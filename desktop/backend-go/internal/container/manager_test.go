package container

import (
	"context"
	"testing"
	"time"
)

func TestNewContainerManager(t *testing.T) {
	ctx := context.Background()

	// Test with default image
	manager, err := NewContainerManager(ctx, "")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}
	defer manager.Close()

	// Verify manager is initialized
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.cli == nil {
		t.Fatal("Docker client should not be nil")
	}

	if manager.defaultImage != "ubuntu:22.04" {
		t.Errorf("Expected default image 'ubuntu:22.04', got '%s'", manager.defaultImage)
	}

	// Test Docker availability
	if !manager.IsDockerAvailable() {
		t.Fatal("Docker should be available")
	}
}

func TestNewContainerManagerWithCustomImage(t *testing.T) {
	ctx := context.Background()
	customImage := "alpine:latest"

	manager, err := NewContainerManager(ctx, customImage)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}
	defer manager.Close()

	if manager.GetDefaultImage() != customImage {
		t.Errorf("Expected custom image '%s', got '%s'", customImage, manager.GetDefaultImage())
	}
}

func TestContainerManagerClose(t *testing.T) {
	ctx := context.Background()

	manager, err := NewContainerManager(ctx, "")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}

	// Close should not error
	if err := manager.Close(); err != nil {
		t.Errorf("Close() should not error: %v", err)
	}

	// Multiple closes should be safe (though may error)
	manager.Close()
}

func TestListContainers(t *testing.T) {
	ctx := context.Background()

	manager, err := NewContainerManager(ctx, "")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}
	defer manager.Close()

	// Initially should have no containers
	containers := manager.ListContainers("")
	if len(containers) != 0 {
		t.Errorf("Expected 0 containers, got %d", len(containers))
	}

	// Add a mock container info
	manager.mu.Lock()
	manager.containers["test-id"] = &ContainerInfo{
		ID:           "test-id",
		UserID:       "user1",
		Image:        "ubuntu:22.04",
		Status:       "running",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	manager.mu.Unlock()

	// Should find the container
	containers = manager.ListContainers("")
	if len(containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(containers))
	}

	// Filter by user
	containers = manager.ListContainers("user1")
	if len(containers) != 1 {
		t.Errorf("Expected 1 container for user1, got %d", len(containers))
	}

	// Filter by different user should return nothing
	containers = manager.ListContainers("user2")
	if len(containers) != 0 {
		t.Errorf("Expected 0 containers for user2, got %d", len(containers))
	}
}

func TestUpdateActivity(t *testing.T) {
	ctx := context.Background()

	manager, err := NewContainerManager(ctx, "")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
		return
	}
	defer manager.Close()

	// Add a container
	now := time.Now()
	manager.mu.Lock()
	manager.containers["test-id"] = &ContainerInfo{
		ID:           "test-id",
		UserID:       "user1",
		Image:        "ubuntu:22.04",
		Status:       "running",
		CreatedAt:    now,
		LastActivity: now,
	}
	manager.mu.Unlock()

	// Wait a bit and update activity
	time.Sleep(10 * time.Millisecond)
	manager.UpdateActivity("test-id")

	// Verify activity was updated
	manager.mu.RLock()
	info := manager.containers["test-id"]
	manager.mu.RUnlock()

	if !info.LastActivity.After(now) {
		t.Error("LastActivity should be updated")
	}
}
