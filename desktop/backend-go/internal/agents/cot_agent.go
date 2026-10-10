package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
	"github.com/rhl/businessos-backend/internal/tools"
)

// COTAgent is an agent that uses Chain of Thought reasoning with real tool calling
type COTAgent struct {
	*BaseAgent
	pool          *pgxpool.Pool
	toolRegistry  *tools.AgentToolRegistry
	maxIterations int
}

// NewCOTAgent creates a new COT-enabled agent
func NewCOTAgent(base *BaseAgent, pool *pgxpool.Pool, cfg *config.Config, userID string) *COTAgent {
	return &COTAgent{
		BaseAgent:   base,
		pool:          pool,
		toolRegistry:  tools.NewAgentToolRegistry(pool, userID),
		maxIterations: 5,
	}
}

// Run executes the agent with COT reasoning and real tool calling
func (a *COTAgent) Run(ctx context.Context, input AgentInput) (<-chan streaming.StreamEvent, <-chan error) {
	events := make(chan streaming.StreamEvent, 100)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		// Send initial thinking event
		events <- streaming.StreamEvent{
			Type: streaming.EventTypeThinking,
			Data: streaming.ThinkingStep{
				Step:    "analyzing",
				Content: "Understanding your request...",
				Agent:   string(a.agentType),
			},
		}

		// Build initial messages
		messages := a.buildMessagesWithTools(input)

		// COT loop - agent can call tools multiple times
		for iteration := 0; iteration < a.maxIterations; iteration++ {
			// Send thinking event for this iteration
			if iteration > 0 {
				events <- streaming.StreamEvent{
					Type: streaming.EventTypeThinking,
					Data: streaming.ThinkingStep{
						Step:    "executing",
						Content: fmt.Sprintf("Processing step %d...", iteration+1),
						Agent:   string(a.agentType),
					},
				}
			}

			// Call LLM
			llm := services.NewLLMService(a.cfg, a.model)
			llm.SetOptions(a.llmOptions)

			// Get full response (non-streaming for tool detection)
			response, err := a.getLLMResponse(ctx, llm, messages)
			if err != nil {
				errs <- err
				return
			}

			// Check for tool calls in response
			toolCalls := a.parseToolCalls(response)

			if len(toolCalls) == 0 {
				// No tool calls - stream the final response
				events <- streaming.StreamEvent{
					Type: streaming.EventTypeThinking,
					Data: streaming.ThinkingStep{
						Step:      "synthesizing",
						Content:   "Preparing response...",
						Agent:     string(a.agentType),
						Completed: true,
					},
				}

				// Stream the response with artifact detection
				detector := streaming.NewArtifactDetector()
				for _, event := range detector.ProcessChunk(response) {
					events <- event
				}
				for _, event := range detector.Flush() {
					events <- event
				}
				events <- streaming.StreamEvent{Type: streaming.EventTypeDone}
				return
			}

			// Execute tool calls
			for _, tc := range toolCalls {
				// Send tool call event
				events <- streaming.StreamEvent{
					Type: streaming.EventTypeToolCall,
					Data: streaming.ToolCallEvent{
						ToolName:   tc.Name,
						Parameters: tc.Parameters,
						Status:     "calling",
					},
				}

				// Execute the tool
				inputJSON, err := json.Marshal(tc.Parameters)
				if err != nil {
					// Send error event and continue to next tool
					events <- streaming.StreamEvent{
						Type: "error",
						Data: map[string]interface{}{
							"message": fmt.Sprintf("Failed to marshal tool parameters for %s: %v", tc.Name, err),
						},
					}
					continue
				}
				result, err := a.toolRegistry.ExecuteTool(ctx, tc.Name, inputJSON)

				status := "success"
				if err != nil {
					status = "error"
					result = err.Error()
				}

				// Send tool result event
				events <- streaming.StreamEvent{
					Type: streaming.EventTypeToolResult,
					Data: streaming.ToolCallEvent{
						ToolName: tc.Name,
						Status:   status,
						Result:   truncateResult(result, 500),
					},
				}

				// Add tool result to messages for next iteration
				messages = append(messages, services.ChatMessage{
					Role:    "assistant",
					Content: fmt.Sprintf("I called %s with result: %s", tc.Name, result),
				})
			}
		}

		// Max iterations reached
		events <- streaming.StreamEvent{
			Type:    streaming.EventTypeError,
			Content: "Maximum iterations reached",
		}
	}()

	return events, errs
}

// ToolCall represents a parsed tool call from LLM response
type ToolCall struct {
	Name       string
	Parameters map[string]interface{}
}

// parseToolCalls extracts tool calls from LLM response
func (a *COTAgent) parseToolCalls(response string) []ToolCall {
	var calls []ToolCall

	// Pattern 1: JSON tool call format
	// {"tool": "tool_name", "parameters": {...}}
	jsonPattern := regexp.MustCompile(`\{[^{}]*"tool"\s*:\s*"([^"]+)"[^{}]*"parameters"\s*:\s*(\{[^{}]*\})[^{}]*\}`)
	matches := jsonPattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(match[2]), &params); err == nil {
				calls = append(calls, ToolCall{
					Name:       match[1],
					Parameters: params,
				})
			}
		}
	}

	// Pattern 2: Function call format
	// <tool_call>{"name": "tool_name", "arguments": {...}}</tool_call>
	toolCallPattern := regexp.MustCompile(`<tool_call>\s*(\{[^<]+\})\s*</tool_call>`)
	toolMatches := toolCallPattern.FindAllStringSubmatch(response, -1)
	for _, match := range toolMatches {
		if len(match) >= 2 {
			var tc struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}
			if err := json.Unmarshal([]byte(match[1]), &tc); err == nil && tc.Name != "" {
				calls = append(calls, ToolCall{
					Name:       tc.Name,
					Parameters: tc.Arguments,
				})
			}
		}
	}

	return calls
}

// getLLMResponse gets a complete response from the LLM
func (a *COTAgent) getLLMResponse(ctx context.Context, llm services.LLMService, messages []services.ChatMessage) (string, error) {
	chunks, errs := llm.StreamChat(ctx, messages, a.systemPrompt)

	var response strings.Builder
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return response.String(), nil
			}
			response.WriteString(chunk)
		case err := <-errs:
			if err != nil {
				return "", err
			}
			return response.String(), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// buildMessagesWithTools adds tool information to the system prompt
func (a *COTAgent) buildMessagesWithTools(input AgentInput) []services.ChatMessage {
	messages := a.buildMessages(input)

	// Add tool definitions to system context
	toolDefs := a.GetToolDefinitions()
	if len(toolDefs) > 0 {
		toolInfo := "\n\n## Available Tools\n\nYou can call these tools by responding with JSON in this format:\n```json\n{\"tool\": \"tool_name\", \"parameters\": {\"param1\": \"value1\"}}\n```\n\nAvailable tools:\n"
		for _, def := range toolDefs {
			if fn, ok := def["function"].(map[string]interface{}); ok {
				name := fn["name"].(string)
				desc := fn["description"].(string)
				toolInfo += fmt.Sprintf("- **%s**: %s\n", name, desc)
			}
		}
		toolInfo += "\nAfter calling a tool, wait for the result before proceeding. Only call tools when necessary.\n"

		// Prepend to first system message or add new one
		if len(messages) > 0 && messages[0].Role == "system" {
			messages[0].Content += toolInfo
		} else {
			messages = append([]services.ChatMessage{{Role: "system", Content: toolInfo}}, messages...)
		}
	}

	return messages
}

// truncateResult truncates a result string to maxLen
func truncateResult(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
