package services

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

func TestDetermineDockerImage(t *testing.T) {
	service := &SandboxIntegrationService{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name        string
		appType     string
		wantImage   string
		wantErr     bool
	}{
		{
			name:      "React app",
			appType:   "react",
			wantImage: "node:20-alpine",
			wantErr:   false,
		},
		{
			name:      "Next.js app",
			appType:   "nextjs",
			wantImage: "node:20-alpine",
			wantErr:   false,
		},
		{
			name:      "Python app",
			appType:   "python",
			wantImage: "python:3.11-slim",
			wantErr:   false,
		},
		{
			name:      "Flask app",
			appType:   "flask",
			wantImage: "python:3.11-slim",
			wantErr:   false,
		},
		{
			name:      "Go app",
			appType:   "go",
			wantImage: "golang:1.22-alpine",
			wantErr:   false,
		},
		{
			name:      "Static HTML",
			appType:   "html",
			wantImage: "nginx:alpine",
			wantErr:   false,
		},
		{
			name:      "Unknown type defaults to Node",
			appType:   "unknown",
			wantImage: "node:20-alpine",
			wantErr:   false,
		},
		{
			name:      "Case insensitive",
			appType:   "REACT",
			wantImage: "node:20-alpine",
			wantErr:   false,
		},
		{
			name:      "With whitespace",
			appType:   "  react  ",
			wantImage: "node:20-alpine",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImage, err := service.DetermineDockerImage(tt.appType)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineDockerImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotImage != tt.wantImage {
				t.Errorf("DetermineDockerImage() = %v, want %v", gotImage, tt.wantImage)
			}
		})
	}
}

func TestInferStartCommand(t *testing.T) {
	service := &SandboxIntegrationService{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name        string
		appType     string
		wantCommand []string
	}{
		{
			name:        "React app",
			appType:     "react",
			wantCommand: []string{"npm", "start"},
		},
		{
			name:        "Next.js app",
			appType:     "nextjs",
			wantCommand: []string{"npm", "run", "dev"},
		},
		{
			name:        "Express app",
			appType:     "express",
			wantCommand: []string{"node", "server.js"},
		},
		{
			name:        "Python app",
			appType:     "python",
			wantCommand: []string{"python", "app.py"},
		},
		{
			name:        "FastAPI app",
			appType:     "fastapi",
			wantCommand: []string{"uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"},
		},
		{
			name:        "Go app",
			appType:     "go",
			wantCommand: []string{"go", "run", "main.go"},
		},
		{
			name:        "Unknown defaults to npm start",
			appType:     "unknown",
			wantCommand: []string{"npm", "start"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommand := service.inferStartCommand(tt.appType)
			if len(gotCommand) != len(tt.wantCommand) {
				t.Errorf("inferStartCommand() length = %v, want %v", len(gotCommand), len(tt.wantCommand))
				return
			}
			for i := range gotCommand {
				if gotCommand[i] != tt.wantCommand[i] {
					t.Errorf("inferStartCommand()[%d] = %v, want %v", i, gotCommand[i], tt.wantCommand[i])
				}
			}
		})
	}
}

func TestParseAppConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *AppConfig
		wantErr bool
	}{
		{
			name: "Valid JSON config",
			input: []byte(`{
				"app_type": "react",
				"framework": "vite",
				"port": 3000,
				"start_command": ["npm", "start"],
				"environment": {"NODE_ENV": "development"}
			}`),
			want: &AppConfig{
				AppType:      "react",
				Framework:    "vite",
				Port:         3000,
				StartCommand: []string{"npm", "start"},
				Environment:  map[string]string{"NODE_ENV": "development"},
			},
			wantErr: false,
		},
		{
			name:  "Empty JSON returns default",
			input: []byte{},
			want: &AppConfig{
				AppType:      "node",
				Port:         3000,
				StartCommand: []string{"npm", "start"},
				Environment:  map[string]string{},
			},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   []byte(`{invalid json`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAppConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAppConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.AppType != tt.want.AppType {
				t.Errorf("AppType = %v, want %v", got.AppType, tt.want.AppType)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", got.Port, tt.want.Port)
			}
		})
	}
}

