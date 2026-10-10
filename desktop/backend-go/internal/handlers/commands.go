package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/tools"
	"github.com/rhl/businessos-backend/internal/utils"
)

// CommandHandler handles slash command operations.
type CommandHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(pool *pgxpool.Pool, cfg *config.Config) *CommandHandler {
	return &CommandHandler{pool: pool, cfg: cfg}
}

// handleSlashCommand routes an incoming slash command to the LLM with the
// appropriate system prompt and context bundle, then streams the response.
func (h *CommandHandler) handleSlashCommand(c *gin.Context, user *middleware.BetterAuthUser, req SendMessageRequest) {
	command := strings.ToLower(strings.TrimPrefix(*req.Command, "/"))

	queries := sqlc.New(h.pool)
	ctx := c.Request.Context()

	// Check built-in commands first, then fall back to user-defined commands.
	cmdInfo, exists := builtInCommands[command]
	if !exists {
		customCmd, err := queries.GetUserCommandByName(ctx, sqlc.GetUserCommandByNameParams{
			Name:   command,
			UserID: user.ID,
		})
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), fmt.Sprintf("Unknown command: %s", command))
			return
		}
		desc := ""
		if customCmd.Description != nil {
			desc = *customCmd.Description
		}
		icon := "sparkles"
		if customCmd.Icon != nil {
			icon = *customCmd.Icon
		}
		cmdInfo = CommandInfo{
			Name:           customCmd.Name,
			DisplayName:    customCmd.DisplayName,
			Description:    desc,
			Icon:           icon,
			Category:       "custom",
			SystemPrompt:   customCmd.SystemPrompt,
			ContextSources: customCmd.ContextSources,
		}
	}

	// Parse optional scope IDs.
	var contextID, projectID *uuid.UUID
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = &parsed
		}
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}

	// Get or create the conversation.
	var conversationID pgtype.UUID
	var convUUID *uuid.UUID
	if req.ConversationID != nil {
		parsed, err := uuid.Parse(*req.ConversationID)
		if err != nil {
			utils.RespondInvalidID(c, slog.Default(), "conversation_id")
			return
		}
		conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
		convUUID = &parsed
	} else {
		var ctxID pgtype.UUID
		if contextID != nil {
			ctxID = pgtype.UUID{Bytes: *contextID, Valid: true}
		}
		title := fmt.Sprintf("/%s: %s", command, truncateString(req.Message, 40))
		conv, err := queries.CreateConversation(ctx, sqlc.CreateConversationParams{
			UserID:    user.ID,
			Title:     &title,
			ContextID: ctxID,
		})
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "create conversation", err)
			return
		}
		conversationID = conv.ID
		parsed := uuid.UUID(conv.ID.Bytes)
		convUUID = &parsed
	}

	// Persist the user message (with command prefix for history readability).
	userMessage := fmt.Sprintf("/%s %s", command, req.Message)
	_, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  conversationID,
		Role:            sqlc.MessageroleUSER,
		Content:         userMessage,
		MessageMetadata: nil,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save message", err)
		return
	}

	// Load context and build the enhanced prompt.
	contextBundle := loadContextBundle(ctx, queries, user.ID, contextID, projectID, cmdInfo.ContextSources)
	enhancedPrompt := buildCommandPrompt(cmdInfo, req.Message, contextBundle)

	// Assemble conversation history (excluding the message we just saved).
	messages, _ := queries.ListMessages(ctx, conversationID)
	chatMessages := make([]services.ChatMessage, 0, len(messages))
	for i := 0; i < len(messages)-1; i++ {
		chatMessages = append(chatMessages, services.ChatMessage{
			Role:    string(messages[i].Role),
			Content: messages[i].Content,
		})
	}
	chatMessages = append(chatMessages, services.ChatMessage{
		Role:    "user",
		Content: enhancedPrompt,
	})

	// Determine model.
	model := h.cfg.DefaultModel
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}

	// Set streaming headers.
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("X-Conversation-Id", uuidToString(conversationID))
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	startTime := time.Now()
	provider := h.cfg.GetActiveProvider()

	llm := services.NewLLMService(h.cfg, model)
	streamCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	chunks, errs := llm.StreamChat(streamCtx, chatMessages, cmdInfo.SystemPrompt)

	var fullResponse string
	var streamErr error
	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return false
			}
			fullResponse += chunk
			w.Write([]byte(chunk))
			return true
		case err := <-errs:
			if err != nil {
				streamErr = err
				w.Write([]byte("\n\n[Error: " + err.Error() + "]"))
			}
			return false
		case <-streamCtx.Done():
			return false
		}
	})

	// Append inline usage metadata for the frontend.
	if streamErr == nil && fullResponse != "" {
		endTime := time.Now()
		inputChars := len(enhancedPrompt)
		for _, msg := range messages {
			inputChars += len(msg.Content)
		}
		outputChars := len(fullResponse)
		inputTokens := inputChars / 4
		outputTokens := outputChars / 4
		totalTokens := inputTokens + outputTokens
		durationMs := endTime.Sub(startTime).Milliseconds()
		tps := float64(0)
		if durationMs > 0 {
			tps = float64(outputTokens) / (float64(durationMs) / 1000)
		}
		estimatedCost := services.CalculateEstimatedCost(provider, model, inputTokens, outputTokens)

		usageJSON := fmt.Sprintf("\n\n<!--USAGE:{\"input_tokens\":%d,\"output_tokens\":%d,\"total_tokens\":%d,\"duration_ms\":%d,\"tps\":%.1f,\"provider\":\"%s\",\"model\":\"%s\",\"estimated_cost\":%.6f,\"command\":\"%s\"}-->",
			inputTokens, outputTokens, totalTokens, durationMs, tps, provider, model, estimatedCost, command)
		c.Writer.Write([]byte(usageJSON))
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Persist the assistant response and log usage asynchronously.
	if fullResponse != "" {
		parsed, err := tools.SaveArtifactsFromResponse(ctx, h.pool, user.ID, convUUID, contextID, fullResponse)
		if err == nil && len(parsed.Artifacts) > 0 {
			fullResponse = parsed.CleanResponse
		}

		queries.CreateMessage(ctx, sqlc.CreateMessageParams{
			ConversationID:  conversationID,
			Role:            sqlc.MessageroleASSISTANT,
			Content:         fullResponse,
			MessageMetadata: []byte(fmt.Sprintf(`{"command":"%s"}`, command)),
		})

		endTime := time.Now()
		inputChars := len(enhancedPrompt)
		for _, msg := range messages {
			inputChars += len(msg.Content)
		}
		inputTokens := inputChars / 4
		outputTokens := len(fullResponse) / 4
		estimatedCost := services.CalculateEstimatedCost(provider, model, inputTokens, outputTokens)

		go func() {
			usageService := services.NewUsageService(h.pool)
			usageService.LogAIUsage(context.Background(), services.LogAIUsageParams{
				UserID:         user.ID,
				ConversationID: convUUID,
				Provider:       provider,
				Model:          model,
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				TotalTokens:    inputTokens + outputTokens,
				AgentName:      "command_" + command,
				RequestType:    "command",
				ProjectID:      projectID,
				DurationMs:     int(endTime.Sub(startTime).Milliseconds()),
				StartedAt:      startTime,
				CompletedAt:    endTime,
				EstimatedCost:  estimatedCost,
			})
		}()
	}
}

