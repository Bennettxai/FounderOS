package streaming

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SSEWriter handles writing Server-Sent Events to a response writer
type SSEWriter struct {
	w       io.Writer
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(w io.Writer) *SSEWriter {
	flusher, _ := w.(http.Flusher)
	return &SSEWriter{
		w:       w,
		flusher: flusher,
	}
}

// WriteEvent writes a StreamEvent as an SSE event
func (s *SSEWriter) WriteEvent(event StreamEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Write SSE format: event: type\ndata: json\n\n
	_, err = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event.Type, string(data))
	if err != nil {
		return err
	}

	s.Flush()
	return nil
}

// WriteToken writes a token event (shorthand for common case)
func (s *SSEWriter) WriteToken(content string) error {
	return s.WriteEvent(StreamEvent{
		Type:    EventTypeToken,
		Content: content,
	})
}

// WriteRaw writes raw content directly (for backward compatibility)
func (s *SSEWriter) WriteRaw(content string) error {
	_, err := s.w.Write([]byte(content))
	if err != nil {
		return err
	}
	s.Flush()
	return nil
}

// Flush flushes the writer if it supports flushing
func (s *SSEWriter) Flush() {
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// StreamProcessor processes LLM output through artifact detection and writes to SSE
type StreamProcessor struct {
	detector *ArtifactDetector
	writer   *SSEWriter
	buffer   string
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(w io.Writer) *StreamProcessor {
	return &StreamProcessor{
		detector: NewArtifactDetector(),
		writer:   NewSSEWriter(w),
	}
}

// ProcessChunk processes a chunk from the LLM and writes events
func (p *StreamProcessor) ProcessChunk(chunk string) error {
	events := p.detector.ProcessChunk(chunk)
	for _, event := range events {
		if err := p.writer.WriteEvent(event); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes any remaining content
func (p *StreamProcessor) Flush() error {
	events := p.detector.Flush()
	for _, event := range events {
		if err := p.writer.WriteEvent(event); err != nil {
			return err
		}
	}
	return nil
}

// WriteRaw writes raw content directly (bypasses artifact detection)
func (p *StreamProcessor) WriteRaw(content string) error {
	return p.writer.WriteRaw(content)
}

// IsInArtifact returns whether we're currently inside an artifact
func (p *StreamProcessor) IsInArtifact() bool {
	return p.detector.IsInArtifact()
}
