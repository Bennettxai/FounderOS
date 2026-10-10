package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// PIPELINES HANDLERS
// ============================================================================

type workspacePipeline struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	PipelineType *string   `json:"pipeline_type"`
	Currency     *string   `json:"currency"`
	IsDefault    bool      `json:"is_default"`
	IsActive     bool      `json:"is_active"`
	Color        *string   `json:"color"`
	Icon         *string   `json:"icon"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// crmWorkspaceScope resolves the active workspace and proves the caller belongs
// to it. Pipelines used to be user-scoped, which allowed one user's records to
// bleed between organizations in the workspace switcher.
func (h *CRMHandler) crmWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	wsID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members
			WHERE workspace_id=$1 AND user_id=$2 AND status='active'
		)
	`, wsID, userID).Scan(&member)
	return wsID, err == nil && member
}

func (h *CRMHandler) pipelineInWorkspace(c *gin.Context, pipelineID, workspaceID uuid.UUID, userID string) bool {
	var exists bool
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM pipelines
			WHERE id=$1 AND workspace_id=$2 AND user_id=$3
		)
	`, pipelineID, workspaceID, userID).Scan(&exists)
	return err == nil && exists
}

func (h *CRMHandler) requirePipelineInActiveWorkspace(c *gin.Context, userID string) (uuid.UUID, bool) {
	pipelineID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "pipeline_id")
		return uuid.Nil, false
	}
	workspaceID, ok := h.crmWorkspaceScope(c, userID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return uuid.Nil, false
	}
	if !h.pipelineInWorkspace(c, pipelineID, workspaceID, userID) {
		utils.RespondNotFound(c, slog.Default(), "Pipeline")
		return uuid.Nil, false
	}
	return pipelineID, true
}

func scanWorkspacePipeline(row interface{ Scan(...any) error }) (workspacePipeline, error) {
	var result workspacePipeline
	var id, workspaceID uuid.UUID
	err := row.Scan(
		&id,
		&workspaceID,
		&result.UserID,
		&result.Name,
		&result.Description,
		&result.PipelineType,
		&result.Currency,
		&result.IsDefault,
		&result.IsActive,
		&result.Color,
		&result.Icon,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	result.ID = id.String()
	result.WorkspaceID = workspaceID.String()
	return result, err
}

// ListPipelines returns all active pipelines for the selected workspace.
func (h *CRMHandler) ListPipelines(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	wsID, ok := h.crmWorkspaceScope(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, workspace_id, user_id, name, description, pipeline_type, currency,
		       is_default, is_active, color, icon, created_at, updated_at
		FROM pipelines
		WHERE workspace_id=$1 AND user_id=$2 AND is_active=TRUE
		ORDER BY is_default DESC, name ASC
	`, wsID, user.ID)
	if err != nil {
		slog.Error("Failed to list pipelines", "error", err, "user_id", user.ID, "workspace_id", wsID)
		utils.RespondInternalError(c, slog.Default(), "list pipelines", nil)
		return
	}
	defer rows.Close()
	pipelines := make([]workspacePipeline, 0)
	for rows.Next() {
		pipeline, err := scanWorkspacePipeline(rows)
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "scan pipeline", nil)
			return
		}
		pipelines = append(pipelines, pipeline)
	}
	if err := rows.Err(); err != nil {
		utils.RespondInternalError(c, slog.Default(), "iterate pipelines", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pipelines": pipelines,
		"count":     len(pipelines),
	})
}

// GetPipeline returns a single pipeline by ID
func (h *CRMHandler) GetPipeline(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "pipeline_id")
		return
	}

	wsID, ok := h.crmWorkspaceScope(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	pipeline, err := scanWorkspacePipeline(h.pool.QueryRow(c.Request.Context(), `
		SELECT id, workspace_id, user_id, name, description, pipeline_type, currency,
		       is_default, is_active, color, icon, created_at, updated_at
		FROM pipelines
		WHERE id=$1 AND workspace_id=$2 AND user_id=$3
	`, id, wsID, user.ID))
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Pipeline")
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// CreatePipelineRequest represents the request to create a pipeline
type CreatePipelineRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  *string `json:"description"`
	PipelineType *string `json:"pipeline_type"` // sales, hiring, projects, custom
	Currency     *string `json:"currency"`
	IsDefault    bool    `json:"is_default"`
	Color        *string `json:"color"`
	Icon         *string `json:"icon"`
}

