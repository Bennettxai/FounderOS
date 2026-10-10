package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// mockOSAClient implements a mock OSA client for testing
type mockOSAClient struct {
	orchestrateFunc func(ctx context.Context, req *osa.OrchestrateRequest) (*osa.OrchestrateResponse, error)
	healthCheckFunc func(ctx context.Context) (*osa.HealthResponse, error)
}

func (m *mockOSAClient) Orchestrate(ctx context.Context, req *osa.OrchestrateRequest) (*osa.OrchestrateResponse, error) {
	if m.orchestrateFunc != nil {
		return m.orchestrateFunc(ctx, req)
	}
	return &osa.OrchestrateResponse{
		Success:       true,
		Output:        "mock output",
		AgentsUsed:    []string{"agent1", "agent2"},
		ExecutionTime: 1000,
	}, nil
}

func (m *mockOSAClient) HealthCheck(ctx context.Context) (*osa.HealthResponse, error) {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return &osa.HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
	}, nil
}

func newTestOsaSkill(t *testing.T, handler http.HandlerFunc) (*OsaSkill, func()) {
	t.Helper()

	if handler == nil {
		handler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/v1/orchestrate":
				if r.Method != http.MethodPost {
					t.Errorf("expected POST /api/v1/orchestrate, got %s", r.Method)
				}
				_ = json.NewDecoder(r.Body).Decode(&map[string]interface{}{})
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"success":      true,
					"output":       "mock output",
					"agents_used":  []string{"agent1", "agent2"},
					"execution_ms": 1000,
					"next_step":    "review",
				})
			case "/health":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "healthy",
					"version": "1.0.0",
				})
			default:
				http.NotFound(w, r)
			}
		}
	}

	server := httptest.NewServer(handler)
	config := osa.DefaultResilientClientConfig()
	config.OSAConfig.BaseURL = server.URL
	config.OSAConfig.SharedSecret = "test-secret"
	config.OSAConfig.Timeout = 500 * time.Millisecond
	config.CircuitBreakerConfig.MaxRetryTime = 0
	config.EnableAutoRecovery = false
	config.QueueSize = 0

	client, err := osa.NewResilientClient(config)
	if err != nil {
		server.Close()
		t.Fatalf("Failed to create resilient client: %v", err)
	}

	return NewOsaSkill(client), func() {
		_ = client.Close()
		server.Close()
	}
}

func TestOsaSkill_Metadata(t *testing.T) {
	skill, cleanup := newTestOsaSkill(t, nil)
	defer cleanup()

	// Test Name
	name := skill.Name()
	if name != "osa_orchestrate" {
		t.Errorf("Name() = %q, want %q", name, "osa_orchestrate")
	}

	// Test Description
	desc := skill.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	// Test Schema
	schema := skill.Schema()
	if schema == nil {
		t.Error("Schema() returned nil")
	}
	if schema.InputSchema == nil {
		t.Error("Schema InputSchema is nil")
	}
	if schema.OutputSchema == nil {
		t.Error("Schema OutputSchema is nil")
	}
	if len(schema.Examples) == 0 {
		t.Error("Schema has no examples")
	}
}

func TestOsaSkill_Execute_ValidParams(t *testing.T) {
	skill, cleanup := newTestOsaSkill(t, nil)
	defer cleanup()

	// Test with valid parameters
	userID := uuid.New()
	params := map[string]interface{}{
		"user_id": userID.String(),
		"input":   "Create a simple CRUD app",
	}

	ctx := context.Background()

	result, err := skill.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Execute() result type = %T, want map[string]interface{}", result)
	}
	if resultMap["success"] != true {
		t.Errorf("success = %v, want true", resultMap["success"])
	}
	if resultMap["output"] != "mock output" {
		t.Errorf("output = %v, want mock output", resultMap["output"])
	}
}

func TestOsaSkill_Execute_InvalidParams(t *testing.T) {
	skill, cleanup := newTestOsaSkill(t, nil)
	defer cleanup()
	ctx := context.Background()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError bool
	}{
		{
			name:      "missing user_id",
			params:    map[string]interface{}{"input": "test"},
			wantError: true,
		},
		{
			name:      "missing input",
			params:    map[string]interface{}{"user_id": uuid.New().String()},
			wantError: true,
		},
		{
			name: "invalid user_id format",
			params: map[string]interface{}{
				"user_id": "not-a-uuid",
				"input":   "test",
			},
			wantError: true,
		},
		{
			name: "user_id not a string",
			params: map[string]interface{}{
				"user_id": 12345,
				"input":   "test",
			},
			wantError: true,
		},
		{
			name: "input not a string",
			params: map[string]interface{}{
				"user_id": uuid.New().String(),
				"input":   12345,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := skill.Execute(ctx, tt.params)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestOsaSkill_Execute_OptionalParams(t *testing.T) {
	skill, cleanup := newTestOsaSkill(t, nil)
	defer cleanup()
	ctx := context.Background()

	// Test with optional parameters
	userID := uuid.New()
	workspaceID := uuid.New()

	params := map[string]interface{}{
		"user_id":      userID.String(),
		"input":        "Create a simple app",
		"workspace_id": workspaceID.String(),
		"phase":        "analysis",
		"context": map[string]interface{}{
			"key": "value",
		},
	}

	if _, err := skill.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestOsaSkill_Execute_InvalidWorkspaceID(t *testing.T) {
	skill, cleanup := newTestOsaSkill(t, nil)
	defer cleanup()
	ctx := context.Background()

	params := map[string]interface{}{
		"user_id":      uuid.New().String(),
		"input":        "test",
		"workspace_id": "not-a-uuid",
	}

	_, err := skill.Execute(ctx, params)
	if err == nil {
		t.Error("Expected error for invalid workspace_id")
	}
}
