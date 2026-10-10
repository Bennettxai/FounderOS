package streaming

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// =============================================================================
// EVENT TYPE TESTS
// =============================================================================

func TestEventTypeConstants(t *testing.T) {
	// Verify all expected event types exist
	expectedTypes := map[EventType]string{
		EventTypeToken:            "token",
		EventTypeArtifactStart:    "artifact_start",
		EventTypeArtifactComplete: "artifact_complete",
		EventTypeArtifactError:    "artifact_error",
		EventTypeToolCall:         "tool_call",
		EventTypeToolResult:       "tool_result",
		EventTypeThinking:         "thinking",
		EventTypeDelegating:       "delegating",
		EventTypeDone:             "done",
		EventTypeError:            "error",
	}

	for eventType, expectedValue := range expectedTypes {
		if string(eventType) != expectedValue {
			t.Errorf("EventType %v should equal '%s', got '%s'", eventType, expectedValue, string(eventType))
		}
	}
}

func TestStreamEventJSON(t *testing.T) {
	event := StreamEvent{
		Type:    EventTypeToken,
		Content: "Hello world",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal StreamEvent: %v", err)
	}

	var decoded StreamEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal StreamEvent: %v", err)
	}

	if decoded.Type != EventTypeToken {
		t.Errorf("Expected type 'token', got '%s'", decoded.Type)
	}
	if decoded.Content != "Hello world" {
		t.Errorf("Expected content 'Hello world', got '%s'", decoded.Content)
	}
}

func TestStreamEventWithData(t *testing.T) {
	artifact := Artifact{
		Type:    "proposal",
		Title:   "Test Proposal",
		Content: "# Proposal Content",
	}

	event := StreamEvent{
		Type: EventTypeArtifactComplete,
		Data: artifact,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal StreamEvent with data: %v", err)
	}

	if !strings.Contains(string(data), "proposal") {
		t.Error("Serialized event should contain artifact data")
	}
}

func TestThinkingStepJSON(t *testing.T) {
	step := ThinkingStep{
		Step:      "analyzing",
		Content:   "Analyzing user request",
		Agent:     "orchestrator",
		Completed: false,
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("Failed to marshal ThinkingStep: %v", err)
	}

	var decoded ThinkingStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ThinkingStep: %v", err)
	}

	if decoded.Step != "analyzing" {
		t.Errorf("Expected step 'analyzing', got '%s'", decoded.Step)
	}
	if decoded.Agent != "orchestrator" {
		t.Errorf("Expected agent 'orchestrator', got '%s'", decoded.Agent)
	}
}

func TestToolCallEventJSON(t *testing.T) {
	toolCall := ToolCallEvent{
		ToolName: "create_task",
		Parameters: map[string]interface{}{
			"title":    "New Task",
			"priority": "high",
		},
		Status: "calling",
	}

	data, err := json.Marshal(toolCall)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCallEvent: %v", err)
	}

	var decoded ToolCallEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolCallEvent: %v", err)
	}

	if decoded.ToolName != "create_task" {
		t.Errorf("Expected tool_name 'create_task', got '%s'", decoded.ToolName)
	}
	if decoded.Status != "calling" {
		t.Errorf("Expected status 'calling', got '%s'", decoded.Status)
	}
}

func TestArtifactJSON(t *testing.T) {
	artifact := Artifact{
		ID:      "art-123",
		Type:    "report",
		Title:   "Q4 Report",
		Content: "# Q4 Financial Report\n\n## Summary",
	}

	data, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("Failed to marshal Artifact: %v", err)
	}

	var decoded Artifact
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Artifact: %v", err)
	}

	if decoded.ID != "art-123" {
		t.Errorf("Expected ID 'art-123', got '%s'", decoded.ID)
	}
	if decoded.Type != "report" {
		t.Errorf("Expected type 'report', got '%s'", decoded.Type)
	}
}

// =============================================================================
// ARTIFACT DETECTOR TESTS
// =============================================================================

func TestNewArtifactDetector(t *testing.T) {
	detector := NewArtifactDetector()
	if detector == nil {
		t.Fatal("NewArtifactDetector returned nil")
	}
	if detector.IsInArtifact() {
		t.Error("New detector should not be in artifact mode")
	}
}

func TestArtifactDetectorSimpleText(t *testing.T) {
	detector := NewArtifactDetector()

	// Process simple text without artifacts
	events := detector.ProcessChunk("Hello, this is a simple message.")

	// Should get token events for the text
	hasToken := false
	for _, event := range events {
		if event.Type == EventTypeToken {
			hasToken = true
		}
	}

	// Flush remaining
	flushEvents := detector.Flush()
	for _, event := range flushEvents {
		if event.Type == EventTypeToken {
			hasToken = true
		}
	}

	if !hasToken {
		t.Error("Expected token events for simple text")
	}
}

