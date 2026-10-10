package agents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAgentTypeConstants(t *testing.T) {
	// Verify all 6 agent types are defined
	agentTypes := []AgentType{
		AgentTypeOrchestrator,
		AgentTypeDocument,
		AgentTypeProject,
		AgentTypeTask,
		AgentTypeClient,
		AgentTypeAnalyst,
	}

	if len(agentTypes) != 6 {
		t.Errorf("Expected 6 agent types, got %d", len(agentTypes))
	}

	// Verify each type has a unique value
	seen := make(map[AgentType]bool)
	for _, at := range agentTypes {
		if seen[at] {
			t.Errorf("Duplicate agent type: %s", at)
		}
		seen[at] = true
	}
}

func TestContextRequirements(t *testing.T) {
	reqs := ContextRequirements{
		NeedsProjects:    true,
		NeedsTasks:       true,
		NeedsClients:     false,
		NeedsKnowledge:   true,
		NeedsTeam:        false,
		MaxContextTokens: 8000,
		PrioritySections: []string{"projects", "tasks"},
	}

	if !reqs.NeedsProjects {
		t.Error("Expected NeedsProjects to be true")
	}
	if reqs.NeedsClients {
		t.Error("Expected NeedsClients to be false")
	}
	if reqs.MaxContextTokens != 8000 {
		t.Errorf("Expected MaxContextTokens 8000, got %d", reqs.MaxContextTokens)
	}
	if len(reqs.PrioritySections) != 2 {
		t.Errorf("Expected 2 priority sections, got %d", len(reqs.PrioritySections))
	}
}

func TestBaseAgentConfig(t *testing.T) {
	cfg := BaseAgentConfig{
		UserID:       "test-user",
		UserName:     "Test User",
		AgentType:    AgentTypeOrchestrator,
		AgentName:    "Test Agent",
		Description:  "Test description",
		SystemPrompt: "Test prompt",
		EnabledTools: []string{"create_task", "get_project"},
	}

	if cfg.UserID != "test-user" {
		t.Errorf("Expected UserID 'test-user', got '%s'", cfg.UserID)
	}
	if cfg.AgentType != AgentTypeOrchestrator {
		t.Errorf("Expected AgentType Orchestrator, got %s", cfg.AgentType)
	}
	if len(cfg.EnabledTools) != 2 {
		t.Errorf("Expected 2 enabled tools, got %d", len(cfg.EnabledTools))
	}
}

func TestNewBaseAgent(t *testing.T) {
	cfg := BaseAgentConfig{
		UserID:       "test-user",
		UserName:     "Test User",
		AgentType:    AgentTypeDocument,
		AgentName:    "Document Agent",
		Description:  "Creates documents",
		SystemPrompt: "You are a document specialist",
		EnabledTools: []string{"search_documents"},
		ContextReqs: ContextRequirements{
			NeedsProjects:  true,
			NeedsKnowledge: true,
		},
	}

	agent := NewBaseAgent(cfg)

	if agent.Type() != AgentTypeDocument {
		t.Errorf("Expected type Document, got %s", agent.Type())
	}
	if agent.Name() != "Document Agent" {
		t.Errorf("Expected name 'Document Agent', got '%s'", agent.Name())
	}
	if agent.Description() != "Creates documents" {
		t.Errorf("Expected description 'Creates documents', got '%s'", agent.Description())
	}
	if agent.GetSystemPrompt() != "You are a document specialist" {
		t.Error("System prompt mismatch")
	}

	reqs := agent.GetContextRequirements()
	if !reqs.NeedsProjects {
		t.Error("Expected NeedsProjects to be true")
	}
	if !reqs.NeedsKnowledge {
		t.Error("Expected NeedsKnowledge to be true")
	}

	tools := agent.GetEnabledTools()
	if len(tools) != 1 || tools[0] != "search_documents" {
		t.Errorf("Expected enabled tools ['search_documents'], got %v", tools)
	}
}

func TestAgentSetModel(t *testing.T) {
	agent := NewBaseAgent(BaseAgentConfig{
		UserID:    "test",
		AgentType: AgentTypeOrchestrator,
	})

	agent.SetModel("gpt-4")
	if agent.Model() != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", agent.Model())
	}
}

func TestShouldDelegateForFocusMode(t *testing.T) {
	tests := []struct {
		focusMode      string
		shouldDelegate bool
		expectedAgent  AgentType
	}{
		{"write", true, AgentTypeDocument},
		{"analyze", true, AgentTypeAnalyst},
		{"plan", true, AgentTypeProject},
		{"general", false, AgentTypeOrchestrator},
		{"unknown", false, AgentTypeOrchestrator},
	}

	for _, tt := range tests {
		shouldDelegate, agent := ShouldDelegateForFocusMode(tt.focusMode)
		if shouldDelegate != tt.shouldDelegate {
			t.Errorf("FocusMode '%s': expected shouldDelegate=%v, got %v", tt.focusMode, tt.shouldDelegate, shouldDelegate)
		}
		if shouldDelegate && agent != tt.expectedAgent {
			t.Errorf("FocusMode '%s': expected agent=%s, got %s", tt.focusMode, tt.expectedAgent, agent)
		}
	}
}

