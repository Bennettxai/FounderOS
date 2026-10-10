package osa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BaseURL:      "http://localhost:8089",
				SharedSecret: "test-secret-key-min-32-bytes-long",
				Timeout:      30 * time.Second,
				MaxRetries:   3,
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			config: &Config{
				SharedSecret: "test-secret",
				Timeout:      30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "missing shared secret",
			config: &Config{
				BaseURL: "http://localhost:8089",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": "1.0.0",
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	health, err := client.HealthCheck(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "1.0.0", health.Version)
}

func TestClient_GenerateApp(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New().String()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/generate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Test App", req["name"])
		assert.Equal(t, userID.String(), req["user_id"])
		assert.Equal(t, workspaceID.String(), req["workspace_id"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"app_id":       appID,
			"status":       "processing",
			"workspace_id": workspaceID.String(),
			"created_at":   time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	req := &AppGenerationRequest{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        "Test App",
		Description: "A test application",
		Type:        "full-stack",
	}

	resp, err := client.GenerateApp(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, appID, resp.AppID)
	assert.Equal(t, "processing", resp.Status)
	assert.Equal(t, workspaceID.String(), resp.WorkspaceID)
}

func TestClient_GetAppStatus(t *testing.T) {
	appID := uuid.New().String()
	userID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/apps/"+appID+"/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"app_id":       appID,
			"status":       "completed",
			"progress":     1.0,
			"current_step": "Done",
			"updated_at":   time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	status, err := client.GetAppStatus(ctx, appID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, appID, status.AppID)
	assert.Equal(t, "completed", status.Status)
	assert.Equal(t, 1.0, status.Progress)
}

func TestClient_Orchestrate(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orchestrate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), req["user_id"])
		assert.Equal(t, "Build me a task manager", req["input"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id":      "sess-123",
			"success":         true,
			"output":          "Task manager created successfully",
			"agents_used":     []string{"StrategyAgent", "ArchitectAgent", "DevelopmentAgent"},
			"execution_ms":    5000,
			"iteration_count": 3,
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      10 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	req := &OrchestrateRequest{
		UserID:      userID,
		Input:       "Build me a task manager",
		WorkspaceID: workspaceID,
	}

	resp, err := client.Orchestrate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Task manager created successfully", resp.Output)
	assert.Len(t, resp.AgentsUsed, 3)
	assert.Equal(t, int64(5000), resp.ExecutionTime)
}

func TestClient_GetWorkspaces(t *testing.T) {
	userID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/workspaces", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workspaces": []map[string]interface{}{
				{
					"id":          uuid.New().String(),
					"name":        "Workspace 1",
					"description": "First workspace",
					"owner_id":    userID.String(),
					"created_at":  time.Now().UTC().Format(time.RFC3339),
					"updated_at":  time.Now().UTC().Format(time.RFC3339),
				},
				{
					"id":          uuid.New().String(),
					"name":        "Workspace 2",
					"description": "Second workspace",
					"owner_id":    userID.String(),
					"created_at":  time.Now().UTC().Format(time.RFC3339),
					"updated_at":  time.Now().UTC().Format(time.RFC3339),
				},
			},
			"total": 2,
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	workspaces, err := client.GetWorkspaces(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, workspaces)
	assert.Equal(t, 2, workspaces.Total)
	assert.Len(t, workspaces.Workspaces, 2)
	assert.Equal(t, "Workspace 1", workspaces.Workspaces[0].Name)
	assert.Equal(t, "Workspace 2", workspaces.Workspaces[1].Name)
}

func TestClient_ErrorHandling(t *testing.T) {
	userID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Missing required field: name",
			"details": "Missing required field: name",
		})
	}))
	defer server.Close()

	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	req := &AppGenerationRequest{
		UserID:      userID,
		WorkspaceID: uuid.New(),
		Type:        "full-stack",
		// Missing Name field
	}

	resp, err := client.GenerateApp(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Missing required field: name")
}

func TestClient_HTTPErrorNoRetry(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// SDK resilience is disabled in NewClient; the BOS ResilientClient layer
	// handles retries. So a single call to the base Client should hit the
	// server exactly once per call, regardless of MaxRetries config.
	config := &Config{
		BaseURL:      server.URL,
		SharedSecret: "test-secret-key-min-32-bytes-long",
		Timeout:      5 * time.Second,
		MaxRetries:   3,
		RetryDelay:   100 * time.Millisecond,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Equal(t, 1, attemptCount, "SDK resilience is disabled; base client makes exactly one attempt")
}
