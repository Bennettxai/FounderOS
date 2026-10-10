package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// AgentTool defines the interface for all agent tools
type AgentTool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// AgentToolRegistry manages available tools for agents
type AgentToolRegistry struct {
	pool             *pgxpool.Pool
	userID           string
	tools            map[string]AgentTool
	embeddingService EmbeddingServiceInterface // Kept for backward-compat callers that use SetEmbeddingService
	contextService   ContextServiceInterface   // Preferred: full context service with search delegation
}

// EmbeddingServiceInterface defines the interface for embedding operations
type EmbeddingServiceInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// ContextServiceInterface defines the interface for context operations used by tools.
// services.ContextService satisfies this interface via contextServiceAdapter.
type ContextServiceInterface interface {
	SearchTree(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error)
	GetContextTree(ctx context.Context, userID string, projectID, nodeID *uuid.UUID) (*ContextTree, error)
	LoadContextItem(ctx context.Context, userID string, itemID uuid.UUID, itemType string) (*ContextItem, error)
}

// TreeSearchParams for context search
type TreeSearchParams struct {
	Query       string   `json:"query"`
	SearchType  string   `json:"search_type"`  // 'title', 'content', 'semantic'
	EntityTypes []string `json:"entity_types"` // 'memories', 'contexts', 'artifacts', 'documents'
	MaxResults  int      `json:"max_results"`
}

// TreeSearchResult represents a search result
type TreeSearchResult struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	Summary        string    `json:"summary,omitempty"`
	RelevanceScore float64   `json:"relevance_score"`
	TreePath       []string  `json:"tree_path"`
	TokenEstimate  int       `json:"token_estimate"`
}

// ContextTree represents the hierarchical context structure
type ContextTree struct {
	RootNode   *ContextTreeNode `json:"root_node"`
	TotalItems int              `json:"total_items"`
}

