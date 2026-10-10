package governance

import (
	"context"
	"testing"

	"github.com/rhl/businessos-backend/internal/signal"
)

type mockHandler struct {
	received []AlgedonicSignal
}

func (m *mockHandler) Handle(_ context.Context, sig AlgedonicSignal) error {
	m.received = append(m.received, sig)
	return nil
}

func TestAlgedonicChannel_Fire(t *testing.T) {
	handler := &mockHandler{}
	channel := NewAlgedonicChannel(nil, handler)

	channel.Fire(context.Background(), AlgedonicSignal{
		Source:      "test",
		Description: "test signal",
		Severity:    AlgedonicCritical,
	})

	if len(handler.received) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(handler.received))
	}
	if handler.received[0].Severity != AlgedonicCritical {
		t.Errorf("expected CRITICAL, got %s", handler.received[0].Severity)
	}
}

func TestAlgedonicChannel_FireFromFailure_Critical(t *testing.T) {
	handler := &mockHandler{}
	channel := NewAlgedonicChannel(nil, handler)

	channel.FireFromFailure(context.Background(), signal.FailureReport{
		DetectorName: "bandwidth_overload",
		Detected:     true,
		Severity:     signal.SeverityCritical,
		Description:  "System overloaded",
	})

	if len(handler.received) != 1 {
		t.Fatalf("expected 1 signal for CRITICAL failure, got %d", len(handler.received))
	}
}

func TestAlgedonicChannel_FireFromFailure_High(t *testing.T) {
	handler := &mockHandler{}
	channel := NewAlgedonicChannel(nil, handler)

	channel.FireFromFailure(context.Background(), signal.FailureReport{
		DetectorName: "feedback_failure",
		Detected:     true,
		Severity:     signal.SeverityHigh,
		Description:  "Feedback loop broken",
	})

	if len(handler.received) != 1 {
		t.Fatalf("expected 1 signal for HIGH failure, got %d", len(handler.received))
	}
}

func TestAlgedonicChannel_FireFromFailure_IgnoreLow(t *testing.T) {
	handler := &mockHandler{}
	channel := NewAlgedonicChannel(nil, handler)

	channel.FireFromFailure(context.Background(), signal.FailureReport{
		DetectorName: "test",
		Detected:     true,
		Severity:     signal.SeverityLow,
		Description:  "Minor issue",
	})

	if len(handler.received) != 0 {
		t.Errorf("LOW severity should not trigger algedonic, got %d signals", len(handler.received))
	}
}

func TestAlgedonicChannel_FireFromFailure_NotDetected(t *testing.T) {
	handler := &mockHandler{}
	channel := NewAlgedonicChannel(nil, handler)

	channel.FireFromFailure(context.Background(), signal.FailureReport{
		DetectorName: "test",
		Detected:     false,
		Severity:     signal.SeverityCritical,
	})

	if len(handler.received) != 0 {
		t.Errorf("non-detected failure should not trigger algedonic, got %d signals", len(handler.received))
	}
}

func TestAlgedonicChannel_MultipleHandlers(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}
	channel := NewAlgedonicChannel(nil, h1, h2)

	channel.Fire(context.Background(), AlgedonicSignal{
		Source:      "test",
		Description: "broadcast signal",
		Severity:    AlgedonicHigh,
	})

	if len(h1.received) != 1 || len(h2.received) != 1 {
		t.Errorf("expected both handlers to receive signal: h1=%d, h2=%d",
			len(h1.received), len(h2.received))
	}
}

func TestLoggingHandler(t *testing.T) {
	// Just verify it doesn't panic
	h := NewLoggingHandler(nil)
	err := h.Handle(context.Background(), AlgedonicSignal{
		Source:      "test",
		Description: "test",
		Severity:    AlgedonicMedium,
	})
	if err != nil {
		t.Fatal(err)
	}
}
