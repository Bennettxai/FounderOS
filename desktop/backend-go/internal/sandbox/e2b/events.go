package e2b

import (
	"encoding/json"
	"fmt"
	"time"
)

// ProgressEvent is the payload emitted at each stage of sandbox execution.
// It is designed to be serialised as a JSON body inside an SSE data frame.
type ProgressEvent struct {
	Event     string    `json:"event"`    // always "e2b_progress"
	Phase     string    `json:"phase"`    // starting|upload|install|build|test|complete|failed
	Progress  int       `json:"progress"` // 0-100
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	SandboxID string    `json:"sandbox_id,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	Attempt   int       `json:"attempt,omitempty"`
}

// Broadcaster is a function that delivers an SSE payload to all listeners
// subscribed to the given workflowID. The message string is expected to be a
// serialised ProgressEvent JSON object.
//
// Implementations must be safe for concurrent use; the sandbox client may call
// them from multiple goroutines during a retry loop.
type Broadcaster func(workflowID, message string)

// phaseProgress maps phase names to their canonical progress percentages so
// the caller does not have to track them manually.
var phaseProgress = map[string]int{
	"starting": 5,
	"upload":   20,
	"install":  40,
	"build":    70,
	"test":     90,
	"complete": 100,
}

// createProgressEvent serialises a ProgressEvent. If JSON marshalling fails
// the function returns a safe minimal fallback string rather than an error,
// because progress events are best-effort.
func createProgressEvent(phase string, progress int, message, sandboxID, tenantID string, attempt int) string {
	ev := ProgressEvent{
		Event:     "e2b_progress",
		Phase:     phase,
		Progress:  progress,
		Message:   message,
		Timestamp: time.Now().UTC(),
		SandboxID: sandboxID,
		TenantID:  tenantID,
		Attempt:   attempt,
	}

	b, err := json.Marshal(ev)
	if err != nil {
		// Fallback: hand-crafted JSON to avoid losing the event entirely.
		return fmt.Sprintf(
			`{"event":"e2b_progress","phase":%q,"progress":%d,"message":%q}`,
			phase, progress, message,
		)
	}
	return string(b)
}

// createProgressErrorEvent is like createProgressEvent but includes an error
// field describing what went wrong.
func createProgressErrorEvent(phase string, progress int, message, errMsg, sandboxID, tenantID string, attempt int) string {
	ev := ProgressEvent{
		Event:     "e2b_progress",
		Phase:     phase,
		Progress:  progress,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Error:     errMsg,
		SandboxID: sandboxID,
		TenantID:  tenantID,
		Attempt:   attempt,
	}

	b, err := json.Marshal(ev)
	if err != nil {
		return fmt.Sprintf(
			`{"event":"e2b_progress","phase":%q,"progress":%d,"message":%q,"error":%q}`,
			phase, progress, message, errMsg,
		)
	}
	return string(b)
}

// emit is the internal helper that guards against nil broadcasters and empty
// workflow IDs before invoking the broadcaster.
func emit(b Broadcaster, workflowID, message string) {
	if b != nil && workflowID != "" {
		b(workflowID, message)
	}
}

// EmitStarting emits the "starting" phase event (progress 5 %).
func EmitStarting(b Broadcaster, workflowID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"starting", 5, "initializing E2B sandbox execution", "", tenantID, attempt,
	))
}

// EmitUpload emits the "upload" phase event (progress 20 %).
func EmitUpload(b Broadcaster, workflowID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"upload", 20, "uploading project files to sandbox", "", tenantID, attempt,
	))
}

// EmitInstall emits the "install" phase event (progress 40 %).
func EmitInstall(b Broadcaster, workflowID, sandboxID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"install", 40, "installing dependencies", sandboxID, tenantID, attempt,
	))
}

// EmitBuild emits the "build" phase event (progress 70 %).
func EmitBuild(b Broadcaster, workflowID, sandboxID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"build", 70, "building application", sandboxID, tenantID, attempt,
	))
}

// EmitTest emits the "test" phase event (progress 90 %).
func EmitTest(b Broadcaster, workflowID, sandboxID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"test", 90, "running smoke tests", sandboxID, tenantID, attempt,
	))
}

// EmitComplete emits the "complete" phase event (progress 100 %).
func EmitComplete(b Broadcaster, workflowID, sandboxID, tenantID string, attempt int) {
	emit(b, workflowID, createProgressEvent(
		"complete", 100, "sandbox execution completed successfully", sandboxID, tenantID, attempt,
	))
}

// EmitFailed emits a failure event with a contextual progress percentage
// derived from the phase in which the failure occurred.
func EmitFailed(b Broadcaster, workflowID, phase, errMsg, sandboxID, tenantID string, attempt int) {
	progress, ok := phaseProgress[phase]
	if !ok {
		progress = 50
	}
	msg := fmt.Sprintf("sandbox execution failed at %s phase", phase)
	emit(b, workflowID, createProgressErrorEvent(
		phase, progress, msg, errMsg, sandboxID, tenantID, attempt,
	))
}