// CreatePipeline creates a new pipeline
func (h *CRMHandler) CreatePipeline(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	wsID, ok := h.crmWorkspaceScope(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	pipeline, err := scanWorkspacePipeline(h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO pipelines (workspace_id, user_id, name, description, pipeline_type, currency, is_default, color, icon)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, workspace_id, user_id, name, description, pipeline_type, currency,
		          is_default, is_active, color, icon, created_at, updated_at
	`, wsID, user.ID, req.Name, req.Description, req.PipelineType, req.Currency, req.IsDefault, req.Color, req.Icon))
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create pipeline", nil)
		return
	}

	c.JSON(http.StatusCreated, pipeline)
}

// UpdatePipelineRequest represents the request to update a pipeline
type UpdatePipelineRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Currency    *string `json:"currency"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	IsActive    bool    `json:"is_active"`
}

// UpdatePipeline updates an existing pipeline
func (h *CRMHandler) UpdatePipeline(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "pipeline_id")
		return
	}

	var req UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	wsID, ok := h.crmWorkspaceScope(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	pipeline, err := scanWorkspacePipeline(h.pool.QueryRow(c.Request.Context(), `
		UPDATE pipelines
		SET name=$4, description=$5, currency=$6, color=$7, icon=$8, is_active=$9, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND user_id=$3
		RETURNING id, workspace_id, user_id, name, description, pipeline_type, currency,
		          is_default, is_active, color, icon, created_at, updated_at
	`, id, wsID, user.ID, req.Name, req.Description, req.Currency, req.Color, req.Icon, req.IsActive))
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update pipeline", nil)
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// DeletePipeline deletes a pipeline
func (h *CRMHandler) DeletePipeline(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "pipeline_id")
		return
	}

	wsID, ok := h.crmWorkspaceScope(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	commandTag, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM pipelines WHERE id=$1 AND workspace_id=$2 AND user_id=$3
	`, id, wsID, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete pipeline", nil)
		return
	}
	if commandTag.RowsAffected() == 0 {
		utils.RespondNotFound(c, slog.Default(), "Pipeline")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pipeline deleted"})
}

// ============================================================================
// PIPELINE STAGES HANDLERS
// ============================================================================

// ListPipelineStages returns all stages for a pipeline
func (h *CRMHandler) ListPipelineStages(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pipelineID, ok := h.requirePipelineInActiveWorkspace(c, user.ID)
	if !ok {
		return
	}

	queries := sqlc.New(h.pool)
	stages, err := queries.ListPipelineStages(c.Request.Context(), pgtype.UUID{Bytes: pipelineID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list stages", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stages": transformPipelineStages(stages),
		"count":  len(stages),
	})
}

// CreatePipelineStageRequest represents the request to create a stage
type CreatePipelineStageRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Position    int32   `json:"position"`
	Probability *int32  `json:"probability"`
	StageType   *string `json:"stage_type"` // open, won, lost
	RottingDays *int32  `json:"rotting_days"`
	Color       *string `json:"color"`
}

// CreatePipelineStage creates a new pipeline stage
func (h *CRMHandler) CreatePipelineStage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pipelineID, ok := h.requirePipelineInActiveWorkspace(c, user.ID)
	if !ok {
		return
	}

	var req CreatePipelineStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	stage, err := queries.CreatePipelineStage(c.Request.Context(), sqlc.CreatePipelineStageParams{
		PipelineID:  pgtype.UUID{Bytes: pipelineID, Valid: true},
		Name:        req.Name,
		Description: req.Description,
		Position:    req.Position,
		Probability: req.Probability,
		StageType:   req.StageType,
		RottingDays: req.RottingDays,
		Color:       req.Color,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create stage", nil)
		return
	}

	c.JSON(http.StatusCreated, transformPipelineStage(stage))
}

// UpdatePipelineStageRequest represents the request to update a stage
type UpdatePipelineStageRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Probability *int32  `json:"probability"`
	RottingDays *int32  `json:"rotting_days"`
	Color       *string `json:"color"`
}

// UpdatePipelineStage updates an existing stage
func (h *CRMHandler) UpdatePipelineStage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	pipelineID, ok := h.requirePipelineInActiveWorkspace(c, user.ID)
	if !ok {
		return
	}

	stageID, err := uuid.Parse(c.Param("stageId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "stage_id")
		return
	}

	var req UpdatePipelineStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify the stage belongs to a pipeline owned by the current user.
	existingStage, err := queries.GetPipelineStage(c.Request.Context(), pgtype.UUID{Bytes: stageID, Valid: true})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Stage")
		return
	}
	if existingStage.PipelineID.Bytes != pipelineID {
		utils.RespondNotFound(c, slog.Default(), "Stage")
		return
	}

	stage, err := queries.UpdatePipelineStage(c.Request.Context(), sqlc.UpdatePipelineStageParams{
		ID:          pgtype.UUID{Bytes: stageID, Valid: true},
		Name:        req.Name,
		Description: req.Description,
		Probability: req.Probability,
		RottingDays: req.RottingDays,
		Color:       req.Color,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update stage", nil)
		return
	}

	c.JSON(http.StatusOK, transformPipelineStage(stage))
}

// ReorderPipelineStages reorders stages in a pipeline
func (h *CRMHandler) ReorderPipelineStages(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	pipelineID, ok := h.requirePipelineInActiveWorkspace(c, user.ID)
	if !ok {
		return
	}

	var req struct {
		StageOrders []struct {
			ID       string `json:"id"`
			Position int32  `json:"position"`
		} `json:"stage_orders"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	for _, order := range req.StageOrders {
		stageID, err := uuid.Parse(order.ID)
		if err != nil {
			continue
		}
		if _, err := h.pool.Exec(c.Request.Context(), `
			UPDATE pipeline_stages
			SET position=$3, updated_at=NOW()
			WHERE id=$1 AND pipeline_id=$2
		`, stageID, pipelineID, order.Position); err != nil {
			utils.RespondInternalError(c, slog.Default(), "reorder stages", nil)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stages reordered"})
}

// DeletePipelineStage deletes a stage
func (h *CRMHandler) DeletePipelineStage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	pipelineID, ok := h.requirePipelineInActiveWorkspace(c, user.ID)
	if !ok {
		return
	}

	stageID, err := uuid.Parse(c.Param("stageId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "stage_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify the stage belongs to a pipeline owned by the current user.
	existingStage, err := queries.GetPipelineStage(c.Request.Context(), pgtype.UUID{Bytes: stageID, Valid: true})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Stage")
		return
	}
	if existingStage.PipelineID.Bytes != pipelineID {
		utils.RespondNotFound(c, slog.Default(), "Stage")
		return
	}

	err = queries.DeletePipelineStage(c.Request.Context(), pgtype.UUID{Bytes: stageID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete stage", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stage deleted"})
}
