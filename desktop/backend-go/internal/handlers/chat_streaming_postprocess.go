package handlers

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
	"github.com/rhl/businessos-backend/internal/subconscious"
	"github.com/rhl/businessos-backend/internal/tools"
)

// postProcessParams groups all values needed after the SSE stream completes.
type postProcessParams struct {
	req               SendMessageRequest
	userID            string
	conversationID    pgtype.UUID
	convUUID          *uuid.UUID
	contextID         *uuid.UUID
	projectID         *uuid.UUID
	nodeID            *uuid.UUID
	contextIDs        []uuid.UUID
	messages          []sqlc.Message
	chatMessages      []services.ChatMessage
	detectedArtifacts []streaming.Artifact
	fullResponse      string
	thinkingContent   string
	pendingArtifact   pendingArtifactState
	agentType         agents.AgentType
	focusModeStr      string
	model             string
	startTime         time.Time
}

// pendingArtifactState tracks artifact-from-tool state accumulated during streaming.
type pendingArtifactState struct {
	title        string
	artifactType string
	contentStart int
}

// postProcessStream saves all side effects that occur after the SSE stream finishes:
// thinking traces, artifacts, assistant message, signal log, subconscious observation,
// context window tracking, conversation title update, and auto-learning triggers.
func (h *ChatHandler) postProcessStream(ctx context.Context, p postProcessParams) {
	queries := sqlc.New(h.pool)

	// If no response content was generated, save a minimal placeholder so the
	// frontend always has an assistant message to render (prevents blank UI).
	if p.fullResponse == "" {
		slog.Warn("[PostProcess] Empty assistant response, saving placeholder",
			"agent_type", string(p.agentType), "focus_mode", p.focusModeStr,
			"user_id", p.userID)
		if _, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
			ConversationID:  p.conversationID,
			Role:            sqlc.MessageroleASSISTANT,
			Content:         "I wasn't able to generate a response. Please try again.",
			MessageMetadata: nil,
		}); err != nil {
			slog.Error("[PostProcess] Failed to save placeholder message", "error", err)
		}
		return
	}

	// Strip thinking tags for clean database storage
	cleanResponse := stripThinkingTags(p.fullResponse)

	// Persist thinking trace if thinking was present
	if p.thinkingContent != "" && p.convUUID != nil {
		saveThinkingTrace(ctx, h.pool, p.userID, *p.convUUID, p.thinkingContent, p.model, p.startTime)
	}

	// Persist detected artifacts (from stream events)
	for _, artifact := range p.detectedArtifacts {
		tools.CreateArtifact(ctx, h.pool, p.userID, p.convUUID, p.contextID, p.projectID, tools.ArtifactData{
			Type:    artifact.Type,
			Title:   artifact.Title,
			Content: artifact.Content,
		})
	}

	// Persist artifact created from a tool call marker (ARTIFACT_START::...)
	if p.pendingArtifact.title != "" && p.pendingArtifact.contentStart > 0 {
		artifactContent := p.fullResponse
		if p.pendingArtifact.contentStart < len(p.fullResponse) {
			artifactContent = p.fullResponse[p.pendingArtifact.contentStart:]
		}
		artifactContent = stripArtifactInstructions(artifactContent)

		if len(artifactContent) > 100 {
			artifact, err := tools.CreateArtifact(ctx, h.pool, p.userID, p.convUUID, p.contextID, p.projectID, tools.ArtifactData{
				Type:    p.pendingArtifact.artifactType,
				Title:   p.pendingArtifact.title,
				Content: artifactContent,
			})
			if err == nil && artifact != nil {
				slog.Debug("ChatV2: Created artifact from tool",
					"title", p.pendingArtifact.title,
					"type", p.pendingArtifact.artifactType,
					"len", len(artifactContent))
			}
		}
	}

	// Parse and persist artifacts embedded in the response text (fallback path)
	parsed, err := tools.SaveArtifactsFromResponse(ctx, h.pool, p.userID, p.convUUID, p.contextID, cleanResponse)
	if err == nil && len(parsed.Artifacts) > 0 {
		cleanResponse = parsed.CleanResponse

		// Link artifacts to project if applicable
		if p.projectID != nil {
			for _, artifactData := range parsed.Artifacts {
				if artifactData.Summary != "" {
					if artifactID, err := uuid.Parse(artifactData.Summary); err == nil {
						queries.LinkArtifact(ctx, sqlc.LinkArtifactParams{
							ID:        pgtype.UUID{Bytes: artifactID, Valid: true},
							ProjectID: pgtype.UUID{Bytes: *p.projectID, Valid: true},
						})
					}
				}
			}
		}
	}

	// Save assistant message (without thinking tags)
	queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  p.conversationID,
		Role:            sqlc.MessageroleASSISTANT,
		Content:         cleanResponse,
		MessageMetadata: nil,
	})

	// Async: signal logging + subconscious observation (zero latency — fire-and-forget)
	go func() {
		signalLogID := logSignal(
			context.Background(), h.pool, p.userID, p.convUUID,
			p.focusModeStr, p.req.Message, len(cleanResponse), p.startTime,
		)
		if h.subconsciousObserver != nil {
			var prevUserMsg string
			for i := len(p.chatMessages) - 1; i >= 0; i-- {
				if p.chatMessages[i].Role == "user" && p.chatMessages[i].Content != p.req.Message {
					prevUserMsg = p.chatMessages[i].Content
					break
				}
			}
			var convID string
			if p.convUUID != nil {
				convID = p.convUUID.String()
			}
			h.subconsciousObserver.Observe(subconscious.ObserveInput{
				UserID:          p.userID,
				UserMessage:     p.req.Message,
				PreviousUserMsg: prevUserMsg,
				AssistantResp:   cleanResponse,
				SignalLogID:     signalLogID,
				AgentType:       string(p.agentType),
				ConversationID:  convID,
				Latency:         time.Since(p.startTime),
			})
		}
	}()

	// Track assistant response in context window budget
	if h.contextTracker != nil && p.convUUID != nil {
		assistantTokens := services.EstimateTokens(cleanResponse)
		h.contextTracker.AddBlock(ctx, p.convUUID.String(), &services.ContextBlock{
			Type:       "assistant",
			Content:    cleanResponse,
			TokenCount: assistantTokens,
			Priority:   3, // Lower priority than user messages for eviction
		})
	}

	// Update conversation title on the first exchange
	if len(p.messages) <= 1 {
		title := p.req.Message
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		queries.UpdateConversation(ctx, sqlc.UpdateConversationParams{
			ID:    p.conversationID,
			Title: &title,
		})
	}

	// Trigger automatic learning from this conversation turn
	if h.autoLearningTriggers != nil && p.convUUID != nil {
		focusModeValue := ""
		if p.req.FocusMode != nil {
			focusModeValue = *p.req.FocusMode
		}

		var workspaceID *uuid.UUID
		if p.req.WorkspaceID != nil && *p.req.WorkspaceID != "" {
			if parsed, err := uuid.Parse(*p.req.WorkspaceID); err == nil {
				workspaceID = &parsed
			}
		}

		h.autoLearningTriggers.ProcessConversationTurn(ctx, services.LearningConversationContext{
			UserID:         p.userID,
			WorkspaceID:    workspaceID,
			ConversationID: *p.convUUID,
			UserMessage:    p.req.Message,
			AgentResponse:  cleanResponse,
			AgentType:      string(p.agentType),
			FocusMode:      focusModeValue,
			ProjectID:      p.projectID,
			NodeID:         p.nodeID,
			ContextIDs:     p.contextIDs,
			Timestamp:      time.Now(),
		})
	}
}

// stripArtifactInstructions removes the boilerplate instruction text prepended to artifact content
// from tool-call markers before persisting the content.
func stripArtifactInstructions(content string) string {
	const boilerplate = "Now write the complete document content below. Everything you write will be saved to the artifact."
	return strings.TrimSpace(strings.TrimPrefix(content, boilerplate))
}
