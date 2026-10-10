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
func (s *GroqService) StreamChat(ctx context.Context, messages []ChatMessage, systemPrompt string) (<-chan string, <-chan error) {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		// Convert messages to Groq format
		groqMsgs := make([]GroqMessage, 0, len(messages)+1)

		for _, msg := range messages {
			role := strings.ToLower(msg.Role)
			slog.Debug("groq message role", "original", msg.Role, "normalized", role)
			if role == "system" {
				// Combine with existing system prompt
				if systemPrompt != "" {
					systemPrompt = systemPrompt + "\n\n" + msg.Content
				} else {
					systemPrompt = msg.Content
				}
				continue
			}
			// Ensure valid role for Groq API
			if role != "user" && role != "assistant" {
				slog.Debug("groq invalid role, defaulting to user", "role", role)
				role = "user" // Default to user for unknown roles
			}
			groqMsgs = append(groqMsgs, GroqMessage{
				Role:    role,
				Content: msg.Content,
			})
		}

		// Add system message first if provided/combined
		if systemPrompt != "" {
			groqMsgs = append([]GroqMessage{{
				Role:    "system",
				Content: systemPrompt,
			}}, groqMsgs...)
		}
		slog.Debug("groq sending messages to API", "count", len(groqMsgs))

		maxTokens := s.options.MaxTokens
		if maxTokens < 1000 {
			maxTokens = 8192 // Default to 8192 if not set properly
		}
		slog.Debug("groq streaming request", "max_tokens", maxTokens)

		reqBody := GroqRequest{
			Model:     s.model,
			Messages:  groqMsgs,
			MaxTokens: maxTokens,
			Stream:    true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)

		slog.Info("[GroqService] Sending stream request", "model", s.model, "message_count", len(groqMsgs))
		resp, err := s.client.Do(req)
		if err != nil {
			slog.Error("[GroqService] Request failed", "error", err, "model", s.model)
			errs <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error("[GroqService] API error",
				"status", resp.StatusCode, "body", string(body), "model", s.model)
			errs <- fmt.Errorf("groq API error (model=%s): %s - %s", s.model, resp.Status, string(body))
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
				slog.Debug("groq stream completed")
				return
			}

			var streamResp GroqStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue // Skip malformed lines
			}

			if len(streamResp.Choices) > 0 {
				// Log finish_reason when set (indicates why stream ended)
				if streamResp.Choices[0].FinishReason != "" {
					slog.Debug("groq stream finish", "reason", streamResp.Choices[0].FinishReason)
				}

				if streamResp.Choices[0].Delta.Content != "" {
					content := sanitizeUTF8(streamResp.Choices[0].Delta.Content)
					select {
					case chunks <- content:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("groq scanner error", "error", err)
			errs <- fmt.Errorf("error reading response: %w", err)
		}
	}()

	return chunks, errs
}

// StreamChatWithUsage streams chat and tracks token usage
func (s *GroqService) StreamChatWithUsage(ctx context.Context, messages []ChatMessage, systemPrompt string) *StreamResult {
	chunks := make(chan string, 100)
	errs := make(chan error, 1)
	result := &StreamResult{
		Chunks: chunks,
		Errors: errs,
	}

	go func() {
		defer close(chunks)
		defer close(errs)

		groqMsgs := make([]GroqMessage, 0, len(messages)+1)
		for _, msg := range messages {
			role := strings.ToLower(msg.Role)
			if role == "system" {
				if systemPrompt != "" {
					systemPrompt = systemPrompt + "\n\n" + msg.Content
				} else {
					systemPrompt = msg.Content
				}
				continue
			}
			if role != "user" && role != "assistant" {
				role = "user"
			}
			groqMsgs = append(groqMsgs, GroqMessage{
				Role:    role,
				Content: msg.Content,
			})
		}

		if systemPrompt != "" {
			groqMsgs = append([]GroqMessage{{
				Role:    "system",
				Content: systemPrompt,
			}}, groqMsgs...)
		}

		maxTokens := s.options.MaxTokens
		if maxTokens < 1000 {
			maxTokens = 8192
		}

		reqBody := GroqRequest{
			Model:     s.model,
			Messages:  groqMsgs,
			MaxTokens: maxTokens,
			Stream:    true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errs <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)

		resp, err := s.client.Do(req)
		if err != nil {
			errs <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- fmt.Errorf("groq API error: %s - %s", resp.Status, string(body))
			return
		}

		var estimatedTokens int
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

			var streamResp GroqStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				content := sanitizeUTF8(streamResp.Choices[0].Delta.Content)
				estimatedTokens += len(content) / 4
				select {
				case chunks <- content:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("error reading response: %w", err)
		}

		result.SetTokenUsage(&TokenUsage{
			InputTokens:  0,
			OutputTokens: estimatedTokens,
			TotalTokens:  estimatedTokens,
			Model:        s.model,
			Provider:     "groq",
		})
	}()

	return result
}
