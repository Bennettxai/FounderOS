package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/signal"
)

// Setpoint is the target value for a proxy metric.
type Setpoint struct {
	MetricName string  `json:"metric_name"`
	Target     float64 `json:"target"`
	Tolerance  float64 `json:"tolerance"`
}

// ActuatorAction is a corrective action triggered by the comparator.
type ActuatorAction string

const (
	ActionModeRebalance    ActuatorAction = "MODE_REBALANCE"
	ActionContextExpansion ActuatorAction = "CONTEXT_EXPANSION"
	ActionPromptRefinement ActuatorAction = "PROMPT_REFINEMENT"
	ActionAlert            ActuatorAction = "ALERT"
)

// CorrectionResult records whether a corrective action worked.
type CorrectionResult struct {
	MetricName       string         `json:"metric_name"`
	ActionTaken      ActuatorAction `json:"action_taken"`
	BeforeValue      float64        `json:"before_value"`
	AfterValue       float64        `json:"after_value"`
	SetpointTarget   float64        `json:"setpoint_target"`
	CorrectionWorked bool           `json:"correction_worked"`
	VerifiedAt       time.Time      `json:"verified_at"`
}

// SetpointStore loads and saves setpoints.
type SetpointStore interface {
	GetSetpoint(ctx context.Context, metricName string) (Setpoint, error)
	SetSetpoint(ctx context.Context, sp Setpoint) error
}

// Actuator applies corrective actions when metrics deviate from setpoints.
type Actuator interface {
	Act(ctx context.Context, action ActuatorAction, metricName string, currentValue, targetValue float64) error
}

// HomeostaticLoop implements the SENSOR-COMPARATOR-ACTUATOR-FEEDBACK control cycle.
type HomeostaticLoop struct {
	metrics             *signal.ProxyMetricsRegistry
	setpoints           SetpointStore
	actuator            Actuator
	interval            time.Duration
	logger              *slog.Logger
	onPersistentFailure AlgedonicCallback
	failureCounts       map[string]int // metric → consecutive correction failures
}

// AlgedonicCallback is called when persistent correction failures are detected.
// source identifies the metric, consecutiveFailures is how many consecutive failures occurred.
type AlgedonicCallback func(ctx context.Context, source string, consecutiveFailures int)

// HomeostaticLoopConfig configures the homeostatic loop.
type HomeostaticLoopConfig struct {
	Metrics             *signal.ProxyMetricsRegistry
	Setpoints           SetpointStore
	Actuator            Actuator
	Interval            time.Duration
	Logger              *slog.Logger
	OnPersistentFailure AlgedonicCallback // Optional: fires on 3+ consecutive correction failures
}

// NewHomeostaticLoop constructs a HomeostaticLoop.
func NewHomeostaticLoop(ctx context.Context, cfg HomeostaticLoopConfig) (*HomeostaticLoop, error) {
	if cfg.Metrics == nil {
		return nil, fmt.Errorf("homeostatic loop: ProxyMetricsRegistry is required")
	}
	if cfg.Setpoints == nil {
		return nil, fmt.Errorf("homeostatic loop: SetpointStore is required")
	}
	if cfg.Actuator == nil {
		return nil, fmt.Errorf("homeostatic loop: Actuator is required")
	}
	interval := cfg.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "homeostatic_loop")
	logger.InfoContext(ctx, "homeostatic loop initialized", "interval", interval)
	return &HomeostaticLoop{
		metrics:             cfg.Metrics,
		setpoints:           cfg.Setpoints,
		actuator:            cfg.Actuator,
		interval:            interval,
		logger:              logger,
		onPersistentFailure: cfg.OnPersistentFailure,
		failureCounts:       make(map[string]int),
	}, nil
}

