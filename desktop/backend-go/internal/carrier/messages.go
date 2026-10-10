package carrier

// Request is a message sent from BOS to the SorxMain reasoning engine.
type Request struct {
	// ID is a unique identifier for this request, typically a UUID v4.
	ID string `json:"id"`

	// Method describes the reasoning operation to invoke.
	// Known values: "search", "deliberate", "verify", "plan".
	Method string `json:"method"`

	// RoutingKey is the AMQP topic routing key that determines which
	// SorxMain subsystem handles the message.
	// Format: "<topic>.<action>" e.g. "mcts.analyze_revenue", "boardroom.strategy".
	RoutingKey string `json:"routing_key"`

	// Params contains method-specific parameters.
	Params map[string]any `json:"params"`

	// Context carries BOS-level metadata scoped to a single OS instance.
	Context MessageContext `json:"context"`

	// Tools lists tool schemas available to the reasoning engine for this request.
	Tools []Tool `json:"tools,omitempty"`

	// TTL is the message time-to-live in milliseconds.
	// Messages not consumed within TTL are dropped by the broker.
	// Defaults to 60000 (60 seconds).
	TTL int `json:"ttl"`

	// Priority controls message ordering within the queue.
	// Range: 1 (lowest) to 10 (highest). Defaults to 5.
	Priority int `json:"priority"`
}

// Response is a message received from the SorxMain reasoning engine.
type Response struct {
	// ID is the unique identifier echoed from the originating Request.
	ID string `json:"id"`

	// CorrelationID matches this response to its pending Request.
	CorrelationID string `json:"correlation_id"`

	// Result holds the reasoning engine's output. Shape varies by Method.
	Result any `json:"result"`

	// Error is non-nil when the reasoning engine reported a failure.
	Error *ResponseError `json:"error,omitempty"`

	// Timestamp is Unix milliseconds when SorxMain produced this response.
	Timestamp int64 `json:"timestamp"`

	// DurationMS is the wall-clock time SorxMain spent processing the request.
	DurationMS int64 `json:"duration_ms"`
}

// MessageContext carries the full BOS context attached to every CARRIER request.
// Optimal uses this to reason with full awareness of the user's state.
type MessageContext struct {
	// --- Identity ---

	// OSInstanceID identifies the BOS instance sending the request.
	OSInstanceID string `json:"os_instance_id"`

	// WorkspaceID scopes the request to a workspace.
	WorkspaceID string `json:"workspace_id"`

	// UserID identifies the user on whose behalf the request is made.
	UserID string `json:"user_id"`

	// --- User Message ---

	// UserMessage is the raw text the user typed in the chat.
	// Empty for scheduler-initiated or proactive requests.
	UserMessage string `json:"user_message,omitempty"`

	// Mode is the classified OSA mode for this request.
	// Values: "BUILD", "EXECUTE", "ANALYZE", "MAINTAIN", "ASSIST".
	Mode string `json:"mode,omitempty"`

	// --- Conversation History ---

	// ConversationID links to the active conversation thread.
	ConversationID string `json:"conversation_id,omitempty"`

	// ConversationHistory carries recent messages for continuity.
	// BOS sends the last N messages (compressed if needed).
	// Each entry has "role" ("user"|"assistant") and "content".
	ConversationHistory []ConversationMessage `json:"conversation_history,omitempty"`

	// --- Tiered Context (L1/L2/L3) ---

	// SelectedProjectID is the project the user has focused on (Level 1).
	SelectedProjectID string `json:"selected_project_id,omitempty"`

	// SelectedContextIDs are document/knowledge IDs the user selected (Level 1).
	SelectedContextIDs []string `json:"selected_context_ids,omitempty"`

	// ProjectSummary is Level 1 full context for the selected project.
	// Includes name, status, tasks, description — pre-formatted by BOS.
	ProjectSummary string `json:"project_summary,omitempty"`

	// DocumentContext is Level 1 content from selected documents.
	// Pre-formatted by BOS's tiered context service.
	DocumentContext string `json:"document_context,omitempty"`

	// AwarenessContext is Level 2 summaries of related items.
	// Other projects, sibling docs, related entities — titles only.
	AwarenessContext string `json:"awareness_context,omitempty"`

	// RAGResults are relevant knowledge blocks from scoped vector search.
	// Pre-retrieved by BOS's embedding service against selected contexts.
	RAGResults []RAGResult `json:"rag_results,omitempty"`

	// --- User Preferences ---

	// Temperature is the user's autonomy setting: "cold", "warm", "hot".
	Temperature string `json:"temperature,omitempty"`

	// OutputStyle is the user's preferred response format.
	OutputStyle string `json:"output_style,omitempty"`

	// UserRole is the user's role in the workspace (admin, member, etc.).
	UserRole string `json:"user_role,omitempty"`

	// --- Capabilities ---

	// InstalledModules lists modules active for this workspace.
	InstalledModules []string `json:"installed_modules,omitempty"`

	// ConnectedIntegrations lists integrations this user has connected.
	// Optimal uses this to know which action commands BOS can execute.
	ConnectedIntegrations []string `json:"connected_integrations,omitempty"`
}

// ConversationMessage is a single message in the conversation history.
type ConversationMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // message text
}

// RAGResult is a relevant knowledge block from vector search.
type RAGResult struct {
	ContextID  string  `json:"context_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"` // cosine similarity score
}

// Tool describes a tool schema that the reasoning engine may invoke.
type Tool struct {
	// Name is the tool identifier.
	Name string `json:"name"`

	// Schema is a JSON Schema object describing the tool's input parameters.
	Schema map[string]any `json:"schema"`
}

// ResponseError carries structured error information from SorxMain.
type ResponseError struct {
	// Code is a machine-readable error identifier (e.g. "MCTS_TIMEOUT").
	Code string `json:"code"`

	// Message is a human-readable description of the error.
	Message string `json:"message"`

	// Details provides additional diagnostic context, if available.
	Details string `json:"details,omitempty"`
}

// Error implements the error interface so ResponseError can be used with
// standard Go error handling idioms.
func (e *ResponseError) Error() string {
	if e.Details != "" {
		return e.Code + ": " + e.Message + " (" + e.Details + ")"
	}
	return e.Code + ": " + e.Message
}

const (
	// DefaultTTL is the default message time-to-live in milliseconds (60s).
	DefaultTTL = 60_000

	// DefaultPriority is the default message priority (mid-range).
	DefaultPriority = 5

	// MinPriority is the lowest valid message priority.
	MinPriority = 1

	// MaxPriority is the highest valid message priority.
	MaxPriority = 10
)

// sanitize fills in default values for optional Request fields and clamps
// Priority to the valid range. It does not validate required fields.
func (r *Request) sanitize() {
	if r.TTL <= 0 {
		r.TTL = DefaultTTL
	}
	if r.Priority < MinPriority {
		r.Priority = DefaultPriority
	}
	if r.Priority > MaxPriority {
		r.Priority = MaxPriority
	}
}
