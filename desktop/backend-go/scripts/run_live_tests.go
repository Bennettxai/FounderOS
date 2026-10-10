//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	categories = flag.String("categories", "all", "Comma-separated list of categories to test (gmail,calendar,slack,notion,linear,hubspot,ai,transform,businessos,all)")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	cleanup    = flag.String("cleanup", "on_success", "Cleanup mode: always, never, on_success")
	timeout    = flag.Duration("timeout", 30*time.Minute, "Overall test timeout")
	reportFile = flag.String("report", "live_test_report.json", "Path to save JSON report")
)

type TestReport struct {
	StartTime     time.Time          `json:"start_time"`
	EndTime       time.Time          `json:"end_time"`
	TotalDuration time.Duration      `json:"total_duration"`
	TotalTests    int                `json:"total_tests"`
	PassedTests   int                `json:"passed_tests"`
	FailedTests   int                `json:"failed_tests"`
	SkippedTests  int                `json:"skipped_tests"`
	Results       []TestResult       `json:"results"`
	Configuration map[string]string  `json:"configuration"`
}

type TestResult struct {
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

func main() {
	flag.Parse()

	printBanner()

	// Check for configuration file
	envFile := checkConfigFile()
	if envFile == "" {
		os.Exit(1)
	}

	fmt.Printf("✓ Using configuration: %s\n", envFile)
	fmt.Println()

	// Display test configuration
	printConfiguration()

	// Set environment variables for test
	os.Setenv("TEST_CATEGORIES", *categories)
	os.Setenv("TEST_VERBOSE", fmt.Sprintf("%v", *verbose))
	os.Setenv("TEST_CLEANUP_MODE", *cleanup)

	// Change to backend-go directory
	backendDir, err := findBackendDir()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
	os.Chdir(backendDir)

	// Build test command
	args := buildTestCommand()

	fmt.Println("Running tests...")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	startTime := time.Now()

	// Run tests
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	err = cmd.Run()
	duration := time.Since(startTime)

	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("Duration: %s\n", formatDuration(duration))
	fmt.Println()

	// Read and display report
	displayReport(*reportFile)

	if err != nil {
		printFailureSummary()
		os.Exit(1)
	}

	printSuccessSummary()
}

func printBanner() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  BusinessOS Live Action Testing")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
}

func checkConfigFile() string {
	envFile := ".env.test.local"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		envFile = ".env.test"
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			fmt.Println("❌ Configuration file not found")
			fmt.Println()
			fmt.Println("Please create .env.test.local with your credentials:")
			fmt.Println("  cp .env.test.local.example .env.test.local")
			fmt.Println("  # Edit .env.test.local with real credentials")
			fmt.Println()
			fmt.Println("For Google OAuth setup, run:")
			fmt.Println("  go run scripts/oauth_setup_helper.go")
			fmt.Println()
			return ""
		}
	}
	return envFile
}

func printConfiguration() {
	fmt.Println("Test Configuration:")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("Categories: %s\n", *categories)
	fmt.Printf("Verbose:    %v\n", *verbose)
	fmt.Printf("Cleanup:    %s\n", *cleanup)
	fmt.Printf("Timeout:    %s\n", *timeout)
	fmt.Printf("Report:     %s\n", *reportFile)
	fmt.Println()
}

func findBackendDir() (string, error) {
	// Try to find backend-go directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check if we're already in backend-go
	if strings.HasSuffix(cwd, "backend-go") {
		return cwd, nil
	}

	// Try desktop/backend-go
	backendDir := filepath.Join(cwd, "desktop", "backend-go")
	if _, err := os.Stat(backendDir); err == nil {
		return backendDir, nil
	}

	// Try going up and finding it
	parentDir := filepath.Dir(cwd)
	backendDir = filepath.Join(parentDir, "desktop", "backend-go")
	if _, err := os.Stat(backendDir); err == nil {
		return backendDir, nil
	}

	return "", fmt.Errorf("cannot find backend-go directory")
}

func buildTestCommand() []string {
	args := []string{
		"test",
		"-tags=integration",
		"-v",
		"-timeout", timeout.String(),
		"./internal/sorx",
		"-run", "TestLiveActions",
	}

	if *verbose {
		args = append(args, "-test.v")
	}

	return args
}