func TestArtifactDetectorArtifactStart(t *testing.T) {
	detector := NewArtifactDetector()

	// Process text with artifact start marker
	content := "Here is a document:\n\n```artifact\n"
	events := detector.ProcessChunk(content)

	hasArtifactStart := false
	for _, event := range events {
		if event.Type == EventTypeArtifactStart {
			hasArtifactStart = true
		}
	}

	if !hasArtifactStart {
		t.Error("Expected artifact_start event when ```artifact marker is found")
	}

	if !detector.IsInArtifact() {
		t.Error("Detector should be in artifact mode after start marker")
	}
}

func TestArtifactDetectorCompleteArtifact(t *testing.T) {
	detector := NewArtifactDetector()

	// Process complete artifact
	artifactJSON := `{"type": "proposal", "title": "Test", "content": "# Test Content"}`
	content := "```artifact\n" + artifactJSON + "\n```"

	events := detector.ProcessChunk(content)
	events = append(events, detector.Flush()...)

	hasArtifactComplete := false
	for _, event := range events {
		if event.Type == EventTypeArtifactComplete {
			hasArtifactComplete = true
			if artifact, ok := event.Data.(Artifact); ok {
				if artifact.Type != "proposal" {
					t.Errorf("Expected artifact type 'proposal', got '%s'", artifact.Type)
				}
			}
		}
	}

	if !hasArtifactComplete {
		t.Error("Expected artifact_complete event for complete artifact")
	}
}

func TestArtifactDetectorInvalidJSON(t *testing.T) {
	detector := NewArtifactDetector()

	// Process artifact with invalid JSON
	content := "```artifact\n{invalid json}\n```"
	events := detector.ProcessChunk(content)
	events = append(events, detector.Flush()...)

	hasError := false
	for _, event := range events {
		if event.Type == EventTypeArtifactError {
			hasError = true
		}
	}

	if !hasError {
		t.Error("Expected artifact_error event for invalid JSON")
	}
}

func TestArtifactDetectorUnclosedArtifact(t *testing.T) {
	detector := NewArtifactDetector()

	// Process unclosed artifact
	content := "```artifact\n{\"type\": \"test\"}"
	detector.ProcessChunk(content)
	events := detector.Flush()

	hasError := false
	for _, event := range events {
		if event.Type == EventTypeArtifactError {
			hasError = true
		}
	}

	if !hasError {
		t.Error("Expected artifact_error event for unclosed artifact")
	}
}

func TestArtifactDetectorReset(t *testing.T) {
	detector := NewArtifactDetector()

	// Start an artifact
	detector.ProcessChunk("```artifact\n")
	if !detector.IsInArtifact() {
		t.Error("Should be in artifact mode")
	}

	// Reset
	detector.Reset()
	if detector.IsInArtifact() {
		t.Error("Should not be in artifact mode after reset")
	}
}

func TestArtifactDetectorChunkedInput(t *testing.T) {
	detector := NewArtifactDetector()

	// Simulate chunked streaming input
	chunks := []string{
		"Here is my ",
		"response with an ",
		"artifact:\n\n```art",
		"ifact\n{\"type\": \"doc",
		"ument\", \"title\": \"Test\", \"content\": \"Hello\"}",
		"\n```\n\nDone!",
	}

	var allEvents []StreamEvent
	for _, chunk := range chunks {
		events := detector.ProcessChunk(chunk)
		allEvents = append(allEvents, events...)
	}
	allEvents = append(allEvents, detector.Flush()...)

	// Check for expected events
	hasToken := false
	hasArtifactComplete := false
	for _, event := range allEvents {
		if event.Type == EventTypeToken {
			hasToken = true
		}
		if event.Type == EventTypeArtifactComplete {
			hasArtifactComplete = true
		}
	}

	if !hasToken {
		t.Error("Expected token events")
	}
	if !hasArtifactComplete {
		t.Error("Expected artifact_complete event")
	}
}

// =============================================================================
// SSE WRITER TESTS
// =============================================================================

func TestNewSSEWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	if writer == nil {
		t.Fatal("NewSSEWriter returned nil")
	}
}

func TestSSEWriterWriteEvent(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	event := StreamEvent{
		Type:    EventTypeToken,
		Content: "Hello",
	}

	err := writer.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: token") {
		t.Error("Output should contain 'event: token'")
	}
	if !strings.Contains(output, "Hello") {
		t.Error("Output should contain content")
	}
}

