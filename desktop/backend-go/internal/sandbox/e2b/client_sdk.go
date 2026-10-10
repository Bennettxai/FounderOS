package e2b

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ---- SDK-style direct execution (no bridge) ---------------------------------

// SDKConfig holds the options for the SDK-backed executor.
type SDKConfig struct {
	// APIKey is the E2B API key. If empty E2B_API_KEY is used.
	APIKey string

	// TenantID tags sandbox operations for isolation tracking.
	TenantID string

	// Execution overrides; nil means DefaultExecutionConfig() is used.
	Execution *ExecutionConfig

	// Logger is used for structured output. If nil slog.Default() is used.
	Logger *slog.Logger
}

// SDKExecutor runs code directly in E2B sandboxes without an HTTP bridge. It
// uses the local filesystem as the source and uploads files itself.
//
// The interface is intentionally kept minimal; complex orchestration belongs in
// the service layer.
type SDKExecutor struct {
	apiKey   string
	tenantID string
	config   *ExecutionConfig
	logger   *slog.Logger
}

// NewSDKExecutor constructs an SDKExecutor. apiKey may be empty; if so
// E2B_API_KEY must be set in the environment.
func NewSDKExecutor(ctx context.Context, cfg SDKConfig) (*SDKExecutor, error) {
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

	if apiKey == "" {
		logger.WarnContext(ctx, "E2B_API_KEY not set; sandbox creation will fail")
	}

	return &SDKExecutor{
		apiKey:   apiKey,
		tenantID: cfg.TenantID,
		config:   exec,
		logger:   logger,
	}, nil
}

// UploadFiles reads all non-ignored files under projectPath and returns a map
// of sandbox-relative paths to their contents. The caller is responsible for
// writing the returned map into a sandbox.
//
// Ignored paths: directories, hidden files, node_modules.
func UploadFiles(projectPath string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if strings.Contains(path, "node_modules") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		rel, err := filepath.Rel(projectPath, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}

		files[filepath.Join("/home/user/project", rel)] = string(content)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("upload files walk: %w", err)
	}
	return files, nil
}

// DetectPackageManager returns the install and test commands appropriate for
// the project at projectPath, based on the presence of well-known manifest
// files.
func DetectPackageManager(projectPath string) (installCmd, testCmd string, ok bool) {
	checks := []struct {
		file    string
		install string
		test    string
	}{
		{"package.json", "npm install", "npm test"},
		{"go.mod", "go mod download", "go test ./..."},
		{"requirements.txt", "pip install -r requirements.txt", "pytest"},
	}

	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(projectPath, c.file)); err == nil {
			return c.install, c.test, true
		}
	}
	return "", "", false
}

// ParseTestSuccess uses heuristics to decide whether command output indicates
// a successful test run.
func ParseTestSuccess(output string) bool {
	lower := strings.ToLower(output)

	failureKeywords := []string{"fail", "error", "✗", "✘", "failed", "failure"}
	for _, kw := range failureKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	successKeywords := []string{"all tests passed", "tests passed", "ok", "pass", "✓", "✔"}
	for _, kw := range successKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}
