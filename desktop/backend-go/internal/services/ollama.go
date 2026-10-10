package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/config"
)

// OllamaService handles LLM inference via Ollama
type OllamaService struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	options LLMOptions
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaOptions represents Ollama-specific options
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // max_tokens in Ollama
	TopP        float64 `json:"top_p,omitempty"`
}

// ChatRequest represents a request to the Ollama chat API
type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  *OllamaOptions `json:"options,omitempty"`
}

// ChatResponse represents a response from Ollama
type ChatResponse struct {
	Model           string      `json:"model"`
	Message         ChatMessage `json:"message"`
	Done            bool        `json:"done"`
	PromptEvalCount int         `json:"prompt_eval_count,omitempty"`
	EvalCount       int         `json:"eval_count,omitempty"`
}

// NewOllamaService creates a new Ollama service instance for LOCAL inference only
// For cloud providers, use NewLLMService which selects the appropriate provider
func NewOllamaService(cfg *config.Config, model string) *OllamaService {
	// Always use local Ollama URL - cloud providers should use NewLLMService instead
	baseURL := cfg.OllamaLocalURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	slog.Info("[OllamaService] Creating service", "model", model, "default_model", cfg.DefaultModel)

	if model == "" {
		model = cfg.DefaultModel
		slog.Info("[OllamaService] Using default model", "value", model)
	}

	// Note: Keep -cloud suffix for remote models served through local Ollama
	// e.g., qwen3-coder:480b-cloud routes to ollama.com

	return &OllamaService{
		baseURL: baseURL,
		apiKey:  "", // Local Ollama doesn't need API key
		model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		options: DefaultLLMOptions(),
	}
}

// SetOptions sets the LLM options for this service
func (s *OllamaService) SetOptions(opts LLMOptions) {
	s.options = opts
}

// GetOptions returns the current LLM options
func (s *OllamaService) GetOptions() LLMOptions {
	return s.options
}

// StreamChat sends a chat request and streams the response
func (s *OllamaService) StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan string, <-chan error) {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)

	slog.Info("[OllamaService] StreamChat starting",
		"model", s.model, "base_url", s.baseURL,
		"message_count", len(messages),
		"has_system_prompt", systemPrompt != "",
		"temperature", s.options.Temperature,
		"max_tokens", s.options.MaxTokens)

	go func() {
		defer close(chunks)
		defer close(errs)

		// Prepend system message if provided
		if systemPrompt != "" {
			messages = append([]ChatMessage{{Role: "system", Content: systemPrompt}}, messages...)
		}

		reqBody := ChatRequest{
			Model:    s.model,
			Messages: messages,
			Stream:   true,
			Options: &OllamaOptions{
				Temperature: s.options.Temperature,
				NumPredict:  s.options.MaxTokens,
				TopP:        s.options.TopP,
			},
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if s.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.apiKey)
		}

		slog.Debug("[OllamaService] Sending request", "url", s.baseURL+"/api/chat")
		resp, err := s.client.Do(req)
		if err != nil {
			slog.Error("[OllamaService] Request failed", "error", err, "model", s.model, "url", s.baseURL)
			errs <- fmt.Errorf("failed to send request to %s: %w", s.baseURL, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error("[OllamaService] API error response",
				"status", resp.StatusCode, "body", string(body),
				"model", s.model, "url", s.baseURL)
			errs <- fmt.Errorf("ollama API error (model=%s): %s - %s", s.model, resp.Status, string(body))
			return
		}

		slog.Debug("[OllamaService] Got 200 OK, reading stream", "model", s.model)
		scanner := bufio.NewScanner(resp.Body)
		chunkCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var chatResp ChatResponse
			if err := json.Unmarshal([]byte(line), &chatResp); err != nil {
				slog.Warn("[OllamaService] Malformed line", "error", err, "line_preview", line[:min(len(line), 100)])
				continue
			}

			if chatResp.Message.Content != "" {
				chunkCount++
				select {
				case chunks <- chatResp.Message.Content:
				case <-ctx.Done():
					slog.Warn("[OllamaService] Context cancelled during streaming",
						"chunks_sent", chunkCount, "model", s.model)
					return
				}
			}

			if chatResp.Done {
				slog.Info("[OllamaService] Stream complete",
					"chunks_sent", chunkCount, "model", s.model)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("[OllamaService] Scanner error", "error", err,
				"chunks_sent", chunkCount, "model", s.model)
			errs <- fmt.Errorf("error reading response: %w", err)
		} else if chunkCount == 0 {
			slog.Warn("[OllamaService] Stream ended with zero chunks",
				"model", s.model, "url", s.baseURL)
		}
	}()

	return chunks, errs
}