func TestSSEWriterWriteToken(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	err := writer.WriteToken("Test content")
	if err != nil {
		t.Fatalf("WriteToken failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: token") {
		t.Error("Output should contain 'event: token'")
	}
	if !strings.Contains(output, "Test content") {
		t.Error("Output should contain 'Test content'")
	}
}

func TestSSEWriterWriteRaw(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	err := writer.WriteRaw("raw content")
	if err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	if buf.String() != "raw content" {
		t.Errorf("Expected 'raw content', got '%s'", buf.String())
	}
}

func TestSSEWriterSSEFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	event := StreamEvent{
		Type:    EventTypeDone,
		Content: "complete",
	}

	writer.WriteEvent(event)

	output := buf.String()
	// SSE format should have event line, data line, and double newline
	if !strings.Contains(output, "event: done\n") {
		t.Error("SSE should have event line")
	}
	if !strings.Contains(output, "data: ") {
		t.Error("SSE should have data line")
	}
	if !strings.HasSuffix(output, "\n\n") {
		t.Error("SSE should end with double newline")
	}
}

// =============================================================================
// STREAM PROCESSOR TESTS
// =============================================================================

func TestNewStreamProcessor(t *testing.T) {
	buf := &bytes.Buffer{}
	processor := NewStreamProcessor(buf)

	if processor == nil {
		t.Fatal("NewStreamProcessor returned nil")
	}
	if processor.IsInArtifact() {
		t.Error("New processor should not be in artifact mode")
	}
}

func TestStreamProcessorProcessChunk(t *testing.T) {
	buf := &bytes.Buffer{}
	processor := NewStreamProcessor(buf)

	err := processor.ProcessChunk("Hello, world!")
	if err != nil {
		t.Fatalf("ProcessChunk failed: %v", err)
	}

	// Flush to get remaining content
	processor.Flush()

	output := buf.String()
	if !strings.Contains(output, "Hello") {
		t.Error("Output should contain processed content")
	}
}

func TestStreamProcessorWithArtifact(t *testing.T) {
	buf := &bytes.Buffer{}
	processor := NewStreamProcessor(buf)

	artifactJSON := `{"type": "test", "title": "Test", "content": "Content"}`
	content := "Here is a document:\n\n```artifact\n" + artifactJSON + "\n```\n\nDone!"

	processor.ProcessChunk(content)
	processor.Flush()

	output := buf.String()
	if !strings.Contains(output, "artifact_start") || !strings.Contains(output, "artifact_complete") {
		t.Error("Output should contain artifact events")
	}
}

func TestStreamProcessorWriteRaw(t *testing.T) {
	buf := &bytes.Buffer{}
	processor := NewStreamProcessor(buf)

	err := processor.WriteRaw("raw data")
	if err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	if buf.String() != "raw data" {
		t.Errorf("Expected 'raw data', got '%s'", buf.String())
	}
}

func TestStreamProcessorIsInArtifact(t *testing.T) {
	buf := &bytes.Buffer{}
	processor := NewStreamProcessor(buf)

	if processor.IsInArtifact() {
		t.Error("Should not be in artifact initially")
	}

	processor.ProcessChunk("```artifact\n")

	if !processor.IsInArtifact() {
		t.Error("Should be in artifact after start marker")
	}
}

// =============================================================================
// EDGE CASE TESTS
// =============================================================================

func TestEmptyInput(t *testing.T) {
	detector := NewArtifactDetector()
	events := detector.ProcessChunk("")

	if len(events) != 0 {
		t.Error("Empty input should produce no events")
	}
}

func TestMultipleArtifacts(t *testing.T) {
	detector := NewArtifactDetector()

	// Simulate streaming by sending multiple chunks - this matches how
	// the detector is designed to work in a real streaming scenario
	chunks := []string{
		"First text\n",
		"```artifact\n",
		`{"type": "doc1", "title": "First", "content": "One"}`,
		"\n```\n",
		"Middle text\n",
		"```artifact\n",
		`{"type": "doc2", "title": "Second", "content": "Two"}`,
		"\n```\n",
		"End text",
	}

	var events []StreamEvent
	for _, chunk := range chunks {
		events = append(events, detector.ProcessChunk(chunk)...)
	}
	events = append(events, detector.Flush()...)

	artifactCount := 0
	for _, event := range events {
		if event.Type == EventTypeArtifactComplete {
			artifactCount++
		}
	}

	if artifactCount != 2 {
		t.Errorf("Expected 2 artifact_complete events, got %d", artifactCount)
	}
}

func TestSpecialCharactersInContent(t *testing.T) {
	detector := NewArtifactDetector()

	content := `Special chars: "quotes", 'apostrophes', \backslash, and emoji`
	events := detector.ProcessChunk(content)
	events = append(events, detector.Flush()...)

	hasToken := false
	for _, event := range events {
		if event.Type == EventTypeToken {
			hasToken = true
		}
	}

	if !hasToken {
		t.Error("Should handle special characters")
	}
}

func TestUnicodeContent(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewSSEWriter(buf)

	event := StreamEvent{
		Type:    EventTypeToken,
		Content: "Unicode: ",
	}

	err := writer.WriteEvent(event)
	if err != nil {
		t.Fatalf("Failed to write unicode content: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Unicode") {
		t.Error("Should preserve unicode characters")
	}
}