func displayReport(reportPath string) {
	// Check if report exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		fmt.Println("⚠ No test report generated")
		return
	}

	// Read report
	data, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Printf("⚠ Failed to read report: %v\n", err)
		return
	}

	var report TestReport
	if err := json.Unmarshal(data, &report); err != nil {
		fmt.Printf("⚠ Failed to parse report: %v\n", err)
		return
	}

	// Display summary
	fmt.Println("Test Summary:")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("Total Tests:   %d\n", report.TotalTests)
	fmt.Printf("✅ Passed:     %d\n", report.PassedTests)
	fmt.Printf("❌ Failed:     %d\n", report.FailedTests)
	fmt.Printf("⏭ Skipped:     %d\n", report.SkippedTests)
	fmt.Printf("Duration:      %s\n", formatDuration(report.TotalDuration))
	fmt.Println()

	// Display results by category
	categorySummary := make(map[string]struct{ passed, failed, skipped int })
	for _, result := range report.Results {
		summary := categorySummary[result.Category]
		if result.Success {
			summary.passed++
		} else {
			summary.failed++
		}
		categorySummary[result.Category] = summary
	}

	if len(categorySummary) > 0 {
		fmt.Println("Results by Category:")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for category, summary := range categorySummary {
			status := "✅"
			if summary.failed > 0 {
				status = "❌"
			}
			fmt.Printf("%s %-12s  Passed: %d  Failed: %d\n",
				status, category, summary.passed, summary.failed)
		}
		fmt.Println()
	}

	// Show failed tests
	if report.FailedTests > 0 {
		fmt.Println("Failed Tests:")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for _, result := range report.Results {
			if !result.Success {
				fmt.Printf("❌ %s\n", result.Action)
				if result.Error != "" {
					fmt.Printf("   Error: %s\n", result.Error)
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("✓ Full report saved to: %s\n", reportPath)
	fmt.Println()
	fmt.Println("To view detailed JSON report:")
	fmt.Printf("  cat %s\n", reportPath)
	fmt.Println("  # Or formatted:")
	fmt.Printf("  jq . %s\n", reportPath)
	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

func printFailureSummary() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  ❌ Tests Failed")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	printTroubleshootingGuide()
}

func printSuccessSummary() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  ✅ All Tests Passed!")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
}

func printTroubleshootingGuide() {
	fmt.Println("Troubleshooting Guide:")
	fmt.Println()
	fmt.Println("1. Check credentials in .env.test.local")
	fmt.Println("   - Ensure all required fields are filled")
	fmt.Println("   - Verify no extra whitespace in values")
	fmt.Println()
	fmt.Println("2. For Google OAuth errors:")
	fmt.Println("   - Run: go run scripts/oauth_setup_helper.go")
	fmt.Println("   - Ensure redirect URI is http://localhost:8080/oauth/callback")
	fmt.Println("   - Enable Gmail API and Google Calendar API in console")
	fmt.Println("   - Check OAuth consent screen is configured")
	fmt.Println()
	fmt.Println("3. For Slack errors:")
	fmt.Println("   - Verify bot token starts with 'xoxb-'")
	fmt.Println("   - Check bot has chat:write and channels:read permissions")
	fmt.Println("   - Ensure test channel exists and bot is invited")
	fmt.Println("   - Get channel ID from channel details (right-click → View details)")
	fmt.Println()
	fmt.Println("4. For Notion errors:")
	fmt.Println("   - Verify integration is created at notion.so/my-integrations")
	fmt.Println("   - Share test database with integration")
	fmt.Println("   - Check database ID is correct (32 hex chars, no hyphens)")
	fmt.Println("   - Ensure properties match your database schema")
	fmt.Println()
	fmt.Println("5. For Linear/HubSpot errors:")
	fmt.Println("   - Verify API keys are valid and not expired")
	fmt.Println("   - Check permissions/scopes are sufficient")
	fmt.Println("   - Watch for rate limits")
	fmt.Println()
	fmt.Println("6. For AI provider errors:")
	fmt.Println("   - Verify API keys are valid")
	fmt.Println("   - Check rate limits and quotas")
	fmt.Println("   - Ensure models are available in your region")
	fmt.Println()
	fmt.Println("7. To test specific categories only:")
	fmt.Println("   go run scripts/run_live_tests.go --categories=gmail,slack")
	fmt.Println()
	fmt.Println("8. To skip cleanup (for debugging):")
	fmt.Println("   go run scripts/run_live_tests.go --cleanup=never")
	fmt.Println()
	fmt.Println("9. For detailed logs:")
	fmt.Println("   go run scripts/run_live_tests.go --verbose")
	fmt.Println()
	fmt.Println("10. Common Issues:")
	fmt.Println("    - Expired refresh tokens → Re-run oauth_setup_helper.go")
	fmt.Println("    - Invalid channel ID → Check Slack channel details")
	fmt.Println("    - Database not shared → Re-share in Notion")
	fmt.Println("    - Wrong team ID → Check Linear URL for team key")
	fmt.Println()
}
