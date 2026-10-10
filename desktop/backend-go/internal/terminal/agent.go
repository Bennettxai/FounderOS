package terminal

import (
	"fmt"
	"log/slog"
	"sync"
)

// AgentType identifies CLI agent tools
type AgentType string

const (
	AgentClaude AgentType = "claude"
	AgentCodex  AgentType = "codex"
	AgentOllama AgentType = "ollama"
)

// AgentCommands maps agent types to their launch commands
var AgentCommands = map[AgentType]string{
	AgentClaude: "claude --dangerously-skip-permissions",
	AgentCodex:  "codex --full-auto",
	AgentOllama: "ollama run",
}

// AgentTracker tracks which sessions have running agents
type AgentTracker struct {
	mu       sync.RWMutex
	sessions map[string]AgentType // sessionID -> running agent type
}

// NewAgentTracker creates a new agent tracker
func NewAgentTracker() *AgentTracker {
	return &AgentTracker{
		sessions: make(map[string]AgentType),
	}
}

// MarkAgentRunning records that an agent was launched in a session
func (t *AgentTracker) MarkAgentRunning(sessionID string, agent AgentType) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessions[sessionID] = agent
	slog.Info("[Terminal] Agent marked as running", "session_id", sessionID[:8], "agent", string(agent))
}

// MarkAgentStopped clears the agent tracking for a session
func (t *AgentTracker) MarkAgentStopped(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sessions, sessionID)
	slog.Info("[Terminal] Agent marked as stopped", "session_id", sessionID[:8])
}

// GetRunningAgent returns the agent type running in a session, or empty string
func (t *AgentTracker) GetRunningAgent(sessionID string) AgentType {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sessions[sessionID]
}

// GetAgentCommand returns the CLI command string for launching an agent
func GetAgentCommand(agent AgentType, model string) (string, error) {
	cmd, ok := AgentCommands[agent]
	if !ok {
		return "", fmt.Errorf("unknown agent type: %s", agent)
	}

	// Ollama needs a model name appended
	if agent == AgentOllama && model != "" {
		cmd = cmd + " " + model
	}

	return cmd + "\n", nil
}
