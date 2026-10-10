package subconscious

import (
	"context"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/signal"
)

// MetricEmitter bridges ExtractedPatterns to the proxy metric
// implementations and feeds observations into the AutopoieticMonitor.
type MetricEmitter struct {
	reEncoding       *InMemoryReEncoding
	signalBounce     *InMemorySignalBounce
	genreRecognition *InMemoryGenreRecognition
	feedbackClosure  *InMemoryFeedbackClosure
	timeToDecide     *InMemoryTimeToDecide
	autopoietic      *feedback.AutopoieticMonitor
	logger           *slog.Logger
}

// NewMetricEmitter creates a new MetricEmitter.
func NewMetricEmitter(
	reEncoding *InMemoryReEncoding,
	signalBounce *InMemorySignalBounce,
	genreRecognition *InMemoryGenreRecognition,
	feedbackClosure *InMemoryFeedbackClosure,
	autopoietic *feedback.AutopoieticMonitor,
	logger *slog.Logger,
) *MetricEmitter {
	if logger == nil {
		logger = slog.Default()
	}
	return &MetricEmitter{
		reEncoding:       reEncoding,
		signalBounce:     signalBounce,
		genreRecognition: genreRecognition,
		feedbackClosure:  feedbackClosure,
		autopoietic:      autopoietic,
		logger:           logger.With("component", "metric_emitter"),
	}
}

// SetTimeToDecide sets the TimeToDecide metric collector (wired after construction).
func (me *MetricEmitter) SetTimeToDecide(ttd *InMemoryTimeToDecide) {
	me.timeToDecide = ttd
}

// Emit records extracted patterns into the proxy metrics and feeds the AutopoieticMonitor.
func (me *MetricEmitter) Emit(ctx context.Context, patterns ExtractedPatterns, classification ClassificationResult, latency time.Duration) {
	now := time.Now().UTC()

	// 1. Re-encoding frequency
	if me.reEncoding != nil {
		if patterns.IsReEncoding {
			_ = me.reEncoding.Increment(ctx, nil, "rephrase")
		} else {
			me.reEncoding.RecordNormal()
		}
		if me.autopoietic != nil {
			val, _ := me.reEncoding.Frequency(ctx, 0)
			me.autopoietic.Observe(ctx, feedback.MetricObservation{
				MetricName: "re_encoding_frequency",
				Value:      val,
				ObservedAt: now,
			})
		}
	}

	// 2. Signal bounce rate
	if me.signalBounce != nil {
		if patterns.IsBounce {
			_ = me.signalBounce.RecordBounce(ctx, nil, signal.ModeAssist, signal.ModeExecute)
		} else {
			me.signalBounce.RecordNoBounce()
		}
		if me.autopoietic != nil {
			val, _ := me.signalBounce.Rate(ctx, 0)
			me.autopoietic.Observe(ctx, feedback.MetricObservation{
				MetricName: "signal_bounce_rate",
				Value:      val,
				ObservedAt: now,
			})
		}
	}

	// 3. Genre recognition rate
	if me.genreRecognition != nil {
		_ = me.genreRecognition.RecordRecognition(ctx, nil, patterns.IsGenreMatch)
		if me.autopoietic != nil {
			val, _ := me.genreRecognition.Rate(ctx, 0)
			me.autopoietic.Observe(ctx, feedback.MetricObservation{
				MetricName: "genre_recognition_rate",
				Value:      val,
				ObservedAt: now,
			})
		}
	}

	// 4. Feedback closure rate
	if me.feedbackClosure != nil {
		_ = me.feedbackClosure.RecordFeedback(ctx, nil, patterns.IsFeedbackClosed)
		if me.autopoietic != nil {
			val, _ := me.feedbackClosure.ClosureRate(ctx, 0)
			me.autopoietic.Observe(ctx, feedback.MetricObservation{
				MetricName: "feedback_closure_rate",
				Value:      val,
				ObservedAt: now,
			})
		}
	}

	// 5. Time to decide (latency from message receipt to first SSE token)
	if me.timeToDecide != nil && latency > 0 {
		me.timeToDecide.RecordDuration(latency)
		if me.autopoietic != nil {
			val, _ := me.timeToDecide.Percentile(ctx, 50, 0)
			me.autopoietic.Observe(ctx, feedback.MetricObservation{
				MetricName: "time_to_decide",
				Value:      val.Seconds(),
				ObservedAt: now,
			})
		}
	}

	me.logger.Info("metrics emitted",
		"re_encoding", patterns.IsReEncoding,
		"bounce", patterns.IsBounce,
		"genre_match", patterns.IsGenreMatch,
		"feedback_closed", patterns.IsFeedbackClosed,
		"genre", classification.Genre,
		"weight", classification.Weight,
		"latency_ms", latency.Milliseconds(),
	)
}
