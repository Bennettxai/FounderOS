package container

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// testAppLogger creates a logger for tests
func testAppLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewAppContainerManager(t *testing.T) {
	logger := testAppLogger()

	// Test creation without Docker client (should work)
	manager := NewAppContainerManager(nil, logger, "")
	if manager == nil {
		t.Fatal("expected manager, got nil")
	}
	if manager.logger == nil {
		t.Fatal("expected logger to be set")
	}
}

func TestAppContainerConfig_Validation(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	tests := []struct {
		name    string
		config  AppContainerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				AppName:       "test-app",
				UserID:        uuid.New(),
				Image:         "node:20-alpine",
				ContainerPort: 3000,
				HostPort:      9001,
			},
			wantErr: false,
		},
		{
			name: "missing app ID",
			config: AppContainerConfig{
				AppName:       "test-app",
				Image:         "node:20-alpine",
				ContainerPort: 3000,
				HostPort:      9001,
			},
			wantErr: true,
		},
		{
			name: "missing app name",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				Image:         "node:20-alpine",
				ContainerPort: 3000,
				HostPort:      9001,
			},
			wantErr: true,
		},
		{
			name: "missing image",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				AppName:       "test-app",
				ContainerPort: 3000,
				HostPort:      9001,
			},
			wantErr: true,
		},
		{
			name: "invalid container port",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				AppName:       "test-app",
				Image:         "node:20-alpine",
				ContainerPort: 0,
				HostPort:      9001,
			},
			wantErr: true,
		},
		{
			name: "invalid host port",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				AppName:       "test-app",
				Image:         "node:20-alpine",
				ContainerPort: 3000,
				HostPort:      0,
			},
			wantErr: true,
		},
		{
			name: "negative container port",
			config: AppContainerConfig{
				AppID:         uuid.New(),
				AppName:       "test-app",
				Image:         "node:20-alpine",
				ContainerPort: -1,
				HostPort:      9001,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildEnvironmentVars(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	appID := uuid.New()
	userID := uuid.New()

	config := AppContainerConfig{
		AppID:         appID,
		AppName:       "test-app",
		UserID:        userID,
		Image:         "node:20-alpine",
		ContainerPort: 3000,
		HostPort:      9001,
		Environment: map[string]string{
			"DATABASE_URL": "postgres://localhost/db",
			"api-key":      "secret123",
		},
	}

	envVars := manager.buildEnvironmentVars(config)

	// Check required environment variables
	expected := map[string]bool{
		"APP_ID=" + appID.String():   false,
		"APP_NAME=test-app":          false,
		"USER_ID=" + userID.String(): false,
		"PORT=3000":                   false,
		"NODE_ENV=production":         false,
		"DATABASE_URL=postgres://localhost/db": false,
		"API_KEY=secret123": false, // Note: api-key becomes API_KEY
	}

	for _, env := range envVars {
		if _, ok := expected[env]; ok {
			expected[env] = true
		}
	}

	for env, found := range expected {
		if !found {
			t.Errorf("expected environment variable %s not found", env)
		}
	}
}

func TestStripDockerLogHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "short line",
			input:    "short",
			expected: "short",
		},
		{
			name:     "line with header",
			input:    "\x01\x00\x00\x00\x00\x00\x00\x0bhello world",
			expected: "hello world", // strips first 8 bytes (header), keeps payload
		},
		{
			name:     "multiple lines",
			input:    "12345678line1\n12345678line2",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripDockerLogHeaders(tt.input)
			if result != tt.expected {
				t.Errorf("stripDockerLogHeaders() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestAppContainerInfo_Fields(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()

	info := AppContainerInfo{
		ContainerID:   "abc123def456",
		AppID:         appID,
		AppName:       "test-app",
		UserID:        userID,
		Image:         "node:20-alpine",
		Status:        "running",
		HostPort:      9001,
		ContainerPort: 3000,
		IPAddress:     "172.17.0.2",
		HealthStatus:  "healthy",
	}

	if info.ContainerID != "abc123def456" {
		t.Errorf("ContainerID = %s, expected abc123def456", info.ContainerID)
	}
	if info.AppID != appID {
		t.Errorf("AppID mismatch")
	}
	if info.AppName != "test-app" {
		t.Errorf("AppName = %s, expected test-app", info.AppName)
	}
	if info.UserID != userID {
		t.Errorf("UserID mismatch")
	}
	if info.Status != "running" {
		t.Errorf("Status = %s, expected running", info.Status)
	}
	if info.HostPort != 9001 {
		t.Errorf("HostPort = %d, expected 9001", info.HostPort)
	}
	if info.ContainerPort != 3000 {
		t.Errorf("ContainerPort = %d, expected 3000", info.ContainerPort)
	}
	if info.HealthStatus != "healthy" {
		t.Errorf("HealthStatus = %s, expected healthy", info.HealthStatus)
	}
}

func TestCreateAppContainer_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	config := AppContainerConfig{
		AppID:         uuid.New(),
		AppName:       "test-app",
		UserID:        uuid.New(),
		Image:         "node:20-alpine",
		ContainerPort: 3000,
		HostPort:      9001,
	}

	_, err := manager.CreateAppContainer(ctx, config)
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestStartAppContainer_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	err := manager.StartAppContainer(ctx, "test-container-id")
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestStopAppContainer_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	err := manager.StopAppContainer(ctx, "test-container-id", nil)
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestRemoveAppContainer_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	err := manager.RemoveAppContainer(ctx, "test-container-id", false)
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestGetAppContainerInfo_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.GetAppContainerInfo(ctx, "test-container-id")
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestGetAppContainerLogs_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.GetAppContainerLogs(ctx, "test-container-id", "100", "")
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestStreamAppContainerLogs_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.StreamAppContainerLogs(ctx, "test-container-id", true)
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestExecInAppContainer_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.ExecInAppContainer(ctx, "test-container-id", []string{"ls", "-la"})
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestListAppContainers_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.ListAppContainers(ctx)
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestListUserAppContainers_NoDockerClient(t *testing.T) {
	logger := testAppLogger()
	manager := NewAppContainerManager(nil, logger, "")

	ctx := context.Background()
	_, err := manager.ListUserAppContainers(ctx, uuid.New())
	if err == nil {
		t.Error("expected error when docker client is nil")
	}
}

func TestPtrHelper(t *testing.T) {
	// Test with int
	intVal := 42
	intPtr := ptr(intVal)
	if *intPtr != 42 {
		t.Errorf("ptr(42) = %d, expected 42", *intPtr)
	}

	// Test with string
	strVal := "hello"
	strPtr := ptr(strVal)
	if *strPtr != "hello" {
		t.Errorf("ptr(hello) = %s, expected hello", *strPtr)
	}

	// Test with int64
	int64Val := int64(100)
	int64Ptr := ptr(int64Val)
	if *int64Ptr != 100 {
		t.Errorf("ptr(100) = %d, expected 100", *int64Ptr)
	}
}

func TestAppContainerConfig_DefaultResources(t *testing.T) {
	// Verify default resource constants
	if appDefaultMemoryLimit != 512*1024*1024 {
		t.Errorf("appDefaultMemoryLimit = %d, expected %d", appDefaultMemoryLimit, 512*1024*1024)
	}
	if appDefaultCPUQuota != 50000 {
		t.Errorf("appDefaultCPUQuota = %d, expected 50000", appDefaultCPUQuota)
	}
	if appDefaultPidsLimit != 100 {
		t.Errorf("appDefaultPidsLimit = %d, expected 100", appDefaultPidsLimit)
	}
}

func TestLabelConstants(t *testing.T) {
	// Verify app container label constants (shared ones are in volume.go)
	if labelAppID != "app_id" {
		t.Errorf("labelAppID = %s, expected 'app_id'", labelAppID)
	}
	if labelAppName != "app_name" {
		t.Errorf("labelAppName = %s, expected 'app_name'", labelAppName)
	}
	if labelValueBusinessOS != "businessos" {
		t.Errorf("labelValueBusinessOS = %s, expected 'businessos'", labelValueBusinessOS)
	}
	if labelValueSandboxApp != "sandbox-app" {
		t.Errorf("labelValueSandboxApp = %s, expected 'sandbox-app'", labelValueSandboxApp)
	}
}
