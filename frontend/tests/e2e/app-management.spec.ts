/**
 * App Management E2E Tests
 *
 * Tests for listing, viewing, deploying, stopping, and deleting generated apps.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForApiCall } from './fixtures/helpers';
import { testApps } from './fixtures/testData';

test.describe('App Management', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);

		// Mock apps list endpoint
		await page.route('**/api/osa/apps', async (route) => {
			if (route.request().method() === 'GET') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ apps: testApps })
				});
			}
		});

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test('list generated apps', async ({ page }) => {
		await page.goto('/generated-apps');

		// Should show all apps
		await expect(page.locator('[data-testid="app-card"]')).toHaveCount(testApps.length);

		// Should display app names and statuses
		for (const app of testApps) {
			await expect(page.locator(`text=${app.name}`)).toBeVisible();
			await expect(page.locator(`text=${app.status}`)).toBeVisible();
		}
	});

	test('view app details', async ({ page }) => {
		// Mock single app endpoint
		await page.route('**/api/osa/apps/app-1', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					app: {
						id: 'app-1',
						name: 'Test CRM App',
						description: 'A CRM system',
						status: 'completed',
						created_at: new Date().toISOString(),
						updated_at: new Date().toISOString()
					}
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Should show app details
		await expect(page.locator('h1:has-text("Test CRM App")')).toBeVisible();
		await expect(page.locator('[data-testid="app-status"]')).toContainText('completed');
		await expect(page.locator('[data-testid="app-description"]')).toBeVisible();
	});

	test('deploy app', async ({ page }) => {
		// Mock deployment endpoint
		await page.route('**/api/osa/deployment/app-1/deploy', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					deployment_url: 'https://app-1.businessos.dev',
					status: 'deploying'
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click deploy button
		await page.click('button:has-text("Deploy")');

		// Should show confirmation dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/confirm.*deploy/i')).toBeVisible();

		// Confirm deployment
		await page.click('button:has-text("Confirm")');

		// Wait for deployment API call
		const response = await waitForApiCall(page, '/api/osa/deployment/app-1/deploy');
		expect(response.status()).toBe(200);

		// Should show deployment status
		await expect(page.locator('text=/deploying/i')).toBeVisible();

		// Should show deployment URL when ready
		await expect(page.locator('a[href*="app-1.businessos.dev"]')).toBeVisible();
	});

	test('stop running app', async ({ page }) => {
		// Mock stop endpoint
		await page.route('**/api/osa/deployment/app-1/stop', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					status: 'stopped'
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click stop button
		await page.click('button:has-text("Stop")');

		// Should show confirmation
		await expect(page.locator('[role="dialog"]')).toBeVisible();

		// Confirm stop
		await page.click('button:has-text("Confirm")');

		// Wait for stop API call
		await waitForApiCall(page, '/api/osa/deployment/app-1/stop');

		// Should update status
		await expect(page.locator('[data-testid="app-status"]')).toContainText('stopped');
	});

	test('delete app with confirmation', async ({ page }) => {
		// Mock delete endpoint
		await page.route('**/api/osa/apps/app-1', async (route) => {
			if (route.request().method() === 'DELETE') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ success: true })
				});
			}
		});

		await page.goto('/generated-apps/app-1');

		// Click delete button
		await page.click('button:has-text("Delete")');

		// Should show confirmation dialog with warning
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/this action cannot be undone/i')).toBeVisible();

		// Type confirmation text
		await page.fill('input[placeholder*="confirm"]', 'DELETE');

		// Confirm button should be enabled
		const confirmButton = page.locator('button:has-text("Confirm Delete")');
		await expect(confirmButton).toBeEnabled();

		// Confirm deletion
		await confirmButton.click();

		// Wait for delete API call
		await page.waitForResponse(response =>
			response.url().includes('/api/osa/apps/app-1') &&
			response.request().method() === 'DELETE'
		);

		// Should redirect to apps list
		await page.waitForURL('**/generated-apps');

		// Should show success message
		await expect(page.locator('text=/deleted.*successfully/i')).toBeVisible();

		// App should not appear in list
		await expect(page.locator('text=Test CRM App')).not.toBeVisible();
	});

	test('cancel app deletion', async ({ page }) => {
		await page.goto('/generated-apps/app-1');

		// Click delete button
		await page.click('button:has-text("Delete")');

		// Should show confirmation dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();

		// Click cancel
		await page.click('button:has-text("Cancel")');

		// Dialog should close
		await expect(page.locator('[role="dialog"]')).not.toBeVisible();

		// Should still be on app page
		await expect(page).toHaveURL(/\/generated-apps\/app-1/);
	});

	test('app status indicators', async ({ page }) => {
		await page.goto('/generated-apps');

		// Should show different status indicators
		await expect(page.locator('[data-testid="status-badge"]:has-text("completed")')).toBeVisible();
		await expect(page.locator('[data-testid="status-badge"]:has-text("building")')).toBeVisible();
		await expect(page.locator('[data-testid="status-badge"]:has-text("pending")')).toBeVisible();

		// Status colors should differ
		const completedBadge = page.locator('[data-testid="status-badge"]:has-text("completed")');
		const buildingBadge = page.locator('[data-testid="status-badge"]:has-text("building")');

		// Different states should have different visual styles
		await expect(completedBadge).toHaveClass(/success|green/);
		await expect(buildingBadge).toHaveClass(/warning|yellow/);
	});

	test('view app logs', async ({ page }) => {
		// Mock logs endpoint
		await page.route('**/api/osa/apps/app-1/logs', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					logs: [
						{ timestamp: new Date().toISOString(), level: 'info', message: 'App started' },
						{ timestamp: new Date().toISOString(), level: 'info', message: 'Server listening on port 3000' }
					]
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click view logs button
		await page.click('button:has-text("View Logs")');

		// Should show logs panel
		await expect(page.locator('[data-testid="logs-panel"]')).toBeVisible();

		// Should display log entries
		await expect(page.locator('text=App started')).toBeVisible();
		await expect(page.locator('text=Server listening on port 3000')).toBeVisible();
	});

	test('restart app', async ({ page }) => {
		// Mock restart endpoint
		await page.route('**/api/osa/apps/app-1/restart', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					status: 'restarting'
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click restart button
		await page.click('button:has-text("Restart")');

		// Wait for restart API call
		await waitForApiCall(page, '/api/osa/apps/app-1/restart');

		// Should show restarting status
		await expect(page.locator('text=/restarting/i')).toBeVisible();
	});

	test('app metrics and analytics', async ({ page }) => {
		// Mock metrics endpoint
		await page.route('**/api/osa/apps/app-1/metrics', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					uptime: '99.9%',
					requests: 15000,
					errors: 12,
					response_time: '45ms'
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Should show metrics section
		await expect(page.locator('[data-testid="app-metrics"]')).toBeVisible();

		// Should display key metrics
		await expect(page.locator('text=99.9%')).toBeVisible(); // uptime
		await expect(page.locator('text=15,000')).toBeVisible(); // requests
		await expect(page.locator('text=45ms')).toBeVisible(); // response time
	});

	test('export app configuration', async ({ page }) => {
		// Mock export endpoint
		await page.route('**/api/osa/apps/app-1/export', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					config: {
						name: 'Test CRM App',
						version: '1.0.0',
						environment: 'production'
					}
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click export button
		await page.click('button:has-text("Export")');

		// Should trigger download
		const downloadPromise = page.waitForEvent('download');
		await page.click('button:has-text("Download Config")');

		const download = await downloadPromise;
		expect(download.suggestedFilename()).toContain('config');
	});
});
