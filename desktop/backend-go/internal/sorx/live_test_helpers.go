package sorx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// LiveTestResult captures the result of a live action test
type LiveTestResult struct {
	Action       string                 `json:"action"`
	Category     string                 `json:"category"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	CleanedUp    bool                   `json:"cleaned_up"`
	CleanupError string                 `json:"cleanup_error,omitempty"`
}

// LiveTestReport contains results from all tests
type LiveTestReport struct {
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	TotalDuration   time.Duration     `json:"total_duration"`
	TotalTests      int               `json:"total_tests"`
	PassedTests     int               `json:"passed_tests"`
	FailedTests     int               `json:"failed_tests"`
	SkippedTests    int               `json:"skipped_tests"`
	Results         []LiveTestResult  `json:"results"`
	Configuration   map[string]string `json:"configuration"`
}

// LiveTestRunner manages live API testing
type LiveTestRunner struct {
	config  *LiveTestConfig
	report  *LiveTestReport
	verbose bool
	logger  *slog.Logger
}

// NewLiveTestRunner creates a new test runner
func NewLiveTestRunner(config *LiveTestConfig) *LiveTestRunner {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &LiveTestRunner{
		config: config,
		report: &LiveTestReport{
			StartTime: time.Now(),
			Results:   make([]LiveTestResult, 0),
			Configuration: map[string]string{
				"categories":    fmt.Sprintf("%v", config.TestCategories),
				"cleanup_mode":  config.TestCleanupMode,
				"verbose":       fmt.Sprintf("%v", config.TestVerbose),
			},
		},
		verbose: config.TestVerbose,
		logger:  logger,
	}
}

// RunActionTest runs a single action test
func (r *LiveTestRunner) RunActionTest(
	t *testing.T,
	actionName string,
	category string,
	setup func() (ActionContext, error),
	validate func(result interface{}) error,
	cleanup func(ctx ActionContext, result interface{}) error,
) {
	result := LiveTestResult{
		Action:    actionName,
		Category:  category,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		r.report.Results = append(r.report.Results, result)
		r.report.TotalTests++
		if result.Success {
			r.report.PassedTests++
		} else {
			r.report.FailedTests++
		}
	}()

	// Check if category should be tested
	if !r.config.ShouldTestCategory(category) {
		if r.verbose {
			r.logger.Info("Skipping test", "action", actionName, "reason", "category not selected")
		}
		r.report.SkippedTests++
		t.Skip(fmt.Sprintf("Category %s not selected for testing", category))
		return
	}

	// Validate credentials for category
	if err := r.config.ValidateForCategory(category); err != nil {
		if r.verbose {
			r.logger.Warn("Skipping test", "action", actionName, "reason", err.Error())
		}
		r.report.SkippedTests++
		t.Skip(fmt.Sprintf("Missing credentials: %v", err))
		return
	}

	// Run setup
	if r.verbose {
		r.logger.Info("Running test", "action", actionName, "category", category)
	}

	actionCtx, err := setup()
	if err != nil {
		result.Error = fmt.Sprintf("setup failed: %v", err)
		if r.verbose {
			r.logger.Error("Setup failed", "action", actionName, "error", err)
		}
		t.Errorf("%s setup failed: %v", actionName, err)
		return
	}

	// Execute action
	handler, exists := GetActionHandler(actionName)
	if !exists {
		result.Error = "action not registered"
		t.Errorf("Action %s not registered", actionName)
		return
	}

	actionResult, err := handler(context.Background(), actionCtx)
	if err != nil {
		result.Error = fmt.Sprintf("execution failed: %v", err)
		if r.verbose {
			r.logger.Error("Execution failed", "action", actionName, "error", err)
		}
		t.Errorf("%s execution failed: %v", actionName, err)
		return
	}

	// Store output
	if resultMap, ok := actionResult.(map[string]interface{}); ok {
		result.Output = resultMap
	} else {
		result.Output = map[string]interface{}{"result": actionResult}
	}

	// Validate result
	if validate != nil {
		if err := validate(actionResult); err != nil {
			result.Error = fmt.Sprintf("validation failed: %v", err)
			if r.verbose {
				r.logger.Error("Validation failed", "action", actionName, "error", err)
			}
			t.Errorf("%s validation failed: %v", actionName, err)
			return
		}
	}

	// Success!
	result.Success = true
	if r.verbose {
		r.logger.Info("Test passed", "action", actionName, "duration", result.Duration)
	}

	// Cleanup (if configured)
	shouldCleanup := r.config.TestCleanupMode == "always" ||
		(r.config.TestCleanupMode == "on_success" && result.Success)

	if shouldCleanup && cleanup != nil {
		if r.verbose {
			r.logger.Info("Running cleanup", "action", actionName)
		}
		if err := cleanup(actionCtx, actionResult); err != nil {
			result.CleanupError = err.Error()
			if r.verbose {
				r.logger.Warn("Cleanup failed", "action", actionName, "error", err)
			}
		} else {
			result.CleanedUp = true
		}
	}
}

// Finalize completes the test run and saves the report
func (r *LiveTestRunner) Finalize() error {
	r.report.EndTime = time.Now()
	r.report.TotalDuration = r.report.EndTime.Sub(r.report.StartTime)

	// Log summary
	r.logger.Info("Test run complete",
		"total", r.report.TotalTests,
		"passed", r.report.PassedTests,
		"failed", r.report.FailedTests,
		"skipped", r.report.SkippedTests,
		"duration", r.report.TotalDuration,
	)

	// Save JSON report
	reportFile := "live_test_report.json"
	data, err := json.MarshalIndent(r.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(reportFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	r.logger.Info("Report saved", "file", reportFile)
	return nil
}

// GetReport returns the current test report
func (r *LiveTestRunner) GetReport() *LiveTestReport {
	return r.report
}

// CreateTestActionContext creates a minimal ActionContext for testing
func CreateTestActionContext(testID string, credentials map[string]string, params map[string]interface{}) ActionContext {
	// Merge credentials into params (integration actions expect credentials in params)
	mergedParams := make(map[string]interface{})
	for k, v := range params {
		mergedParams[k] = v
	}
	if credentials != nil {
		mergedParams["credentials"] = credentials
	}

	return ActionContext{
		Execution: &Execution{
			UserID: "test-user",
			Params: mergedParams,
		},
		Params: mergedParams,
	}
}

// CreateTestCredentials creates test credentials map
func CreateTestCredentials(config *LiveTestConfig, provider string) map[string]string {
	creds := make(map[string]string)

	switch provider {
	case "google":
		creds["client_id"] = config.GoogleClientID
		creds["client_secret"] = config.GoogleClientSecret
		creds["refresh_token"] = config.GoogleRefreshToken
	case "slack":
		creds["bot_token"] = config.SlackBotToken
	case "notion":
		creds["api_key"] = config.NotionAPIKey
	case "linear":
		creds["api_key"] = config.LinearAPIKey
	case "hubspot":
		creds["api_key"] = config.HubSpotAPIKey
	case "anthropic":
		creds["api_key"] = config.AnthropicAPIKey
	case "openai":
		creds["api_key"] = config.OpenAIAPIKey
	case "groq":
		creds["api_key"] = config.GroqAPIKey
	}

	return creds
}

// GenerateTestID generates a unique test identifier
func GenerateTestID() string {
	return fmt.Sprintf("test_%d", time.Now().Unix())
}

// WaitForCondition waits for a condition to be true (for async operations)
func WaitForCondition(condition func() bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for condition")
}
