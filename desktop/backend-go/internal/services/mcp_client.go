package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MCPClientTool represents a tool discovered from an external MCP server
type MCPClientTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPClient connects to an external MCP server via SSE/HTTP and discovers+executes tools
type MCPClient struct {
	serverURL  string
	authType   string // "none" | "api_key" | "bearer"
	authToken  string
	headers    map[string]string
	httpClient *http.Client
}

// NewMCPClient creates a new MCP client for connecting to an external MCP server
func NewMCPClient(serverURL, authType, authToken string, customHeaders map[string]string) *MCPClient {
	return &MCPClient{
		serverURL: strings.TrimRight(serverURL, "/"),
		authType:  authType,
		authToken: authToken,
		headers:   customHeaders,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     5 * time.Minute,
				DisableCompression:  false,
				MaxConnsPerHost:     5,
				MaxIdleConnsPerHost: 5,
			},
		},
	}
}

// ValidateMCPServerURL checks that a URL is safe to connect to (SSRF protection)
func ValidateMCPServerURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	// Protocol check
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https protocols are allowed")
	}

	// Must have a hostname
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Block cloud metadata endpoints explicitly
	if hostname == "169.254.169.254" || hostname == "metadata.google.internal" {
		return fmt.Errorf("cloud metadata endpoints are not allowed")
	}

	// Resolve hostname to IP and check against blocklist
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("cannot resolve hostname: %s", hostname)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() {
			return fmt.Errorf("loopback addresses are not allowed")
		}
		if ip.IsPrivate() {
			return fmt.Errorf("private network addresses are not allowed")
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("link-local addresses are not allowed")
		}
		if ip.IsUnspecified() {
			return fmt.Errorf("unspecified addresses are not allowed")
		}
	}

	return nil
}

// DiscoverTools connects to the MCP server and discovers available tools
func (c *MCPClient) DiscoverTools(ctx context.Context) ([]MCPClientTool, error) {
	// MCP protocol: POST to the server with a tools/list request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("MCP server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check if the response is SSE (text/event-stream) or JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.parseSSEToolsResponse(resp.Body)
	}

	// Direct JSON response
	return c.parseJSONToolsResponse(resp.Body)
}

// parseSSEToolsResponse parses an SSE stream for the tools/list response
func (c *MCPClient) parseSSEToolsResponse(body io.Reader) ([]MCPClientTool, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max

	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		} else if line == "" && len(dataLines) > 0 {
			// End of event — process accumulated data
			data := strings.Join(dataLines, "\n")
			dataLines = nil

			tools, err := c.parseToolsFromJSON([]byte(data))
			if err == nil {
				return tools, nil
			}
			// If parsing fails, continue reading more events
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSE stream: %w", err)
	}

	return nil, fmt.Errorf("no tools/list response found in SSE stream")
}

// parseJSONToolsResponse parses a direct JSON response for tools/list
func (c *MCPClient) parseJSONToolsResponse(body io.Reader) ([]MCPClientTool, error) {
	limitedBody := io.LimitReader(body, 1024*1024) // 1MB max
	data, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.parseToolsFromJSON(data)
}

// parseToolsFromJSON extracts tools from a JSON-RPC response
func (c *MCPClient) parseToolsFromJSON(data []byte) ([]MCPClientTool, error) {
	var rpcResp struct {
		Result struct {
			Tools []MCPClientTool `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP server error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result.Tools, nil
}

// ExecuteTool calls a tool on the external MCP server
func (c *MCPClient) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("MCP tool call returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check if SSE or JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.parseSSEToolCallResponse(resp.Body)
	}

	return c.parseJSONToolCallResponse(resp.Body)
}

// parseSSEToolCallResponse parses SSE stream for tool call result
func (c *MCPClient) parseSSEToolCallResponse(body io.Reader) (interface{}, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		} else if line == "" && len(dataLines) > 0 {
			data := strings.Join(dataLines, "\n")
			dataLines = nil

			result, err := c.parseToolCallFromJSON([]byte(data))
			if err == nil {
				return result, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSE stream: %w", err)
	}

	return nil, fmt.Errorf("no tool call response found in SSE stream")
}

// parseJSONToolCallResponse parses a direct JSON response for tool call
func (c *MCPClient) parseJSONToolCallResponse(body io.Reader) (interface{}, error) {
	limitedBody := io.LimitReader(body, 1024*1024) // 1MB max
	data, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.parseToolCallFromJSON(data)
}

// parseToolCallFromJSON extracts tool call result from JSON-RPC
func (c *MCPClient) parseToolCallFromJSON(data []byte) (interface{}, error) {
	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP tool error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// HealthCheck performs a basic health check on the MCP server
func (c *MCPClient) HealthCheck(ctx context.Context) error {
	// Try to discover tools as a health check — if it works, the server is healthy
	_, err := c.DiscoverTools(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	slog.Debug("MCP server health check passed", "url", c.serverURL)
	return nil
}

// setHeaders applies auth and custom headers to the request
func (c *MCPClient) setHeaders(req *http.Request) {
	// Auth headers
	switch c.authType {
	case "api_key":
		if c.authToken != "" {
			req.Header.Set("X-API-Key", c.authToken)
		}
	case "bearer":
		if c.authToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		}
	}

	// Custom headers
	for key, value := range c.headers {
		// Prevent overriding security-critical headers
		lower := strings.ToLower(key)
		if lower == "host" || lower == "content-length" || lower == "transfer-encoding" {
			continue
		}
		req.Header.Set(key, value)
	}
}
