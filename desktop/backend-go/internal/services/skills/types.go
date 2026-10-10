package skills

import (
	"context"
	"encoding/json"
)

// Skill represents an executable capability that can be registered and invoked
type Skill interface {
	// Name returns the unique identifier for this skill
	Name() string

	// Description returns a human-readable description of what this skill does
	Description() string

	// Execute runs the skill with the given parameters
	// params should be a JSON-serializable map of parameters
	// Returns the result as a JSON-serializable value, or an error
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)

	// Schema returns the JSON schema for the skill's parameters (optional)
	// This helps with validation and documentation
	Schema() *SkillSchema
}

// SkillSchema defines the input/output schema for a skill
type SkillSchema struct {
	// InputSchema is the JSON schema for the input parameters
	InputSchema json.RawMessage `json:"input_schema,omitempty"`

	// OutputSchema is the JSON schema for the output
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`

	// Examples provides example inputs and outputs
	Examples []SkillExample `json:"examples,omitempty"`
}

// SkillExample shows an example of how to use the skill
type SkillExample struct {
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
}

// SkillMetadata provides additional information about a skill
type SkillMetadata struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Version     string        `json:"version,omitempty"`
	Author      string        `json:"author,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Schema      *SkillSchema  `json:"schema,omitempty"`
}

// SkillExecutionResult wraps the result of skill execution with metadata
type SkillExecutionResult struct {
	SkillName string      `json:"skill_name"`
	Success   bool        `json:"success"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
}
