package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

// EnginesHandler powers the Engines module: internal automation / multi-agent
// workflows. A workflow is a named, ordered list of steps; running it executes
// each step in sequence (AI steps call Claude, note steps just record) and
// persists the combined output as a run. Workspace-scoped via X-Workspace-ID.
// Holds both pool and cfg so run steps can call the LLM (like IntelligenceHandler).
type EnginesHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewEnginesHandler(pool *pgxpool.Pool, cfg *config.Config) *EnginesHandler {
	return &EnginesHandler{pool: pool, cfg: cfg}
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *EnginesHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// engineStep is one step in a workflow. AI steps carry a prompt; note/http steps
// carry a config string. Type is one of ai|note|http.
type engineStep struct {
	Type   string `json:"type"` // "ai" | "note" | "http"
	Label  string `json:"label"`
	Prompt string `json:"prompt"` // used by "ai" steps
	Config string `json:"config"` // used by "note" / "http" steps
}

type engineWorkflow struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Trigger     string       `json:"trigger"`
	Steps       []engineStep `json:"steps"`
	Status      string       `json:"status"`
	CreatedBy   *string      `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type engineRun struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	Status     string    `json:"status"`
	Output     string    `json:"output"`
	CreatedAt  time.Time `json:"created_at"`
}

// normalizeTrigger keeps the trigger within the allowed set, defaulting to manual.
func normalizeTrigger(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "scheduled":
		return "scheduled"
	case "event":
		return "event"
	default:
		return "manual"
	}
}

// normalizeEngineStatus keeps the workflow status within the allowed set.
func normalizeEngineStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "paused":
		return "paused"
	case "archived":
		return "archived"
	default:
		return "active"
	}
}

// normalizeStepType keeps a step type within the allowed set, defaulting to note.
func normalizeStepType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ai":
		return "ai"
	case "http":
		return "http"
	default:
		return "note"
	}
}

// normalizeSteps cleans/normalizes the incoming step list.
func normalizeSteps(in []engineStep) []engineStep {
	out := make([]engineStep, 0, len(in))
	for _, s := range in {
		s.Type = normalizeStepType(s.Type)
		s.Label = strings.TrimSpace(s.Label)
		if s.Label == "" && s.Type == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

// scanWorkflow reads a workflow row, unmarshaling the JSONB steps column.
func scanWorkflow(row interface{ Scan(...any) error }) (engineWorkflow, error) {
	var w engineWorkflow
	var id uuid.UUID
	var stepsRaw []byte
	if err := row.Scan(&id, &w.Name, &w.Description, &w.Trigger, &stepsRaw, &w.Status, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return w, err
	}
	w.ID = id.String()
	w.Steps = []engineStep{}
	if len(stepsRaw) > 0 {
		_ = json.Unmarshal(stepsRaw, &w.Steps)
	}
	if w.Steps == nil {
		w.Steps = []engineStep{}
	}
	return w, nil
}

const workflowCols = `id, name, description, trigger, steps, status, created_by, created_at, updated_at`

// ListWorkflows returns the workspace's workflows. GET /api/v1/engines
func (h *EnginesHandler) ListWorkflows(c *gin.Context) {
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
		SELECT `+workflowCols+`
		FROM   engines_workflows
		WHERE  workspace_id = $1
		ORDER  BY updated_at DESC
	`, wsID)
	if err != nil {
		RespondInternalErr(c, "list workflows", err)
		return
	}
	defer rows.Close()

	workflows := make([]engineWorkflow, 0)
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err != nil {
			RespondInternalErr(c, "scan workflow", err)
			return
		}
		workflows = append(workflows, w)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate workflows", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"workflows": workflows, "count": len(workflows)})
}

type workflowInput struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Trigger     string       `json:"trigger"`
	Steps       []engineStep `json:"steps"`
	Status      string       `json:"status"`
}

// CreateWorkflow adds a workflow. POST /api/v1/engines
func (h *EnginesHandler) CreateWorkflow(c *gin.Context) {
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
	var in workflowInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}

	stepsJSON, _ := json.Marshal(normalizeSteps(in.Steps))
	row := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO engines_workflows (workspace_id, name, description, trigger, steps, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+workflowCols+`
	`, wsID, strings.TrimSpace(in.Name), in.Description, normalizeTrigger(in.Trigger), stepsJSON, normalizeEngineStatus(in.Status), user.ID)
	w, err := scanWorkflow(row)
	if err != nil {
		RespondInternalErr(c, "create workflow", err)
		return
	}
	c.JSON(http.StatusCreated, w)
}

// UpdateWorkflow edits a workflow. PUT /api/v1/engines/:id
func (h *EnginesHandler) UpdateWorkflow(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid workflow id")
		return
	}
	var in workflowInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	stepsJSON, _ := json.Marshal(normalizeSteps(in.Steps))
	row := h.pool.QueryRow(c.Request.Context(), `
		UPDATE engines_workflows
		SET    name        = COALESCE(NULLIF($3,''), name),
		       description = $4,
		       trigger     = $5,
		       steps       = $6,
		       status      = $7,
		       updated_at  = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING `+workflowCols+`
	`, id, wsID, strings.TrimSpace(in.Name), in.Description, normalizeTrigger(in.Trigger), stepsJSON, normalizeEngineStatus(in.Status))
	w, err := scanWorkflow(row)
	if err != nil {
		RespondNotFoundErr(c, "workflow")
		return
	}
	c.JSON(http.StatusOK, w)
}

