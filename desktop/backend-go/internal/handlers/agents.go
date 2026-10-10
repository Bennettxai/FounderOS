package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// AgentHandler handles custom AI agent CRUD operations
type AgentHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(pool *pgxpool.Pool, cfg *config.Config) *AgentHandler {
	return &AgentHandler{pool: pool, cfg: cfg}
}

// CreateCustomAgentRequest represents request to create custom agent
type CreateCustomAgentRequest struct {
	Name                 string   `json:"name" binding:"required"`
	DisplayName          string   `json:"display_name" binding:"required"`
	Description          string   `json:"description"`
	Avatar               string   `json:"avatar"`
	SystemPrompt         string   `json:"system_prompt" binding:"required"`
	ModelPreference      string   `json:"model_preference"`
	Temperature          float64  `json:"temperature"`
	MaxTokens            int32    `json:"max_tokens"`
	Capabilities         []string `json:"capabilities"`
	ToolsEnabled         []string `json:"tools_enabled"`
	ContextSources       []string `json:"context_sources"`
	ThinkingEnabled      bool     `json:"thinking_enabled"`
	StreamingEnabled     bool     `json:"streaming_enabled"`
	ApplyPersonalization bool     `json:"apply_personalization"`
	WelcomeMessage       string   `json:"welcome_message"`
	SuggestedPrompts     []string `json:"suggested_prompts"`
	Category             string   `json:"category"`
	IsPublic             bool     `json:"is_public"`
	IsFeatured           bool     `json:"is_featured"`
}

// UpdateCustomAgentRequest represents request to update custom agent
type UpdateCustomAgentRequest struct {
	Name                 *string  `json:"name"`
	DisplayName          *string  `json:"display_name"`
	Description          *string  `json:"description"`
	Avatar               *string  `json:"avatar"`
	SystemPrompt         *string  `json:"system_prompt"`
	ModelPreference      *string  `json:"model_preference"`
	Temperature          *float64 `json:"temperature"`
	MaxTokens            *int32   `json:"max_tokens"`
	Capabilities         []string `json:"capabilities"`
	ToolsEnabled         []string `json:"tools_enabled"`
	ContextSources       []string `json:"context_sources"`
	ThinkingEnabled      *bool    `json:"thinking_enabled"`
	StreamingEnabled     *bool    `json:"streaming_enabled"`
	ApplyPersonalization *bool    `json:"apply_personalization"`
	WelcomeMessage       *string  `json:"welcome_message"`
	SuggestedPrompts     []string `json:"suggested_prompts"`
	Category             *string  `json:"category"`
	IsActive             *bool    `json:"is_active"`
	IsPublic             *bool    `json:"is_public"`
	IsFeatured           *bool    `json:"is_featured"`
}

// CreateAgentFromPresetRequest represents request to create agent from preset
type CreateAgentFromPresetRequest struct {
	Name string `json:"name" binding:"required"`
}

// TestAgentRequest represents request to test an agent prompt
type TestAgentRequest struct {
	SystemPrompt string   `json:"system_prompt"`
	TestMessage  string   `json:"test_message" binding:"required"`
	Model        *string  `json:"model"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
}

// TestAgentResponse represents the sandbox test response
type TestAgentResponse struct {
	Response   string `json:"response"`
	TokensUsed int    `json:"tokens_used"`
	DurationMs int64  `json:"duration_ms"`
	Model      string `json:"model"`
}
