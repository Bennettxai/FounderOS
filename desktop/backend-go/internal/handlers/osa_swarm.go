package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// OSASwarmHandler handles OSA swarm orchestration endpoints.
type OSASwarmHandler struct {
	osaClient *osa.ResilientClient
}

// NewOSASwarmHandler creates a new OSASwarmHandler.
func NewOSASwarmHandler(osaClient *osa.ResilientClient) *OSASwarmHandler {
	return &OSASwarmHandler{osaClient: osaClient}
}

// LaunchSwarmRequest is the body expected by HandleLaunchSwarm.
type LaunchSwarmRequest struct {
	Pattern   string                 `json:"pattern" binding:"required"`
	Task      string                 `json:"task" binding:"required"`
	Config    map[string]interface{} `json:"config,omitempty"`
	MaxAgents int                    `json:"max_agents,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// ExecuteToolRequest is the body expected by HandleExecuteTool.
type ExecuteToolRequest struct {
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// DispatchInstructionRequest is the body expected by HandleDispatchInstruction.
type DispatchInstructionRequest struct {
	SpecVersion string                 `json:"specversion"`
	Type        string                 `json:"type" binding:"required"`
	Source      string                 `json:"source" binding:"required"`
	ID          string                 `json:"id" binding:"required"`
	Data        map[string]interface{} `json:"data"`
}

// HandleLaunchSwarm - POST /api/osa/swarm/launch
func (h *OSASwarmHandler) HandleLaunchSwarm(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req LaunchSwarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.osaClient.LaunchSwarm(c.Request.Context(), osa.SwarmRequest{
		Pattern:   req.Pattern,
		Task:      req.Task,
		Config:    req.Config,
		MaxAgents: req.MaxAgents,
		SessionID: req.SessionID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to launch swarm",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleListSwarms - GET /api/osa/swarm
func (h *OSASwarmHandler) HandleListSwarms(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	swarms, err := h.osaClient.ListSwarms(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list swarms",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"swarms": swarms,
	})
}

// HandleGetSwarm - GET /api/osa/swarm/:id
func (h *OSASwarmHandler) HandleGetSwarm(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	swarmID := c.Param("id")
	if swarmID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "swarm id required"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	status, err := h.osaClient.GetSwarm(c.Request.Context(), swarmID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Swarm not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleCancelSwarm - DELETE /api/osa/swarm/:id
func (h *OSASwarmHandler) HandleCancelSwarm(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	swarmID := c.Param("id")
	if swarmID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "swarm id required"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.osaClient.CancelSwarm(c.Request.Context(), swarmID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cancel swarm",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleDispatchInstruction - POST /api/osa/fleet/:agent_id/dispatch
func (h *OSASwarmHandler) HandleDispatchInstruction(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id required"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DispatchInstructionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.osaClient.DispatchInstruction(c.Request.Context(), agentID, osa.Instruction{
		SpecVersion: req.SpecVersion,
		Type:        req.Type,
		Source:      req.Source,
		ID:          req.ID,
		Data:        req.Data,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to dispatch instruction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleListTools - GET /api/osa/tools
func (h *OSASwarmHandler) HandleListTools(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tools, err := h.osaClient.ListTools(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list tools",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": tools,
		"count": len(tools),
	})
}

// HandleExecuteTool - POST /api/osa/tools/:name/execute
func (h *OSASwarmHandler) HandleExecuteTool(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	toolName := c.Param("name")
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name required"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.osaClient.ExecuteTool(c.Request.Context(), toolName, req.Parameters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute tool",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
