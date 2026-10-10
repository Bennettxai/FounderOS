package services

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rhl/businessos-backend/internal/config"
)

// sanitizeUTF8 ensures the string contains only valid UTF-8 characters
// Invalid bytes are replaced with empty string to avoid encoding issues
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Replace invalid UTF-8 sequences
	var result strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid byte, skip it
			i++
			continue
		}
		result.WriteRune(r)
		i += size
	}
	return result.String()
}

// GroqService handles LLM inference via Groq API
type GroqService struct {
	apiKey  string
	model   string
	client  *http.Client
	options LLMOptions
}

// GroqToolCall represents a tool call from the LLM
type GroqToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// GroqTool represents a tool definition for the LLM
type GroqTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// GroqMessage represents a message in the Groq format
type GroqMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []GroqToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// GroqRequest represents a request to the Groq API
type GroqRequest struct {
	Model       string        `json:"model"`
	Messages    []GroqMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
	Tools       []GroqTool    `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"`
}

// GroqResponse represents a non-streaming response from Groq
type GroqResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Message struct {
			Role      string         `json:"role"`
			Content   string         `json:"content"`
			ToolCalls []GroqToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GroqStreamResponse represents a streaming chunk from Groq
type GroqStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// NewGroqService creates a new Groq service instance
func NewGroqService(cfg *config.Config, model string) *GroqService {
	if model == "" {
		model = cfg.GroqModel
	}
	if model == "" {
		model = "llama-3.3-70b-versatile" // Default Groq model
	}

	return &GroqService{
		apiKey: cfg.GroqAPIKey,
		model:  model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		options: DefaultLLMOptions(),
	}
}

// SetOptions sets the LLM options for this service
func (s *GroqService) SetOptions(opts LLMOptions) {
	s.options = opts
}

// GetOptions returns the current LLM options
func (s *GroqService) GetOptions() LLMOptions {
	return s.options
}

// HealthCheck checks if Groq API is available
func (s *GroqService) HealthCheck(ctx context.Context) bool {
	return s.apiKey != ""
}

// GetModel returns the model name
func (s *GroqService) GetModel() string {
	return s.model
}

// GetProvider returns the provider name
func (s *GroqService) GetProvider() string {
	return "groq"
}
