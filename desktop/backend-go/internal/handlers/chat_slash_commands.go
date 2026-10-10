package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
	"github.com/rhl/businessos-backend/internal/utils"
)

// handleSlashCommandStreaming processes slash commands via the streaming SSE architecture.
func (h *ChatHandler) handleSlashCommandStreaming(c *gin.Context, user *middleware.BetterAuthUser, req SendMessageRequest) {
	// Engine-backed commands skip the LLM entirely — they ask the engine
	// for a typed answer (RAG envelope or recall result) and stream it
	// back as a single chunk + done event. Cheaper than going through
	// Ollama, and the answer is grounded in the workspace's memory + wiki
	// rather than the model's prior.
	if h.engineSlashHandled(c, user, req) {
		return
	}

	// Pre-stream provider health check: fail fast for ollama_local
	if h.cfg.GetActiveProvider() == "ollama_local" {
		if !isOllamaReachable(h.cfg.OllamaLocalURL) {
			slog.Error("[ChatV2-Slash] Ollama local is not reachable",
				"url", h.cfg.OllamaLocalURL, "command", *req.Command)
			c.JSON(503, gin.H{
				"error":    "Ollama is not running or not reachable",
				"provider": "ollama_local",
				"hint":     "Start Ollama with 'ollama serve' or switch to a cloud provider in Settings > AI",
			})
			return
		}
	}

	command := *req.Command
	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Map commands to agent types (6 agents per doc)
	commandAgentMap := map[string]agents.AgentType{
		"analyze":  agents.AgentTypeAnalyst,
		"analysis": agents.AgentTypeAnalyst,
		"document": agents.AgentTypeDocument,
		"write":    agents.AgentTypeDocument,
		"plan":     agents.AgentTypeProject,
		"project":  agents.AgentTypeProject,
		"task":     agents.AgentTypeTask,
		"tasks":    agents.AgentTypeTask,
		"todo":     agents.AgentTypeTask,
		"client":   agents.AgentTypeClient,
		"crm":      agents.AgentTypeClient,
		"deal":     agents.AgentTypeClient,
		"event":    agents.AgentTypeOrchestrator,
		"note":     agents.AgentTypeOrchestrator,
	}

	// Determine agent type from command
	agentType, ok := commandAgentMap[command]
	if !ok {
		agentType = agents.AgentTypeOrchestrator
	}

	// Parse IDs
	var contextID *uuid.UUID
	var projectID *uuid.UUID
	var contextIDs []uuid.UUID

	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = &parsed
		}
	}
	for _, cidStr := range req.ContextIDs {
		if parsed, err := uuid.Parse(cidStr); err == nil {
			contextIDs = append(contextIDs, parsed)
		}
	}
	if len(contextIDs) == 0 && contextID != nil {
		contextIDs = append(contextIDs, *contextID)
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}

	// Get or create conversation
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

	// Save user message with command prefix
	userMessage := fmt.Sprintf("/%s %s", command, req.Message)
	_, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  conversationID,
		Role:            sqlc.MessageroleUSER,
		Content:         userMessage,
		MessageMetadata: nil,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save user message", err)
		return
	}

	// Get conversation history
	messages, err := queries.ListMessages(ctx, conversationID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get conversation messages", err)
		return
	}

	// Convert to chat messages
	chatMessages := make([]services.ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = services.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Inject per-command context sources into the final user message.
	// Look up the command's declared ContextSources from the registry so that,
	// for example, /proposal gets client+project data while /timeline gets
	// tasks+projects — rather than every command receiving the same generic
	// TieredContext blob.
	if cmdInfo, ok := builtInCommands[command]; ok && len(cmdInfo.ContextSources) > 0 {
		bundle := loadContextBundle(ctx, queries, user.ID, contextID, projectID, cmdInfo.ContextSources)
		if bundle != nil {
			enhanced := buildCommandPrompt(cmdInfo, req.Message, bundle)
			// Replace the last user message with the context-enriched version.
			if len(chatMessages) > 0 {
				chatMessages[len(chatMessages)-1].Content = enhanced
			}
			slog.Info("[ChatV2-Slash] Per-command context injected",
				"command", command,
				"sources", cmdInfo.ContextSources,
				"clients", len(bundle.Clients),
				"projects", len(bundle.Projects),
				"tasks", len(bundle.Tasks),
				"documents", len(bundle.Documents),
				"artifacts", len(bundle.Artifacts),
			)
		}
	}

	// Build tiered context
	var tieredCtx *services.TieredContext
	if h.tieredContextService != nil {
		tieredReq := services.TieredContextRequest{
			UserID:     user.ID,
			ContextIDs: contextIDs,
			ProjectID:  projectID,
		}
		tieredCtx, _ = h.tieredContextService.BuildTieredContext(ctx, tieredReq)
	}

	// Create agent registry and get agent
	registry := agents.NewAgentRegistry(h.pool, h.cfg, h.embeddingService, h.promptPersonalizer, h.signalHints)
	agent := registry.GetAgent(agentType, user.ID, user.Name, convUUID, tieredCtx)

	// Set model - use provider-aware default
	model := h.cfg.GetModelForProvider()
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}
	agent.SetModel(model)

	// Set LLM options
	llmOptions := services.DefaultLLMOptions()
	if req.Temperature != nil {
		llmOptions.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		llmOptions.MaxTokens = *req.MaxTokens
	}
	agent.SetOptions(llmOptions)

	// Inject user onboarding profile context for personalization
	slog.Info("[ChatV2-Slash] Profile injection check", "user_id", user.ID)
	onboardingSvc := services.NewOnboardingService(h.pool, nil, nil, nil)
	userProf, err := onboardingSvc.GetUserProfile(ctx, user.ID)
	if err == nil && userProf != nil {
		profileCtx := services.BuildProfilePrefix(userProf)
		agent.SetProfileContext(profileCtx)
		slog.Info("[ChatV2-Slash] Injected user profile context", "chars", len(profileCtx))
	}

	// Inject role context if workspace_id is provided (Feature 1: Role-based permissions)
	if req.WorkspaceID != nil && *req.WorkspaceID != "" && h.roleContextService != nil {
		workspaceID, err := uuid.Parse(*req.WorkspaceID)
		if err == nil {
			roleCtx, err := h.roleContextService.GetUserRoleContext(ctx, user.ID, workspaceID)
			if err == nil {
				// Use the service method to build role context prompt
				rolePrompt := roleCtx.GetRoleContextPrompt()
				agent.SetRoleContextPrompt(rolePrompt)
				slog.Info("[ChatV2-Slash] Injected role context",
					"role", roleCtx.RoleName, "level", roleCtx.HierarchyLevel, "permissions", len(roleCtx.Permissions))
			}
		}
	}

	// Set streaming headers
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Conversation-Id", uuid.UUID(conversationID.Bytes).String())
	c.Header("X-Agent-Type", string(agentType))
	c.Header("X-Command", command)

	// Track timing
	startTime := time.Now()
	provider := h.cfg.GetActiveProvider()

	// Run agent with streaming
	streamCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	agentInput := agents.AgentInput{
		Messages:       chatMessages,
		Context:        tieredCtx,
		FocusMode:      command,
		ConversationID: *convUUID,
		UserID:         user.ID,
		UserName:       user.Name,
	}

	events, errs := agent.RunWithTools(streamCtx, agentInput)

	var fullResponse string
	var thinkingContent string
	var detectedArtifacts []streaming.Artifact
	var streamErr error

	// Stream response
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				// Send usage data
				sendUsageEvent(w, startTime, req.Message, messages, fullResponse, provider, model, 0)
				return false
			}

			switch event.Type {
			case streaming.EventTypeToken:
				if event.Content != "" {
					fullResponse += event.Content
				} else if content, ok := event.Data.(string); ok {
					fullResponse += content
				}
			case streaming.EventTypeThinking:
				if content, ok := event.Data.(string); ok {
					thinkingContent += content
				} else if mapData, ok := event.Data.(map[string]interface{}); ok {
					if c, ok := mapData["content"].(string); ok {
						thinkingContent += c
					}
				}
			case streaming.EventTypeArtifactComplete:
				if artifact, ok := event.Data.(streaming.Artifact); ok {
					detectedArtifacts = append(detectedArtifacts, artifact)
				}
			}

			writeSSEEvent(w, event)
			return true

		case err := <-errs:
			if err != nil {
				streamErr = err
				slog.Error("[ChatV2-Slash] Stream error",
					"command", command, "error", err,
					"provider", provider, "model", model,
					"user_id", user.ID)
				errorEvent := streaming.StreamEvent{
					Type:    streaming.EventTypeError,
					Content: err.Error(),
					Data:    err.Error(),
				}
				writeSSEEvent(w, errorEvent)
			}
			return false

		case <-streamCtx.Done():
			slog.Warn("[ChatV2-Slash] Stream context cancelled",
				"command", command, "reason", streamCtx.Err(),
				"elapsed", time.Since(startTime))
			return false
		}
	})

	// Log empty response for debugging
	if fullResponse == "" && streamErr == nil {
		slog.Warn("[ChatV2-Slash] Empty response from agent",
			"command", command, "agent_type", string(agentType),
			"provider", provider, "model", model,
			"user_id", user.ID, "elapsed", time.Since(startTime))
	}

	// Post-process using the same pipeline as regular messages
	focusModeStr := command
	h.postProcessStream(ctx, postProcessParams{
		req:               req,
		userID:            user.ID,
		conversationID:    conversationID,
		convUUID:          convUUID,
		contextID:         contextID,
		projectID:         projectID,
		contextIDs:        contextIDs,
		messages:          messages,
		chatMessages:      chatMessages,
		detectedArtifacts: detectedArtifacts,
		fullResponse:      fullResponse,
		thinkingContent:   thinkingContent,
		agentType:         agentType,
		focusModeStr:      focusModeStr,
		model:             model,
		startTime:         startTime,
	})

	// Post-stream action: persist a real record based on the command type.
	// The entity name is taken directly from the user's input (the text after the
	// command word), so the record name reflects exactly what the user typed.
	if streamErr == nil && fullResponse != "" {
		h.executePostStreamAction(ctx, command, req.Message, user.ID, convUUID)
	}
}

