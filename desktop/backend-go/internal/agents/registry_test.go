package agents_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/rhl/businessos-backend/internal/agents"
)

// mockAgent is a test double that satisfies the RunnableAgent interface.
type mockAgent struct {
	id   string
	name string
	caps []agents.AgentCapability
}

func (m *mockAgent) AgentID() string                          { return m.id }
func (m *mockAgent) AgentName() string                        { return m.name }
func (m *mockAgent) Capabilities() []agents.AgentCapability   { return m.caps }
func (m *mockAgent) AgentTools() []agents.Tool                { return nil }
func (m *mockAgent) AgentStatus() agents.AgentStatus          { return agents.AgentStatusIdle }
func (m *mockAgent) Execute(_ context.Context, _ agents.AgentTask) (*agents.TaskResult, error) {
	return &agents.TaskResult{Content: "ok"}, nil
}

// noopWriter discards all log output so test output stays clean.
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// newTestRegistry creates a Registry backed by a no-op slog logger.
func newTestRegistry() *agents.Registry {
	return agents.NewRegistry(slog.New(slog.NewTextHandler(noopWriter{}, nil)))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewRegistry(t *testing.T) {
	r := newTestRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 agents, got %d", r.Count())
	}
}

func TestRegister(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	a := &mockAgent{
		id:   "agent-1",
		name: "Agent One",
		caps: []agents.AgentCapability{agents.CapabilityChat},
	}

	if err := r.Register(ctx, a); err != nil {
		t.Fatalf("Register returned unexpected error: %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("expected 1 agent after Register, got %d", r.Count())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	a := &mockAgent{id: "agent-dup", name: "Dup Agent"}

	if err := r.Register(ctx, a); err != nil {
		t.Fatalf("first Register returned unexpected error: %v", err)
	}

	err := r.Register(ctx, a)
	if err == nil {
		t.Fatal("expected error on duplicate Register, got nil")
	}

	if !errors.Is(err, agents.ErrAgentAlreadyRegistered) {
		t.Errorf("expected ErrAgentAlreadyRegistered, got: %v", err)
	}
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	a := &mockAgent{id: "agent-get", name: "Get Agent"}
	if err := r.Register(ctx, a); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := r.Get(ctx, "agent-get")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if got.AgentID() != "agent-get" {
		t.Errorf("expected ID %q, got %q", "agent-get", got.AgentID())
	}
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	_, err := r.Get(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for missing agent, got nil")
	}

	if !errors.Is(err, agents.ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound, got: %v", err)
	}
}

func TestFindByCapability(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	chat := &mockAgent{
		id:   "chat-agent",
		name: "Chat",
		caps: []agents.AgentCapability{agents.CapabilityChat},
	}
	code := &mockAgent{
		id:   "code-agent",
		name: "Code",
		caps: []agents.AgentCapability{agents.CapabilityCode},
	}
	multi := &mockAgent{
		id:   "multi-agent",
		name: "Multi",
		caps: []agents.AgentCapability{agents.CapabilityChat, agents.CapabilityCode},
	}

	for _, a := range []agents.RunnableAgent{chat, code, multi} {
		if err := r.Register(ctx, a); err != nil {
			t.Fatalf("Register %q: %v", a.AgentID(), err)
		}
	}

	chatAgents := r.FindByCapability(ctx, agents.CapabilityChat)
	if len(chatAgents) != 2 {
		t.Errorf("FindByCapability(chat): expected 2, got %d", len(chatAgents))
	}

	codeAgents := r.FindByCapability(ctx, agents.CapabilityCode)
	if len(codeAgents) != 2 {
		t.Errorf("FindByCapability(code): expected 2, got %d", len(codeAgents))
	}

	searchAgents := r.FindByCapability(ctx, agents.CapabilitySearch)
	if len(searchAgents) != 0 {
		t.Errorf("FindByCapability(search): expected 0, got %d", len(searchAgents))
	}
}

func TestUnregister(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	a := &mockAgent{id: "agent-rm", name: "Remove Me"}
	if err := r.Register(ctx, a); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := r.Unregister(ctx, "agent-rm"); err != nil {
		t.Fatalf("Unregister returned unexpected error: %v", err)
	}

	if r.Count() != 0 {
		t.Errorf("expected 0 agents after Unregister, got %d", r.Count())
	}

	// Unregistering a second time must return ErrAgentNotFound.
	err := r.Unregister(ctx, "agent-rm")
	if err == nil {
		t.Fatal("expected error on second Unregister, got nil")
	}

	if !errors.Is(err, agents.ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound, got: %v", err)
	}
}

func TestCount(t *testing.T) {
	ctx := context.Background()
	r := newTestRegistry()

	if r.Count() != 0 {
		t.Fatalf("initial Count: expected 0, got %d", r.Count())
	}

	for i := range 3 {
		a := &mockAgent{
			id:   fmt.Sprintf("a-%d", i),
			name: fmt.Sprintf("Agent %d", i),
		}
		if err := r.Register(ctx, a); err != nil {
			t.Fatalf("Register a-%d: %v", i, err)
		}
	}

	if r.Count() != 3 {
		t.Errorf("Count after 3 registrations: expected 3, got %d", r.Count())
	}
}
