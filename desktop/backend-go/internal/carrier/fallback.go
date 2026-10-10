package carrier

import (
	"errors"
	"fmt"
)

// Reason codes for fallback conditions.
const (
	// ReasonDisconnected indicates the AMQP connection is not established.
	ReasonDisconnected = "amqp_disconnected"

	// ReasonTimeout indicates SorxMain did not reply within the send timeout.
	ReasonTimeout = "response_timeout"

	// ReasonQueueFull indicates the broker rejected the publish due to resource limits.
	ReasonQueueFull = "queue_full"

	// ReasonDisabled indicates CARRIER is disabled via the feature flag.
	ReasonDisabled = "carrier_disabled"
)

// FallbackError is returned when CARRIER cannot reach SorxMain. The SORX
// engine inspects this type via IsFallback to decide whether to degrade
// gracefully to a local LLM call instead of propagating the error.
type FallbackError struct {
	// OriginalError is the underlying transport or timeout error, if any.
	OriginalError error

	// Reason is a machine-readable code describing why the fallback occurred.
	// See the Reason* constants defined in this file.
	Reason string
}

// Error implements the error interface.
func (f *FallbackError) Error() string {
	if f.OriginalError != nil {
		return fmt.Sprintf("carrier fallback (%s): %v", f.Reason, f.OriginalError)
	}
	return fmt.Sprintf("carrier fallback (%s)", f.Reason)
}

// Unwrap allows errors.Is and errors.As to inspect the original error.
func (f *FallbackError) Unwrap() error {
	return f.OriginalError
}

// IsFallback reports whether err is (or wraps) a FallbackError. The SORX
// engine calls this to distinguish degraded-mode situations from hard failures.
//
// Usage:
//
//	resp, err := carrier.Send(ctx, req)
//	if carrier.IsFallback(err) {
//	    // SorxMain unavailable — run local Tier 3/4 LLM call
//	    return localFallback(ctx, req)
//	}
//	if err != nil {
//	    return nil, err
//	}
func IsFallback(err error) bool {
	var f *FallbackError
	return errors.As(err, &f)
}

// newFallback constructs a FallbackError with the given reason and optional
// underlying error.
func newFallback(reason string, cause error) *FallbackError {
	return &FallbackError{
		Reason:        reason,
		OriginalError: cause,
	}
}
