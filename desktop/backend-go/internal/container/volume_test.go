package container

import (
	"testing"
)

// TestGetVolumeName tests volume name generation
func TestGetVolumeName(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		expected string
	}{
		{
			name:     "standard user ID",
			userID:   "user123",
			expected: "workspace_user123",
		},
		{
			name:     "UUID user ID",
			userID:   "550e8400-e29b-41d4-a716-446655440000",
			expected: "workspace_550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "numeric user ID",
			userID:   "12345",
			expected: "workspace_12345",
		},
	}

	cm := &ContainerManager{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.getVolumeName(tt.userID)
			if result != tt.expected {
				t.Errorf("getVolumeName(%s) = %s, want %s", tt.userID, result, tt.expected)
			}
		})
	}
}

// TestCreateVolumeValidation tests input validation for CreateVolume
func TestCreateVolumeValidation(t *testing.T) {
	cm := &ContainerManager{
		cli: nil, // Not needed for validation tests
	}

	// Test empty userID
	_, err := cm.CreateVolume("")
	if err == nil {
		t.Error("CreateVolume with empty userID should return error")
	}
}

// TestRemoveVolumeValidation tests input validation for RemoveVolume
func TestRemoveVolumeValidation(t *testing.T) {
	cm := &ContainerManager{
		cli: nil, // Not needed for validation tests
	}

	// Test empty userID
	err := cm.RemoveVolume("")
	if err == nil {
		t.Error("RemoveVolume with empty userID should return error")
	}
}

// TestVolumeExistsValidation tests input validation for VolumeExists
func TestVolumeExistsValidation(t *testing.T) {
	cm := &ContainerManager{
		cli: nil, // Not needed for validation tests
	}

	// Test empty userID
	_, err := cm.VolumeExists("")
	if err == nil {
		t.Error("VolumeExists with empty userID should return error")
	}
}

// TestListUserVolumesValidation tests input validation for ListUserVolumes
func TestListUserVolumesValidation(t *testing.T) {
	cm := &ContainerManager{
		cli: nil, // Not needed for validation tests
	}

	// Test empty userID
	_, err := cm.ListUserVolumes("")
	if err == nil {
		t.Error("ListUserVolumes with empty userID should return error")
	}
}

// TestGetVolumeInfoValidation tests input validation for GetVolumeInfo
func TestGetVolumeInfoValidation(t *testing.T) {
	cm := &ContainerManager{
		cli: nil, // Not needed for validation tests
	}

	// Test empty volumeName
	_, err := cm.GetVolumeInfo("")
	if err == nil {
		t.Error("GetVolumeInfo with empty volumeName should return error")
	}
}

// Integration tests below require Docker to be running
// These can be enabled in CI/CD or local development with Docker available

/*
func TestCreateVolumeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cm, err := NewContainerManager()
	if err != nil {
		t.Fatalf("Failed to create container manager: %v", err)
	}
	defer cm.Close()

	testUserID := "test-user-123"

	// Clean up any existing test volume
	_ = cm.RemoveVolume(testUserID)

	// Create volume
	volumeName, err := cm.CreateVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	expectedVolumeName := "workspace_test-user-123"
	if volumeName != expectedVolumeName {
		t.Errorf("Expected volume name %s, got %s", expectedVolumeName, volumeName)
	}

	// Verify volume exists
	exists, err := cm.VolumeExists(testUserID)
	if err != nil {
		t.Fatalf("Failed to check volume existence: %v", err)
	}
	if !exists {
		t.Error("Volume should exist after creation")
	}

	// Clean up
	err = cm.RemoveVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}

	// Verify volume is removed
	exists, err = cm.VolumeExists(testUserID)
	if err != nil {
		t.Fatalf("Failed to check volume existence after removal: %v", err)
	}
	if exists {
		t.Error("Volume should not exist after removal")
	}
}

func TestListUserVolumesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cm, err := NewContainerManager()
	if err != nil {
		t.Fatalf("Failed to create container manager: %v", err)
	}
	defer cm.Close()

	testUserID := "test-user-456"

	// Clean up any existing test volume
	_ = cm.RemoveVolume(testUserID)

	// Create volume
	_, err = cm.CreateVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	// List user volumes
	volumes, err := cm.ListUserVolumes(testUserID)
	if err != nil {
		t.Fatalf("Failed to list user volumes: %v", err)
	}

	if len(volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(volumes))
	}

	// Verify volume labels
	if len(volumes) > 0 {
		vol := volumes[0]
		if vol.Labels["app"] != "businessos" {
			t.Errorf("Expected app label 'businessos', got %s", vol.Labels["app"])
		}
		if vol.Labels["user_id"] != testUserID {
			t.Errorf("Expected user_id label %s, got %s", testUserID, vol.Labels["user_id"])
		}
		if vol.Labels["type"] != "workspace" {
			t.Errorf("Expected type label 'workspace', got %s", vol.Labels["type"])
		}
	}

	// Clean up
	err = cm.RemoveVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}
}

func TestGetVolumeInfoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cm, err := NewContainerManager()
	if err != nil {
		t.Fatalf("Failed to create container manager: %v", err)
	}
	defer cm.Close()

	testUserID := "test-user-789"

	// Clean up any existing test volume
	_ = cm.RemoveVolume(testUserID)

	// Create volume
	volumeName, err := cm.CreateVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	// Get volume info
	volInfo, err := cm.GetVolumeInfo(volumeName)
	if err != nil {
		t.Fatalf("Failed to get volume info: %v", err)
	}

	if volInfo.Name != volumeName {
		t.Errorf("Expected volume name %s, got %s", volumeName, volInfo.Name)
	}

	// Clean up
	err = cm.RemoveVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}
}

func TestVolumeStatsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cm, err := NewContainerManager()
	if err != nil {
		t.Fatalf("Failed to create container manager: %v", err)
	}
	defer cm.Close()

	// Get initial stats
	stats, err := cm.GetVolumeStats()
	if err != nil {
		t.Fatalf("Failed to get volume stats: %v", err)
	}

	initialTotal := stats.TotalVolumes

	// Create test volume
	testUserID := "test-stats-user"
	_ = cm.RemoveVolume(testUserID)

	_, err = cm.CreateVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	// Get updated stats
	stats, err = cm.GetVolumeStats()
	if err != nil {
		t.Fatalf("Failed to get volume stats: %v", err)
	}

	if stats.TotalVolumes != initialTotal+1 {
		t.Errorf("Expected total volumes to increase by 1, got %d (initial: %d)", stats.TotalVolumes, initialTotal)
	}

	// Clean up
	err = cm.RemoveVolume(testUserID)
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}
}
*/
