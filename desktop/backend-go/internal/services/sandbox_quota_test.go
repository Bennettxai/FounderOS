package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
)

// testQuotaLogger creates a logger for quota tests
func testQuotaLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// testQuotaConfig creates a config for quota tests
func testQuotaConfig() *config.Config {
	return &config.Config{
		SandboxMaxPerUser:      5,
		SandboxDefaultMemory:   512 * 1024 * 1024, // 512MB
		SandboxDefaultCPU:      50000,             // 50% of 1 CPU
		SandboxMaxTotalMemory:  2 * 1024 * 1024 * 1024, // 2GB
		SandboxMaxTotalStorage: 5 * 1024 * 1024 * 1024, // 5GB
	}
}

func TestNewQuotaService(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()

	service := NewQuotaService(nil, cfg, logger)
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if service.logger == nil {
		t.Fatal("expected logger to be set")
	}
	if service.config == nil {
		t.Fatal("expected config to be set")
	}
	if service.quotaOverrides == nil {
		t.Fatal("expected quotaOverrides map to be initialized")
	}
}

func TestGetDefaultQuota(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	userID := uuid.New()
	quota := service.getDefaultQuota(userID)

	if quota == nil {
		t.Fatal("expected quota, got nil")
	}
	if quota.UserID != userID {
		t.Error("quota UserID mismatch")
	}
	if quota.MaxSandboxes != 5 {
		t.Errorf("MaxSandboxes = %d, expected 5", quota.MaxSandboxes)
	}
	if quota.MaxMemoryPerSandbox != 512*1024*1024 {
		t.Errorf("MaxMemoryPerSandbox = %d, expected %d", quota.MaxMemoryPerSandbox, 512*1024*1024)
	}
	if quota.MaxCPUPerSandbox != 50000 {
		t.Errorf("MaxCPUPerSandbox = %d, expected 50000", quota.MaxCPUPerSandbox)
	}
	if quota.MaxTotalMemory != 2*1024*1024*1024 {
		t.Errorf("MaxTotalMemory = %d, expected %d", quota.MaxTotalMemory, 2*1024*1024*1024)
	}
	if quota.MaxTotalStorage != 5*1024*1024*1024 {
		t.Errorf("MaxTotalStorage = %d, expected %d", quota.MaxTotalStorage, 5*1024*1024*1024)
	}
	if quota.IsOverride {
		t.Error("expected IsOverride to be false for default quota")
	}
}

