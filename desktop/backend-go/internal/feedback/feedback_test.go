package feedback

import (
	"context"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/signal"
)

func TestNewHomeostaticLoop_NilMetrics(t *testing.T) {
	ctx := context.Background()
	_, err := NewHomeostaticLoop(ctx, HomeostaticLoopConfig{
		Metrics:   nil,
		Setpoints: &mockSetpointStore{},
		Actuator:  &mockActuator{},
	})
	if err == nil {
		t.Fatal("expected error for nil Metrics")
	}
}

func TestNewHomeostaticLoop_NilSetpoints(t *testing.T) {
	ctx := context.Background()
	_, err := NewHomeostaticLoop(ctx, HomeostaticLoopConfig{
		Metrics:   &signal.ProxyMetricsRegistry{},
		Setpoints: nil,
		Actuator:  &mockActuator{},
	})
	if err == nil {
		t.Fatal("expected error for nil Setpoints")
	}
}

func TestNewHomeostaticLoop_NilActuator(t *testing.T) {
	ctx := context.Background()
	_, err := NewHomeostaticLoop(ctx, HomeostaticLoopConfig{
		Metrics:   &signal.ProxyMetricsRegistry{},
		Setpoints: &mockSetpointStore{},
		Actuator:  nil,
	})
	if err == nil {
		t.Fatal("expected error for nil Actuator")
	}
}

func TestNewAutopoieticMonitor_DefaultThresholds(t *testing.T) {
	ctx := context.Background()
	m, err := NewAutopoieticMonitor(ctx, AutopoieticMonitorConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.thresholds) != 5 {
		t.Errorf("expected 5 default thresholds, got %d", len(m.thresholds))
	}
}

func TestAutopoieticMonitor_Observe_SlidingWindow(t *testing.T) {
	ctx := context.Background()
	m, err := NewAutopoieticMonitor(ctx, AutopoieticMonitorConfig{
		WindowSize: 2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	// Add observations spanning 3 hours (exceeds 2-hour window)
	for i := 0; i < 6; i++ {
		m.Observe(ctx, MetricObservation{
			MetricName: "action_completion_rate",
			Value:      0.9 - float64(i)*0.02,
			ObservedAt: now.Add(time.Duration(i) * 30 * time.Minute),
		})
	}

	obs := m.observations["action_completion_rate"]
	// Observations older than 2 hours from the latest should be pruned
	if len(obs) > 5 {
		t.Errorf("expected sliding window to prune old observations, got %d", len(obs))
	}
}

func TestAutopoieticMonitor_Analyze_TumorSignal(t *testing.T) {
	ctx := context.Background()
	m, err := NewAutopoieticMonitor(ctx, AutopoieticMonitorConfig{
		WindowSize: 6 * time.Hour,
		Thresholds: map[string]float64{
			"action_completion_rate": 0.70,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	// Declining from 0.95 to 0.80 over 6 hours — still above 0.70 but trending down
	m.Observe(ctx, MetricObservation{
		MetricName: "action_completion_rate",
		Value:      0.95,
		ObservedAt: now.Add(-6 * time.Hour),
	})
	m.Observe(ctx, MetricObservation{
		MetricName: "action_completion_rate",
		Value:      0.80,
		ObservedAt: now,
	})

	analyses, err := m.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(analyses) != 1 {
		t.Fatalf("expected 1 analysis, got %d", len(analyses))
	}
	if !analyses[0].IsTumorSignal {
		t.Error("expected tumor signal to be detected")
	}
	if analyses[0].PredictedFailure == nil {
		t.Error("expected predicted failure time to be set")
	}
}

func TestAutopoieticMonitor_Analyze_NotEnoughData(t *testing.T) {
	ctx := context.Background()
	m, err := NewAutopoieticMonitor(ctx, AutopoieticMonitorConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one observation — not enough for trend
	m.Observe(ctx, MetricObservation{
		MetricName: "action_completion_rate",
		Value:      0.9,
		ObservedAt: time.Now(),
	})

	analyses, err := m.Analyze(ctx)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if len(analyses) != 0 {
		t.Errorf("expected 0 analyses with insufficient data, got %d", len(analyses))
	}
}

// Mock types for testing

type mockSetpointStore struct{}

func (m *mockSetpointStore) GetSetpoint(_ context.Context, metricName string) (Setpoint, error) {
	return Setpoint{MetricName: metricName, Target: 0.8, Tolerance: 0.1}, nil
}

func (m *mockSetpointStore) SetSetpoint(_ context.Context, _ Setpoint) error {
	return nil
}

type mockActuator struct{}

func (m *mockActuator) Act(_ context.Context, _ ActuatorAction, _ string, _, _ float64) error {
	return nil
}
