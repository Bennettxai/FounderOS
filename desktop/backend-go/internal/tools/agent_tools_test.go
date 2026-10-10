package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAgentToolInterface(t *testing.T) {
	// Verify GetProjectTool implements AgentTool interface
	var _ AgentTool = &GetProjectTool{}
	var _ AgentTool = &GetTaskTool{}
	var _ AgentTool = &GetClientTool{}
	var _ AgentTool = &ListTasksTool{}
	var _ AgentTool = &ListProjectsTool{}
	var _ AgentTool = &SearchDocumentsTool{}
	var _ AgentTool = &CreateTaskTool{}
	var _ AgentTool = &UpdateTaskTool{}
	var _ AgentTool = &CreateNoteTool{}
	var _ AgentTool = &CreateProjectTool{}
	var _ AgentTool = &UpdateProjectTool{}
	var _ AgentTool = &BulkCreateTasksTool{}
	var _ AgentTool = &MoveTaskTool{}
	var _ AgentTool = &AssignTaskTool{}
	var _ AgentTool = &CreateClientTool{}
	var _ AgentTool = &UpdateClientTool{}
	var _ AgentTool = &GetTeamCapacityTool{}
	var _ AgentTool = &QueryMetricsTool{}
	var _ AgentTool = &LogActivityTool{}
}

func TestToolNames(t *testing.T) {
	tools := map[string]AgentTool{
		"get_project":            &GetProjectTool{},
		"get_task":               &GetTaskTool{},
		"get_client":             &GetClientTool{},
		"list_tasks":             &ListTasksTool{},
		"list_projects":          &ListProjectsTool{},
		"search_documents":       &SearchDocumentsTool{},
		"create_task":            &CreateTaskTool{},
		"update_task":            &UpdateTaskTool{},
		"create_note":            &CreateNoteTool{},
		"create_project":         &CreateProjectTool{},
		"update_project":         &UpdateProjectTool{},
		"bulk_create_tasks":      &BulkCreateTasksTool{},
		"move_task":              &MoveTaskTool{},
		"assign_task":            &AssignTaskTool{},
		"create_client":          &CreateClientTool{},
		"update_client":          &UpdateClientTool{},
		"get_team_capacity":      &GetTeamCapacityTool{},
		"query_metrics":          &QueryMetricsTool{},
		"log_activity":           &LogActivityTool{},
		"update_client_pipeline": &UpdateClientPipelineTool{},
		"log_client_interaction": &LogClientInteractionTool{},
	}

	for expectedName, tool := range tools {
		if tool.Name() != expectedName {
			t.Errorf("Expected tool name '%s', got '%s'", expectedName, tool.Name())
		}
	}
}

func TestToolDescriptions(t *testing.T) {
	tools := []AgentTool{
		&GetProjectTool{},
		&CreateTaskTool{},
		&QueryMetricsTool{},
	}

	for _, tool := range tools {
		desc := tool.Description()
		if desc == "" {
			t.Errorf("Tool %s has empty description", tool.Name())
		}
	}
}

func TestToolInputSchemas(t *testing.T) {
	tools := []AgentTool{
		&GetProjectTool{},
		&CreateTaskTool{},
		&QueryMetricsTool{},
		&BulkCreateTasksTool{},
	}

	for _, tool := range tools {
		schema := tool.InputSchema()
		if schema == nil {
			t.Errorf("Tool %s has nil input schema", tool.Name())
			continue
		}

		// Verify schema has type
		if _, ok := schema["type"]; !ok {
			t.Errorf("Tool %s schema missing 'type' field", tool.Name())
		}

		// Verify schema has properties
		if _, ok := schema["properties"]; !ok {
			t.Errorf("Tool %s schema missing 'properties' field", tool.Name())
		}
	}
}

func TestCreateTaskToolSchema(t *testing.T) {
	tool := &CreateTaskTool{}
	schema := tool.InputSchema()

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to get properties from schema")
	}

	// Verify required fields exist
	requiredFields := []string{"title"}
	for _, field := range requiredFields {
		if _, ok := props[field]; !ok {
			t.Errorf("Missing required field '%s' in schema", field)
		}
	}

	// Verify optional fields exist
	optionalFields := []string{"description", "priority", "project_id", "due_date"}
	for _, field := range optionalFields {
		if _, ok := props[field]; !ok {
			t.Errorf("Missing optional field '%s' in schema", field)
		}
	}
}

func TestQueryMetricsToolSchema(t *testing.T) {
	tool := &QueryMetricsTool{}
	schema := tool.InputSchema()

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to get properties from schema")
	}

	// Verify metric_type field
	metricType, ok := props["metric_type"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing metric_type field")
	}

	// Verify enum values
	enum, ok := metricType["enum"].([]string)
	if !ok {
		t.Fatal("Missing enum for metric_type")
	}

	expectedEnums := []string{"tasks", "projects", "clients", "overview"}
	for _, expected := range expectedEnums {
		found := false
		for _, actual := range enum {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing enum value '%s' in metric_type", expected)
		}
	}
}

func TestMoveTaskToolSchema(t *testing.T) {
	tool := &MoveTaskTool{}
	schema := tool.InputSchema()

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to get properties from schema")
	}

	// Verify status field has correct enum
	status, ok := props["status"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing status field")
	}

	enum, ok := status["enum"].([]string)
	if !ok {
		t.Fatal("Missing enum for status")
	}

	expectedStatuses := []string{"todo", "in_progress", "done", "cancelled"}
	for _, expected := range expectedStatuses {
		found := false
		for _, actual := range enum {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing status value '%s'", expected)
		}
	}
}

func TestToolExecuteWithoutPool(t *testing.T) {
	// Tools should handle nil pool gracefully
	tool := &GetProjectTool{pool: nil, userID: "test"}

	input := json.RawMessage(`{"project_id": "123"}`)
	_, err := tool.Execute(context.Background(), input)

	// Should return an error, not panic
	if err == nil {
		t.Log("Tool returned nil error with nil pool - may need connection")
	}
}
