package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
)

// testSandboxLogger creates a logger for tests
func testSandboxLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// testSandboxConfig creates a config for tests
func testSandboxConfig() *config.Config {
	return &config.Config{
		SandboxPortMin:    9000,
		SandboxPortMax:    9100,
		SandboxMaxPerUser: 5,
	}
}

func TestNewSandboxDeploymentService(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	// Test creation without pool (should work)
	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if service.portAllocator == nil {
		t.Fatal("expected port allocator to be set")
	}
	if service.containerManager == nil {
		t.Fatal("expected container manager to be set")
	}
	if service.logger == nil {
		t.Fatal("expected logger to be set")
	}
}

func TestSandboxDeploymentRequest_Validation(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	// Test with nil app ID
	_, err = service.Deploy(ctx, SandboxDeploymentRequest{
		AppName: "test-app",
		UserID:  uuid.New(),
		Image:   "node:20-alpine",
	})
	if err == nil {
		t.Error("expected error for nil app ID")
	}
}

func TestSandboxStatus_Constants(t *testing.T) {
	tests := []struct {
		status   SandboxStatus
		expected string
	}{
		{SandboxStatusPending, "pending"},
		{SandboxStatusBuilding, "building"},
		{SandboxStatusRunning, "running"},
		{SandboxStatusStopped, "stopped"},
		{SandboxStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("SandboxStatus = %s, expected %s", tt.status, tt.expected)
		}
	}
}

func TestSandboxInfo_Fields(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()

	info := SandboxInfo{
		AppID:        appID,
		AppName:      "test-app",
		UserID:       userID,
		ContainerID:  "abc123",
		Status:       SandboxStatusRunning,
		Port:         9001,
		URL:          "http://localhost:9001",
		Image:        "node:20-alpine",
		HealthStatus: "healthy",
	}

	if info.AppID != appID {
		t.Error("AppID mismatch")
	}
	if info.AppName != "test-app" {
		t.Errorf("AppName = %s, expected test-app", info.AppName)
	}
	if info.UserID != userID {
		t.Error("UserID mismatch")
	}
	if info.ContainerID != "abc123" {
		t.Errorf("ContainerID = %s, expected abc123", info.ContainerID)
	}
	if info.Status != SandboxStatusRunning {
		t.Errorf("Status = %s, expected running", info.Status)
	}
	if info.Port != 9001 {
		t.Errorf("Port = %d, expected 9001", info.Port)
	}
	if info.URL != "http://localhost:9001" {
		t.Errorf("URL = %s, expected http://localhost:9001", info.URL)
	}
	if info.Image != "node:20-alpine" {
		t.Errorf("Image = %s, expected node:20-alpine", info.Image)
	}
	if info.HealthStatus != "healthy" {
		t.Errorf("HealthStatus = %s, expected healthy", info.HealthStatus)
	}
}

func TestSandboxDeploymentRequest_Fields(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()

	req := SandboxDeploymentRequest{
		AppID:         appID,
		AppName:       "test-app",
		UserID:        userID,
		Image:         "node:20-alpine",
		ContainerPort: 3000,
		WorkspacePath: "/workspace/test",
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		StartCommand: []string{"npm", "start"},
		WorkingDir:   "/app",
		MemoryLimit:  512 * 1024 * 1024,
		CPUQuota:     50000,
	}

	if req.AppID != appID {
		t.Error("AppID mismatch")
	}
	if req.AppName != "test-app" {
		t.Errorf("AppName = %s, expected test-app", req.AppName)
	}
	if req.UserID != userID {
		t.Error("UserID mismatch")
	}
	if req.Image != "node:20-alpine" {
		t.Errorf("Image = %s, expected node:20-alpine", req.Image)
	}
	if req.ContainerPort != 3000 {
		t.Errorf("ContainerPort = %d, expected 3000", req.ContainerPort)
	}
	if req.WorkspacePath != "/workspace/test" {
		t.Errorf("WorkspacePath = %s, expected /workspace/test", req.WorkspacePath)
	}
	if req.Environment["NODE_ENV"] != "production" {
		t.Error("Environment NODE_ENV mismatch")
	}
	if len(req.StartCommand) != 2 || req.StartCommand[0] != "npm" {
		t.Error("StartCommand mismatch")
	}
	if req.WorkingDir != "/app" {
		t.Errorf("WorkingDir = %s, expected /app", req.WorkingDir)
	}
	if req.MemoryLimit != 512*1024*1024 {
		t.Errorf("MemoryLimit = %d, expected %d", req.MemoryLimit, 512*1024*1024)
	}
	if req.CPUQuota != 50000 {
		t.Errorf("CPUQuota = %d, expected 50000", req.CPUQuota)
	}
}

func TestMapContainerStatus(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		dockerStatus   string
		expectedStatus SandboxStatus
	}{
		{"running", SandboxStatusRunning},
		{"exited", SandboxStatusStopped},
		{"dead", SandboxStatusStopped},
		{"created", SandboxStatusBuilding},
		{"restarting", SandboxStatusBuilding},
		{"unknown", SandboxStatusPending},
		{"", SandboxStatusPending},
	}

	for _, tt := range tests {
		result := service.mapContainerStatus(tt.dockerStatus)
		if result != tt.expectedStatus {
			t.Errorf("mapContainerStatus(%s) = %s, expected %s", tt.dockerStatus, result, tt.expectedStatus)
		}
	}
}

func TestGetStats_NoPool(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	// Should have in_progress_deployments
	if _, ok := stats["in_progress_deployments"]; !ok {
		t.Error("expected in_progress_deployments in stats")
	}

	// Should have port allocator stats
	if _, ok := stats["port_min_port"]; !ok {
		t.Error("expected port_min_port in stats")
	}
	if _, ok := stats["port_max_port"]; !ok {
		t.Error("expected port_max_port in stats")
	}
}

func TestGetSandboxInfo_NoPool(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	_, err = service.GetSandboxInfo(ctx, uuid.New())
	if err != ErrSandboxNotFound {
		t.Errorf("expected ErrSandboxNotFound, got %v", err)
	}
}

func TestListUserSandboxes_NoPool(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	sandboxes, err := service.ListUserSandboxes(ctx, uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sandboxes) != 0 {
		t.Errorf("expected empty list, got %d sandboxes", len(sandboxes))
	}
}

func TestCheckUserQuota_NoPool(t *testing.T) {
	logger := testSandboxLogger()
	cfg := testSandboxConfig()

	service, err := NewSandboxDeploymentService(nil, nil, cfg, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	// Should not return error when pool is nil
	err = service.checkUserQuota(ctx, uuid.New())
	if err != nil {
		t.Errorf("expected no error when pool is nil, got %v", err)
	}
}

func TestErrors(t *testing.T) {
	// Verify error messages
	tests := []struct {
		err      error
		expected string
	}{
		{ErrSandboxNotFound, "sandbox not found"},
		{ErrSandboxAlreadyRunning, "sandbox is already running"},
		{ErrSandboxNotRunning, "sandbox is not running"},
		{ErrMaxSandboxesReached, "maximum number of sandboxes reached for user"},
		{ErrDeploymentFailed, "sandbox deployment failed"},
		{ErrInvalidAppID, "invalid app ID"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error = %s, expected %s", tt.err.Error(), tt.expected)
		}
	}
}