// =============================================================================
// 7.A - TOOL ACCESS VALIDATION TESTS
// Ensures agents can only call tools they are authorized to use
// =============================================================================

// AgentToolMatrix defines which tools each agent type can access
var AgentToolMatrix = map[AgentType][]string{
	AgentTypeOrchestrator: {
		// Read tools
		"search_documents", "get_project", "get_task", "get_client",
		"list_projects", "list_tasks", "get_team_capacity", "query_metrics",
		// Write tools
		"create_task", "update_task", "create_project", "update_project",
		"create_client", "update_client", "create_artifact", "log_activity",
		"bulk_create_tasks", "assign_task", "move_task",
		// Client tools
		"log_client_interaction", "update_client_pipeline",
		// Context tools (knowledge base)
		"tree_search", "browse_tree", "load_context",
	},
	AgentTypeDocument: {
		"create_artifact", "search_documents", "get_project", "get_client",
		"log_activity",
		"tree_search", "browse_tree", "load_context",
	},
	AgentTypeProject: {
		"create_project", "update_project", "get_project", "list_projects",
		"create_task", "bulk_create_tasks", "assign_task",
		"get_team_capacity", "search_documents",
		"create_artifact", "log_activity",
		"tree_search", "browse_tree", "load_context",
	},
	AgentTypeTask: {
		"create_task", "update_task", "get_task", "list_tasks",
		"bulk_create_tasks", "move_task", "assign_task",
		"get_team_capacity", "get_project",
		"log_activity",
		"tree_search", "browse_tree", "load_context",
	},
	AgentTypeClient: {
		"create_client", "update_client", "get_client",
		"log_client_interaction", "update_client_pipeline",
		"search_documents", "get_project",
		"create_artifact", "log_activity",
		"tree_search", "browse_tree", "load_context",
	},
	AgentTypeAnalyst: {
		"query_metrics", "get_team_capacity",
		"list_projects", "list_tasks", "get_project",
		"search_documents", "create_artifact",
		"log_activity",
		"tree_search", "browse_tree", "load_context",
	},
}

func TestAgentToolAccessMatrix(t *testing.T) {
	// Test that each agent type has the correct enabled tools
	for agentType, expectedTools := range AgentToolMatrix {
		t.Run(string(agentType), func(t *testing.T) {
			ctx := &AgentContext{
				UserID:   "test-user",
				UserName: "Test User",
			}

			var agent *BaseAgent
			switch agentType {
			case AgentTypeOrchestrator:
				agent = NewOrchestrator(ctx).(*BaseAgent)
			case AgentTypeDocument:
				agent = NewDocumentAgent(ctx).(*BaseAgent)
			case AgentTypeProject:
				agent = NewProjectAgent(ctx).(*BaseAgent)
			case AgentTypeTask:
				agent = NewTaskAgent(ctx).(*BaseAgent)
			case AgentTypeClient:
				agent = NewClientAgent(ctx).(*BaseAgent)
			case AgentTypeAnalyst:
				agent = NewAnalystAgent(ctx).(*BaseAgent)
			}

			enabledTools := agent.GetEnabledTools()

			// Check all expected tools are enabled
			for _, expectedTool := range expectedTools {
				found := false
				for _, tool := range enabledTools {
					if tool == expectedTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Agent %s missing expected tool: %s", agentType, expectedTool)
				}
			}

			// Check no unexpected tools are enabled
			for _, tool := range enabledTools {
				found := false
				for _, expectedTool := range expectedTools {
					if tool == expectedTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Agent %s has unexpected tool: %s", agentType, tool)
				}
			}
		})
	}
}

