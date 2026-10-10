package streaming

// EventType represents different SSE event types
type EventType string

const (
	EventTypeToken            EventType = "token"
	EventTypeArtifactStart    EventType = "artifact_start"
	EventTypeArtifactComplete EventType = "artifact_complete"
	EventTypeArtifactError    EventType = "artifact_error"
	EventTypeToolCall         EventType = "tool_call"
	EventTypeToolResult       EventType = "tool_result"
	EventTypeThinking         EventType = "thinking"
	EventTypeThinkingStart    EventType = "thinking_start"
	EventTypeThinkingChunk    EventType = "thinking_chunk"
	EventTypeThinkingEnd      EventType = "thinking_end"
	EventTypeContentStart     EventType = "content_start"
	EventTypeContentEnd       EventType = "content_end"
	EventTypeDelegating       EventType = "delegating"
	EventTypeBlocks           EventType = "blocks"
	EventTypeDone             EventType = "done"
	EventTypeError            EventType = "error"
)

// ThinkingStep represents a COT thinking step
type ThinkingStep struct {
	Step      string `json:"step"`      // "analyzing", "planning", "executing", "synthesizing"
	Content   string `json:"content"`   // What the agent is thinking
	Agent     string `json:"agent"`     // Which agent is thinking
	Completed bool   `json:"completed"` // Whether this step is done
}

// ToolCallEvent represents a tool being called
type ToolCallEvent struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Status     string                 `json:"status"` // "calling", "success", "error"
	Result     string                 `json:"result,omitempty"`
}

// StreamEvent represents an event to send to the frontend
type StreamEvent struct {
	Type    EventType   `json:"type"`
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Artifact represents a parsed artifact
type Artifact struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
