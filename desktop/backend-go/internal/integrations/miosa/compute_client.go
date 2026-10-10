package miosa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	miosasdk "github.com/Miosa-osa/miosa-go"
)

// MIOSAComputer is the compute resource returned by the MIOSA platform API.
// It preserves the BusinessOS handler contract while the implementation uses
// the official MIOSA Go SDK underneath.
type MIOSAComputer struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Status       string   `json:"status"`
	TemplateType string   `json:"template_type"`
	Size         string   `json:"size"`
	VMID         string   `json:"vm_id"`
	DesktopURL   string   `json:"desktop_url"`
	IPAddress    string   `json:"ip_address"`
	VCPUs        int      `json:"vcpus"`
	MemoryMB     int      `json:"memory_mb"`
	DiskGB       int      `json:"disk_gb"`
	SelectedApps []string `json:"selected_apps"`
	Resolution   string   `json:"resolution"`
	TenantID     string   `json:"tenant_id"`
	OwnerUserID  string   `json:"owner_user_id"`
	InsertedAt   string   `json:"inserted_at"`
	UpdatedAt    string   `json:"updated_at"`
	CreatedAt    string   `json:"created_at"`
}

// TerminalSession is returned by the MIOSA terminal session endpoint.
type TerminalSession struct {
	SessionID  string `json:"session_id"`
	StreamAuth string `json:"stream_auth"`
	ExpiresAt  int64  `json:"expires_at"`
}

// CreditBalance is returned by GET /api/v1/credits/balance.
type CreditBalance struct {
	Balance        int    `json:"balance_credits"`
	LifetimeEarned int    `json:"lifetime_earned"`
	LifetimeSpent  int    `json:"lifetime_spent"`
	TenantID       string `json:"tenant_id"`
}

// UserInfo is returned by GET /api/v1/auth/me.
type UserInfo struct {
	User struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	} `json:"user"`
}

// StreamToken is returned by POST /api/v1/computers/:id/stream-token.
type StreamToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// ComputeClient calls the MIOSA platform API for computer provisioning,
// terminal sessions, streams, credits, and lifecycle management.
type ComputeClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	sdk        *miosasdk.Client
}

// NewComputeClient constructs a ComputeClient.
func NewComputeClient(baseURL, apiKey string) *ComputeClient {
	if baseURL == "" {
		baseURL = "https://api.miosa.ai"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	apiBaseURL := normalizeAPIBaseURL(baseURL)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	return &ComputeClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
		sdk: miosasdk.NewClient(
			apiKey,
			miosasdk.WithBaseURL(apiBaseURL),
			miosasdk.WithHTTPClient(httpClient),
		),
	}
}

// ListComputers returns computers visible to the configured MIOSA API key.
func (c *ComputeClient) ListComputers(ctx context.Context) ([]MIOSAComputer, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	resp, err := c.sdk.Computers.List(ctx, miosasdk.ListComputersInput{})
	if err != nil {
		return nil, err
	}
	computers := make([]MIOSAComputer, 0, len(resp.Data))
	for _, item := range resp.Data {
		computers = append(computers, mapSDKComputer(item))
	}
	return computers, nil
}

// GetComputer returns a single MIOSA computer.
func (c *ComputeClient) GetComputer(ctx context.Context, id string) (*MIOSAComputer, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	comp, err := c.sdk.Computers.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	mapped := mapSDKComputer(comp.ComputerData)
	return &mapped, nil
}

// CreateComputer provisions a MIOSA computer from a template and size.
func (c *ComputeClient) CreateComputer(ctx context.Context, name, templateType, size string) (*MIOSAComputer, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	comp, err := c.sdk.Computers.Create(ctx, miosasdk.CreateComputerInput{
		Name:         name,
		TemplateType: templateType,
		Size:         miosasdk.ComputerSize(size),
	})
	if err != nil {
		return nil, err
	}
	mapped := mapSDKComputer(comp.ComputerData)
	return &mapped, nil
}

// DeleteComputer terminates a MIOSA computer.
func (c *ComputeClient) DeleteComputer(ctx context.Context, id string) error {
	if err := c.requireAPIKey(); err != nil {
		return err
	}
	return c.sdk.Computers.Delete(ctx, id)
}

// StartComputer starts a stopped MIOSA computer.
func (c *ComputeClient) StartComputer(ctx context.Context, id string) error {
	if err := c.requireAPIKey(); err != nil {
		return err
	}
	comp, err := c.sdk.Computers.Get(ctx, id)
	if err != nil {
		return err
	}
	return comp.Start(ctx)
}

// StopComputer stops a running MIOSA computer.
func (c *ComputeClient) StopComputer(ctx context.Context, id string) error {
	if err := c.requireAPIKey(); err != nil {
		return err
	}
	comp, err := c.sdk.Computers.Get(ctx, id)
	if err != nil {
		return err
	}
	return comp.Stop(ctx)
}

