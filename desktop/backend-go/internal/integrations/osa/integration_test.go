// +build integration

package osa

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests require OSA to be running on localhost:8089
// Run with: go test -tags=integration ./internal/integrations/osa/...

func getTestConfig(t *testing.T) *Config {
	baseURL := os.Getenv("OSA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8089"
	}

	secret := os.Getenv("OSA_SHARED_SECRET")
	if secret == "" {
		// Use a test secret - in real integration tests, this should match OSA's secret
		secret = "test-secret-key-min-32-bytes-long"
	}

	return &Config{
		BaseURL:      baseURL,
		SharedSecret: secret,
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryDelay:   2 * time.Second,
	}
}

func TestIntegration_HealthCheck(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.HealthCheck(ctx)

	if err != nil {
		t.Skipf("OSA not available at %s: %v", config.BaseURL, err)
		return
	}

	assert.NotNil(t, health)
	assert.NotEmpty(t, health.Status)
	t.Logf("OSA Health: %s (version: %s)", health.Status, health.Version)
}

func TestIntegration_GenerateApp(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	// First check if OSA is available
	ctx := context.Background()
	if _, err := client.HealthCheck(ctx); err != nil {
		t.Skipf("OSA not available: %v", err)
		return
	}

	// Create test request
	userID := uuid.New()
	workspaceID := uuid.New()

	req := &AppGenerationRequest{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        "Integration Test App",
		Description: "This is a test application generated during integration testing",
		Type:        "module",
		Parameters: map[string]interface{}{
			"test": true,
		},
	}

	// Generate app
	resp, err := client.GenerateApp(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AppID)
	assert.Equal(t, workspaceID.String(), resp.WorkspaceID)
	t.Logf("App generation started: %s (status: %s)", resp.AppID, resp.Status)

	// Poll status for a bit
	appID := resp.AppID
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		status, err := client.GetAppStatus(ctx, appID, userID)
		if err != nil {
			t.Logf("Status check failed: %v", err)
			continue
		}

		t.Logf("App status: %s (progress: %.0f%%)", status.Status, status.Progress*100)

		if status.Status == "completed" || status.Status == "failed" {
			break
		}
	}
}

func TestIntegration_Orchestrate(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Check if OSA is available
	ctx := context.Background()
	if _, err := client.HealthCheck(ctx); err != nil {
		t.Skipf("OSA not available: %v", err)
		return
	}

	// Create orchestration request
	userID := uuid.New()
	workspaceID := uuid.New()

	req := &OrchestrateRequest{
		UserID:      userID,
		Input:       "Analyze the feasibility of creating a simple todo list application",
		Phase:       "analysis",
		WorkspaceID: workspaceID,
		Context: map[string]interface{}{
			"test": true,
		},
	}

	// Execute orchestration
	resp, err := client.Orchestrate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	t.Logf("Orchestration success: %v", resp.Success)
	t.Logf("Execution time: %dms", resp.ExecutionTime)
	t.Logf("Agents used: %v", resp.AgentsUsed)

	if resp.Success {
		assert.NotEmpty(t, resp.Output)
		assert.Greater(t, len(resp.AgentsUsed), 0)
		t.Logf("Output length: %d chars", len(resp.Output))
	}
}

func TestIntegration_GetWorkspaces(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Check if OSA is available
	ctx := context.Background()
	if _, err := client.HealthCheck(ctx); err != nil {
		t.Skipf("OSA not available: %v", err)
		return
	}

	// Get workspaces
	userID := uuid.New()
	workspaces, err := client.GetWorkspaces(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, workspaces)
	t.Logf("Found %d workspaces", workspaces.Total)

	if workspaces.Total > 0 {
		for i, ws := range workspaces.Workspaces {
			t.Logf("Workspace %d: %s (%s)", i+1, ws.Name, ws.ID)
		}
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Check if OSA is available
	ctx := context.Background()
	if _, err := client.HealthCheck(ctx); err != nil {
		t.Skipf("OSA not available: %v", err)
		return
	}

	// Make multiple concurrent health checks
	concurrency := 5
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			health, err := client.HealthCheck(ctx)
			if err != nil {
				done <- err
				return
			}
			t.Logf("Concurrent request %d: %s", id, health.Status)
			done <- nil
		}(i)
	}

	// Wait for all requests
	for i := 0; i < concurrency; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestIntegration_Timeout(t *testing.T) {
	config := getTestConfig(t)
	config.Timeout = 1 * time.Millisecond // Very short timeout

	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.HealthCheck(ctx)

	// Should timeout
	assert.Error(t, err)
	t.Logf("Expected timeout error: %v", err)
}

func TestIntegration_ContextCancellation(t *testing.T) {
	config := getTestConfig(t)
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Create context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.HealthCheck(ctx)

	// Should fail due to cancelled context
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
