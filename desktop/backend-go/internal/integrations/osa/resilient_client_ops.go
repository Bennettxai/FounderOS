package osa

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// GenerateApp generates an app with full resilience
func (r *ResilientClient) GenerateApp(ctx context.Context, req *AppGenerationRequest) (*AppGenerationResponse, error) {
	var resp *AppGenerationResponse
	var err error

	err = r.circuitBreaker.Execute(ctx, func() error {
		err = RetryWithBackoffTimeout(ctx, func() error {
			resp, err = r.client.GenerateApp(ctx, req)
			return err
		}, r.circuitBreaker.maxRetryTime)
		return err
	})

	if err != nil {
		slog.Error("generate app failed after retries",
			"error", err,
			"circuit_state", r.circuitBreaker.State())

		resp, fallbackErr := r.fallbackClient.GenerateAppWithFallback(ctx, req)
		if fallbackErr != nil {
			if r.circuitBreaker.State() == StateOpen {
				queueID, queueErr := r.requestQueue.Enqueue("generate_app", req, req.UserID)
				if queueErr != nil {
					slog.Error("failed to queue request", "error", queueErr)
				} else {
					slog.Info("request queued for later processing", "queue_id", queueID)
				}
			}
			return nil, fmt.Errorf("all resilience strategies failed: primary=%w, fallback=%w", err, fallbackErr)
		}

		slog.Info("fallback successful for generate app")
		return resp, nil
	}

	r.fallbackClient.cache.Set(
		r.fallbackClient.cacheKey("generate_app", req.UserID, req.WorkspaceID),
		resp,
	)
	return resp, nil
}

// GetAppStatus gets app status with full resilience
func (r *ResilientClient) GetAppStatus(ctx context.Context, appID string, userID uuid.UUID) (*AppStatusResponse, error) {
	var resp *AppStatusResponse
	var err error

	err = r.circuitBreaker.Execute(ctx, func() error {
		err = RetryWithBackoffTimeout(ctx, func() error {
			resp, err = r.client.GetAppStatus(ctx, appID, userID)
			return err
		}, r.circuitBreaker.maxRetryTime)
		return err
	})

	if err != nil {
		slog.Error("get app status failed after retries",
			"app_id", appID,
			"error", err,
			"circuit_state", r.circuitBreaker.State())

		resp, fallbackErr := r.fallbackClient.GetAppStatusWithFallback(ctx, appID, userID)
		if fallbackErr != nil {
			return nil, fmt.Errorf("all resilience strategies failed: primary=%w, fallback=%w", err, fallbackErr)
		}

		slog.Info("fallback successful for get app status")
		return resp, nil
	}

	cacheKey := r.fallbackClient.cacheKey("app_status", userID, uuid.Nil) + ":" + appID
	r.fallbackClient.cache.Set(cacheKey, resp)
	return resp, nil
}

// Orchestrate orchestrates with full resilience
func (r *ResilientClient) Orchestrate(ctx context.Context, req *OrchestrateRequest) (*OrchestrateResponse, error) {
	var resp *OrchestrateResponse
	var err error

	err = r.circuitBreaker.Execute(ctx, func() error {
		err = RetryWithBackoffTimeout(ctx, func() error {
			resp, err = r.client.Orchestrate(ctx, req)
			return err
		}, r.circuitBreaker.maxRetryTime)
		return err
	})

	if err != nil {
		slog.Error("orchestrate failed after retries",
			"error", err,
			"circuit_state", r.circuitBreaker.State())

		resp, fallbackErr := r.fallbackClient.OrchestrateWithFallback(ctx, req)
		if fallbackErr != nil {
			if r.circuitBreaker.State() == StateOpen {
				queueID, queueErr := r.requestQueue.Enqueue("orchestrate", req, req.UserID)
				if queueErr != nil {
					slog.Error("failed to queue request", "error", queueErr)
				} else {
					slog.Info("request queued for later processing", "queue_id", queueID)
				}
			}
			return nil, fmt.Errorf("all resilience strategies failed: primary=%w, fallback=%w", err, fallbackErr)
		}

		slog.Info("fallback successful for orchestrate")
		return resp, nil
	}

	r.fallbackClient.cache.Set(
		r.fallbackClient.cacheKey("orchestrate", req.UserID, req.WorkspaceID),
		resp,
	)
	return resp, nil
}

// GetWorkspaces gets workspaces with full resilience
func (r *ResilientClient) GetWorkspaces(ctx context.Context, userID uuid.UUID) (*WorkspacesResponse, error) {
	var resp *WorkspacesResponse
	var err error

	err = r.circuitBreaker.Execute(ctx, func() error {
		err = RetryWithBackoffTimeout(ctx, func() error {
			resp, err = r.client.GetWorkspaces(ctx, userID)
			return err
		}, r.circuitBreaker.maxRetryTime)
		return err
	})

	if err != nil {
		slog.Error("get workspaces failed after retries",
			"error", err,
			"circuit_state", r.circuitBreaker.State())

		resp, fallbackErr := r.fallbackClient.GetWorkspacesWithFallback(ctx, userID)
		if fallbackErr != nil {
			return nil, fmt.Errorf("all resilience strategies failed: primary=%w, fallback=%w", err, fallbackErr)
		}

		slog.Info("fallback successful for get workspaces")
		return resp, nil
	}

	r.fallbackClient.cache.Set(
		r.fallbackClient.cacheKey("workspaces", userID, uuid.Nil),
		resp,
	)
	return resp, nil
}

// Stream connects to OSA's real-time event stream with circuit breaker protection.
func (r *ResilientClient) Stream(ctx context.Context, sessionID string) (<-chan Event, error) {
	var events <-chan Event

	err := r.circuitBreaker.Execute(ctx, func() error {
		var streamErr error
		events, streamErr = r.client.Stream(ctx, sessionID)
		return streamErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "stream failed",
			"session_id", sessionID,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to connect to OSA event stream: %w", err)
	}

	return events, nil
}

// GenerateAppFromTemplate generates an app using a template with full resilience
func (r *ResilientClient) GenerateAppFromTemplate(
	ctx context.Context,
	templateName string,
	variables map[string]interface{},
	userID uuid.UUID,
	workspaceID *uuid.UUID,
) (*AppGenerationResponse, error) {
	var resp *AppGenerationResponse
	var err error

	err = r.circuitBreaker.Execute(ctx, func() error {
		err = RetryWithBackoffTimeout(ctx, func() error {
			resp, err = r.client.GenerateAppFromTemplate(ctx, templateName, variables, userID, workspaceID)
			return err
		}, r.circuitBreaker.maxRetryTime)
		return err
	})

	if err != nil {
		slog.Error("generate app from template failed after retries",
			"template", templateName,
			"error", err,
			"circuit_state", r.circuitBreaker.State())

		if r.circuitBreaker.State() == StateOpen {
			queueReq := map[string]interface{}{
				"template":  templateName,
				"variables": variables,
			}
			queueID, queueErr := r.requestQueue.Enqueue("generate_app_template", queueReq, userID)
			if queueErr != nil {
				slog.Error("failed to queue template request", "error", queueErr)
			} else {
				slog.Info("template request queued for later processing",
					"queue_id", queueID,
					"template", templateName)
			}
		}

		return nil, fmt.Errorf("template generation failed: %w", err)
	}

	cacheKey := r.fallbackClient.cacheKey("generate_app_template", userID, *workspaceID) + ":" + templateName
	r.fallbackClient.cache.Set(cacheKey, resp)
	return resp, nil
}
