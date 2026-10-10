package task

import (
	"context"

	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/prompts"
	prompts_agents "github.com/rhl/businessos-backend/internal/prompts/agents"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// TaskAgent handles task management and prioritization
type TaskAgent struct {
	*agents.BaseAgent
}

// New creates a new TaskAgent
func New(ctx *agents.AgentContext) *TaskAgent {
	systemPrompt := prompts.Compose(prompts_agents.TaskAgentPrompt)

	base := agents.NewBaseAgent(agents.BaseAgentConfig{
		Pool:           ctx.Pool,
		Config:         ctx.Config,
		UserID:         ctx.UserID,
		UserName:       ctx.UserName,
		ConversationID: ctx.ConversationID,
		AgentType:      agents.AgentTypeTask,
		AgentName:      "Task Specialist",
		Description:    "Task management, prioritization, scheduling, and dependencies",
		SystemPrompt:   systemPrompt,
		ContextReqs: agents.ContextRequirements{
			NeedsProjects:    true,
			NeedsTasks:       true,
			NeedsTeam:        true,
			MaxContextTokens: 8000,
			PrioritySections: []string{"active_tasks", "project_tasks", "team_capacity"},
		},
		EnabledTools: []string{
			"create_task", "update_task", "get_task", "list_tasks",
			"bulk_create_tasks", "move_task", "assign_task",
			"get_team_capacity", "get_project",
			"log_activity",
		},
	})

	return &TaskAgent{
		BaseAgent: base,
	}
}

// Type returns the agent type
func (a *TaskAgent) Type() agents.AgentType {
	return agents.AgentTypeTask
}

// Run executes the task agent - delegates to base implementation
func (a *TaskAgent) Run(ctx context.Context, input agents.AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	return a.BaseAgent.Run(ctx, input)
}
