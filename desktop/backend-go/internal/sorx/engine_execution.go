package sorx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecuteSkill starts a skill execution.
func (e *Engine) ExecuteSkill(ctx context.Context, req ExecuteRequest) (*Execution, error) {
	// Validate skill exists
	skillDef, ok := e.skills.Load(req.SkillID)
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", req.SkillID)
	}
	skill := skillDef.(*SkillDefinition)

	// Check required integrations
	for _, provider := range skill.RequiredIntegrations {
		hasAccess, err := e.checkIntegrationAccess(ctx, req.UserID, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to check integration %s: %w", provider, err)
		}
		if !hasAccess {
			return nil, fmt.Errorf("integration %s not connected", provider)
		}
	}

	// Create execution record
	exec := &Execution{
		ID:          uuid.New(),
		SkillID:     req.SkillID,
		UserID:      req.UserID,
		Status:      StatusPending,
		Params:      req.Params,
		Context:     make(map[string]interface{}),
		StepResults: make(map[string]interface{}),
		StartedAt:   time.Now().UTC(),
	}

	// Store execution
	e.executions.Store(exec.ID.String(), exec)

	// Persist to database
	if err := e.persistExecution(ctx, exec); err != nil {
		e.logger.Error("Failed to persist execution", "error", err, "execution_id", exec.ID)
	}

	// Start execution in background
	go e.runExecution(ctx, exec, skill)

	return exec, nil
}

// GetExecution retrieves an execution by ID.
func (e *Engine) GetExecution(id uuid.UUID) (*Execution, bool) {
	val, ok := e.executions.Load(id.String())
	if !ok {
		return nil, false
	}
	return val.(*Execution), true
}

// runExecution processes a skill execution.
func (e *Engine) runExecution(ctx context.Context, exec *Execution, skill *SkillDefinition) {
	// Inject the skill tier into execution context so action handlers can
	// read it for CARRIER routing decisions without needing engine access.
	exec.Context["_skill_tier"] = int(skill.Tier)

	exec.Status = StatusRunning
	e.updateExecution(ctx, exec)

	// Execute each step
	for i, step := range skill.Steps {
		exec.CurrentStep = i

		e.logger.Info("Executing step",
			"execution_id", exec.ID,
			"step_id", step.ID,
			"step_type", step.Type)

		result, err := e.executeStep(ctx, exec, &step)
		if err != nil {
			exec.Status = StatusFailed
			exec.Error = err.Error()
			e.updateExecution(ctx, exec)
			return
		}

		// Store step result
		exec.StepResults[step.ID] = result

		// Check if step requires human decision
		if step.RequiresDecision && result != nil {
			if decision, ok := result.(map[string]interface{}); ok {
				if decision["status"] == "pending" {
					exec.Status = StatusWaitingCallback
					e.updateExecution(ctx, exec)
					return // Will be resumed when decision is made
				}
			}
		}
	}

	// All steps complete
	exec.Status = StatusComplete
	exec.CompletedAt = timePtr(time.Now().UTC())
	e.updateExecution(ctx, exec)

	e.logger.Info("Execution completed", "execution_id", exec.ID)
}

// executeStep runs a single step in the skill.
func (e *Engine) executeStep(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	switch step.Type {
	case StepTypeAction:
		return e.executeAction(ctx, exec, step)
	case StepTypeDecision:
		return e.requestDecision(ctx, exec, step)
	case StepTypeCondition:
		return e.evaluateCondition(ctx, exec, step)
	case StepTypeLoop:
		return e.executeLoop(ctx, exec, step)
	case StepTypeParallel:
		return e.executeParallel(ctx, exec, step)
	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// executeAction performs an integration action.
func (e *Engine) executeAction(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	// Get action handler
	handler, ok := actionHandlers[step.Action]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", step.Action)
	}

	// Get integration credentials if needed
	var creds *Credentials
	if step.Integration != "" {
		var err error
		creds, err = e.getCredentials(ctx, exec.UserID, step.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to get credentials for %s: %w", step.Integration, err)
		}
	}

	// Execute action
	return handler(ctx, ActionContext{
		Execution:   exec,
		Step:        step,
		Credentials: creds,
		Params:      step.Params,
	})
}

// requestDecision creates a human-in-the-loop decision.
func (e *Engine) requestDecision(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	decisionID := uuid.New()

	// Insert pending decision
	_, err := e.pool.Exec(ctx, `
		INSERT INTO pending_decisions (
			id, execution_id, skill_id, step_id, user_id,
			question, options, input_fields, context, priority, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
	`, decisionID, exec.ID, exec.SkillID, step.ID, exec.UserID,
		step.DecisionQuestion, step.DecisionOptions, step.InputFields,
		exec.Context, step.Priority)

	if err != nil {
		return nil, fmt.Errorf("failed to create decision: %w", err)
	}

	e.logger.Info("Awaiting human decision",
		"execution_id", exec.ID,
		"decision_id", decisionID,
		"question", step.DecisionQuestion)

	return map[string]interface{}{
		"status":      "pending",
		"decision_id": decisionID,
	}, nil
}

// evaluateCondition checks a condition and returns the branch to take.
func (e *Engine) evaluateCondition(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	// Simple condition evaluation
	// In production, this would use a proper expression evaluator
	condition := step.Condition
	if condition == "" {
		return map[string]interface{}{"branch": "default"}, nil
	}

	// For now, just return the condition result from params if present
	if result, ok := exec.Params[condition]; ok {
		return map[string]interface{}{"branch": result}, nil
	}

	return map[string]interface{}{"branch": "default"}, nil
}

// executeLoop runs steps in a loop.
func (e *Engine) executeLoop(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	// Get items to iterate over
	items, ok := step.Params["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("loop step requires 'items' array in params")
	}

	var results []interface{}
	for i, item := range items {
		exec.Context["loop_index"] = i
		exec.Context["loop_item"] = item

		// Execute loop body (substeps)
		for _, substep := range step.Substeps {
			result, err := e.executeStep(ctx, exec, &substep)
			if err != nil {
				return nil, fmt.Errorf("loop iteration %d failed: %w", i, err)
			}
			results = append(results, result)
		}
	}

	return map[string]interface{}{"results": results}, nil
}

// executeParallel runs steps in parallel.
func (e *Engine) executeParallel(ctx context.Context, exec *Execution, step *Step) (interface{}, error) {
	var wg sync.WaitGroup
	results := make(map[string]interface{})
	var mu sync.Mutex
	var firstErr error

	for _, substep := range step.Substeps {
		wg.Add(1)
		go func(s Step) {
			defer wg.Done()
			result, err := e.executeStep(ctx, exec, &s)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			results[s.ID] = result
		}(substep)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// ExecuteAction looks up an action by name in the package-level action handler
// registry and invokes it with the provided params. This is the public entry
// point used by callers outside the engine (e.g. the Optimal proactive
// consumer) that need to run an integration action without a full skill
// execution context.
//
// Returns an error if no handler is registered for the given action name.
func (e *Engine) ExecuteAction(ctx context.Context, action string, params map[string]any) (any, error) {
	handler, ok := actionHandlers[action]
	if !ok {
		return nil, fmt.Errorf("sorx: unknown action %q", action)
	}

	// Build a minimal ActionContext. The Optimal proactive consumer supplies
	// params directly; credentials are not passed here — any action that
	// requires credentials must look them up from the params (e.g. user_id).
	ac := ActionContext{
		Params: params,
	}

	return handler(ctx, ac)
}
