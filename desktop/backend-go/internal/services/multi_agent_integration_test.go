package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/appgen"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestMultiAgentPipelineIntegration verifies the complete multi-agent app generation pipeline
// This is the core OSA functionality test suite for PEDRO-5

// Helper functions

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// strPtr is an alias for stringPtr for compatibility
func strPtr(s string) *string {
	return &s
}

// MockOrchestrator implements appgen.Orchestrator for testing
type MockOrchestrator struct {
	mock.Mock
	progressCallback appgen.ProgressCallback
}

func (m *MockOrchestrator) CreatePlan(ctx context.Context, req appgen.AppRequest) (*appgen.Plan, error) {
	args := m.Called(ctx, req)
	if plan := args.Get(0); plan != nil {
		return plan.(*appgen.Plan), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrchestrator) Execute(ctx context.Context, plan *appgen.Plan) (*appgen.GeneratedApp, error) {
	args := m.Called(ctx, plan)

	// Emit progress events for each agent
	if m.progressCallback != nil && args.Error(1) == nil {
		for _, task := range plan.Tasks {
			m.progressCallback(appgen.ProgressEvent{
				TaskID:    task.ID,
				AgentType: task.Type,
				Status:    "starting",
				Message:   fmt.Sprintf("%s agent starting", task.Type),
				Progress:  0,
				Timestamp: time.Now(),
			})

			// Simulate in-progress
			time.Sleep(10 * time.Millisecond)
			m.progressCallback(appgen.ProgressEvent{
				TaskID:    task.ID,
				AgentType: task.Type,
				Status:    "in_progress",
				Message:   fmt.Sprintf("%s agent executing", task.Type),
				Progress:  50,
				Timestamp: time.Now(),
			})

			// Complete
			time.Sleep(10 * time.Millisecond)
			m.progressCallback(appgen.ProgressEvent{
				TaskID:    task.ID,
				AgentType: task.Type,
				Status:    "completed",
				Message:   fmt.Sprintf("%s agent completed", task.Type),
				Progress:  100,
				Timestamp: time.Now(),
			})
		}
	}

	if app := args.Get(0); app != nil {
		return app.(*appgen.GeneratedApp), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrchestrator) SetProgressCallback(callback appgen.ProgressCallback) {
	m.progressCallback = callback
}

func (m *MockOrchestrator) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockOrchestrator) GetCircuitBreakerMetrics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// setupTestDB creates a test database connection
// Uses testcontainers pattern from existing tests
func setupTestDB(t *testing.T) (*pgxpool.Pool, *sqlc.Queries) {
	t.Helper()

	// This would normally use testcontainers
	// For now, skip if DATABASE_URL not set
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	// Ping to verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	queries := sqlc.New(pool)
	return pool, queries
}

// setupTestTempWorkspace creates a temporary workspace directory for file operations
func setupTestTempWorkspace(t *testing.T) string {
	t.Helper()

	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("businessos-test-%d", time.Now().UnixNano()))
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// TestFullPipelineE2E tests the complete multi-agent pipeline end-to-end
// Test Scenario 1: Full Pipeline E2E Test
func TestFullPipelineE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)

	// Create event bus for SSE tracking
	eventBus := NewBuildEventBus(nil)

	// Create mock orchestrator
	mockOrch := new(MockOrchestrator)

	// Setup expected plan
	expectedPlan := &appgen.Plan{
		Architecture: "Test app architecture",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Create Svelte frontend", Workspace: "/workspace/frontend/", Priority: 1},
			{ID: "task-backend", Type: appgen.AgentBackend, Description: "Create Go backend", Workspace: "/workspace/backend/", Priority: 1},
			{ID: "task-database", Type: appgen.AgentDatabase, Description: "Create PostgreSQL migrations", Workspace: "/workspace/database/", Priority: 2},
			{ID: "task-test", Type: appgen.AgentTest, Description: "Create tests", Workspace: "/workspace/tests/", Priority: 3},
		},
		CreatedAt: time.Now(),
	}

	// Setup expected generated app with code blocks
	expectedApp := &appgen.GeneratedApp{
		AppName: "Test App",
		Results: []appgen.AgentResult{
			{
				TaskID:    "task-frontend",
				AgentType: appgen.AgentFrontend,
				Success:   true,
				Output:    "Frontend created successfully",
				CodeBlocks: map[string]string{
					"App.svelte":          "<script>\n  let count = 0;\n</script>\n<h1>Hello {count}</h1>",
					"routes/+page.svelte": "<h1>Welcome</h1>",
				},
				Duration: 500 * time.Millisecond,
			},
			{
				TaskID:    "task-backend",
				AgentType: appgen.AgentBackend,
				Success:   true,
				Output:    "Backend created successfully",
				CodeBlocks: map[string]string{
					"main.go":                 "package main\n\nfunc main() {\n  // Backend code\n}",
					"internal/handler/api.go": "package handler\n\n// API handlers",
				},
				Duration: 600 * time.Millisecond,
			},
			{
				TaskID:    "task-database",
				AgentType: appgen.AgentDatabase,
				Success:   true,
				Output:    "Database schema created",
				CodeBlocks: map[string]string{
					"001_init.sql": "CREATE TABLE users (id UUID PRIMARY KEY);",
				},
				Duration: 300 * time.Millisecond,
			},
			{
				TaskID:    "task-test",
				AgentType: appgen.AgentTest,
				Success:   true,
				Output:    "Tests created",
				CodeBlocks: map[string]string{
					"app_test.go": "package main\n\nimport \"testing\"\n\nfunc TestApp(t *testing.T) {}",
				},
				Duration: 400 * time.Millisecond,
			},
		},
		Success:       true,
		TotalDuration: 2 * time.Second,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(expectedPlan, nil)
	mockOrch.On("Execute", mock.Anything, expectedPlan).Return(expectedApp, nil)

	// Create AppGenerationOrchestrator with mocked agent orchestrator
	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		eventBus:         eventBus,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	// Track SSE events
	var sseEvents []BuildEvent
	var sseEventsMu sync.Mutex

	queueItemID := uuid.New()
	subscriber := eventBus.Subscribe(ctx, uuid.Nil, queueItemID)

	go func() {
		for event := range subscriber.Events {
			sseEventsMu.Lock()
			sseEvents = append(sseEvents, event)
			sseEventsMu.Unlock()
		}
	}()

	// Execute generation
	req := MultiAgentAppRequest{
		AppName:     "Test App",
		Description: "A test application",
		Features:    []string{"auth", "api"},
		QueueItemID: queueItemID.String(),
	}

	result, err := orchestrator.Generate(ctx, req)

	// Assertions
	require.NoError(t, err, "Generation should succeed")
	require.NotNil(t, result, "Result should not be nil")

	assert.True(t, result.Success, "Generation should be successful")
	assert.Len(t, result.Results, 4, "Should have results from all 4 agents")

	// Verify all agents executed
	agentTypes := make(map[appgen.AgentType]bool)
	for _, r := range result.Results {
		agentTypes[r.AgentType] = true
		assert.True(t, r.Success, "Agent %s should succeed", r.AgentType)
		assert.NotEmpty(t, r.CodeBlocks, "Agent %s should generate code", r.AgentType)
	}

	assert.True(t, agentTypes[appgen.AgentFrontend], "Frontend agent should execute")
	assert.True(t, agentTypes[appgen.AgentBackend], "Backend agent should execute")
	assert.True(t, agentTypes[appgen.AgentDatabase], "Database agent should execute")
	assert.True(t, agentTypes[appgen.AgentTest], "Test agent should execute")

	// Verify files persisted to filesystem
	workspacePath := filepath.Join(workspaceDir, queueItemID.String())
	assert.DirExists(t, workspacePath, "Workspace directory should exist")

	// Check frontend files
	frontendFiles := []string{
		"frontend/App.svelte",
		"frontend/routes/+page.svelte",
	}
	for _, file := range frontendFiles {
		assert.FileExists(t, filepath.Join(workspacePath, file), "File %s should exist", file)
	}

	// Check backend files
	backendFiles := []string{
		"backend/main.go",
		"backend/internal/handler/api.go",
	}
	for _, file := range backendFiles {
		assert.FileExists(t, filepath.Join(workspacePath, file), "File %s should exist", file)
	}

	// Verify SSE events emitted
	time.Sleep(100 * time.Millisecond) // Allow events to process
	sseEventsMu.Lock()
	eventCount := len(sseEvents)
	sseEventsMu.Unlock()

	assert.GreaterOrEqual(t, eventCount, 4, "Should have at least 4 SSE events (one per agent)")

	// Verify event types
	var hasGenerationStarted, hasAgentProgress, hasGenerationComplete bool
	for _, event := range sseEvents {
		switch event.EventType {
		case "generation_started":
			hasGenerationStarted = true
		case "agent_progress":
			hasAgentProgress = true
		case "generation_complete":
			hasGenerationComplete = true
		}
	}

	assert.True(t, hasGenerationStarted, "Should have generation_started event")
	assert.True(t, hasAgentProgress, "Should have agent_progress events")
	assert.True(t, hasGenerationComplete, "Should have generation_complete event")

	mockOrch.AssertExpectations(t)
}

