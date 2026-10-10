package e2b_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/sandbox/e2b"
)

// ---- test helpers -----------------------------------------------------------

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func newClient(t *testing.T, baseURL string) *e2b.Client {
	t.Helper()
	c, err := e2b.NewClient(context.Background(), baseURL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func newClientWithConfig(t *testing.T, cfg e2b.ClientConfig) *e2b.Client {
	t.Helper()
	c, err := e2b.NewClientWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClientWithConfig: %v", err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ---- Error tests ------------------------------------------------------------

func TestE2BError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *e2b.E2BError
		wantSub string
	}{
		{
			name: "with phase",
			err: &e2b.E2BError{
				Type:    e2b.ErrorTypeExecution,
				Phase:   e2b.PhaseInstall,
				Message: "npm install failed",
			},
			wantSub: "install",
		},
		{
			name: "without phase",
			err: &e2b.E2BError{
				Type:    e2b.ErrorTypeNetwork,
				Message: "connection refused",
			},
			wantSub: "network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("Error() = %q; want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestE2BError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := e2b.NewE2BError(e2b.ErrorTypeService, "X", "msg", false).WithCause(cause)

	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v; want %v", err.Unwrap(), cause)
	}
}

func TestE2BError_AddDetail(t *testing.T) {
	err := e2b.NewE2BError(e2b.ErrorTypeValidation, "V", "msg", false)
	err.AddDetail("field", "email")

	if err.Details["field"] != "email" {
		t.Errorf("expected detail field=email, got %v", err.Details)
	}
}

func TestClassifyError_NetworkError(t *testing.T) {
	netErr := fmt.Errorf("connection refused")
	classified := e2b.ClassifyError(netErr, e2b.PhaseSetup, "sb-123")

	if classified.Type != e2b.ErrorTypeNetwork {
		t.Errorf("Type = %q; want %q", classified.Type, e2b.ErrorTypeNetwork)
	}
	if !classified.Retryable {
		t.Error("expected Retryable = true for network error")
	}
}

func TestClassifyError_AlreadyE2BError(t *testing.T) {
	original := e2b.NewE2BError(e2b.ErrorTypeRateLimit, "R", "too many", true)
	result := e2b.ClassifyError(original, e2b.PhaseBuild, "")

	if result != original {
		t.Error("ClassifyError should return existing *E2BError unchanged")
	}
}

func TestClassifyError_TimeoutKeyword(t *testing.T) {
	err := fmt.Errorf("context deadline exceeded")
	classified := e2b.ClassifyError(err, e2b.PhaseSetup, "")

	if classified.Type != e2b.ErrorTypeTimeout {
		t.Errorf("Type = %q; want %q", classified.Type, e2b.ErrorTypeTimeout)
	}
}

func TestShouldRetry(t *testing.T) {
	strategies := e2b.DefaultRetryStrategies()

	tests := []struct {
		name      string
		err       error
		attempt   int
		wantRetry bool
	}{
		{
			name:      "nil error never retries",
			err:       nil,
			attempt:   1,
			wantRetry: false,
		},
		{
			name:      "non-retryable error never retries",
			err:       e2b.NewE2BError(e2b.ErrorTypeValidation, "V", "bad input", false),
			attempt:   1,
			wantRetry: false,
		},
		{
			name:      "retryable network error on attempt 1",
			err:       e2b.NewE2BError(e2b.ErrorTypeNetwork, "NET", "refused", true),
			attempt:   1,
			wantRetry: true,
		},
		{
			name:      "network error at max attempts does not retry",
			err:       e2b.NewE2BError(e2b.ErrorTypeNetwork, "NET", "refused", true),
			attempt:   strategies[e2b.ErrorTypeNetwork].MaxAttempts,
			wantRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, delay := e2b.ShouldRetry(tt.err, tt.attempt, strategies)
			if ok != tt.wantRetry {
				t.Errorf("ShouldRetry() retry = %v; want %v", ok, tt.wantRetry)
			}
			if !ok && delay != 0 {
				t.Errorf("expected zero delay when not retrying, got %v", delay)
			}
		})
	}
}

// ---- Client construction tests ----------------------------------------------

func TestNewClient_EmptyBaseURL(t *testing.T) {
	_, err := e2b.NewClient(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty BaseURL")
	}
}

func TestNewClientWithConfig_SetsDefaults(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:  srv.URL,
		TenantID: "tenant-abc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := c.GetConfig()
	if cfg.MaxRetries == 0 {
		t.Error("expected default MaxRetries to be set")
	}
	_ = c
}

// ---- HTTP endpoint tests ----------------------------------------------------

func TestClient_TestExecution_Success(t *testing.T) {
	want := e2b.ExecutionResult{
		SandboxID: "sb-001",
		Success:   true,
		Phase:     e2b.PhaseStart,
	}

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-execution" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, want)
	})

	c := newClient(t, srv.URL)
	got, err := c.TestExecution(context.Background(), "/some/path")

	if err != nil {
		t.Fatalf("TestExecution: %v", err)
	}
	if !got.IsSuccess() {
		t.Errorf("expected IsSuccess=true, got false")
	}
	if got.SandboxID != want.SandboxID {
		t.Errorf("SandboxID = %q; want %q", got.SandboxID, want.SandboxID)
	}
}

