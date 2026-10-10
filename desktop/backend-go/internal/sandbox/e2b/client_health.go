package e2b

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ---- Health checking --------------------------------------------------------

// CheckHealth verifies the bridge is reachable and responding. It returns nil
// on success.
func (c *Client) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}
	c.applyHeaders(req, false)

	c.logger.DebugContext(ctx, "checking e2b bridge health", "url", c.baseURL+"/health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("e2b bridge unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("e2b bridge unhealthy: HTTP %d", resp.StatusCode)
	}

	c.logger.DebugContext(ctx, "e2b bridge is healthy")
	return nil
}

// CheckHealthWithRetry checks bridge health with exponential backoff.
func (c *Client) CheckHealthWithRetry(ctx context.Context, maxRetries int) error {
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := c.CheckHealth(ctx); err == nil {
			if attempt > 1 {
				c.logger.InfoContext(ctx, "e2b bridge became available", "attempt", attempt)
			}
			return nil
		} else {
			c.logger.WarnContext(ctx, "e2b health check failed",
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err,
			)

			if attempt == maxRetries {
				return fmt.Errorf("e2b bridge unavailable after %d attempts: %w", maxRetries, err)
			}
		}

		if err := sleepContext(ctx, backoff); err != nil {
			return err
		}
		backoff *= 2
	}

	return fmt.Errorf("e2b bridge unavailable after %d attempts", maxRetries)
}
