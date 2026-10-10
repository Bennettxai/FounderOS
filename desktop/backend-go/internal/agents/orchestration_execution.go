package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// cotPrompt is the shared COT system prompt injected into agents
const cotPrompt = `
CRITICAL: You MUST use Chain of Thought (COT) reasoning for this request.

## COT PROCESS (MANDATORY - MINIMUM 3 STEPS):

1. **Understanding** (Step 1/3+):
   - What is being asked?
   - What context is relevant?
   - What are the key constraints?

2. **Analysis** (Step 2/3+):
   - What are the possible approaches?
   - What are the trade-offs?
   - What additional information might be needed?

3. **Planning** (Step 3/3+):
   - What is the best approach given the analysis?
   - How should the response be structured?
   - What specific details need to be included?

4. **Additional Steps** (if needed):
   - Add more reasoning steps as complexity requires
   - Each step should build on previous insights

## OUTPUT FORMAT:
After your thinking process, provide your final response clearly and comprehensively.

IMPORTANT: Show your thinking process naturally in your response, then provide the final answer.
`

// ProcessWithCOT processes a user request with full chain of thought tracking
func (o *OrchestratorCOT) ProcessWithCOT(
	ctx context.Context,
	input AgentInput,
	userID string,
	userName string,
	conversationID *uuid.UUID,
	llmOptions services.LLMOptions,
) (<-chan streaming.StreamEvent, <-chan error, *ChainOfThought) {
	events := make(chan streaming.StreamEvent, 100)
	errs := make(chan error, 1)

	// Get user message
	var userMessage string
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if strings.ToLower(input.Messages[i].Role) == "user" {
			userMessage = input.Messages[i].Content
			break
		}
	}

	slog.Default().Debug("[COT] ProcessWithCOT called",
		"user", userID,
		"message", userMessage,
		"message_count", len(input.Messages))

	// Create chain of thought
	cot := NewChainOfThought(userMessage)
	o.chainsMu.Lock()
	o.activeChains[cot.ID] = cot
	o.chainsMu.Unlock()

	go func() {
		defer close(events)
		defer close(errs)
		startTime := time.Now()

		// Step 1: Analysis - Understand the request
		analysisStep := &ThoughtStep{
			Agent:     AgentTypeOrchestrator,
			Action:    "analyze",
			Input:     userMessage,
			Reasoning: "Analyzing user request to determine intent and required agents",
		}
		cot.AddStep(analysisStep)
		analysisStep.Status = "running"

		// Send thinking event
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeToken,
			Data: "Analyzing request...\n\n",
		}

		// Classify intent using SmartIntentRouter
		intent := o.router.ClassifyIntent(ctx, input.Messages, input.Context)
		analysisStep.Output = fmt.Sprintf("Intent: %s (confidence: %.2f)", intent.TargetAgent, intent.Confidence)
		analysisStep.Confidence = intent.Confidence
		cot.UpdateStep(analysisStep.ID, analysisStep.Output, "completed")

		// Step 2: Planning - Decide execution strategy
		planStep := &ThoughtStep{
			Agent:     AgentTypeOrchestrator,
			Action:    "plan",
			Input:     analysisStep.Output,
			Reasoning: "Creating execution plan based on intent analysis",
		}
		cot.AddStep(planStep)
		planStep.Status = "running"

		plan := o.createExecutionPlan(ctx, userMessage, intent, input)
		planStep.Output = fmt.Sprintf("Strategy: %s, Primary: %s, Steps: %d",
			plan.Strategy, plan.PrimaryAgent, len(plan.Steps))
		planStep.Confidence = plan.Confidence
		cot.UpdateStep(planStep.ID, planStep.Output, "completed")

		// REMOVED: Routing box confuses users - they think it's the final response
		// routingBox := o.formatRoutingBox(intent, plan)
		// events <- streaming.StreamEvent{
		// 	Type: streaming.EventTypeToken,
		// 	Data: routingBox,
		// }

		cot.Status = "executing"
		slog.Default().Debug("[COT] Starting execution", "strategy", plan.Strategy)

		// Step 3: Execute based on strategy
		switch plan.Strategy {
		case "direct":
			// Orchestrator handles directly
			o.executeDirectly(ctx, cot, input, llmOptions, events, errs)

		case "delegate":
			// Single agent delegation
			o.executeDelegation(ctx, cot, plan, input, userID, userName, conversationID, llmOptions, events, errs)

		case "multi-agent":
			// Multiple agents in parallel
			o.executeMultiAgent(ctx, cot, plan, input, userID, userName, conversationID, llmOptions, events, errs)

		case "sequential":
			// Agents in sequence
			o.executeSequential(ctx, cot, plan, input, userID, userName, conversationID, llmOptions, events, errs)
		}

		// Step 4: Synthesis (if multiple agents were involved)
		if len(cot.GetAgentsInvolved()) > 1 {
			cot.Status = "synthesizing"
			o.synthesizeResults(ctx, cot, events)
		}

		// Complete
		cot.Status = "completed"
		cot.TotalDuration = time.Since(startTime)
		cot.AgentsInvolved = cot.GetAgentsInvolved()

		// Send completion event with COT summary
		cotSummary := o.formatCOTSummary(cot)
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeToken,
			Data: cotSummary,
		}
	}()

	return events, errs, cot
}