func TestClient_TestExecution_ServerError_IsRetryable(t *testing.T) {
	calls := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	})

	cfg := e2b.DefaultExecutionConfig()
	cfg.MaxRetries = 1

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:   srv.URL,
		Execution: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Override service retry strategy to avoid real sleep delays in tests.
	c.SetRetryStrategy(e2b.ErrorTypeService, &e2b.RetryStrategy{
		MaxAttempts:   2,
		BaseDelay:     time.Millisecond,
		MaxDelay:      2 * time.Millisecond,
		BackoffFactor: 1.0,
	})

	_, err = c.TestExecution(context.Background(), "/path")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestClient_GetSandboxStatus(t *testing.T) {
	want := e2b.SandboxStatus{SandboxID: "sb-42", Status: "running"}

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/sandbox-status/") {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, want)
	})

	c := newClient(t, srv.URL)
	got, err := c.GetSandboxStatus(context.Background(), "sb-42")
	if err != nil {
		t.Fatalf("GetSandboxStatus: %v", err)
	}
	if got.Status != want.Status {
		t.Errorf("Status = %q; want %q", got.Status, want.Status)
	}
}

func TestClient_GetSandboxStatus_EmptyID(t *testing.T) {
	c := newClient(t, "http://localhost:1") // unreachable, should fail before HTTP call
	_, err := c.GetSandboxStatus(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty sandbox ID")
	}
}

func TestClient_CleanupSandbox(t *testing.T) {
	want := e2b.CleanupResult{SandboxID: "sb-99", Deleted: true}

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, want)
	})

	c := newClient(t, srv.URL)
	got, err := c.CleanupSandbox(context.Background(), "sb-99")
	if err != nil {
		t.Fatalf("CleanupSandbox: %v", err)
	}
	if !got.Deleted {
		t.Error("expected Deleted=true")
	}
}

func TestClient_UpdateSandboxFiles_EmptySandboxID(t *testing.T) {
	c := newClient(t, "http://localhost:1")
	_, err := c.UpdateSandboxFiles(context.Background(), "", map[string]string{"a": "b"})
	if err == nil {
		t.Error("expected error for empty sandbox ID")
	}
}

func TestClient_UpdateSandboxFiles_EmptyFiles(t *testing.T) {
	c := newClient(t, "http://localhost:1")
	_, err := c.UpdateSandboxFiles(context.Background(), "sb-1", nil)
	if err == nil {
		t.Error("expected error for nil files map")
	}
}

