package task

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

func TestTaskAgentType(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Type() != agents.AgentTypeTask {
		t.Errorf("Expected type %s, got %s", agents.AgentTypeTask, agent.Type())
	}
}

func TestTaskAgentName(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Name() == "" {
		t.Error("Agent name should not be empty")
	}
	if agent.Name() != "Task Specialist" {
		t.Errorf("Expected name 'Task Specialist', got '%s'", agent.Name())
	}
}

func TestTaskAgentDescription(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Description() == "" {
		t.Error("Agent description should not be empty")
	}
}

func TestTaskAgentSystemPrompt(t *testing.T) {
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

func TestTaskAgentContextRequirements(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if !reqs.NeedsProjects {
		t.Error("Task agent should need projects context")
	}
	if !reqs.NeedsTasks {
		t.Error("Task agent should need tasks context")
	}
	if !reqs.NeedsTeam {
		t.Error("Task agent should need team context")
	}
	if reqs.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be set")
	}
}

func TestTaskAgentEnabledTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("Task agent should have enabled tools")
	}

	// Check for expected tools
	expectedTools := []string{"create_task", "update_task", "get_task", "list_tasks", "bulk_create_tasks", "assign_task"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Task agent should have tool: %s", expected)
		}
	}
}

func TestTaskAgentNoUnauthorizedTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	// Task agent should NOT have client or analytics tools
	forbiddenTools := []string{"create_client", "update_client", "query_metrics", "create_artifact"}
	for _, forbidden := range forbiddenTools {
		for _, tool := range tools {
			if tool == forbidden {
				t.Errorf("Task agent should NOT have tool: %s", forbidden)
			}
		}
	}
}

func TestTaskAgentPrioritySections(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if len(reqs.PrioritySections) == 0 {
		t.Error("Task agent should have priority sections defined")
	}

	// Check for expected priority sections
	expectedSections := []string{"active_tasks", "project_tasks", "team_capacity"}
	for _, expected := range expectedSections {
		found := false
		for _, section := range reqs.PrioritySections {
			if section == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Task agent should have priority section: %s", expected)
		}
	}
}
