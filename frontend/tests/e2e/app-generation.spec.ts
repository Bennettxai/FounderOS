/**
 * App Generation E2E Tests
 *
 * Tests for OSA app generation, build monitoring, and deployment.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForElement, waitForApiCall } from './fixtures/helpers';
import { mockOSAApi } from './fixtures/mockApis';
import { testChatMessages } from './fixtures/testData';

test.describe('App Generation', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);
		await mockOSAApi(page);

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test('generate app from chat message', async ({ page }) => {
		// Navigate to chat
		await page.goto('/chat');

		// Type app generation request
		await page.fill('textarea[name="message"]', testChatMessages.appGeneration);

		// Send message
		await page.click('button[aria-label="Send"]');

		// Wait for OSA API call
		const response = await waitForApiCall(page, '/api/osa/generate');
		expect(response.status()).toBe(200);

		// Should show confirmation message
		await expect(page.locator('text=/app.*generation.*started/i')).toBeVisible();

		// Should provide link to generated apps
		await expect(page.locator('a[href="/generated-apps"]')).toBeVisible();
	});

	test('view generated apps list', async ({ page }) => {
		// Navigate to generated apps
		await page.goto('/generated-apps');

		// Should show app list
		await expect(page.locator('[data-testid="app-card"]')).toHaveCount(1);

		// Should show app details
		await expect(page.locator('text=Test CRM')).toBeVisible();
		await expect(page.locator('text=completed')).toBeVisible();
	});

	test('monitor app build progress', async ({ page }) => {
		// Mock SSE streaming for build progress
		await page.route('**/api/osa/generate/*/stream', async (route) => {
			const stream = `
data: {"type":"progress","progress":10,"message":"Initializing..."}

data: {"type":"progress","progress":30,"message":"Analyzing requirements..."}

data: {"type":"progress","progress":60,"message":"Generating code..."}

data: {"type":"progress","progress":90,"message":"Building app..."}

data: {"type":"complete","app_id":"test-app-123"}
`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/generated-apps/test-app-123');

		// Should show build progress component
		await expect(page.locator('[data-testid="build-progress"]')).toBeVisible();

		// Should show progress bar
		await expect(page.locator('[data-testid="progress-bar"]')).toBeVisible();

		// Wait for progress updates
		await page.waitForTimeout(2000);

		// Should show current status
		await expect(page.locator('text=/initializing|analyzing|generating|building/i')).toBeVisible();

		// Should update progress percentage
		const progressText = await page.locator('[data-testid="progress-percentage"]').textContent();
		expect(progressText).toMatch(/\d+%/);
	});

	test('app details page shows metadata', async ({ page }) => {
		await page.goto('/generated-apps/app-1');

		// Should show app name
		await expect(page.locator('h1:has-text("Test CRM")')).toBeVisible();

		// Should show status
		await expect(page.locator('[data-testid="app-status"]')).toContainText('completed');

		// Should show created date
		await expect(page.locator('[data-testid="created-date"]')).toBeVisible();

		// Should show description
		await expect(page.locator('[data-testid="app-description"]')).toBeVisible();
	});

	test('deploy generated app', async ({ page }) => {
		// Mock deployment endpoint
		await page.route('**/api/osa/deployment/*/deploy', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					deployment_url: 'https://test-app.businessos.dev'
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click deploy button
		await page.click('button:has-text("Deploy")');

		// Should show deployment confirmation dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/confirm.*deployment/i')).toBeVisible();

		// Confirm deployment
		await page.click('button:has-text("Confirm")');

		// Wait for deployment API call
		const response = await waitForApiCall(page, '/api/osa/deployment/app-1/deploy');
		expect(response.status()).toBe(200);

		// Should show deployment success message
		await expect(page.locator('text=/deployed.*successfully/i')).toBeVisible();

		// Should show deployment URL
		await expect(page.locator('a[href*="test-app.businessos.dev"]')).toBeVisible();
	});

	test('browse generated app files', async ({ page }) => {
		// Mock file browsing endpoint
		await page.route('**/api/osa/apps/*/files', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					files: [
						{ name: 'src/', type: 'directory' },
						{ name: 'package.json', type: 'file' },
						{ name: 'README.md', type: 'file' }
					]
				})
			});
		});

		await page.goto('/generated-apps/app-1');

		// Click view files button
		await page.click('button:has-text("View Files")');

		// Should show file browser
		await expect(page.locator('[data-testid="file-browser"]')).toBeVisible();

		// Should show file list
		await expect(page.locator('[data-testid="file-item"]')).toHaveCount(3);
		await expect(page.locator('text=src/')).toBeVisible();
		await expect(page.locator('text=package.json')).toBeVisible();
	});

	test('delete generated app', async ({ page }) => {
		// Mock delete endpoint
		await page.route('**/api/osa/apps/*', async (route) => {
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

		// Should show confirmation dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/confirm.*delete/i')).toBeVisible();

		// Confirm deletion
		await page.click('button:has-text("Confirm Delete")');

		// Wait for delete API call
		await page.waitForResponse(response =>
			response.url().includes('/api/osa/apps/app-1') &&
			response.request().method() === 'DELETE'
		);

		// Should redirect to apps list
		await page.waitForURL('**/generated-apps');

		// Should show success message
		await expect(page.locator('text=/deleted.*successfully/i')).toBeVisible();
	});

	test('filter apps by status', async ({ page }) => {
		await page.goto('/generated-apps');

		// Should show status filter
		await expect(page.locator('[data-testid="status-filter"]')).toBeVisible();

		// Select completed filter
		await page.click('[data-testid="status-filter"]');
		await page.click('text=Completed');

		// Should filter apps
		await expect(page.locator('[data-testid="app-card"]')).toHaveCount(1);
		await expect(page.locator('[data-testid="app-card"]:has-text("completed")')).toBeVisible();
	});

	test('search apps by name', async ({ page }) => {
		await page.goto('/generated-apps');

		// Type in search box
		await page.fill('input[placeholder*="Search"]', 'CRM');

		// Should filter results
		await page.waitForTimeout(500);
		await expect(page.locator('text=Test CRM')).toBeVisible();
	});

	test('app generation error handling', async ({ page }) => {
		// Mock error response
		await page.route('**/api/osa/generate', async (route) => {
			await route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({
					success: false,
					error: 'Failed to generate app'
				})
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', testChatMessages.appGeneration);
		await page.click('button[aria-label="Send"]');

		// Should show error message
		await expect(page.locator('text=/failed.*generate/i')).toBeVisible();
	});

	test('update app metadata', async ({ page }) => {
		// Mock update endpoint
		await page.route('**/api/osa/apps/*', async (route) => {
			if (route.request().method() === 'PUT') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ success: true })
				});
			}
		});

		await page.goto('/generated-apps/app-1');

		// Click edit button
		await page.click('button[aria-label="Edit app details"]');

		// Should show edit form
		await expect(page.locator('[data-testid="edit-app-form"]')).toBeVisible();

		// Update name
		await page.fill('input[name="name"]', 'Updated CRM App');
		await page.fill('textarea[name="description"]', 'Updated description');

		// Save
		await page.click('button:has-text("Save")');

		// Wait for update API call
		await page.waitForResponse(response =>
			response.url().includes('/api/osa/apps/app-1') &&
			response.request().method() === 'PUT'
		);

		// Should show success message
		await expect(page.locator('text=/updated.*successfully/i')).toBeVisible();

		// Should show updated name
		await expect(page.locator('h1:has-text("Updated CRM App")')).toBeVisible();
	});

	test('intent detection routes to OSA', async ({ page }) => {
		await page.goto('/chat');

		// Send message that should trigger OSA
		await page.fill('textarea[name="message"]', 'Build a CRM app');
		await page.click('button[aria-label="Send"]');

		// Should detect app generation intent
		await expect(page.locator('text=/detected.*app.*generation/i')).toBeVisible();

		// Should call OSA API
		await waitForApiCall(page, '/api/osa/generate');
	});

	// ═══════════════════════════════════════════════════════════════════════════
	// EDGE CASE TESTS (ISR-1)
	// ═══════════════════════════════════════════════════════════════════════════

	test.describe('SSE Streaming Edge Cases', () => {
		test('SSE connection drop mid-generation', async ({ page, context }) => {
			let progressSent = 0;

			// Mock SSE that drops connection after 2 updates
			await page.route('**/api/osa/generate/*/stream', async (route) => {
				if (progressSent < 2) {
					const stream = `data: {"type":"progress","progress":${progressSent * 30},"message":"Step ${progressSent}"}

`;
					progressSent++;
					await route.fulfill({
						status: 200,
						contentType: 'text/event-stream',
						body: stream
					});
				} else {
					// Simulate connection drop
					await route.abort('failed');
				}
			});

			await page.goto('/generated-apps/test-app-123');

			// Should show initial progress
			await expect(page.locator('[data-testid="build-progress"]')).toBeVisible();

			// Wait for connection drop
			await page.waitForTimeout(3000);

			// Should show connection error message
			await expect(page.locator('text=/connection.*lost|disconnected/i')).toBeVisible();

			// Should show retry button
			await expect(page.locator('button:has-text("Retry")')).toBeVisible();
		});

		test('SSE reconnection after network failure', async ({ page }) => {
			let connectionAttempt = 0;

			await page.route('**/api/osa/generate/*/stream', async (route) => {
				connectionAttempt++;

				// First attempt fails
				if (connectionAttempt === 1) {
					await route.abort('failed');
					return;
				}

				// Second attempt succeeds
				const stream = `
data: {"type":"progress","progress":50,"message":"Reconnected..."}

data: {"type":"complete","app_id":"test-app-123"}
`;
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: stream
				});
			});

			await page.goto('/generated-apps/test-app-123');

			// Should show error first
			await expect(page.locator('text=/connection.*lost|error/i')).toBeVisible();

			// Click retry
			await page.click('button:has-text("Retry")');

			// Should reconnect and show progress
			await expect(page.locator('text=/reconnected/i')).toBeVisible();
			await expect(page.locator('text=/complete/i')).toBeVisible();
		});

		test('stuck progress bar scenario', async ({ page }) => {
			// Mock SSE that gets stuck at 50%
			await page.route('**/api/osa/generate/*/stream', async (route) => {
				const stream = `
data: {"type":"progress","progress":10,"message":"Starting..."}

data: {"type":"progress","progress":50,"message":"Processing..."}
`;
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: stream
				});
			});

			await page.goto('/generated-apps/test-app-123');

			// Wait for progress to appear
			await expect(page.locator('[data-testid="progress-bar"]')).toBeVisible();

			// Wait for timeout (simulate stuck progress)
			await page.waitForTimeout(35000); // Longer than 30s timeout

			// Should show timeout warning
			await expect(page.locator('text=/taking longer than expected|timeout/i')).toBeVisible();

			// Should show option to continue waiting or cancel
			await expect(page.locator('button:has-text("Continue Waiting")')).toBeVisible();
			await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
		});

		test('SSE error recovery UI', async ({ page }) => {
			// Mock SSE with error event
			await page.route('**/api/osa/generate/*/stream', async (route) => {
				const stream = `
data: {"type":"progress","progress":30,"message":"Analyzing..."}

data: {"type":"error","error":"Build failed: Syntax error","details":"Missing semicolon at line 42"}
`;
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: stream
				});
			});

			await page.goto('/generated-apps/test-app-123');

			// Should show error message
			await expect(page.locator('[data-testid="error-message"]')).toBeVisible();
			await expect(page.locator('text=/build failed.*syntax error/i')).toBeVisible();

			// Should show error details
			await expect(page.locator('text=/missing semicolon/i')).toBeVisible();

			// Should show recovery options
			await expect(page.locator('button:has-text("Try Again")')).toBeVisible();
			await expect(page.locator('button:has-text("View Logs")')).toBeVisible();
		});

		test('multiple SSE connections handling', async ({ page }) => {
			// Mock multiple app generation streams
			await page.route('**/api/osa/generate/app-1/stream', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: 'data: {"type":"progress","progress":50,"message":"App 1 building..."}\n\n'
				});
			});

			await page.route('**/api/osa/generate/app-2/stream', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: 'data: {"type":"progress","progress":75,"message":"App 2 building..."}\n\n'
				});
			});

			// Open first app in main page
			await page.goto('/generated-apps/app-1');
			await expect(page.locator('text=/app 1 building/i')).toBeVisible();

			// Open second app in new tab
			const newPage = await page.context().newPage();
			await newPage.goto('/generated-apps/app-2');
			await expect(newPage.locator('text=/app 2 building/i')).toBeVisible();

			// Both streams should work independently
			await expect(page.locator('text=/50%|progress.*50/i')).toBeVisible();
			await expect(newPage.locator('text=/75%|progress.*75/i')).toBeVisible();

			await newPage.close();
		});
	});

	test.describe('Network Error Recovery', () => {
		test('timeout handling for long requests', async ({ page }) => {
			// Mock API that takes too long
			await page.route('**/api/osa/deployment/test-app/deploy', async (route) => {
				await page.waitForTimeout(35000); // Exceeds 30s timeout
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ success: true })
				});
			});

			await page.goto('/generated-apps/test-app');

			// Click deploy
			await page.click('button:has-text("Deploy")');
			await page.click('button:has-text("Confirm")');

			// Should show timeout error
			await expect(page.locator('text=/request.*timed out|timeout/i')).toBeVisible({
				timeout: 40000
			});

			// Should show retry option
			await expect(page.locator('button:has-text("Retry")')).toBeVisible();
		});

		test('connection lost during fetch', async ({ page }) => {
			// Mock network failure
			await page.route('**/api/osa/apps', async (route) => {
				await route.abort('connectionfailed');
			});

			await page.goto('/generated-apps');

			// Should show network error message
			await expect(page.locator('text=/network error|connection failed/i')).toBeVisible();

			// Should show offline indicator
			await expect(page.locator('[data-testid="offline-indicator"]')).toBeVisible();
		});

		test('retry mechanism with exponential backoff', async ({ page }) => {
			let attemptCount = 0;

			await page.route('**/api/osa/apps/test-app', async (route) => {
				attemptCount++;

				// Fail first 2 attempts, succeed on 3rd
				if (attemptCount < 3) {
					await route.fulfill({
						status: 500,
						contentType: 'application/json',
						body: JSON.stringify({ error: 'Server error' })
					});
				} else {
					await route.fulfill({
						status: 200,
						contentType: 'application/json',
						body: JSON.stringify({
							app: {
								id: 'test-app',
								name: 'Test App',
								status: 'completed'
							}
						})
					});
				}
			});

			await page.goto('/generated-apps/test-app');

			// Should show retry indicator
			await expect(page.locator('text=/retrying|attempt/i')).toBeVisible();

			// Should eventually succeed
			await expect(page.locator('h1:has-text("Test App")')).toBeVisible({
				timeout: 15000
			});
		});

		test('error messages display correctly', async ({ page }) => {
			// Mock 404 error
			await page.route('**/api/osa/apps/nonexistent', async (route) => {
				await route.fulfill({
					status: 404,
					contentType: 'application/json',
					body: JSON.stringify({
						error: 'App not found',
						message: 'The requested app does not exist'
					})
				});
			});

			await page.goto('/generated-apps/nonexistent');

			// Should show user-friendly 404 message
			await expect(page.locator('text=/app not found/i')).toBeVisible();
			await expect(page.locator('text=/does not exist/i')).toBeVisible();

			// Should show link back to apps list
			await expect(page.locator('a[href="/generated-apps"]')).toBeVisible();
		});

		test('API rate limiting handling', async ({ page }) => {
			// Mock 429 rate limit error
			await page.route('**/api/osa/generate', async (route) => {
				await route.fulfill({
					status: 429,
					contentType: 'application/json',
					headers: {
						'Retry-After': '60'
					},
					body: JSON.stringify({
						error: 'Rate limit exceeded',
						retry_after: 60
					})
				});
			});

			await page.goto('/chat');

			await page.fill('textarea[name="message"]', 'Build a CRM app');
			await page.click('button[aria-label="Send"]');

			// Should show rate limit message
			await expect(page.locator('text=/rate limit|too many requests/i')).toBeVisible();

			// Should show retry time
			await expect(page.locator('text=/try again in.*60.*seconds/i')).toBeVisible();
		});
	});

	test.describe('Concurrent Operations', () => {
		test('multiple rapid clicks on Deploy button', async ({ page }) => {
			let deployCallCount = 0;

			await page.route('**/api/osa/deployment/*/deploy', async (route) => {
				deployCallCount++;
				// Simulate slow deployment
				await page.waitForTimeout(2000);
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ success: true })
				});
			});

			await page.goto('/generated-apps/app-1');

			const deployButton = page.locator('button:has-text("Deploy")');

			// Click multiple times rapidly
			await deployButton.click();
			await page.click('button:has-text("Confirm")');

			// Try clicking again while deploying
			await page.waitForTimeout(100);
			const confirmButton = page.locator('button:has-text("Confirm")');
			const isDisabled = await confirmButton.isDisabled();

			// Button should be disabled during deployment
			expect(isDisabled).toBe(true);

			// Should only trigger one deployment
			await page.waitForTimeout(3000);
			expect(deployCallCount).toBe(1);
		});

		test('delete during generation', async ({ page }) => {
			// Mock SSE for ongoing generation
			await page.route('**/api/osa/generate/*/stream', async (route) => {
				const stream = `data: {"type":"progress","progress":50,"message":"Building..."}

`;
				await route.fulfill({
					status: 200,
					contentType: 'text/event-stream',
					body: stream
				});
			});

			await page.goto('/generated-apps/building-app');

			// Should show warning when trying to delete building app
			await page.click('button:has-text("Delete")');

			await expect(page.locator('text=/app is currently building/i')).toBeVisible();
			await expect(page.locator('text=/are you sure.*cancel/i')).toBeVisible();

			// Should have stronger confirmation
			await expect(page.locator('input[placeholder*="type DELETE"]')).toBeVisible();
		});

		test('state consistency after errors', async ({ page }) => {
			// Mock deployment that fails
			await page.route('**/api/osa/deployment/app-1/deploy', async (route) => {
				await route.fulfill({
					status: 500,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'Deployment failed' })
				});
			});

			await page.goto('/generated-apps/app-1');

			// Initial status
			const initialStatus = await page.locator('[data-testid="app-status"]').textContent();

			// Try to deploy
			await page.click('button:has-text("Deploy")');
			await page.click('button:has-text("Confirm")');

			// Wait for error
			await expect(page.locator('text=/deployment failed/i')).toBeVisible();

			// Status should remain unchanged
			const afterErrorStatus = await page.locator('[data-testid="app-status"]').textContent();
			expect(afterErrorStatus).toBe(initialStatus);

			// Deploy button should still be available
			await expect(page.locator('button:has-text("Deploy")')).toBeEnabled();
		});

		test('concurrent updates to same app', async ({ page, context }) => {
			let updateCount = 0;

			await page.route('**/api/osa/apps/app-1', async (route) => {
				if (route.request().method() === 'PUT') {
					updateCount++;
					await route.fulfill({
						status: 200,
						contentType: 'application/json',
						body: JSON.stringify({ success: true })
					});
				}
			});

			// Open app in two tabs
			await page.goto('/generated-apps/app-1');
			const secondPage = await context.newPage();
			await secondPage.goto('/generated-apps/app-1');

			// Try to update from both tabs
			await page.click('button[aria-label="Edit app details"]');
			await secondPage.click('button[aria-label="Edit app details"]');

			await page.fill('input[name="name"]', 'Update from Tab 1');
			await secondPage.fill('input[name="name"]', 'Update from Tab 2');

			await page.click('button:has-text("Save")');
			await secondPage.click('button:has-text("Save")');

			// Should handle gracefully (last write wins or show conflict)
			await page.waitForTimeout(2000);

			// Should show some indication of concurrent update
			const hasConflictWarning = await page.isVisible('text=/conflict|updated by another/i');
			const hasSuccessMessage = await page.isVisible('text=/updated successfully/i');

			expect(hasConflictWarning || hasSuccessMessage).toBe(true);

			await secondPage.close();
		});
	});

	test.describe('Empty States & Transitions', () => {
		test('initial load to loading to empty state', async ({ page }) => {
			// Mock empty apps list
			await page.route('**/api/osa/apps', async (route) => {
				// Simulate slow network
				await page.waitForTimeout(1000);
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ apps: [] })
				});
			});

			await page.goto('/generated-apps');

			// Should show loading state first
			await expect(page.locator('[data-testid="loading-spinner"]')).toBeVisible();

			// Should transition to empty state
			await expect(page.locator('[data-testid="empty-state"]')).toBeVisible({
				timeout: 5000
			});

			// Should show helpful empty state message
			await expect(page.locator('text=/no apps yet|get started/i')).toBeVisible();

			// Should show CTA to create first app
			await expect(page.locator('button:has-text("Generate App")')).toBeVisible();
		});

		test('filter to no results to clear filter', async ({ page }) => {
			// Mock apps list
			await page.route('**/api/osa/apps', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: [
							{ id: 'app-1', name: 'CRM App', status: 'completed' }
						]
					})
				});
			});

			await page.goto('/generated-apps');

			// Should show app
			await expect(page.locator('text=CRM App')).toBeVisible();

			// Apply filter that returns no results
			await page.click('[data-testid="status-filter"]');
			await page.click('text=Failed');

			// Should show "no results" state
			await expect(page.locator('text=/no.*apps.*found|no results/i')).toBeVisible();

			// Should show clear filters button
			const clearButton = page.locator('button:has-text("Clear Filters")');
			await expect(clearButton).toBeVisible();

			// Clear filters
			await clearButton.click();

			// Should show apps again
			await expect(page.locator('text=CRM App')).toBeVisible();
		});

		test('search to no matches to UI feedback', async ({ page }) => {
			await page.route('**/api/osa/apps', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: [
							{ id: 'app-1', name: 'CRM App', status: 'completed' }
						]
					})
				});
			});

			await page.goto('/generated-apps');

			// Search for non-existent app
			const searchInput = page.locator('input[placeholder*="Search"]');
			await searchInput.fill('NonexistentApp');

			// Should show no matches message
			await expect(page.locator('text=/no apps match.*search/i')).toBeVisible();

			// Should show search query in message
			await expect(page.locator('text=/nonexistentapp/i')).toBeVisible();

			// Clear search
			await searchInput.clear();

			// Should show apps again
			await expect(page.locator('text=CRM App')).toBeVisible();
		});

		test('loading state transitions smoothly', async ({ page }) => {
			let requestCount = 0;

			await page.route('**/api/osa/apps', async (route) => {
				requestCount++;
				await page.waitForTimeout(500);
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: [{ id: 'app-1', name: 'Test App', status: 'completed' }]
					})
				});
			});

			await page.goto('/generated-apps');

			// Should show skeleton loaders during initial load
			const hasSkeletons = await page.isVisible('[data-testid="skeleton-card"]');

			// Content should appear smoothly (no flash of empty state)
			await expect(page.locator('text=Test App')).toBeVisible({
				timeout: 2000
			});

			// Shouldn't have shown empty state during loading
			const emptyStateShown = await page.isVisible('[data-testid="empty-state"]');
			expect(emptyStateShown).toBe(false);
		});

		test('pagination edge cases', async ({ page }) => {
			// Mock large dataset
			const generateApps = (page: number, perPage: number) => {
				return Array.from({ length: perPage }, (_, i) => ({
					id: `app-${page}-${i}`,
					name: `App ${page}-${i}`,
					status: 'completed'
				}));
			};

			await page.route('**/api/osa/apps*', async (route) => {
				const url = new URL(route.request().url());
				const page = parseInt(url.searchParams.get('page') || '1');
				const perPage = parseInt(url.searchParams.get('per_page') || '10');

				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: page <= 3 ? generateApps(page, perPage) : [],
						total: 25,
						page,
						per_page: perPage,
						total_pages: 3
					})
				});
			});

			await page.goto('/generated-apps');

			// Should show page 1
			await expect(page.locator('[data-testid="app-card"]')).toHaveCount(10);

			// Navigate to last page
			await page.click('button[aria-label="Last page"]');

			// Should show remaining items
			await expect(page.locator('[data-testid="app-card"]')).toHaveCount(5);

			// Next button should be disabled
			await expect(page.locator('button[aria-label="Next page"]')).toBeDisabled();
		});
	});
});