func TestClient_UpdateSandboxFiles_Success(t *testing.T) {
	want := e2b.UpdateFilesResult{
		SandboxID:    "sb-10",
		UpdatedFiles: []string{"main.go", "go.mod"},
	}

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, want)
	})

	c := newClient(t, srv.URL)
	got, err := c.UpdateSandboxFiles(context.Background(), "sb-10", map[string]string{"main.go": "package main"})
	if err != nil {
		t.Fatalf("UpdateSandboxFiles: %v", err)
	}
	if len(got.UpdatedFiles) != 2 {
		t.Errorf("UpdatedFiles count = %d; want 2", len(got.UpdatedFiles))
	}
}

// ---- Health check tests -----------------------------------------------------

func TestClient_CheckHealth_OK(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	c := newClient(t, srv.URL)
	if err := c.CheckHealth(context.Background()); err != nil {
		t.Errorf("CheckHealth: %v", err)
	}
}

func TestClient_CheckHealth_Unhealthy(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := newClient(t, srv.URL)
	if err := c.CheckHealth(context.Background()); err == nil {
		t.Error("expected error for 503 response")
	}
}

func TestClient_CheckHealthWithRetry_EventualSuccess(t *testing.T) {
	calls := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL: srv.URL,
		Execution: &e2b.ExecutionConfig{
			Timeout:    5 * time.Second,
			MaxRetries: 3,
			RetryDelay: time.Millisecond,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// maxRetries=2: fails once (backoff=1s), succeeds on call 2.
	if err := c.CheckHealthWithRetry(context.Background(), 2); err != nil {
		t.Errorf("CheckHealthWithRetry: %v", err)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 calls, got %d", calls)
	}
}

// ---- ExecuteWithFixLoop tests -----------------------------------------------

func TestClient_ExecuteWithFixLoop_SuccessOnFirst(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, e2b.ExecutionResult{
			SandboxID: "sb-loop",
			Success:   true,
			Phase:     e2b.PhaseTest,
		})
	})

	c := newClient(t, srv.URL)
	summary, err := c.ExecuteWithFixLoop(context.Background(), "/project", nil)

	if err != nil {
		t.Fatalf("ExecuteWithFixLoop: %v", err)
	}
	if summary.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d; want 1", summary.SuccessfulRuns)
	}
	if summary.TotalAttempts != 1 {
		t.Errorf("TotalAttempts = %d; want 1", summary.TotalAttempts)
	}
}

func TestClient_ExecuteWithFixLoop_FixApplied(t *testing.T) {
	attempt := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test-execution":
			attempt++
			if attempt == 1 {
				writeJSON(w, http.StatusOK, e2b.ExecutionResult{
					SandboxID: "sb-fx",
					Success:   false,
					Phase:     e2b.PhaseBuild,
					Error:     "compilation error",
				})
			} else {
				writeJSON(w, http.StatusOK, e2b.ExecutionResult{
					SandboxID: "sb-fx",
					Success:   true,
					Phase:     e2b.PhaseTest,
				})
			}
		case "/update-sandbox-files":
			writeJSON(w, http.StatusOK, e2b.UpdateFilesResult{
				SandboxID:    "sb-fx",
				UpdatedFiles: []string{"main.go"},
			})
		default:
			http.NotFound(w, r)
		}
	})

	fixerCalled := false
	fixer := e2b.FixerFunc(func(ctx context.Context, result *e2b.ExecutionResult) (map[string]string, error) {
		fixerCalled = true
		return map[string]string{"main.go": "package main\n"}, nil
	})

	cfg := e2b.DefaultExecutionConfig()
	cfg.RetryDelay = time.Millisecond

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:   srv.URL,
		Execution: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := c.ExecuteWithFixLoop(context.Background(), "/project", fixer)
	if err != nil {
		t.Fatalf("ExecuteWithFixLoop: %v", err)
	}
	if !fixerCalled {
		t.Error("expected fixer to be called")
	}
	if summary.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d; want 1", summary.SuccessfulRuns)
	}
	if summary.FixesApplied != 1 {
		t.Errorf("FixesApplied = %d; want 1", summary.FixesApplied)
	}
}

