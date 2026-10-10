package appgen

import (
	"context"
	"os"
	"testing"
	"time"
)

func skipAppgenIntegrationUnlessEnabled(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("RUN_APPGEN_INTEGRATION") != "true" {
		t.Skip("Skipping appgen integration test; set RUN_APPGEN_INTEGRATION=true with Claude agent access")
	}
}

func TestOrchestratorCreatePlan(t *testing.T) {
	// Requires ANTHROPIC_API_KEY to be set
	skipAppgenIntegrationUnlessEnabled(t)

	orchestrator := NewOrchestrator(nil) // nil pool = in-memory learning storage

	req := AppRequest{
		AppName:     "Test Todo App",
		Description: "A simple todo application with create, read, update, delete functionality",
		Features:    []string{"user authentication", "task management"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plan, err := orchestrator.CreatePlan(ctx, req)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// Verify plan structure
	if plan == nil {
		t.Fatal("Plan is nil")
	}

	if len(plan.Tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(plan.Tasks))
	}

	// Verify task types
	expectedTypes := map[AgentType]bool{
		AgentFrontend: false,
		AgentBackend:  false,
		AgentDatabase: false,
		AgentTest:     false,
	}

	for _, task := range plan.Tasks {
		if _, exists := expectedTypes[task.Type]; !exists {
			t.Errorf("Unexpected task type: %s", task.Type)
		}
		expectedTypes[task.Type] = true

		if task.ID == "" {
			t.Error("Task ID is empty")
		}
		if task.Description == "" {
			t.Error("Task description is empty")
		}
		if task.Workspace == "" {
			t.Error("Task workspace is empty")
		}
	}

	// Verify all task types are present
	for taskType, found := range expectedTypes {
		if !found {
			t.Errorf("Missing task type: %s", taskType)
		}
	}
}

func TestOrchestratorExecute(t *testing.T) {
	// Requires ANTHROPIC_API_KEY to be set
	skipAppgenIntegrationUnlessEnabled(t)

	orchestrator := NewOrchestrator(nil)

	// Create a simple plan manually (skip CreatePlan to save API calls)
	plan := &Plan{
		Architecture: "Test architecture",
		Tasks: []Task{
			{
				ID:          "test-frontend",
				Type:        AgentFrontend,
				Description: "Create a simple button component",
				Workspace:   "/workspace/frontend/",
				Priority:    1,
			},
		},
		CreatedAt: time.Now(),
	}

	// Track progress events
	var progressEvents []ProgressEvent
	orchestrator.SetProgressCallback(func(event ProgressEvent) {
		progressEvents = append(progressEvents, event)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := orchestrator.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Result is nil")
	}

	if len(result.Results) != len(plan.Tasks) {
		t.Errorf("Expected %d results, got %d", len(plan.Tasks), len(result.Results))
	}

	// Verify at least some progress events were emitted
	if len(progressEvents) == 0 {
		t.Error("No progress events were emitted")
	}

	// Verify result has output
	for _, agentResult := range result.Results {
		if agentResult.Output == "" {
			t.Errorf("Agent %s produced no output", agentResult.AgentType)
		}
	}
}

func TestCircuitBreakerIntegration(t *testing.T) {
	orchestrator := NewOrchestrator(nil).(*orchestratorImpl)

	// Circuit breaker should start in closed state
	metrics := orchestrator.circuitBreaker.GetMetrics()
	if state, ok := metrics["state"].(string); !ok || state != "closed" {
		t.Errorf("Expected circuit breaker to start in closed state, got %v", state)
	}

	// Test manual reset
	orchestrator.circuitBreaker.Reset()
	metrics = orchestrator.circuitBreaker.GetMetrics()
	if state, ok := metrics["state"].(string); !ok || state != "closed" {
		t.Error("Circuit breaker reset failed")
	}
}

func TestGracefulShutdown(t *testing.T) {
	orchestrator := NewOrchestrator(nil).(*orchestratorImpl)

	// Start a goroutine that tracks orchestrator wg
	orchestrator.wg.Add(1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		orchestrator.wg.Done()
	}()

	// Shutdown should wait for goroutine
	start := time.Now()
	err := orchestrator.Shutdown()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	if duration < 100*time.Millisecond {
		t.Error("Shutdown did not wait for goroutine to complete")
	}

	if duration > 5*time.Second {
		t.Error("Shutdown took too long (should timeout after 30s)")
	}
}

func TestShutdownCancelsNewExecutions(t *testing.T) {
	orchestrator := NewOrchestrator(nil).(*orchestratorImpl)

	// Shutdown the orchestrator
	orchestrator.cancel()

	// Try to execute a plan (should fail immediately)
	plan := &Plan{
		Tasks: []Task{
			{ID: "test-1", Type: AgentFrontend, Description: "Test", Workspace: "/test/", Priority: 1},
		},
	}

	ctx := context.Background()
	_, err := orchestrator.Execute(ctx, plan)

	if err == nil {
		t.Error("Expected Execute to fail after shutdown, but it succeeded")
	}

	if err.Error() != "orchestrator is shutting down" {
		t.Errorf("Expected shutdown error, got: %v", err)
	}
}
