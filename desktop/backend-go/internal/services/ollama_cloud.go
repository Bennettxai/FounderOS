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

// OllamaCloudService handles LLM inference via Ollama Cloud native API
type OllamaCloudService struct {
	apiKey  string
	model   string
	client  *http.Client
	options LLMOptions
}

const ollamaCloudBaseURL = "https://api.ollama.com/api"

// OllamaCloudMessage represents a message in the native Ollama format
type OllamaCloudMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaCloudRequest represents a request to the native Ollama Cloud API
type OllamaCloudRequest struct {
	Model    string               `json:"model"`
	Messages []OllamaCloudMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
	Options  *OllamaRequestOpts   `json:"options,omitempty"`
}

// OllamaRequestOpts holds model generation options
type OllamaRequestOpts struct {
	NumPredict int `json:"num_predict,omitempty"`
}

// OllamaCloudResponse represents a native Ollama non-streaming/streaming response
type OllamaCloudResponse struct {
	Model           string             `json:"model"`
	CreatedAt       string             `json:"created_at"`
	Message         OllamaCloudMessage `json:"message"`
	Done            bool               `json:"done"`
	Error           string             `json:"error,omitempty"`
	TotalDuration   int64              `json:"total_duration,omitempty"`
	EvalCount       int                `json:"eval_count,omitempty"`
	PromptEvalCount int                `json:"prompt_eval_count,omitempty"`
}

// NewOllamaCloudService creates a new Ollama Cloud service instance
func NewOllamaCloudService(cfg *config.Config, model string) *OllamaCloudService {
	if model == "" {
		model = cfg.OllamaCloudModel
	}
	if model == "" {
		model = "llama3.2"
	}

	return &OllamaCloudService{
		apiKey: cfg.OllamaCloudAPIKey,
		model:  model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		options: DefaultLLMOptions(),
	}
}

// SetOptions sets the LLM options for this service
func (s *OllamaCloudService) SetOptions(opts LLMOptions) {
	s.options = opts
}

// GetOptions returns the current LLM options
func (s *OllamaCloudService) GetOptions() LLMOptions {
	return s.options
}

func (s *OllamaCloudService) buildMessages(messages []ChatMessage, systemPrompt string) []OllamaCloudMessage {
	cloudMsgs := make([]OllamaCloudMessage, 0, len(messages)+1)

	for _, msg := range messages {
		if msg.Role == "system" {
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
			continue
		}
		cloudMsgs = append(cloudMsgs, OllamaCloudMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if systemPrompt != "" {
		cloudMsgs = append([]OllamaCloudMessage{{
			Role:    "system",
			Content: systemPrompt,
		}}, cloudMsgs...)
	}

	return cloudMsgs
}

func (s *OllamaCloudService) doRequest(ctx context.Context, cloudMsgs []OllamaCloudMessage, stream bool) (*http.Response, error) {
	reqBody := OllamaCloudRequest{
		Model:    s.model,
		Messages: cloudMsgs,
		Stream:   stream,
		Options: &OllamaRequestOpts{
			NumPredict: 8192,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaCloudBaseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	slog.Info("[OllamaCloudService] Sending request", "model", s.model, "stream", stream)
	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("[OllamaCloudService] Request failed", "error", err, "model", s.model)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		slog.Error("[OllamaCloudService] API error",
			"status", resp.StatusCode, "body", string(errBody), "model", s.model)
		return nil, fmt.Errorf("ollama cloud API error (model=%s): %s - %s", s.model, resp.Status, string(errBody))
	}

	return resp, nil
}

// StreamChat sends a chat request and streams the response
func (s *OllamaCloudService) StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan string, <-chan error) {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(errs)
		defer close(chunks)

		cloudMsgs := s.buildMessages(messages, systemPrompt)

		resp, err := s.doRequest(ctx, cloudMsgs, true)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()

		// Native Ollama streaming: NDJSON lines (one JSON object per line)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large responses
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var streamResp OllamaCloudResponse
			if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
				slog.Warn("ollama cloud: failed to parse stream line", "error", err)
				continue
			}

			if streamResp.Error != "" {
				errs <- fmt.Errorf("ollama cloud stream error: %s", streamResp.Error)
				return
			}

			if streamResp.Done {
				return
			}

			if streamResp.Message.Content != "" {
				select {
				case chunks <- streamResp.Message.Content:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("error reading response: %w", err)
		}
	}()

	return chunks, errs
}

// ChatComplete sends a non-streaming chat request
func (s *OllamaCloudService) ChatComplete(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	cloudMsgs := s.buildMessages(messages, systemPrompt)

	resp, err := s.doRequest(ctx, cloudMsgs, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cloudResp OllamaCloudResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloudResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if cloudResp.Message.Content == "" && !cloudResp.Done {
		return "", fmt.Errorf("no response from Ollama Cloud")
	}

	return cloudResp.Message.Content, nil
}

// HealthCheck checks if Ollama Cloud API is available
func (s *OllamaCloudService) HealthCheck(ctx context.Context) bool {
	return s.apiKey != ""
}

// GetModel returns the model name
func (s *OllamaCloudService) GetModel() string {
	return s.model
}

// GetProvider returns the provider name
func (s *OllamaCloudService) GetProvider() string {
	return "ollama_cloud"
}

// StreamChatWithUsage streams chat and tracks token usage
func (s *OllamaCloudService) StreamChatWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) *StreamResult {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)
	result := &StreamResult{
		Chunks: chunks,
		Errors: errs,
	}

	go func() {
		defer close(errs)
		defer close(chunks)

		cloudMsgs := s.buildMessages(messages, systemPrompt)

		resp, err := s.doRequest(ctx, cloudMsgs, true)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()

		var evalCount, promptEvalCount int

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var streamResp OllamaCloudResponse
			if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
				slog.Warn("ollama cloud: failed to parse stream line", "error", err)
				continue
			}

			if streamResp.Error != "" {
				errs <- fmt.Errorf("ollama cloud stream error: %s", streamResp.Error)
				return
			}

			if streamResp.Done {
				evalCount = streamResp.EvalCount
				promptEvalCount = streamResp.PromptEvalCount
				return
			}

			if streamResp.Message.Content != "" {
				select {
				case chunks <- streamResp.Message.Content:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("error reading response: %w", err)
		}

		result.SetTokenUsage(&TokenUsage{
			InputTokens:  promptEvalCount,
			OutputTokens: evalCount,
			TotalTokens:  promptEvalCount + evalCount,
			Model:        s.model,
			Provider:     "ollama_cloud",
		})
	}()

	return result
}

// ChatCompleteWithUsage sends a non-streaming chat request and returns token usage
func (s *OllamaCloudService) ChatCompleteWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, *TokenUsage, error) {
	cloudMsgs := s.buildMessages(messages, systemPrompt)

	resp, err := s.doRequest(ctx, cloudMsgs, false)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var cloudResp OllamaCloudResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloudResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if cloudResp.Message.Content == "" && !cloudResp.Done {
		return "", nil, fmt.Errorf("no response from Ollama Cloud")
	}

	usage := &TokenUsage{
		InputTokens:  cloudResp.PromptEvalCount,
		OutputTokens: cloudResp.EvalCount,
		TotalTokens:  cloudResp.PromptEvalCount + cloudResp.EvalCount,
		Model:        s.model,
		Provider:     "ollama_cloud",
	}

	return cloudResp.Message.Content, usage, nil
}
