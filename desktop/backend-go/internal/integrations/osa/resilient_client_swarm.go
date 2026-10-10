package osa

import (
	"context"
	"fmt"
	"log/slog"
)

// LaunchSwarm starts a multi-agent swarm with circuit breaker protection.
func (r *ResilientClient) LaunchSwarm(ctx context.Context, req SwarmRequest) (*SwarmResponse, error) {
	var resp *SwarmResponse
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			var callErr error
			resp, callErr = r.client.LaunchSwarm(ctx, req)
			return callErr
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("launch swarm failed after retries",
			"pattern", req.Pattern,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to launch swarm: %w", err)
	}
	return resp, nil
}

// ListSwarms returns all swarms with circuit breaker protection.
func (r *ResilientClient) ListSwarms(ctx context.Context) ([]SwarmStatus, error) {
	var swarms []SwarmStatus
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			var callErr error
			swarms, callErr = r.client.ListSwarms(ctx)
			return callErr
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("list swarms failed after retries",
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to list swarms: %w", err)
	}
	return swarms, nil
}

// GetSwarm retrieves a swarm by ID with circuit breaker protection.
func (r *ResilientClient) GetSwarm(ctx context.Context, swarmID string) (*SwarmStatus, error) {
	var status *SwarmStatus
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			var callErr error
			status, callErr = r.client.GetSwarm(ctx, swarmID)
			return callErr
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("get swarm failed after retries",
			"swarm_id", swarmID,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to get swarm: %w", err)
	}
	return status, nil
}

// CancelSwarm cancels a running swarm with circuit breaker protection.
func (r *ResilientClient) CancelSwarm(ctx context.Context, swarmID string) error {
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			return r.client.CancelSwarm(ctx, swarmID)
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("cancel swarm failed after retries",
			"swarm_id", swarmID,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return fmt.Errorf("failed to cancel swarm: %w", err)
	}
	return nil
}

// DispatchInstruction sends an instruction to a fleet agent with circuit breaker protection.
func (r *ResilientClient) DispatchInstruction(ctx context.Context, agentID string, instruction Instruction) error {
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			return r.client.DispatchInstruction(ctx, agentID, instruction)
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("dispatch instruction failed after retries",
			"agent_id", agentID,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return fmt.Errorf("failed to dispatch instruction: %w", err)
	}
	return nil
}

// ListTools returns all available tools with circuit breaker protection.
func (r *ResilientClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	var tools []ToolDefinition
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			var callErr error
			tools, callErr = r.client.ListTools(ctx)
			return callErr
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("list tools failed after retries",
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return tools, nil
}

// ExecuteTool runs a tool with circuit breaker protection.
func (r *ResilientClient) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	var result *ToolResult
	err := r.circuitBreaker.Execute(ctx, func() error {
		return RetryWithBackoffTimeout(ctx, func() error {
			var callErr error
			result, callErr = r.client.ExecuteTool(ctx, toolName, params)
			return callErr
		}, r.circuitBreaker.maxRetryTime)
	})
	if err != nil {
		slog.Error("execute tool failed after retries",
			"tool", toolName,
			"error", err,
			"circuit_state", r.circuitBreaker.State())
		return nil, fmt.Errorf("failed to execute tool: %w", err)
	}
	return result, nil
}
