package services

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/config"
)

// AnthropicService handles LLM inference via Anthropic's Claude API
type AnthropicService struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
	options LLMOptions
}

// AnthropicMessage represents a message in the Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicThinking represents extended thinking configuration
type AnthropicThinking struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // Max tokens for thinking (1024-32768)
}

// AnthropicRequest represents a request to the Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []AnthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
	Thinking  *AnthropicThinking `json:"thinking,omitempty"` // Extended thinking support
}

// AnthropicContentBlock represents a content block in the response
type AnthropicContentBlock struct {
	Type     string `json:"type"`               // "text" or "thinking"
	Text     string `json:"text,omitempty"`     // For text blocks
	Thinking string `json:"thinking,omitempty"` // For thinking blocks
}

// AnthropicResponse represents a non-streaming response from Anthropic
type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []AnthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		// Extended thinking usage
		CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	} `json:"usage"`
}

// AnthropicStreamEvent represents a streaming event from Anthropic
type AnthropicStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type     string `json:"type,omitempty"`
		Text     string `json:"text,omitempty"`
		Thinking string `json:"thinking,omitempty"` // For thinking_delta events
	} `json:"delta,omitempty"`
	ContentBlock struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		Thinking string `json:"thinking,omitempty"`
	} `json:"content_block,omitempty"`
	Message struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

// NewAnthropicService creates a new Anthropic service instance
func NewAnthropicService(cfg *config.Config, model string) *AnthropicService {
	if model == "" {
		model = cfg.AnthropicModel
	}
	if model == "" {
		model = "claude-sonnet-4-20250514" // Default to Claude Sonnet
	}

	baseURL := strings.TrimRight(cfg.AnthropicBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &AnthropicService{
		apiKey:  cfg.AnthropicAPIKey,
		model:   model,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		options: DefaultLLMOptions(),
	}
}

// SetOptions sets the LLM options for this service
func (s *AnthropicService) SetOptions(opts LLMOptions) {
	s.options = opts
}

// GetOptions returns the current LLM options
func (s *AnthropicService) GetOptions() LLMOptions {
	return s.options
}

// GetBaseURL returns the configured base URL for the API
func (s *AnthropicService) GetBaseURL() string {
	return s.baseURL
}

// messagesURL returns the full messages endpoint URL
func (s *AnthropicService) messagesURL() string {
	return s.baseURL + "/v1/messages"
}

// HealthCheck checks if Anthropic API is available
func (s *AnthropicService) HealthCheck(ctx context.Context) bool {
	// Simple check - just verify API key is set
	return s.apiKey != ""
}

// GetModel returns the model name
func (s *AnthropicService) GetModel() string {
	return s.model
}

// GetProvider returns the provider name
func (s *AnthropicService) GetProvider() string {
	return "anthropic"
}

// SupportsExtendedThinking returns true if the current model supports extended thinking
func (s *AnthropicService) SupportsExtendedThinking() bool {
	// Extended thinking is only available on the real Anthropic API.
	// Non-Anthropic endpoints (e.g. Ollama Cloud) may serve models with
	// "claude" in the name but do not support the extended thinking API.
	if s.baseURL != "" && s.baseURL != "https://api.anthropic.com" {
		return false
	}

	// Extended thinking is supported on Claude 3.5 Sonnet and Claude 3 Opus and newer
	supportedModels := []string{
		"claude-sonnet-4",
		"claude-opus-4",
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"claude-3-opus",
	}
	for _, supported := range supportedModels {
		if strings.Contains(s.model, supported) {
			return true
		}
	}
	return false
}