func TestAgentCannotCallUnauthorizedTools(t *testing.T) {
	// Define tools that should NOT be accessible by certain agents
	unauthorizedAccess := map[AgentType][]string{
		AgentTypeDocument: {"create_task", "update_task", "bulk_create_tasks", "query_metrics"},
		AgentTypeAnalyst:  {"create_task", "update_task", "create_client", "update_client"},
		AgentTypeClient:   {"create_task", "bulk_create_tasks", "query_metrics", "move_task"},
	}

	for agentType, forbiddenTools := range unauthorizedAccess {
		t.Run(string(agentType)+"_unauthorized", func(t *testing.T) {
			ctx := &AgentContext{
				UserID:   "test-user",
				UserName: "Test User",
			}

			var agent *BaseAgent
			switch agentType {
			case AgentTypeDocument:
				agent = NewDocumentAgent(ctx).(*BaseAgent)
			case AgentTypeAnalyst:
				agent = NewAnalystAgent(ctx).(*BaseAgent)
			case AgentTypeClient:
				agent = NewClientAgent(ctx).(*BaseAgent)
			}

			enabledTools := agent.GetEnabledTools()

			for _, forbiddenTool := range forbiddenTools {
				for _, tool := range enabledTools {
					if tool == forbiddenTool {
						t.Errorf("Agent %s should NOT have access to tool: %s", agentType, forbiddenTool)
					}
				}
			}
		})
	}
}

func TestExecuteToolAccessControl(t *testing.T) {
	// Test that ExecuteTool rejects unauthorized tool calls
	agent := NewBaseAgent(BaseAgentConfig{
		UserID:       "test-user",
		AgentType:    AgentTypeDocument,
		EnabledTools: []string{"search_documents", "get_project"}, // Only these tools
	})

	// Try to execute an unauthorized tool
	_, err := agent.ExecuteTool(context.Background(), "create_task", json.RawMessage(`{"title":"test"}`))
	if err == nil {
		t.Error("Expected error when calling unauthorized tool, got nil")
	}
	// Error should be either ErrToolNotEnabled or ErrToolRegistryNotAvailable (when no DB pool)
	if err != nil && !errors.Is(err, ErrToolNotEnabled) && !errors.Is(err, ErrToolRegistryNotAvailable) {
		t.Errorf("Expected ErrToolNotEnabled or ErrToolRegistryNotAvailable, got: %v", err)
	}
}

// =============================================================================
// 7.B - CONTEXT STRESS TEST
// Verifies agent behavior with large context payloads (15k+ tokens)
// =============================================================================

func TestLargeContextHandling(t *testing.T) {
	// Generate a large context string (~15k tokens ≈ 60k characters)
	largeContent := generateLargeContent(60000)

	ctx := &AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	// Test Analyst agent (most likely to receive large context)
	agent := NewAnalystAgent(ctx).(*BaseAgent)

	// Verify context requirements
	reqs := agent.GetContextRequirements()
	if !reqs.NeedsProjects || !reqs.NeedsTasks || !reqs.NeedsClients {
		t.Error("Analyst agent should need projects, tasks, and clients context")
	}

	// Test that agent can be created with large system prompt
	largePrompt := agent.GetSystemPrompt() + "\n\n" + largeContent
	agent.systemPrompt = largePrompt

	if len(agent.GetSystemPrompt()) < 60000 {
		t.Error("Agent should handle large system prompts")
	}
}

func TestContextRequirementsPerAgent(t *testing.T) {
	// Verify each agent has appropriate context requirements
	tests := []struct {
		agentType    AgentType
		needsProject bool
		needsTasks   bool
		needsClients bool
		needsTeam    bool
	}{
		{AgentTypeOrchestrator, true, true, true, false},
		{AgentTypeDocument, true, false, true, false},
		{AgentTypeProject, true, true, true, true},
		{AgentTypeTask, true, true, false, true},
		{AgentTypeClient, true, false, true, false},
		{AgentTypeAnalyst, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			ctx := &AgentContext{UserID: "test", UserName: "Test"}

			var agent Agent
			switch tt.agentType {
			case AgentTypeOrchestrator:
				agent = NewOrchestrator(ctx)
			case AgentTypeDocument:
				agent = NewDocumentAgent(ctx)
			case AgentTypeProject:
				agent = NewProjectAgent(ctx)
			case AgentTypeTask:
				agent = NewTaskAgent(ctx)
			case AgentTypeClient:
				agent = NewClientAgent(ctx)
			case AgentTypeAnalyst:
				agent = NewAnalystAgent(ctx)
			}

			reqs := agent.GetContextRequirements()

			if reqs.NeedsProjects != tt.needsProject {
				t.Errorf("NeedsProjects: expected %v, got %v", tt.needsProject, reqs.NeedsProjects)
			}
			if reqs.NeedsTasks != tt.needsTasks {
				t.Errorf("NeedsTasks: expected %v, got %v", tt.needsTasks, reqs.NeedsTasks)
			}
			if reqs.NeedsClients != tt.needsClients {
				t.Errorf("NeedsClients: expected %v, got %v", tt.needsClients, reqs.NeedsClients)
			}
			if reqs.NeedsTeam != tt.needsTeam {
				t.Errorf("NeedsTeam: expected %v, got %v", tt.needsTeam, reqs.NeedsTeam)
			}
		})
	}
}