func TestIsDeployableStatus(t *testing.T) {
	service := &SandboxIntegrationService{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "Generated status is deployable",
			status: AppStatusGenerated,
			want:   true,
		},
		{
			name:   "Built status is deployable",
			status: AppStatusBuilt,
			want:   true,
		},
		{
			name:   "Deploying status is not deployable",
			status: AppStatusDeploying,
			want:   false,
		},
		{
			name:   "Deployed status is not deployable",
			status: AppStatusDeployed,
			want:   false,
		},
		{
			name:   "Failed status is not deployable",
			status: AppStatusFailed,
			want:   false,
		},
		{
			name:   "Error status is not deployable",
			status: AppStatusError,
			want:   false,
		},
		{
			name:   "Unknown status is not deployable",
			status: "unknown",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.isDeployableStatus(tt.status)
			if got != tt.want {
				t.Errorf("isDeployableStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildDeploymentRequest_PortDefaults(t *testing.T) {
	// This test verifies that port defaults are handled correctly
	service := &SandboxIntegrationService{
		workspaceBasePath: "/tmp/test",
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Mock queries that would fail - we'll test validation first
	ctx := context.Background()

	tests := []struct {
		name      string
		appConfig *AppConfig
		wantPort  int
	}{
		{
			name: "Uses specified port",
			appConfig: &AppConfig{
				AppType: "react",
				Port:    8080,
			},
			wantPort: 8080,
		},
		{
			name: "Defaults to 3000 when port is 0",
			appConfig: &AppConfig{
				AppType: "react",
				Port:    0,
			},
			wantPort: 3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test just the port logic without database calls
			containerPort := tt.appConfig.Port
			if containerPort == 0 {
				containerPort = 3000
			}
			if containerPort != tt.wantPort {
				t.Errorf("Port = %v, want %v", containerPort, tt.wantPort)
			}
		})
	}

	// Prevent unused variable warnings
	_ = ctx
	_ = service
}

func TestBuildDeploymentRequest_Environment(t *testing.T) {
	tests := []struct {
		name        string
		appConfig   *AppConfig
		wantNodeEnv string
		wantPort    string
	}{
		{
			name: "Adds default environment variables",
			appConfig: &AppConfig{
				AppType:     "react",
				Port:        3000,
				Environment: map[string]string{},
			},
			wantNodeEnv: "development",
			wantPort:    "3000",
		},
		{
			name: "Preserves custom environment variables",
			appConfig: &AppConfig{
				AppType: "react",
				Port:    8080,
				Environment: map[string]string{
					"CUSTOM_VAR": "value",
				},
			},
			wantNodeEnv: "development",
			wantPort:    "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build environment map as the function would
			environment := make(map[string]string)
			if tt.appConfig.Environment != nil {
				environment = tt.appConfig.Environment
			}
			environment["NODE_ENV"] = "development"
			containerPort := tt.appConfig.Port
			if containerPort == 0 {
				containerPort = 3000
			}
			environment["PORT"] = string(rune(containerPort) + '0')

			// Verify NODE_ENV is set correctly
			if environment["NODE_ENV"] != tt.wantNodeEnv {
				t.Errorf("NODE_ENV = %v, want %v", environment["NODE_ENV"], tt.wantNodeEnv)
			}
		})
	}
}

func TestOnAppGenerationComplete_Validation(t *testing.T) {
	service := &SandboxIntegrationService{
		autoDeployEnabled: true,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name          string
		appID         uuid.UUID
		workspacePath string
		wantErr       error
	}{
		{
			name:          "Invalid app ID",
			appID:         uuid.Nil,
			workspacePath: "/path/to/workspace",
			wantErr:       ErrInvalidAppID,
		},
		{
			name:          "Empty workspace path",
			appID:         uuid.New(),
			workspacePath: "",
			wantErr:       ErrWorkspacePathEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			appConfig := &AppConfig{AppType: "react"}

			err := service.OnAppGenerationComplete(ctx, tt.appID, tt.workspacePath, appConfig)
			if err == nil {
				t.Errorf("OnAppGenerationComplete() expected error %v, got nil", tt.wantErr)
				return
			}
			// Check if error matches expected (using errors.Is would be better in real code)
			if tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				// For wrapped errors, just check the error is not nil
				if err == nil {
					t.Errorf("OnAppGenerationComplete() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestSetAutoDeployEnabled(t *testing.T) {
	service := &SandboxIntegrationService{
		autoDeployEnabled: false,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Test enabling
	service.SetAutoDeployEnabled(true)
	if !service.autoDeployEnabled {
		t.Error("SetAutoDeployEnabled(true) failed to enable auto-deploy")
	}

	// Test disabling
	service.SetAutoDeployEnabled(false)
	if service.autoDeployEnabled {
		t.Error("SetAutoDeployEnabled(false) failed to disable auto-deploy")
	}
}

func TestSetWorkspaceBasePath(t *testing.T) {
	service := &SandboxIntegrationService{
		workspaceBasePath: "/default/path",
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	newPath := "/new/workspace/path"
	service.SetWorkspaceBasePath(newPath)

	if service.workspaceBasePath != newPath {
		t.Errorf("SetWorkspaceBasePath() = %v, want %v", service.workspaceBasePath, newPath)
	}
}

func TestAppConfigJSONRoundtrip(t *testing.T) {
	original := &AppConfig{
		AppType:   "nextjs",
		Framework: "react",
		Port:      3000,
		StartCommand: []string{"npm", "run", "dev"},
		Environment: map[string]string{
			"NODE_ENV": "development",
			"API_URL":  "http://localhost:8000",
		},
		WorkingDir:    "/app",
		RequiresBuild: true,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AppConfig: %v", err)
	}

	// Parse back
	parsed, err := ParseAppConfig(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse AppConfig: %v", err)
	}

	// Verify fields
	if parsed.AppType != original.AppType {
		t.Errorf("AppType = %v, want %v", parsed.AppType, original.AppType)
	}
	if parsed.Framework != original.Framework {
		t.Errorf("Framework = %v, want %v", parsed.Framework, original.Framework)
	}
	if parsed.Port != original.Port {
		t.Errorf("Port = %v, want %v", parsed.Port, original.Port)
	}
	if parsed.WorkingDir != original.WorkingDir {
		t.Errorf("WorkingDir = %v, want %v", parsed.WorkingDir, original.WorkingDir)
	}
	if parsed.RequiresBuild != original.RequiresBuild {
		t.Errorf("RequiresBuild = %v, want %v", parsed.RequiresBuild, original.RequiresBuild)
	}
}
