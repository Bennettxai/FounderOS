package e2b

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ---- Domain types -----------------------------------------------------------

// ExecutionConfig controls sandbox lifecycle behaviour.
type ExecutionConfig struct {
	// Timeout is the HTTP-level deadline for a single sandbox operation.
	Timeout time.Duration

	// MaxRetries is the number of build-test-fix iterations in ExecuteWithFixLoop.
	MaxRetries int

	// RetryDelay is a fixed pause between fix-loop iterations (not used for
	// transient HTTP errors; those follow the RetryStrategy backoff).
	RetryDelay time.Duration

	// KeepSandbox instructs the bridge to leave the sandbox running after
	// execution so callers can inspect its state.
	KeepSandbox bool
}

// DefaultExecutionConfig returns sensible defaults.
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		Timeout:     10 * time.Minute,
		MaxRetries:  3,
		RetryDelay:  3 * time.Second,
		KeepSandbox: false,
	}
}

// ExecutionResult holds the outcome of a single sandbox run.
type ExecutionResult struct {
	SandboxID string         `json:"sandbox_id"`
	Success   bool           `json:"success"`
	Phase     ExecutionPhase `json:"phase"`
	Error     string         `json:"error,omitempty"`
	Stdout    string         `json:"stdout,omitempty"`
	Stderr    string         `json:"stderr,omitempty"`
}

// IsSuccess reports whether the sandbox run completed without errors.
func (r *ExecutionResult) IsSuccess() bool {
	return r != nil && r.Success && r.Error == ""
}

// HasError reports whether the result contains an error description.
func (r *ExecutionResult) HasError() bool {
	return r != nil && r.Error != ""
}

// GetErrorDetails returns a human-readable string summarising the error.
func (r *ExecutionResult) GetErrorDetails() string {
	if r == nil {
		return ""
	}
	if r.Error != "" && r.Stderr != "" {
		return fmt.Sprintf("[%s] %s\nSTDERR: %s", r.Phase, r.Error, r.Stderr)
	}
	if r.Error != "" {
		return fmt.Sprintf("[%s] %s", r.Phase, r.Error)
	}
	return ""
}

// ExecutionSummary accumulates results across all fix-loop iterations.
type ExecutionSummary struct {
	TotalAttempts  int
	SuccessfulRuns int
	FixesApplied   int
	AllResults     []*ExecutionResult
	LastResult     *ExecutionResult
	FinalSandboxID string
	ErrorsSummary  []string
	FilesUpdated   []string
	TotalDuration  time.Duration
}

// SandboxStatus holds the runtime state returned by the bridge.
type SandboxStatus struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CleanupResult is the bridge's response to a sandbox deletion request.
type CleanupResult struct {
	SandboxID string `json:"sandbox_id"`
	Deleted   bool   `json:"deleted"`
}

// UpdateFilesResult is the bridge's response to a file-update request.
type UpdateFilesResult struct {
	SandboxID    string   `json:"sandbox_id"`
	UpdatedFiles []string `json:"updated_files"`
}

// wire types for HTTP requests and error responses.
type testExecutionRequest struct {
	Path        string `json:"path"`
	IterationID string `json:"iteration_id,omitempty"`
	KeepSandbox bool   `json:"keep_sandbox"`
}

type updateFilesRequest struct {
	SandboxID string            `json:"sandbox_id"`
	Files     map[string]string `json:"files"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// ---- Client -----------------------------------------------------------------

// ClientConfig holds the parameters required to construct a Client.
type ClientConfig struct {
	// BaseURL is the URL of the E2B bridge service (e.g. "http://localhost:8080").
	BaseURL string

	// APIKey is the E2B API key. If empty the E2B_API_KEY environment variable
	// is used.
	APIKey string

	// TenantID tags every request with a tenant identifier for isolation.
	TenantID string

	// Execution overrides; nil means DefaultExecutionConfig() is used.
	Execution *ExecutionConfig

	// Logger is used for structured output. If nil slog.Default() is used.
	Logger *slog.Logger
}

// Client is the primary E2B bridge client. It is safe for concurrent use.
//
// Create with NewClient or NewClientWithConfig; never embed or copy by value.
type Client struct {
	baseURL         string
	apiKey          string
	tenantID        string
	httpClient      *http.Client
	config          *ExecutionConfig
	retryStrategies map[ErrorType]*RetryStrategy
	logger          *slog.Logger
}

// NewClient constructs a Client using defaults and the E2B_API_KEY environment
// variable.
func NewClient(ctx context.Context, baseURL string) (*Client, error) {
	return NewClientWithConfig(ctx, ClientConfig{BaseURL: baseURL})
}

// NewClientWithConfig constructs a Client with explicit configuration. The
// context is reserved for future use (e.g. acquiring initial auth tokens).
func NewClientWithConfig(_ context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("e2b client: BaseURL must not be empty")
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("E2B_API_KEY")
	}

	exec := cfg.Execution
	if exec == nil {
		exec = DefaultExecutionConfig()
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	c := &Client{
		baseURL:  cfg.BaseURL,
		apiKey:   apiKey,
		tenantID: cfg.TenantID,
		httpClient: &http.Client{
			Timeout: exec.Timeout,
		},
		config:          exec,
		retryStrategies: DefaultRetryStrategies(),
		logger:          logger,
	}
	return c, nil
}

// SetRetryStrategy overrides the retry strategy for a specific error type.
func (c *Client) SetRetryStrategy(errorType ErrorType, strategy *RetryStrategy) {
	if c.retryStrategies == nil {
		c.retryStrategies = make(map[ErrorType]*RetryStrategy)
	}
	c.retryStrategies[errorType] = strategy
}

// GetConfig returns a copy of the current execution configuration.
func (c *Client) GetConfig() ExecutionConfig {
	return *c.config
}
