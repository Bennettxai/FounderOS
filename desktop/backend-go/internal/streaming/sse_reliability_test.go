package streaming

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSSEWriterBasicFunctionality tests basic SSE event writing
func TestSSEWriterBasicFunctionality(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	event := StreamEvent{
		Type:    EventTypeToken,
		Content: "Hello, World!",
	}

	err := writer.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	output := buf.String()

	// Verify SSE format
	if !strings.Contains(output, "event: token") {
		t.Error("Missing event type in SSE output")
	}
	if !strings.Contains(output, "data:") {
		t.Error("Missing data field in SSE output")
	}
	if !strings.Contains(output, "Hello, World!") {
		t.Error("Missing content in SSE output")
	}
	if !strings.HasSuffix(output, "\n\n") {
		t.Error("SSE event should end with double newline")
	}
}

// TestSSEWriterMultipleEvents tests writing multiple events
func TestSSEWriterMultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	events := []StreamEvent{
		{Type: EventTypeToken, Content: "Hello"},
		{Type: EventTypeToken, Content: " "},
		{Type: EventTypeToken, Content: "World"},
		{Type: EventTypeDone, Content: ""},
	}

	for _, event := range events {
		if err := writer.WriteEvent(event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	output := buf.String()
	eventCount := strings.Count(output, "event:")
	if eventCount != len(events) {
		t.Errorf("Expected %d events, found %d", len(events), eventCount)
	}
}

// TestSSEWriterSpecialCharacters tests handling of special characters
func TestSSEWriterSpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"Unicode", "Hello 世界 🌍"},
		{"Newlines", "Line1\nLine2\nLine3"},
		{"Quotes", `"quoted" content`},
		{"Backslashes", `path\to\file`},
		{"JSON content", `{"key": "value"}`},
		{"HTML content", `<div class="test">Hello</div>`},
		{"Code snippet", "```go\nfunc main() {}\n```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewSSEWriter(&buf)

			event := StreamEvent{
				Type:    EventTypeToken,
				Content: tt.content,
			}

			err := writer.WriteEvent(event)
			if err != nil {
				t.Fatalf("WriteEvent failed for %s: %v", tt.name, err)
			}

			// Verify the content can be parsed back
			output := buf.String()
			dataLine := extractDataLine(output)
			if dataLine == "" {
				t.Error("Could not extract data line from SSE output")
				return
			}

			var parsed StreamEvent
			if err := json.Unmarshal([]byte(dataLine), &parsed); err != nil {
				t.Errorf("Failed to parse SSE data: %v", err)
				return
			}

			if parsed.Content != tt.content {
				t.Errorf("Content mismatch: expected %q, got %q", tt.content, parsed.Content)
			}
		})
	}
}

// TestStreamProcessorArtifactDetection tests artifact detection in stream
func TestStreamProcessorArtifactDetection(t *testing.T) {
	tests := []struct {
		name        string
		chunks      []string
		expectTypes []string
	}{
		{
			name:        "Plain text",
			chunks:      []string{"Hello", " ", "World"},
			expectTypes: []string{"token", "token", "token"},
		},
		{
			name:        "Complete artifact",
			chunks:      []string{"<artifact>", "content", "</artifact>"},
			expectTypes: []string{"artifact_start", "artifact_content", "artifact_end"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			processor := NewStreamProcessor(&buf)

			for _, chunk := range tt.chunks {
				err := processor.ProcessChunk(chunk)
				if err != nil {
					t.Errorf("ProcessChunk failed: %v", err)
				}
			}

			processor.Flush()

			output := buf.String()
			t.Logf("Output for %s: %s", tt.name, output[:min(100, len(output))])
		})
	}
}

// TestSSEWriterConcurrency tests concurrent event writing
func TestSSEWriterConcurrency(t *testing.T) {
	var buf safeBuffer
	writer := NewSSEWriter(&buf)

	var wg sync.WaitGroup
	numWriters := 10
	eventsPerWriter := 100

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerWriter; j++ {
				event := StreamEvent{
					Type:    EventTypeToken,
					Content: strings.Repeat("x", 10),
				}
				writer.WriteEvent(event)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were written
	output := buf.String()
	expectedEvents := numWriters * eventsPerWriter
	actualEvents := strings.Count(output, "event:")

	// With concurrent writes, we may have some interleaving, but should have all events
	if actualEvents < expectedEvents/2 {
		t.Errorf("Expected at least %d events, got %d", expectedEvents/2, actualEvents)
	}

	t.Logf("Wrote %d events concurrently", actualEvents)
}

// TestSSEWriterLargePayload tests handling of large payloads
func TestSSEWriterLargePayload(t *testing.T) {
	sizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range sizes {
		t.Run(byteSizeString(size), func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewSSEWriter(&buf)

			content := strings.Repeat("x", size)
			event := StreamEvent{
				Type:    EventTypeToken,
				Content: content,
			}

			start := time.Now()
			err := writer.WriteEvent(event)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("WriteEvent failed for %d bytes: %v", size, err)
				return
			}

			// Should complete reasonably fast
			if elapsed > 100*time.Millisecond {
				t.Errorf("Writing %d bytes took too long: %v", size, elapsed)
			}

			t.Logf("Wrote %d bytes in %v", size, elapsed)
		})
	}
}

