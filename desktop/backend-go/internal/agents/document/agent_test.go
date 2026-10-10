package document

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

func TestDocumentAgentType(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Type() != agents.AgentTypeDocument {
		t.Errorf("Expected type %s, got %s", agents.AgentTypeDocument, agent.Type())
	}
}

func TestDocumentAgentName(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Name() == "" {
		t.Error("Agent name should not be empty")
	}
	if agent.Name() != "Document Specialist" {
		t.Errorf("Expected name 'Document Specialist', got '%s'", agent.Name())
	}
}

func TestDocumentAgentDescription(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)

	if agent.Description() == "" {
		t.Error("Agent description should not be empty")
	}
}

func TestDocumentAgentSystemPrompt(t *testing.T) {
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

func TestDocumentAgentContextRequirements(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	reqs := agent.GetContextRequirements()

	if !reqs.NeedsProjects {
		t.Error("Document agent should need projects context")
	}
	if !reqs.NeedsKnowledge {
		t.Error("Document agent should need knowledge context")
	}
	if !reqs.NeedsClients {
		t.Error("Document agent should need clients context")
	}
	if reqs.MaxContextTokens == 0 {
		t.Error("MaxContextTokens should be set")
	}
}

func TestDocumentAgentEnabledTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("Document agent should have enabled tools")
	}

	// Check for expected tools
	expectedTools := []string{"create_artifact", "search_documents", "get_project", "get_client"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Document agent should have tool: %s", expected)
		}
	}
}

func TestDocumentAgentNoUnauthorizedTools(t *testing.T) {
	ctx := &agents.AgentContext{
		UserID:   "test-user",
		UserName: "Test User",
	}

	agent := New(ctx)
	tools := agent.GetEnabledTools()

	// Document agent should NOT have write tools for tasks
	forbiddenTools := []string{"create_task", "update_task", "bulk_create_tasks"}
	for _, forbidden := range forbiddenTools {
		for _, tool := range tools {
			if tool == forbidden {
				t.Errorf("Document agent should NOT have tool: %s", forbidden)
			}
		}
	}
}
