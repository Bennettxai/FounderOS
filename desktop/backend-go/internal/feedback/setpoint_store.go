package feedback

import (
	"context"
	"fmt"
	"sync"
)

// InMemorySetpointStore is a thread-safe in-memory SetpointStore with
// sensible defaults for all 5 metrics the homeostatic loop monitors.
// Can be swapped for a database-backed store later without changing callers.
type InMemorySetpointStore struct {
	mu        sync.RWMutex
	setpoints map[string]Setpoint
}

// NewInMemorySetpointStore creates a store pre-loaded with default setpoints.
//
// Default thresholds:
//
//	action_completion_rate: 80% ± 10% (Shannon channel capacity)
//	re_encoding_frequency:  15% ± 5%  (Ashby variety failure)
//	signal_bounce_rate:     10% ± 5%  (routing failure)
//	genre_recognition_rate: 85% ± 10% (classification accuracy)
//	feedback_closure_rate:  70% ± 10% (Wiener feedback loop)
func NewInMemorySetpointStore() *InMemorySetpointStore {
	return &InMemorySetpointStore{
		setpoints: map[string]Setpoint{
			"action_completion_rate": {
				MetricName: "action_completion_rate",
				Target:     0.80,
				Tolerance:  0.10,
			},
			"re_encoding_frequency": {
				MetricName: "re_encoding_frequency",
				Target:     0.15,
				Tolerance:  0.05,
			},
			"signal_bounce_rate": {
				MetricName: "signal_bounce_rate",
				Target:     0.10,
				Tolerance:  0.05,
			},
			"genre_recognition_rate": {
				MetricName: "genre_recognition_rate",
				Target:     0.85,
				Tolerance:  0.10,
			},
			"feedback_closure_rate": {
				MetricName: "feedback_closure_rate",
				Target:     0.70,
				Tolerance:  0.10,
			},
		},
	}
}

// GetSetpoint returns the setpoint for the named metric.
func (s *InMemorySetpointStore) GetSetpoint(_ context.Context, metricName string) (Setpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sp, ok := s.setpoints[metricName]
	if !ok {
		return Setpoint{}, fmt.Errorf("setpoint not found: %s", metricName)
	}
	return sp, nil
}

// SetSetpoint upserts a setpoint. Used for runtime tuning.
func (s *InMemorySetpointStore) SetSetpoint(_ context.Context, sp Setpoint) error {
	if sp.MetricName == "" {
		return fmt.Errorf("setpoint: metric_name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setpoints[sp.MetricName] = sp
	return nil
}