// ListCommands returns all available slash commands (built-in and custom).
func (h *CommandHandler) ListCommands(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	commands := make([]gin.H, 0, len(builtInCommands)+10)
	for _, cmd := range builtInCommands {
		commands = append(commands, gin.H{
			"name":            cmd.Name,
			"display_name":    cmd.DisplayName,
			"description":     cmd.Description,
			"icon":            cmd.Icon,
			"category":        cmd.Category,
			"context_sources": cmd.ContextSources,
			"system_prompt":   cmd.SystemPrompt,
			"is_custom":       false,
		})
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)
	userCommands, err := queries.ListUserCommands(ctx, user.ID)
	if err == nil {
		for _, cmd := range userCommands {
			desc := ""
			if cmd.Description != nil {
				desc = *cmd.Description
			}
			icon := "sparkles"
			if cmd.Icon != nil {
				icon = *cmd.Icon
			}
			commands = append(commands, gin.H{
				"id":              cmd.ID,
				"name":            cmd.Name,
				"display_name":    cmd.DisplayName,
				"description":     desc,
				"icon":            icon,
				"category":        "custom",
				"context_sources": cmd.ContextSources,
				"system_prompt":   cmd.SystemPrompt,
				"is_custom":       true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"commands": commands})
}

// CreateUserCommandRequest is the request body for creating a custom command.
type CreateUserCommandRequest struct {
	Name           string   `json:"name" binding:"required"`
	DisplayName    string   `json:"display_name" binding:"required"`
	Description    string   `json:"description"`
	Icon           string   `json:"icon"`
	SystemPrompt   string   `json:"system_prompt" binding:"required"`
	ContextSources []string `json:"context_sources"`
}

// CreateUserCommand creates a new custom slash command.
func (h *CommandHandler) CreateUserCommand(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateUserCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	name := strings.ToLower(strings.TrimSpace(req.Name))
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			utils.RespondBadRequest(c, slog.Default(), "Command name can only contain lowercase letters, numbers, and hyphens")
			return
		}
	}
	if _, exists := builtInCommands[name]; exists {
		utils.RespondConflict(c, slog.Default(), "Cannot use a built-in command name")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var icon *string
	if req.Icon != "" {
		icon = &req.Icon
	}

	cmd, err := queries.CreateUserCommand(ctx, sqlc.CreateUserCommandParams{
		UserID:         user.ID,
		Name:           name,
		DisplayName:    req.DisplayName,
		Description:    desc,
		Icon:           icon,
		SystemPrompt:   req.SystemPrompt,
		ContextSources: req.ContextSources,
		IsActive:       boolPtr(true),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create command", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"command": cmd})
}

// GetUserCommand retrieves a specific custom command by ID.
func (h *CommandHandler) GetUserCommand(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "command_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	cmd, err := queries.GetUserCommand(ctx, sqlc.GetUserCommandParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Command")
		return
	}

	c.JSON(http.StatusOK, gin.H{"command": cmd})
}

// UpdateUserCommandRequest is the request body for updating a custom command.
type UpdateUserCommandRequest struct {
	Name           *string  `json:"name"`
	DisplayName    *string  `json:"display_name"`
	Description    *string  `json:"description"`
	Icon           *string  `json:"icon"`
	SystemPrompt   *string  `json:"system_prompt"`
	ContextSources []string `json:"context_sources"`
	IsActive       *bool    `json:"is_active"`
}

// UpdateUserCommand updates an existing custom command.
func (h *CommandHandler) UpdateUserCommand(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "command_id")
		return
	}

	var req UpdateUserCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	existing, err := queries.GetUserCommand(ctx, sqlc.GetUserCommandParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Command")
		return
	}

	name := existing.Name
	displayName := existing.DisplayName
	systemPrompt := existing.SystemPrompt
	description := existing.Description
	icon := existing.Icon
	contextSources := existing.ContextSources
	isActive := existing.IsActive

	if req.Name != nil {
		name = strings.ToLower(strings.TrimSpace(*req.Name))
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				utils.RespondBadRequest(c, slog.Default(), "Command name can only contain lowercase letters, numbers, and hyphens")
				return
			}
		}
		if _, exists := builtInCommands[name]; exists {
			utils.RespondConflict(c, slog.Default(), "Cannot use a built-in command name")
			return
		}
	}
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	if req.SystemPrompt != nil {
		systemPrompt = *req.SystemPrompt
	}
	if req.Description != nil {
		description = req.Description
	}
	if req.Icon != nil {
		icon = req.Icon
	}
	if req.ContextSources != nil {
		contextSources = req.ContextSources
	}
	if req.IsActive != nil {
		isActive = req.IsActive
	}

	cmd, err := queries.UpdateUserCommand(ctx, sqlc.UpdateUserCommandParams{
		ID:             pgtype.UUID{Bytes: id, Valid: true},
		UserID:         user.ID,
		Name:           name,
		DisplayName:    displayName,
		Description:    description,
		Icon:           icon,
		SystemPrompt:   systemPrompt,
		ContextSources: contextSources,
		IsActive:       isActive,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update command", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"command": cmd})
}

// DeleteUserCommand deletes a custom command by ID.
func (h *CommandHandler) DeleteUserCommand(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "command_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	err = queries.DeleteUserCommand(ctx, sqlc.DeleteUserCommandParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete command", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}