// executePostStreamAction creates a real database record after the LLM stream
// completes for action-oriented slash commands (/task, /project, /client, /deal,
// /note, /event). The entity name is extracted from the raw user input so it
// always matches what the user actually typed, not the LLM's paraphrase.
//
// An "action_completed" SSE event cannot be sent here because the gin response
// writer is already flushed. Instead callers can poll their respective list
// endpoints; the frontend will pick up the new record on its next fetch cycle.
func (h *ChatHandler) executePostStreamAction(ctx context.Context, command, userInput, userID string, convUUID *uuid.UUID) {
	queries := sqlc.New(h.pool)

	// Extract the entity name: everything the user typed after the command word.
	// e.g. "/task Build landing page" → userInput == "Build landing page"
	name := strings.TrimSpace(userInput)
	if name == "" {
		return
	}

	switch command {
	case "task", "tasks", "todo":
		task, err := queries.CreateTask(ctx, sqlc.CreateTaskParams{
			UserID: userID,
			Title:  name,
		})
		if err != nil {
			slog.Error("[SlashAction] Failed to create task",
				"command", command, "name", name, "user_id", userID, "error", err)
			return
		}
		slog.Info("[SlashAction] Task created",
			"id", uuid.UUID(task.ID.Bytes).String(), "name", task.Title, "user_id", userID)

	case "project", "plan":
		proj, err := queries.CreateProject(ctx, sqlc.CreateProjectParams{
			UserID: userID,
			Name:   name,
		})
		if err != nil {
			slog.Error("[SlashAction] Failed to create project",
				"command", command, "name", name, "user_id", userID, "error", err)
			return
		}
		slog.Info("[SlashAction] Project created",
			"id", uuid.UUID(proj.ID.Bytes).String(), "name", proj.Name, "user_id", userID)

	case "client", "crm":
		client, err := queries.CreateClient(ctx, sqlc.CreateClientParams{
			UserID: userID,
			Name:   name,
		})
		if err != nil {
			slog.Error("[SlashAction] Failed to create client",
				"command", command, "name", name, "user_id", userID, "error", err)
			return
		}
		slog.Info("[SlashAction] Client created",
			"id", uuid.UUID(client.ID.Bytes).String(), "name", client.Name, "user_id", userID)

	case "deal":
		deal, err := queries.CreateCRMDeal(ctx, sqlc.CreateCRMDealParams{
			UserID: userID,
			Name:   name,
		})
		if err != nil {
			slog.Error("[SlashAction] Failed to create deal",
				"command", command, "name", name, "user_id", userID, "error", err)
			return
		}
		slog.Info("[SlashAction] CRM deal created",
			"id", uuid.UUID(deal.ID.Bytes).String(), "name", deal.Name, "user_id", userID)

	case "note":
		now := pgtype.Date{}
		if err := now.Scan(time.Now()); err != nil {
			slog.Error("[SlashAction] Failed to build date for daily log", "error", err)
			return
		}
		log, err := queries.CreateDailyLog(ctx, sqlc.CreateDailyLogParams{
			UserID:  userID,
			Date:    now,
			Content: name,
		})
		if err != nil {
			slog.Error("[SlashAction] Failed to create daily log",
				"command", command, "user_id", userID, "error", err)
			return
		}
		slog.Info("[SlashAction] Daily log created",
			"id", log.ID, "date", log.Date, "user_id", userID)

	case "event":
		// TODO: CreateCalendarEvent requires start_time and end_time which cannot
		// be reliably inferred from a short natural-language phrase without
		// additional NLP parsing. The command is registered and routed to the LLM
		// for a descriptive response; a future iteration should parse date/time
		// from the user's input (e.g. via a time-extraction service) and then call
		// queries.CreateCalendarEvent.
		slog.Info("[SlashAction] /event command acknowledged; calendar record creation pending NLP date parsing",
			"input", name, "user_id", userID)
	}
}

// sanitizeJSONResponse checks if the LLM response looks like raw JSON (starts
// with '{' or '[') and logs a warning. It does not strip the content because
// some commands legitimately return structured data to be rendered as a table;
// the real fix is the system-prompt formatting rules.
func sanitizeJSONResponse(command, response string) string {
	trimmed := strings.TrimSpace(response)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		slog.Warn("[SlashSanitize] LLM returned raw JSON — check system prompt formatting rules",
			"command", command, "preview", truncateString(trimmed, 120))
	}
	return response
}

// slashActionCompletePayload is the structured payload sent as an SSE
// action_completed event so the frontend can optimistically update its state.
type slashActionCompletePayload struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// marshalActionEvent serialises a slashActionCompletePayload to JSON, swallowing
// errors since this is best-effort telemetry for the frontend.
func marshalActionEvent(p slashActionCompletePayload) string {
	b, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(b)
}
