package handlers

import (
	"context"
	"io"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	osa "github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// osaRoutingResult signals whether OSA handled the request and the handler should return.
type osaRoutingResult struct {
	handled bool
}

// tryOSARouting attempts to route the request through the OSA orchestrator.
// If OSA is available and handles the request, it streams events to the client and returns
// handled=true. If OSA is unavailable or disabled, it returns handled=false so the caller
// can fall through to local agent routing.
func (h *ChatHandler) tryOSARouting(
	c *gin.Context,
	ctx context.Context,
	req SendMessageRequest,
	userID string,
	conversationID pgtype.UUID,
) osaRoutingResult {
	if h.osaClient == nil || !h.cfg.OSAEnabled {
		return osaRoutingResult{handled: false}
	}

	sessionID := uuid.NewString()

	var osaWorkspaceID uuid.UUID
	if req.WorkspaceID != nil && *req.WorkspaceID != "" {
		if parsed, err := uuid.Parse(*req.WorkspaceID); err == nil {
			osaWorkspaceID = parsed
		}
	}

	// OSA's SSE stream is a persistent session — it does NOT close automatically
	// after a single response. We use a dedicated context so we can cancel the
	// SSE connection once Orchestrate completes, which causes the SDK scanner to
	// exit, closes the events channel, and allows mapOSAEventsToStreamEvents to
	// inject the Done event that auto-completes the stream.
	streamCtx, cancelStream := context.WithCancel(ctx)

	osaEvents, streamErr := h.osaClient.Stream(streamCtx, sessionID)
	if streamErr != nil {
		cancelStream()
		slog.Warn("OSA stream unavailable, falling back to local agents", "error", streamErr)
		return osaRoutingResult{handled: false}
	}

	// Kick off orchestration in background — events flow through the stream channel.
	// Cancel the stream context when Orchestrate finishes so the SSE connection
	// closes and the Done event propagates to the client.
	go func() {
		defer cancelStream()
		userUUID, _ := uuid.Parse(userID)
		osaReq := &osa.OrchestrateRequest{
			UserID:      userUUID,
			Input:       req.Message,
			SessionID:   sessionID,
			WorkspaceID: osaWorkspaceID,
		}
		if _, err := h.osaClient.Orchestrate(ctx, osaReq); err != nil {
			slog.Error("OSA orchestration failed", "error", err, "session_id", sessionID)
		}
	}()

	bosEvents := mapOSAEventsToStreamEvents(osaEvents)

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("X-Conversation-Id", uuidToString(conversationID))
	c.Header("X-OSA-Routing", "true")
	c.Header("X-OSA-Session-Id", sessionID)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var fullResp string
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-bosEvents:
			if !ok {
				return false
			}
			if event.Type == streaming.EventTypeToken {
				fullResp += event.Content
			}
			writeSSEEvent(w, event)
			return event.Type != streaming.EventTypeDone
		case <-ctx.Done():
			return false
		}
	})

	// Persist assistant response asynchronously
	if fullResp != "" {
		go func() {
			bgCtx := context.Background()
			bgQueries := sqlc.New(h.pool)
			bgQueries.CreateMessage(bgCtx, sqlc.CreateMessageParams{
				ConversationID:  conversationID,
				Role:            sqlc.MessageroleASSISTANT,
				Content:         fullResp,
				MessageMetadata: nil,
			})
		}()
	}

	slog.Info("OSA handled chat request", "session_id", sessionID, "response_len", len(fullResp))
	return osaRoutingResult{handled: true}
}