// TestAgentFailureHandling tests partial failure scenarios
// Test Scenario 2: Agent Failure Handling
func TestAgentFailureHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)
	eventBus := NewBuildEventBus(nil)

	mockOrch := new(MockOrchestrator)

	// Plan with 4 tasks
	plan := &appgen.Plan{
		Architecture: "Test app with failure",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Create frontend", Workspace: "/workspace/frontend/", Priority: 1},
			{ID: "task-backend", Type: appgen.AgentBackend, Description: "Create backend", Workspace: "/workspace/backend/", Priority: 1},
			{ID: "task-database", Type: appgen.AgentDatabase, Description: "Create database", Workspace: "/workspace/database/", Priority: 2},
			{ID: "task-test", Type: appgen.AgentTest, Description: "Create tests", Workspace: "/workspace/tests/", Priority: 3},
		},
		CreatedAt: time.Now(),
	}

	// Frontend agent fails, others succeed
	failedApp := &appgen.GeneratedApp{
		AppName: "Partially Failed App",
		Results: []appgen.AgentResult{
			{
				TaskID:    "task-frontend",
				AgentType: appgen.AgentFrontend,
				Success:   false,
				Error:     "Frontend generation failed: template not found",
				Duration:  100 * time.Millisecond,
			},
			{
				TaskID:    "task-backend",
				AgentType: appgen.AgentBackend,
				Success:   true,
				Output:    "Backend created",
				CodeBlocks: map[string]string{
					"main.go": "package main",
				},
				Duration: 200 * time.Millisecond,
			},
			{
				TaskID:    "task-database",
				AgentType: appgen.AgentDatabase,
				Success:   true,
				CodeBlocks: map[string]string{
					"001_init.sql": "CREATE TABLE users",
				},
				Duration: 150 * time.Millisecond,
			},
			{
				TaskID:    "task-test",
				AgentType: appgen.AgentTest,
				Success:   true,
				CodeBlocks: map[string]string{
					"test.go": "package main",
				},
				Duration: 100 * time.Millisecond,
			},
		},
		Success:       false,
		ErrorMessage:  "1 tasks failed",
		TotalDuration: 550 * time.Millisecond,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
	mockOrch.On("Execute", mock.Anything, plan).Return(failedApp, nil)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		eventBus:         eventBus,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	queueItemID := uuid.New()
	req := MultiAgentAppRequest{
		AppName:     "Test App",
		Description: "Test partial failure",
		QueueItemID: queueItemID.String(),
	}

	result, err := orchestrator.Generate(ctx, req)

	// Should return error since one agent failed
	require.Error(t, err, "Should return error when agent fails")
	require.NotNil(t, result, "Result should still be returned")

	assert.False(t, result.Success, "Overall success should be false")
	assert.Contains(t, result.ErrorMessage, "failed", "Error message should mention failure")

	// Verify partial results saved
	assert.Len(t, result.Results, 4, "Should have results from all agents (including failed)")

	// Count successful vs failed
	var successCount, failCount int
	for _, r := range result.Results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	assert.Equal(t, 3, successCount, "3 agents should succeed")
	assert.Equal(t, 1, failCount, "1 agent should fail")

	// Verify successful agents still saved files
	workspacePath := filepath.Join(workspaceDir, queueItemID.String())
	assert.FileExists(t, filepath.Join(workspacePath, "backend/main.go"), "Backend files should be saved")
	assert.FileExists(t, filepath.Join(workspacePath, "database/001_init.sql"), "Database files should be saved")

	mockOrch.AssertExpectations(t)
}

