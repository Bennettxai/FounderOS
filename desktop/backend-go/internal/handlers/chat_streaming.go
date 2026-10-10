package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/streaming"
	"github.com/rhl/businessos-backend/internal/utils"
)

// Package-level compiled regexes for thinking tag detection (avoids recompiling per token)
var (
	thinkingStartRe = regexp.MustCompile(`<think[a-z]*\s*>`)
	thinkingEndRe   = regexp.MustCompile(`</think[a-z]*\s*>`)
)

// isOllamaReachable performs a fast health check against the local Ollama instance.
// Returns false if Ollama is not running or not reachable within 3 seconds.
func isOllamaReachable(baseURL string) bool {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// SendMessage handles chat messages using streaming SSE events with artifact detection.
// The implementation is split across:
//   - chat_streaming_setup.go    — pre-stream setup (prepareStream)
//   - chat_streaming_osa.go      — OSA routing (tryOSARouting)
//   - chat_streaming_postprocess.go — post-stream persistence (postProcessStream)
//   - chat_streaming_sse.go      — SSE write helpers (writeSSEEvent, sendUsageEvent, logSignal)
func (h *ChatHandler) SendMessage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	focusModeStr := "nil"
	if req.FocusMode != nil {
		focusModeStr = *req.FocusMode
	}
	slog.Debug("ChatV2: Request received",
		"focus", focusModeStr,
		"cot", req.UseCOT,
		"workspace_id", req.WorkspaceID,
	)

	// Handle slash commands
	if req.Command != nil && *req.Command != "" {
		h.handleSlashCommandStreaming(c, user, req)
		return
	}

	ctx := c.Request.Context()

	// Pre-stream provider health check: fail fast for ollama_local instead of hanging 60s
	provider := h.cfg.GetActiveProvider()
	if provider == "ollama_local" {
		if !isOllamaReachable(h.cfg.OllamaLocalURL) {
			slog.Error("[ChatV2] Ollama local is not reachable, cannot process message",
				"url", h.cfg.OllamaLocalURL)
			c.JSON(503, gin.H{
				"error":    "Ollama is not running or not reachable",
				"provider": "ollama_local",
				"hint":     "Start Ollama with 'ollama serve' or switch to a cloud provider (groq, anthropic) in Settings > AI",
			})
			return
		}
	}

	// OSA routing: attempt before doing any local setup
	if h.cfg.OSAEnabled && h.osaClient == nil {
		slog.Warn("OSA unavailable, routing to local orchestrator")
	}

	// Pre-stream setup (ID parsing, conversation, history, model, agent config)
	setup, ok := h.prepareStream(c, ctx, req, user.ID, user.Name, focusModeStr)
	if !ok {
		return // HTTP error already written by prepareStream
	}

	// Try OSA routing now that we have conversationID
	if result := h.tryOSARouting(c, ctx, req, user.ID, setup.conversationID); result.handled {
		return
	}

	// Set streaming headers
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("X-Conversation-Id", uuidToString(setup.conversationID))
	c.Header("X-Agent-Type", string(setup.agentType))
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	startTime := time.Now()

	streamCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// Augment last user message with search results for non-Claude models
	if !strings.Contains(strings.ToLower(setup.model), "claude") &&
		setup.searchContextText != "" &&
		len(setup.chatMessages) > 0 {
		for i := len(setup.chatMessages) - 1; i >= 0; i-- {
			if strings.EqualFold(setup.chatMessages[i].Role, "user") {
				setup.chatMessages[i].Content = fmt.Sprintf(`Based on web search:

%s

---
Question: %s

INSTRUCTIONS:
1. Provide a comprehensive, detailed answer based on the search results above
2. Be thorough - do NOT stop mid-sentence
3. CRITICAL: You MUST end your response with a "## Sources" section
4. In the Sources section, list ALL sources you referenced as markdown links

Example ending:
## Sources
- [Source Title 1](url1)
- [Source Title 2](url2)`, setup.searchContextText, setup.chatMessages[i].Content)
				break
			}
		}
	}

	// Compute session health (cognitive load metrics)
	var sessionHealthHint string
	if h.sessionHealthSvc != nil && setup.convUUID != nil {
		var sessionStart time.Time
		if len(setup.messages) > 0 {
			sessionStart = setup.messages[0].CreatedAt.Time
		} else {
			sessionStart = time.Now()
		}
		health, err := h.sessionHealthSvc.ComputeHealth(ctx, setup.convUUID.String(), len(setup.messages), sessionStart)
		if err == nil && health.Hint != "" {
			sessionHealthHint = health.Hint
			slog.Debug("ChatV2: Session health computed",
				"flip_rate", health.ModeFlipRate,
				"msg_freq", health.MessageFrequency,
				"confused", health.IsConfused,
				"frustrated", health.IsFrustrated,
				"hint", health.Hint,
			)
		}
	}

	// Build agent input
	input := agents.AgentInput{
		Messages:       setup.chatMessages,
		Context:        setup.tieredCtx,
		FocusMode:      "",
		ConversationID: *setup.convUUID,
		UserID:         user.ID,
		UserName:       user.Name,
		MemoryContext:  setup.memoryContextStr,
		RoleContext:    setup.roleContextStr,
		SignalEnvelope: &setup.signalEnvelope,
	}
	if req.FocusMode != nil {
		input.FocusMode = *req.FocusMode
	}
	if sessionHealthHint != "" {
		if input.MemoryContext != "" {
			input.MemoryContext += "\n\n"
		}
		input.MemoryContext += "[Session Health] " + sessionHealthHint
	}
	slog.Debug("ChatV2: AgentInput created",
		"memory_context_len", len(setup.memoryContextStr),
		"role_context_len", len(setup.roleContextStr),
	)

	// Run agent (with or without COT)
	var events <-chan streaming.StreamEvent
	var errs <-chan error
	if setup.useCOT && setup.cotOrchestrator != nil {
		events, errs, _ = setup.cotOrchestrator.ProcessWithCOT(
			streamCtx, input, user.ID, user.Name, setup.convUUID, setup.llmOptions)
		c.Header("X-COT-Enabled", "true")
	} else {
		events, errs = setup.agent.RunWithTools(streamCtx, input)
	}

	// SSE stream state
	var fullResponse string
	var detectedArtifacts []streaming.Artifact
	var firstTokenReceived bool
	var pendingArtifact pendingArtifactState
	var signalSent bool
	var thinkingEventSent bool
	var insideThinking bool
	var thinkingStartSent bool
	var thinkingEndSent bool
	var thinkingContent string

	slog.Debug("ChatV2: Starting stream", "agent", setup.agentType)

	c.Stream(func(w io.Writer) bool {
		// Emit signal classification as the very first SSE event
		if !signalSent {
			signalSent = true
			writeSSEEvent(w, streaming.StreamEvent{
				Type: "signal_classified",
				Data: map[string]any{
					"mode":       string(setup.signalEnvelope.Mode),
					"genre":      string(setup.signalEnvelope.Genre),
					"doc_type":   setup.signalEnvelope.DocType,
					"weight":     setup.signalEnvelope.Weight,
					"confidence": setup.signalEnvelope.Confidence,
				},
			})
		}

		// Send initial thinking event
		if !thinkingEventSent {
			thinkingEventSent = true
			writeSSEEvent(w, streaming.StreamEvent{
				Type: streaming.EventTypeThinking,
				Data: streaming.ThinkingStep{
					Step:      "analyzing",
					Content:   "Processing your request...",
					Agent:     string(setup.agentType),
					Completed: false,
				},
			})
			if setup.searchResultCount > 0 {
				writeSSEEvent(w, streaming.StreamEvent{
					Type: streaming.EventTypeThinking,
					Data: streaming.ThinkingStep{
						Step:      "search_complete",
						Content:   fmt.Sprintf("Found %d sources from web search", setup.searchResultCount),
						Agent:     string(setup.agentType),
						Completed: true,
					},
				})
			}
		}

		select {
		case event, ok := <-events:
			if !ok {
				slog.Debug("ChatV2: Stream ended", "responseLen", len(fullResponse))
				return h.handleStreamEnd(ctx, w, req, fullResponse, detectedArtifacts, pendingArtifact,
					thinkingContent, startTime, setup.messages, provider, setup.model)
			}

			if !firstTokenReceived {
				firstTokenReceived = true
				writeSSEEvent(w, streaming.StreamEvent{
					Type: streaming.EventTypeThinking,
					Data: streaming.ThinkingStep{
						Step:      "responding",
						Content:   "Generating response...",
						Agent:     string(setup.agentType),
						Completed: true,
					},
				})
			}

			switch event.Type {
			case streaming.EventTypeToken:
				tokenContent := event.Content
				fullResponse += tokenContent
				thinkingContent, insideThinking, thinkingStartSent, thinkingEndSent = h.processThinkingToken(
					w, tokenContent, fullResponse, thinkingContent,
					insideThinking, thinkingStartSent, thinkingEndSent,
					string(setup.agentType),
				)
				pendingArtifact = processArtifactMarker(w, fullResponse, pendingArtifact)
				// Write token to chat only if NOT in artifact or thinking mode
				if pendingArtifact.title == "" && !insideThinking {
					if !strings.Contains(tokenContent, "<thinking>") &&
						!strings.Contains(tokenContent, "</thinking>") {
						writeSSEEvent(w, event)
					}
				}

			case streaming.EventTypeThinkingStart:
				if !thinkingStartSent {
					thinkingStartSent = true
					insideThinking = true
					slog.Debug("ChatV2: Thinking started (from detector)")
				}
				writeSSEEvent(w, event)

			case streaming.EventTypeThinkingChunk:
				if data, ok := event.Data.(map[string]interface{}); ok {
					if content, ok := data["content"].(string); ok {
						thinkingContent += content
					}
				}
				writeSSEEvent(w, event)

			case streaming.EventTypeThinkingEnd:
				thinkingEndSent = true
				insideThinking = false
				slog.Debug("ChatV2: Thinking ended (from detector)", "chars", len(thinkingContent))
				writeSSEEvent(w, event)

			case streaming.EventTypeArtifactStart:
				writeSSEEvent(w, event)

			case streaming.EventTypeArtifactComplete:
				if artifact, ok := event.Data.(streaming.Artifact); ok {
					detectedArtifacts = append(detectedArtifacts, artifact)
				}
				writeSSEEvent(w, event)

			case streaming.EventTypeArtifactError:
				writeSSEEvent(w, event)

			case streaming.EventTypeDone:
				slog.Debug("ChatV2: EventTypeDone received", "responseLen", len(fullResponse))
				return h.handleStreamEnd(ctx, w, req, fullResponse, detectedArtifacts, pendingArtifact,
					thinkingContent, startTime, setup.messages, provider, setup.model)

			default:
				writeSSEEvent(w, event)
			}
			return true

		case err := <-errs:
			slog.Debug("ChatV2: Error channel received", "err", err, "responseLen", len(fullResponse))
			if err != nil {
				slog.Error("ChatV2: Error details", "err", err)
				writeSSEEvent(w, streaming.StreamEvent{
					Type:    streaming.EventTypeError,
					Content: err.Error(),
				})
			} else {
				// nil error = stream completed via error channel
				if len(detectedArtifacts) == 0 && len(fullResponse) > 200 {
					if artifact := detectStructuredArtifact(fullResponse, req.Message); artifact != nil {
						detectedArtifacts = append(detectedArtifacts, *artifact)
						writeSSEEvent(w, streaming.StreamEvent{
							Type: streaming.EventTypeArtifactComplete,
							Data: *artifact,
						})
						slog.Debug("ChatV2: Auto-artifact on error channel",
							"title", artifact.Title, "type", artifact.Type)
					}
				}
				if req.StructuredOutput != nil && *req.StructuredOutput && h.blockMapper != nil {
					if doc, err := h.blockMapper.ParseMarkdown(ctx, stripThinkingTags(fullResponse), nil); err == nil {
						writeSSEEvent(w, streaming.StreamEvent{Type: streaming.EventTypeBlocks, Data: doc})
						slog.Debug("ChatV2: Structured blocks on channel close", "totalBlocks", len(doc.Blocks))
					}
				}
				sendUsageEvent(w, startTime, req.Message, setup.messages, fullResponse, provider, setup.model, len(thinkingContent)/4)
			}
			return false

		case <-streamCtx.Done():
			slog.Debug("ChatV2: Context done", "reason", streamCtx.Err())
			return false
		}
	})

	// Post-stream: persist all side effects
	h.postProcessStream(ctx, postProcessParams{
		req:               req,
		userID:            user.ID,
		conversationID:    setup.conversationID,
		convUUID:          setup.convUUID,
		contextID:         setup.contextID,
		projectID:         setup.projectID,
		nodeID:            setup.nodeID,
		contextIDs:        setup.contextIDs,
		messages:          setup.messages,
		chatMessages:      setup.chatMessages,
		detectedArtifacts: detectedArtifacts,
		fullResponse:      fullResponse,
		thinkingContent:   thinkingContent,
		pendingArtifact:   pendingArtifact,
		agentType:         setup.agentType,
		focusModeStr:      focusModeStr,
		model:             setup.model,
		startTime:         startTime,
	})
}

