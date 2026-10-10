package subconscious

import (
	"context"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/signal"
)

// slidingWindow is a thread-safe sliding window of boolean observations
// over a configurable duration. Used by the 4 in-memory proxy metrics.
type slidingWindow struct {
	mu      sync.RWMutex
	window  time.Duration
	entries []windowEntry
}

type windowEntry struct {
	at      time.Time
	success bool
}

func newSlidingWindow(window time.Duration) *slidingWindow {
	if window == 0 {
		window = 30 * time.Minute
	}
	return &slidingWindow{
		window: window,
	}
}

func (w *slidingWindow) record(success bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().UTC()
	w.entries = append(w.entries, windowEntry{at: now, success: success})
	w.prune(now)
}

func (w *slidingWindow) rate() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-w.window)
	var total, successes int
	for _, e := range w.entries {
		if e.at.After(cutoff) {
			total++
			if e.success {
				successes++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(successes) / float64(total)
}

func (w *slidingWindow) count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-w.window)
	count := 0
	for _, e := range w.entries {
		if e.at.After(cutoff) {
			count++
		}
	}
	return count
}

// prune removes entries outside the window. Must be called with w.mu held.
func (w *slidingWindow) prune(now time.Time) {
	cutoff := now.Add(-w.window)
	start := 0
	for start < len(w.entries) && w.entries[start].at.Before(cutoff) {
		start++
	}
	if start > 0 {
		w.entries = w.entries[start:]
	}
}

// ────────────────────────────────────────────────────────────────────────
// InMemoryReEncoding implements signal.ReEncodingFrequency
// ────────────────────────────────────────────────────────────────────────

// InMemoryReEncoding tracks re-encoding events in a sliding window.
type InMemoryReEncoding struct {
	sw *slidingWindow
}

// NewInMemoryReEncoding creates a new in-memory re-encoding metric.
func NewInMemoryReEncoding(window time.Duration) *InMemoryReEncoding {
	return &InMemoryReEncoding{sw: newSlidingWindow(window)}
}

func (m *InMemoryReEncoding) Name() string { return "re_encoding_frequency" }

func (m *InMemoryReEncoding) Collect(_ context.Context) (signal.MetricValue, error) {
	return signal.MetricValue{
		Name:      m.Name(),
		Value:     m.sw.rate(),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (m *InMemoryReEncoding) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

func (m *InMemoryReEncoding) Increment(_ context.Context, _ *signal.Signal, _ string) error {
	m.sw.record(true) // true = re-encoding happened (success = "event occurred")
	return nil
}

// RecordNormal records a turn that did NOT require re-encoding.
func (m *InMemoryReEncoding) RecordNormal() {
	m.sw.record(false)
}

func (m *InMemoryReEncoding) Frequency(_ context.Context, _ time.Duration) (float64, error) {
	return m.sw.rate(), nil
}

// Compile-time check.
var _ signal.ReEncodingFrequency = (*InMemoryReEncoding)(nil)

// ────────────────────────────────────────────────────────────────────────
// InMemorySignalBounce implements signal.SignalBounceRate
// ────────────────────────────────────────────────────────────────────────

// InMemorySignalBounce tracks signal bounces in a sliding window.
type InMemorySignalBounce struct {
	sw *slidingWindow
}

// NewInMemorySignalBounce creates a new in-memory signal bounce metric.
func NewInMemorySignalBounce(window time.Duration) *InMemorySignalBounce {
	return &InMemorySignalBounce{sw: newSlidingWindow(window)}
}

func (m *InMemorySignalBounce) Name() string { return "signal_bounce_rate" }

func (m *InMemorySignalBounce) Collect(_ context.Context) (signal.MetricValue, error) {
	return signal.MetricValue{
		Name:      m.Name(),
		Value:     m.sw.rate(),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (m *InMemorySignalBounce) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

func (m *InMemorySignalBounce) RecordBounce(_ context.Context, _ *signal.Signal, _, _ signal.Mode) error {
	m.sw.record(true)
	return nil
}

// RecordNoBounce records a turn that was handled without bouncing.
func (m *InMemorySignalBounce) RecordNoBounce() {
	m.sw.record(false)
}

func (m *InMemorySignalBounce) Rate(_ context.Context, _ time.Duration) (float64, error) {
	return m.sw.rate(), nil
}

// Compile-time check.
var _ signal.SignalBounceRate = (*InMemorySignalBounce)(nil)

// ────────────────────────────────────────────────────────────────────────
// InMemoryGenreRecognition implements signal.GenreRecognitionRate
// ────────────────────────────────────────────────────────────────────────

// InMemoryGenreRecognition tracks genre recognition accuracy in a sliding window.
type InMemoryGenreRecognition struct {
	sw *slidingWindow
}

// NewInMemoryGenreRecognition creates a new in-memory genre recognition metric.
func NewInMemoryGenreRecognition(window time.Duration) *InMemoryGenreRecognition {
	return &InMemoryGenreRecognition{sw: newSlidingWindow(window)}
}

func (m *InMemoryGenreRecognition) Name() string { return "genre_recognition_rate" }

func (m *InMemoryGenreRecognition) Collect(_ context.Context) (signal.MetricValue, error) {
	return signal.MetricValue{
		Name:      m.Name(),
		Value:     m.sw.rate(),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (m *InMemoryGenreRecognition) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

func (m *InMemoryGenreRecognition) RecordRecognition(_ context.Context, _ *signal.Signal, correct bool) error {
	m.sw.record(correct)
	return nil
}

func (m *InMemoryGenreRecognition) Rate(_ context.Context, _ time.Duration) (float64, error) {
	return m.sw.rate(), nil
}

// Compile-time check.
var _ signal.GenreRecognitionRate = (*InMemoryGenreRecognition)(nil)

// ────────────────────────────────────────────────────────────────────────
// InMemoryFeedbackClosure implements signal.FeedbackLoopClosure
// ────────────────────────────────────────────────────────────────────────

// InMemoryFeedbackClosure tracks feedback loop closure in a sliding window.
type InMemoryFeedbackClosure struct {
	sw *slidingWindow
}

// NewInMemoryFeedbackClosure creates a new in-memory feedback closure metric.
func NewInMemoryFeedbackClosure(window time.Duration) *InMemoryFeedbackClosure {
	return &InMemoryFeedbackClosure{sw: newSlidingWindow(window)}
}

func (m *InMemoryFeedbackClosure) Name() string { return "feedback_closure_rate" }

func (m *InMemoryFeedbackClosure) Collect(_ context.Context) (signal.MetricValue, error) {
	return signal.MetricValue{
		Name:      m.Name(),
		Value:     m.sw.rate(),
		Timestamp: time.Now().UTC(),
	}, nil
}

func (m *InMemoryFeedbackClosure) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

func (m *InMemoryFeedbackClosure) RecordFeedback(_ context.Context, _ *signal.Signal, closed bool) error {
	m.sw.record(closed)
	return nil
}

func (m *InMemoryFeedbackClosure) ClosureRate(_ context.Context, _ time.Duration) (float64, error) {
	return m.sw.rate(), nil
}

// Compile-time check.
var _ signal.FeedbackLoopClosure = (*InMemoryFeedbackClosure)(nil)
