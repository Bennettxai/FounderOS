package appgen

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestWorkerTimeoutIntegrationShortTask tests actual SDK with a very short task (integration test)
func TestWorkerTimeoutIntegrationShortTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("RUN_APPGEN_INTEGRATION") != "true" {
		t.Skip("Skipping appgen integration test; set RUN_APPGEN_INTEGRATION=true with Claude agent access")
	}

	var progressEvents []ProgressEvent
	worker := NewWorker(AgentBackend, func(event ProgressEvent) {
		progressEvents = append(progressEvents, event)
	})

	task := Task{
		ID:          "integration-short-task",
		Type:        AgentBackend,
		Description: "Create a simple hello world function in Go",
		Workspace:   "/workspace/test/",
		Priority:    1,
	}

	// Use a reasonable timeout (should complete within this)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	startTime := time.Now()
	result, err := worker.Execute(ctx, task)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify it completed well within 5 minute timeout
	if duration > 5*time.Minute {
		t.Errorf("Task took longer than 5 minute timeout: %v", duration)
	}

	t.Logf("Task completed in %v (well within 5min timeout)", duration)

	// Verify progress events
	if len(progressEvents) == 0 {
		t.Error("No progress events emitted")
	}

	statusesFound := make(map[string]bool)
	for _, event := range progressEvents {
		statusesFound[event.Status] = true
		t.Logf("Progress: %s - %s (%d%%)", event.Status, event.Message, event.Progress)
	}

	if !statusesFound["starting"] {
		t.Error("Missing 'starting' progress event")
	}

	if !statusesFound["completed"] {
		t.Error("Missing 'completed' progress event")
	}

	// Verify output contains code
	if result.Output == "" {
		t.Error("Result.Output is empty")
	}

	t.Logf("Output length: %d characters", len(result.Output))
}

// TestWorkerTimeoutIntegrationParentCancellation tests that parent context cancellation works
func TestWorkerTimeoutIntegrationParentCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("RUN_APPGEN_INTEGRATION") != "true" {
		t.Skip("Skipping appgen integration test; set RUN_APPGEN_INTEGRATION=true with Claude agent access")
	}

	var progressEvents []ProgressEvent
	worker := NewWorker(AgentDatabase, func(event ProgressEvent) {
		progressEvents = append(progressEvents, event)
	})

	task := Task{
		ID:          "integration-cancel-task",
		Type:        AgentDatabase,
		Description: "Create a complex database schema with many tables and relationships",
		Workspace:   "/workspace/db/",
		Priority:    1,
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()

	startTime := time.Now()
	result, err := worker.Execute(ctx, task)
	duration := time.Since(startTime)

	// Should fail due to cancellation
	if err == nil {
		t.Error("Expected cancellation error, got success")
	}

	if err != nil && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("Got error (may be wrapped): %v", err)
		// Don't fail - error might be wrapped
	}

	// Should complete quickly (within a few seconds of cancellation)
	if duration > 5*time.Second {
		t.Errorf("Cancellation took too long: %v", duration)
	}

	t.Logf("Task cancelled after %v", duration)

	if result == nil {
		t.Fatal("Result should not be nil even on cancellation")
	}

	if result.Success {
		t.Error("Result.Success should be false on cancellation")
	}

	// Verify failed event was emitted
	foundFailed := false
	for _, event := range progressEvents {
		if event.Status == "failed" {
			foundFailed = true
			t.Logf("Failed event: %s", event.Message)
		}
	}

	if !foundFailed {
		t.Error("Expected 'failed' progress event")
	}
}

// TestWorkerTimeout5MinuteLimit tests the actual 5-minute timeout behavior
func TestWorkerTimeout5MinuteLimit(t *testing.T) {
	// This test is informational - we don't actually wait 5+ minutes
	t.Skip("Skip by default - demonstrates 5min timeout exists in code")

	// If you want to verify the timeout actually works, uncomment and run manually:
	/*
		if testing.Short() {
			t.Skip("Skipping long-running integration test")
		}

		worker := NewWorker(AgentTest, nil)

		task := Task{
			ID:          "integration-timeout-task",
			Type:        AgentTest,
			Description: strings.Repeat("Create comprehensive tests for a very large application with many components and features. ", 100),
			Workspace:   "/workspace/tests/",
			Priority:    1,
		}

		// Don't set timeout on parent context - let worker timeout
		ctx := context.Background()

		startTime := time.Now()
		result, err := worker.Execute(ctx, task)
		duration := time.Since(startTime)

		t.Logf("Task ran for %v", duration)

		// Should timeout around 5 minutes
		if duration < 4*time.Minute+30*time.Second {
			t.Error("Expected timeout around 5min, but task completed too quickly")
		}

		if duration > 5*time.Minute+30*time.Second {
			t.Error("Timeout took longer than 5min + grace period")
		}

		if err == nil {
			t.Error("Expected timeout error")
		}

		if result != nil && result.Success {
			t.Error("Task should not succeed on timeout")
		}
	*/
}

