package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// ChainHistory tracks the full history of a multi-hop delegation
type ChainHistory struct {
	Steps []ChainStep `json:"steps"`
}

// ChainStep represents a single step in the chain history
type ChainStep struct {
	Order     int       `json:"order"`
	Agent     AgentType `json:"agent"`
	Task      string    `json:"task"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Reasoning string    `json:"reasoning,omitempty"`
	Model     string    `json:"model,omitempty"`
}

// FormatAsContext formats the chain history as context for the next agent
func (ch *ChainHistory) FormatAsContext() string {
	if len(ch.Steps) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Previous Agent Chain History\n\n")
	sb.WriteString("The following is the history of previous agents in this chain. Use this context to inform your response.\n\n")

	for _, step := range ch.Steps {
		sb.WriteString(fmt.Sprintf("### Step %d: @%s Agent\n", step.Order, step.Agent))
		if step.Task != "" {
			sb.WriteString(fmt.Sprintf("**Task:** %s\n", step.Task))
		}
		sb.WriteString(fmt.Sprintf("**Output:**\n%s\n\n", truncateOutput(step.Output, 2000)))
		sb.WriteString("---\n\n")
	}

	sb.WriteString("## Your Task\n")
	sb.WriteString("Based on the above context, provide your specialized contribution to this request.\n\n")

	return sb.String()
}

// executeMultiAgent runs multiple agents in parallel
func (o *OrchestratorCOT) executeMultiAgent(
	ctx context.Context,
	cot *ChainOfThought,
	plan *ExecutionPlan,
	input AgentInput,
	userID string,
	userName string,
	conversationID *uuid.UUID,
	llmOptions services.LLMOptions,
	events chan<- streaming.StreamEvent,
	_ chan<- error, // errs - reserved for future error handling
) {
	events <- streaming.StreamEvent{
		Type: streaming.EventTypeToken,
		Data: fmt.Sprintf("Multi-Agent Execution (%d agents)\n\n", len(plan.Steps)),
	}

	var wg sync.WaitGroup
	results := make(chan AgentResult, len(plan.Steps))

	// Launch parallel agents
	for _, step := range plan.Steps {
		if !step.CanParallel {
			continue // Handle non-parallel steps later
		}

		wg.Add(1)
		go func(s PlannedStep) {
			defer wg.Done()

			// Create execution step in COT
			execStep := &ThoughtStep{
				Agent:     s.Agent,
				Action:    "execute",
				Input:     s.Task,
				Reasoning: "Parallel execution as part of multi-agent plan",
			}
			cot.AddStep(execStep)
			execStep.Status = "running"

			// Run agent with LLM options
			agent := o.registry.GetAgent(s.Agent, userID, userName, conversationID, input.Context)
			agent.SetOptions(llmOptions)

			// Inject role and memory contexts if available (Feature: Memory Hierarchy + Role-based permissions)
			if input.RoleContext != "" {
				agent.SetRoleContextPrompt(input.RoleContext)
				slog.Default().Debug("[COT-Multi] Injected role context into agent",
					"agent", s.Agent,
					"chars", len(input.RoleContext))
			}
			if input.MemoryContext != "" {
				agent.SetMemoryContext(input.MemoryContext)
				slog.Default().Debug("[COT-Multi] Injected memory context into agent",
					"agent", s.Agent,
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
						results <- AgentResult{AgentType: s.Agent, Output: output.String()}
						return
					}
					if event.Type == streaming.EventTypeToken {
						if content, ok := event.Data.(string); ok {
							output.WriteString(content)
						}
					}
				case err := <-agentErrs:
					if err != nil {
						execStep.Error = err.Error()
						cot.UpdateStep(execStep.ID, "", "failed")
						results <- AgentResult{AgentType: s.Agent, Error: err}
					}
					return
				case <-ctx.Done():
					return
				}
			}
		}(step)
	}

	// Wait for all parallel agents
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allOutputs strings.Builder
	for result := range results {
		if result.Error != nil {
			continue
		}
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeToken,
			Data: fmt.Sprintf("\n---\n### %s Agent:\n%s\n", result.AgentType, truncateOutput(result.Output, 500)),
		}
		allOutputs.WriteString(result.Output)
		allOutputs.WriteString("\n\n")
	}

	cot.FinalOutput = allOutputs.String()
}

// executeSequential runs agents in sequence with full chain history
func (o *OrchestratorCOT) executeSequential(
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
	events <- streaming.StreamEvent{
		Type: streaming.EventTypeToken,
		Data: fmt.Sprintf("**Multi-Hop Sequential Execution** (%d steps)\n\n", len(plan.Steps)),
	}

	// Initialize chain history for multi-hop tracking
	chainHistory := &ChainHistory{
		Steps: make([]ChainStep, 0, len(plan.Steps)),
	}

	var previousOutput string
	var accumulatedContext strings.Builder // Accumulate context across all steps

	for i, step := range plan.Steps {
		// Create execution step in COT
		execStep := &ThoughtStep{
			Agent:     step.Agent,
			Action:    "execute",
			Input:     step.Task,
			Reasoning: fmt.Sprintf("Multi-hop step %d/%d - building on previous agent outputs", i+1, len(plan.Steps)),
		}
		cot.AddStep(execStep)
		execStep.Status = "running"

		// Send step indicator with agent info
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeToken,
			Data: fmt.Sprintf("\n**Step %d/%d: @%s Agent**\n", i+1, len(plan.Steps), step.Agent),
		}

		// Build modified input with chain history
		modifiedInput := input

		// Add chain history as context message if we have previous steps
		if len(chainHistory.Steps) > 0 {
			chainContext := chainHistory.FormatAsContext()
			contextMsg := services.ChatMessage{
				Role:    "system",
				Content: chainContext,
			}
			modifiedInput.Messages = append([]services.ChatMessage{contextMsg}, modifiedInput.Messages...)
		}

		// Add step-specific context if provided
		if step.Context != nil {
			stepContext := formatStepContext(step.Context)
			if stepContext != "" {
				stepContextMsg := services.ChatMessage{
					Role:    "system",
					Content: fmt.Sprintf("## Step-Specific Context\n%s", stepContext),
				}
				modifiedInput.Messages = append([]services.ChatMessage{stepContextMsg}, modifiedInput.Messages...)
			}
		}

		// Prepare LLM options for this step
		stepLLMOptions := llmOptions

		// Apply model override if specified for this step
		if step.ModelOverride != "" {
			stepLLMOptions.Model = &step.ModelOverride
		}

		// Get and run the agent
		agent := o.registry.GetAgent(step.Agent, userID, userName, conversationID, input.Context)
		agent.SetOptions(stepLLMOptions)

		// Inject role and memory contexts if available (Feature: Memory Hierarchy + Role-based permissions)
		if input.RoleContext != "" {
			agent.SetRoleContextPrompt(input.RoleContext)
			slog.Default().Debug("[COT-Seq] Injected role context into agent",
				"agent", step.Agent,
				"chars", len(input.RoleContext))
		}
		if input.MemoryContext != "" {
			agent.SetMemoryContext(input.MemoryContext)
			slog.Default().Debug("[COT-Seq] Injected memory context into agent",
				"agent", step.Agent,
				"chars", len(input.MemoryContext))
		}

		// If step needs search, enable it in the input
		if step.NeedsSearch {
			modifiedInput.FocusMode = "research"
		}

		agentEvents, agentErrs := agent.Run(ctx, modifiedInput)

		var output strings.Builder
		done := false
		for !done {
			select {
			case event, ok := <-agentEvents:
				if !ok {
					done = true
					break
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

					// Send failure notification
					events <- streaming.StreamEvent{
						Type: streaming.EventTypeToken,
						Data: fmt.Sprintf("\n[Step %d failed: %s - continuing with next step]\n", i+1, err.Error()),
					}

					// Don't return on failure if there are more steps - let the chain continue
					if step.Order < len(plan.Steps) {
						continue
					}
					return
				}
				done = true
			case <-ctx.Done():
				return
			}
		}

		stepOutput := output.String()
		previousOutput = stepOutput
		execStep.Output = previousOutput
		cot.UpdateStep(execStep.ID, execStep.Output, "completed")

		// Add to chain history for next agent
		chainHistory.Steps = append(chainHistory.Steps, ChainStep{
			Order:     i + 1,
			Agent:     step.Agent,
			Task:      step.Task,
			Input:     cot.UserMessage,
			Output:    stepOutput,
			Reasoning: execStep.Reasoning,
		})

		// Accumulate context
		accumulatedContext.WriteString(fmt.Sprintf("\n### @%s Output:\n%s\n", step.Agent, truncateOutput(stepOutput, 1500)))

		// Send step completion indicator
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeToken,
			Data: fmt.Sprintf("\n[Step %d complete - @%s finished]\n", i+1, step.Agent),
		}
	}

	// Final output is the combined accumulated context
	cot.FinalOutput = previousOutput

	// Send chain completion summary
	events <- streaming.StreamEvent{
		Type: streaming.EventTypeToken,
		Data: fmt.Sprintf("\n**Chain Complete** - %d agents collaborated\n", len(chainHistory.Steps)),
	}
}