// ChatComplete sends a non-streaming chat request
func (s *OllamaService) ChatComplete(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	// Prepend system message if provided
	if systemPrompt != "" {
		messages = append([]ChatMessage{{Role: "system", Content: systemPrompt}}, messages...)
	}

	reqBody := ChatRequest{
		Model:    s.model,
		Messages: messages,
		Stream:   false,
		Options: &OllamaOptions{
			Temperature: s.options.Temperature,
			NumPredict:  s.options.MaxTokens,
			TopP:        s.options.TopP,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// HealthCheck checks if Ollama service is available
func (s *OllamaService) HealthCheck(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// System prompts for different contexts
var SystemPrompts = map[string]string{
	"default": `You are an expert business operations assistant in Business OS - an internal command center for managing businesses, projects, and strategic initiatives.

## Your Role
You are a knowledgeable advisor who provides comprehensive, actionable guidance on:
- Business operations and process optimization
- Project management and task prioritization
- Strategic planning and decision-making
- Documentation creation (proposals, frameworks, SOPs, reports)
- Data analysis and insights generation
- Team coordination and resource allocation

## Response Guidelines
1. **Be Thorough**: Provide detailed, well-structured responses. Don't give surface-level answers - dig deep and explain the reasoning.
2. **Be Actionable**: Include specific next steps, recommendations, or frameworks the user can immediately apply.
3. **Be Structured**: Use clear headings, bullet points, and numbered lists for complex information.
4. **Be Context-Aware**: Reference the user's business context when available to tailor advice.
5. **Create Artifacts**: When asked to create documents, proposals, or frameworks, provide complete, polished drafts that are ready to use.

## Output Formats
- For questions: Provide comprehensive answers with examples and explanations
- For analysis: Include observations, insights, and recommendations
- For documents: Create complete, professional-quality content with proper structure
- For planning: Provide step-by-step plans with clear milestones and success criteria

Always think from a business owner's perspective - what would actually help move the needle?`,

	"daily_planning": `You are an executive daily planning assistant specializing in productivity and prioritization.

## Your Role
Help the user optimize their day for maximum impact by:
- Reviewing and ruthlessly prioritizing tasks based on strategic importance
- Identifying potential blockers before they become problems
- Time-blocking and energy management recommendations
- Connecting daily work to quarterly/annual goals

## Response Guidelines
1. Start with the 2-3 highest leverage activities for the day
2. Identify tasks that can be delegated, deferred, or deleted
3. Suggest specific time blocks with buffer time included
4. Flag any deadline risks or dependency issues
5. End with a clear "if you only do one thing today, do X" recommendation

Be direct, practical, and focused on outcomes over activity.`,

	"document_creation": `You are an expert business writer creating professional documents.

## Your Role
Create polished, comprehensive business documents including:
- Business proposals and pitches
- Strategic frameworks and playbooks
- Standard operating procedures (SOPs)
- Reports and executive summaries
- Meeting agendas and action plans

## Response Guidelines
1. Use professional formatting with clear structure
2. Include executive summaries for longer documents
3. Make content specific and actionable, not generic
4. Use data and examples where relevant
5. Consider the audience and adjust tone appropriately

Produce documents that are ready to share externally or present to stakeholders.`,
}

// GetSystemPrompt returns a system prompt by name
func GetSystemPrompt(name string) string {
	if prompt, ok := SystemPrompts[name]; ok {
		return prompt
	}
	return SystemPrompts["default"]
}

// GetModel returns the model name
func (s *OllamaService) GetModel() string {
	return s.model
}

// GetProvider returns the provider name
func (s *OllamaService) GetProvider() string {
	return "ollama_local"
}

// StreamChatWithUsage streams chat and tracks token usage
func (s *OllamaService) StreamChatWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) *StreamResult {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)
	result := &StreamResult{
		Chunks: chunks,
		Errors: errs,
	}

	go func() {
		defer close(chunks)
		defer close(errs)

		var inputTokens, outputTokens int

		// Prepend system message if provided
		if systemPrompt != "" {
			messages = append([]ChatMessage{{Role: "system", Content: systemPrompt}}, messages...)
		}

		reqBody := ChatRequest{
			Model:    s.model,
			Messages: messages,
			Stream:   true,
			Options: &OllamaOptions{
				Temperature: s.options.Temperature,
				NumPredict:  s.options.MaxTokens,
				TopP:        s.options.TopP,
			},
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if s.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.apiKey)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			errs <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var chatResp ChatResponse
			if err := json.Unmarshal([]byte(line), &chatResp); err != nil {
				continue
			}

			if chatResp.Message.Content != "" {
				select {
				case chunks <- chatResp.Message.Content:
				case <-ctx.Done():
					return
				}
			}

			// Capture token counts from final response
			if chatResp.Done {
				inputTokens = chatResp.PromptEvalCount
				outputTokens = chatResp.EvalCount
				break
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("error reading response: %w", err)
		}

		result.SetTokenUsage(&TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
			Model:        s.model,
			Provider:     "ollama_local",
		})
	}()

	return result
}

// ChatCompleteWithUsage sends a non-streaming chat request and returns token usage
func (s *OllamaService) ChatCompleteWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, *TokenUsage, error) {
	// Prepend system message if provided
	if systemPrompt != "" {
		messages = append([]ChatMessage{{Role: "system", Content: systemPrompt}}, messages...)
	}

	reqBody := ChatRequest{
		Model:    s.model,
		Messages: messages,
		Stream:   false,
		Options: &OllamaOptions{
			Temperature: s.options.Temperature,
			NumPredict:  s.options.MaxTokens,
			TopP:        s.options.TopP,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	usage := &TokenUsage{
		InputTokens:  chatResp.PromptEvalCount,
		OutputTokens: chatResp.EvalCount,
		TotalTokens:  chatResp.PromptEvalCount + chatResp.EvalCount,
		Model:        s.model,
		Provider:     "ollama_local",
	}

	return chatResp.Message.Content, usage, nil
}
