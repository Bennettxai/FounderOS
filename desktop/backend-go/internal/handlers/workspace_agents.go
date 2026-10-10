package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// WorkspaceAgentsHandler powers the Agents module: the workspace's own roster of
// AI workers. Each agent is a named role with a system prompt and a chosen Claude
// model; running one sends its system prompt + a user input to Claude and records
// the result as a run. This is the new agent system that replaces Dalya.
// Workspace-scoped via X-Workspace-ID.
type WorkspaceAgentsHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewWorkspaceAgentsHandler(pool *pgxpool.Pool, cfg *config.Config) *WorkspaceAgentsHandler {
	return &WorkspaceAgentsHandler{pool: pool, cfg: cfg}
}

// allowedModels is the set of Claude models an agent may be configured with.
// Anything outside this set falls back to the balanced default.
const defaultAgentModel = "claude-sonnet-4-5-20250929"

var allowedModels = map[string]bool{
	"claude-sonnet-4-5-20250929": true, // balanced (default)
	"claude-opus-4-1-20250805":     true, // most capable
	"claude-haiku-4-5-20251001":  true, // fast/cheap
}

func normalizeModel(m string) string {
	m = strings.TrimSpace(m)
	if allowedModels[m] {
		return m
	}
	return defaultAgentModel
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *WorkspaceAgentsHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

type workspaceAgent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Description  string    `json:"description"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	Status       string    `json:"status"`
	CreatedBy    *string   `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type workspaceAgentInput struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	Description  string `json:"description"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
	Status       string `json:"status"`
}

type workspaceAgentRun struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Model     string    `json:"model"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ListAgents returns the workspace's agents. GET /api/workspace-agents
func (h *WorkspaceAgentsHandler) ListAgents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, name, role, description, model, system_prompt, status, created_by, created_at, updated_at
		FROM   workspace_agents
		WHERE  workspace_id = $1
		ORDER  BY created_at DESC
	`, wsID)
	if err != nil {
		RespondInternalErr(c, "list agents", err)
		return
	}
	defer rows.Close()

	agents := make([]workspaceAgent, 0)
	for rows.Next() {
		var a workspaceAgent
		var id uuid.UUID
		if err := rows.Scan(&id, &a.Name, &a.Role, &a.Description, &a.Model, &a.SystemPrompt, &a.Status, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan agent", err)
			return
		}
		a.ID = id.String()
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate agents", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"agents": agents, "count": len(agents), "ai_available": h.cfg.GetActiveProvider() != ""})
}

// CreateAgent adds an agent. POST /api/workspace-agents
func (h *WorkspaceAgentsHandler) CreateAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	var in workspaceAgentInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}

	var a workspaceAgent
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO workspace_agents (workspace_id, name, role, description, model, system_prompt, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, role, description, model, system_prompt, status, created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Role), in.Description, normalizeModel(in.Model), in.SystemPrompt, normalizeAgentStatus(in.Status), user.ID).
		Scan(&id, &a.Name, &a.Role, &a.Description, &a.Model, &a.SystemPrompt, &a.Status, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		RespondInternalErr(c, "create agent", err)
		return
	}
	a.ID = id.String()
	c.JSON(http.StatusCreated, a)
}

// normalizeAgentStatus keeps status within the allowed set, defaulting to active.
func normalizeAgentStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "paused":
		return "paused"
	case "archived":
		return "archived"
	default:
		return "active"
	}
}

// UpdateAgent edits an agent. PUT /api/workspace-agents/:id
func (h *WorkspaceAgentsHandler) UpdateAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid agent id")
		return
	}
	var in workspaceAgentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var a workspaceAgent
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE workspace_agents
		SET    name          = COALESCE(NULLIF($3,''), name),
		       role          = $4,
		       description   = $5,
		       model         = $6,
		       system_prompt = $7,
		       status        = $8,
		       updated_at    = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, role, description, model, system_prompt, status, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Role), in.Description, normalizeModel(in.Model), in.SystemPrompt, normalizeAgentStatus(in.Status)).
		Scan(&rid, &a.Name, &a.Role, &a.Description, &a.Model, &a.SystemPrompt, &a.Status, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "agent")
		return
	}
	a.ID = rid.String()
	c.JSON(http.StatusOK, a)
}

// DeleteAgent removes an agent (and its runs, via ON DELETE CASCADE).
// DELETE /api/workspace-agents/:id
func (h *WorkspaceAgentsHandler) DeleteAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid agent id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM workspace_agents WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete agent", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "agent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "agent deleted"})
}

type runAgentInput struct {
	Input string `json:"input"`
}

// RunAgent loads the agent, calls Claude with its system prompt + the input using
// the agent's configured model, stores a run row, and returns the output.
// POST /api/workspace-agents/:id/run
func (h *WorkspaceAgentsHandler) RunAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid agent id")
		return
	}
	var in runAgentInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Input) == "" {
		RespondBadRequestErr(c, "input is required")
		return
	}

	// Load the agent (scoped to the workspace).
	var systemPrompt, model, name string
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT name, system_prompt, model FROM workspace_agents WHERE id=$1 AND workspace_id=$2`,
		id, wsID).Scan(&name, &systemPrompt, &model)
	if err != nil {
		RespondNotFoundErr(c, "agent")
		return
	}
	model = normalizeModel(model)
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = "You are " + name + ", a helpful AI agent for a business operating system. Respond concisely and usefully."
	}

	// Call Claude with the agent's chosen model.
	llm := services.NewLLMService(h.cfg, model)
	output, llmErr := llm.ChatComplete(c.Request.Context(),
		[]services.ChatMessage{{Role: "user", Content: strings.TrimSpace(in.Input)}}, systemPrompt)

	status := "done"
	if llmErr != nil {
		status = "error"
		output = "The agent run failed: " + llmErr.Error()
	}

	// Record the run regardless of outcome.
	var runID uuid.UUID
	var createdAt time.Time
	insErr := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO workspace_agent_runs (workspace_id, agent_id, input, output, model, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, wsID, id, strings.TrimSpace(in.Input), output, model, status).Scan(&runID, &createdAt)
	if insErr != nil {
		RespondInternalErr(c, "record agent run", insErr)
		return
	}

	if llmErr != nil {
		RespondInternalErr(c, "agent run", llmErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output":     output,
		"model":      model,
		"status":     status,
		"run_id":     runID.String(),
		"created_at": createdAt,
	})
}

// ListRuns returns recent runs for an agent. GET /api/workspace-agents/:id/runs
func (h *WorkspaceAgentsHandler) ListRuns(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid agent id")
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, agent_id, input, output, model, status, created_at
		FROM   workspace_agent_runs
		WHERE  workspace_id = $1 AND agent_id = $2
		ORDER  BY created_at DESC
		LIMIT  25
	`, wsID, id)
	if err != nil {
		RespondInternalErr(c, "list runs", err)
		return
	}
	defer rows.Close()

	out := make([]workspaceAgentRun, 0)
	for rows.Next() {
		var r workspaceAgentRun
		var rid, aid uuid.UUID
		if err := rows.Scan(&rid, &aid, &r.Input, &r.Output, &r.Model, &r.Status, &r.CreatedAt); err != nil {
			RespondInternalErr(c, "scan run", err)
			return
		}
		r.ID = rid.String()
		r.AgentID = aid.String()
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate runs", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": out, "count": len(out)})
}