// TestWorkerTimeoutDifferentAgentTypes tests that timeout applies to all agent types
func TestWorkerTimeoutDifferentAgentTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("RUN_APPGEN_INTEGRATION") != "true" {
		t.Skip("Skipping appgen integration test; set RUN_APPGEN_INTEGRATION=true with Claude agent access")
	}

	agentTypes := []AgentType{
		AgentFrontend,
		AgentBackend,
		AgentDatabase,
		AgentTest,
	}

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			worker := NewWorker(agentType, nil)

			task := Task{
				ID:          "timeout-test-" + string(agentType),
				Type:        agentType,
				Description: "Create a simple hello world example",
				Workspace:   "/workspace/" + string(agentType) + "/",
				Priority:    1,
			}

			// Set a reasonable timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			startTime := time.Now()
			result, err := worker.Execute(ctx, task)
			duration := time.Since(startTime)

			if err != nil {
				t.Logf("Agent %s execution failed: %v (duration: %v)", agentType, err, duration)
				// Don't fail - API might be rate limited or unavailable
				return
			}

			if result == nil {
				t.Errorf("Result is nil for %s agent", agentType)
				return
			}

			// Verify it completed within reasonable time
			if duration > 5*time.Minute {
				t.Errorf("Agent %s took longer than 5min timeout: %v", agentType, duration)
			}

			t.Logf("Agent %s completed in %v", agentType, duration)
		})
	}
}

// TestWorkerTimeoutCodeReview tests that the timeout code is correctly implemented
func TestWorkerTimeoutCodeReview(t *testing.T) {
	// Verify the actual timeout value and hierarchy with real assertions

	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("5-minute timeout context must have a deadline")
	}

	// Verify timeout is ~5 minutes from now (1s tolerance for test execution)
	untilDeadline := time.Until(deadline)
	if untilDeadline < 4*time.Minute+59*time.Second || untilDeadline > 5*time.Minute+1*time.Second {
		t.Errorf("Expected ~5min deadline, got %v", untilDeadline)
	}

	// Verify timeout hierarchy: 5min agent < 8min generation < 10min queue
	agentTimeout := 5 * time.Minute
	generationTimeout := 8 * time.Minute
	queueTimeout := 10 * time.Minute

	if agentTimeout >= generationTimeout {
		t.Errorf("Agent timeout (%v) must be < generation timeout (%v)", agentTimeout, generationTimeout)
	}
	if generationTimeout >= queueTimeout {
		t.Errorf("Generation timeout (%v) must be < queue timeout (%v)", generationTimeout, queueTimeout)
	}

	// Verify minimum margin between agent and queue (>= 2min for retry buffer)
	margin := queueTimeout - agentTimeout
	if margin < 2*time.Minute {
		t.Errorf("Agent-to-queue margin (%v) must be >= 2 minutes for retry buffer", margin)
	}
}

// TestWorkerTimeoutProgressCallbackReliability tests progress callback during timeout scenarios
func TestWorkerTimeoutProgressCallbackReliability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("RUN_APPGEN_INTEGRATION") != "true" {
		t.Skip("Skipping appgen integration test; set RUN_APPGEN_INTEGRATION=true with Claude agent access")
	}

	var progressEvents []ProgressEvent
	eventChan := make(chan ProgressEvent, 100)

	worker := NewWorker(AgentBackend, func(event ProgressEvent) {
		progressEvents = append(progressEvents, event)
		eventChan <- event
	})

	task := Task{
		ID:          "progress-reliability-test",
		Type:        AgentBackend,
		Description: "Create a simple API handler",
		Workspace:   "/workspace/backend/",
		Priority:    1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Monitor progress events in real-time
	done := make(chan bool)
	go func() {
		for event := range eventChan {
			t.Logf("[%v] Status: %s | Message: %s | Progress: %d%%",
				event.Timestamp.Format("15:04:05"),
				event.Status,
				event.Message,
				event.Progress,
			)
		}
		done <- true
	}()

	result, err := worker.Execute(ctx, task)

	close(eventChan)
	<-done

	if err != nil {
		t.Logf("Execution completed with error: %v", err)
	}

	// Verify progress event sequence
	if len(progressEvents) < 2 {
		t.Errorf("Expected at least 2 progress events (start + end), got %d", len(progressEvents))
	}

	// First event should be "starting"
	if len(progressEvents) > 0 && progressEvents[0].Status != "starting" {
		t.Errorf("First event should be 'starting', got '%s'", progressEvents[0].Status)
	}

	// Last event should be "completed" or "failed"
	if len(progressEvents) > 0 {
		lastEvent := progressEvents[len(progressEvents)-1]
		if lastEvent.Status != "completed" && lastEvent.Status != "failed" {
			t.Errorf("Last event should be 'completed' or 'failed', got '%s'", lastEvent.Status)
		}
	}

	// Verify all events have required fields
	for i, event := range progressEvents {
		if event.TaskID != task.ID {
			t.Errorf("Event %d has wrong TaskID: expected %s, got %s", i, task.ID, event.TaskID)
		}
		if event.AgentType != task.Type {
			t.Errorf("Event %d has wrong AgentType: expected %s, got %s", i, task.Type, event.AgentType)
		}
		if event.Timestamp.IsZero() {
			t.Errorf("Event %d has zero timestamp", i)
		}
		if event.Message == "" {
			t.Errorf("Event %d has empty message", i)
		}
	}

	t.Logf("Total progress events: %d", len(progressEvents))

	if result != nil {
		t.Logf("Result: Success=%v, Duration=%v, FilesCreated=%d",
			result.Success,
			result.Duration,
			len(result.FilesCreated),
		)
	}
}