func TestClient_ExecuteWithFixLoop_AllAttemptsExhausted(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, e2b.ExecutionResult{
			SandboxID: "sb-fail",
			Success:   false,
			Phase:     e2b.PhaseBuild,
			Error:     "always broken",
		})
	})

	cfg := e2b.DefaultExecutionConfig()
	cfg.MaxRetries = 2
	cfg.RetryDelay = time.Millisecond

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:   srv.URL,
		Execution: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Without a fixer the loop breaks at the first failure (no retries to apply).
	// The summary should reflect exactly 1 attempt and an error returned.
	summary, err := c.ExecuteWithFixLoop(context.Background(), "/project", nil)
	if err == nil {
		t.Error("expected error when all attempts fail")
	}
	// With no fixer, the loop stops at the first non-success result.
	if summary.TotalAttempts != 1 {
		t.Errorf("TotalAttempts = %d; want 1 (no fixer, breaks immediately)", summary.TotalAttempts)
	}
}

// ---- ExecutionResult helper method tests ------------------------------------

func TestExecutionResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name string
		r    *e2b.ExecutionResult
		want bool
	}{
		{"nil result", nil, false},
		{"success true, no error", &e2b.ExecutionResult{Success: true}, true},
		{"success true, with error", &e2b.ExecutionResult{Success: true, Error: "oops"}, false},
		{"success false", &e2b.ExecutionResult{Success: false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionResult_GetErrorDetails(t *testing.T) {
	r := &e2b.ExecutionResult{
		Phase:  e2b.PhaseBuild,
		Error:  "build failed",
		Stderr: "undefined: Foo",
	}
	details := r.GetErrorDetails()
	if !strings.Contains(details, "build") {
		t.Errorf("expected details to contain 'build', got: %q", details)
	}
	if !strings.Contains(details, "Foo") {
		t.Errorf("expected details to contain 'Foo', got: %q", details)
	}
}

// ---- Workspace / secret redaction tests ------------------------------------

func TestRedactSecrets_CommonKeys(t *testing.T) {
	input := map[string]string{
		"API_KEY":      "abc123",
		"NORMAL_VAR":   "hello",
		"DB_PASSWORD":  "secret",
		"PUBLIC_VALUE": "visible",
	}

	redacted := e2b.RedactSecrets(input)

	if redacted["API_KEY"] != e2b.RedactedPlaceholder {
		t.Errorf("API_KEY should be redacted, got %q", redacted["API_KEY"])
	}
	if redacted["DB_PASSWORD"] != e2b.RedactedPlaceholder {
		t.Errorf("DB_PASSWORD should be redacted, got %q", redacted["DB_PASSWORD"])
	}
	if redacted["NORMAL_VAR"] == e2b.RedactedPlaceholder {
		t.Error("NORMAL_VAR should not be redacted")
	}
	if redacted["PUBLIC_VALUE"] == e2b.RedactedPlaceholder {
		t.Error("PUBLIC_VALUE should not be redacted")
	}
}

func TestRedactSecrets_NilMap(t *testing.T) {
	result := e2b.RedactSecrets(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestRedactURLCredentials(t *testing.T) {
	tests := []struct {
		input   string
		wantSub string
		wantNot string
	}{
		{
			input:   "postgresql://user:secret@localhost:5432/db",
			wantSub: "***:***",
			wantNot: "secret",
		},
		{
			input:   "https://admin:p4ssw0rd@example.com/api",
			wantSub: "***:***",
			wantNot: "p4ssw0rd",
		},
		{
			input:   "no-credentials-here",
			wantSub: "no-credentials-here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := e2b.RedactURLCredentials(tt.input)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("RedactURLCredentials(%q) = %q; want substring %q", tt.input, got, tt.wantSub)
			}
			if tt.wantNot != "" && strings.Contains(got, tt.wantNot) {
				t.Errorf("RedactURLCredentials(%q) = %q; should not contain %q", tt.input, got, tt.wantNot)
			}
		})
	}
}

func TestGetRedactionStats(t *testing.T) {
	original := map[string]string{
		"API_KEY": "secret",
		"PUBLIC":  "visible",
	}
	redacted := e2b.RedactSecrets(original)
	stats := e2b.GetRedactionStats(original, redacted)

	if stats.TotalKeys != 2 {
		t.Errorf("TotalKeys = %d; want 2", stats.TotalKeys)
	}
	if stats.RedactedKeys != 1 {
		t.Errorf("RedactedKeys = %d; want 1", stats.RedactedKeys)
	}
	if stats.PreservedKeys != 1 {
		t.Errorf("PreservedKeys = %d; want 1", stats.PreservedKeys)
	}
}

// ---- Workspace cleanup tests ------------------------------------------------

func TestCleanWorkspace_RemovesEnvFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := []string{".env", ".env.local", "main.go", "README.md"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &e2b.CleanupConfig{RemoveEnvFiles: true}
	result, err := e2b.CleanWorkspace(dir, cfg, nil)
	if err != nil {
		t.Fatalf("CleanWorkspace: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".env")); !os.IsNotExist(err) {
		t.Error(".env should have been removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); os.IsNotExist(err) {
		t.Error("main.go should not have been removed")
	}

	removedSet := make(map[string]bool)
	for _, p := range result.RemovedPaths {
		removedSet[p] = true
	}
	if !removedSet[".env"] {
		t.Error("expected .env in RemovedPaths")
	}
}

func TestCleanWorkspace_RemovesNodeModules(t *testing.T) {
	dir := t.TempDir()

	// Create a node_modules directory with a file inside
	nmDir := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("module"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &e2b.CleanupConfig{RemoveNodeModules: true}
	_, err := e2b.CleanWorkspace(dir, cfg, nil)
	if err != nil {
		t.Fatalf("CleanWorkspace: %v", err)
	}

	if _, err := os.Stat(nmDir); !os.IsNotExist(err) {
		t.Error("node_modules should have been removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should not have been removed")
	}
}

func TestCreateCleanCopy_DoesNotModifyOriginal(t *testing.T) {
	src := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, ".env"), []byte("SECRET=x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "app.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &e2b.CleanupConfig{RemoveEnvFiles: true}
	tmpDir, _, err := e2b.CreateCleanCopy(src, cfg, nil)
	if err != nil {
		t.Fatalf("CreateCleanCopy: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Original must still have .env
	if _, err := os.Stat(filepath.Join(src, ".env")); os.IsNotExist(err) {
		t.Error(".env should still exist in original source")
	}
	// Copy must have .env removed
	if _, err := os.Stat(filepath.Join(tmpDir, ".env")); !os.IsNotExist(err) {
		t.Error(".env should be absent from the clean copy")
	}
	// Copy must have app.go
	if _, err := os.Stat(filepath.Join(tmpDir, "app.go")); os.IsNotExist(err) {
		t.Error("app.go should be present in the clean copy")
	}
}

// ---- Event tests ------------------------------------------------------------

func TestEmitFunctions_NilBroadcaster(t *testing.T) {
	// All Emit* functions should be no-ops when broadcaster is nil; they must
	// not panic.
	e2b.EmitStarting(nil, "wf-1", "t-1", 1)
	e2b.EmitUpload(nil, "wf-1", "t-1", 1)
	e2b.EmitInstall(nil, "wf-1", "sb-1", "t-1", 1)
	e2b.EmitBuild(nil, "wf-1", "sb-1", "t-1", 1)
	e2b.EmitTest(nil, "wf-1", "sb-1", "t-1", 1)
	e2b.EmitComplete(nil, "wf-1", "sb-1", "t-1", 1)
	e2b.EmitFailed(nil, "wf-1", "build", "msg", "sb-1", "t-1", 1)
}

func TestEmitFunctions_EmptyWorkflowID(t *testing.T) {
	called := false
	b := e2b.Broadcaster(func(wid, msg string) { called = true })

	e2b.EmitStarting(b, "", "t-1", 1)
	if called {
		t.Error("broadcaster should not be called for empty workflowID")
	}
}

func TestEmitComplete_ValidJSON(t *testing.T) {
	var captured string
	b := e2b.Broadcaster(func(_, msg string) { captured = msg })

	e2b.EmitComplete(b, "wf-test", "sb-complete", "t-1", 2)

	var ev map[string]interface{}
	if err := json.Unmarshal([]byte(captured), &ev); err != nil {
		t.Fatalf("captured event is not valid JSON: %v\nmessage: %s", err, captured)
	}
	if ev["phase"] != "complete" {
		t.Errorf("phase = %v; want complete", ev["phase"])
	}
	if ev["progress"].(float64) != 100 {
		t.Errorf("progress = %v; want 100", ev["progress"])
	}
}

func TestEmitFailed_PhaseProgress(t *testing.T) {
	tests := []struct {
		phase    string
		wantProg float64
	}{
		{"build", 70},
		{"install", 40},
		{"upload", 20},
		{"unknown-phase", 50},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			var captured string
			b := e2b.Broadcaster(func(_, msg string) { captured = msg })

			e2b.EmitFailed(b, "wf-1", tt.phase, "error details", "sb-1", "t-1", 1)

			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(captured), &ev); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if got := ev["progress"].(float64); got != tt.wantProg {
				t.Errorf("progress = %.0f; want %.0f (phase=%s)", got, tt.wantProg, tt.phase)
			}
		})
	}
}

// ---- Package-level utility tests -------------------------------------------

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   string
		wantInstall string
		wantTest    string
		wantOK      bool
	}{
		{"node", "package.json", "npm install", "npm test", true},
		{"go", "go.mod", "go mod download", "go test ./...", true},
		{"python", "requirements.txt", "pip install -r requirements.txt", "pytest", true},
		{"unknown", "Makefile", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.setupFile), []byte(""), 0644); err != nil {
				t.Fatal(err)
			}

			install, test, ok := e2b.DetectPackageManager(dir)
			if ok != tt.wantOK {
				t.Errorf("ok = %v; want %v", ok, tt.wantOK)
			}
			if install != tt.wantInstall {
				t.Errorf("install = %q; want %q", install, tt.wantInstall)
			}
			if test != tt.wantTest {
				t.Errorf("test = %q; want %q", test, tt.wantTest)
			}
		})
	}
}

