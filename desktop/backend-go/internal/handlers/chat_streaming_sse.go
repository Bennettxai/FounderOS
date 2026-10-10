package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// sanitizeContent replaces problematic Unicode characters with ASCII equivalents
func sanitizeContent(content string) string {
	// Replace Unicode bullet points with ASCII dashes
	content = strings.ReplaceAll(content, "\u2022", "-")   // BULLET
	content = strings.ReplaceAll(content, "\u25CF", "-")   // BLACK CIRCLE
	content = strings.ReplaceAll(content, "\u25CB", "-")   // WHITE CIRCLE
	content = strings.ReplaceAll(content, "\u25E6", "-")   // WHITE BULLET
	content = strings.ReplaceAll(content, "\u25AA", "-")   // BLACK SMALL SQUARE
	content = strings.ReplaceAll(content, "\u25B8", "-")   // BLACK RIGHT-POINTING SMALL TRIANGLE
	content = strings.ReplaceAll(content, "\u25BA", "-")   // BLACK RIGHT-POINTING POINTER
	content = strings.ReplaceAll(content, "\u2023", "-")   // TRIANGULAR BULLET
	content = strings.ReplaceAll(content, "\u2043", "-")   // HYPHEN BULLET
	content = strings.ReplaceAll(content, "\u2013", "-")   // EN DASH
	content = strings.ReplaceAll(content, "\u2014", "-")   // EM DASH
	content = strings.ReplaceAll(content, "\u201C", "\"")  // LEFT DOUBLE QUOTATION MARK
	content = strings.ReplaceAll(content, "\u201D", "\"")  // RIGHT DOUBLE QUOTATION MARK
	content = strings.ReplaceAll(content, "\u2018", "'")   // LEFT SINGLE QUOTATION MARK
	content = strings.ReplaceAll(content, "\u2019", "'")   // RIGHT SINGLE QUOTATION MARK
	content = strings.ReplaceAll(content, "\u2026", "...") // HORIZONTAL ELLIPSIS
	return content
}

// writeSSEEvent writes a streaming event in SSE format
func writeSSEEvent(w io.Writer, event streaming.StreamEvent) {
	// Sanitize content in the event
	if event.Content != "" {
		event.Content = sanitizeContent(event.Content)
	}
	if str, ok := event.Data.(string); ok {
		event.Data = sanitizeContent(str)
	}
	// Sanitize artifact content
	if artifact, ok := event.Data.(streaming.Artifact); ok {
		artifact.Content = sanitizeContent(artifact.Content)
		artifact.Title = sanitizeContent(artifact.Title)
		event.Data = artifact
	}
	// Sanitize map data (for thinking events)
	if mapData, ok := event.Data.(map[string]interface{}); ok {
		if content, exists := mapData["content"]; exists {
			if contentStr, isStr := content.(string); isStr {
				mapData["content"] = sanitizeContent(contentStr)
				event.Data = mapData
			}
		}
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// sendUsageEvent sends usage statistics as an SSE event
func sendUsageEvent(w io.Writer, startTime time.Time, userMessage string, messages []sqlc.Message, fullResponse string, provider string, model string, thinkingTokens int) {
	endTime := time.Now()
	inputChars := len(userMessage)
	for _, msg := range messages {
		inputChars += len(msg.Content)
	}
	outputChars := len(fullResponse)
	inputTokens := inputChars / 4
	outputTokens := outputChars / 4
	totalTokens := inputTokens + outputTokens + thinkingTokens
	durationMs := endTime.Sub(startTime).Milliseconds()
	tps := float64(0)
	if durationMs > 0 {
		tps = float64(outputTokens) / (float64(durationMs) / 1000)
	}
	estimatedCost := services.CalculateEstimatedCost(provider, model, inputTokens, outputTokens)

	usageData := map[string]interface{}{
		"input_tokens":    inputTokens,
		"output_tokens":   outputTokens,
		"thinking_tokens": thinkingTokens,
		"total_tokens":    totalTokens,
		"duration_ms":     durationMs,
		"tps":             tps,
		"provider":        provider,
		"model":           model,
		"estimated_cost":  estimatedCost,
	}

	event := streaming.StreamEvent{
		Type: streaming.EventTypeDone,
		Data: usageData,
	}
	writeSSEEvent(w, event)
}

// logSignal writes a signal_log row asynchronously. Called in a goroutine — zero latency impact.
// Returns the signal_log row ID for downstream use (e.g. classifier enrichment).
func logSignal(ctx context.Context, pool *pgxpool.Pool, userID string, convUUID *uuid.UUID, focusMode string, userMessage string, responseLen int, startTime time.Time) string {
	elapsed := time.Since(startTime).Milliseconds()

	// Truncate message preview
	preview := userMessage
	if len(preview) > 200 {
		preview = preview[:200]
	}

	// Normalize focus mode
	mode := "ASSIST"
	if focusMode != "" && focusMode != "nil" {
		mode = strings.ToUpper(focusMode)
	}

	var signalLogID string
	err := pool.QueryRow(ctx, `
		INSERT INTO signal_log (user_id, conversation_id, mode, signal_type, format, message_preview, response_length, latency_ms)
		VALUES ($1, $2, $3, 'chat', 'MARKDOWN', $4, $5, $6)
		RETURNING id
	`, userID, convUUID, mode, preview, responseLen, elapsed).Scan(&signalLogID)
	if err != nil {
		slog.Warn("logSignal failed", "error", err)
	}
	return signalLogID
}
