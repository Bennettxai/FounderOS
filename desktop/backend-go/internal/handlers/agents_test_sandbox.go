package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TestCustomAgent provides a sandbox to test agent prompts without saving
// POST /api/agents/:id/test - Test existing agent with custom message
// POST /api/agents/sandbox - Test arbitrary prompt (no agent ID needed)
func (h *AgentHandler) TestCustomAgent(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req TestAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// If agent ID is provided, load the agent's settings
	idStr := c.Param("id")
	var systemPrompt string
	var model string
	var temperature float64 = 0.7

	if idStr != "" && idStr != "sandbox" {
		// Testing existing agent
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
			return
		}

		agent, err := queries.GetCustomAgent(ctx, sqlc.GetCustomAgentParams{
			ID:     pgtype.UUID{Bytes: id, Valid: true},
			UserID: user.ID,
		})
		if err != nil {
			utils.RespondNotFound(c, slog.Default(), "Agent")
			return
		}

		// Use agent's prompt if not overridden
		if req.SystemPrompt != "" {
			systemPrompt = req.SystemPrompt
		} else {
			systemPrompt = agent.SystemPrompt
		}

		// Use agent's model preference if set
		if agent.ModelPreference != nil && *agent.ModelPreference != "" {
			model = *agent.ModelPreference
		}

		// Use agent's temperature if set
		if agent.Temperature.Valid {
			tempFloat, _ := agent.Temperature.Float64Value()
			if tempFloat.Valid {
				temperature = tempFloat.Float64
			}
		}
	} else {
		// Sandbox mode - use provided prompt
		systemPrompt = req.SystemPrompt
		if systemPrompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "system_prompt is required for sandbox mode"})
			return
		}
	}

	// Override with request parameters if provided
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}
	if model == "" {
		model = h.cfg.DefaultModel
	}
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	// Create LLM service
	llmService := services.NewLLMService(h.cfg, model)

	// Set options
	opts := services.DefaultLLMOptions()
	opts.Temperature = temperature
	if req.MaxTokens != nil {
		opts.MaxTokens = *req.MaxTokens
	} else {
		opts.MaxTokens = 1000 // Limit for sandbox testing
	}
	llmService.SetOptions(opts)

	// Build messages
	chatMessages := []services.ChatMessage{
		{Role: "user", Content: req.TestMessage},
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Flush headers
	c.Writer.Flush()

	// Stream response and collect
	var fullResponse strings.Builder
	chunks, errs := llmService.StreamChat(ctx, chatMessages, systemPrompt)

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				goto done
			}
			fullResponse.WriteString(chunk)

			// Send SSE content event
			event := map[string]interface{}{
				"type": "content",
				"data": chunk,
			}
			jsonData, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				slog.Error("failed to marshal SSE content event", "error", marshalErr)
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
			c.Writer.Flush()

		case streamErr := <-errs:
			if streamErr != nil {
				// Send SSE error event
				event := map[string]interface{}{
					"type":    "error",
					"message": "LLM error: " + streamErr.Error(),
				}
				jsonData, marshalErr := json.Marshal(event)
				if marshalErr != nil {
					slog.Error("failed to marshal SSE error event", "error", marshalErr)
					return
				}
				fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
				c.Writer.Flush()
				return
			}
			goto done
		case <-ctx.Done():
			// Send SSE timeout event
			event := map[string]interface{}{
				"type":    "error",
				"message": "Request timeout",
			}
			jsonData, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				slog.Error("failed to marshal SSE timeout event", "error", marshalErr)
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
			c.Writer.Flush()
			return
		}
	}

done:
	response := fullResponse.String()
	tokensUsed := len(response) / 4 // Rough estimate

	// Send final SSE done event
	event := map[string]interface{}{
		"type":   "done",
		"tokens": tokensUsed,
		"model":  model,
	}
	jsonData, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		slog.Error("failed to marshal SSE done event", "error", marshalErr)
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
	c.Writer.Flush()
}