func TestParseTestSuccess(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"go ok output", "ok  github.com/foo/bar  0.123s", true},
		{"fail keyword", "FAIL: TestFoo\n1 test failed", false},
		{"error keyword", "error: undefined variable", false},
		{"empty output", "", false},
		{"passed keyword", "2 tests passed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e2b.ParseTestSuccess(tt.output); got != tt.want {
				t.Errorf("ParseTestSuccess(%q) = %v; want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestUploadFiles_SkipsHiddenAndNodeModules(t *testing.T) {
	dir := t.TempDir()

	paths := []struct {
		rel string
		ok  bool // should it appear in the output?
	}{
		{"main.go", true},
		{".hidden", false},
		{filepath.Join("node_modules", "pkg", "index.js"), false},
	}

	for _, p := range paths {
		full := filepath.Join(dir, p.rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := e2b.UploadFiles(dir)
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}

	// main.go must appear under /home/user/project/main.go
	if _, ok := files["/home/user/project/main.go"]; !ok {
		t.Errorf("expected /home/user/project/main.go in result, keys: %v", mapKeys(files))
	}

	// Hidden and node_modules must be absent
	for k := range files {
		if strings.Contains(k, ".hidden") {
			t.Errorf("hidden file should not be included: %q", k)
		}
		if strings.Contains(k, "node_modules") {
			t.Errorf("node_modules should not be included: %q", k)
		}
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ---- Context cancellation test ----------------------------------------------

func TestClient_ContextCancellation(t *testing.T) {
	// Server that responds immediately with 500 — the client cancels before
	// any retry sleep completes. We use a pre-cancelled context to ensure
	// the test is instantaneous.
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "boom"})
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL: srv.URL,
		Execution: &e2b.ExecutionConfig{
			Timeout:    5 * time.Second,
			MaxRetries: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.TestExecution(ctx, "/project")
	if err == nil {
		t.Error("expected error when context is already cancelled")
	}
}

// ---- Tenant isolation header test ------------------------------------------

func TestClient_TenantIDHeader(t *testing.T) {
	var receivedTenant string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedTenant = r.Header.Get("X-Tenant-ID")
		writeJSON(w, http.StatusOK, e2b.ExecutionResult{Success: true})
	})

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:  srv.URL,
		TenantID: "tenant-xyz",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _ = c.TestExecution(context.Background(), "/project")

	if receivedTenant != "tenant-xyz" {
		t.Errorf("X-Tenant-ID = %q; want %q", receivedTenant, "tenant-xyz")
	}
}

// ---- Retry strategy override test -------------------------------------------

func TestClient_SetRetryStrategy(t *testing.T) {
	calls := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
	})

	cfg := e2b.DefaultExecutionConfig()
	cfg.MaxRetries = 1

	c, err := e2b.NewClientWithConfig(context.Background(), e2b.ClientConfig{
		BaseURL:   srv.URL,
		Execution: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Set an aggressive rate-limit strategy so the test completes quickly.
	c.SetRetryStrategy(e2b.ErrorTypeRateLimit, &e2b.RetryStrategy{
		MaxAttempts:   2,
		BaseDelay:     time.Millisecond,
		MaxDelay:      10 * time.Millisecond,
		BackoffFactor: 1.0,
	})

	_, err = c.TestExecution(context.Background(), "/project")
	if err == nil {
		t.Error("expected error for rate-limited response")
	}
}

// ---- DefaultRetryStrategies coverage ----------------------------------------

func TestDefaultRetryStrategies_AllTypesPresent(t *testing.T) {
	strategies := e2b.DefaultRetryStrategies()
	required := []e2b.ErrorType{
		e2b.ErrorTypeNetwork,
		e2b.ErrorTypeTimeout,
		e2b.ErrorTypeExecution,
		e2b.ErrorTypeRateLimit,
		e2b.ErrorTypeService,
	}
	for _, et := range required {
		if _, ok := strategies[et]; !ok {
			t.Errorf("missing strategy for error type %q", et)
		}
	}
}

// ---- ExecutionSummary duration test -----------------------------------------

func TestClient_ExecuteWithFixLoop_RecordsDuration(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, e2b.ExecutionResult{Success: true})
	})

	c := newClient(t, srv.URL)
	summary, err := c.ExecuteWithFixLoop(context.Background(), "/project", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalDuration <= 0 {
		t.Error("expected non-zero TotalDuration")
	}
}