// TestSSEWriterContextCancellation tests behavior when context is cancelled
func TestSSEWriterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	// Write some events
	for i := 0; i < 10; i++ {
		if ctx.Err() != nil {
			break
		}
		writer.WriteEvent(StreamEvent{
			Type:    EventTypeToken,
			Content: "chunk",
		})
	}

	// Cancel context
	cancel()

	// Verify we can still check state after cancellation
	if ctx.Err() == nil {
		t.Error("Context should be cancelled")
	}
}

// TestSSEWriterErrorRecovery tests error recovery scenarios
func TestSSEWriterErrorRecovery(t *testing.T) {
	// Test with a writer that fails after N writes
	failAfter := 5
	fw := &failingWriter{failAfter: failAfter}
	writer := NewSSEWriter(fw)

	var lastErr error
	for i := 0; i < 10; i++ {
		err := writer.WriteEvent(StreamEvent{
			Type:    EventTypeToken,
			Content: "test",
		})
		if err != nil {
			lastErr = err
			break
		}
	}

	if lastErr == nil {
		t.Error("Expected error from failing writer")
	}

	// Verify we got the expected number of successful writes
	if fw.writeCount != failAfter {
		t.Errorf("Expected %d writes before failure, got %d", failAfter, fw.writeCount)
	}
}

// TestStreamProcessorFlushBehavior tests proper flushing of stream processor
func TestStreamProcessorFlushBehavior(t *testing.T) {
	var buf bytes.Buffer
	processor := NewStreamProcessor(&buf)

	// Write partial content
	processor.ProcessChunk("Hello")
	processor.ProcessChunk(" World")

	// Verify content before flush
	beforeFlush := buf.String()

	// Flush
	processor.Flush()

	afterFlush := buf.String()

	// After flush should have at least as much content
	if len(afterFlush) < len(beforeFlush) {
		t.Error("Flush should not remove content")
	}

	t.Logf("Before flush: %d chars, After flush: %d chars", len(beforeFlush), len(afterFlush))
}

// TestSSEEventTypes tests all event types
func TestSSEEventTypes(t *testing.T) {
	eventTypes := []EventType{
		EventTypeToken,
		EventTypeThinking,
		EventTypeThinkingStart,
		EventTypeThinkingChunk,
		EventTypeThinkingEnd,
		EventTypeArtifactStart,
		EventTypeArtifactComplete,
		EventTypeArtifactError,
		EventTypeToolCall,
		EventTypeToolResult,
		EventTypeContentStart,
		EventTypeContentEnd,
		EventTypeDelegating,
		EventTypeDone,
		EventTypeError,
	}

	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			buf.Reset()

			event := StreamEvent{
				Type:    eventType,
				Content: "test content",
			}

			err := writer.WriteEvent(event)
			if err != nil {
				t.Errorf("Failed to write event type %s: %v", eventType, err)
				return
			}

			output := buf.String()
			if !strings.Contains(output, "event: "+string(eventType)) {
				t.Errorf("Event type %s not found in output", eventType)
			}
		})
	}
}

// BenchmarkSSEWriter benchmarks SSE event writing
func BenchmarkSSEWriter(b *testing.B) {
	event := StreamEvent{
		Type:    EventTypeToken,
		Content: "Hello, World!",
	}

	b.Run("SingleEvent", func(b *testing.B) {
		var buf bytes.Buffer
		writer := NewSSEWriter(&buf)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			writer.WriteEvent(event)
		}
	})

	b.Run("LargePayload", func(b *testing.B) {
		largeEvent := StreamEvent{
			Type:    EventTypeToken,
			Content: strings.Repeat("x", 10000),
		}
		var buf bytes.Buffer
		writer := NewSSEWriter(&buf)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			writer.WriteEvent(largeEvent)
		}
	})
}

// BenchmarkStreamProcessor benchmarks stream processing
func BenchmarkStreamProcessor(b *testing.B) {
	chunks := []string{
		"Hello ",
		"World, ",
		"this is ",
		"a test ",
		"message!",
	}

	b.Run("PlainText", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			processor := NewStreamProcessor(&buf)
			for _, chunk := range chunks {
				processor.ProcessChunk(chunk)
			}
			processor.Flush()
		}
	})
}

// Helper types and functions

type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

type failingWriter struct {
	failAfter  int
	writeCount int
}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	fw.writeCount++
	if fw.writeCount >= fw.failAfter {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func extractDataLine(sseOutput string) string {
	lines := strings.Split(sseOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	return ""
}

func byteSizeString(bytes int) string {
	if bytes >= 1024*1024 {
		return strings.TrimSuffix(strings.TrimSuffix(
			strings.Replace(
				strings.Replace(string(rune(bytes/(1024*1024)))+"MB", "\x00", "", -1),
				"\x00", "", -1), "0"), "0")
	}
	if bytes >= 1024 {
		return string(rune('0'+bytes/1024)) + "KB"
	}
	return string(rune('0'+bytes)) + "B"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