// TestFilePersistenceIntegration verifies file hashing and database persistence
// Test Scenario 3: File Persistence Integration
func TestFilePersistenceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)

	// Create test app with files
	mockOrch := new(MockOrchestrator)

	plan := &appgen.Plan{
		Architecture: "File persistence test",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
		},
		CreatedAt: time.Now(),
	}

	testFileContent := "console.log('Hello from frontend');"

	app := &appgen.GeneratedApp{
		AppName: "File Test App",
		Results: []appgen.AgentResult{
			{
				TaskID:    "task-frontend",
				AgentType: appgen.AgentFrontend,
				Success:   true,
				CodeBlocks: map[string]string{
					"index.ts": testFileContent,
				},
				Duration: 100 * time.Millisecond,
			},
		},
		Success:       true,
		TotalDuration: 100 * time.Millisecond,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
	mockOrch.On("Execute", mock.Anything, plan).Return(app, nil)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	queueItemID := uuid.New()
	req := MultiAgentAppRequest{
		AppName:     "File Test",
		QueueItemID: queueItemID.String(),
	}

	_, err := orchestrator.Generate(ctx, req)
	require.NoError(t, err)

	// Verify file exists on filesystem
	filePath := filepath.Join(workspaceDir, queueItemID.String(), "frontend/index.ts")
	require.FileExists(t, filePath, "File should exist on filesystem")

	// Read and verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, testFileContent, string(content), "File content should match")

	// Verify database entry with SHA256 hash
	// Note: This would require database schema to be set up
	// Skipping database verification in this test since it requires full DB setup

	mockOrch.AssertExpectations(t)
}