// handleStreamEnd handles the terminal state of the SSE stream: emits pending artifacts,
// structured blocks, and the usage/done event. Returns false to end the stream.
func (h *ChatHandler) handleStreamEnd(
	ctx context.Context,
	w io.Writer,
	req SendMessageRequest,
	fullResponse string,
	detectedArtifacts []streaming.Artifact,
	pendingArtifact pendingArtifactState,
	thinkingContent string,
	startTime time.Time,
	messages []sqlc.Message,
	provider, model string,
) bool {
	if pendingArtifact.title != "" && pendingArtifact.contentStart > 0 {
		artifactContent := fullResponse
		if pendingArtifact.contentStart < len(fullResponse) {
			artifactContent = fullResponse[pendingArtifact.contentStart:]
		}
		artifactContent = stripArtifactInstructions(artifactContent)
		if len(artifactContent) > 50 {
			writeSSEEvent(w, streaming.StreamEvent{
				Type: streaming.EventTypeArtifactComplete,
				Data: streaming.Artifact{
					Type:    pendingArtifact.artifactType,
					Title:   pendingArtifact.title,
					Content: artifactContent,
				},
			})
			slog.Debug("ChatV2: Artifact complete sent", "title", pendingArtifact.title, "len", len(artifactContent))
		}
	} else if len(detectedArtifacts) == 0 {
		if artifact := detectStructuredArtifact(fullResponse, req.Message); artifact != nil {
			writeSSEEvent(w, streaming.StreamEvent{
				Type: streaming.EventTypeArtifactComplete,
				Data: *artifact,
			})
			slog.Debug("ChatV2: Auto-artifact detected",
				"title", artifact.Title, "type", artifact.Type, "len", len(artifact.Content))
		}
	}

	if req.StructuredOutput != nil && *req.StructuredOutput && h.blockMapper != nil {
		if doc, err := h.blockMapper.ParseMarkdown(ctx, stripThinkingTags(fullResponse), nil); err == nil {
			writeSSEEvent(w, streaming.StreamEvent{Type: streaming.EventTypeBlocks, Data: doc})
			slog.Debug("ChatV2: Structured blocks sent", "totalBlocks", len(doc.Blocks))
		}
	}

	sendUsageEvent(w, startTime, req.Message, messages, fullResponse, provider, model, len(thinkingContent)/4)
	return false
}

