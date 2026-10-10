// Package handlers provides HTTP handlers for BusinessOS.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/sorx"
	"github.com/rhl/businessos-backend/internal/utils"
)

// SorxHandler handles Sorx integration endpoints.
type SorxHandler struct {
	sorxService *services.SorxService
	engine      *sorx.Engine
}

// NewSorxHandler creates a new Sorx handler.
func NewSorxHandler(sorxService *services.SorxService, pool *pgxpool.Pool, logger *slog.Logger) *SorxHandler {
	return &SorxHandler{
		sorxService: sorxService,
		engine:      sorx.NewEngine(pool, logger),
	}
}

// ============================================================================
// Credential Endpoints
// ============================================================================

// RequestCredentialTicket handles POST /api/sorx/credential-ticket
// Sorx requests a ticket to retrieve a credential.
func (h *SorxHandler) RequestCredentialTicket(c *gin.Context) {
	var req services.CredentialTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid credential ticket request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid request: "+err.Error())
		return
	}

	// Validate the request
	if err := h.sorxService.ValidateTicketRequest(c.Request.Context(), req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Credential ticket validation failed", "error", err)
		utils.RespondForbidden(c, slog.Default(), err.Error())
		return
	}

	// Issue ticket
	ticket, err := h.sorxService.IssueCredentialTicket(c.Request.Context(), req)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to issue credential ticket", "error", err)
		utils.RespondInternalError(c, slog.Default(), "issue ticket", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"ticket":  ticket,
	})
}

// RedeemCredential handles POST /api/sorx/redeem-credential
// Sorx exchanges a ticket for the encrypted credential.
func (h *SorxHandler) RedeemCredential(c *gin.Context) {
	var ticket services.CredentialTicket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid ticket format", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid ticket: "+err.Error())
		return
	}

	// Redeem the ticket
	response, err := h.sorxService.RedeemTicket(c.Request.Context(), &ticket)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to redeem ticket", "error", err)
		utils.RespondForbidden(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"credential": response,
	})
}

// ============================================================================
// Callback Endpoint
// ============================================================================

// HandleCallback handles POST /api/sorx/callback
// Main callback endpoint for all Sorx skill callbacks.
func (h *SorxHandler) HandleCallback(c *gin.Context) {
	var req services.CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid callback format", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid callback: "+err.Error())
		return
	}

	// Process the callback
	response, err := h.sorxService.HandleCallback(c.Request.Context(), req)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to handle callback", "error", err)
		utils.RespondInternalError(c, slog.Default(), "handle callback", err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// Decision Endpoints
// ============================================================================

// GetPendingDecisions handles GET /api/sorx/decisions
// Returns all pending decisions for the authenticated user.
func (h *SorxHandler) GetPendingDecisions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	decisions, err := h.sorxService.GetPendingDecisions(c.Request.Context(), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to get pending decisions",
			"user_id", userID,
			"error", err)
		utils.RespondInternalError(c, slog.Default(), "get decisions", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// GetDecision handles GET /api/sorx/decisions/:id
// Returns a single pending decision after verifying ownership (OWASP A01: IDOR prevention).
func (h *SorxHandler) GetDecision(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid decision ID format", "id", idStr, "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid decision ID")
		return
	}

	decision, err := h.sorxService.GetPendingDecision(c.Request.Context(), id)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Decision not found", "id", id, "error", err)
		utils.RespondNotFound(c, slog.Default(), "Decision not found")
		return
	}

	// Verify ownership: the requesting user must own the decision.
	if ownerID, ok := decision["user_id"].(string); !ok || ownerID != user.ID {
		slog.WarnContext(c.Request.Context(), "IDOR attempt: user tried to access another user's decision",
			slog.String("requesting_user", user.ID),
			slog.String("decision_id", id.String()),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"decision": decision,
	})
}

// RespondToDecisionRequest represents the request body for responding to a decision.
type RespondToDecisionRequest struct {
	Decision string                 `json:"decision" binding:"required"`
	Inputs   map[string]interface{} `json:"inputs"`
	Comment  string                 `json:"comment"`
}

// RespondToDecision handles POST /api/sorx/decisions/:id/respond
// Submits a human's decision response.
func (h *SorxHandler) RespondToDecision(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid decision ID format", "id", idStr, "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid decision ID")
		return
	}

	// Verify ownership before accepting the response (OWASP A01: IDOR prevention).
	decision, err := h.sorxService.GetPendingDecision(c.Request.Context(), id)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Decision not found", "id", id, "error", err)
		utils.RespondNotFound(c, slog.Default(), "Decision not found")
		return
	}
	if ownerID, ok := decision["user_id"].(string); !ok || ownerID != userID {
		slog.WarnContext(c.Request.Context(), "IDOR attempt: user tried to respond to another user's decision",
			slog.String("requesting_user", userID),
			slog.String("decision_id", id.String()),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req RespondToDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid decision response format", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid request: "+err.Error())
		return
	}

	// Record the decision
	if err := h.sorxService.RespondToDecision(c.Request.Context(), id, userID, req.Decision, req.Inputs); err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to record decision",
			"decision_id", id,
			"user_id", userID,
			"error", err)
		utils.RespondInternalError(c, slog.Default(), "record decision", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Decision recorded",
	})
}

// ============================================================================
// Skill Execution Endpoints
// ============================================================================

// TriggerSkillRequest represents a request to trigger a skill.
type TriggerSkillRequest struct {
	SkillID string                 `json:"skill_id" binding:"required"`
	Params  map[string]interface{} `json:"params"`
}

