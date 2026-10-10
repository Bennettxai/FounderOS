package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	bossignal "github.com/rhl/businessos-backend/internal/signal"
	"github.com/rhl/businessos-backend/internal/utils"
)

// streamSetupResult holds all values produced during pre-stream setup that the
// SSE loop and post-processing phases require.
type streamSetupResult struct {
	conversationID pgtype.UUID
	convUUID       *uuid.UUID
	contextID      *uuid.UUID
	projectID      *uuid.UUID
	nodeID         *uuid.UUID
	contextIDs     []uuid.UUID
	messages       []sqlc.Message
	chatMessages   []services.ChatMessage
	tieredCtx      *services.TieredContext
	signalEnvelope bossignal.SignalEnvelope
	model          string
	llmOptions     services.LLMOptions
	agentType      agents.AgentType
	// Agent context injected values
	roleContextStr   string
	memoryContextStr string
	cotOrchestrator  *agents.OrchestratorCOT
	agent            agents.Agent
	registry         *agents.AgentRegistry
	// Focus mode values
	focusSystemPrompt string
	searchContextText string
	searchResultCount int
	// Flags
	useCOT bool
}

// prepareStream performs all pre-stream setup: parse IDs, get-or-create conversation,
// save user message, load history, apply compression, build tiered context, classify
// signal, resolve LLM model and options, apply reasoning templates, resolve custom agent
// from @mention, apply focus mode, and configure the agent. Writes HTTP error responses
// directly and returns (nil, false) when setup must abort. Returns (result, true) on success.
func (h *ChatHandler) prepareStream(
	c *gin.Context,
	ctx context.Context,
	req SendMessageRequest,
	userID string,
	userName string,
	focusModeStr string,
) (*streamSetupResult, bool) {
	queries := sqlc.New(h.pool)
	res := &streamSetupResult{}

	// Parse optional IDs
	var contextID *uuid.UUID
	var projectID *uuid.UUID
	var nodeID *uuid.UUID
	var contextIDs []uuid.UUID
	var documentIDs []uuid.UUID

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
	for _, docIDStr := range req.DocumentIDs {
		if parsed, err := uuid.Parse(docIDStr); err == nil {
			documentIDs = append(documentIDs, parsed)
		}
	}
	if len(documentIDs) > 0 {
		slog.Info("[ChatV2] Attached documents", "value", documentIDs)
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}
	if req.NodeID != nil {
		if parsed, err := uuid.Parse(*req.NodeID); err == nil {
			nodeID = &parsed
		}
	}
	res.contextID = contextID
	res.projectID = projectID
	res.nodeID = nodeID
	res.contextIDs = contextIDs

	// Get or create conversation
	var conversationID pgtype.UUID
	var convUUID *uuid.UUID
	if req.ConversationID != nil {
		parsed, err := uuid.Parse(*req.ConversationID)
		if err != nil {
			utils.RespondInvalidID(c, slog.Default(), "conversation_id")
			return nil, false
		}
		conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
		convUUID = &parsed
	} else {
		var ctxID pgtype.UUID
		if contextID != nil {
			ctxID = pgtype.UUID{Bytes: *contextID, Valid: true}
		}
		defaultTitle := "New Conversation"
		conv, err := queries.CreateConversation(ctx, sqlc.CreateConversationParams{
			UserID:    userID,
			Title:     &defaultTitle,
			ContextID: ctxID,
		})
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "create conversation", err)
			return nil, false
		}
		conversationID = conv.ID
		parsed := uuid.UUID(conv.ID.Bytes)
		convUUID = &parsed
	}
	res.conversationID = conversationID
	res.convUUID = convUUID

	// Save user message
	if _, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  conversationID,
		Role:            sqlc.MessageroleUSER,
		Content:         req.Message,
		MessageMetadata: nil,
	}); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save user message", err)
		return nil, false
	}

	// Load conversation history
	messages, err := queries.ListMessages(ctx, conversationID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get conversation messages", err)
		return nil, false
	}
	res.messages = messages

	chatMessages := make([]services.ChatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = services.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Compress conversation if too long (Deep Context Integration - Phase 4)
	if h.tieredContextService != nil {
		compressed, summary, err := h.tieredContextService.CompressConversation(ctx, chatMessages, 20)
		if err == nil && summary != "" {
			chatMessages = compressed
			slog.Debug("ChatV2: Hierarchical summarization applied", "summaryLen", len(summary))
		}
	}
	res.chatMessages = chatMessages

	// Track context window budget
	if h.contextTracker != nil && convUUID != nil {
		convCtx, err := h.contextTracker.GetOrCreateContext(ctx, convUUID.String(), userID)
		if err != nil {
			slog.Warn("ChatV2: failed to init context tracker", "error", err)
		} else {
			userTokens := services.EstimateTokens(req.Message)
			h.contextTracker.AddBlock(ctx, convUUID.String(), &services.ContextBlock{
				Type:       "user",
				Content:    req.Message,
				TokenCount: userTokens,
				Priority:   5,
			})
			slog.Debug("ChatV2: Context tracked",
				"conversation_id", convUUID.String(),
				"user_tokens", userTokens,
				"total_tokens", convCtx.TotalTokens+userTokens,
				"max_tokens", convCtx.MaxTokens,
			)
		}
	}

	// Determine model and build LLM options
	model := h.cfg.GetModelForProvider()
	if req.Model != nil && *req.Model != "" {
		model = *req.Model
	}
	model = normalizeModelName(model)

	llmOptions := services.DefaultLLMOptions()
	if req.Temperature != nil {
		llmOptions.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		llmOptions.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		llmOptions.TopP = *req.TopP
	}
	if req.ThinkingEnabled != nil && *req.ThinkingEnabled {
		llmOptions.ThinkingEnabled = true
		slog.Debug("ChatV2: ThinkingEnabled set from request.ThinkingEnabled")
	}
	if req.UseCOT != nil && *req.UseCOT {
		llmOptions.ThinkingEnabled = true
		slog.Debug("ChatV2: ThinkingEnabled set from request.UseCOT")
	}

	// Apply reasoning template if thinking is enabled
	if llmOptions.ThinkingEnabled {
		var appliedTemplateID *uuid.UUID
		if req.ReasoningTemplateID != nil && *req.ReasoningTemplateID != "" {
			if templateUUID, err := uuid.Parse(*req.ReasoningTemplateID); err == nil {
				template, err := queries.GetReasoningTemplate(ctx, sqlc.GetReasoningTemplateParams{
					ID:     pgtype.UUID{Bytes: templateUUID, Valid: true},
					UserID: userID,
				})
				if err == nil {
					applyReasoningTemplate(&llmOptions, template)
					appliedTemplateID = &templateUUID
					slog.Debug("ChatV2: Applied requested reasoning template", "name", template.Name)
				}
			}
		} else {
			defaultTemplate, err := queries.GetDefaultReasoningTemplate(ctx, userID)
			if err == nil {
				applyReasoningTemplate(&llmOptions, defaultTemplate)
				if defaultTemplate.ID.Valid {
					templateUUID := defaultTemplate.ID.Bytes
					appliedTemplateID = (*uuid.UUID)(&templateUUID)
				}
				slog.Debug("ChatV2: Applied default reasoning template", "name", defaultTemplate.Name)
			}
		}
		if appliedTemplateID != nil {
			go func(templateID uuid.UUID) {
				queries.IncrementTemplateUsage(context.Background(), pgtype.UUID{Bytes: templateID, Valid: true})
			}(*appliedTemplateID)
		}
	}

	// Build tiered context
	var tieredCtx *services.TieredContext
	if h.tieredContextService != nil {
		tieredCtx, _ = h.tieredContextService.BuildTieredContext(ctx, services.TieredContextRequest{
			UserID:      userID,
			ContextIDs:  contextIDs,
			ProjectID:   projectID,
			NodeID:      nodeID,
			DocumentIDs: documentIDs,
		})
	}
	res.tieredCtx = tieredCtx

	// L1: Fast signal classification (zero-LLM, <1ms)
	fastClassifier := bossignal.NewFastClassifier()
	signalEnvelope := fastClassifier.Classify(req.Message, focusModeStr, projectID != nil, false)
	res.signalEnvelope = signalEnvelope

	// Local agent routing setup
	res.useCOT = req.UseCOT != nil && *req.UseCOT

	// Resolve @agent mentions
	mentions := parseAgentMentions(strings.ToLower(req.Message))
	var customAgent *sqlc.CustomAgent
	var customAgentSystemPrompt string

	slog.Info("[ChatV2] @Agent parsing - message: %q, found %d mentions", "value0", req.Message, "value1", len(mentions))
	for i, m := range mentions {
		slog.Info("[ChatV2] @Agent mention %d: name=%q pos=%d-%d", "value0", i, "value1", m.AgentName, "value2", m.StartPos, "value3", m.EndPos)
	}
	if len(mentions) > 0 {
		for _, mention := range mentions {
			slog.Info("[ChatV2] Looking up custom agent: name=%q user_id=%v", "value0", mention.AgentName, "value1", userID)
			agent, err := queries.GetCustomAgentByName(ctx, sqlc.GetCustomAgentByNameParams{
				Lower:  mention.AgentName,
				UserID: userID,
			})
			if err != nil {
				slog.Info("[ChatV2] Agent lookup failed for @", "id", mention.AgentName, "error", err)
				continue
			}
			customAgent = &agent
			customAgentSystemPrompt = agent.SystemPrompt
			go queries.IncrementAgentUsage(context.Background(), agent.ID)
			slog.Info("[ChatV2] Resolved custom agent", "mention", mention.AgentName, "agent", agent.DisplayName, "prompt_chars", len(customAgentSystemPrompt))
			break
		}
	}

	// Always Orchestrator — LLM self-routes via SelfRoutingCapabilities prompt.
	agentType := agents.AgentTypeOrchestrator
	res.agentType = agentType

	// Focus mode: tunes LLM params and optionally fetches web search context.
	var focusSystemPrompt string
	var searchContextText string
	var searchResultCount int
	if req.FocusMode != nil && *req.FocusMode != "" {
		slog.Info("[ChatV2] FocusMode received", "value", *req.FocusMode)
		focusService := services.NewFocusService(h.pool)
		focusCtx, err := focusService.BuildPreflightContext(ctx, userID, *req.FocusMode, req.Message, nil, nil)
		if err == nil {
			llmOptions.Temperature = focusCtx.LLMOptions.Temperature
			llmOptions.MaxTokens = focusCtx.LLMOptions.MaxTokens
			if focusCtx.LLMOptions.ThinkingEnabled {
				llmOptions.ThinkingEnabled = true
			}
			if focusCtx.LLMOptions.Model != nil && *focusCtx.LLMOptions.Model != "" {
				if req.Model == nil || *req.Model == "" {
					model = *focusCtx.LLMOptions.Model
					slog.Info("[ChatV2] Focus mode model override", "id", model)
				}
			}
			focusSystemPrompt = focusCtx.SystemPrompt
			if len(focusCtx.SearchContext) > 0 {
				searchContextText = focusService.FormatContextForPrompt(focusCtx)
				searchResultCount = len(focusCtx.SearchContext)
				slog.Info("[ChatV2] Web search returned  results for focus mode", "count", searchResultCount)
			}
			slog.Info("[ChatV2] Applied focus settings:,,,,", "temp", focusCtx.LLMOptions.Temperature, "maxTokens", focusCtx.LLMOptions.MaxTokens, "thinking", focusCtx.LLMOptions.ThinkingEnabled, "thinking", focusCtx.LLMOptions.Model, "maxTokens", len(focusCtx.SearchContext))
		} else {
			focusSettings, settingsErr := focusService.GetEffectiveSettings(ctx, userID, *req.FocusMode)
			if settingsErr == nil {
				llmOptions.Temperature = focusSettings.Temperature
				llmOptions.MaxTokens = focusSettings.MaxTokens
				if focusSettings.ThinkingEnabled {
					llmOptions.ThinkingEnabled = true
				}
				if focusSettings.EffectiveModel != nil && *focusSettings.EffectiveModel != "" {
					if req.Model == nil || *req.Model == "" {
						model = *focusSettings.EffectiveModel
						slog.Info("[ChatV2] Focus mode model override (fallback)", "model", model)
					}
				}
				focusSystemPrompt = focusSettings.SystemPromptPrefix
			}
		}
	}
	res.focusSystemPrompt = focusSystemPrompt
	res.searchContextText = searchContextText
	res.searchResultCount = searchResultCount

	runtimeCfg, llmAccess, err := services.ResolveLLMAccess(ctx, h.pool, h.cfg, userID, model)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), err.Error())
		return nil, false
	}
	model = llmAccess.Model
	slog.Debug("ChatV2: LLM access resolved",
		"provider", llmAccess.Provider,
		"model", llmAccess.Model,
		"key_source", llmAccess.KeySource,
		"billing_mode", llmAccess.BillingMode,
	)

	registry := agents.NewAgentRegistry(h.pool, runtimeCfg, h.embeddingService, h.promptPersonalizer, h.signalHints)
	res.registry = registry

	// Configure agent with all runtime context layers
	agent := registry.GetAgent(agentType, userID, userName, convUUID, tieredCtx)
	agent.SetModel(model)
	agent.SetOptions(llmOptions)

	wsIDStr := ""
	if req.WorkspaceID != nil {
		wsIDStr = *req.WorkspaceID
	}
	agentCtx := h.injectAgentContext(
		ctx, req, userID, userName, wsIDStr,
		agent, registry, tieredCtx, signalEnvelope, agentType,
		focusModeStr, customAgent, customAgentSystemPrompt,
		focusSystemPrompt, searchContextText,
		res.useCOT, model, llmOptions,
	)
	res.agent = agent
	res.roleContextStr = agentCtx.roleContextStr
	res.memoryContextStr = agentCtx.memoryContextStr
	res.model = agentCtx.model
	res.llmOptions = agentCtx.llmOptions
	res.cotOrchestrator = agentCtx.cotOrchestrator

	return res, true
}
