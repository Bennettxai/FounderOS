package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatComplete sends a non-streaming chat request
func (s *AnthropicService) ChatComplete(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	// Convert messages to Anthropic format
	anthropicMsgs := make([]AnthropicMessage, 0, len(messages))
	for _, msg := range messages {
		if strings.ToLower(msg.Role) == "system" {
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
			continue
		}
		anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
			Role:    strings.ToLower(msg.Role),
			Content: msg.Content,
		})
	}

	maxTokens := s.options.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	reqBody := AnthropicRequest{
		Model:     s.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  anthropicMsgs,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.messagesURL(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from content blocks
	var result strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	return result.String(), nil
}

// ChatCompleteWithUsage sends a non-streaming chat request and returns token usage
func (s *AnthropicService) ChatCompleteWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, *TokenUsage, error) {
	// Convert messages to Anthropic format
	anthropicMsgs := make([]AnthropicMessage, 0, len(messages))
	for _, msg := range messages {
		if strings.ToLower(msg.Role) == "system" {
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
			continue
		}
		anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
			Role:    strings.ToLower(msg.Role),
			Content: msg.Content,
		})
	}

	maxTokens := s.options.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	reqBody := AnthropicRequest{
		Model:     s.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  anthropicMsgs,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.messagesURL(), bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from content blocks
	var resultText strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			resultText.WriteString(block.Text)
		}
	}

	usage := &TokenUsage{
		InputTokens:  anthropicResp.Usage.InputTokens,
		OutputTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:  anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		Model:        s.model,
		Provider:     "anthropic",
	}

	return resultText.String(), usage, nil
}
