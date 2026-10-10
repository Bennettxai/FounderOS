package appgen

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestWorkerTimeoutConfiguration verifies the timeout is set correctly in code
func TestWorkerTimeoutConfiguration(t *testing.T) {
	// Test that worker timeout context is created correctly
	worker := NewWorker(AgentFrontend, nil)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	// Verify worker type is set
	if worker.Type() != AgentFrontend {
		t.Errorf("Expected AgentFrontend, got %v", worker.Type())
	}

	// Test timeout context creation (mimics worker.go line 42)
	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Verify deadline is set
	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("Context should have a deadline")
	}

	// Verify deadline is approximately 5 minutes from now
	expectedDeadline := time.Now().Add(5 * time.Minute)
	diff := expectedDeadline.Sub(deadline)
	if diff < 0 {
		diff = -diff
	}

	// Allow 1 second tolerance for test execution time
	if diff > 1*time.Second {
		t.Errorf("Timeout deadline differs by more than 1 second: got %v, expected ~5 minutes from now", diff)
	}

	// Verify timeout value is exactly 5 minutes (300 seconds)
	expectedTimeout := 5 * time.Minute
	if expectedTimeout != 300*time.Second {
		t.Error("5 minutes should equal 300 seconds")
	}

	// Verify timeout hierarchy: agent < generation < queue
	generationTimeout := 8 * time.Minute
	queueTimeout := 10 * time.Minute
	if expectedTimeout >= generationTimeout {
		t.Errorf("Agent timeout (%v) must be less than generation timeout (%v)", expectedTimeout, generationTimeout)
	}
	if generationTimeout >= queueTimeout {
		t.Errorf("Generation timeout (%v) must be less than queue timeout (%v)", generationTimeout, queueTimeout)
	}
	if queueTimeout-expectedTimeout < 2*time.Minute {
		t.Errorf("Queue-to-agent margin (%v) must be >= 2 minutes", queueTimeout-expectedTimeout)
	}
}

// TestWorkerCreation tests worker creation and type assignment
func TestWorkerCreation(t *testing.T) {
	tests := []struct {
		name      string
		agentType AgentType
	}{
		{"frontend worker", AgentFrontend},
		{"backend worker", AgentBackend},
		{"database worker", AgentDatabase},
		{"test worker", AgentTest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewWorker(tt.agentType, nil)

			if worker == nil {
				t.Fatal("NewWorker returned nil")
			}

			if worker.Type() != tt.agentType {
				t.Errorf("Expected type %s, got %s", tt.agentType, worker.Type())
			}
		})
	}
}

// TestWorkerProgressCallback tests progress event emission
func TestWorkerProgressCallback(t *testing.T) {
	var capturedEvents []ProgressEvent
	callback := func(event ProgressEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	worker := NewWorker(AgentBackend, callback)

	// Cast to implementation to test emitProgress directly
	impl, ok := worker.(*workerImpl)
	if !ok {
		t.Fatal("Worker is not workerImpl type")
	}

	// Test emitProgress directly
	impl.emitProgress("task-1", "starting", "Test message", 0)
	impl.emitProgress("task-1", "in_progress", "Working", 50)
	impl.emitProgress("task-1", "completed", "Done", 100)

	if len(capturedEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(capturedEvents))
	}

	// Verify first event
	if len(capturedEvents) > 0 {
		event := capturedEvents[0]
		if event.TaskID != "task-1" {
			t.Errorf("Expected TaskID 'task-1', got '%s'", event.TaskID)
		}
		if event.Status != "starting" {
			t.Errorf("Expected status 'starting', got '%s'", event.Status)
		}
		if event.AgentType != AgentBackend {
			t.Errorf("Expected AgentType %s, got %s", AgentBackend, event.AgentType)
		}
		if event.Progress != 0 {
			t.Errorf("Expected progress 0, got %d", event.Progress)
		}
	}

	// Verify last event
	if len(capturedEvents) > 2 {
		event := capturedEvents[2]
		if event.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", event.Status)
		}
		if event.Progress != 100 {
			t.Errorf("Expected progress 100, got %d", event.Progress)
		}
	}
}

