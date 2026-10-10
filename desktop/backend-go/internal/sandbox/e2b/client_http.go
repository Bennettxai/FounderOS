package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ---- HTTP transport helpers -------------------------------------------------

// makeRequest encodes the request body (if any), sends the HTTP request, and
// decodes the response into result. A single retry pass is used for the
// underlying transport call when the error strategy allows it.
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body, result interface{}) error {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	var rawJSON []byte

	if body != nil {
		var err error
		rawJSON, err = json.Marshal(body)
		if err != nil {
			return NewValidationError(fmt.Sprintf("marshal request body: %v", err))
		}
		reqBody = bytes.NewReader(rawJSON)
	}

	const maxAttempts = 2 // internal transport-level retry
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if body != nil && attempt > 1 {
			reqBody = bytes.NewReader(rawJSON)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return NewValidationError(fmt.Sprintf("create request: %v", err))
		}

		c.applyHeaders(req, body != nil)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			e2bErr := ClassifyError(err, PhaseSetup, "")
			lastErr = e2bErr

			if attempt < maxAttempts {
				if ok, delay := ShouldRetry(e2bErr, attempt, c.retryStrategies); ok {
					c.logger.WarnContext(ctx, "transport error, retrying",
						"attempt", attempt,
						"delay", delay,
						"error", e2bErr,
					)
					if sleepErr := sleepContext(ctx, delay); sleepErr != nil {
						return sleepErr
					}
					continue
				}
			}
			return e2bErr
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return NewE2BError(ErrorTypeNetwork, "READ_ERROR",
				fmt.Sprintf("read response body: %v", readErr), true)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			e2bErr := c.classifyHTTPError(resp.StatusCode, respBody)
			lastErr = e2bErr

			if attempt < maxAttempts {
				if ok, delay := ShouldRetry(e2bErr, attempt, c.retryStrategies); ok {
					c.logger.WarnContext(ctx, "HTTP error, retrying",
						"status", resp.StatusCode,
						"attempt", attempt,
						"delay", delay,
					)
					if sleepErr := sleepContext(ctx, delay); sleepErr != nil {
						return sleepErr
					}
					continue
				}
			}
			return e2bErr
		}

		if result != nil {
			if unmarshalErr := json.Unmarshal(respBody, result); unmarshalErr != nil {
				return NewE2BError(ErrorTypeService, "PARSE_ERROR",
					fmt.Sprintf("unmarshal response: %v", unmarshalErr), false)
			}
		}
		return nil
	}

	return lastErr
}

// applyHeaders sets all standard headers on req.
func (c *Client) applyHeaders(req *http.Request, hasBody bool) {
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "BusinessOS-E2B-Client/1.0")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}
}

// classifyHTTPError converts an HTTP error status code and response body into a
// typed *E2BError.
func (c *Client) classifyHTTPError(statusCode int, respBody []byte) *E2BError {
	var errResp errorResponse
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
		switch statusCode {
		case http.StatusBadRequest:
			return NewE2BError(ErrorTypeValidation, "BAD_REQUEST", errResp.Error, false)
		case http.StatusNotFound:
			return NewE2BError(ErrorTypeValidation, "NOT_FOUND", errResp.Error, false)
		case http.StatusTooManyRequests:
			return NewE2BError(ErrorTypeRateLimit, "RATE_LIMITED", errResp.Error, true)
		case http.StatusInternalServerError:
			return NewE2BError(ErrorTypeService, "INTERNAL_ERROR", errResp.Error, true)
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return NewE2BError(ErrorTypeService, "SERVICE_UNAVAILABLE", errResp.Error, true)
		default:
			return NewE2BError(ErrorTypeService, "HTTP_ERROR", errResp.Error, statusCode >= 500)
		}
	}

	msg := fmt.Sprintf("HTTP %d: %s", statusCode, string(respBody))
	switch {
	case statusCode >= 500:
		return NewE2BError(ErrorTypeService, "SERVER_ERROR", msg, true)
	case statusCode == http.StatusTooManyRequests:
		return NewE2BError(ErrorTypeRateLimit, "RATE_LIMITED", msg, true)
	case statusCode >= 400:
		return NewE2BError(ErrorTypeValidation, "CLIENT_ERROR", msg, false)
	default:
		return NewE2BError(ErrorTypeUnknown, "HTTP_ERROR", msg, false)
	}
}

// sleepContext blocks for d or until ctx is cancelled, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