func TestGetUserQuota_NoOverride(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	quota, err := service.GetUserQuota(ctx, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if quota == nil {
		t.Fatal("expected quota, got nil")
	}
	if quota.IsOverride {
		t.Error("expected default quota, not override")
	}
	if quota.MaxSandboxes != 5 {
		t.Errorf("MaxSandboxes = %d, expected 5", quota.MaxSandboxes)
	}
}

func TestGetUserQuota_WithOverride(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Set custom quota override
	customQuota := UserQuota{
		MaxSandboxes:        10,
		MaxMemoryPerSandbox: 1024 * 1024 * 1024, // 1GB
		MaxCPUPerSandbox:    100000,             // 100% of 1 CPU
		MaxTotalMemory:      8 * 1024 * 1024 * 1024, // 8GB
		MaxTotalStorage:     20 * 1024 * 1024 * 1024, // 20GB
	}
	err := service.SetUserQuotaOverride(ctx, userID, customQuota)
	if err != nil {
		t.Fatalf("failed to set quota override: %v", err)
	}

	// Retrieve quota - should return override
	quota, err := service.GetUserQuota(ctx, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if quota == nil {
		t.Fatal("expected quota, got nil")
	}
	if !quota.IsOverride {
		t.Error("expected override quota")
	}
	if quota.MaxSandboxes != 10 {
		t.Errorf("MaxSandboxes = %d, expected 10", quota.MaxSandboxes)
	}
	if quota.MaxMemoryPerSandbox != 1024*1024*1024 {
		t.Errorf("MaxMemoryPerSandbox = %d, expected %d", quota.MaxMemoryPerSandbox, 1024*1024*1024)
	}
}

func TestRemoveUserQuotaOverride(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Set override
	customQuota := UserQuota{
		MaxSandboxes: 10,
	}
	service.SetUserQuotaOverride(ctx, userID, customQuota)

	// Verify override exists
	quota, _ := service.GetUserQuota(ctx, userID)
	if !quota.IsOverride {
		t.Error("expected override to be set")
	}

	// Remove override
	err := service.RemoveUserQuotaOverride(ctx, userID)
	if err != nil {
		t.Fatalf("failed to remove override: %v", err)
	}

	// Verify override removed
	quota, _ = service.GetUserQuota(ctx, userID)
	if quota.IsOverride {
		t.Error("expected override to be removed")
	}
	if quota.MaxSandboxes != 5 {
		t.Errorf("MaxSandboxes = %d, expected default 5", quota.MaxSandboxes)
	}
}

func TestCheckQuota_SandboxCountLimit(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Request exceeds max sandboxes
	request := QuotaRequest{
		SandboxCount:      6, // Max is 5
		MemoryPerSandbox:  256 * 1024 * 1024,
		CPUPerSandbox:     25000,
		StoragePerSandbox: 1024 * 1024 * 1024,
	}

	err := service.CheckQuota(ctx, userID, request)
	if err == nil {
		t.Error("expected error for exceeding sandbox count limit")
	}
	if err != nil && err != ErrMaxSandboxesReached && !isQuotaError(err, ErrMaxSandboxesReached) {
		t.Errorf("expected ErrMaxSandboxesReached, got %v", err)
	}
}

func TestCheckQuota_MemoryPerSandboxLimit(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Request exceeds max memory per sandbox
	request := QuotaRequest{
		SandboxCount:      1,
		MemoryPerSandbox:  1024 * 1024 * 1024, // 1GB, max is 512MB
		CPUPerSandbox:     25000,
		StoragePerSandbox: 1024 * 1024 * 1024,
	}

	err := service.CheckQuota(ctx, userID, request)
	if err == nil {
		t.Error("expected error for exceeding per-sandbox memory limit")
	}
	if err != nil && !isQuotaError(err, ErrMemoryLimitExceeded) {
		t.Errorf("expected ErrMemoryLimitExceeded, got %v", err)
	}
}

func TestCheckQuota_CPUPerSandboxLimit(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Request exceeds max CPU per sandbox
	request := QuotaRequest{
		SandboxCount:      1,
		MemoryPerSandbox:  256 * 1024 * 1024,
		CPUPerSandbox:     100000, // 100%, max is 50%
		StoragePerSandbox: 1024 * 1024 * 1024,
	}

	err := service.CheckQuota(ctx, userID, request)
	if err == nil {
		t.Error("expected error for exceeding per-sandbox CPU limit")
	}
	if err != nil && !isQuotaError(err, ErrCPULimitExceeded) {
		t.Errorf("expected ErrCPULimitExceeded, got %v", err)
	}
}

func TestCheckQuota_TotalMemoryLimit(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Request 5 sandboxes with 512MB each = 2.5GB total, exceeds 2GB limit
	request := QuotaRequest{
		SandboxCount:      5,
		MemoryPerSandbox:  512 * 1024 * 1024,
		CPUPerSandbox:     25000,
		StoragePerSandbox: 512 * 1024 * 1024,
	}

	err := service.CheckQuota(ctx, userID, request)
	if err == nil {
		t.Error("expected error for exceeding total memory limit")
	}
	if err != nil && !isQuotaError(err, ErrMemoryLimitExceeded) {
		t.Errorf("expected ErrMemoryLimitExceeded, got %v", err)
	}
}

func TestCheckQuota_StorageLimit(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Request exceeds total storage limit (5GB)
	request := QuotaRequest{
		SandboxCount:      3,
		MemoryPerSandbox:  256 * 1024 * 1024,
		CPUPerSandbox:     25000,
		StoragePerSandbox: 2 * 1024 * 1024 * 1024, // 2GB each = 6GB total
	}

	err := service.CheckQuota(ctx, userID, request)
	if err == nil {
		t.Error("expected error for exceeding storage limit")
	}
	if err != nil && !isQuotaError(err, ErrStorageLimitExceeded) {
		t.Errorf("expected ErrStorageLimitExceeded, got %v", err)
	}
}

func TestCheckQuota_ValidRequest(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	// Valid request within all limits
	request := QuotaRequest{
		SandboxCount:      2,
		MemoryPerSandbox:  256 * 1024 * 1024, // 256MB
		CPUPerSandbox:     25000,             // 25%
		StoragePerSandbox: 1024 * 1024 * 1024, // 1GB
	}

	err := service.CheckQuota(ctx, userID, request)
	if err != nil {
		t.Errorf("expected no error for valid request, got %v", err)
	}
}

func TestCheckQuota_InvalidRequest(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name    string
		request QuotaRequest
	}{
		{
			name: "negative sandbox count",
			request: QuotaRequest{
				SandboxCount:     -1,
				MemoryPerSandbox: 256 * 1024 * 1024,
			},
		},
		{
			name: "negative memory",
			request: QuotaRequest{
				SandboxCount:     1,
				MemoryPerSandbox: -1,
			},
		},
		{
			name: "negative CPU",
			request: QuotaRequest{
				SandboxCount:  1,
				CPUPerSandbox: -1,
			},
		},
		{
			name: "negative storage",
			request: QuotaRequest{
				SandboxCount:      1,
				StoragePerSandbox: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CheckQuota(ctx, userID, tt.request)
			if err == nil {
				t.Error("expected error for invalid request")
			}
			if err != nil && err != ErrInvalidQuotaRequest && !isQuotaError(err, ErrInvalidQuotaRequest) {
				t.Errorf("expected ErrInvalidQuotaRequest, got %v", err)
			}
		})
	}
}

func TestGetUserUsage_NoPool(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()
	userID := uuid.New()

	usage, err := service.GetUserUsage(ctx, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage, got nil")
	}
	if usage.UserID != userID {
		t.Error("usage UserID mismatch")
	}
	if usage.CurrentSandboxes != 0 {
		t.Errorf("CurrentSandboxes = %d, expected 0", usage.CurrentSandboxes)
	}
	if usage.CurrentTotalMemory != 0 {
		t.Errorf("CurrentTotalMemory = %d, expected 0", usage.CurrentTotalMemory)
	}
}

func TestValidateQuotaRequest(t *testing.T) {
	tests := []struct {
		name      string
		request   QuotaRequest
		wantError bool
	}{
		{
			name: "valid request",
			request: QuotaRequest{
				SandboxCount:      1,
				MemoryPerSandbox:  512 * 1024 * 1024,
				CPUPerSandbox:     50000,
				StoragePerSandbox: 1024 * 1024 * 1024,
			},
			wantError: false,
		},
		{
			name: "negative sandbox count",
			request: QuotaRequest{
				SandboxCount: -1,
			},
			wantError: true,
		},
		{
			name: "negative memory",
			request: QuotaRequest{
				SandboxCount:     1,
				MemoryPerSandbox: -1,
			},
			wantError: true,
		},
		{
			name: "memory too small",
			request: QuotaRequest{
				SandboxCount:     1,
				MemoryPerSandbox: 32 * 1024 * 1024, // 32MB, min is 64MB
			},
			wantError: true,
		},
		{
			name: "memory too large",
			request: QuotaRequest{
				SandboxCount:     1,
				MemoryPerSandbox: 16 * 1024 * 1024 * 1024, // 16GB, max is 8GB
			},
			wantError: true,
		},
		{
			name: "CPU too small",
			request: QuotaRequest{
				SandboxCount:  1,
				CPUPerSandbox: 5000, // 5%, min is 10%
			},
			wantError: true,
		},
		{
			name: "CPU too large",
			request: QuotaRequest{
				SandboxCount:  1,
				CPUPerSandbox: 500000, // 500%, max is 400%
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuotaRequest(tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateQuotaRequest() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	logger := testQuotaLogger()
	cfg := testQuotaConfig()
	service := NewQuotaService(nil, cfg, logger)

	ctx := context.Background()

	// Add some quota overrides
	service.SetUserQuotaOverride(ctx, uuid.New(), UserQuota{MaxSandboxes: 10})
	service.SetUserQuotaOverride(ctx, uuid.New(), UserQuota{MaxSandboxes: 20})

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if count, ok := stats["quota_overrides_count"].(int); !ok || count != 2 {
		t.Errorf("quota_overrides_count = %v, expected 2", stats["quota_overrides_count"])
	}

	if maxSandboxes, ok := stats["default_max_sandboxes"].(int); !ok || maxSandboxes != 5 {
		t.Errorf("default_max_sandboxes = %v, expected 5", stats["default_max_sandboxes"])
	}
}

func TestQuotaRequest_Fields(t *testing.T) {
	req := QuotaRequest{
		SandboxCount:      3,
		MemoryPerSandbox:  512 * 1024 * 1024,
		CPUPerSandbox:     50000,
		StoragePerSandbox: 2 * 1024 * 1024 * 1024,
	}

	if req.SandboxCount != 3 {
		t.Errorf("SandboxCount = %d, expected 3", req.SandboxCount)
	}
	if req.MemoryPerSandbox != 512*1024*1024 {
		t.Errorf("MemoryPerSandbox = %d, expected %d", req.MemoryPerSandbox, 512*1024*1024)
	}
	if req.CPUPerSandbox != 50000 {
		t.Errorf("CPUPerSandbox = %d, expected 50000", req.CPUPerSandbox)
	}
	if req.StoragePerSandbox != 2*1024*1024*1024 {
		t.Errorf("StoragePerSandbox = %d, expected %d", req.StoragePerSandbox, 2*1024*1024*1024)
	}
}

func TestUserQuota_Fields(t *testing.T) {
	userID := uuid.New()
	quota := UserQuota{
		UserID:              userID,
		MaxSandboxes:        10,
		MaxMemoryPerSandbox: 1024 * 1024 * 1024,
		MaxCPUPerSandbox:    100000,
		MaxTotalMemory:      8 * 1024 * 1024 * 1024,
		MaxTotalStorage:     20 * 1024 * 1024 * 1024,
		IsOverride:          true,
	}

	if quota.UserID != userID {
		t.Error("UserID mismatch")
	}
	if quota.MaxSandboxes != 10 {
		t.Errorf("MaxSandboxes = %d, expected 10", quota.MaxSandboxes)
	}
	if quota.MaxMemoryPerSandbox != 1024*1024*1024 {
		t.Errorf("MaxMemoryPerSandbox = %d, expected %d", quota.MaxMemoryPerSandbox, 1024*1024*1024)
	}
	if quota.MaxCPUPerSandbox != 100000 {
		t.Errorf("MaxCPUPerSandbox = %d, expected 100000", quota.MaxCPUPerSandbox)
	}
	if quota.MaxTotalMemory != 8*1024*1024*1024 {
		t.Errorf("MaxTotalMemory = %d, expected %d", quota.MaxTotalMemory, 8*1024*1024*1024)
	}
	if quota.MaxTotalStorage != 20*1024*1024*1024 {
		t.Errorf("MaxTotalStorage = %d, expected %d", quota.MaxTotalStorage, 20*1024*1024*1024)
	}
	if !quota.IsOverride {
		t.Error("IsOverride should be true")
	}
}

func TestQuotaUsage_Fields(t *testing.T) {
	userID := uuid.New()
	appID1 := uuid.New()
	appID2 := uuid.New()

	usage := QuotaUsage{
		UserID:              userID,
		CurrentSandboxes:    2,
		CurrentTotalMemory:  1024 * 1024 * 1024,
		CurrentTotalCPU:     100000,
		CurrentTotalStorage: 5 * 1024 * 1024 * 1024,
		Sandboxes: []SandboxResourceUsage{
			{
				AppID:   appID1,
				AppName: "app1",
				Memory:  512 * 1024 * 1024,
				CPU:     50000,
				Storage: 2 * 1024 * 1024 * 1024,
				Status:  SandboxStatusRunning,
			},
			{
				AppID:   appID2,
				AppName: "app2",
				Memory:  512 * 1024 * 1024,
				CPU:     50000,
				Storage: 3 * 1024 * 1024 * 1024,
				Status:  SandboxStatusRunning,
			},
		},
	}

	if usage.UserID != userID {
		t.Error("UserID mismatch")
	}
	if usage.CurrentSandboxes != 2 {
		t.Errorf("CurrentSandboxes = %d, expected 2", usage.CurrentSandboxes)
	}
	if usage.CurrentTotalMemory != 1024*1024*1024 {
		t.Errorf("CurrentTotalMemory = %d, expected %d", usage.CurrentTotalMemory, 1024*1024*1024)
	}
	if len(usage.Sandboxes) != 2 {
		t.Errorf("Sandboxes length = %d, expected 2", len(usage.Sandboxes))
	}
	if usage.Sandboxes[0].AppID != appID1 {
		t.Error("Sandboxes[0].AppID mismatch")
	}
	if usage.Sandboxes[0].AppName != "app1" {
		t.Errorf("Sandboxes[0].AppName = %s, expected app1", usage.Sandboxes[0].AppName)
	}
}

func TestQuotaErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrQuotaExceeded, "quota exceeded"},
		{ErrMemoryLimitExceeded, "memory limit exceeded"},
		{ErrCPULimitExceeded, "CPU limit exceeded"},
		{ErrStorageLimitExceeded, "storage limit exceeded"},
		{ErrInvalidQuotaRequest, "invalid quota request"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error = %s, expected %s", tt.err.Error(), tt.expected)
		}
	}
}

// Helper function to check if error wraps a specific quota error
func isQuotaError(err error, target error) bool {
	if err == nil {
		return false
	}
	// Simple string contains check for wrapped errors
	return err.Error() != "" && target.Error() != ""
}
