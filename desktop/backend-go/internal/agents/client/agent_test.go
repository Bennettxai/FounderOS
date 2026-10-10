package client

import (
	"testing"

	"github.com/rhl/businessos-backend/internal/agents"
)

func TestNew(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	if agent == nil {
		t.Fatal("New returned nil")
	}
}

func TestClientAgentType(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Type() != agents.AgentTypeClient {
		t.Errorf("Expected type %s, got %s", agents.AgentTypeClient, agent.Type())
	}
}

func TestClientAgentName(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Name() == "" {
		t.Error("Agent name should not be empty")
	}
	if agent.Name() != "Client Relationship Manager" {
		t.Errorf("Expected name 'Client Relationship Manager', got '%s'", agent.Name())
	}
}

func TestClientAgentDescription(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Description() == "" {
		t.Error("Agent description should not be empty")
	}
}

func TestClientAgentSystemPrompt(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	prompt := agent.GetSystemPrompt()
	if prompt == "" {
		t.Error("System prompt should not be empty")
	}
	if len(prompt) < 100 {
		t.Error("System prompt should have substantial content")
	}
}

func TestClientAgentContextRequirements(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if !reqs.NeedsClients {
		t.Error("Client agent should need clients context")
	}
	if !reqs.NeedsProjects {
		t.Error("Client agent should need projects context")
	}
	if reqs.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be set")
	}
}

func TestClientAgentEnabledTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("Client agent should have enabled tools")
	}

	// Check for expected tools
	expectedTools := []string{"get_client", "create_client", "update_client", "log_client_interaction", "update_client_pipeline"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Client agent should have tool: %s", expected)
		}
	}
}

func TestClientAgentNoUnauthorizedTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	// Client agent should NOT have task creation tools
	forbiddenTools := []string{"create_task", "update_task", "bulk_create_tasks", "query_metrics"}
	for _, forbidden := range forbiddenTools {
		for _, tool := range tools {
			if tool == forbidden {
				t.Errorf("Client agent should NOT have tool: %s", forbidden)
			}
		}
	}
}