// executeDirectly handles the request directly via orchestrator
func (o *OrchestratorCOT) executeDirectly(
	ctx context.Context,
	cot *ChainOfThought,
	input AgentInput,
	llmOptions services.LLMOptions,
	events chan<- streaming.StreamEvent,
	errs chan<- error,
) {
	step := &ThoughtStep{
		Agent:     AgentTypeOrchestrator,
		Action:    "execute",
		Input:     cot.UserMessage,
		Reasoning: "Handling directly as general request",
	}
	cot.AddStep(step)
	step.Status = "running"

	// Get orchestrator agent and set LLM options
	agent := o.registry.GetAgent(AgentTypeOrchestrator, input.UserID, input.UserName, &input.ConversationID, input.Context)
	agent.SetOptions(llmOptions)

	// Add COT-specific system prompt to force extended thinking
	agent.SetFocusModePrompt(cotPrompt)
	slog.Default().Debug("[COT] Injected COT prompt (forces minimum 3 thinking steps)")

	// Inject role and memory contexts if available (Feature: Memory Hierarchy + Role-based permissions)
	if input.RoleContext != "" {
		agent.SetRoleContextPrompt(input.RoleContext)
		slog.Default().Debug("[COT] Injected role context into orchestrator",
			"chars", len(input.RoleContext))
	}
	if input.MemoryContext != "" {
		agent.SetMemoryContext(input.MemoryContext)
		slog.Default().Debug("[COT] Injected memory context into orchestrator",
			"chars", len(input.MemoryContext))
	}

	slog.Default().Debug("[COT] executeDirectly: Starting agent.Run() for orchestrator")
	agentEvents, agentErrs := agent.Run(ctx, input)
	slog.Default().Debug("[COT] executeDirectly: agent.Run() returned, starting event loop")

	var output strings.Builder
	eventCount := 0
	for {
		select {
		case event, ok := <-agentEvents:
			if !ok {
				slog.Default().Debug("[COT] executeDirectly: agentEvents channel closed",
					"total_events", eventCount,
					"output_length", output.Len())
				step.Output = output.String()
				cot.UpdateStep(step.ID, step.Output, "completed")
				cot.FinalOutput = output.String()
				events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
				return
			}
			eventCount++
			slog.Default().Debug("[COT] executeDirectly: Received event",
				"event_number", eventCount,
				"type", event.Type)
			events <- event
			if event.Type == streaming.EventTypeToken {
				if content, ok := event.Data.(string); ok {
					output.WriteString(content)
					slog.Default().Debug("[COT] executeDirectly: Token content",
						"content_length", len(content),
						"total_output", output.Len())
				}
			}
		case err := <-agentErrs:
			if err != nil {
				slog.Default().Error("[COT] executeDirectly: ERROR received", "error", err)
				step.Error = err.Error()
				cot.UpdateStep(step.ID, "", "failed")
				errs <- err
			} else {
				slog.Default().Debug("[COT] executeDirectly: agentErrs channel closed with nil error")
			}
			return
		case <-ctx.Done():
			slog.Default().Warn("[COT] executeDirectly: Context cancelled", "error", ctx.Err())
			return
		}
	}
}

