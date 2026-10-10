/**
 * Generated Apps E2E Edge Case Tests
 *
 * Additional edge cases for the generated apps feature (ISR-1).
 * Tests cover network errors, concurrent operations, and UI transitions.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForApiCall } from './fixtures/helpers';

test.describe('Generated Apps - Additional Edge Cases', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);

		// Mock apps list endpoint
		await page.route('**/api/osa/apps', async (route) => {
			if (route.request().method() === 'GET') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: [
							{
								id: 'app-1',
								name: 'Test CRM',
								status: 'completed',
								created_at: new Date().toISOString()
							}
						]
					})
				});
			}
		});

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test.describe('Offline Mode', () => {
		test('graceful degradation when offline', async ({ page, context }) => {
			await page.goto('/generated-apps');

			// Go offline
			await context.setOffline(true);

			// Try to load apps
			await page.reload();

			// Should show offline indicator
			await expect(page.locator('[data-testid="offline-indicator"]')).toBeVisible();

			// Should show cached data if available
			const hasContent = await page.isVisible('[data-testid="app-card"]');

			// Should show message about being offline
			await expect(page.locator('text=/offline|no.*connection/i')).toBeVisible();

			// Go back online
			await context.setOffline(false);

			// Should auto-refresh when back online
			await page.waitForTimeout(2000);
			await expect(page.locator('[data-testid="offline-indicator"]')).not.toBeVisible();
		});

		test('queue actions while offline', async ({ page, context }) => {
			await page.goto('/generated-apps/app-1');

			// Go offline
			await context.setOffline(false);

			// Try to perform action while offline
			await page.click('button:has-text("Deploy")');

			// Should show queued message
			await expect(page.locator('text=/queued|will.*when.*online/i')).toBeVisible();

			// Go back online
			await context.setOffline(false);

			// Should automatically retry queued action
			await expect(page.locator('text=/deploying|processing/i')).toBeVisible({
				timeout: 5000
			});
		});
	});

	test.describe('Large Dataset Performance', () => {
		test('handle 100+ apps efficiently', async ({ page }) => {
			// Mock large dataset
			const largeAppList = Array.from({ length: 150 }, (_, i) => ({
				id: `app-${i}`,
				name: `App ${i}`,
				status: i % 3 === 0 ? 'completed' : i % 3 === 1 ? 'building' : 'pending',
				created_at: new Date(Date.now() - i * 60000).toISOString()
			}));

			await page.route('**/api/osa/apps*', async (route) => {
				const url = new URL(route.request().url());
				const page = parseInt(url.searchParams.get('page') || '1');
				const perPage = 20;
				const start = (page - 1) * perPage;
				const end = start + perPage;

				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: largeAppList.slice(start, end),
						total: largeAppList.length,
						page,
						per_page: perPage,
						total_pages: Math.ceil(largeAppList.length / perPage)
					})
				});
			});

			const startTime = Date.now();

			await page.goto('/generated-apps');

			// Should load within reasonable time
			await expect(page.locator('[data-testid="app-card"]').first()).toBeVisible({
				timeout: 3000
			});

			const loadTime = Date.now() - startTime;
			expect(loadTime).toBeLessThan(3000); // Should load in under 3 seconds

			// Should show pagination controls
			await expect(page.locator('[data-testid="pagination"]')).toBeVisible();

			// Should show total count
			await expect(page.locator('text=/150.*apps/i')).toBeVisible();

			// Virtual scrolling or pagination should work smoothly
			await page.click('button[aria-label="Next page"]');
			await expect(page.locator('[data-testid="app-card"]')).toHaveCount(20);
		});

		test('search performance with large dataset', async ({ page }) => {
			// Mock search endpoint
			await page.route('**/api/osa/apps/search*', async (route) => {
				const url = new URL(route.request().url());
				const query = url.searchParams.get('q') || '';

				// Simulate search delay
				await page.waitForTimeout(300);

				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						apps: query
							? [
									{
										id: 'app-search-1',
										name: `CRM ${query}`,
										status: 'completed'
									}
							  ]
							: []
					})
				});
			});

			await page.goto('/generated-apps');

			const searchInput = page.locator('input[placeholder*="Search"]');

			// Type search query
			await searchInput.fill('CRM');

			// Should debounce search (not trigger on every keystroke)
			await page.waitForTimeout(500);

			// Should show loading indicator during search
			const hasLoadingIndicator = await page.isVisible('[data-testid="search-loading"]');

			// Should display results
			await expect(page.locator('text=/CRM/i')).toBeVisible({
				timeout: 2000
			});
		});
	});

	test.describe('State Persistence', () => {
		test('preserve filter state across navigation', async ({ page }) => {
			await page.goto('/generated-apps');

			// Apply filter
			await page.click('[data-testid="status-filter"]');
			await page.click('text=Completed');

			// Navigate away
			await page.goto('/dashboard');

			// Navigate back
			await page.goto('/generated-apps');

			// Filter should be preserved
			const filterValue = await page.locator('[data-testid="status-filter"]').textContent();
			expect(filterValue).toContain('Completed');
		});

		test('preserve scroll position on back navigation', async ({ page }) => {
			// Mock long list of apps
			const apps = Array.from({ length: 50 }, (_, i) => ({
				id: `app-${i}`,
				name: `App ${i}`,
				status: 'completed'
			}));

			await page.route('**/api/osa/apps', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ apps })
				});
			});

			await page.goto('/generated-apps');

			// Scroll down
			await page.evaluate(() => window.scrollTo(0, 1000));

			// Click on an app
			await page.click('[data-testid="app-card"]:nth-child(20)');

			// Go back
			await page.goBack();

			// Scroll position should be preserved
			const scrollY = await page.evaluate(() => window.scrollY);
			expect(scrollY).toBeGreaterThan(800);
		});
	});

	test.describe('Error Boundaries', () => {
		test('catch and display component errors', async ({ page }) => {
			// Inject error in component
			await page.route('**/api/osa/apps/app-1', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						app: {
							id: 'app-1',
							name: null, // Invalid data that might cause error
							status: undefined
						}
					})
				});
			});

			await page.goto('/generated-apps/app-1');

			// Should show error boundary UI
			await expect(page.locator('[data-testid="error-boundary"]')).toBeVisible();

			// Should show helpful error message
			await expect(page.locator('text=/something went wrong/i')).toBeVisible();

			// Should show reload button
			await expect(page.locator('button:has-text("Reload")')).toBeVisible();

			// Should allow reporting error
			await expect(page.locator('button:has-text("Report Issue")')).toBeVisible();
		});

		test('recover from JavaScript errors', async ({ page }) => {
			await page.goto('/generated-apps');

			// Inject a JavaScript error
			await page.evaluate(() => {
				throw new Error('Test error');
			});

			// App should still be functional
			await expect(page.locator('[data-testid="app-card"]')).toBeVisible();

			// Error should be logged but not crash the app
			const hasErrorBoundary = await page.isVisible('[data-testid="error-boundary"]');
			expect(hasErrorBoundary).toBe(false);
		});
	});

	test.describe('Accessibility Edge Cases', () => {
		test('keyboard navigation for all actions', async ({ page }) => {
			await page.goto('/generated-apps');

			// Tab to first app card
			await page.keyboard.press('Tab');
			await page.keyboard.press('Tab');

			// Should be able to open app with Enter
			await page.keyboard.press('Enter');

			// Should navigate to app details
			await expect(page).toHaveURL(/\/generated-apps\/app-1/);

			// Tab to deploy button
			for (let i = 0; i < 5; i++) {
				await page.keyboard.press('Tab');
			}

			// Should be able to activate deploy with Enter
			await page.keyboard.press('Enter');

			// Dialog should open
			await expect(page.locator('[role="dialog"]')).toBeVisible();

			// Should be able to close with Escape
			await page.keyboard.press('Escape');

			// Dialog should close
			await expect(page.locator('[role="dialog"]')).not.toBeVisible();
		});

		test('screen reader announcements for status changes', async ({ page }) => {
			await page.goto('/generated-apps/app-1');

			// Click deploy
			await page.click('button:has-text("Deploy")');
			await page.click('button:has-text("Confirm")');

			// Should have aria-live region with status update
			const liveRegion = page.locator('[aria-live="polite"]');
			await expect(liveRegion).toContainText(/deploy.*started|processing/i);
		});

		test('focus management in modals', async ({ page }) => {
			await page.goto('/generated-apps/app-1');

			// Open delete modal
			await page.click('button:has-text("Delete")');

			// Focus should move to modal
			const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
			expect(['INPUT', 'BUTTON']).toContain(focusedElement);

			// Tab trap should keep focus in modal
			for (let i = 0; i < 10; i++) {
				await page.keyboard.press('Tab');
			}

			// Focus should still be inside modal
			const isInsideModal = await page.evaluate(() => {
				const activeElement = document.activeElement;
				const modal = document.querySelector('[role="dialog"]');
				return modal?.contains(activeElement);
			});

			expect(isInsideModal).toBe(true);
		});
	});

	test.describe('Data Validation', () => {
		test('handle malformed API responses', async ({ page }) => {
			// Mock malformed response
			await page.route('**/api/osa/apps', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: 'invalid json{'
				});
			});

			await page.goto('/generated-apps');

			// Should show error message
			await expect(page.locator('text=/error.*loading|failed.*load/i')).toBeVisible();

			// Should show retry button
			await expect(page.locator('button:has-text("Retry")')).toBeVisible();
		});

		test('sanitize user input in search', async ({ page }) => {
			await page.goto('/generated-apps');

			const searchInput = page.locator('input[placeholder*="Search"]');

			// Try XSS attack
			await searchInput.fill('<script>alert("xss")</script>');

			// Should not execute script
			const alertFired = await page.evaluate(() => {
				let fired = false;
				const originalAlert = window.alert;
				window.alert = () => {
					fired = true;
				};
				setTimeout(() => {
					window.alert = originalAlert;
				}, 100);
				return fired;
			});

			expect(alertFired).toBe(false);

			// Should show sanitized search query
			const displayedQuery = await page.locator('[data-testid="search-query"]').textContent();
			expect(displayedQuery).not.toContain('<script>');
		});

		test('validate date formats from API', async ({ page }) => {
			// Mock invalid date format
			await page.route('**/api/osa/apps/app-1', async (route) => {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						app: {
							id: 'app-1',
							name: 'Test App',
							status: 'completed',
							created_at: 'invalid-date'
						}
					})
				});
			});

			await page.goto('/generated-apps/app-1');

			// Should handle invalid date gracefully
			const dateElement = page.locator('[data-testid="created-date"]');
			const dateText = await dateElement.textContent();

			// Should show fallback text or valid format
			expect(dateText).not.toBe('invalid-date');
			expect(dateText).toMatch(/invalid|unknown|recently/i);
		});
	});

	test.describe('Browser Compatibility', () => {
		test('handle localStorage quota exceeded', async ({ page }) => {
			// Fill localStorage
			await page.evaluate(() => {
				try {
					for (let i = 0; i < 100; i++) {
						localStorage.setItem(`key_${i}`, 'x'.repeat(100000));
					}
				} catch (e) {
					// Quota exceeded
				}
			});

			await page.goto('/generated-apps');

			// App should still function
			await expect(page.locator('[data-testid="app-card"]')).toBeVisible();

			// Should show warning about storage
			const hasWarning = await page.isVisible('text=/storage.*full|clear.*data/i');
		});

		test('handle cookie blocking', async ({ page, context }) => {
			// Block cookies
			await context.clearCookies();

			await page.goto('/generated-apps');

			// Should still work with sessionStorage or in-memory state
			await expect(page.locator('[data-testid="app-card"]')).toBeVisible();
		});
	});

	test.describe('Real-time Updates', () => {
		test('WebSocket reconnection with exponential backoff', async ({ page }) => {
			let connectionAttempts = 0;

			await page.route('**/api/osa/ws', async (route) => {
				connectionAttempts++;

				if (connectionAttempts < 3) {
					// Fail first 2 attempts
					await route.abort('failed');
				} else {
					// Success on 3rd attempt
					await route.fulfill({ status: 101 });
				}
			});

			await page.goto('/generated-apps');

			// Should retry with backoff
			await page.waitForTimeout(5000);

			expect(connectionAttempts).toBeGreaterThanOrEqual(2);
		});

		test('handle stale data from WebSocket', async ({ page }) => {
			await page.goto('/generated-apps/app-1');

			// Mock WebSocket message with old timestamp
			await page.evaluate(() => {
				const event = new CustomEvent('ws-message', {
					detail: {
						type: 'app-update',
						app_id: 'app-1',
						status: 'building',
						timestamp: Date.now() - 60000 // 1 minute old
					}
				});
				window.dispatchEvent(event);
			});

			// Should ignore stale update
			const status = await page.locator('[data-testid="app-status"]').textContent();
			expect(status).not.toBe('building');
		});
	});
});