// TriggerSkill handles POST /api/sorx/execute
// Triggers a skill execution from BusinessOS.
func (h *SorxHandler) TriggerSkill(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req TriggerSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid trigger skill request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid request: "+err.Error())
		return
	}

	// Execute skill using the Sorx engine
	execution, err := h.engine.ExecuteSkill(c.Request.Context(), sorx.ExecuteRequest{
		SkillID: req.SkillID,
		UserID:  userID,
		Params:  req.Params,
	})

	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to execute skill",
			"skill_id", req.SkillID,
			"user_id", userID,
			"error", err)
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"execution_id": execution.ID,
		"skill_id":     execution.SkillID,
		"status":       execution.Status,
		"message":      "Skill execution started",
	})
}

// GetSkillExecution handles GET /api/sorx/executions/:id
// Returns the status of a skill execution.
func (h *SorxHandler) GetSkillExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid execution ID format", "id", idStr, "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid execution ID")
		return
	}

	// Get execution from engine
	execution, ok := h.engine.GetExecution(id)
	if !ok {
		slog.ErrorContext(c.Request.Context(), "Execution not found", "id", id)
		utils.RespondNotFound(c, slog.Default(), "Execution not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"execution": gin.H{
			"id":           execution.ID,
			"skill_id":     execution.SkillID,
			"status":       execution.Status,
			"current_step": execution.CurrentStep,
			"params":       execution.Params,
			"result":       execution.Result,
			"error":        execution.Error,
			"step_results": execution.StepResults,
			"started_at":   execution.StartedAt,
			"completed_at": execution.CompletedAt,
		},
	})
}

// ListSkills handles GET /api/sorx/skills
// Returns all available skills.
func (h *SorxHandler) ListSkills(c *gin.Context) {
	skills := h.engine.ListSkills()

	// Convert to response format
	var skillList []gin.H
	for _, skill := range skills {
		skillList = append(skillList, gin.H{
			"id":                    skill.ID,
			"name":                  skill.Name,
			"description":           skill.Description,
			"category":              skill.Category,
			"required_integrations": skill.RequiredIntegrations,
			"steps_count":           len(skill.Steps),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"skills":  skillList,
		"count":   len(skillList),
	})
}

// GetSkill handles GET /api/sorx/skills/:id
// Returns a single skill definition.
func (h *SorxHandler) GetSkill(c *gin.Context) {
	skillID := c.Param("id")

	skills := h.engine.ListSkills()
	for _, skill := range skills {
		if skill.ID == skillID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"skill":   skill,
			})
			return
		}
	}

	slog.ErrorContext(c.Request.Context(), "Skill not found", "skill_id", skillID)
	utils.RespondNotFound(c, slog.Default(), "Skill not found")
}

// ============================================================================
// Skill Command Endpoints
// ============================================================================

// ListSkillCommands handles GET /api/sorx/commands
// Returns all skill-based commands that can trigger Sorx skills.
func (h *SorxHandler) ListSkillCommands(c *gin.Context) {
	commands := sorx.ListSkillCommands()

	// Enrich with skill availability info
	var enrichedCommands []gin.H
	for _, cmd := range commands {
		// Check if the associated skill exists
		skills := h.engine.ListSkills()
		skillExists := false
		var skillInfo *sorx.SkillDefinition
		for _, skill := range skills {
			if skill.ID == cmd.SkillID {
				skillExists = true
				skillInfo = skill
				break
			}
		}

		enriched := gin.H{
			"name":         cmd.Name,
			"display_name": cmd.DisplayName,
			"description":  cmd.Description,
			"icon":         cmd.Icon,
			"category":     cmd.Category,
			"skill_id":     cmd.SkillID,
			"params":       cmd.Params,
			"skill_exists": skillExists,
		}

		if skillInfo != nil {
			enriched["required_integrations"] = skillInfo.RequiredIntegrations
			enriched["steps_count"] = len(skillInfo.Steps)
		}

		enrichedCommands = append(enrichedCommands, enriched)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"commands": enrichedCommands,
		"count":    len(enrichedCommands),
	})
}

// ExecuteSkillCommandRequest represents a request to execute a skill command.
type ExecuteSkillCommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params"`
}

// ExecuteSkillCommand handles POST /api/sorx/commands/execute
// Executes a skill command by triggering the associated Sorx skill.
func (h *SorxHandler) ExecuteSkillCommand(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var req ExecuteSkillCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid skill command request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "Invalid request: "+err.Error())
		return
	}

	// Get the skill command
	cmd, ok := sorx.GetSkillCommand(req.Command)
	if !ok {
		slog.ErrorContext(c.Request.Context(), "Unknown skill command", "command", req.Command)
		utils.RespondNotFound(c, slog.Default(), "Unknown skill command: "+req.Command)
		return
	}

	// Execute the associated skill
	execution, err := h.engine.ExecuteSkill(c.Request.Context(), sorx.ExecuteRequest{
		SkillID: cmd.SkillID,
		UserID:  userID,
		Params:  req.Params,
	})

	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to execute skill command",
			"command", req.Command,
			"skill_id", cmd.SkillID,
			"user_id", userID,
			"error", err)
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"execution_id": execution.ID,
		"skill_id":     execution.SkillID,
		"command":      req.Command,
		"status":       execution.Status,
		"message":      "Skill command execution started",
	})
}