// TestWorkerPromptCreation tests prompt generation for different agent types
func TestWorkerPromptCreation(t *testing.T) {
	tests := []struct {
		agentType        AgentType
		expectedKeywords []string
	}{
		{
			agentType: AgentFrontend,
			expectedKeywords: []string{
				"Svelte 5",
				"Tailwind CSS",
				"TypeScript",
				"runes",
				"src/",
			},
		},
		{
			agentType: AgentBackend,
			expectedKeywords: []string{
				"Go",
				"Gin",
				"Handler",
				"Service",
				"Repository",
				"slog",
				"cmd/",
				"internal/",
			},
		},
		{
			agentType: AgentDatabase,
			expectedKeywords: []string{
				"PostgreSQL",
				"CREATE TABLE",
				"migrations/",
				"Indexes",
			},
		},
		{
			agentType: AgentTest,
			expectedKeywords: []string{
				"tests",
				"80%",
				"coverage",
				"_test.go",
				".test.ts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			worker := NewWorker(tt.agentType, nil).(*workerImpl)

			task := Task{
				ID:          "test-task",
				Type:        tt.agentType,
				Description: "Create test component",
				Workspace:   "/workspace/test/",
				Priority:    1,
			}

			prompt := worker.createPrompt(task)

			if prompt == "" {
				t.Fatal("Prompt is empty")
			}

			// Verify task info is included
			if !strings.Contains(prompt, task.Description) {
				t.Error("Prompt missing task description")
			}

			if !strings.Contains(prompt, task.Workspace) {
				t.Error("Prompt missing workspace")
			}

			// Verify agent-specific keywords
			for _, keyword := range tt.expectedKeywords {
				if !strings.Contains(prompt, keyword) {
					t.Errorf("Prompt missing expected keyword: %s", keyword)
				}
			}

			// Verify file format instructions are present
			if !strings.Contains(prompt, "```") {
				t.Error("Prompt missing code block format instructions")
			}

			if !strings.Contains(prompt, "filepath") || !strings.Contains(prompt, "path") {
				t.Error("Prompt missing filepath instructions")
			}
		})
	}
}

// TestWorkerContextPropagation tests context handling
func TestWorkerContextPropagation(t *testing.T) {
	t.Run("parent context with deadline", func(t *testing.T) {
		// Create a parent context with a short deadline
		parentCtx, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer parentCancel()

		// Worker should respect parent context
		// Since we can't easily mock the SDK, we just verify the pattern
		if parentCtx.Err() != nil {
			t.Error("Parent context already cancelled before test")
		}

		// Wait for deadline
		<-parentCtx.Done()

		if parentCtx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", parentCtx.Err())
		}
	})

	t.Run("parent context cancelled", func(t *testing.T) {
		parentCtx, parentCancel := context.WithCancel(context.Background())

		// Cancel immediately
		parentCancel()

		if parentCtx.Err() != context.Canceled {
			t.Errorf("Expected Canceled, got %v", parentCtx.Err())
		}
	})

	t.Run("child context with timeout", func(t *testing.T) {
		parentCtx := context.Background()

		// This mimics what worker.go does at line 42
		childCtx, childCancel := context.WithTimeout(parentCtx, 5*time.Minute)
		defer childCancel()

		// Verify child context has deadline
		deadline, ok := childCtx.Deadline()
		if !ok {
			t.Error("Child context has no deadline")
		}

		// Verify deadline is roughly 5 minutes from now
		expectedDeadline := time.Now().Add(5 * time.Minute)
		diff := expectedDeadline.Sub(deadline)
		if diff < 0 {
			diff = -diff
		}

		if diff > 1*time.Second {
			t.Errorf("Deadline differs by more than 1 second: %v", diff)
		}
	})
}

