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

// ToolDefinition represents a tool for the LLM
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCallResult represents a tool call from the LLM
type ToolCallResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatWithToolsResponse represents the response from a chat with tools
type ChatWithToolsResponse struct {
	Content   string           `json:"content"`
	ToolCalls []ToolCallResult `json:"tool_calls,omitempty"`
	Usage     *TokenUsage      `json:"usage,omitempty"`
}

// ChatWithTools sends a chat request with tool definitions and returns tool calls if any
func (s *GroqService) ChatWithTools(ctx context.Context, messages []ChatMessage, systemPrompt string, tools []ToolDefinition) (*ChatWithToolsResponse, error) {
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
			Role:    strings.ToLower(msg.Role),
			Content: msg.Content,
		})
	}

	if systemPrompt != "" {
		groqMsgs = append([]GroqMessage{{
			Role:    "system",
			Content: systemPrompt,
		}}, groqMsgs...)
	}

	// Convert tool definitions to Groq format
	groqTools := make([]GroqTool, 0, len(tools))
	for _, t := range tools {
		gt := GroqTool{Type: "function"}
		gt.Function.Name = t.Name
		gt.Function.Description = t.Description
		gt.Function.Parameters = t.Parameters
		groqTools = append(groqTools, gt)
	}

	reqBody := GroqRequest{
		Model:       s.model,
		Messages:    groqMsgs,
		MaxTokens:   s.options.MaxTokens,
		Temperature: s.options.Temperature,
		Stream:      false,
		Tools:       groqTools,
		ToolChoice:  "auto",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("groq API error: %s - %s", resp.Status, string(body))
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Groq")
	}

	result := &ChatWithToolsResponse{
		Content: groqResp.Choices[0].Message.Content,
		Usage: &TokenUsage{
			InputTokens:  groqResp.Usage.PromptTokens,
			OutputTokens: groqResp.Usage.CompletionTokens,
			TotalTokens:  groqResp.Usage.TotalTokens,
			Model:        s.model,
			Provider:     "groq",
		},
	}

	// Extract tool calls if any
	for _, tc := range groqResp.Choices[0].Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCallResult{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return result, nil
}

// ContinueWithToolResults continues the conversation after tool execution.
// OpenAI-compatible APIs require the assistant message that requested the tools
// to appear immediately before the corresponding tool result messages.
func (s *GroqService) ContinueWithToolResults(ctx context.Context, messages []ChatMessage, systemPrompt string, toolCalls []ToolCallResult, toolResults map[string]string) (string, error) {
	groqMsgs := make([]GroqMessage, 0, len(messages)+len(toolResults)+1)

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
			Role:    strings.ToLower(msg.Role),
			Content: msg.Content,
		})
	}

	if systemPrompt != "" {
		groqMsgs = append([]GroqMessage{{
			Role:    "system",
			Content: systemPrompt,
		}}, groqMsgs...)
	}

	assistantToolCalls := make([]GroqToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		assistantToolCalls = append(assistantToolCalls, GroqToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Name,
				Arguments: tc.Arguments,
			},
		})
	}

	if len(assistantToolCalls) > 0 {
		groqMsgs = append(groqMsgs, GroqMessage{
			Role:      "assistant",
			ToolCalls: assistantToolCalls,
		})
	}

	// Add tool results as tool messages in the same order as the calls.
	for _, tc := range toolCalls {
		result, ok := toolResults[tc.ID]
		if !ok {
			result = ""
		}
		groqMsgs = append(groqMsgs, GroqMessage{
			Role:       "tool",
			Content:    result,
			ToolCallID: tc.ID,
		})
	}

	reqBody := GroqRequest{
		Model:       s.model,
		Messages:    groqMsgs,
		MaxTokens:   s.options.MaxTokens,
		Temperature: s.options.Temperature,
		Stream:      false,
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