// RunOnce executes one control cycle: SENSE -> COMPARE -> ACT -> VERIFY.
func (h *HomeostaticLoop) RunOnce(ctx context.Context) ([]CorrectionResult, error) {
	window := 5 * time.Minute
	var results []CorrectionResult

	type metricReader struct {
		name    string
		collect func() (float64, error)
	}

	// Build readers only for non-nil collectors to handle zero-value registry
	// gracefully in test scenarios.
	var readers []metricReader

	if h.metrics.ActionCompletion != nil {
		readers = append(readers, metricReader{"action_completion_rate", func() (float64, error) {
			return h.metrics.ActionCompletion.Rate(ctx, window)
		}})
	}
	if h.metrics.ReEncoding != nil {
		readers = append(readers, metricReader{"re_encoding_frequency", func() (float64, error) {
			return h.metrics.ReEncoding.Frequency(ctx, window)
		}})
	}
	if h.metrics.SignalBounce != nil {
		readers = append(readers, metricReader{"signal_bounce_rate", func() (float64, error) {
			return h.metrics.SignalBounce.Rate(ctx, window)
		}})
	}
	if h.metrics.GenreRecognition != nil {
		readers = append(readers, metricReader{"genre_recognition_rate", func() (float64, error) {
			return h.metrics.GenreRecognition.Rate(ctx, window)
		}})
	}
	if h.metrics.FeedbackClosure != nil {
		readers = append(readers, metricReader{"feedback_closure_rate", func() (float64, error) {
			return h.metrics.FeedbackClosure.ClosureRate(ctx, window)
		}})
	}

	for _, m := range readers {
		currentValue, err := m.collect()
		if err != nil {
			h.logger.WarnContext(ctx, "metric collection failed",
				"metric", m.name, "error", err)
			continue
		}

		// COMPARATOR: check against setpoint
		sp, err := h.setpoints.GetSetpoint(ctx, m.name)
		if err != nil {
			h.logger.WarnContext(ctx, "setpoint not found",
				"metric", m.name, "error", err)
			continue
		}

		deviation := currentValue - sp.Target
		if deviation < 0 {
			deviation = -deviation
		}
		if deviation <= sp.Tolerance {
			continue
		}

		h.logger.InfoContext(ctx, "metric out of range",
			"metric", m.name, "current", currentValue, "target", sp.Target,
			"deviation", deviation, "tolerance", sp.Tolerance)

		// ACTUATOR: apply correction
		action := selectAction(m.name, currentValue, sp.Target)
		if err := h.actuator.Act(ctx, action, m.name, currentValue, sp.Target); err != nil {
			h.logger.ErrorContext(ctx, "actuator failed",
				"metric", m.name, "action", action, "error", err)
			results = append(results, CorrectionResult{
				MetricName:       m.name,
				ActionTaken:      action,
				BeforeValue:      currentValue,
				SetpointTarget:   sp.Target,
				CorrectionWorked: false,
				VerifiedAt:       time.Now().UTC(),
			})
			continue
		}

		// FEEDBACK: Re-measure after actuation to verify correction worked.
		// NOTE: For metrics that need a delay before re-measurement (e.g. rate
		// calculations over a time window), AfterValue will equal BeforeValue
		// and CorrectionWorked will be false. The NEXT RunOnce cycle will
		// observe the corrected value through normal SENSOR flow.
		afterValue := currentValue
		correctionWorked := false
		if newValue, err := m.collect(); err == nil {
			afterValue = newValue
			afterDeviation := afterValue - sp.Target
			if afterDeviation < 0 {
				afterDeviation = -afterDeviation
			}
			correctionWorked = afterDeviation < deviation
		}

		results = append(results, CorrectionResult{
			MetricName:       m.name,
			ActionTaken:      action,
			BeforeValue:      currentValue,
			AfterValue:       afterValue,
			SetpointTarget:   sp.Target,
			CorrectionWorked: correctionWorked,
			VerifiedAt:       time.Now().UTC(),
		})

		// Track consecutive failures for algedonic escalation
		if correctionWorked {
			h.failureCounts[m.name] = 0
		} else {
			h.failureCounts[m.name]++
			if h.failureCounts[m.name] >= 3 && h.onPersistentFailure != nil {
				h.onPersistentFailure(ctx, m.name, h.failureCounts[m.name])
			}
		}
	}

	return results, nil
}

func selectAction(metricName string, current, target float64) ActuatorAction {
	switch {
	case metricName == "signal_bounce_rate" && current > target:
		return ActionModeRebalance
	case metricName == "re_encoding_frequency" && current > target:
		return ActionContextExpansion
	case metricName == "genre_recognition_rate" && current < target:
		return ActionPromptRefinement
	default:
		return ActionAlert
	}
}

// Run starts the homeostatic loop at the configured interval.
// Blocks until ctx is cancelled.
func (h *HomeostaticLoop) Run(ctx context.Context) error {
	h.logger.InfoContext(ctx, "homeostatic loop starting", "interval", h.interval)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			h.logger.InfoContext(ctx, "homeostatic loop stopped")
			return nil
		case <-ticker.C:
			results, err := h.RunOnce(ctx)
			if err != nil {
				h.logger.ErrorContext(ctx, "cycle failed", "error", err)
				continue
			}
			h.logger.InfoContext(ctx, "cycle complete", "corrections", len(results))
		}
	}
}