// ContextTreeNode represents a node in the context tree
type ContextTreeNode struct {
	ID          uuid.UUID          `json:"id"`
	Type        string             `json:"type"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Icon        string             `json:"icon,omitempty"`
	ItemCount   int                `json:"item_count"`
	Children    []*ContextTreeNode `json:"children,omitempty"`
}

// ContextItem represents a loaded context item
type ContextItem struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
}

// contextServiceAdapter adapts *services.ContextService to ContextServiceInterface.
// It converts between the services-package types and the tools-package types so the
// tools package does not need to depend on the services types directly.
type contextServiceAdapter struct {
	svc *services.ContextService
}

func (a *contextServiceAdapter) SearchTree(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error) {
	svcParams := services.TreeSearchParams{
		Query:       params.Query,
		SearchType:  params.SearchType,
		EntityTypes: params.EntityTypes,
		MaxResults:  params.MaxResults,
	}
	svcResults, err := a.svc.SearchTree(ctx, userID, svcParams)
	if err != nil {
		return nil, err
	}
	results := make([]TreeSearchResult, len(svcResults))
	for i, r := range svcResults {
		results[i] = TreeSearchResult{
			ID:             r.ID,
			Title:          r.Title,
			Type:           r.Type,
			Summary:        r.Summary,
			RelevanceScore: r.RelevanceScore,
			TreePath:       r.TreePath,
			TokenEstimate:  r.TokenEstimate,
		}
	}
	return results, nil
}

func (a *contextServiceAdapter) GetContextTree(ctx context.Context, userID string, projectID, nodeID *uuid.UUID) (*ContextTree, error) {
	svcTree, err := a.svc.GetContextTree(ctx, userID, projectID, nodeID)
	if err != nil {
		return nil, err
	}
	return convertContextTree(svcTree), nil
}

func (a *contextServiceAdapter) LoadContextItem(ctx context.Context, userID string, itemID uuid.UUID, itemType string) (*ContextItem, error) {
	svcItem, err := a.svc.LoadContextItem(ctx, userID, itemID, itemType)
	if err != nil {
		return nil, err
	}
	return &ContextItem{
		ID:         svcItem.ID,
		Type:       svcItem.Type,
		Title:      svcItem.Title,
		Content:    svcItem.Content,
		TokenCount: svcItem.TokenCount,
	}, nil
}

func convertContextTree(t *services.ContextTree) *ContextTree {
	if t == nil {
		return nil
	}
	return &ContextTree{
		RootNode:   convertContextTreeNode(t.RootNode),
		TotalItems: t.TotalItems,
	}
}

func convertContextTreeNode(n *services.ContextTreeNode) *ContextTreeNode {
	if n == nil {
		return nil
	}
	node := &ContextTreeNode{
		ID:          n.ID,
		Type:        n.Type,
		Name:        n.Name,
		Description: n.Description,
		Icon:        n.Icon,
		ItemCount:   n.ItemCount,
	}
	if len(n.Children) > 0 {
		node.Children = make([]*ContextTreeNode, len(n.Children))
		for i, child := range n.Children {
			node.Children[i] = convertContextTreeNode(child)
		}
	}
	return node
}

// NewAgentToolRegistry creates a new tool registry for an agent
func NewAgentToolRegistry(pool *pgxpool.Pool, userID string) *AgentToolRegistry {
	registry := &AgentToolRegistry{
		pool:   pool,
		userID: userID,
		tools:  make(map[string]AgentTool),
	}

	// Register all available tools
	registry.registerTools()

	return registry
}

// NewAgentToolRegistryWithEmbedding creates a registry with embedding service for semantic search.
// Deprecated: prefer NewAgentToolRegistryWithContext which delegates to ContextService.
func NewAgentToolRegistryWithEmbedding(pool *pgxpool.Pool, userID string, embeddingService EmbeddingServiceInterface) *AgentToolRegistry {
	registry := &AgentToolRegistry{
		pool:             pool,
		userID:           userID,
		tools:            make(map[string]AgentTool),
		embeddingService: embeddingService,
	}

	// Register all available tools including context tools
	registry.registerTools()

	return registry
}

// NewAgentToolRegistryWithContext creates a registry that delegates context operations to
// ContextService. This is the preferred constructor when a ContextService is available.
func NewAgentToolRegistryWithContext(pool *pgxpool.Pool, userID string, svc *services.ContextService) *AgentToolRegistry {
	registry := &AgentToolRegistry{
		pool:           pool,
		userID:         userID,
		tools:          make(map[string]AgentTool),
		contextService: &contextServiceAdapter{svc: svc},
	}
	registry.registerTools()
	registry.registerContextTools()
	return registry
}

// SetEmbeddingService sets the embedding service to enable semantic search tools.
// Deprecated: prefer setting ContextService via NewAgentToolRegistryWithContext.
func (r *AgentToolRegistry) SetEmbeddingService(embeddingService EmbeddingServiceInterface) {
	r.embeddingService = embeddingService
	// Re-register tools to include context tools
	r.registerContextTools()
}

func (r *AgentToolRegistry) registerTools() {
	// Read tools
	r.tools["get_project"] = &GetProjectTool{pool: r.pool, userID: r.userID}
	r.tools["get_task"] = &GetTaskTool{pool: r.pool, userID: r.userID}
	r.tools["get_client"] = &GetClientTool{pool: r.pool, userID: r.userID}
	r.tools["list_tasks"] = &ListTasksTool{pool: r.pool, userID: r.userID}
	r.tools["list_projects"] = &ListProjectsTool{pool: r.pool, userID: r.userID}
	r.tools["search_documents"] = &SearchDocumentsTool{pool: r.pool, userID: r.userID}
	r.tools["get_team_capacity"] = &GetTeamCapacityTool{pool: r.pool, userID: r.userID}
	r.tools["query_metrics"] = &QueryMetricsTool{pool: r.pool, userID: r.userID}

	// Write tools
	r.tools["create_task"] = &CreateTaskTool{pool: r.pool, userID: r.userID}
	r.tools["update_task"] = &UpdateTaskTool{pool: r.pool, userID: r.userID}
	r.tools["create_note"] = &CreateNoteTool{pool: r.pool, userID: r.userID}
	r.tools["update_client_pipeline"] = &UpdateClientPipelineTool{pool: r.pool, userID: r.userID}
	r.tools["log_client_interaction"] = &LogClientInteractionTool{pool: r.pool, userID: r.userID}
	r.tools["create_project"] = &CreateProjectTool{pool: r.pool, userID: r.userID}
	r.tools["update_project"] = &UpdateProjectTool{pool: r.pool, userID: r.userID}
	r.tools["bulk_create_tasks"] = &BulkCreateTasksTool{pool: r.pool, userID: r.userID}
	r.tools["move_task"] = &MoveTaskTool{pool: r.pool, userID: r.userID}
	r.tools["assign_task"] = &AssignTaskTool{pool: r.pool, userID: r.userID}
	r.tools["create_client"] = &CreateClientTool{pool: r.pool, userID: r.userID}
	r.tools["update_client"] = &UpdateClientTool{pool: r.pool, userID: r.userID}
	r.tools["log_activity"] = &LogActivityTool{pool: r.pool, userID: r.userID}
	r.tools["create_artifact"] = &CreateArtifactTool{pool: r.pool, userID: r.userID}

	// Dashboard tools
	r.tools["configure_dashboard"] = &ConfigureDashboardTool{pool: r.pool, userID: r.userID}

	// Search tools — delegate to WebSearchService for multi-provider support.
	r.tools["web_search"] = &WebSearchTool{
		pool:      r.pool,
		userID:    r.userID,
		searchSvc: services.NewWebSearchService(),
	}
}

// registerContextTools registers the context navigation tools.
// Prefers contextService (full delegation); falls back to embeddingService for legacy callers.
func (r *AgentToolRegistry) registerContextTools() {
	if r.contextService != nil {
		r.tools["tree_search"] = &TreeSearchTool{userID: r.userID, contextService: r.contextService}
		r.tools["browse_tree"] = &BrowseTreeTool{pool: r.pool, userID: r.userID}
		r.tools["load_context"] = &LoadContextTool{pool: r.pool, userID: r.userID}
		return
	}
	if r.embeddingService != nil {
		// Legacy: build a context service adapter from embeddingService alone is not possible;
		// fall back to pool-based tree search via BrowseTreeTool and LoadContextTool only.
		r.tools["browse_tree"] = &BrowseTreeTool{pool: r.pool, userID: r.userID}
		r.tools["load_context"] = &LoadContextTool{pool: r.pool, userID: r.userID}
	}
}

// RegisterTool adds an external tool to the registry. This is used for tools
// that require dependencies not available at registry creation time (e.g., the
// OSA client for escalate_to_osa).
func (r *AgentToolRegistry) RegisterTool(tool AgentTool) {
	r.tools[tool.Name()] = tool
}

// GetTool returns a tool by name
func (r *AgentToolRegistry) GetTool(name string) (AgentTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetAllTools returns all registered tools
func (r *AgentToolRegistry) GetAllTools() []AgentTool {
	tools := make([]AgentTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolDefinitions returns tool definitions for LLM
func (r *AgentToolRegistry) GetToolDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, map[string]interface{}{
			"name":         tool.Name(),
			"description":  tool.Description(),
			"input_schema": tool.InputSchema(),
		})
	}
	return defs
}

// ExecuteTool executes a tool by name
func (r *AgentToolRegistry) ExecuteTool(ctx context.Context, name string, input json.RawMessage) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(ctx, input)
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