// ExecInVM runs a command through the MIOSA envd bridge.
func (c *ComputeClient) ExecInVM(ctx context.Context, computerIDOrSlug, command string) (string, error) {
	if err := c.requireAPIKey(); err != nil {
		return "", err
	}
	comp, err := c.sdk.Computers.Get(ctx, computerIDOrSlug)
	if err != nil {
		return "", err
	}
	result, err := comp.Bash(ctx, command)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// GetCreditBalance returns current MIOSA credit balance.
func (c *ComputeClient) GetCreditBalance(ctx context.Context) (*CreditBalance, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	balance, err := c.sdk.Credits.Balance(ctx)
	if err != nil {
		return nil, err
	}
	return &CreditBalance{Balance: balance.Balance}, nil
}

// GetCurrentUser returns the MIOSA identity attached to the API key.
func (c *ComputeClient) GetCurrentUser(ctx context.Context) (*UserInfo, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var info UserInfo
	if err := c.doJSON(ctx, http.MethodGet, "/auth/me", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetStreamToken returns a short-lived VNC stream token.
func (c *ComputeClient) GetStreamToken(ctx context.Context, computerID string) (*StreamToken, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	comp, err := c.sdk.Computers.Get(ctx, computerID)
	if err != nil {
		return nil, err
	}
	raw, err := comp.StreamToken(ctx)
	if err != nil {
		return nil, err
	}
	return parseStreamToken(raw), nil
}

// CreateTerminalSession returns terminal WebSocket auth material.
func (c *ComputeClient) CreateTerminalSession(ctx context.Context, computerID string, ttlMinutes, idleTimeoutMinutes int) (*TerminalSession, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	comp, err := c.sdk.Computers.Get(ctx, computerID)
	if err != nil {
		return nil, err
	}
	raw, err := comp.Terminal.Create(ctx, miosasdk.CreateTerminalInput{
		Cols:  120,
		Rows:  30,
		Shell: "bash",
	})
	if err != nil {
		return nil, err
	}
	return parseTerminalSession(raw), nil
}

func (c *ComputeClient) requireAPIKey() error {
	if c.apiKey == "" {
		return fmt.Errorf("MIOSA API key is required")
	}
	return nil
}

func (c *ComputeClient) doJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	var body io.Reader
	if payload != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, normalizeAPIBaseURL(c.baseURL)+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errBody struct {
			Error   string `json:"error"`
			Message string `json:"message"`
			Details string `json:"details"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Details
		if msg == "" {
			msg = errBody.Message
		}
		if msg == "" {
			msg = errBody.Error
		}
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("MIOSA API status %d: %s", resp.StatusCode, msg)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func normalizeAPIBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/api/v1") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/api") {
		return baseURL + "/v1"
	}
	return baseURL + "/api/v1"
}

func mapSDKComputer(comp miosasdk.ComputerData) MIOSAComputer {
	return MIOSAComputer{
		ID:           comp.ID,
		Name:         comp.Name,
		Slug:         comp.Slug,
		Status:       string(comp.Status),
		TemplateType: comp.TemplateType,
		Size:         string(comp.Size),
		DesktopURL:   publicURL(comp),
		IPAddress:    comp.IPAddress,
		TenantID:     comp.TenantID,
		UpdatedAt:    comp.UpdatedAt,
		CreatedAt:    comp.CreatedAt,
		InsertedAt:   comp.CreatedAt,
	}
}

func publicURL(comp miosasdk.ComputerData) string {
	if comp.PreviewDomain != "" {
		return "https://" + strings.TrimPrefix(comp.PreviewDomain, "https://")
	}
	if comp.Slug != "" {
		return "https://" + comp.Slug + ".sandbox.miosa.ai"
	}
	return ""
}

func parseStreamToken(raw map[string]interface{}) *StreamToken {
	candidates := []map[string]interface{}{raw}
	for _, key := range []string{"token", "data", "stream_token"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			candidates = append(candidates, nested)
		}
	}
	for _, item := range candidates {
		token := stringValue(item, "token", "stream_auth", "auth")
		if token == "" {
			continue
		}
		return &StreamToken{Token: token, ExpiresAt: int64Value(item, "expires_at")}
	}
	return &StreamToken{}
}

func parseTerminalSession(raw map[string]interface{}) *TerminalSession {
	candidates := []map[string]interface{}{raw}
	for _, key := range []string{"session", "data", "terminal_session"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			candidates = append(candidates, nested)
		}
	}
	for _, item := range candidates {
		sessionID := stringValue(item, "session_id", "id")
		if sessionID == "" {
			continue
		}
		return &TerminalSession{
			SessionID:  sessionID,
			StreamAuth: stringValue(item, "stream_auth", "auth", "token"),
			ExpiresAt:  int64Value(item, "expires_at"),
		}
	}
	return &TerminalSession{}
}

func stringValue(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		switch value := values[key].(type) {
		case string:
			return value
		case fmt.Stringer:
			return value.String()
		}
	}
	return ""
}

func int64Value(values map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		switch value := values[key].(type) {
		case int64:
			return value
		case int:
			return int64(value)
		case float64:
			return int64(value)
		case json.Number:
			n, _ := value.Int64()
			return n
		}
	}
	return 0
}
