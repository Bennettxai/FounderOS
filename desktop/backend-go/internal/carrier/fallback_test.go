package carrier

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// FallbackError.Error()
// ============================================================================

func TestFallbackError_Error_WithoutOriginalError(t *testing.T) {
	f := &FallbackError{Reason: ReasonDisabled}

	got := f.Error()

	assert.Equal(t, "carrier fallback (carrier_disabled)", got)
}

func TestFallbackError_Error_WithOriginalError(t *testing.T) {
	cause := errors.New("connection refused")
	f := &FallbackError{
		Reason:        ReasonDisconnected,
		OriginalError: cause,
	}

	got := f.Error()

	assert.Equal(t, "carrier fallback (amqp_disconnected): connection refused", got)
}

func TestFallbackError_Error_AllReasonCodes(t *testing.T) {
	tests := []struct {
		reason   string
		expected string
	}{
		{ReasonDisconnected, "carrier fallback (amqp_disconnected)"},
		{ReasonTimeout, "carrier fallback (response_timeout)"},
		{ReasonQueueFull, "carrier fallback (queue_full)"},
		{ReasonDisabled, "carrier fallback (carrier_disabled)"},
	}

	for _, tc := range tests {
		t.Run(tc.reason, func(t *testing.T) {
			f := &FallbackError{Reason: tc.reason}
			assert.Equal(t, tc.expected, f.Error())
		})
	}
}

// ============================================================================
// FallbackError.Unwrap()
// ============================================================================

func TestFallbackError_Unwrap_ReturnsOriginalError(t *testing.T) {
	cause := errors.New("underlying transport failure")
	f := &FallbackError{
		Reason:        ReasonDisconnected,
		OriginalError: cause,
	}

	unwrapped := f.Unwrap()

	assert.Equal(t, cause, unwrapped)
}

func TestFallbackError_Unwrap_ReturnsNilWhenNoOriginalError(t *testing.T) {
	f := &FallbackError{Reason: ReasonDisabled}

	unwrapped := f.Unwrap()

	assert.Nil(t, unwrapped)
}

func TestFallbackError_ErrorsIs_WorksThroughWrapping(t *testing.T) {
	sentinel := errors.New("sentinel error")
	f := &FallbackError{
		Reason:        ReasonTimeout,
		OriginalError: sentinel,
	}

	// errors.Is should traverse via Unwrap.
	assert.True(t, errors.Is(f, sentinel))
}

func TestFallbackError_ImplementsErrorInterface(t *testing.T) {
	var _ error = &FallbackError{}
}

// ============================================================================
// IsFallback
// ============================================================================

func TestIsFallback_ReturnsTrueForFallbackError(t *testing.T) {
	f := &FallbackError{Reason: ReasonDisconnected}

	assert.True(t, IsFallback(f))
}

func TestIsFallback_ReturnsTrueWhenWrapped(t *testing.T) {
	f := &FallbackError{Reason: ReasonTimeout}
	wrapped := fmt.Errorf("outer: %w", f)

	assert.True(t, IsFallback(wrapped))
}

func TestIsFallback_ReturnsFalseForNonFallbackError(t *testing.T) {
	plainErr := errors.New("plain error")

	assert.False(t, IsFallback(plainErr))
}

func TestIsFallback_ReturnsFalseForNil(t *testing.T) {
	assert.False(t, IsFallback(nil))
}

func TestIsFallback_ReturnsFalseForResponseError(t *testing.T) {
	re := &ResponseError{Code: "MCTS_TIMEOUT", Message: "timed out"}

	assert.False(t, IsFallback(re))
}

// ============================================================================
// newFallback
// ============================================================================

func TestNewFallback_ConstructsCorrectly_WithCause(t *testing.T) {
	cause := errors.New("dial tcp: connection refused")

	f := newFallback(ReasonDisconnected, cause)

	require.NotNil(t, f)
	assert.Equal(t, ReasonDisconnected, f.Reason)
	assert.Equal(t, cause, f.OriginalError)
}

func TestNewFallback_ConstructsCorrectly_WithNilCause(t *testing.T) {
	f := newFallback(ReasonDisabled, nil)

	require.NotNil(t, f)
	assert.Equal(t, ReasonDisabled, f.Reason)
	assert.Nil(t, f.OriginalError)
}

func TestNewFallback_ReturnsFallbackError(t *testing.T) {
	f := newFallback(ReasonQueueFull, nil)

	// Should satisfy IsFallback immediately.
	assert.True(t, IsFallback(f))
}

// ============================================================================
// Reason constants
// ============================================================================

func TestReasonConstants_HaveExpectedValues(t *testing.T) {
	assert.Equal(t, "amqp_disconnected", ReasonDisconnected)
	assert.Equal(t, "response_timeout", ReasonTimeout)
	assert.Equal(t, "queue_full", ReasonQueueFull)
	assert.Equal(t, "carrier_disabled", ReasonDisabled)
}
