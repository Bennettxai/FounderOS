package document

import (
	"context"

	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/prompts"
	prompts_agents "github.com/rhl/businessos-backend/internal/prompts/agents"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// DocumentAgent creates formal business documents
type DocumentAgent struct {
	*agents.BaseAgent
}

// New creates a new DocumentAgent
func New(ctx *agents.AgentContext) *DocumentAgent {
	systemPrompt := prompts.DefaultComposer.ComposeForDocument(prompts_agents.DocumentAgentPrompt)

	base := agents.NewBaseAgent(agents.BaseAgentConfig{
		Pool:           ctx.Pool,
		Config:         ctx.Config,
		UserID:         ctx.UserID,
		UserName:       ctx.UserName,
		ConversationID: ctx.ConversationID,
		AgentType:      agents.AgentTypeDocument,
		AgentName:      "Document Specialist",
		Description:    "Creates formal business documents: proposals, SOPs, reports, frameworks",
		SystemPrompt:   systemPrompt,
		ContextReqs: agents.ContextRequirements{
			NeedsProjects:    true,
			NeedsKnowledge:   true,
			NeedsClients:     true,
			MaxContextTokens: 10000,
			PrioritySections: []string{"project_details", "selected_documents", "client_info"},
		},
		EnabledTools: []string{
			"create_artifact", "search_documents",
			"get_project", "get_client",
			"log_activity",
		},
	})

	return &DocumentAgent{
		BaseAgent: base,
	}
}

// Type returns the agent type
func (a *DocumentAgent) Type() agents.AgentType {
	return agents.AgentTypeDocument
}

// Run executes the document agent - delegates to base implementation
func (a *DocumentAgent) Run(ctx context.Context, input agents.AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	return a.BaseAgent.Run(ctx, input)
}
