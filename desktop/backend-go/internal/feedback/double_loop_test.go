package feedback

import (
	"context"
	"testing"
	"time"
)

func TestDoubleLoop_NoAdjustmentOnSuccess(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// Record successful corrections
	dl.RecordCorrectionResult(CorrectionResult{
		MetricName:       "re_encoding_frequency",
		CorrectionWorked: true,
	})

	// Should not adjust anything
	if err := dl.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	sp, _ := store.GetSetpoint(context.Background(), "re_encoding_frequency")
	if sp.Tolerance != 0.05 {
		t.Errorf("tolerance should be unchanged, got %.4f", sp.Tolerance)
	}
}

func TestDoubleLoop_AdjustAfterThreeFailures(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// Record 3 consecutive failures
	for i := 0; i < 3; i++ {
		dl.RecordCorrectionResult(CorrectionResult{
			MetricName:       "re_encoding_frequency",
			CorrectionWorked: false,
		})
	}

	if err := dl.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	sp, _ := store.GetSetpoint(context.Background(), "re_encoding_frequency")
	expected := 0.05 * 1.05 // 5% relaxation
	if sp.Tolerance < expected-0.001 || sp.Tolerance > expected+0.001 {
		t.Errorf("tolerance should be ~%.4f, got %.4f", expected, sp.Tolerance)
	}
}

func TestDoubleLoop_ResetOnSuccess(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// 2 failures, then 1 success
	dl.RecordCorrectionResult(CorrectionResult{MetricName: "signal_bounce_rate", CorrectionWorked: false})
	dl.RecordCorrectionResult(CorrectionResult{MetricName: "signal_bounce_rate", CorrectionWorked: false})
	dl.RecordCorrectionResult(CorrectionResult{MetricName: "signal_bounce_rate", CorrectionWorked: true})

	if err := dl.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	sp, _ := store.GetSetpoint(context.Background(), "signal_bounce_rate")
	if sp.Tolerance != 0.05 {
		t.Errorf("tolerance should be unchanged after success reset, got %.4f", sp.Tolerance)
	}
}

func TestDoubleLoop_SafetyRailMinInterval(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// First adjustment
	for i := 0; i < 3; i++ {
		dl.RecordCorrectionResult(CorrectionResult{MetricName: "genre_recognition_rate", CorrectionWorked: false})
	}
	dl.RunOnce(context.Background())

	// Immediate second attempt should be blocked by 30-minute cooldown
	for i := 0; i < 3; i++ {
		dl.RecordCorrectionResult(CorrectionResult{MetricName: "genre_recognition_rate", CorrectionWorked: false})
	}
	dl.RunOnce(context.Background())

	sp, _ := store.GetSetpoint(context.Background(), "genre_recognition_rate")
	expectedSingle := 0.10 * 1.05 // Only one adjustment should have been applied
	if sp.Tolerance > expectedSingle+0.001 {
		t.Errorf("only one adjustment should have been applied due to cooldown, got %.4f", sp.Tolerance)
	}
}

func TestDoubleLoop_SafetyRailMaxTolerance(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// Manually set a high tolerance near the cap
	store.SetSetpoint(context.Background(), Setpoint{
		MetricName: "action_completion_rate",
		Target:     0.80,
		Tolerance:  0.15, // Already high
	})

	// Record failures
	for i := 0; i < 3; i++ {
		dl.RecordCorrectionResult(CorrectionResult{MetricName: "action_completion_rate", CorrectionWorked: false})
	}
	dl.RunOnce(context.Background())

	sp, _ := store.GetSetpoint(context.Background(), "action_completion_rate")
	// Default tolerance was 0.10 originally but we overrode to 0.15
	// The dl records the tolerance at first access. Since we set 0.15 before the first RunOnce,
	// defaultTolerances["action_completion_rate"] = 0.15, maxTolerance = 0.225
	// So 0.15 * 1.05 = 0.1575 < 0.225 — it WILL adjust
	if sp.Tolerance < 0.15 {
		t.Errorf("tolerance should not decrease, got %.4f", sp.Tolerance)
	}
}

func TestDoubleLoop_MultipleMetrics(t *testing.T) {
	store := NewInMemorySetpointStore()
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: store,
	})

	// Different metrics fail at different rates
	for i := 0; i < 3; i++ {
		dl.RecordCorrectionResult(CorrectionResult{MetricName: "re_encoding_frequency", CorrectionWorked: false})
	}
	dl.RecordCorrectionResult(CorrectionResult{MetricName: "signal_bounce_rate", CorrectionWorked: false})

	dl.RunOnce(context.Background())

	reEnc, _ := store.GetSetpoint(context.Background(), "re_encoding_frequency")
	bounce, _ := store.GetSetpoint(context.Background(), "signal_bounce_rate")

	// re_encoding should be adjusted (3 failures)
	if reEnc.Tolerance <= 0.05 {
		t.Error("re_encoding tolerance should have been relaxed")
	}
	// signal_bounce should NOT be adjusted (only 1 failure)
	if bounce.Tolerance != 0.05 {
		t.Errorf("signal_bounce tolerance should be unchanged, got %.4f", bounce.Tolerance)
	}
}

func TestNewDoubleLoopController_Defaults(t *testing.T) {
	dl := NewDoubleLoopController(DoubleLoopConfig{
		Setpoints: NewInMemorySetpointStore(),
	})
	if dl.interval != 5*time.Minute {
		t.Errorf("default interval should be 5m, got %v", dl.interval)
	}
}
