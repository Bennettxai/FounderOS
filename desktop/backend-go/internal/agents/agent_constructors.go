package agents

import (
	"github.com/rhl/businessos-backend/internal/prompts"
	prompts_agents "github.com/rhl/businessos-backend/internal/prompts/agents"
)

// NewOrchestrator creates a new orchestrator agent
func NewOrchestrator(ctx *AgentContext) Agent {
	systemPrompt := prompts.ComposeWithUserContext(
		prompts_agents.OrchestratorAgentPrompt+prompts.ArtifactInstruction,
		ctx.UserName, "", "",
	)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeOrchestrator,
		AgentName:          "OSA Orchestrator",
		Description:        "Primary interface that handles general requests and routes to specialists",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsProjects:    true,
			NeedsTasks:       true,
			NeedsClients:     true,
			NeedsKnowledge:   true,
			MaxContextTokens: 10000,
		},
		EnabledTools: []string{
			// Read tools
			"search_documents", "get_project", "get_task", "get_client",
			"list_projects", "list_tasks", "get_team_capacity", "query_metrics",
			// Write tools
			"create_task", "update_task", "create_project", "update_project",
			"create_client", "update_client", "create_artifact", "log_activity",
			"bulk_create_tasks", "assign_task", "move_task",
			// Client tools
			"log_client_interaction", "update_client_pipeline",
			// Context tools (knowledge base)
			"tree_search", "browse_tree", "load_context",
		},
	})
}

// NewDocumentAgent creates a new document agent
func NewDocumentAgent(ctx *AgentContext) Agent {
	systemPrompt := prompts.DefaultComposer.ComposeForDocument(prompts_agents.DocumentAgentPrompt)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeDocument,
		AgentName:          "Document Specialist",
		Description:        "Creates formal business documents: proposals, SOPs, reports, frameworks",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsProjects:    true,
			NeedsKnowledge:   true,
			NeedsClients:     true,
			MaxContextTokens: 10000,
		},
		EnabledTools: []string{
			"create_artifact", "search_documents", "get_project", "get_client",
			"log_activity",
			"tree_search", "browse_tree", "load_context", // Context tools for knowledge base
		},
	})
}

// NewProjectAgent creates a new project/planning agent
func NewProjectAgent(ctx *AgentContext) Agent {
	systemPrompt := prompts.Compose(prompts_agents.ProjectAgentPrompt)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeProject,
		AgentName:          "Project Specialist",
		Description:        "Project management and planning specialist",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsProjects:    true,
			NeedsTasks:       true,
			NeedsTeam:        true,
			NeedsClients:     true,
			MaxContextTokens: 8000,
		},
		EnabledTools: []string{
			"create_project", "update_project", "get_project", "list_projects",
			"create_task", "bulk_create_tasks", "assign_task",
			"get_team_capacity", "search_documents",
			"create_artifact", "log_activity",
			"tree_search", "browse_tree", "load_context", // Context tools for knowledge base
		},
	})
}

// NewClientAgent creates a new client agent
func NewClientAgent(ctx *AgentContext) Agent {
	systemPrompt := prompts.Compose(prompts_agents.ClientAgentPrompt)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeClient,
		AgentName:          "Client Specialist",
		Description:        "Client relationship and pipeline specialist",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsClients:     true,
			NeedsProjects:    true,
			NeedsKnowledge:   true,
			MaxContextTokens: 6000,
		},
		EnabledTools: []string{
			"create_client", "update_client", "get_client",
			"log_client_interaction", "update_client_pipeline",
			"search_documents", "get_project",
			"create_artifact", "log_activity",
			"tree_search", "browse_tree", "load_context", // Context tools for knowledge base
		},
	})
}

// NewAnalystAgent creates a new analyst agent
func NewAnalystAgent(ctx *AgentContext) Agent {
	systemPrompt := prompts.DefaultComposer.ComposeForAnalysis(prompts_agents.AnalystAgentPrompt)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeAnalyst,
		AgentName:          "Analyst Specialist",
		Description:        "Data analysis and insights specialist",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsProjects:    true,
			NeedsTasks:       true,
			NeedsClients:     true,
			NeedsTeam:        true,
			MaxContextTokens: 8000,
		},
		EnabledTools: []string{
			"query_metrics", "get_team_capacity",
			"list_projects", "list_tasks", "get_project",
			"search_documents", "create_artifact",
			"log_activity",
			"tree_search", "browse_tree", "load_context", // Context tools for knowledge base
		},
	})
}

// NewTaskAgent creates a new task management agent
func NewTaskAgent(ctx *AgentContext) Agent {
	systemPrompt := prompts.Compose(prompts_agents.TaskAgentPrompt)
	return NewBaseAgent(BaseAgentConfig{
		Pool:               ctx.Pool,
		Config:             ctx.Config,
		UserID:             ctx.UserID,
		UserName:           ctx.UserName,
		ConversationID:     ctx.ConversationID,
		EmbeddingService:   ctx.EmbeddingService,
		PromptPersonalizer: ctx.PromptPersonalizer,
		SignalHints:        ctx.SignalHints,
		AgentType:          AgentTypeTask,
		AgentName:          "Task Specialist",
		Description:        "Task management, prioritization, scheduling, and dependencies",
		SystemPrompt:       systemPrompt,
		ContextReqs: ContextRequirements{
			NeedsProjects:    true,
			NeedsTasks:       true,
			NeedsTeam:        true,
			MaxContextTokens: 8000,
		},
		EnabledTools: []string{
			"create_task", "update_task", "get_task", "list_tasks",
			"bulk_create_tasks", "move_task", "assign_task",
			"get_team_capacity", "get_project",
			"log_activity",
			"tree_search", "browse_tree", "load_context", // Context tools for knowledge base
		},
	})
}
