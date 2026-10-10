package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Registry manages registered RunnableAgent instances. It is not a global
// singleton — always create one with NewRegistry.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]RunnableAgent
	logger *slog.Logger
}

// NewRegistry creates a new, empty agent registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		agents: make(map[string]RunnableAgent),
		logger: logger,
	}
}

// Register adds an agent to the registry. Returns an error if an agent with
// the same ID is already registered.
func (r *Registry) Register(ctx context.Context, agent RunnableAgent) error {
	if agent == nil {
		return fmt.Errorf("register agent: agent must not be nil")
	}

	id := agent.AgentID()
	if id == "" {
		return fmt.Errorf("register agent: agent ID must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		return fmt.Errorf("register agent %q: %w", id, ErrAgentAlreadyRegistered)
	}

	r.agents[id] = agent

	r.logger.InfoContext(ctx, "agent registered",
		slog.String("agent_id", id),
		slog.String("agent_name", agent.AgentName()),
	)

	return nil
}

// Get retrieves an agent by ID. Returns an error (not a panic) if not found.
func (r *Registry) Get(ctx context.Context, id string) (RunnableAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[id]
	if !ok {
		return nil, fmt.Errorf("get agent %q: %w", id, ErrAgentNotFound)
	}

	r.logger.DebugContext(ctx, "agent retrieved",
		slog.String("agent_id", id),
	)

	return agent, nil
}

// List returns all registered agents. The order of the returned slice is
// non-deterministic.
func (r *Registry) List(ctx context.Context) []RunnableAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]RunnableAgent, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, a)
	}

	r.logger.DebugContext(ctx, "agents listed",
		slog.Int("count", len(result)),
	)

	return result
}

// Unregister removes an agent by ID. Returns an error if the agent is not found.
func (r *Registry) Unregister(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.agents[id]; !ok {
		return fmt.Errorf("unregister agent %q: %w", id, ErrAgentNotFound)
	}

	delete(r.agents, id)

	r.logger.InfoContext(ctx, "agent unregistered",
		slog.String("agent_id", id),
	)

	return nil
}

// FindByCapability returns all agents that have the given capability. The order
// of the returned slice is non-deterministic.
func (r *Registry) FindByCapability(ctx context.Context, cap AgentCapability) []RunnableAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []RunnableAgent
	for _, a := range r.agents {
		for _, c := range a.Capabilities() {
			if c == cap {
				result = append(result, a)
				break
			}
		}
	}

	r.logger.DebugContext(ctx, "agents found by capability",
		slog.String("capability", string(cap)),
		slog.Int("count", len(result)),
	)

	return result
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.agents)
}

// Sentinel errors for registry operations.
var (
	// ErrAgentNotFound is returned when a requested agent ID does not exist in
	// the registry.
	ErrAgentNotFound = fmt.Errorf("agent not found")

	// ErrAgentAlreadyRegistered is returned when an agent with the same ID is
	// already present in the registry.
	ErrAgentAlreadyRegistered = fmt.Errorf("agent already registered")
)
