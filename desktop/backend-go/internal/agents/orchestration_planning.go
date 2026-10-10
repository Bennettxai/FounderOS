package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// createExecutionPlan determines how to handle the request
func (o *OrchestratorCOT) createExecutionPlan(
	_ context.Context,
	userMessage string,
	intent Intent,
	input AgentInput,
) *ExecutionPlan {
	plan := &ExecutionPlan{
		PrimaryAgent: intent.TargetAgent,
		Confidence:   intent.Confidence,
		Steps:        make([]PlannedStep, 0),
	}

	msgLower := strings.ToLower(userMessage)

	// ========== DETECT REQUIREMENTS ==========

	// Web search indicators
	needsWebSearch := strings.Contains(msgLower, "search") ||
		strings.Contains(msgLower, "find") ||
		strings.Contains(msgLower, "look up") ||
		strings.Contains(msgLower, "what is") ||
		strings.Contains(msgLower, "how does") ||
		strings.Contains(msgLower, "latest") ||
		strings.Contains(msgLower, "current") ||
		strings.Contains(msgLower, "news") ||
		strings.Contains(msgLower, "2024") ||
		strings.Contains(msgLower, "2025") ||
		input.FocusMode == "deep" ||
		input.FocusMode == "research"

	// Multi-step indicators
	hasSequentialSteps := strings.Contains(msgLower, " then ") ||
		strings.Contains(msgLower, " after ") ||
		strings.Contains(msgLower, " next ") ||
		strings.Contains(msgLower, "first ") ||
		strings.Contains(msgLower, "step by step")

	hasParallelTasks := strings.Contains(msgLower, " and ") ||
		strings.Contains(msgLower, " also ") ||
		strings.Contains(msgLower, " plus ") ||
		strings.Contains(msgLower, "both ")

	// Domain detection with weights
	domains := make(map[AgentType]float64)

	// Analysis domain
	if strings.Contains(msgLower, "analyze") || strings.Contains(msgLower, "analysis") ||
		strings.Contains(msgLower, "metrics") || strings.Contains(msgLower, "data") ||
		strings.Contains(msgLower, "statistics") || strings.Contains(msgLower, "compare") {
		domains[AgentTypeAnalyst] = 0.9
	}

	// Document domain
	if strings.Contains(msgLower, "document") || strings.Contains(msgLower, "report") ||
		strings.Contains(msgLower, "proposal") || strings.Contains(msgLower, "write") ||
		strings.Contains(msgLower, "draft") || strings.Contains(msgLower, "create a") {
		domains[AgentTypeDocument] = 0.9
	}

	// Project domain
	if strings.Contains(msgLower, "project") || strings.Contains(msgLower, "plan") ||
		strings.Contains(msgLower, "roadmap") || strings.Contains(msgLower, "timeline") ||
		strings.Contains(msgLower, "milestone") {
		domains[AgentTypeProject] = 0.9
	}

	// Task domain
	if strings.Contains(msgLower, "task") || strings.Contains(msgLower, "todo") ||
		strings.Contains(msgLower, "prioritize") || strings.Contains(msgLower, "schedule") ||
		strings.Contains(msgLower, "deadline") {
		domains[AgentTypeTask] = 0.9
	}

	// Client domain
	if strings.Contains(msgLower, "client") || strings.Contains(msgLower, "customer") ||
		strings.Contains(msgLower, "pipeline") || strings.Contains(msgLower, "lead") ||
		strings.Contains(msgLower, "crm") {
		domains[AgentTypeClient] = 0.9
	}

	// Research/general queries go to analyst
	if strings.Contains(msgLower, "how") || strings.Contains(msgLower, "what") ||
		strings.Contains(msgLower, "why") || strings.Contains(msgLower, "explain") {
		if _, exists := domains[AgentTypeAnalyst]; !exists {
			domains[AgentTypeAnalyst] = 0.7
		}
	}

	domainCount := len(domains)

	// ========== DETERMINE STRATEGY ==========

	if !intent.ShouldDelegate || intent.Confidence < 0.4 {
		// Low confidence - orchestrator handles directly
		plan.Strategy = "direct"
		plan.Reasoning = "Low confidence or general chat - orchestrator handling directly"
		plan.Steps = append(plan.Steps, PlannedStep{
			Order:       1,
			Agent:       AgentTypeOrchestrator,
			Task:        userMessage,
			NeedsSearch: needsWebSearch,
		})
	} else if domainCount > 1 && hasSequentialSteps {
		// Multiple domains with sequence - sequential execution
		plan.Strategy = "sequential"
		plan.Reasoning = fmt.Sprintf("Sequential workflow detected with %d domains", domainCount)
		o.buildSequentialSteps(plan, domains, userMessage, needsWebSearch)
	} else if domainCount > 1 && hasParallelTasks {
		// Multiple domains in parallel
		plan.Strategy = "multi-agent"
		plan.Reasoning = fmt.Sprintf("Parallel tasks detected across %d domains", domainCount)
		o.buildParallelSteps(plan, domains, userMessage, needsWebSearch)
	} else if domainCount == 1 {
		// Single domain - delegate
		plan.Strategy = "delegate"
		for agent := range domains {
			plan.Reasoning = fmt.Sprintf("Single domain detected - delegating to @%s", agent)
			plan.Steps = append(plan.Steps, PlannedStep{
				Order:       1,
				Agent:       agent,
				Task:        userMessage,
				NeedsSearch: needsWebSearch,
			})
			break
		}
	} else {
		// No specific domain detected - use intent target
		plan.Strategy = "delegate"
		plan.Reasoning = fmt.Sprintf("Delegating to @%s based on intent analysis", intent.TargetAgent)
		plan.Steps = append(plan.Steps, PlannedStep{
			Order:       1,
			Agent:       intent.TargetAgent,
			Task:        userMessage,
			NeedsSearch: needsWebSearch,
		})
	}

	return plan
}

