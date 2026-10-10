package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MetricObservation is a single time-stamped metric reading.
type MetricObservation struct {
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	ObservedAt time.Time `json:"observed_at"`
}

// TrendAnalysis is the result of analyzing a metric's trajectory.
type TrendAnalysis struct {
	MetricName       string        `json:"metric_name"`
	WindowDuration   time.Duration `json:"window_duration"`
	Observations     int           `json:"observations"`
	CurrentValue     float64       `json:"current_value"`
	StartValue       float64       `json:"start_value"`
	SlopePerHour     float64       `json:"slope_per_hour"`
	PredictedFailure *time.Time    `json:"predicted_failure,omitempty"`
	IsTumorSignal    bool          `json:"is_tumor_signal"`
	FailureThreshold float64       `json:"failure_threshold"`
}

// AutopoieticMonitor tracks metric trajectories and detects degradation trends
// before they reach failure thresholds. All public methods are safe for
// concurrent use.
type AutopoieticMonitor struct {
	mu           sync.RWMutex
	windowSize   time.Duration
	observations map[string][]MetricObservation
	thresholds   map[string]float64
	logger       *slog.Logger
}

// AutopoieticMonitorConfig configures the autopoietic monitor.
type AutopoieticMonitorConfig struct {
	WindowSize time.Duration
	Thresholds map[string]float64
	Logger     *slog.Logger
}

// NewAutopoieticMonitor constructs an AutopoieticMonitor.
func NewAutopoieticMonitor(ctx context.Context, cfg AutopoieticMonitorConfig) (*AutopoieticMonitor, error) {
	windowSize := cfg.WindowSize
	if windowSize == 0 {
		windowSize = 6 * time.Hour
	}
	thresholds := cfg.Thresholds
	if thresholds == nil {
		thresholds = map[string]float64{
			"action_completion_rate": 0.70,
			"re_encoding_frequency":  0.30,
			"signal_bounce_rate":     0.20,
			"genre_recognition_rate": 0.80,
			"feedback_closure_rate":  0.75,
		}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "autopoietic_monitor")
	logger.InfoContext(ctx, "autopoietic monitor initialized",
		"window_size", windowSize, "threshold_count", len(thresholds))
	return &AutopoieticMonitor{
		windowSize:   windowSize,
		observations: make(map[string][]MetricObservation),
		thresholds:   thresholds,
		logger:       logger,
	}, nil
}

// Observe records a new metric observation, maintaining the sliding window.
func (m *AutopoieticMonitor) Observe(_ context.Context, obs MetricObservation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	window := m.observations[obs.MetricName]
	window = append(window, obs)
	cutoff := obs.ObservedAt.Add(-m.windowSize)
	start := 0
	for start < len(window) && window[start].ObservedAt.Before(cutoff) {
		start++
	}
	m.observations[obs.MetricName] = window[start:]
}

// Analyze computes the trajectory analysis for all observed metrics.
func (m *AutopoieticMonitor) Analyze(ctx context.Context) ([]TrendAnalysis, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var analyses []TrendAnalysis
	for metricName, obs := range m.observations {
		if len(obs) < 2 {
			continue
		}
		analysis, err := m.analyzeTrend(ctx, metricName, obs)
		if err != nil {
			m.logger.WarnContext(ctx, "trend analysis failed",
				"metric", metricName, "error", err)
			continue
		}
		analyses = append(analyses, analysis)

		if analysis.IsTumorSignal {
			m.logger.WarnContext(ctx, "TUMOR SIGNAL DETECTED",
				"metric", metricName,
				"current_value", analysis.CurrentValue,
				"slope_per_hour", analysis.SlopePerHour,
				"predicted_failure", analysis.PredictedFailure)
		}
	}
	return analyses, nil
}

func (m *AutopoieticMonitor) analyzeTrend(_ context.Context, metricName string, obs []MetricObservation) (TrendAnalysis, error) {
	if len(obs) == 0 {
		return TrendAnalysis{}, fmt.Errorf("autopoiesis: no observations for %s", metricName)
	}

	first := obs[0]
	last := obs[len(obs)-1]
	durationHours := last.ObservedAt.Sub(first.ObservedAt).Hours()
	if durationHours == 0 {
		return TrendAnalysis{}, fmt.Errorf("autopoiesis: zero duration window for %s", metricName)
	}

	slopePerHour := (last.Value - first.Value) / durationHours
	threshold := m.thresholds[metricName]

	analysis := TrendAnalysis{
		MetricName:       metricName,
		WindowDuration:   m.windowSize,
		Observations:     len(obs),
		CurrentValue:     last.Value,
		StartValue:       first.Value,
		SlopePerHour:     slopePerHour,
		FailureThreshold: threshold,
	}

	isHighGoodMetric := metricName == "action_completion_rate" ||
		metricName == "genre_recognition_rate" ||
		metricName == "feedback_closure_rate"

	if isHighGoodMetric && last.Value > threshold && slopePerHour < 0 {
		hoursToFailure := (last.Value - threshold) / (-slopePerHour)
		if hoursToFailure > 0 && hoursToFailure < float64(m.windowSize.Hours())*2 {
			failureTime := last.ObservedAt.Add(time.Duration(hoursToFailure * float64(time.Hour)))
			analysis.PredictedFailure = &failureTime
			analysis.IsTumorSignal = true
		}
	}

	return analysis, nil
}
