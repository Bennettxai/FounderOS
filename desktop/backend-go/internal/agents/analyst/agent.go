package analyst

import (
	"context"

	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/prompts"
	prompts_agents "github.com/rhl/businessos-backend/internal/prompts/agents"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// AnalystAgent handles data analysis and insights
type AnalystAgent struct {
	*agents.BaseAgent
}

// New creates a new AnalystAgent
func New(ctx *agents.AgentContext) *AnalystAgent {
	systemPrompt := prompts.DefaultComposer.ComposeForAnalysis(prompts_agents.AnalystAgentPrompt)

	base := agents.NewBaseAgent(agents.BaseAgentConfig{
		Pool:           ctx.Pool,
		Config:         ctx.Config,
		UserID:         ctx.UserID,
		UserName:       ctx.UserName,
		ConversationID: ctx.ConversationID,
		AgentType:      agents.AgentTypeAnalyst,
		AgentName:      "Business Analyst",
		Description:    "Analyzes data, metrics, trends, and provides business insights",
		SystemPrompt:   systemPrompt,
		ContextReqs: agents.ContextRequirements{
			NeedsProjects:    true,
			NeedsMetrics:     true,
			MaxContextTokens: 8000,
			PrioritySections: []string{"metrics_data", "historical_trends", "kpis"},
		},
		EnabledTools: []string{
			"query_metrics", "get_team_capacity",
			"list_projects", "list_tasks", "get_project",
			"search_documents",
			"create_artifact", "log_activity",
		},
	})

	return &AnalystAgent{
		BaseAgent: base,
	}
}

// Type returns the agent type
func (a *AnalystAgent) Type() agents.AgentType {
	return agents.AgentTypeAnalyst
}

// Run executes the analyst agent - delegates to base implementation
func (a *AnalystAgent) Run(ctx context.Context, input agents.AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	return a.BaseAgent.Run(ctx, input)
}
