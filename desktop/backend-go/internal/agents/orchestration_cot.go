package agents

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AddStep adds a new step to the chain
func (cot *ChainOfThought) AddStep(step *ThoughtStep) {
	cot.mu.Lock()
	defer cot.mu.Unlock()
	step.ID = uuid.New().String()
	step.CreatedAt = time.Now()
	step.Status = "pending"
	cot.Steps = append(cot.Steps, step)
}

// UpdateStep updates an existing step
func (cot *ChainOfThought) UpdateStep(stepID string, output string, status string) {
	cot.mu.Lock()
	defer cot.mu.Unlock()
	for _, step := range cot.Steps {
		if step.ID == stepID {
			step.Output = output
			step.Status = status
			if status == "completed" || status == "failed" {
				now := time.Now()
				step.CompletedAt = &now
				step.Duration = now.Sub(step.CreatedAt)
			}
			break
		}
	}
}

// GetAgentsInvolved returns unique list of agents used
func (cot *ChainOfThought) GetAgentsInvolved() []AgentType {
	cot.mu.RLock()
	defer cot.mu.RUnlock()
	agentMap := make(map[AgentType]bool)
	for _, step := range cot.Steps {
		agentMap[step.Agent] = true
	}
	agents := make([]AgentType, 0, len(agentMap))
	for agent := range agentMap {
		agents = append(agents, agent)
	}
	return agents
}

// GetCOTAsJSON returns the chain of thought as JSON for debugging/logging
func (cot *ChainOfThought) GetCOTAsJSON() (string, error) {
	cot.mu.RLock()
	defer cot.mu.RUnlock()
	data, err := json.MarshalIndent(cot, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