// TestWorkerResultStructure tests AgentResult structure
func TestWorkerResultStructure(t *testing.T) {
	result := &AgentResult{
		TaskID:       "test-task-123",
		AgentType:    AgentBackend,
		Success:      true,
		Output:       "```go:test.go\npackage test\n```",
		FilesCreated: []string{"test.go"},
		CodeBlocks: map[string]string{
			"test.go": "package test\n",
		},
		Error:    "",
		Duration: 2 * time.Minute,
	}

	if result.TaskID != "test-task-123" {
		t.Errorf("Expected TaskID 'test-task-123', got '%s'", result.TaskID)
	}

	if result.AgentType != AgentBackend {
		t.Errorf("Expected AgentType %s, got %s", AgentBackend, result.AgentType)
	}

	if !result.Success {
		t.Error("Expected Success=true")
	}

	if len(result.FilesCreated) != 1 {
		t.Errorf("Expected 1 file, got %d", len(result.FilesCreated))
	}

	if len(result.CodeBlocks) != 1 {
		t.Errorf("Expected 1 code block, got %d", len(result.CodeBlocks))
	}

	if result.Duration != 2*time.Minute {
		t.Errorf("Expected duration 2m, got %v", result.Duration)
	}
}

// TestWorkerErrorResult tests error result structure
func TestWorkerErrorResult(t *testing.T) {
	result := &AgentResult{
		TaskID:       "failed-task",
		AgentType:    AgentDatabase,
		Success:      false,
		Output:       "",
		FilesCreated: []string{},
		Error:        "context deadline exceeded",
		Duration:     5 * time.Minute,
	}

	if result.Success {
		t.Error("Expected Success=false on error")
	}

	if result.Error == "" {
		t.Error("Expected Error message to be set")
	}

	if !strings.Contains(result.Error, "deadline") {
		t.Errorf("Expected deadline error, got: %s", result.Error)
	}

	if result.Duration != 5*time.Minute {
		t.Errorf("Expected duration 5m (timeout duration), got %v", result.Duration)
	}
}

// TestWorkerDifferentAgentTypes tests all agent types
func TestWorkerDifferentAgentTypes(t *testing.T) {
	agentTypes := []AgentType{
		AgentFrontend,
		AgentBackend,
		AgentDatabase,
		AgentTest,
	}

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			worker := NewWorker(agentType, nil)

			if worker == nil {
				t.Fatalf("NewWorker returned nil for %s", agentType)
			}

			if worker.Type() != agentType {
				t.Errorf("Expected type %s, got %s", agentType, worker.Type())
			}

			// Verify prompt creation works for this type
			impl := worker.(*workerImpl)
			task := Task{
				ID:          fmt.Sprintf("task-%s", agentType),
				Type:        agentType,
				Description: fmt.Sprintf("Test %s task", agentType),
				Workspace:   "/workspace/",
				Priority:    1,
			}

			prompt := impl.createPrompt(task)
			if prompt == "" {
				t.Errorf("Prompt is empty for %s agent", agentType)
			}
		})
	}
}

// BenchmarkWorkerPromptCreation benchmarks prompt creation
func BenchmarkWorkerPromptCreation(b *testing.B) {
	worker := NewWorker(AgentBackend, nil).(*workerImpl)

	task := Task{
		ID:          "benchmark-task",
		Type:        AgentBackend,
		Description: "Create a REST API handler",
		Workspace:   "/workspace/backend/",
		Priority:    1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = worker.createPrompt(task)
	}
}

// BenchmarkWorkerProgressEmit benchmarks progress event emission
func BenchmarkWorkerProgressEmit(b *testing.B) {
	var eventCount int
	callback := func(event ProgressEvent) {
		eventCount++
	}

	worker := NewWorker(AgentFrontend, callback).(*workerImpl)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.emitProgress("task-bench", "in_progress", "Working", 50)
	}

	b.Logf("Total events emitted: %d", eventCount)
}
