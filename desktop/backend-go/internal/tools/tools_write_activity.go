package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogActivityTool logs an activity to the daily log
type LogActivityTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *LogActivityTool) Name() string { return "log_activity" }
func (t *LogActivityTool) Description() string {
	return "Log an activity or note to the daily log"
}
func (t *LogActivityTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{"type": "string", "description": "Activity content/note"},
			"type":    map[string]interface{}{"type": "string", "enum": []string{"note", "task", "meeting", "idea", "decision"}, "description": "Type of activity"},
		},
		"required": []string{"content"},
	}
}
func (t *LogActivityTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}
	if params.Type == "" {
		params.Type = "note"
	}

	// Generate UUID for log entry
	logID := uuid.New()

	// daily_logs table requires date field
	query := `INSERT INTO daily_logs (id, user_id, date, content, created_at, updated_at)
	          VALUES ($1, $2, CURRENT_DATE, $3, NOW(), NOW())`
	_, err := t.pool.Exec(ctx, query, logID, t.userID, params.Content)
	if err != nil {
		return "", fmt.Errorf("failed to log activity: %w", err)
	}

	typeIcon := "📝"
	switch params.Type {
	case "task":
		typeIcon = "✅"
	case "meeting":
		typeIcon = "🤝"
	case "idea":
		typeIcon = "💡"
	case "decision":
		typeIcon = "🎯"
	}

	return fmt.Sprintf("%s Activity logged: %s", typeIcon, params.Content), nil
}

// CreateArtifactTool starts a document artifact - content will be captured from chat response
type CreateArtifactTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *CreateArtifactTool) Name() string { return "create_artifact" }
func (t *CreateArtifactTool) Description() string {
	return "Start creating a document artifact. Call this FIRST with type and title, then write the document content in your response. The content you write after calling this tool will automatically be saved to the artifact."
}
func (t *CreateArtifactTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Type of document: proposal, plan, report, sop, framework, document",
				"enum":        []string{"proposal", "plan", "report", "sop", "framework", "document"},
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Title of the document",
			},
		},
		"required": []string{"type", "title"},
	}
}
func (t *CreateArtifactTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Type  string `json:"type"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Title == "" {
		return "", fmt.Errorf("title is required")
	}

	// Return a marker that the handler will use to capture content
	return fmt.Sprintf("ARTIFACT_START::%s::%s::Now write the complete document content below. Everything you write will be saved to the artifact.", params.Type, params.Title), nil
}
