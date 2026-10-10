package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// ExampleSkillHandler demonstrates how to use the skill registry in a handler
type ExampleSkillHandler struct {
	registry *Registry
}

// NewExampleSkillHandler creates a new example handler
func NewExampleSkillHandler(registry *Registry) *ExampleSkillHandler {
	return &ExampleSkillHandler{
		registry: registry,
	}
}

// ListSkills returns all available skills
func (h *ExampleSkillHandler) ListSkills(c *gin.Context) {
	skills := h.registry.List()

	c.JSON(http.StatusOK, gin.H{
		"skills": skills,
		"count":  len(skills),
	})
}

// ExecuteSkillRequest represents the request body for executing a skill
type ExecuteSkillRequest struct {
	SkillName  string                 `json:"skill_name" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ExecuteSkill executes a skill by name
func (h *ExampleSkillHandler) ExecuteSkill(c *gin.Context) {
	var req ExecuteSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	slog.Info("executing skill via handler",
		"skill_name", req.SkillName,
		"params_count", len(req.Parameters))

	result, err := h.registry.Execute(ctx, req.SkillName, req.Parameters)
	if err != nil {
		slog.Error("skill execution failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSkillSchema returns the schema for a specific skill
func (h *ExampleSkillHandler) GetSkillSchema(c *gin.Context) {
	skillName := c.Param("name")

	skill, err := h.registry.Get(skillName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	schema := skill.Schema()
	if schema == nil {
		c.JSON(http.StatusOK, gin.H{
			"skill_name": skillName,
			"schema":     nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"skill_name": skillName,
		"schema":     schema,
	})
}

// SetupSkillRoutes sets up the routes for skill management
// This is an example of how to wire the skill registry into your router
func SetupSkillRoutes(router *gin.RouterGroup, registry *Registry) {
	handler := NewExampleSkillHandler(registry)

	// Public skill endpoints
	router.GET("/skills", handler.ListSkills)
	router.POST("/skills/execute", handler.ExecuteSkill)
	router.GET("/skills/:name/schema", handler.GetSkillSchema)
}

// InitializeSkillRegistry creates and configures a skill registry with OSA
// This is an example of how to initialize the registry in your main.go or setup
func InitializeSkillRegistry(osaClient *osa.ResilientClient) (*Registry, error) {
	registry := NewRegistry()

	// Register OSA skill
	osaSkill := NewOsaSkill(osaClient)
	if err := registry.Register(osaSkill); err != nil {
		return nil, fmt.Errorf("failed to register OSA skill: %w", err)
	}

	slog.Info("skill registry initialized",
		"skills_count", registry.Count())

	return registry, nil
}

// Example of how to use the skill registry programmatically
func ExampleUsage(ctx context.Context, registry *Registry) error {
	// List all available skills
	skills := registry.List()
	slog.Info("available skills", "count", len(skills))
	for _, skill := range skills {
		slog.Info("skill", "name", skill.Name, "description", skill.Description)
	}

	// Execute OSA orchestration skill
	params := map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"input":   "Create a task management app",
	}

	result, err := registry.Execute(ctx, "osa_orchestrate", params)
	if err != nil {
		return fmt.Errorf("skill execution failed: %w", err)
	}

	// Log the result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	slog.Info("skill execution result", "result", string(resultJSON))

	return nil
}
