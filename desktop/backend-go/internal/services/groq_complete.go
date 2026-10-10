package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ChatComplete sends a non-streaming chat request
func (s *GroqService) ChatComplete(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	// Convert messages to Groq format
	groqMsgs := make([]GroqMessage, 0, len(messages)+1)

	for _, msg := range messages {
		if msg.Role == "system" {
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
			continue
		}
		groqMsgs = append(groqMsgs, GroqMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if systemPrompt != "" {
		groqMsgs = append([]GroqMessage{{
			Role:    "system",
			Content: systemPrompt,
		}}, groqMsgs...)
	}

	reqBody := GroqRequest{
		Model:     s.model,
		Messages:  groqMsgs,
		MaxTokens: 8192,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq API error: %s - %s", resp.Status, string(body))
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// ChatCompleteWithUsage sends a non-streaming chat request and returns token usage
func (s *GroqService) ChatCompleteWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, *TokenUsage, error) {
	groqMsgs := make([]GroqMessage, 0, len(messages)+1)

	for _, msg := range messages {
		if msg.Role == "system" {
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
			continue
		}
		groqMsgs = append(groqMsgs, GroqMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if systemPrompt != "" {
		groqMsgs = append([]GroqMessage{{
			Role:    "system",
			Content: systemPrompt,
		}}, groqMsgs...)
	}

	reqBody := GroqRequest{
		Model:     s.model,
		Messages:  groqMsgs,
		MaxTokens: 8192,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("groq API error: %s - %s", resp.Status, string(body))
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no response from Groq")
	}

	usage := &TokenUsage{
		InputTokens:  groqResp.Usage.PromptTokens,
		OutputTokens: groqResp.Usage.CompletionTokens,
		TotalTokens:  groqResp.Usage.TotalTokens,
		Model:        s.model,
		Provider:     "groq",
	}

	return groqResp.Choices[0].Message.Content, usage, nil
}
