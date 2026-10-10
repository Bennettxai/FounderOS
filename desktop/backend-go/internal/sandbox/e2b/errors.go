// Package e2b provides an E2B sandbox client for isolated code execution.
// It supports the full sandbox lifecycle: create, execute, update files, and destroy.
package e2b

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
)

// Sentinel errors for common E2B operation failures. Callers should use
// errors.Is to match these rather than comparing error strings.
var (
	ErrSandboxNotFound    = errors.New("sandbox not found")
	ErrSandboxTimeout     = errors.New("sandbox operation timeout")
	ErrSandboxUnavailable = errors.New("sandbox service unavailable")
	ErrInvalidProjectPath = errors.New("invalid project path")
	ErrEmptyFiles         = errors.New("no files provided for update")
	ErrExecutionFailed    = errors.New("code execution failed")
	ErrBuildFailed        = errors.New("build process failed")
	ErrInstallFailed      = errors.New("dependency installation failed")
	ErrStartupFailed      = errors.New("application startup failed")
	ErrNetworkFailure     = errors.New("network connection failed")
	ErrRateLimited        = errors.New("rate limit exceeded")
)

// ErrorType categorises different classes of E2B errors so callers can apply
// appropriate handling (e.g. retry strategies differ by type).
type ErrorType string

const (
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeExecution  ErrorType = "execution"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeService    ErrorType = "service"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// ExecutionPhase identifies which step of the sandbox lifecycle failed.
type ExecutionPhase string

const (
	PhaseSetup   ExecutionPhase = "setup"
	PhaseInstall ExecutionPhase = "install"
	PhaseBuild   ExecutionPhase = "build"
	PhaseStart   ExecutionPhase = "start"
	PhaseTest    ExecutionPhase = "test"
)

// E2BError is a structured error from any E2B operation. It implements the
// standard error interface and carries enough context for retry decisions and
// structured logging.
type E2BError struct {
	Type      ErrorType         `json:"type"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Phase     ExecutionPhase    `json:"phase,omitempty"`
	SandboxID string            `json:"sandbox_id,omitempty"`
	TenantID  string            `json:"tenant_id,omitempty"`
	Retryable bool              `json:"retryable"`
	Details   map[string]string `json:"details,omitempty"`
	Cause     error             `json:"-"`
}

// Error implements the error interface.
func (e *E2BError) Error() string {
	if e.Phase != "" {
		return fmt.Sprintf("e2b %s error in %s phase: %s", e.Type, e.Phase, e.Message)
	}
	return fmt.Sprintf("e2b %s error: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause so errors.Is/As work through the chain.
func (e *E2BError) Unwrap() error {
	return e.Cause
}

// IsRetryable reports whether the operation should be attempted again.
func (e *E2BError) IsRetryable() bool {
	return e.Retryable
}

// AddDetail attaches a key-value annotation to the error. It returns e to
// allow method chaining.
func (e *E2BError) AddDetail(key, value string) *E2BError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithCause records the underlying error that triggered this E2BError.
func (e *E2BError) WithCause(cause error) *E2BError {
	e.Cause = cause
	return e
}

// WithSandboxID records the sandbox that experienced the failure.
func (e *E2BError) WithSandboxID(sandboxID string) *E2BError {
	e.SandboxID = sandboxID
	return e
}

// NewE2BError constructs a minimal E2BError.
func NewE2BError(errorType ErrorType, code, message string, retryable bool) *E2BError {
	return &E2BError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Details:   make(map[string]string),
	}
}

// NewExecutionError creates an E2BError scoped to a specific execution phase.
func NewExecutionError(phase ExecutionPhase, message, sandboxID string) *E2BError {
	return &E2BError{
		Type:      ErrorTypeExecution,
		Code:      fmt.Sprintf("%s_FAILED", strings.ToUpper(string(phase))),
		Message:   message,
		Phase:     phase,
		SandboxID: sandboxID,
		Retryable: false,
		Details:   make(map[string]string),
	}
}

// NewValidationError creates a non-retryable input validation error.
func NewValidationError(message string) *E2BError {
	return &E2BError{
		Type:      ErrorTypeValidation,
		Code:      "VALIDATION_ERROR",
		Message:   message,
		Retryable: false,
		Details:   make(map[string]string),
	}
}

// RetryStrategy controls exponential-backoff retry behaviour for a given
// ErrorType.
type RetryStrategy struct {
	MaxAttempts   int           `json:"max_attempts"`
	BaseDelay     time.Duration `json:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	Jitter        bool          `json:"jitter"`
}

// DefaultRetryStrategies returns sensible retry strategies keyed by ErrorType.
func DefaultRetryStrategies() map[ErrorType]*RetryStrategy {
	return map[ErrorType]*RetryStrategy{
		ErrorTypeNetwork: {
			MaxAttempts:   5,
			BaseDelay:     2 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		},
		ErrorTypeTimeout: {
			MaxAttempts:   3,
			BaseDelay:     5 * time.Second,
			MaxDelay:      60 * time.Second,
			BackoffFactor: 1.5,
			Jitter:        false,
		},
		ErrorTypeExecution: {
			MaxAttempts:   3,
			BaseDelay:     3 * time.Second,
			MaxDelay:      15 * time.Second,
			BackoffFactor: 1.0,
			Jitter:        false,
		},
		ErrorTypeRateLimit: {
			MaxAttempts:   10,
			BaseDelay:     10 * time.Second,
			MaxDelay:      300 * time.Second,
			BackoffFactor: 1.5,
			Jitter:        true,
		},
		ErrorTypeService: {
			MaxAttempts:   3,
			BaseDelay:     5 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		},
	}
}

// ClassifyError analyses an arbitrary error and wraps it as a typed E2BError.
// If the error is already an *E2BError it is returned unchanged. The phase and
// sandboxID parameters annotate the returned error with operational context.
func ClassifyError(err error, phase ExecutionPhase, sandboxID string) *E2BError {
	if err == nil {
		return nil
	}

	// Pass through existing structured errors without re-wrapping.
	var existing *E2BError
	if errors.As(err, &existing) {
		return existing
	}

	classified := &E2BError{
		Phase:     phase,
		SandboxID: sandboxID,
		Cause:     err,
		Details:   make(map[string]string),
	}

	// Network errors take priority because they are almost always retryable.
	if isNetworkError(err) {
		classified.Type = ErrorTypeNetwork
		classified.Code = "NETWORK_ERROR"
		classified.Message = "network connection failed"
		classified.Retryable = true
		return classified
	}

	if isTimeoutError(err) {
		classified.Type = ErrorTypeTimeout
		classified.Code = "TIMEOUT"
		classified.Message = "operation timed out"
		classified.Retryable = true
		return classified
	}

	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") {
		classified.Type = ErrorTypeRateLimit
		classified.Code = "RATE_LIMITED"
		classified.Message = "rate limit exceeded"
		classified.Retryable = true
		return classified
	}

	// Phase-specific classification governs the remaining cases.
	switch phase {
	case PhaseInstall:
		classified.Type = ErrorTypeExecution
		classified.Code = "INSTALL_FAILED"
		classified.Message = "dependency installation failed"
		// Network problems during install (e.g. registry unreachable) are retryable.
		classified.Retryable = strings.Contains(errMsg, "network")
	case PhaseBuild:
		classified.Type = ErrorTypeExecution
		classified.Code = "BUILD_FAILED"
		classified.Message = "build process failed"
		classified.Retryable = false
	case PhaseStart:
		classified.Type = ErrorTypeExecution
		classified.Code = "START_FAILED"
		classified.Message = "application startup failed"
		classified.Retryable = false
	case PhaseSetup:
		classified.Type = ErrorTypeService
		classified.Code = "SETUP_FAILED"
		classified.Message = "sandbox setup failed"
		classified.Retryable = true
	default:
		classified.Type = ErrorTypeUnknown
		classified.Code = "UNKNOWN"
		classified.Message = err.Error()
		classified.Retryable = false
	}

	// Override with service-level markers when present.
	if strings.Contains(errMsg, "service unavailable") ||
		strings.Contains(errMsg, "502") ||
		strings.Contains(errMsg, "503") {
		classified.Type = ErrorTypeService
		classified.Code = "SERVICE_UNAVAILABLE"
		classified.Message = "e2b service temporarily unavailable"
		classified.Retryable = true
	}

	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "missing") ||
		strings.Contains(errMsg, "empty") {
		classified.Type = ErrorTypeValidation
		classified.Code = "VALIDATION_ERROR"
		classified.Message = "request validation failed"
		classified.Retryable = false
	}

	return classified
}

