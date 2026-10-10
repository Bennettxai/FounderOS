package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// ProjectTemplateHandler exposes reusable project blueprints (the Projects
// primitive configured into a feature). A template carries phases (milestones)
// and deliverables; "create from template" materializes them into a project so
// a known engagement like the Growth Systems Audit becomes a one-click project.
//
// Templates are workspace-scoped (workspace_id) plus global built-ins
// (workspace_id IS NULL). Mirrors GlossaryHandler: raw pgx, X-Workspace-ID gate.
type ProjectTemplateHandler struct {
	pool *pgxpool.Pool
}

func NewProjectTemplateHandler(pool *pgxpool.Pool) *ProjectTemplateHandler {
	return &ProjectTemplateHandler{pool: pool}
}

// projectTemplate is the wire shape. phases/deliverables stay as raw JSON so the
// frontend can render them without the backend knowing their internal shape.
type projectTemplate struct {
	ID           string          `json:"id"`
	WorkspaceID  *string         `json:"workspace_id"`
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Phases       json.RawMessage `json:"phases"`
	Deliverables json.RawMessage `json:"deliverables"`
	IsBuiltin    bool            `json:"is_builtin"`
	CreatedBy    *string         `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *ProjectTemplateHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListTemplates returns global built-ins plus this workspace's own templates.
// GET /api/projects/templates
func (h *ProjectTemplateHandler) ListTemplates(c *gin.Context) {
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
		SELECT id, workspace_id, key, name, description, phases, deliverables,
		       is_builtin, created_by, created_at, updated_at
		FROM   delivery_templates
		WHERE  workspace_id IS NULL OR workspace_id = $1
		ORDER  BY is_builtin DESC, name ASC
	`, wsID)
	if err != nil {
		RespondInternalErr(c, "list project templates", err)
		return
	}
	defer rows.Close()

	templates := make([]projectTemplate, 0)
	for rows.Next() {
		var t projectTemplate
		var id uuid.UUID
		var ws *uuid.UUID
		if err := rows.Scan(&id, &ws, &t.Key, &t.Name, &t.Description, &t.Phases,
			&t.Deliverables, &t.IsBuiltin, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan project template", err)
			return
		}
		t.ID = id.String()
		if ws != nil {
			s := ws.String()
			t.WorkspaceID = &s
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate project templates", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": templates, "count": len(templates)})
}

type createFromTemplateInput struct {
	TemplateKey string  `json:"template_key"`
	Name        string  `json:"name"`
	ClientName  *string `json:"client_name"`
	DueDate     *string `json:"due_date"`
	Priority    *string `json:"priority"`
}

// CreateFromTemplate materializes a template into a real project: it copies the
// template's phases + deliverables into project_metadata and stamps project_type
// with the template key. Workspace-scoped via X-Workspace-ID so members see it.
// POST /api/projects/templates/:key/use
func (h *ProjectTemplateHandler) CreateFromTemplate(c *gin.Context) {
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

	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		RespondBadRequestErr(c, "template key is required")
		return
	}

	var in createFromTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	// Load the template (global built-in or this workspace's own).
	var (
		tplName      string
		tplDesc      string
		phases       json.RawMessage
		deliverables json.RawMessage
	)
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT name, description, phases, deliverables
		FROM   delivery_templates
		WHERE  key = $1 AND (workspace_id IS NULL OR workspace_id = $2)
		ORDER  BY (workspace_id = $2) DESC
		LIMIT  1
	`, key, wsID).Scan(&tplName, &tplDesc, &phases, &deliverables)
	if err != nil {
		RespondNotFoundErr(c, "template")
		return
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = tplName
		if in.ClientName != nil && strings.TrimSpace(*in.ClientName) != "" {
			name = tplName + " — " + strings.TrimSpace(*in.ClientName)
		}
	}

	// projectstatus / projectpriority enums store uppercase labels.
	priority := "MEDIUM"
	if in.Priority != nil && strings.TrimSpace(*in.Priority) != "" {
		switch strings.ToUpper(strings.TrimSpace(*in.Priority)) {
		case "CRITICAL", "HIGH", "MEDIUM", "LOW":
			priority = strings.ToUpper(strings.TrimSpace(*in.Priority))
		}
	}

	// Snapshot the template into project_metadata so later template edits do not
	// retroactively change a live project. The frontend renders these phases.
	metadata := map[string]interface{}{
		"template_key":  key,
		"template_name": tplName,
		"phases":        rawOrEmptyArray(phases),
		"deliverables":  rawOrEmptyArray(deliverables),
	}
	metadataJSON, _ := json.Marshal(metadata)

	var clientName *string
	if in.ClientName != nil && strings.TrimSpace(*in.ClientName) != "" {
		cn := strings.TrimSpace(*in.ClientName)
		clientName = &cn
	}

	var dueDate *time.Time
	if in.DueDate != nil && strings.TrimSpace(*in.DueDate) != "" {
		if t, perr := time.Parse("2006-01-02", strings.TrimSpace(*in.DueDate)); perr == nil {
			dueDate = &t
		}
	}

	var (
		projectID uuid.UUID
		createdAt time.Time
	)
	err = h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO projects
		    (user_id, name, description, status, priority, client_name,
		     project_type, project_metadata, due_date, owner_id, workspace_id)
		VALUES
		    ($1, $2, $3, 'ACTIVE'::projectstatus, $4::projectpriority, $5, $6, $7::jsonb, $8, $1, $9)
		RETURNING id, created_at
	`, user.ID, name, tplDesc, priority, clientName, key, string(metadataJSON), dueDate, wsID).
		Scan(&projectID, &createdAt)
	if err != nil {
		RespondInternalErr(c, "create project from template", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":               projectID.String(),
		"name":             name,
		"description":      tplDesc,
		"status":           "active",
		"priority":         strings.ToLower(priority),
		"client_name":      clientName,
		"project_type":     key,
		"project_metadata": metadata,
		"due_date":         in.DueDate,
		"created_at":       createdAt.Format(time.RFC3339),
		"workspace_id":     wsID.String(),
	})
}

// rawOrEmptyArray decodes raw JSON into a generic value, defaulting to an empty
// array so the response is always well-formed even if a column was NULL.
func rawOrEmptyArray(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return []interface{}{}
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return []interface{}{}
	}
	return v
}
