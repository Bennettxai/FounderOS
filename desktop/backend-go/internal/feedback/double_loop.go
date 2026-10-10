package feedback

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// DoubleLoopController implements Argyris double-loop learning.
// When the homeostatic loop's single-loop corrections fail persistently,
// the double-loop relaxes setpoint tolerances to prevent oscillation.
//
// Safety rails:
//   - Minimum 30 minutes between adjustments per metric
//   - Maximum tolerance relaxation: 50% above default
//   - Requires 3 consecutive failed corrections before acting
type DoubleLoopController struct {
	setpoints SetpointStore
	interval  time.Duration
	logger    *slog.Logger

	mu                sync.Mutex
	failureCounts     map[string]int       // metric → consecutive failure count
	lastAdjustments   map[string]time.Time // metric → last adjustment time
	defaultTolerances map[string]float64   // metric → original tolerance (for safety cap)
}

// DoubleLoopConfig configures the double-loop controller.
type DoubleLoopConfig struct {
	Setpoints SetpointStore
	Interval  time.Duration // Default: 5 minutes
	Logger    *slog.Logger
}

// NewDoubleLoopController creates a new DoubleLoopController.
func NewDoubleLoopController(cfg DoubleLoopConfig) *DoubleLoopController {
	interval := cfg.Interval
	if interval == 0 {
		interval = 5 * time.Minute
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &DoubleLoopController{
		setpoints:         cfg.Setpoints,
		interval:          interval,
		logger:            logger.With("component", "double_loop"),
		failureCounts:     make(map[string]int),
		lastAdjustments:   make(map[string]time.Time),
		defaultTolerances: make(map[string]float64),
	}
}

// RecordCorrectionResult records whether a single-loop correction worked.
// Called by the homeostatic loop after each correction attempt.
func (dl *DoubleLoopController) RecordCorrectionResult(result CorrectionResult) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if result.CorrectionWorked {
		// Reset failure count on success
		dl.failureCounts[result.MetricName] = 0
		return
	}

	// Increment failure count
	dl.failureCounts[result.MetricName]++
	dl.logger.Info("correction failure recorded",
		"metric", result.MetricName,
		"consecutive_failures", dl.failureCounts[result.MetricName],
	)
}

// RunOnce checks for persistent failures and adjusts setpoints.
func (dl *DoubleLoopController) RunOnce(ctx context.Context) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now().UTC()

	for metric, failures := range dl.failureCounts {
		if failures < 3 {
			continue // Need 3 consecutive failures before acting
		}

		// Safety rail: minimum 30 minutes between adjustments
		if lastAdj, ok := dl.lastAdjustments[metric]; ok {
			if now.Sub(lastAdj) < 30*time.Minute {
				continue
			}
		}

		// Get current setpoint
		sp, err := dl.setpoints.GetSetpoint(ctx, metric)
		if err != nil {
			dl.logger.Warn("cannot get setpoint for double-loop",
				"metric", metric, "error", err)
			continue
		}

		// Record original tolerance if not already recorded
		if _, ok := dl.defaultTolerances[metric]; !ok {
			dl.defaultTolerances[metric] = sp.Tolerance
		}

		// Safety cap: don't relax beyond 50% of original
		maxTolerance := dl.defaultTolerances[metric] * 1.5
		newTolerance := sp.Tolerance * 1.05 // Relax by 5%
		if newTolerance > maxTolerance {
			dl.logger.Warn("tolerance cap reached, skipping adjustment",
				"metric", metric, "current", sp.Tolerance, "max", maxTolerance)
			continue
		}

		// Apply the relaxed tolerance
		sp.Tolerance = newTolerance
		if err := dl.setpoints.SetSetpoint(ctx, sp); err != nil {
			dl.logger.Error("failed to update setpoint",
				"metric", metric, "error", err)
			continue
		}

		dl.lastAdjustments[metric] = now
		dl.failureCounts[metric] = 0 // Reset after adjustment
		dl.logger.Info("double-loop: setpoint tolerance relaxed",
			"metric", metric,
			"new_tolerance", newTolerance,
			"original_tolerance", dl.defaultTolerances[metric],
		)
	}

	return nil
}

// Run starts the double-loop controller at the configured interval.
func (dl *DoubleLoopController) Run(ctx context.Context) error {
	dl.logger.InfoContext(ctx, "double-loop controller starting", "interval", dl.interval)
	ticker := time.NewTicker(dl.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			dl.logger.InfoContext(ctx, "double-loop controller stopped")
			return nil
		case <-ticker.C:
			if err := dl.RunOnce(ctx); err != nil {
				dl.logger.ErrorContext(ctx, "double-loop cycle failed", "error", err)
			}
		}
	}
}
