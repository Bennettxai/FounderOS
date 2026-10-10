package services

import (
	"context"
	"log/slog"
	"sync"

	"github.com/rhl/businessos-backend/internal/config"
)

// LLMOptions holds configurable parameters for LLM calls
type LLMOptions struct {
	Temperature float64
	MaxTokens   int
	TopP        float64
	// Model override (e.g. from focus mode)
	Model *string
	// Thinking/COT options
	ThinkingEnabled     bool
	ThinkingInstruction string
	MaxThinkingTokens   int
	ReasoningTemplateID string
}

// DefaultLLMOptions returns sensible defaults
func DefaultLLMOptions() LLMOptions {
	return LLMOptions{
		Temperature:       0.7,
		MaxTokens:         8192,
		TopP:              0.9,
		ThinkingEnabled:   false,
		MaxThinkingTokens: 4096,
	}
}

// ThinkingChunk represents a chunk of thinking/reasoning output
type ThinkingChunk struct {
	Step      int    `json:"step"`
	Content   string `json:"content"`
	Type      string `json:"type"` // analysis, planning, reflection, tool_use, reasoning, evaluation
	Completed bool   `json:"completed"`
}

// StreamResultWithThinking extends StreamResult with thinking support
type StreamResultWithThinking struct {
	*StreamResult
	ThinkingChunks <-chan ThinkingChunk
}

// TokenUsage tracks token consumption for LLM calls
type TokenUsage struct {
	InputTokens    int    `json:"input_tokens"`
	OutputTokens   int    `json:"output_tokens"`
	ThinkingTokens int    `json:"thinking_tokens,omitempty"` // Tokens used for extended thinking/COT
	TotalTokens    int    `json:"total_tokens"`
	Model          string `json:"model"`
	Provider       string `json:"provider"`
}

// AgentTrace tracks the flow of an agent request
type AgentTrace struct {
	AgentName   string       `json:"agent_name"`
	DelegatedTo string       `json:"delegated_to,omitempty"`
	TokenUsage  *TokenUsage  `json:"token_usage,omitempty"`
	SubTraces   []AgentTrace `json:"sub_traces,omitempty"`
	StartTime   int64        `json:"start_time"`
	EndTime     int64        `json:"end_time,omitempty"`
	DurationMs  int64        `json:"duration_ms,omitempty"`
}

// StreamResult wraps streaming results with token usage
type StreamResult struct {
	Chunks     <-chan string
	Errors     <-chan error
	TokenUsage *TokenUsage
	mu         sync.Mutex
}

// SetTokenUsage safely sets token usage (called when stream completes)
func (sr *StreamResult) SetTokenUsage(usage *TokenUsage) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.TokenUsage = usage
}

// GetTokenUsage safely gets token usage
func (sr *StreamResult) GetTokenUsage() *TokenUsage {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.TokenUsage
}

// ExtendedThinkingResult wraps streaming results with separate thinking channel
type ExtendedThinkingResult struct {
	Chunks         chan string // Regular response content
	ThinkingChunks chan string // Extended thinking content
	Errors         chan error  // Errors
	TokenUsage     *TokenUsage // Final token usage
	mu             sync.Mutex
}

// SetTokenUsage safely sets token usage (called when stream completes)
func (etr *ExtendedThinkingResult) SetTokenUsage(usage *TokenUsage) {
	etr.mu.Lock()
	defer etr.mu.Unlock()
	etr.TokenUsage = usage
}

// GetTokenUsage safely gets token usage
func (etr *ExtendedThinkingResult) GetTokenUsage() *TokenUsage {
	etr.mu.Lock()
	defer etr.mu.Unlock()
	return etr.TokenUsage
}

// LLMService interface for language model services
type LLMService interface {
	StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan string, <-chan error)
	StreamChatWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) *StreamResult
	ChatComplete(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error)
	ChatCompleteWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, *TokenUsage, error)
	HealthCheck(ctx context.Context) bool
	GetModel() string
	GetProvider() string
	SetOptions(opts LLMOptions)
	GetOptions() LLMOptions
}

// ToolCallingService is implemented by LLM providers that support native tool/function calling.
// Providers that don't support this will fall back to prompt-based tool descriptions.
type ToolCallingService interface {
	LLMService
	ChatWithTools(ctx context.Context, messages []ChatMessage, systemPrompt string, tools []ToolDefinition) (*ChatWithToolsResponse, error)
	ContinueWithToolResults(ctx context.Context, messages []ChatMessage, systemPrompt string, toolCalls []ToolCallResult, toolResults map[string]string) (string, error)
}