// DeleteWorkflow removes a workflow (and its runs, via ON DELETE CASCADE).
// DELETE /api/v1/engines/:id
func (h *EnginesHandler) DeleteWorkflow(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid workflow id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM engines_workflows WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete workflow", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "workflow")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"})
}

// executeSteps runs a workflow's steps in order, returning the combined output
// and whether any step errored. AI steps call Claude; note/http steps record.
func (h *EnginesHandler) executeSteps(ctx context.Context, w engineWorkflow) (string, bool) {
	var out strings.Builder
	hadError := false

	fmt.Fprintf(&out, "# %s\n", w.Name)
	if strings.TrimSpace(w.Description) != "" {
		fmt.Fprintf(&out, "%s\n", strings.TrimSpace(w.Description))
	}
	out.WriteString("\n")

	if len(w.Steps) == 0 {
		out.WriteString("_No steps defined for this workflow._\n")
		return out.String(), false
	}

	llm := services.NewLLMService(h.cfg, "")
	aiAvailable := h.cfg.GetActiveProvider() != ""

	for i, s := range w.Steps {
		label := s.Label
		if label == "" {
			label = fmt.Sprintf("Step %d", i+1)
		}
		fmt.Fprintf(&out, "## %d. %s (%s)\n", i+1, label, s.Type)

		switch s.Type {
		case "ai":
			if strings.TrimSpace(s.Prompt) == "" {
				out.WriteString("_Skipped: no prompt provided._\n\n")
				continue
			}
			if !aiAvailable {
				out.WriteString("_Skipped: no AI provider configured._\n\n")
				hadError = true
				continue
			}
			system := "You are a step in an automated business workflow named \"" + w.Name + "\". " +
				"Execute the instruction precisely and return only the useful result, no preamble."
			resp, err := llm.ChatComplete(ctx,
				[]services.ChatMessage{{Role: "user", Content: s.Prompt}}, system)
			if err != nil {
				fmt.Fprintf(&out, "_Error running AI step: %s_\n\n", err.Error())
				hadError = true
				continue
			}
			fmt.Fprintf(&out, "%s\n\n", strings.TrimSpace(resp))
		case "http":
			cfg := strings.TrimSpace(s.Config)
			if cfg == "" {
				out.WriteString("_HTTP step recorded (no config)._\n\n")
			} else {
				fmt.Fprintf(&out, "HTTP step recorded: %s\n\n", cfg)
			}
		default: // note
			body := strings.TrimSpace(s.Config)
			if body == "" {
				body = strings.TrimSpace(s.Prompt)
			}
			if body == "" {
				out.WriteString("_Note recorded._\n\n")
			} else {
				fmt.Fprintf(&out, "%s\n\n", body)
			}
		}
	}

	return out.String(), hadError
}

// RunWorkflow executes a workflow and stores a run. POST /api/v1/engines/:id/run
func (h *EnginesHandler) RunWorkflow(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid workflow id")
		return
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		SELECT `+workflowCols+`
		FROM   engines_workflows
		WHERE  id = $1 AND workspace_id = $2
	`, id, wsID)
	w, err := scanWorkflow(row)
	if err != nil {
		RespondNotFoundErr(c, "workflow")
		return
	}

	output, hadError := h.executeSteps(c.Request.Context(), w)
	status := "done"
	if hadError {
		status = "error"
	}

	var run engineRun
	var rid, wfid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO engines_runs (workspace_id, workflow_id, status, output)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workflow_id, status, output, created_at
	`, wsID, id, status, output).
		Scan(&rid, &wfid, &run.Status, &run.Output, &run.CreatedAt)
	if err != nil {
		RespondInternalErr(c, "store run", err)
		return
	}
	run.ID = rid.String()
	run.WorkflowID = wfid.String()
	c.JSON(http.StatusOK, run)
}

// ListRuns returns recent run history for a workflow.
// GET /api/v1/engines/:id/runs
func (h *EnginesHandler) ListRuns(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid workflow id")
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, workflow_id, status, output, created_at
		FROM   engines_runs
		WHERE  workspace_id = $1 AND workflow_id = $2
		ORDER  BY created_at DESC
		LIMIT  25
	`, wsID, id)
	if err != nil {
		RespondInternalErr(c, "list runs", err)
		return
	}
	defer rows.Close()

	runs := make([]engineRun, 0)
	for rows.Next() {
		var run engineRun
		var rid, wfid uuid.UUID
		if err := rows.Scan(&rid, &wfid, &run.Status, &run.Output, &run.CreatedAt); err != nil {
			RespondInternalErr(c, "scan run", err)
			return
		}
		run.ID = rid.String()
		run.WorkflowID = wfid.String()
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate runs", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs, "count": len(runs)})
}
