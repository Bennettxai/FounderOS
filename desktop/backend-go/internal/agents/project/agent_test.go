package project

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

func TestProjectAgentType(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Type() != agents.AgentTypeProject {
		t.Errorf("Expected type %s, got %s", agents.AgentTypeProject, agent.Type())
	}
}

func TestProjectAgentName(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Name() == "" {
		t.Error("Agent name should not be empty")
	}
	if agent.Name() != "Project Manager" {
		t.Errorf("Expected name 'Project Manager', got '%s'", agent.Name())
	}
}

func TestProjectAgentDescription(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Description() == "" {
		t.Error("Agent description should not be empty")
	}
}

func TestProjectAgentSystemPrompt(t *testing.T) {
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

func TestProjectAgentContextRequirements(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if !reqs.NeedsProjects {
		t.Error("Project agent should need projects context")
	}
	if !reqs.NeedsTasks {
		t.Error("Project agent should need tasks context")
	}
	if !reqs.NeedsTeam {
		t.Error("Project agent should need team context")
	}
	if reqs.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be set")
	}
}

func TestProjectAgentEnabledTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("Project agent should have enabled tools")
	}

	// Check for expected tools
	expectedTools := []string{"create_project", "update_project", "create_task", "bulk_create_tasks", "get_team_capacity"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Project agent should have tool: %s", expected)
		}
	}
}

func TestProjectAgentNoUnauthorizedTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	// Project agent should NOT have client-specific tools
	forbiddenTools := []string{"create_client", "update_client", "log_client_interaction", "query_metrics"}
	for _, forbidden := range forbiddenTools {
		for _, tool := range tools {
			if tool == forbidden {
				t.Errorf("Project agent should NOT have tool: %s", forbidden)
			}
		}
	}
}
