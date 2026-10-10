package osa

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client talks directly to the configured OSA runtime over HTTP. BusinessOS
// owns this adapter so runtime changes stay explicit at the integration edge.
type Client struct {
	config     *Config
	httpClient *http.Client
	token      string
}

// NewClient creates a new OSA client for a local/runtime endpoint.
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return newHTTPClient(config, config.SharedSecret), nil
}

func newHTTPClient(config *Config, token string) *Client {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token: token,
	}
}

// GenerateApp triggers application generation in OSA.
func (c *Client) GenerateApp(ctx context.Context, req *AppGenerationRequest) (*AppGenerationResponse, error) {
	payload := map[string]interface{}{
		"user_id":      req.UserID.String(),
		"workspace_id": req.WorkspaceID.String(),
		"name":         req.Name,
		"description":  req.Description,
		"type":         req.Type,
		"parameters":   req.Parameters,
	}

	var resp AppGenerationResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/generate", payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to generate app: %w", err)
	}
	return &resp, nil
}

// GetAppStatus retrieves the status of an app generation.
func (c *Client) GetAppStatus(ctx context.Context, appID string, userID uuid.UUID) (*AppStatusResponse, error) {
	_ = userID // User scoping is enforced by the runtime authentication token.

	var raw struct {
		AppID       string                 `json:"app_id"`
		Status      string                 `json:"status"`
		Progress    float64                `json:"progress"`
		CurrentStep string                 `json:"current_step,omitempty"`
		Output      interface{}            `json:"output,omitempty"`
		Error       string                 `json:"error,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		UpdatedAt   time.Time              `json:"updated_at"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/apps/"+url.PathEscape(appID)+"/status", nil, &raw); err != nil {
		return nil, fmt.Errorf("failed to get app status: %w", err)
	}

	output := ""
	switch v := raw.Output.(type) {
	case string:
		output = v
	case nil:
		output = ""
	default:
		if b, err := json.Marshal(v); err == nil {
			output = string(b)
		}
	}

	return &AppStatusResponse{
		AppID:       raw.AppID,
		Status:      raw.Status,
		Progress:    raw.Progress,
		CurrentStep: raw.CurrentStep,
		Output:      output,
		Error:       raw.Error,
		Metadata:    raw.Metadata,
		UpdatedAt:   raw.UpdatedAt,
	}, nil
}

// Orchestrate triggers the full agent orchestration workflow.
func (c *Client) Orchestrate(ctx context.Context, req *OrchestrateRequest) (*OrchestrateResponse, error) {
	payload := map[string]interface{}{
		"user_id":    req.UserID.String(),
		"input":      req.Input,
		"session_id": req.SessionID,
		"phase":      req.Phase,
		"context":    req.Context,
	}
	if req.WorkspaceID != uuid.Nil {
		payload["workspace_id"] = req.WorkspaceID.String()
	}

	var raw struct {
		Success     bool                   `json:"success"`
		Output      string                 `json:"output"`
		AgentsUsed  []string               `json:"agents_used,omitempty"`
		ExecutionMS int64                  `json:"execution_ms"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		NextStep    string                 `json:"next_step,omitempty"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/orchestrate", payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to orchestrate: %w", err)
	}

	return &OrchestrateResponse{
		Success:       raw.Success,
		Output:        raw.Output,
		AgentsUsed:    raw.AgentsUsed,
		ExecutionTime: raw.ExecutionMS,
		Data:          raw.Metadata,
		NextStep:      raw.NextStep,
	}, nil
}

// GetWorkspaces retrieves the list of workspaces for a user.
func (c *Client) GetWorkspaces(ctx context.Context, userID uuid.UUID) (*WorkspacesResponse, error) {
	_ = userID // User scoping is enforced by the runtime authentication token.

	var resp WorkspacesResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/workspaces", nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get workspaces: %w", err)
	}
	return &resp, nil
}

// HealthCheck checks if OSA is healthy and reachable.
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, &resp); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	return &resp, nil
}

// GenerateAppFromTemplate generates an app using a predefined template.
func (c *Client) GenerateAppFromTemplate(
	ctx context.Context,
	templateName string,
	variables map[string]interface{},
	userID uuid.UUID,
	workspaceID *uuid.UUID,
) (*AppGenerationResponse, error) {
	payload := map[string]interface{}{
		"template_name": templateName,
		"variables":     variables,
		"user_id":       userID.String(),
	}
	if workspaceID != nil {
		payload["workspace_id"] = workspaceID.String()
	}

	var resp AppGenerationResponse
	path := "/api/v1/templates/" + url.PathEscape(templateName) + "/generate"
	if err := c.doJSON(ctx, http.MethodPost, path, payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to generate app from template: %w", err)
	}
	return &resp, nil
}

