package hooks

import (
	"context"
	"sync"
	"time"
)

// HookEvent represents lifecycle events in agent execution
type HookEvent string

const (
	PreAgentExecute  HookEvent = "PreAgentExecute"  // Before agent starts
	PostAgentExecute HookEvent = "PostAgentExecute" // After agent completes
	OnAgentError     HookEvent = "OnAgentError"     // When agent fails
	OnComplete       HookEvent = "OnComplete"       // When all agents complete
)

// HookContext contains all data for hook execution
type HookContext struct {
	AgentType  interface{} // Can be agent.AgentType, string, or any agent identifier
	TaskID     string
	Input      interface{}
	Output     interface{}
	Error      error
	Metadata   map[string]interface{}
	TokensUsed int
	StartTime  time.Time
	EndTime    time.Time
}

// Hook defines the interface for lifecycle hooks
type Hook interface {
	Name() string
	Execute(ctx context.Context, hookCtx HookContext) error
}

// HookRegistry manages hook registration and triggering
type HookRegistry struct {
	hooks map[HookEvent][]Hook
	mu    sync.RWMutex
}

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		hooks: make(map[HookEvent][]Hook),
	}
}

// Register adds a hook for a specific event
func (r *HookRegistry) Register(event HookEvent, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[event] = append(r.hooks[event], hook)
}

// Trigger executes all hooks for an event
func (r *HookRegistry) Trigger(ctx context.Context, event HookEvent, hookCtx HookContext) error {
	r.mu.RLock()
	eventHooks := r.hooks[event]
	r.mu.RUnlock()

	for _, hook := range eventHooks {
		if err := hook.Execute(ctx, hookCtx); err != nil {
			// Log but don't fail - hooks should not block agent execution
			// slog will be used in the hook implementation
			continue
		}
	}
	return nil
}

// GetHookCount returns the number of hooks registered for an event
func (r *HookRegistry) GetHookCount(event HookEvent) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.hooks[event])
}