// executeDelegation delegates to a single specialist agent
func (o *OrchestratorCOT) executeDelegation(
	ctx context.Context,
	cot *ChainOfThought,
	plan *ExecutionPlan,
	input AgentInput,
	userID string,
	userName string,
	conversationID *uuid.UUID,
	llmOptions services.LLMOptions,
	events chan<- streaming.StreamEvent,
	errs chan<- error,
) {
	if len(plan.Steps) == 0 {
		return
	}

	targetAgent := plan.Steps[0].Agent

	// Create delegation message
	delegateMsg := NewAgentMessage(AgentTypeOrchestrator, targetAgent, MsgTypeDelegate, cot.UserMessage)

	// Log the delegation step
	delegateStep := &ThoughtStep{
		Agent:     AgentTypeOrchestrator,
		Action:    "delegate",
		Input:     cot.UserMessage,
		Output:    fmt.Sprintf("Delegating to %s agent", targetAgent),
		Reasoning: plan.Reasoning,
	}
	cot.AddStep(delegateStep)
	cot.UpdateStep(delegateStep.ID, delegateStep.Output, "completed")

	// Execute step
	execStep := &ThoughtStep{
		Agent:     targetAgent,
		Action:    "execute",
		Input:     delegateMsg.Content,
		Reasoning: "Executing delegated task from orchestrator",
	}
	cot.AddStep(execStep)
	execStep.Status = "running"

	// Send agent execution indicator
	events <- streaming.StreamEvent{
		Type: streaming.EventTypeToken,
		Data: fmt.Sprintf("**Executing: @%s**\n\n", targetAgent),
	}

	// Get and run the agent with LLM options
	agent := o.registry.GetAgent(targetAgent, userID, userName, conversationID, input.Context)
	agent.SetOptions(llmOptions)

	// Add COT-specific system prompt to force extended thinking
	agent.SetFocusModePrompt(cotPrompt)
	slog.Default().Debug("[COT] Injected COT prompt into agent (forces minimum 3 thinking steps)",
		"agent", targetAgent)

	// Inject role and memory contexts if available (Feature: Memory Hierarchy + Role-based permissions)
	if input.RoleContext != "" {
		agent.SetRoleContextPrompt(input.RoleContext)
		slog.Default().Debug("[COT] Injected role context into agent",
			"agent", targetAgent,
			"chars", len(input.RoleContext))
	}
	if input.MemoryContext != "" {
		agent.SetMemoryContext(input.MemoryContext)
		slog.Default().Debug("[COT] Injected memory context into agent",
			"agent", targetAgent,
			"chars", len(input.MemoryContext))
	}

	agentEvents, agentErrs := agent.Run(ctx, input)

	var output strings.Builder
	for {
		select {
		case event, ok := <-agentEvents:
			if !ok {
				execStep.Output = output.String()
				cot.UpdateStep(execStep.ID, execStep.Output, "completed")
				cot.FinalOutput = output.String()
				events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
				return
			}
			events <- event
			if event.Type == streaming.EventTypeToken {
				if content, ok := event.Data.(string); ok {
					output.WriteString(content)
				}
			}
		case err := <-agentErrs:
			if err != nil {
				execStep.Error = err.Error()
				cot.UpdateStep(execStep.ID, "", "failed")
				errs <- err
			}
			return
		case <-ctx.Done():
			return
		}
	}
}