// buildSequentialSteps creates ordered steps for sequential execution
func (o *OrchestratorCOT) buildSequentialSteps(plan *ExecutionPlan, domains map[AgentType]float64, task string, needsSearch bool) {
	// Define execution order priority
	orderPriority := []AgentType{
		AgentTypeAnalyst,  // Research/analysis first
		AgentTypeProject,  // Planning second
		AgentTypeTask,     // Task breakdown third
		AgentTypeClient,   // Client handling fourth
		AgentTypeDocument, // Document creation last
	}

	order := 1
	var prevOrders []int
	for _, agent := range orderPriority {
		if _, exists := domains[agent]; exists {
			step := PlannedStep{
				Order:       order,
				Agent:       agent,
				Task:        fmt.Sprintf("Handle %s aspects of: %s", agent, task),
				CanParallel: false,
				NeedsSearch: needsSearch && order == 1, // Only first step needs search
			}
			if len(prevOrders) > 0 {
				step.DependsOn = prevOrders
			}
			plan.Steps = append(plan.Steps, step)
			prevOrders = []int{order}
			order++
		}
	}

	// Add synthesis step if multiple agents
	if len(plan.Steps) > 1 {
		allOrders := make([]int, len(plan.Steps))
		for i := range plan.Steps {
			allOrders[i] = i + 1
		}
		plan.Steps = append(plan.Steps, PlannedStep{
			Order:       order,
			Agent:       AgentTypeOrchestrator,
			Task:        "Synthesize results from all agents",
			DependsOn:   allOrders,
			CanParallel: false,
		})
	}
}

// buildParallelSteps creates parallel steps for concurrent execution
func (o *OrchestratorCOT) buildParallelSteps(plan *ExecutionPlan, domains map[AgentType]float64, task string, needsSearch bool) {
	order := 1
	parallelOrders := []int{}

	for agent := range domains {
		step := PlannedStep{
			Order:       order,
			Agent:       agent,
			Task:        fmt.Sprintf("Handle %s aspects of: %s", agent, task),
			CanParallel: true,
			NeedsSearch: needsSearch,
		}
		plan.Steps = append(plan.Steps, step)
		parallelOrders = append(parallelOrders, order)
		order++
	}

	// Add synthesis step to combine all parallel results
	if len(plan.Steps) > 1 {
		plan.Steps = append(plan.Steps, PlannedStep{
			Order:       order,
			Agent:       AgentTypeOrchestrator,
			Task:        "Synthesize and combine results from all agents",
			DependsOn:   parallelOrders,
			CanParallel: false,
		})
	}
}

// formatStepContext converts step context map to a formatted string
func formatStepContext(ctx map[string]any) string {
	if len(ctx) == 0 {
		return ""
	}

	var sb strings.Builder
	for key, value := range ctx {
		switch v := value.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("- **%s:** %s\n", key, v))
		case []string:
			sb.WriteString(fmt.Sprintf("- **%s:** %s\n", key, strings.Join(v, ", ")))
		default:
			jsonBytes, _ := json.Marshal(v)
			sb.WriteString(fmt.Sprintf("- **%s:** %s\n", key, string(jsonBytes)))
		}
	}
	return sb.String()
}
