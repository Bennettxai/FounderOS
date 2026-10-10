package handlers

import (
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// mapOSAEventsToStreamEvents bridges an OSA SDK event channel into a BOS
// StreamEvent channel that the existing chat SSE writer can consume directly.
// The goroutine drains osaEvents and closes the returned channel when done.
func mapOSAEventsToStreamEvents(osaEvents <-chan osa.Event) <-chan streaming.StreamEvent {
	out := make(chan streaming.StreamEvent, 64)
	go func() {
		defer close(out)
		inThinking := false
		for evt := range osaEvents {
			mapped := mapSingleEvent(evt, &inThinking)
			if mapped != nil {
				out <- *mapped
			}
		}
		// Signal stream completion
		out <- streaming.StreamEvent{Type: streaming.EventTypeDone}
	}()
	return out
}

func mapSingleEvent(evt osa.Event, inThinking *bool) *streaming.StreamEvent {
	// Extract text from OSA streaming_token events (sent as various event types)
	text, _ := evt.Data["text"].(string)

	switch evt.Type {
	case osa.EventThinking:
		content, _ := evt.Data["content"].(string)
		step, _ := evt.Data["step"].(string)
		if step == "" {
			step = "analyzing"
		}
		return &streaming.StreamEvent{
			Type: streaming.EventTypeThinkingChunk,
			Data: streaming.ThinkingStep{
				Step:    step,
				Content: content,
				Agent:   "osa",
			},
		}

	case osa.EventResponse:
		content, _ := evt.Data["content"].(string)
		if content == "" {
			content, _ = evt.Data["output"].(string)
		}
		return &streaming.StreamEvent{
			Type:    streaming.EventTypeToken,
			Content: content,
		}

	case osa.EventSkillStarted:
		name, _ := evt.Data["skill"].(string)
		return &streaming.StreamEvent{
			Type: streaming.EventTypeToolCall,
			Data: streaming.ToolCallEvent{
				ToolName: name,
				Status:   "calling",
			},
		}

	case osa.EventSkillCompleted:
		name, _ := evt.Data["skill"].(string)
		result, _ := evt.Data["result"].(string)
		return &streaming.StreamEvent{
			Type: streaming.EventTypeToolResult,
			Data: streaming.ToolCallEvent{
				ToolName: name,
				Status:   "success",
				Result:   result,
			},
		}

	case osa.EventSkillFailed:
		name, _ := evt.Data["skill"].(string)
		errMsg, _ := evt.Data["error"].(string)
		return &streaming.StreamEvent{
			Type: streaming.EventTypeToolResult,
			Data: streaming.ToolCallEvent{
				ToolName: name,
				Status:   "error",
				Result:   errMsg,
			},
		}

	case osa.EventError:
		msg, _ := evt.Data["message"].(string)
		if msg == "" {
			msg, _ = evt.Data["error"].(string)
		}
		return &streaming.StreamEvent{
			Type:    streaming.EventTypeError,
			Content: fmt.Sprintf("OSA error: %s", msg),
		}

	case osa.EventConnected, osa.EventSignal:
		// Reset inThinking on a new connection to avoid stale state across reconnects.
		*inThinking = false
		return nil

	default:
		// Handle streaming_token, llm_request, system_event, and other OSA events.
		// OSA's streaming_token events carry text in the "text" field.
		if text == "" {
			// Also try "content" for other event types
			text, _ = evt.Data["content"].(string)
		}
		if text == "" {
			return nil
		}

		// Filter out model thinking content (e.g. qwen3 <think>...</think>).
		//
		// qwen3 can send thinking in several forms:
		//   1. A single token that IS exactly "<think>" or "</think>"
		//   2. A token that CONTAINS "<think>" or "</think>" embedded in other text
		//   3. Opening and closing tags within the same token
		//   4. Multi-token thinking spans that start with <think> and end with </think>
		//
		// We strip all <think>...</think> blocks from the token first, then apply
		// the cross-token inThinking state for spans that cross token boundaries.
		text = stripThinkBlocks(text, inThinking)
		if text == "" {
			return nil
		}

		return &streaming.StreamEvent{
			Type:    streaming.EventTypeToken,
			Content: text,
		}
	}
}

// stripThinkBlocks removes <think>...</think> content from a streaming token,
// correctly handling all the ways qwen3 (and similar models) may emit thinking:
//
//   - Token is exactly "<think>" or starts with it → enter thinking state, return ""
//   - Token is exactly "</think>" or starts with it → exit thinking state, return ""
//   - Token contains a complete <think>...</think> block inline → strip the block
//   - Token arrives while inThinking=true → discard until closing tag found
//
// inThinking is mutated to track cross-token thinking spans.
func stripThinkBlocks(text string, inThinking *bool) string {
	const openTag = "<think>"
	const closeTag = "</think>"

	var out strings.Builder

	for len(text) > 0 {
		if *inThinking {
			// We are inside a thinking span — scan forward for the closing tag.
			idx := strings.Index(text, closeTag)
			if idx == -1 {
				// No closing tag in this token; discard everything.
				return out.String()
			}
			// Found the closing tag — skip past it and exit thinking mode.
			*inThinking = false
			text = text[idx+len(closeTag):]
			// Trim a single leading newline that models often emit after </think>
			text = strings.TrimPrefix(text, "\n")
			continue
		}

		// Not currently in a thinking span — look for the next opening tag.
		idx := strings.Index(text, openTag)
		if idx == -1 {
			// No opening tag; emit the entire remaining text.
			out.WriteString(text)
			return out.String()
		}

		// Emit everything before the opening tag.
		out.WriteString(text[:idx])
		text = text[idx+len(openTag):]
		*inThinking = true
		// Continue the loop: the closing tag may be in the same token.
	}

	return out.String()
}
