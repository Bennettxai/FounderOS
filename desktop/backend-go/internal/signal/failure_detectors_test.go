package signal

import (
	"context"
	"testing"
	"time"
)

func TestBandwidthOverload_NoOverload(t *testing.T) {
	d := NewBandwidthOverloadDetector(10.0)
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Detected {
		t.Error("should not detect overload with no signals")
	}
}

func TestBandwidthOverload_Overloaded(t *testing.T) {
	d := NewBandwidthOverloadDetector(0.5) // Very low threshold
	// Record many signals
	for i := 0; i < 100; i++ {
		d.RecordSignal()
	}
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Detected {
		t.Error("should detect overload with high signal rate")
	}
	if report.Severity != SeverityHigh {
		t.Errorf("expected HIGH severity, got %s", report.Severity)
	}
}

func TestFeedbackFailure_NothingToDetect(t *testing.T) {
	d := NewFeedbackFailureDetector(5 * time.Minute)
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Detected {
		t.Error("should not detect with no data")
	}
}

func TestFeedbackFailure_LowClosureRate(t *testing.T) {
	d := NewFeedbackFailureDetector(5 * time.Minute)
	// Open 10 loops, close only 2
	for i := 0; i < 10; i++ {
		id := "signal-" + string(rune('a'+i))
		d.OpenLoop(id)
	}
	d.CloseLoop("signal-a")
	d.CloseLoop("signal-b")

	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Detected {
		t.Error("should detect low closure rate")
	}
}

func TestFeedbackFailure_LoopClosed(t *testing.T) {
	d := NewFeedbackFailureDetector(5 * time.Minute)
	d.OpenLoop("test-signal")
	d.CloseLoop("test-signal")

	closed, err := d.LoopClosed(context.Background(), "test-signal")
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Error("loop should be closed")
	}
}

func TestRoutingFailure_LowBounceRate(t *testing.T) {
	d := NewRoutingFailureDetector()
	// Record mostly successful routes
	for i := 0; i < 10; i++ {
		d.RecordRouting(false)
	}
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Detected {
		t.Error("should not detect with low bounce rate")
	}
}

func TestRoutingFailure_HighBounceRate(t *testing.T) {
	d := NewRoutingFailureDetector()
	// Record mostly bounced routes
	for i := 0; i < 10; i++ {
		d.RecordRouting(true)
	}
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Detected {
		t.Error("should detect high bounce rate")
	}
}

func TestGenreMismatch_GoodRecognition(t *testing.T) {
	d := NewGenreMismatchDetector()
	for i := 0; i < 10; i++ {
		d.RecordRecognition(true)
	}
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Detected {
		t.Error("should not detect with good recognition")
	}
}

func TestGenreMismatch_PoorRecognition(t *testing.T) {
	d := NewGenreMismatchDetector()
	for i := 0; i < 10; i++ {
		d.RecordRecognition(false) // All mismatches
	}
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Detected {
		t.Error("should detect poor genre recognition")
	}
}

func TestGenreMismatch_ExpectedGenres(t *testing.T) {
	d := NewGenreMismatchDetector()
	genres, err := d.ExpectedGenres(context.Background(), ModeExecute)
	if err != nil {
		t.Fatal(err)
	}
	if len(genres) == 0 {
		t.Error("expected genres for EXECUTE mode")
	}
}

func TestBridgeFailure_NoOscillation(t *testing.T) {
	d := NewBridgeFailureDetector()
	d.RecordTransition(ModeAssist, ModeExecute)
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Detected {
		t.Error("should not detect with single transition")
	}
}

func TestBridgeFailure_Oscillation(t *testing.T) {
	d := NewBridgeFailureDetector()
	// Create A→B→A oscillation pattern
	d.RecordTransition(ModeAssist, ModeExecute)
	d.RecordTransition(ModeExecute, ModeAssist)
	d.RecordTransition(ModeAssist, ModeExecute)
	report, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Detected {
		t.Error("should detect oscillation pattern")
	}
}
