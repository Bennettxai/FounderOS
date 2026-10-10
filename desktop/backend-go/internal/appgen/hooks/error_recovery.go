package hooks

import (
	"context"
	"fmt"
	"log/slog"
)

// ErrorRecoveryHook implements the VIGIL pattern for self-healing agents
// It analyzes errors, proposes recovery strategies, and saves resolutions
type ErrorRecoveryHook struct {
	maxAttempts int
	logger      *slog.Logger
}

// NewErrorRecoveryHook creates a new error recovery hook
func NewErrorRecoveryHook(maxAttempts int, logger *slog.Logger) *ErrorRecoveryHook {
	return &ErrorRecoveryHook{
		maxAttempts: maxAttempts,
		logger:      logger,
	}
}

func (h *ErrorRecoveryHook) Name() string {
	return "error-recovery"
}

func (h *ErrorRecoveryHook) Execute(ctx context.Context, hookCtx HookContext) error {
	// Only process if there's an error
	if hookCtx.Error == nil {
		return nil
	}

	h.logger.WarnContext(ctx, "agent error detected - analyzing for recovery",
		"agent_type", hookCtx.AgentType,
		"task_id", hookCtx.TaskID,
		"error", hookCtx.Error)

	// VIGIL pattern: Reflect on the error
	reflection := h.analyzeError(ctx, hookCtx.Error)

	// Propose recovery strategy
	strategy := h.proposeRecoveryStrategy(reflection)

	h.logger.InfoContext(ctx, "recovery strategy proposed",
		"agent_type", hookCtx.AgentType,
		"error_type", reflection.ErrorType,
		"strategy", strategy.Action,
		"confidence", strategy.Confidence)

	// Save error resolution for learning
	if err := h.saveErrorResolution(ctx, hookCtx, reflection, strategy); err != nil {
		h.logger.WarnContext(ctx, "failed to save error resolution",
			"error", err)
	}

	return nil
}

// ErrorReflection represents analysis of an error
type ErrorReflection struct {
	ErrorType   string // "timeout", "api_error", "validation_error", etc.
	Severity    string // "low", "medium", "high"
	Retriable   bool
	RootCause   string
	ContextInfo map[string]interface{}
}

// RecoveryStrategy represents a proposed solution
type RecoveryStrategy struct {
	Action     string  // "retry", "fallback", "escalate", "skip"
	Confidence float64 // 0.0 to 1.0
	Details    string
}

// analyzeError performs reflection on the error (VIGIL pattern)
func (h *ErrorRecoveryHook) analyzeError(ctx context.Context, err error) ErrorReflection {
	reflection := ErrorReflection{
		ErrorType:   "unknown",
		Severity:    "medium",
		Retriable:   false,
		RootCause:   err.Error(),
		ContextInfo: make(map[string]interface{}),
	}

	errStr := err.Error()

	// Classify error type
	switch {
	case contains(errStr, "timeout"):
		reflection.ErrorType = "timeout"
		reflection.Retriable = true
		reflection.Severity = "medium"

	case contains(errStr, "api", "request"):
		reflection.ErrorType = "api_error"
		reflection.Retriable = true
		reflection.Severity = "medium"

	case contains(errStr, "validation", "invalid"):
		reflection.ErrorType = "validation_error"
		reflection.Retriable = false
		reflection.Severity = "low"

	case contains(errStr, "unauthorized", "forbidden"):
		reflection.ErrorType = "auth_error"
		reflection.Retriable = false
		reflection.Severity = "high"

	case contains(errStr, "not found"):
		reflection.ErrorType = "not_found"
		reflection.Retriable = false
		reflection.Severity = "low"

	default:
		reflection.ErrorType = "unknown"
		reflection.Retriable = true
		reflection.Severity = "medium"
	}

	return reflection
}

// proposeRecoveryStrategy suggests how to handle the error
func (h *ErrorRecoveryHook) proposeRecoveryStrategy(reflection ErrorReflection) RecoveryStrategy {
	strategy := RecoveryStrategy{
		Confidence: 0.5,
	}

	switch reflection.ErrorType {
	case "timeout", "api_error":
		strategy.Action = "retry"
		strategy.Confidence = 0.8
		strategy.Details = fmt.Sprintf("Retry with exponential backoff (max %d attempts)", h.maxAttempts)

	case "validation_error":
		strategy.Action = "escalate"
		strategy.Confidence = 0.9
		strategy.Details = "Validation error requires input correction - escalate to user"

	case "auth_error":
		strategy.Action = "escalate"
		strategy.Confidence = 0.95
		strategy.Details = "Authentication issue - check credentials and permissions"

	case "not_found":
		strategy.Action = "fallback"
		strategy.Confidence = 0.7
		strategy.Details = "Resource not found - use default or skip"

	default:
		strategy.Action = "retry"
		strategy.Confidence = 0.5
		strategy.Details = "Unknown error - attempt retry with caution"
	}

	return strategy
}

// saveErrorResolution saves the error and recovery strategy for learning
func (h *ErrorRecoveryHook) saveErrorResolution(ctx context.Context, hookCtx HookContext,
	reflection ErrorReflection, strategy RecoveryStrategy) error {

	h.logger.InfoContext(ctx, "error resolution saved",
		"agent_type", hookCtx.AgentType,
		"error_type", reflection.ErrorType,
		"strategy", strategy.Action,
		"retriable", reflection.Retriable)

	// In production, this would save to a database for pattern learning
	// For now, just log it

	return nil
}

// contains checks if a string contains any of the given substrings
func contains(str string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
