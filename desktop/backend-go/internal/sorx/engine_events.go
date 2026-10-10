package sorx

import (
	"context"

	"github.com/google/uuid"
)

func (e *Engine) processEvents() {
	for {
		select {
		case event := <-e.events:
			e.handleEvent(event)
		case <-e.done:
			return
		}
	}
}

func (e *Engine) handleEvent(event Event) {
	switch event.Type {
	case EventDecisionMade:
		e.resumeFromDecision(event.ExecutionID, event.Data)
	case EventIntegrationConnected:
		e.logger.Info("Integration connected", "data", event.Data)
	case EventIntegrationDisconnected:
		e.logger.Info("Integration disconnected", "data", event.Data)
	}
}

func (e *Engine) resumeFromDecision(executionID uuid.UUID, data interface{}) {
	exec, ok := e.GetExecution(executionID)
	if !ok {
		e.logger.Error("Execution not found for decision", "execution_id", executionID)
		return
	}

	if exec.Status != StatusWaitingCallback {
		e.logger.Warn("Execution not waiting for callback", "execution_id", executionID, "status", exec.Status)
		return
	}

	// Store decision result and continue
	exec.Context["decision_result"] = data
	exec.Status = StatusRunning

	// Resume execution from current step
	skill, ok := e.skills.Load(exec.SkillID)
	if !ok {
		e.logger.Error("Skill not found", "skill_id", exec.SkillID)
		return
	}

	go e.runExecution(context.Background(), exec, skill.(*SkillDefinition))
}
