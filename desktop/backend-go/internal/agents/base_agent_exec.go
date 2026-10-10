package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// RunWithTools executes the agent with tool calling support.
// Uses the configured LLM provider if it supports tool calling (ToolCallingService),
// otherwise falls back to streaming with tool descriptions injected into the prompt.
func (a *BaseAgent) RunWithTools(ctx context.Context, input AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	events := make(chan streaming.StreamEvent, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		// Build messages with context
		messages := a.buildMessages(input)

		// Check if we have tools enabled
		if a.toolRegistry == nil || len(a.enabledTools) == 0 {
			// No tools, fall back to regular streaming
			a.runStreaming(ctx, input, events, errs)
			return
		}

		// Get tool definitions for enabled tools
		toolDefs := make([]services.ToolDefinition, 0)
		for _, toolName := range a.enabledTools {
			if tool, ok := a.toolRegistry.GetTool(toolName); ok {
				toolDefs = append(toolDefs, services.ToolDefinition{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.InputSchema(),
				})
			}
		}

		// Create LLM service using the configured provider (not hardcoded Groq)
		llm := services.NewLLMService(a.cfg, a.model)
		llm.SetOptions(a.llmOptions)

		// Check if the provider supports native tool calling
		toolService, supportsTools := services.AsToolCallingService(llm)
		if !supportsTools {
			// Provider doesn't support tool calling — fall back to streaming
			// with tool descriptions injected into the system prompt so the LLM
			// is still aware of what tools are available.
			slog.Warn("[Agent] Provider does not support tool calling, falling back to streaming",
				"provider", llm.GetProvider(), "model", llm.GetModel(),
				"tools_count", len(toolDefs))
			a.runStreaming(ctx, input, events, errs)
			return
		}

		// Build system prompt with thinking instructions if enabled
		systemPrompt := a.buildSystemPromptWithThinking()

		// First call with tools
		resp, err := toolService.ChatWithTools(ctx, messages, systemPrompt, toolDefs)
		if err != nil {
			slog.Error("[Agent] Tool calling failed", "provider", llm.GetProvider(), "err", err)
			errs <- fmt.Errorf("tool calling failed (%s): %w", llm.GetProvider(), err)
			return
		}

		// Process tool calls if any
		if len(resp.ToolCalls) > 0 {
			// Send thinking event
			events <- streaming.StreamEvent{
				Type: streaming.EventTypeThinking,
				Data: "Executing tools...",
			}

			toolResults := make(map[string]string)
			for _, tc := range resp.ToolCalls {
				// Send tool call event for frontend visibility
				events <- streaming.StreamEvent{
					Type: "tool_call",
					Data: map[string]interface{}{
						"tool_name":    tc.Name,
						"tool_call_id": tc.ID,
						"params":       tc.Arguments,
						"status":       "running",
					},
				}

				// Execute the tool
				result, toolErr := a.toolRegistry.ExecuteTool(ctx, tc.Name, json.RawMessage(tc.Arguments))
				if toolErr != nil {
					toolResults[tc.ID] = fmt.Sprintf("Error: %s", toolErr.Error())
					events <- streaming.StreamEvent{
						Type: "tool_result",
						Data: map[string]interface{}{
							"tool_call_id": tc.ID,
							"status":       "error",
							"result":       toolResults[tc.ID],
						},
					}
				} else {
					toolResults[tc.ID] = result
					events <- streaming.StreamEvent{
						Type: "tool_result",
						Data: map[string]interface{}{
							"tool_call_id": tc.ID,
							"status":       "completed",
							"result":       result,
						},
					}
				}
			}

			// Continue conversation with tool results
			finalResponse, err := toolService.ContinueWithToolResults(ctx, messages, systemPrompt, resp.ToolCalls, toolResults)
			if err != nil {
				errs <- fmt.Errorf("tool continuation failed: %w", err)
				return
			}

			// Stream the final response
			for _, chunk := range splitIntoChunks(finalResponse, 50) {
				events <- streaming.StreamEvent{
					Type:    streaming.EventTypeToken,
					Content: chunk,
				}
			}
		} else {
			// No tool calls, stream the response directly
			for _, chunk := range splitIntoChunks(resp.Content, 50) {
				events <- streaming.StreamEvent{
					Type:    streaming.EventTypeToken,
					Content: chunk,
				}
			}
		}

		events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
	}()

	return events, errs
}

// runStreaming handles regular streaming without tools
func (a *BaseAgent) runStreaming(ctx context.Context, input AgentInput, events chan<- streaming.StreamEvent, errs chan<- error) {
	messages := a.buildMessages(input)
	llm := services.NewLLMService(a.cfg, a.model)
	llm.SetOptions(a.llmOptions)
	detector := streaming.NewArtifactDetector()

	// Build system prompt with thinking instructions if enabled
	systemPrompt := a.buildSystemPromptWithThinking()
	chunks, llmErrs := llm.StreamChat(ctx, messages, systemPrompt)

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				for _, event := range detector.Flush() {
					events <- event
				}
				events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
				return
			}
			for _, event := range detector.ProcessChunk(chunk) {
				events <- event
			}
		case err, ok := <-llmErrs:
			if ok && err != nil {
				errs <- err
				return
			}
			// Channel closed or nil error — stop selecting on it
			llmErrs = nil
		case <-ctx.Done():
			return
		}
	}
}

// splitIntoChunks splits a string into chunks for streaming
func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// Run executes the agent with streaming output
func (a *BaseAgent) Run(ctx context.Context, input AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	events := make(chan streaming.StreamEvent, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		// Build messages with context
		messages := a.buildMessages(input)

		// Create LLM service
		llm := services.NewLLMService(a.cfg, a.model)
		llm.SetOptions(a.llmOptions)

		provider := llm.GetProvider()
		model := llm.GetModel()
		slog.Info("[Agent.Run] Starting stream",
			"agent_type", a.agentType, "provider", provider, "model", model,
			"message_count", len(messages), "focus_mode", input.FocusMode,
			"user_id", input.UserID)

		// Build system prompt with thinking instructions if enabled
		systemPrompt := a.buildSystemPromptWithThinking()

		// Create artifact detector for streaming
		detector := streaming.NewArtifactDetector()

		// Stream response
		chunks, llmErrs := llm.StreamChat(ctx, messages, systemPrompt)

		chunkCount := 0
		// Process chunks through artifact detector
		for {
			select {
			case chunk, ok := <-chunks:
				if !ok {
					// Stream ended - flush detector
					for _, event := range detector.Flush() {
						events <- event
					}
					events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
					slog.Info("[Agent.Run] Stream completed",
						"agent_type", a.agentType, "provider", provider,
						"model", model, "chunks_received", chunkCount)
					return
				}
				chunkCount++
				// Process chunk through artifact detector
				for _, event := range detector.ProcessChunk(chunk) {
					events <- event
				}

			case err, ok := <-llmErrs:
				if ok && err != nil {
					slog.Error("[Agent.Run] LLM error",
						"agent_type", a.agentType, "provider", provider,
						"model", model, "error", err,
						"chunks_before_error", chunkCount)
					errs <- err
					return
				}
				// Channel closed or nil error — stop selecting on it
				llmErrs = nil

			case <-ctx.Done():
				slog.Warn("[Agent.Run] Context cancelled",
					"agent_type", a.agentType, "provider", provider,
					"model", model, "reason", ctx.Err(),
					"chunks_before_cancel", chunkCount)
				return
			}
		}
	}()

	return events, errs
}
