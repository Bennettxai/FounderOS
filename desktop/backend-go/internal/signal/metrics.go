package signal

import (
	"context"
	"fmt"
	"time"
)

// MetricValue holds a single metric observation.
type MetricValue struct {
	Name      string            // metric name
	Value     float64           // current value
	Timestamp time.Time         // when observed
	Labels    map[string]string // additional dimensions
}

// MetricCollector is the base interface all proxy metrics implement.
type MetricCollector interface {
	// Name returns the metric's canonical name.
	Name() string
	// Collect gathers the current metric value.
	Collect(ctx context.Context) (MetricValue, error)
	// Register registers a collection point with this metric.
	Register(ctx context.Context, point CollectionPoint) error
}

// CollectionPoint is a location in the system that emits metric observations.
type CollectionPoint struct {
	ID     string            // unique identifier for this collection point
	Mode   Mode              // which OSA mode this point belongs to
	Labels map[string]string // static labels
}

// --- The 6 Proxy Metrics ---

// ActionCompletionRate measures the ratio of signals that complete their
// intended action versus signals that are abandoned, rejected, or timed out.
// Maps to: Shannon Channel Capacity (successful transmissions / total transmissions).
type ActionCompletionRate interface {
	MetricCollector
	// Record records the completion or failure of a signal action.
	Record(ctx context.Context, signal *Signal, completed bool) error
	// Rate returns the rolling completion rate over the given window.
	Rate(ctx context.Context, window time.Duration) (float64, error)
}

// ReEncodingFrequency measures how often a signal must be re-encoded or
// reformatted before it is understood. High frequency indicates genre mismatch
// or format incompatibility (Ashby variety failure).
type ReEncodingFrequency interface {
	MetricCollector
	// Increment records a re-encoding event for the given signal.
	Increment(ctx context.Context, signal *Signal, reason string) error
	// Frequency returns the rolling re-encoding rate over the given window.
	Frequency(ctx context.Context, window time.Duration) (float64, error)
}

// TimeToDecide measures the latency between a signal being received and the
// system making a decision in response. Maps to VSM System 3 response time.
type TimeToDecide interface {
	MetricCollector
	// RecordLatency records the decision latency for a signal.
	RecordLatency(ctx context.Context, signal *Signal, latency time.Duration) error
	// Percentile returns the Nth percentile latency over the given window.
	Percentile(ctx context.Context, n int, window time.Duration) (time.Duration, error)
}

// SignalBounceRate measures the ratio of signals that are rejected by the
// receiving mode and redirected (bounced) to a different mode. High bounce
// rate indicates routing failure or genre mismatch.
type SignalBounceRate interface {
	MetricCollector
	// RecordBounce records that a signal was bounced from one mode to another.
	RecordBounce(ctx context.Context, signal *Signal, fromMode, toMode Mode) error
	// Rate returns the rolling bounce rate over the given window.
	Rate(ctx context.Context, window time.Duration) (float64, error)
}

// GenreRecognitionRate measures how accurately the system identifies the
// communicative genre (Direct, Inform, Commit, Decide, Express) of incoming
// signals. Low rate indicates the signal classification pipeline is failing.
type GenreRecognitionRate interface {
	MetricCollector
	// RecordRecognition records whether the genre was correctly identified.
	RecordRecognition(ctx context.Context, signal *Signal, correct bool) error
	// Rate returns the rolling genre recognition accuracy over the given window.
	Rate(ctx context.Context, window time.Duration) (float64, error)
}

// FeedbackLoopClosure measures whether feedback signals are completing their
// loop (returning to the originating mode with confirmation or correction).
// Low closure rate indicates the Wiener feedback loop is broken.
type FeedbackLoopClosure interface {
	MetricCollector
	// RecordFeedback records whether a feedback signal successfully closed its loop.
	RecordFeedback(ctx context.Context, originSignal *Signal, closed bool) error
	// ClosureRate returns the rolling feedback closure rate over the given window.
	ClosureRate(ctx context.Context, window time.Duration) (float64, error)
}

// ProxyMetricsRegistry holds all 6 proxy metric collectors.
// Constructed via NewProxyMetricsRegistry -- no global singletons.
type ProxyMetricsRegistry struct {
	ActionCompletion ActionCompletionRate
	ReEncoding       ReEncodingFrequency
	TimeToDecide     TimeToDecide
	SignalBounce     SignalBounceRate
	GenreRecognition GenreRecognitionRate
	FeedbackClosure  FeedbackLoopClosure
}

// NewProxyMetricsRegistry constructs a metrics registry with the given collectors.
// All 6 collectors are required. Returns an error if any is nil.
func NewProxyMetricsRegistry(
	ctx context.Context,
	actionCompletion ActionCompletionRate,
	reEncoding ReEncodingFrequency,
	timeToDecide TimeToDecide,
	signalBounce SignalBounceRate,
	genreRecognition GenreRecognitionRate,
	feedbackClosure FeedbackLoopClosure,
) (*ProxyMetricsRegistry, error) {
	if actionCompletion == nil {
		return nil, fmt.Errorf("metrics: ActionCompletionRate is required")
	}
	if reEncoding == nil {
		return nil, fmt.Errorf("metrics: ReEncodingFrequency is required")
	}
	if timeToDecide == nil {
		return nil, fmt.Errorf("metrics: TimeToDecide is required")
	}
	if signalBounce == nil {
		return nil, fmt.Errorf("metrics: SignalBounceRate is required")
	}
	if genreRecognition == nil {
		return nil, fmt.Errorf("metrics: GenreRecognitionRate is required")
	}
	if feedbackClosure == nil {
		return nil, fmt.Errorf("metrics: FeedbackLoopClosure is required")
	}
	return &ProxyMetricsRegistry{
		ActionCompletion: actionCompletion,
		ReEncoding:       reEncoding,
		TimeToDecide:     timeToDecide,
		SignalBounce:     signalBounce,
		GenreRecognition: genreRecognition,
		FeedbackClosure:  feedbackClosure,
	}, nil
}
