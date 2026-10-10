// Package handlers — chat_engine_slash.go
//
// Engine-backed slash commands. When a chat message starts with one of
// these commands, we skip the LLM entirely and stream the engine's typed
// answer (a RAG envelope or a recall result) back as one SSE chunk plus
// a done event.
//
// Why server-side dispatch (not client-side)? The frontend stays
// command-agnostic — `/ask`, `/who`, `/recall` look identical to `/plan`
// or `/proposal` from the chat UI's POV. The server decides at runtime
// whether to route to the LLM (agent) or the engine (recall/RAG). That
// also means engine commands work from every chat surface (main chat,
// dock popup, command palette) without any client coordination.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/optimalengine"
)

// engineSlashCommands lists every command that bypasses the LLM and goes
// straight to the engine. Keep in sync with the engineSlash dispatch
// switch below — adding a name here without a case is harmless (falls
// through to LLM); adding a case without listing here means the command
// won't be recognized.
var engineSlashCommands = map[string]struct{}{
	"ask":     {},
	"recall":  {},
	"who":     {},
	"when":    {},
	"where":   {},
	"owns":    {},
	"actions": {},
}

// engineSlashHandled tries to dispatch the request as an engine command.
// Returns true when the command was engine-routed (caller should NOT
// continue to the LLM agent path); false when the command isn't ours
// (caller continues with the existing slash dispatch).
//
// The function takes care of conversation creation, persisting both
// the user message and the engine response, and streaming the response
// back as SSE so the frontend's existing message renderer just works.
func (h *ChatHandler) engineSlashHandled(c *gin.Context, user *middleware.BetterAuthUser, req SendMessageRequest) bool {
	if req.Command == nil {
		return false
	}
	cmd := strings.ToLower(*req.Command)
	if _, ok := engineSlashCommands[cmd]; !ok {
		return false
	}

	engine := h.engineClient()
	if engine == nil || !engine.Enabled() {
		// Engine isn't wired — let the LLM path handle it (better than
		// returning 503; the model can usually respond to "/ask <q>"
		// even without engine grounding).
		return false
	}

	ctx := c.Request.Context()

	// Run the engine call. Each command extracts whatever args it needs
	// from the message body; format is intentionally permissive so users
	// can type natural language (`/who owns the platform module`) instead
	// of memorizing a key=value DSL.
	answer, err := h.runEngineSlash(ctx, engine, cmd, req.Message, user.ID)
	if err != nil {
		// Surface the failure as a normal SSE error event so the chat UI
		// shows it instead of a generic network error. The client treats
		// `event: error` like any other rendering hiccup.
		h.streamEngineError(c, cmd, err)
		return true
	}

	if err := h.persistEngineSlash(ctx, user, req, cmd, answer); err != nil {
		// Persistence failure isn't fatal for the user-visible answer —
		// they still see the response, but we log it for triage.
		slog.WarnContext(ctx, "engine slash: persistence failed",
			"command", cmd, "user_id", user.ID, "error", err)
	}

	h.streamEngineAnswer(c, cmd, answer)
	return true
}

// runEngineSlash dispatches to the right engine method per command and
// returns the textual answer the chat UI should render.
//
// The engine returns a structured AskResponse (a map). We pull the
// `answer` / `body` field that every recall + RAG endpoint produces;
// when the response shape varies we fall back to JSON-stringifying the
// whole envelope so nothing is lost.
func (h *ChatHandler) runEngineSlash(ctx context.Context, engine *optimalengine.Client, cmd, message, userID string) (string, error) {
	// Tighter timeout than the engine's default (30s) — slash commands
	// are interactive and a hung engine should fail fast so the chat UI
	// can fall back to the LLM path on retry.
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch cmd {
	case "ask", "recall":
		// `/ask <question>` → full RAG. `/recall` is an alias optimized
		// for "what do we know about X" type queries; same backend.
		resp, err := engine.Ask(ctx, optimalengine.AskRequest{
			Query:     message,
			Workspace: "default",
			Format:    "markdown",
			Bandwidth: "medium",
		})
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, "I couldn't find anything in the workspace memory for that question."), nil

	case "who":
		// "who owns X" / "who manages Y" — the engine builds the
		// canonical question itself, so we just pass the raw topic.
		resp, err := engine.RecallWho(ctx, message, "owner", "default")
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, "I don't know of an owner for that yet."), nil

	case "when":
		resp, err := engine.RecallWhen(ctx, message, "default")
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, "I don't have a schedule for that yet."), nil

	case "where":
		resp, err := engine.RecallWhere(ctx, message, "default")
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, "I don't see that recorded anywhere yet."), nil

	case "owns":
		// `/owns me` (or any actor name in the message). Empty message
		// = "what am I committed to" — defaults to the calling user.
		actor := strings.TrimSpace(message)
		if actor == "" || strings.EqualFold(actor, "me") {
			actor = userID
		}
		resp, err := engine.RecallOwns(ctx, actor, "default")
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, fmt.Sprintf("No open commitments tracked for %s.", actor)), nil

	case "actions":
		// `/actions about <topic>` or just `/actions` (= recent actions
		// across the workspace). Topic extraction is intentionally
		// loose — the engine handles natural language.
		topic := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(message), "about "))
		resp, err := engine.RecallActions(ctx, "", topic, "", "default")
		if err != nil {
			return "", err
		}
		return formatEngineResponse(resp, "No recent actions or decisions found."), nil
	}

	return "", fmt.Errorf("unrecognized engine command: %s", cmd)
}

