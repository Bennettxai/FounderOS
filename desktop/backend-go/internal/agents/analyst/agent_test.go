package analyst

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

func TestAnalystAgentType(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Type() != agents.AgentTypeAnalyst {
		t.Errorf("Expected type %s, got %s", agents.AgentTypeAnalyst, agent.Type())
	}
}

func TestAnalystAgentName(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Name() == "" {
		t.Error("Agent name should not be empty")
	}
	if agent.Name() != "Business Analyst" {
		t.Errorf("Expected name 'Business Analyst', got '%s'", agent.Name())
	}
}

func TestAnalystAgentDescription(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Description() == "" {
		t.Error("Agent description should not be empty")
	}
}

func TestAnalystAgentSystemPrompt(t *testing.T) {
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

func TestAnalystAgentContextRequirements(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if !reqs.NeedsProjects {
		t.Error("Analyst agent should need projects context")
	}
	if !reqs.NeedsMetrics {
		t.Error("Analyst agent should need metrics context")
	}
	if reqs.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be set")
	}
}

func TestAnalystAgentEnabledTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("Analyst agent should have enabled tools")
	}

	// Check for expected tools
	expectedTools := []string{"query_metrics", "get_team_capacity", "list_projects", "list_tasks"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Analyst agent should have tool: %s", expected)
		}
	}
}

func TestAnalystAgentNoUnauthorizedTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	// Analyst agent should NOT have write tools
	forbiddenTools := []string{"create_task", "update_task", "create_client", "update_client"}
	for _, forbidden := range forbiddenTools {
		for _, tool := range tools {
			if tool == forbidden {
				t.Errorf("Analyst agent should NOT have tool: %s", forbidden)
			}
		}
	}
}