// processThinkingToken handles thinking tag detection and SSE emission for a single token.
// Returns updated thinking state.
func (h *ChatHandler) processThinkingToken(
	w io.Writer,
	tokenContent string,
	fullResponse string,
	thinkingContent string,
	insideThinking bool,
	thinkingStartSent bool,
	thinkingEndSent bool,
	agentTypeStr string,
) (newThinkingContent string, newInsideThinking bool, newThinkingStartSent bool, newThinkingEndSent bool) {
	newThinkingContent = thinkingContent
	newInsideThinking = insideThinking
	newThinkingStartSent = thinkingStartSent
	newThinkingEndSent = thinkingEndSent

	if thinkingEndSent {
		return
	}

	startMatch := thinkingStartRe.FindStringIndex(fullResponse)
	endMatch := thinkingEndRe.FindStringIndex(fullResponse)

	if !insideThinking && startMatch != nil && endMatch == nil {
		newInsideThinking = true
		if !thinkingStartSent {
			newThinkingStartSent = true
			writeSSEEvent(w, streaming.StreamEvent{
				Type: streaming.EventTypeThinkingStart,
				Data: map[string]interface{}{
					"step":  1,
					"agent": agentTypeStr,
				},
			})
			slog.Debug("ChatV2: Thinking started", "tag", fullResponse[startMatch[0]:startMatch[1]])
		}
	}

	if newInsideThinking && endMatch != nil {
		newInsideThinking = false
		newThinkingEndSent = true
		if startMatch != nil && startMatch[1] < endMatch[0] {
			newThinkingContent = fullResponse[startMatch[1]:endMatch[0]]
		}
		writeSSEEvent(w, streaming.StreamEvent{
			Type: streaming.EventTypeThinkingEnd,
			Data: map[string]interface{}{
				"step":    1,
				"content": sanitizeContent(newThinkingContent),
			},
		})
		slog.Debug("ChatV2: Thinking ended", "chars", len(newThinkingContent))
	} else if newInsideThinking {
		writeSSEEvent(w, streaming.StreamEvent{
			Type: streaming.EventTypeThinkingChunk,
			Data: map[string]interface{}{
				"content": sanitizeContent(tokenContent),
				"step":    1,
			},
		})
	}
	return
}

// processArtifactMarker checks for ARTIFACT_START:: tool-call markers in the accumulated
// response and emits an artifact_start SSE event if a new marker is found.
func processArtifactMarker(w io.Writer, fullResponse string, pending pendingArtifactState) pendingArtifactState {
	if !strings.Contains(fullResponse, "ARTIFACT_START::") || pending.title != "" {
		return pending
	}
	idx := strings.Index(fullResponse, "ARTIFACT_START::")
	if idx == -1 {
		return pending
	}
	markerEnd := strings.Index(fullResponse[idx:], "::Now write")
	if markerEnd == -1 {
		return pending
	}
	parts := strings.Split(fullResponse[idx:idx+markerEnd], "::")
	if len(parts) < 3 {
		return pending
	}
	pending.artifactType = parts[1]
	pending.title = parts[2]
	pending.contentStart = len(fullResponse)
	writeSSEEvent(w, streaming.StreamEvent{
		Type: streaming.EventTypeArtifactStart,
		Data: map[string]string{
			"type":  pending.artifactType,
			"title": pending.title,
		},
	})
	slog.Debug("ChatV2: Artifact started", "title", pending.title, "type", pending.artifactType)
	return pending
}