// ---- NewSDKExecutor test ----------------------------------------------------

func TestNewSDKExecutor_NoAPIKey(t *testing.T) {
	// Unset the env var so we can test the warning path without crashing.
	t.Setenv("E2B_API_KEY", "")

	ex, err := e2b.NewSDKExecutor(context.Background(), e2b.SDKConfig{})
	if err != nil {
		t.Fatalf("NewSDKExecutor: %v", err)
	}
	if ex == nil {
		t.Error("expected non-nil executor")
	}
}

func TestNewSDKExecutor_WithAPIKey(t *testing.T) {
	ex, err := e2b.NewSDKExecutor(context.Background(), e2b.SDKConfig{
		APIKey:   "test-key",
		TenantID: "t-123",
	})
	if err != nil {
		t.Fatalf("NewSDKExecutor: %v", err)
	}
	if ex == nil {
		t.Error("expected non-nil executor")
	}
}

// ---- 400-level HTTP error is non-retryable test ----------------------------

func TestClient_400_IsNonRetryable(t *testing.T) {
	calls := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad input"})
	})

	c := newClient(t, srv.URL)
	_, err := c.TestExecution(context.Background(), "/project")
	if err == nil {
		t.Error("expected error for 400 response")
	}
	// Should have called exactly once — no transport-level retry for client errors.
	if calls > 2 {
		t.Errorf("expected at most 2 HTTP calls for client error, got %d", calls)
	}
}
