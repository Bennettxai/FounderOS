package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateClientPipelineTool updates client pipeline stage
type UpdateClientPipelineTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *UpdateClientPipelineTool) Name() string { return "update_client_pipeline" }
func (t *UpdateClientPipelineTool) Description() string {
	return "Move a client to a different pipeline stage. Use this to update client status in the sales pipeline."
}
func (t *UpdateClientPipelineTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"client_id": map[string]interface{}{
				"type":        "string",
				"description": "The client UUID",
			},
			"stage": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"lead", "prospect", "proposal", "negotiation", "won", "lost"},
				"description": "The new pipeline stage",
			},
		},
		"required": []string{"client_id", "stage"},
	}
}
func (t *UpdateClientPipelineTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ClientID string `json:"client_id"`
		Stage    string `json:"stage"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	clientUUID, err := uuid.Parse(params.ClientID)
	if err != nil {
		return "", fmt.Errorf("invalid client ID")
	}

	query := `UPDATE clients SET status = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3 RETURNING name`
	var clientName string
	err = t.pool.QueryRow(ctx, query, params.Stage, clientUUID, t.userID).Scan(&clientName)
	if err != nil {
		return "", fmt.Errorf("failed to update client: %w", err)
	}

	return fmt.Sprintf("📊 Client pipeline updated!\n\n**Client:** %s\n**New Stage:** %s", clientName, params.Stage), nil
}

// LogClientInteractionTool logs a client interaction
type LogClientInteractionTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *LogClientInteractionTool) Name() string { return "log_client_interaction" }
func (t *LogClientInteractionTool) Description() string {
	return "Log an interaction with a client (meeting, call, email, etc.)"
}
func (t *LogClientInteractionTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"client_id": map[string]interface{}{
				"type":        "string",
				"description": "The client UUID",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"meeting", "call", "email", "note"},
				"description": "Type of interaction",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "Summary of the interaction",
			},
		},
		"required": []string{"client_id", "type", "summary"},
	}
}
func (t *LogClientInteractionTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ClientID string `json:"client_id"`
		Type     string `json:"type"`
		Summary  string `json:"summary"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	clientUUID, err := uuid.Parse(params.ClientID)
	if err != nil {
		return "", fmt.Errorf("invalid client ID")
	}

	query := `INSERT INTO client_interactions (client_id, user_id, type, summary, created_at)
	          VALUES ($1, $2, $3, $4, NOW()) RETURNING id`
	var interactionID uuid.UUID
	err = t.pool.QueryRow(ctx, query, clientUUID, t.userID, params.Type, params.Summary).Scan(&interactionID)
	if err != nil {
		return "", fmt.Errorf("failed to log interaction: %w", err)
	}

	typeIcon := "📝"
	switch params.Type {
	case "meeting":
		typeIcon = "🤝"
	case "call":
		typeIcon = "📞"
	case "email":
		typeIcon = "📧"
	}

	return fmt.Sprintf("%s Interaction logged!\n\n**Type:** %s\n**Summary:** %s", typeIcon, params.Type, params.Summary), nil
}

// CreateClientTool creates a new client
type CreateClientTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *CreateClientTool) Name() string { return "create_client" }
func (t *CreateClientTool) Description() string {
	return "Create a new client in the CRM"
}
func (t *CreateClientTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":           map[string]interface{}{"type": "string", "description": "Client/company name"},
			"email":          map[string]interface{}{"type": "string", "description": "Primary email"},
			"pipeline_stage": map[string]interface{}{"type": "string", "enum": []string{"lead", "prospect", "proposal", "negotiation", "won", "lost"}},
			"notes":          map[string]interface{}{"type": "string", "description": "Initial notes"},
		},
		"required": []string{"name"},
	}
}
func (t *CreateClientTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		PipelineStage string `json:"pipeline_stage"`
		Notes         string `json:"notes"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}
	if params.PipelineStage == "" {
		params.PipelineStage = "lead"
	}

	query := `INSERT INTO clients (user_id, name, email, pipeline_stage, notes, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`
	var clientID uuid.UUID
	err := t.pool.QueryRow(ctx, query, t.userID, params.Name, params.Email, params.PipelineStage, params.Notes).Scan(&clientID)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	return fmt.Sprintf("✅ Client created!\n\n**Name:** %s\n**ID:** %s\n**Stage:** %s", params.Name, clientID, params.PipelineStage), nil
}

// UpdateClientTool updates client information
type UpdateClientTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *UpdateClientTool) Name() string { return "update_client" }
func (t *UpdateClientTool) Description() string {
	return "Update client information"
}
func (t *UpdateClientTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"client_id": map[string]interface{}{"type": "string", "description": "Client UUID"},
			"name":      map[string]interface{}{"type": "string"},
			"email":     map[string]interface{}{"type": "string"},
			"notes":     map[string]interface{}{"type": "string"},
		},
		"required": []string{"client_id"},
	}
}
func (t *UpdateClientTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ClientID string `json:"client_id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Notes    string `json:"notes"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	clientUUID, err := uuid.Parse(params.ClientID)
	if err != nil {
		return "", fmt.Errorf("invalid client ID")
	}

	updates := []string{}
	args := []interface{}{clientUUID, t.userID}
	argNum := 3

	if params.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, params.Name)
		argNum++
	}
	if params.Email != "" {
		updates = append(updates, fmt.Sprintf("email = $%d", argNum))
		args = append(args, params.Email)
		argNum++
	}
	if params.Notes != "" {
		updates = append(updates, fmt.Sprintf("notes = $%d", argNum))
		args = append(args, params.Notes)
		argNum++
	}

	if len(updates) == 0 {
		return "No updates provided", nil
	}

	query := fmt.Sprintf("UPDATE clients SET %s, updated_at = NOW() WHERE id = $1 AND user_id = $2", joinStrings(updates, ", "))
	_, err = t.pool.Exec(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to update client: %w", err)
	}

	return "✅ Client updated successfully", nil
}
