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
	"strings"
)

// StreamChat sends a chat request and streams the response
func (s *AnthropicService) StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan string, <-chan error) {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		slog.Info("[AnthropicService] StreamChat starting", "model", s.model, "message_count", len(messages))

		// Convert messages to Anthropic format (filter out system messages)
		anthropicMsgs := make([]AnthropicMessage, 0, len(messages))
		for _, msg := range messages {
			if strings.ToLower(msg.Role) == "system" {
				// Combine with system prompt
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
			Stream:    true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.messagesURL(), bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := s.client.Do(req)
		if err != nil {
			slog.Error("[AnthropicService] Request failed", "error", err, "model", s.model)
			errs <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error("[AnthropicService] API error",
				"status", resp.StatusCode, "body", string(body), "model", s.model)
			errs <- fmt.Errorf("anthropic API error (model=%s): %s - %s", s.model, resp.Status, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue // Skip malformed lines
			}

			// Handle different event types
			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					select {
					case chunks <- event.Delta.Text:
					case <-ctx.Done():
						return
					}
				}
			case "message_stop":
				return
			case "error":
				slog.Error("[AnthropicService] Stream error event", "model", s.model)
				errs <- fmt.Errorf("anthropic stream error")
				return
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("[AnthropicService] Scanner error", "error", err, "model", s.model)
			errs <- fmt.Errorf("error reading response: %w", err)
		}
	}()

	return chunks, errs
}

// StreamChatWithUsage streams chat and tracks token usage
func (s *AnthropicService) StreamChatWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) *StreamResult {
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

		// Convert messages to Anthropic format (filter out system messages)
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
			Stream:    true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.messagesURL(), bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := s.client.Do(req)
		if err != nil {
			errs <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message_start":
				inputTokens = event.Message.Usage.InputTokens
			case "content_block_delta":
				if event.Delta.Text != "" {
					select {
					case chunks <- event.Delta.Text:
					case <-ctx.Done():
						return
					}
				}
			case "message_delta":
				outputTokens = event.Usage.OutputTokens
			case "message_stop":
				// Stream complete
			case "error":
				errs <- fmt.Errorf("anthropic stream error")
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("error reading response: %w", err)
		}

		// Set final token usage
		result.SetTokenUsage(&TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
			Model:        s.model,
			Provider:     "anthropic",
		})
	}()

	return result
}

// StreamChatWithThinking streams chat with extended thinking support
func (s *AnthropicService) StreamChatWithThinking(ctx context.Context, messages []ChatMessage, systemPrompt string) *ExtendedThinkingResult {
	result := &ExtendedThinkingResult{
		Chunks:         make(chan string, 100),
		ThinkingChunks: make(chan string, 100),
		Errors:         make(chan error, 1),
	}

	go func() {
		defer close(result.Chunks)
		defer close(result.ThinkingChunks)
		defer close(result.Errors)

		var inputTokens, outputTokens, thinkingTokens int
		var currentBlockType string

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

		// Build request with extended thinking enabled
		reqBody := AnthropicRequest{
			Model:     s.model,
			MaxTokens: s.options.MaxTokens,
			System:    systemPrompt,
			Messages:  anthropicMsgs,
			Stream:    true,
		}

		// Enable extended thinking if supported and enabled in options
		if s.options.ThinkingEnabled && s.SupportsExtendedThinking() {
			budgetTokens := s.options.MaxThinkingTokens
			if budgetTokens < 1024 {
				budgetTokens = 1024
			}
			if budgetTokens > 32768 {
				budgetTokens = 32768
			}
			reqBody.Thinking = &AnthropicThinking{
				Type:         "enabled",
				BudgetTokens: budgetTokens,
			}
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			result.Errors <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.messagesURL(), bytes.NewReader(body))
		if err != nil {
			result.Errors <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := s.client.Do(req)
		if err != nil {
			result.Errors <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			result.Errors <- fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		// Increase buffer size for large thinking blocks
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message_start":
				inputTokens = event.Message.Usage.InputTokens

			case "content_block_start":
				// Track what type of block we're in
				currentBlockType = event.ContentBlock.Type

			case "content_block_delta":
				if currentBlockType == "thinking" {
					// This is thinking content
					if event.Delta.Thinking != "" {
						select {
						case result.ThinkingChunks <- event.Delta.Thinking:
							thinkingTokens++ // Approximate token count
						case <-ctx.Done():
							return
						}
					}
				} else if currentBlockType == "text" {
					// This is regular text content
					if event.Delta.Text != "" {
						select {
						case result.Chunks <- event.Delta.Text:
						case <-ctx.Done():
							return
						}
					}
				}

			case "content_block_stop":
				currentBlockType = ""

			case "message_delta":
				outputTokens = event.Usage.OutputTokens

			case "message_stop":
				// Stream complete

			case "error":
				result.Errors <- fmt.Errorf("anthropic stream error")
				return
			}
		}

		if err := scanner.Err(); err != nil {
			result.Errors <- fmt.Errorf("error reading response: %w", err)
		}

		// Set final token usage
		result.SetTokenUsage(&TokenUsage{
			InputTokens:    inputTokens,
			OutputTokens:   outputTokens,
			ThinkingTokens: thinkingTokens,
			TotalTokens:    inputTokens + outputTokens + thinkingTokens,
			Model:          s.model,
			Provider:       "anthropic",
		})
	}()

	return result
}