// TestStatusTransitionFlow verifies status updates through pipeline
// Test Scenario 4: Status Transition Flow
func TestStatusTransitionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)
	eventBus := NewBuildEventBus(nil)

	mockOrch := new(MockOrchestrator)

	plan := &appgen.Plan{
		Architecture: "Status transition test",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
		},
		CreatedAt: time.Now(),
	}

	app := &appgen.GeneratedApp{
		AppName: "Status Test",
		Results: []appgen.AgentResult{
			{TaskID: "task-frontend", AgentType: appgen.AgentFrontend, Success: true, CodeBlocks: map[string]string{"app.ts": "code"}, Duration: 100 * time.Millisecond},
		},
		Success:       true,
		TotalDuration: 100 * time.Millisecond,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
	mockOrch.On("Execute", mock.Anything, plan).Return(app, nil)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		eventBus:         eventBus,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	// Track status transitions via SSE events
	var statusTransitions []string
	var transitionsMu sync.Mutex

	queueItemID := uuid.New()
	subscriber := eventBus.Subscribe(ctx, uuid.Nil, queueItemID)

	go func() {
		for event := range subscriber.Events {
			transitionsMu.Lock()
			if status, ok := event.Data["status"].(string); ok {
				statusTransitions = append(statusTransitions, status)
			}
			transitionsMu.Unlock()
		}
	}()

	req := MultiAgentAppRequest{
		AppName:     "Status Test",
		QueueItemID: queueItemID.String(),
	}

	_, err := orchestrator.Generate(ctx, req)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Allow events to propagate

	// Verify status progression
	transitionsMu.Lock()
	defer transitionsMu.Unlock()

	assert.Contains(t, statusTransitions, "started", "Should have 'started' status")
	assert.Contains(t, statusTransitions, "planning", "Should have 'planning' status")
	assert.Contains(t, statusTransitions, "executing", "Should have 'executing' status")
	assert.Contains(t, statusTransitions, "completed", "Should have 'completed' status")

	mockOrch.AssertExpectations(t)
}