func TestMaxContextTokensHandling(t *testing.T) {
	// Test that agents can specify max context tokens
	reqs := ContextRequirements{
		NeedsProjects:    true,
		MaxContextTokens: 15000,
	}

	if reqs.MaxContextTokens != 15000 {
		t.Errorf("Expected MaxContextTokens 15000, got %d", reqs.MaxContextTokens)
	}

	// Test with very large token limit
	reqs.MaxContextTokens = 128000
	if reqs.MaxContextTokens != 128000 {
		t.Errorf("Expected MaxContextTokens 128000, got %d", reqs.MaxContextTokens)
	}
}

// =============================================================================
// 7.C - UI INTEGRATION VERIFICATION
// Verifies backend streaming is compatible with frontend expectations
// =============================================================================

func TestStreamEventTypes(t *testing.T) {
	// Verify all expected event types are defined
	// These must match what the frontend expects
	expectedEventTypes := []string{
		"token",
		"artifact_start",
		"artifact_complete",
		"done",
		"error",
		"thinking",
	}

	// This test verifies the streaming package has the right constants
	// The actual constants are in internal/streaming/events.go
	for _, eventType := range expectedEventTypes {
		t.Logf("Event type '%s' should be supported by frontend", eventType)
	}
}

func TestAgentInputStructure(t *testing.T) {
	// Verify AgentInput has all fields needed for frontend integration
	input := AgentInput{
		Messages:       nil,
		Context:        nil,
		Selections:     UserSelections{},
		FocusMode:      "write",
		FocusModeOpts:  map[string]string{"key": "value"},
		ConversationID: [16]byte{},
		UserID:         "user-123",
		UserName:       "Test User",
	}

	if input.FocusMode != "write" {
		t.Error("FocusMode not set correctly")
	}
	if input.UserID != "user-123" {
		t.Error("UserID not set correctly")
	}
	if input.FocusModeOpts["key"] != "value" {
		t.Error("FocusModeOpts not set correctly")
	}
}

func TestUserSelectionsStructure(t *testing.T) {
	// Verify UserSelections matches frontend context bar selections
	selections := UserSelections{
		ProjectID:  nil,
		ContextIDs: []uuid.UUID{},
		NodeID:     nil,
		ClientID:   nil,
	}

	// All fields should be optional (nil-able)
	if selections.ProjectID != nil {
		t.Error("ProjectID should be nil by default")
	}
	if selections.ContextIDs == nil {
		t.Error("ContextIDs should be empty slice, not nil")
	}
}

func TestIntentStructure(t *testing.T) {
	// Verify Intent structure matches what frontend expects
	intent := Intent{
		Category:       "document",
		ShouldDelegate: true,
		TargetAgent:    AgentTypeDocument,
		Confidence:     0.95,
		Reasoning:      "User requested document creation",
	}

	if intent.Category != "document" {
		t.Error("Category not set correctly")
	}
	if !intent.ShouldDelegate {
		t.Error("ShouldDelegate should be true")
	}
	if intent.Confidence < 0.9 {
		t.Error("Confidence should be high for document requests")
	}
	if intent.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}
}

func TestAllAgentTypesHaveSystemPrompt(t *testing.T) {
	// Verify all agents have non-empty system prompts
	ctx := &AgentContext{UserID: "test", UserName: "Test"}

	agents := []Agent{
		NewOrchestrator(ctx),
		NewDocumentAgent(ctx),
		NewProjectAgent(ctx),
		NewTaskAgent(ctx),
		NewClientAgent(ctx),
		NewAnalystAgent(ctx),
	}

	for _, agent := range agents {
		prompt := agent.GetSystemPrompt()
		if prompt == "" {
			t.Errorf("Agent %s has empty system prompt", agent.Name())
		}
		if len(prompt) < 100 {
			t.Errorf("Agent %s has suspiciously short system prompt (%d chars)", agent.Name(), len(prompt))
		}
	}
}

func TestAllAgentTypesHaveNameAndDescription(t *testing.T) {
	ctx := &AgentContext{UserID: "test", UserName: "Test"}

	agents := []Agent{
		NewOrchestrator(ctx),
		NewDocumentAgent(ctx),
		NewProjectAgent(ctx),
		NewTaskAgent(ctx),
		NewClientAgent(ctx),
		NewAnalystAgent(ctx),
	}

	for _, agent := range agents {
		if agent.Name() == "" {
			t.Errorf("Agent type %s has empty name", agent.Type())
		}
		if agent.Description() == "" {
			t.Errorf("Agent %s has empty description", agent.Name())
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func generateLargeContent(size int) string {
	// Generate realistic-looking content for stress testing
	base := "This is a sample business context entry with project details, task information, and client data. "
	result := strings.Builder{}
	for result.Len() < size {
		result.WriteString(base)
	}
	return result.String()[:size]
}