// formatEngineResponse extracts the human-readable text from an engine
// response map. Falls through several common shapes — answer, body,
// summary, content — and finally JSON-stringifies the whole map if no
// known field is present (never lose information).
func formatEngineResponse(resp optimalengine.AskResponse, empty string) string {
	if resp == nil {
		return empty
	}
	for _, key := range []string{"answer", "body", "summary", "content", "envelope"} {
		if v, ok := resp[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	// Some recall endpoints return the answer under a "result" object.
	if r, ok := resp["result"].(map[string]interface{}); ok {
		for _, key := range []string{"answer", "body", "summary"} {
			if v, ok := r[key]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
	}
	// Last resort: pretty-print so the user sees the raw structure
	// rather than a generic empty message — better to over-share than
	// silently hide signal.
	if b, err := json.MarshalIndent(resp, "", "  "); err == nil && len(b) > 2 {
		return "```json\n" + string(b) + "\n```"
	}
	return empty
}

// persistEngineSlash saves the user message + engine response into the
// conversation history so reloading the chat shows them. Mirrors the
// shape the LLM dispatch path produces so the chat UI doesn't need a
// special case for engine messages.
func (h *ChatHandler) persistEngineSlash(ctx context.Context, user *middleware.BetterAuthUser, req SendMessageRequest, cmd, answer string) error {
	queries := sqlc.New(h.pool)

	var conversationID pgtype.UUID
	if req.ConversationID != nil {
		parsed, err := uuid.Parse(*req.ConversationID)
		if err != nil {
			return fmt.Errorf("conversation_id: %w", err)
		}
		conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
	} else {
		title := fmt.Sprintf("/%s: %s", cmd, truncateString(req.Message, 40))
		conv, err := queries.CreateConversation(ctx, sqlc.CreateConversationParams{
			UserID: user.ID,
			Title:  &title,
		})
		if err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		conversationID = conv.ID
	}

	userMsg := fmt.Sprintf("/%s %s", cmd, req.Message)
	if _, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  conversationID,
		Role:            sqlc.MessageroleUSER,
		Content:         userMsg,
		MessageMetadata: nil,
	}); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// Tag the assistant message with the engine source so the UI (and
	// future analytics) can distinguish engine-grounded answers from
	// LLM completions.
	meta, _ := json.Marshal(map[string]string{
		"source":  "optimal-engine",
		"command": cmd,
	})
	if _, err := queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID:  conversationID,
		Role:            sqlc.MessageroleASSISTANT,
		Content:         answer,
		MessageMetadata: meta,
	}); err != nil {
		return fmt.Errorf("save assistant message: %w", err)
	}
	return nil
}

// streamEngineAnswer emits the engine response as Server-Sent Events
// matching the chat streaming protocol (chunk events with content +
// final done event). The chat client renders these identically to LLM
// streaming output so no UI change is needed.
func (h *ChatHandler) streamEngineAnswer(c *gin.Context, cmd, answer string) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	// One chunk event with the full answer (engine answers are already
	// composed end-to-end; no point in artificially streaming them
	// token-by-token). The frontend's chunk handler appends to the
	// rendered message body.
	chunk, _ := json.Marshal(map[string]string{
		"type":    "chunk",
		"content": answer,
		"source":  "optimal-engine",
		"command": cmd,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)

	done, _ := json.Marshal(map[string]string{
		"type":    "done",
		"source":  "optimal-engine",
		"command": cmd,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", done)
	c.Writer.Flush()
}

// streamEngineError emits an error SSE event. The chat client renders
// these as a system message rather than aborting the stream, so the
// user sees what went wrong and can retry.
func (h *ChatHandler) streamEngineError(c *gin.Context, cmd string, err error) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	payload, _ := json.Marshal(map[string]string{
		"type":    "error",
		"message": err.Error(),
		"source":  "optimal-engine",
		"command": cmd,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	c.Writer.Flush()
}