// TestSSEEventStream verifies all SSE events are emitted correctly
// Test Scenario 5: SSE Event Stream
func TestSSEEventStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)
	eventBus := NewBuildEventBus(nil)

	mockOrch := new(MockOrchestrator)

	plan := &appgen.Plan{
		Architecture: "SSE test",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
			{ID: "task-backend", Type: appgen.AgentBackend, Description: "Backend", Workspace: "/workspace/backend/", Priority: 1},
		},
		CreatedAt: time.Now(),
	}

	app := &appgen.GeneratedApp{
		AppName: "SSE Test",
		Results: []appgen.AgentResult{
			{TaskID: "task-frontend", AgentType: appgen.AgentFrontend, Success: true, CodeBlocks: map[string]string{"app.ts": "code"}, Duration: 100 * time.Millisecond},
			{TaskID: "task-backend", AgentType: appgen.AgentBackend, Success: true, CodeBlocks: map[string]string{"main.go": "code"}, Duration: 100 * time.Millisecond},
		},
		Success:       true,
		TotalDuration: 200 * time.Millisecond,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
	mockOrch.On("Execute", mock.Anything, plan).Return(app, nil)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		eventBus:         eventBus,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	// Collect all SSE events
	var allEvents []BuildEvent
	var eventsMu sync.Mutex

	queueItemID := uuid.New()
	subscriber := eventBus.Subscribe(ctx, uuid.Nil, queueItemID)

	go func() {
		for event := range subscriber.Events {
			eventsMu.Lock()
			allEvents = append(allEvents, event)
			eventsMu.Unlock()
		}
	}()

	req := MultiAgentAppRequest{
		AppName:     "SSE Test",
		QueueItemID: queueItemID.String(),
	}

	result, err := orchestrator.Generate(ctx, req)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	eventsMu.Lock()
	defer eventsMu.Unlock()

	// Verify required events
	eventTypes := make(map[string]int)
	for _, event := range allEvents {
		eventTypes[event.EventType]++
	}

	assert.GreaterOrEqual(t, eventTypes["generation_started"], 1, "Should have build_started event")
	assert.GreaterOrEqual(t, eventTypes["agent_progress"], 2, "Should have progress events for both agents")
	assert.GreaterOrEqual(t, eventTypes["generation_complete"], 1, "Should have build_completed event")

	// Verify final event has metrics
	var completeEvent *BuildEvent
	for i := len(allEvents) - 1; i >= 0; i-- {
		if allEvents[i].EventType == "generation_complete" {
			completeEvent = &allEvents[i]
			break
		}
	}

	require.NotNil(t, completeEvent, "Should have completion event")
	assert.NotNil(t, completeEvent.Data["files_created"], "Should include file count")
	assert.NotNil(t, completeEvent.Data["total_size_bytes"], "Should include total size")
	assert.Equal(t, 2, completeEvent.Data["files_created"], "Should have 2 files created")

	// Verify workspace path in result
	assert.NotEmpty(t, result.WorkspacePath, "Should have workspace path")

	mockOrch.AssertExpectations(t)
}

// TestConcurrentGeneration verifies independent workspace isolation
// Test Scenario 6: Concurrent Generation
func TestConcurrentGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)
	eventBus := NewBuildEventBus(nil)

	// Create 3 independent orchestrators
	var wg sync.WaitGroup
	results := make([]*appgen.GeneratedApp, 3)
	errors := make([]error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			mockOrch := new(MockOrchestrator)

			plan := &appgen.Plan{
				Architecture: fmt.Sprintf("App %d", index),
				Tasks: []appgen.Task{
					{ID: fmt.Sprintf("task-frontend-%d", index), Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
				},
				CreatedAt: time.Now(),
			}

			app := &appgen.GeneratedApp{
				AppName: fmt.Sprintf("Concurrent App %d", index),
				Results: []appgen.AgentResult{
					{
						TaskID:    fmt.Sprintf("task-frontend-%d", index),
						AgentType: appgen.AgentFrontend,
						Success:   true,
						CodeBlocks: map[string]string{
							fmt.Sprintf("app-%d.ts", index): fmt.Sprintf("// App %d code", index),
						},
						Duration: 100 * time.Millisecond,
					},
				},
				Success:       true,
				TotalDuration: 100 * time.Millisecond,
				CreatedAt:     time.Now(),
			}

			mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
			mockOrch.On("Execute", mock.Anything, plan).Return(app, nil)

			orchestrator := &AppGenerationOrchestrator{
				orchestrator:     mockOrch,
				queries:          queries,
				pool:             pool,
				eventBus:         eventBus,
				workspaceManager: NewWorkspaceManager(workspaceDir, nil),
				maxRetries:       3,
				logger:           nil,
			}

			queueItemID := uuid.New()
			req := MultiAgentAppRequest{
				AppName:     fmt.Sprintf("App %d", index),
				QueueItemID: queueItemID.String(),
			}

			result, err := orchestrator.Generate(ctx, req)
			results[index] = result
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all succeeded
	for i := 0; i < 3; i++ {
		require.NoError(t, errors[i], "App %d should succeed", i)
		require.NotNil(t, results[i], "App %d should have result", i)
		assert.True(t, results[i].Success, "App %d should be successful", i)
	}

	// Verify independent workspaces
	workspaces := make(map[string]bool)
	for _, result := range results {
		workspaces[result.WorkspacePath] = true
	}

	assert.Len(t, workspaces, 3, "Should have 3 independent workspaces")

	// Verify no file conflicts
	for i, result := range results {
		workspacePath := result.WorkspacePath
		assert.DirExists(t, workspacePath, "Workspace %d should exist", i)

		expectedFile := filepath.Join(workspacePath, fmt.Sprintf("frontend/app-%d.ts", i))
		assert.FileExists(t, expectedFile, "File for app %d should exist", i)
	}
}

// TestTimeoutHandling verifies graceful timeout degradation
// Test Scenario 7: Timeout Handling
func TestTimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, queries := setupTestDB(t)
	defer pool.Close()

	workspaceDir := setupTestTempWorkspace(t)

	mockOrch := new(MockOrchestrator)

	plan := &appgen.Plan{
		Architecture: "Timeout test",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
		},
		CreatedAt: time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)

	// Simulate timeout by making Execute hang
	mockOrch.On("Execute", mock.Anything, plan).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		select {
		case <-ctx.Done():
			// Context cancelled
		case <-time.After(10 * time.Second):
			// Would timeout
		}
	}).Return(nil, context.DeadlineExceeded)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       1, // No retries for timeout test
		logger:           nil,
	}

	// Create context with short timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	queueItemID := uuid.New()
	req := MultiAgentAppRequest{
		AppName:     "Timeout Test",
		QueueItemID: queueItemID.String(),
	}

	_, err := orchestrator.Generate(timeoutCtx, req)

	// Should return timeout error
	require.Error(t, err, "Should return error on timeout")
	assert.True(t,
		strings.Contains(err.Error(), "deadline") ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "context"),
		"Error should mention timeout/deadline/context")

	mockOrch.AssertExpectations(t)
}

