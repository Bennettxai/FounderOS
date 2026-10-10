package subconscious

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/signal"
)

// InMemoryTimeToDecide implements signal.TimeToDecide with a sliding window.
type InMemoryTimeToDecide struct {
	mu      sync.RWMutex
	samples []latencySample
	window  time.Duration
}

type latencySample struct {
	latency    time.Duration
	recordedAt time.Time
}

// NewInMemoryTimeToDecide creates a new InMemoryTimeToDecide with the given window.
func NewInMemoryTimeToDecide(window time.Duration) *InMemoryTimeToDecide {
	if window == 0 {
		window = 30 * time.Minute
	}
	return &InMemoryTimeToDecide{
		window: window,
	}
}

// Name returns the metric name.
func (m *InMemoryTimeToDecide) Name() string { return "time_to_decide" }

// Collect returns the current metric value (p50 latency in seconds).
func (m *InMemoryTimeToDecide) Collect(_ context.Context) (signal.MetricValue, error) {
	p50, _ := m.Percentile(context.Background(), 50, 0)
	return signal.MetricValue{
		Name:      m.Name(),
		Value:     p50.Seconds(),
		Timestamp: time.Now().UTC(),
	}, nil
}

// Register is a no-op for in-memory implementation.
func (m *InMemoryTimeToDecide) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

// RecordLatency records a decision latency sample.
func (m *InMemoryTimeToDecide) RecordLatency(_ context.Context, _ *signal.Signal, latency time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.samples = append(m.samples, latencySample{
		latency:    latency,
		recordedAt: time.Now().UTC(),
	})
	m.prune()
	return nil
}

// RecordDuration records a latency duration directly (convenience method).
func (m *InMemoryTimeToDecide) RecordDuration(latency time.Duration) {
	m.RecordLatency(context.Background(), nil, latency)
}

// Percentile returns the Nth percentile latency over the given window.
// If window is 0, uses the default window.
func (m *InMemoryTimeToDecide) Percentile(_ context.Context, n int, window time.Duration) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if window == 0 {
		window = m.window
	}
	cutoff := time.Now().UTC().Add(-window)

	var durations []time.Duration
	for _, s := range m.samples {
		if s.recordedAt.After(cutoff) {
			durations = append(durations, s.latency)
		}
	}

	if len(durations) == 0 {
		return 0, nil
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	idx := (n * len(durations)) / 100
	if idx >= len(durations) {
		idx = len(durations) - 1
	}
	return durations[idx], nil
}

// prune removes samples outside the window. Must be called with mu held.
func (m *InMemoryTimeToDecide) prune() {
	cutoff := time.Now().UTC().Add(-m.window)
	kept := m.samples[:0]
	for _, s := range m.samples {
		if s.recordedAt.After(cutoff) {
			kept = append(kept, s)
		}
	}
	m.samples = kept
}