// AsToolCallingService checks if an LLMService supports tool calling and returns the
// ToolCallingService interface if so. Returns (nil, false) for providers without tool support.
func AsToolCallingService(llm LLMService) (ToolCallingService, bool) {
	tc, ok := llm.(ToolCallingService)
	return tc, ok
}

// ExtendedThinkingService interface for providers that support native extended thinking
type ExtendedThinkingService interface {
	LLMService
	// StreamChatWithThinking streams chat with extended thinking support
	// Returns separate channels for thinking content and response content
	StreamChatWithThinking(ctx context.Context, messages []ChatMessage, systemPrompt string) *ExtendedThinkingResult
	// SupportsExtendedThinking returns true if the current model supports extended thinking
	SupportsExtendedThinking() bool
}

// NewLLMService creates the appropriate LLM service based on configuration
func NewLLMService(cfg *config.Config, model string) LLMService {
	provider := cfg.GetActiveProvider()
	slog.Debug("creating LLM service", "provider", provider, "model", model)
	switch provider {
	case "ollama_cloud":
		return NewOllamaCloudService(cfg, model)
	case "anthropic":
		return NewAnthropicService(cfg, model)
	case "groq":
		return NewGroqService(cfg, model)
	default:
		// Default to local Ollama
		return NewOllamaService(cfg, model)
	}
}

// NewLLMServiceWithThinking creates an LLM service that may support extended thinking
// Returns the service and a boolean indicating if extended thinking is supported
func NewLLMServiceWithThinking(cfg *config.Config, model string) (LLMService, bool) {
	provider := cfg.GetActiveProvider()
	slog.Debug("creating thinking-aware LLM service", "provider", provider, "model", model)

	switch provider {
	case "anthropic":
		service := NewAnthropicService(cfg, model)
		return service, service.SupportsExtendedThinking()
	case "ollama_cloud":
		return NewOllamaCloudService(cfg, model), false
	case "groq":
		return NewGroqService(cfg, model), false
	default:
		return NewOllamaService(cfg, model), false
	}
}

// AsExtendedThinkingService attempts to cast an LLMService to ExtendedThinkingService
func AsExtendedThinkingService(service LLMService) (ExtendedThinkingService, bool) {
	ets, ok := service.(ExtendedThinkingService)
	if !ok {
		return nil, false
	}
	// Also check if the model actually supports it
	if !ets.SupportsExtendedThinking() {
		return nil, false
	}
	return ets, true
}

// StreamWithNativeThinking streams a chat response using native extended thinking if available
// Otherwise falls back to standard streaming with prompt-based thinking
func StreamWithNativeThinking(ctx context.Context, service LLMService, messages []ChatMessage, systemPrompt string, opts LLMOptions) (<-chan string, <-chan string, <-chan error) {
	contentChan := make(chan string, 100)
	thinkingChan := make(chan string, 100)
	errChan := make(chan error, 1)

	// Check if native extended thinking is available
	if opts.ThinkingEnabled {
		if ets, ok := AsExtendedThinkingService(service); ok {
			// Use native extended thinking
			result := ets.StreamChatWithThinking(ctx, messages, systemPrompt)

			go func() {
				defer close(contentChan)
				defer close(thinkingChan)
				defer close(errChan)

				// Forward thinking chunks
				go func() {
					for chunk := range result.ThinkingChunks {
						select {
						case thinkingChan <- chunk:
						case <-ctx.Done():
							return
						}
					}
				}()

				// Forward content chunks
				for chunk := range result.Chunks {
					select {
					case contentChan <- chunk:
					case <-ctx.Done():
						return
					}
				}

				// Forward errors
				for err := range result.Errors {
					select {
					case errChan <- err:
					case <-ctx.Done():
						return
					}
				}
			}()

			return contentChan, thinkingChan, errChan
		}
	}

	// Fall back to standard streaming (prompt-based thinking handled by caller)
	chunks, errs := service.StreamChat(ctx, messages, systemPrompt)

	go func() {
		defer close(contentChan)
		defer close(thinkingChan) // No thinking content for non-native
		defer close(errChan)

		for chunk := range chunks {
			select {
			case contentChan <- chunk:
			case <-ctx.Done():
				return
			}
		}

		for err := range errs {
			select {
			case errChan <- err:
			case <-ctx.Done():
				return
			}
		}
	}()

	return contentChan, thinkingChan, errChan
}
