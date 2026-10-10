package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/streaming"
)

// synthesizeResults combines outputs from multiple agents
func (o *OrchestratorCOT) synthesizeResults(
	_ context.Context,
	cot *ChainOfThought,
	_ chan<- streaming.StreamEvent,
) {
	synthStep := &ThoughtStep{
		Agent:     AgentTypeOrchestrator,
		Action:    "synthesize",
		Input:     "Multiple agent outputs",
		Reasoning: "Combining outputs from multiple specialists",
	}
	cot.AddStep(synthStep)
	synthStep.Status = "running"

	// Collect all outputs
	var outputs []string
	for _, step := range cot.Steps {
		if step.Action == "execute" && step.Status == "completed" && step.Output != "" {
			outputs = append(outputs, fmt.Sprintf("**%s**: %s", step.Agent, truncateOutput(step.Output, 300)))
		}
	}

	synthStep.Output = fmt.Sprintf("Synthesized %d agent outputs", len(outputs))
	cot.UpdateStep(synthStep.ID, synthStep.Output, "completed")
}

// formatRoutingBox creates a formatted routing display
func (o *OrchestratorCOT) formatRoutingBox(intent Intent, plan *ExecutionPlan) string {
	var sb strings.Builder

	sb.WriteString("**Routing Decision**\n")
	sb.WriteString(fmt.Sprintf("- Strategy: `%s`\n", plan.Strategy))
	sb.WriteString(fmt.Sprintf("- Primary Agent: `@%s`\n", intent.TargetAgent))
	sb.WriteString(fmt.Sprintf("- Confidence: %.0f%%\n", intent.Confidence*100))

	if len(plan.Steps) > 1 {
		sb.WriteString(fmt.Sprintf("- Steps: %d agents\n", len(plan.Steps)))
		sb.WriteString("\n**Execution Plan:**\n")
		for _, step := range plan.Steps {
			indicator := "→"
			if step.CanParallel {
				indicator = "⇉"
			}
			searchFlag := ""
			if step.NeedsSearch {
				searchFlag = " [web search]"
			}
			sb.WriteString(fmt.Sprintf("  %s Step %d: `@%s`%s\n", indicator, step.Order, step.Agent, searchFlag))
		}
	} else if len(plan.Steps) == 1 && plan.Steps[0].NeedsSearch {
		sb.WriteString("- Web Search: enabled\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// formatCOTSummary creates a readable summary of the chain of thought
func (o *OrchestratorCOT) formatCOTSummary(cot *ChainOfThought) string {
	var sb strings.Builder
	sb.WriteString("\n\n---\n")
	sb.WriteString("### Chain of Thought Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", cot.TotalDuration.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("**Agents:** %v\n", cot.AgentsInvolved))
	sb.WriteString(fmt.Sprintf("**Steps:** %d\n", len(cot.Steps)))
	return sb.String()
}
