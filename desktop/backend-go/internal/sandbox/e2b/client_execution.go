package e2b

import (
	"context"
	"fmt"
	"time"
)

// ---- High-level execution methods ------------------------------------------

// ExecuteWithRetry runs a sandbox execution test, retrying on transient errors
// up to MaxRetries times with intelligent backoff.
func (c *Client) ExecuteWithRetry(ctx context.Context, projectPath string) (*ExecutionResult, error) {
	var (
		lastResult *ExecutionResult
		lastErr    error
	)

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		c.logger.DebugContext(ctx, "sandbox execution attempt",
			"attempt", attempt,
			"max_retries", c.config.MaxRetries,
			"project_path", projectPath,
			"tenant_id", c.tenantID,
		)

		result, err := c.TestExecution(ctx, projectPath)
		if err != nil {
			e2bErr := ClassifyError(err, PhaseSetup, "")
			lastErr = e2bErr

			shouldRetry, delay := ShouldRetry(e2bErr, attempt, c.retryStrategies)
			if !shouldRetry || attempt >= c.config.MaxRetries {
				return nil, fmt.Errorf("execution failed after %d attempt(s): %w", attempt, e2bErr)
			}

			c.logger.WarnContext(ctx, "retryable error, waiting before retry",
				"attempt", attempt,
				"delay", delay,
				"error", e2bErr,
			)
			if err := sleepContext(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		lastResult = result

		if result.IsSuccess() {
			c.logger.InfoContext(ctx, "sandbox execution succeeded", "attempt", attempt)
			return result, nil
		}

		execErr := NewExecutionError(result.Phase, result.Error, result.SandboxID)
		lastErr = execErr

		shouldRetry, delay := ShouldRetry(execErr, attempt, c.retryStrategies)
		if !shouldRetry || attempt >= c.config.MaxRetries {
			break
		}

		c.logger.WarnContext(ctx, "sandbox execution failed, retrying",
			"attempt", attempt,
			"phase", result.Phase,
			"delay", delay,
		)
		if err := sleepContext(ctx, delay); err != nil {
			return nil, err
		}
	}

	return lastResult, fmt.Errorf("execution failed after %d attempt(s): %w", c.config.MaxRetries, lastErr)
}

// FixerFunc is called when execution fails to generate file patches that should
// be applied before the next attempt. It receives the failing ExecutionResult
// and must return a map of sandbox-relative file paths to their new contents.
// Returning (nil, nil) or (empty map, nil) skips the update step.
type FixerFunc func(ctx context.Context, result *ExecutionResult) (map[string]string, error)

// ExecuteWithFixLoop runs the build-test-fix retry loop. On each failure it
// calls fixer (if non-nil) to generate file patches, applies them to the
// sandbox, then retries. The loop runs up to MaxRetries times.
func (c *Client) ExecuteWithFixLoop(ctx context.Context, projectPath string, fixer FixerFunc) (*ExecutionSummary, error) {
	summary := &ExecutionSummary{
		AllResults:    make([]*ExecutionResult, 0),
		ErrorsSummary: make([]string, 0),
		FilesUpdated:  make([]string, 0),
	}

	start := time.Now()
	defer func() {
		summary.TotalDuration = time.Since(start)
	}()

	var currentSandboxID string

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		summary.TotalAttempts = attempt

		result, err := c.TestExecution(ctx, projectPath)
		if err != nil {
			return summary, fmt.Errorf("execution test on attempt %d: %w", attempt, err)
		}

		summary.AllResults = append(summary.AllResults, result)
		summary.LastResult = result
		currentSandboxID = result.SandboxID

		if result.IsSuccess() {
			summary.SuccessfulRuns = 1
			summary.FinalSandboxID = currentSandboxID
			c.logger.InfoContext(ctx, "fix-loop succeeded",
				"attempt", attempt,
				"sandbox_id", currentSandboxID,
				"tenant_id", c.tenantID,
			)
			return summary, nil
		}

		if result.HasError() {
			summary.ErrorsSummary = append(summary.ErrorsSummary, result.GetErrorDetails())
		}

		if attempt == c.config.MaxRetries || fixer == nil {
			break
		}

		fixes, err := fixer(ctx, result)
		if err != nil {
			c.logger.WarnContext(ctx, "fixer returned error, continuing to next attempt",
				"attempt", attempt,
				"error", err,
			)
			if sleepErr := sleepContext(ctx, c.config.RetryDelay); sleepErr != nil {
				return summary, sleepErr
			}
			continue
		}

		if len(fixes) == 0 {
			c.logger.DebugContext(ctx, "fixer produced no patches, continuing", "attempt", attempt)
			if sleepErr := sleepContext(ctx, c.config.RetryDelay); sleepErr != nil {
				return summary, sleepErr
			}
			continue
		}

		if currentSandboxID != "" {
			updateResult, updateErr := c.UpdateSandboxFiles(ctx, currentSandboxID, fixes)
			if updateErr != nil {
				c.logger.WarnContext(ctx, "failed to apply fixes to sandbox, will retry with fresh upload",
					"attempt", attempt,
					"sandbox_id", currentSandboxID,
					"error", updateErr,
				)
			} else {
				summary.FixesApplied++
				summary.FilesUpdated = append(summary.FilesUpdated, updateResult.UpdatedFiles...)
				c.logger.InfoContext(ctx, "applied fixes to sandbox",
					"attempt", attempt,
					"files_count", len(fixes),
					"sandbox_id", currentSandboxID,
				)
			}
		}

		if sleepErr := sleepContext(ctx, c.config.RetryDelay); sleepErr != nil {
			return summary, sleepErr
		}
	}

	summary.FinalSandboxID = currentSandboxID
	return summary, fmt.Errorf("execution failed after %d attempt(s)", c.config.MaxRetries)
}
