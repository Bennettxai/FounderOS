package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OsaSkill implements the Skill interface for OSA orchestration
type OsaSkill struct {
	client *osa.ResilientClient
}

// NewOsaSkill creates a new OSA skill using the resilient client
func NewOsaSkill(client *osa.ResilientClient) *OsaSkill {
	return &OsaSkill{
		client: client,
	}
}

// Name returns the skill identifier
func (s *OsaSkill) Name() string {
	return "osa_orchestrate"
}

// Description returns what this skill does
func (s *OsaSkill) Description() string {
	return "Triggers the full 21-agent OSA orchestration workflow for complex app generation tasks"
}

// Schema returns the parameter schema for this skill
func (s *OsaSkill) Schema() *SkillSchema {
	inputSchema := json.RawMessage(`{
		"type": "object",
		"required": ["user_id", "input"],
		"properties": {
			"user_id": {
				"type": "string",
				"format": "uuid",
				"description": "The user ID requesting the orchestration"
			},
			"input": {
				"type": "string",
				"description": "The natural language description of what to generate"
			},
			"workspace_id": {
				"type": "string",
				"format": "uuid",
				"description": "Optional workspace ID for context"
			},
			"phase": {
				"type": "string",
				"enum": ["analysis", "strategy", "development", "deployment"],
				"description": "Optional orchestration phase to target"
			},
			"context": {
				"type": "object",
				"description": "Additional context data for orchestration"
			}
		}
	}`)

	outputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"success": {
				"type": "boolean",
				"description": "Whether orchestration succeeded"
			},
			"output": {
				"type": "string",
				"description": "The orchestration result or generated content"
			},
			"agents_used": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of agents that participated"
			},
			"execution_ms": {
				"type": "integer",
				"description": "Execution time in milliseconds"
			},
			"next_step": {
				"type": "string",
				"description": "Suggested next action"
			}
		}
	}`)

	return &SkillSchema{
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Examples: []SkillExample{
			{
				Description: "Generate a simple CRUD app",
				Input: map[string]interface{}{
					"user_id": "550e8400-e29b-41d4-a716-446655440000",
					"input":   "Create a task management app with user authentication",
				},
				Output: map[string]interface{}{
					"success":    true,
					"output":     "Generated task management app with authentication...",
					"agents_used": []string{"architect", "backend-specialist", "frontend-specialist"},
					"execution_ms": 15000,
				},
			},
		},
	}
}

// Execute runs the OSA orchestration
func (s *OsaSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse required parameters
	userIDStr, ok := params["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required and must be a string")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	input, ok := params["input"].(string)
	if !ok {
		return nil, fmt.Errorf("input is required and must be a string")
	}

	// Build orchestration request
	req := &osa.OrchestrateRequest{
		UserID: userID,
		Input:  input,
	}

	// Optional workspace_id
	if workspaceIDStr, ok := params["workspace_id"].(string); ok && workspaceIDStr != "" {
		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid workspace_id: %w", err)
		}
		req.WorkspaceID = workspaceID
	}

	// Optional phase
	if phase, ok := params["phase"].(string); ok {
		req.Phase = phase
	}

	// Optional context
	if contextData, ok := params["context"].(map[string]interface{}); ok {
		req.Context = contextData
	}

	// Log execution
	slog.Info("executing OSA orchestration",
		"user_id", userID,
		"input_length", len(input),
		"workspace_id", req.WorkspaceID,
		"phase", req.Phase)

	// Execute orchestration
	resp, err := s.client.Orchestrate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("orchestration failed: %w", err)
	}

	// Convert response to map for JSON serialization
	result := map[string]interface{}{
		"success":        resp.Success,
		"output":         resp.Output,
		"agents_used":    resp.AgentsUsed,
		"execution_ms":   resp.ExecutionTime,
		"next_step":      resp.NextStep,
	}

	// Include additional data if present
	if resp.Data != nil {
		result["data"] = resp.Data
	}

	slog.Info("OSA orchestration completed",
		"user_id", userID,
		"success", resp.Success,
		"execution_ms", resp.ExecutionTime,
		"agents_count", len(resp.AgentsUsed))

	return result, nil
}

// HealthCheck checks if OSA is available
func (s *OsaSkill) HealthCheck(ctx context.Context) error {
	resp, err := s.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("OSA health check failed: %w", err)
	}

	if resp.Status != "healthy" {
		return fmt.Errorf("OSA is unhealthy: status=%s", resp.Status)
	}

	return nil
}
