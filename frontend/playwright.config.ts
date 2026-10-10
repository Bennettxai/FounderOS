import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Testing Configuration for BusinessOS
 *
 * This configuration sets up comprehensive end-to-end testing:
 * - Multiple browsers (Chromium, Firefox, WebKit)
 * - Parallel execution with 4 workers
 * - Automatic retries on failure
 * - Screenshots and videos on failure
 * - 30-second timeout per test
 */
export default defineConfig({
	testDir: './tests/e2e',

	// Maximum time one test can run
	timeout: 30 * 1000,

	// Test expectations timeout
	expect: {
		timeout: 5000
	},

	// Run tests in parallel
	fullyParallel: true,

	// Fail the build on CI if you accidentally left test.only in the source code
	forbidOnly: !!process.env.CI,

	// Retry on CI only
	retries: process.env.CI ? 2 : 0,

	// Parallel workers
	workers: process.env.CI ? 2 : 4,

	// Reporter configuration
	reporter: [
		['html'],
		['list'],
		process.env.CI ? ['github'] : ['list']
	],

	// Shared settings for all projects
	use: {
		// Base URL for navigation
		baseURL: 'http://localhost:5173',

		// Collect trace on first retry of a failed test
		trace: 'on-first-retry',

		// Screenshot on failure
		screenshot: 'only-on-failure',

		// Video on failure
		video: 'retain-on-failure',

		// Browser context options
		viewport: { width: 1440, height: 900 },

		// Ignore HTTPS errors (for local development)
		ignoreHTTPSErrors: true,

		// Default navigation timeout
		navigationTimeout: 10000,

		// Default action timeout
		actionTimeout: 5000,
	},

	// Configure projects for major browsers
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] },
		},

		{
			name: 'firefox',
			use: { ...devices['Desktop Firefox'] },
		},

		{
			name: 'webkit',
			use: { ...devices['Desktop Safari'] },
		},

		// Mobile viewports for responsive testing
		{
			name: 'Mobile Chrome',
			use: { ...devices['Pixel 5'] },
		},

		{
			name: 'Mobile Safari',
			use: { ...devices['iPhone 12'] },
		},
	],

	// Run your local dev server before starting the tests
	webServer: {
		command: 'npm run dev',
		url: 'http://localhost:5173',
		reuseExistingServer: !process.env.CI,
		timeout: 120 * 1000,
		stdout: 'ignore',
		stderr: 'pipe',
	},
});
