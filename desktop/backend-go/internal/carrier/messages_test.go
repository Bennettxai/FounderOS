package carrier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Request.sanitize
// ============================================================================

func TestRequest_Sanitize_FillsDefaultTTL(t *testing.T) {
	r := &Request{TTL: 0, Priority: DefaultPriority}
	r.sanitize()

	assert.Equal(t, DefaultTTL, r.TTL)
}

func TestRequest_Sanitize_PreservesExplicitTTL(t *testing.T) {
	r := &Request{TTL: 5000, Priority: DefaultPriority}
	r.sanitize()

	assert.Equal(t, 5000, r.TTL)
}

func TestRequest_Sanitize_NegativeTTL_ReplacedWithDefault(t *testing.T) {
	r := &Request{TTL: -1, Priority: DefaultPriority}
	r.sanitize()

	assert.Equal(t, DefaultTTL, r.TTL)
}

func TestRequest_Sanitize_ZeroPriority_SetToDefault(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: 0}
	r.sanitize()

	assert.Equal(t, DefaultPriority, r.Priority)
}

func TestRequest_Sanitize_NegativePriority_SetToDefault(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: -3}
	r.sanitize()

	assert.Equal(t, DefaultPriority, r.Priority)
}

func TestRequest_Sanitize_PriorityAboveMax_ClampedToMax(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: 99}
	r.sanitize()

	assert.Equal(t, MaxPriority, r.Priority)
}

func TestRequest_Sanitize_PriorityAtMax_Unchanged(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: MaxPriority}
	r.sanitize()

	assert.Equal(t, MaxPriority, r.Priority)
}

func TestRequest_Sanitize_PriorityAtMin_Unchanged(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: MinPriority}
	r.sanitize()

	assert.Equal(t, MinPriority, r.Priority)
}

func TestRequest_Sanitize_MidRangePriority_Unchanged(t *testing.T) {
	r := &Request{TTL: DefaultTTL, Priority: 7}
	r.sanitize()

	assert.Equal(t, 7, r.Priority)
}

// Table-driven: priority clamping edge cases.
func TestRequest_Sanitize_PriorityBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		inputPriority int
		wantPriority  int
	}{
		{"below min by 1", MinPriority - 1, DefaultPriority},
		{"at min", MinPriority, MinPriority},
		{"at max", MaxPriority, MaxPriority},
		{"above max by 1", MaxPriority + 1, MaxPriority},
		{"far above max", 100, MaxPriority},
		{"far below min", -100, DefaultPriority},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Request{TTL: DefaultTTL, Priority: tc.inputPriority}
			r.sanitize()
			assert.Equal(t, tc.wantPriority, r.Priority,
				"Priority(%d) should become %d after sanitize", tc.inputPriority, tc.wantPriority)
		})
	}
}

// ============================================================================
// ResponseError
// ============================================================================

func TestResponseError_Error_WithoutDetails(t *testing.T) {
	e := &ResponseError{
		Code:    "MCTS_TIMEOUT",
		Message: "search timed out",
	}

	got := e.Error()

	assert.Equal(t, "MCTS_TIMEOUT: search timed out", got)
}

func TestResponseError_Error_WithDetails(t *testing.T) {
	e := &ResponseError{
		Code:    "PLAN_FAILED",
		Message: "could not generate plan",
		Details: "no valid action sequence found",
	}

	got := e.Error()

	assert.Equal(t, "PLAN_FAILED: could not generate plan (no valid action sequence found)", got)
}

func TestResponseError_ImplementsErrorInterface(t *testing.T) {
	var _ error = &ResponseError{}
}

func TestResponseError_Error_EmptyCode(t *testing.T) {
	e := &ResponseError{
		Code:    "",
		Message: "something went wrong",
	}

	got := e.Error()

	assert.Equal(t, ": something went wrong", got)
}

func TestResponseError_Error_EmptyDetailsIsOmitted(t *testing.T) {
	e := &ResponseError{
		Code:    "ERR",
		Message: "bad",
		Details: "",
	}

	got := e.Error()

	// When Details is empty the parenthetical should not appear.
	assert.NotContains(t, got, "(")
}

// ============================================================================
// Constants
// ============================================================================

func TestMessageConstants_DefaultTTL(t *testing.T) {
	assert.Equal(t, 60_000, DefaultTTL)
}

func TestMessageConstants_PriorityRange(t *testing.T) {
	assert.Equal(t, 1, MinPriority)
	assert.Equal(t, 5, DefaultPriority)
	assert.Equal(t, 10, MaxPriority)
	assert.Less(t, MinPriority, DefaultPriority)
	assert.Less(t, DefaultPriority, MaxPriority)
}