// Stream connects to OSA's real-time event stream for a session.
func (c *Client) Stream(ctx context.Context, sessionID string) (<-chan Event, error) {
	path := "/api/v1/stream/" + url.PathEscape(sessionID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return nil, c.decodeAPIError(resp)
	}

	events := make(chan Event, 64)
	go func() {
		defer close(events)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		eventType := ""
		var dataLines []string
		flush := func() {
			if eventType == "" && len(dataLines) == 0 {
				return
			}
			payload := strings.Join(dataLines, "\n")
			evt := Event{Type: eventType, Data: map[string]interface{}{}}
			if payload != "" {
				if err := json.Unmarshal([]byte(payload), &evt.Data); err != nil {
					evt.Data["text"] = payload
				}
			}
			if evt.Type == "" {
				if typ, _ := evt.Data["type"].(string); typ != "" {
					evt.Type = typ
				} else {
					evt.Type = "message"
				}
			}
			select {
			case events <- evt:
			case <-ctx.Done():
			}
			eventType = ""
			dataLines = dataLines[:0]
		}

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				flush()
				if ctx.Err() != nil {
					return
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "event:"):
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		flush()
	}()

	return events, nil
}

// LaunchSwarm starts a multi-agent swarm with the given pattern and task.
func (c *Client) LaunchSwarm(ctx context.Context, req SwarmRequest) (*SwarmResponse, error) {
	var resp SwarmResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/swarm/launch", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to launch swarm: %w", err)
	}
	return &resp, nil
}

// ListSwarms returns all swarms for the current session.
func (c *Client) ListSwarms(ctx context.Context) ([]SwarmStatus, error) {
	var raw struct {
		Swarms []SwarmStatus `json:"swarms"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/swarm", nil, &raw); err != nil {
		return nil, fmt.Errorf("failed to list swarms: %w", err)
	}
	return raw.Swarms, nil
}

// GetSwarm retrieves the status of a specific swarm by ID.
func (c *Client) GetSwarm(ctx context.Context, swarmID string) (*SwarmStatus, error) {
	var resp SwarmStatus
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/swarm/"+url.PathEscape(swarmID), nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get swarm: %w", err)
	}
	return &resp, nil
}

// CancelSwarm cancels a running swarm.
func (c *Client) CancelSwarm(ctx context.Context, swarmID string) error {
	if err := c.doJSON(ctx, http.MethodDelete, "/api/v1/swarm/"+url.PathEscape(swarmID), nil, nil); err != nil {
		return fmt.Errorf("failed to cancel swarm: %w", err)
	}
	return nil
}

// DispatchInstruction sends an instruction directly to a specific fleet agent.
func (c *Client) DispatchInstruction(ctx context.Context, agentID string, instruction Instruction) error {
	path := "/api/v1/fleet/" + url.PathEscape(agentID) + "/dispatch"
	if err := c.doJSON(ctx, http.MethodPost, path, instruction, nil); err != nil {
		return fmt.Errorf("failed to dispatch instruction: %w", err)
	}
	return nil
}

// ListTools returns all available tools (built-in + TOOL.md + MCP).
func (c *Client) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	var raw struct {
		Tools []ToolDefinition `json:"tools"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/tools", nil, &raw); err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return raw.Tools, nil
}

// ExecuteTool runs a specific tool by name with the given parameters.
func (c *Client) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	var resp ToolResult
	path := "/api/v1/tools/" + url.PathEscape(toolName) + "/execute"
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]interface{}{"parameters": params}, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute tool: %w", err)
	}
	return &resp, nil
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	var body io.Reader
	if payload != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
		body = &buf
	}

	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return c.decodeAPIError(resp)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	base := strings.TrimRight(c.config.BaseURL, "/")
	if base == "" {
		return nil, fmt.Errorf("OSA base URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func (c *Client) decodeAPIError(resp *http.Response) error {
	var errorBody ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&errorBody)
	details := errorBody.Details
	if details == "" {
		details = errorBody.Error
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		ErrorCode:  errorBody.Code,
		Details:    details,
	}
}