// ShouldRetry decides whether attempt number attempt should be repeated and, if
// so, how long to wait. strategies is the caller-supplied retry configuration.
func ShouldRetry(err error, attempt int, strategies map[ErrorType]*RetryStrategy) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	var e2bErr *E2BError
	if !errors.As(err, &e2bErr) {
		e2bErr = ClassifyError(err, "", "")
	}

	if !e2bErr.Retryable {
		return false, 0
	}

	strategy, ok := strategies[e2bErr.Type]
	if !ok {
		return false, 0
	}

	if attempt >= strategy.MaxAttempts {
		return false, 0
	}

	// Exponential backoff: delay = base * factor^(attempt-1), capped at max.
	delay := time.Duration(float64(strategy.BaseDelay) *
		(1.0 + (strategy.BackoffFactor-1.0)*float64(attempt-1)))

	if delay > strategy.MaxDelay {
		delay = strategy.MaxDelay
	}

	// Optional ±10 % jitter to spread bursts across retrying clients.
	if strategy.Jitter && delay > 0 {
		jitter := time.Duration(float64(delay) * 0.1)
		sign := int64(time.Now().UnixNano())%2*2 - 1 // +1 or -1
		delay += time.Duration(sign) * jitter
	}

	return true, delay
}

// isNetworkError checks whether err represents a transient network problem.
func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var addrErr *net.AddrError
	if errors.As(err, &addrErr) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	for _, kw := range []string{
		"connection refused",
		"no such host",
		"network unreachable",
		"connection reset",
		"broken pipe",
		"no route to host",
	} {
		if strings.Contains(errMsg, kw) {
			return true
		}
	}

	return false
}

// isTimeoutError checks whether err represents a deadline or timeout.
func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	for _, kw := range []string{
		"timeout",
		"deadline exceeded",
		"context deadline exceeded",
		"request timeout",
	} {
		if strings.Contains(errMsg, kw) {
			return true
		}
	}

	return false
}