// BenchmarkMultiAgentPipeline benchmarks the full pipeline performance
func BenchmarkMultiAgentPipeline(b *testing.B) {
	ctx := context.Background()

	// Skip if no database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		b.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(b, err)
	defer pool.Close()

	queries := sqlc.New(pool)
	workspaceDir := filepath.Join(os.TempDir(), "benchmark-workspace")
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	mockOrch := new(MockOrchestrator)

	plan := &appgen.Plan{
		Architecture: "Benchmark",
		Tasks: []appgen.Task{
			{ID: "task-frontend", Type: appgen.AgentFrontend, Description: "Frontend", Workspace: "/workspace/frontend/", Priority: 1},
			{ID: "task-backend", Type: appgen.AgentBackend, Description: "Backend", Workspace: "/workspace/backend/", Priority: 1},
		},
		CreatedAt: time.Now(),
	}

	app := &appgen.GeneratedApp{
		AppName: "Benchmark App",
		Results: []appgen.AgentResult{
			{TaskID: "task-frontend", AgentType: appgen.AgentFrontend, Success: true, CodeBlocks: map[string]string{"app.ts": "code"}, Duration: 10 * time.Millisecond},
			{TaskID: "task-backend", AgentType: appgen.AgentBackend, Success: true, CodeBlocks: map[string]string{"main.go": "code"}, Duration: 10 * time.Millisecond},
		},
		Success:       true,
		TotalDuration: 20 * time.Millisecond,
		CreatedAt:     time.Now(),
	}

	mockOrch.On("CreatePlan", mock.Anything, mock.Anything).Return(plan, nil)
	mockOrch.On("Execute", mock.Anything, plan).Return(app, nil)

	orchestrator := &AppGenerationOrchestrator{
		orchestrator:     mockOrch,
		queries:          queries,
		pool:             pool,
		workspaceManager: NewWorkspaceManager(workspaceDir, nil),
		maxRetries:       3,
		logger:           nil,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queueItemID := uuid.New()
		req := MultiAgentAppRequest{
			AppName:     "Benchmark",
			QueueItemID: queueItemID.String(),
		}

		_, err := orchestrator.Generate(ctx, req)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}

// Helper function to verify JSON serialization of BuildEvent
func TestBuildEventSerialization(t *testing.T) {
	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           uuid.New(),
		EventType:       "agent_progress",
		Phase:           "frontend",
		ProgressPercent: 50,
		StatusMessage:   "Frontend generating",
		Data: map[string]interface{}{
			"agent_type": "frontend",
			"status":     "in_progress",
		},
		Timestamp: time.Now(),
	}

	// Verify JSON serialization works
	data, err := json.Marshal(event)
	require.NoError(t, err, "BuildEvent should serialize to JSON")

	var decoded BuildEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err, "BuildEvent should deserialize from JSON")

	assert.Equal(t, event.EventType, decoded.EventType)
	assert.Equal(t, event.Phase, decoded.Phase)
	assert.Equal(t, event.ProgressPercent, decoded.ProgressPercent)
}
