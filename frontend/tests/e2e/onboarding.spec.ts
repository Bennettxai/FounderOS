/**
 * Onboarding E2E Tests
 *
 * Tests for the complete onboarding flow from signup to completion.
 */

import { test, expect } from '@playwright/test';
import { createUniqueTestUser } from './fixtures/testUsers';
import { setupTestIsolation, waitForElement, waitForText } from './fixtures/helpers';
import { mockAuthApi, mockGmailApi } from './fixtures/mockApis';

test.describe('Onboarding Flow', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);
		await mockAuthApi(page);
		await mockGmailApi(page);
	});

	test('complete onboarding flow', async ({ page }) => {
		const user = createUniqueTestUser('onboarding');

		// STEP 1: Signin/Signup
		await page.goto('/onboarding/signin');

		await page.fill('input[name="email"]', user.email);
		await page.fill('input[name="password"]', user.password);
		await page.click('button[type="submit"]');

		// STEP 2: Username
		await page.waitForURL('**/onboarding/username', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/username/);

		await page.fill('input[name="name"]', user.name);
		await page.click('button:has-text("Continue")');

		// STEP 3: Gmail Connection
		await page.waitForURL('**/onboarding/gmail', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/gmail/);

		// Click connect Gmail
		await page.click('button:has-text("Connect Gmail")');

		// Should show connecting state
		await expect(page.locator('text=/connecting/i')).toBeVisible();

		// Mock successful Gmail connection
		await page.evaluate(() => {
			window.dispatchEvent(new CustomEvent('gmail-connected', {
				detail: { success: true }
			}));
		});

		// STEP 4: Analyzing Emails
		await page.waitForURL('**/onboarding/analyzing', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/analyzing/);

		// Should show analyzing animation
		await expect(page.locator('[data-testid="analyzing-animation"]')).toBeVisible();

		// Wait for analysis to complete (mocked)
		await page.waitForTimeout(2000);

		// Trigger analysis complete
		await page.evaluate(() => {
			window.dispatchEvent(new CustomEvent('analysis-complete', {
				detail: {
					insights: {
						total_emails: 150,
						categories: { work: 80, personal: 50, marketing: 20 }
					}
				}
			}));
		});

		// STEP 5: Meet OSA
		await page.waitForURL('**/onboarding/meet-osa', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/meet-osa/);

		// Should show OSA introduction
		await expect(page.locator('text=/meet.*osa/i')).toBeVisible();

		await page.click('button:has-text("Continue")');

		// STEP 6: Starter Apps
		await page.waitForURL('**/onboarding/starter-apps', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/starter-apps/);

		// Should show app suggestions
		await expect(page.locator('[data-testid="app-suggestion"]')).toHaveCount(3);

		// Select an app
		await page.click('[data-testid="app-suggestion"]:first-child');

		await page.click('button:has-text("Continue")');

		// STEP 7: Ready
		await page.waitForURL('**/onboarding/ready', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/ready/);

		// Should show completion message
		await expect(page.locator('text=/ready/i')).toBeVisible();

		// Click to enter app
		await page.click('button:has-text("Get Started")');

		// STEP 8: Redirect to Dashboard
		await page.waitForURL('**/dashboard', { timeout: 10000 });
		await expect(page).toHaveURL(/.*dashboard/);

		// Should show welcome message
		await expect(page.locator('text=/welcome/i')).toBeVisible();
	});

	test('skip Gmail connection during onboarding', async ({ page }) => {
		const user = createUniqueTestUser('skip-gmail');

		// Start onboarding
		await page.goto('/onboarding/signin');
		await page.fill('input[name="email"]', user.email);
		await page.fill('input[name="password"]', user.password);
		await page.click('button[type="submit"]');

		// Username
		await page.waitForURL('**/onboarding/username');
		await page.fill('input[name="name"]', user.name);
		await page.click('button:has-text("Continue")');

		// Gmail - skip
		await page.waitForURL('**/onboarding/gmail');
		await page.click('button:has-text("Skip")');

		// Should proceed to next step
		await page.waitForURL('**/onboarding/meet-osa', { timeout: 10000 });
		await expect(page).toHaveURL(/.*onboarding\/meet-osa/);
	});

	test('back navigation during onboarding', async ({ page }) => {
		const user = createUniqueTestUser('back-nav');

		// Start onboarding and get to username step
		await page.goto('/onboarding/signin');
		await page.fill('input[name="email"]', user.email);
		await page.fill('input[name="password"]', user.password);
		await page.click('button[type="submit"]');

		await page.waitForURL('**/onboarding/username');

		// Click back
		await page.click('button[aria-label="Back"]');

		// Should go back to signin
		await page.waitForURL('**/onboarding/signin', { timeout: 5000 });
		await expect(page).toHaveURL(/.*onboarding\/signin/);
	});

	test('email analysis shows insights', async ({ page }) => {
		const user = createUniqueTestUser('analysis');

		// Get to analyzing step
		await page.goto('/onboarding/signin');
		await page.fill('input[name="email"]', user.email);
		await page.fill('input[name="password"]', user.password);
		await page.click('button[type="submit"]');

		await page.waitForURL('**/onboarding/username');
		await page.fill('input[name="name"]', user.name);
		await page.click('button:has-text("Continue")');

		await page.waitForURL('**/onboarding/gmail');
		await page.click('button:has-text("Connect Gmail")');

		// Wait for analyzing page
		await page.waitForURL('**/onboarding/analyzing');

		// Trigger analysis complete with insights
		await page.evaluate(() => {
			window.dispatchEvent(new CustomEvent('analysis-complete', {
				detail: {
					insights: {
						total_emails: 150,
						categories: {
							work: 80,
							personal: 50,
							marketing: 20
						},
						top_contacts: [
							{ email: 'john@example.com', count: 25 },
							{ email: 'jane@example.com', count: 20 }
						]
					}
				}
			}));
		});

		// Should show insights
		await expect(page.locator('text=/150.*emails/i')).toBeVisible();
		await expect(page.locator('text=/work.*80/i')).toBeVisible();
	});

	test('starter apps selection', async ({ page }) => {
		const user = createUniqueTestUser('starter-apps');

		// Navigate directly to starter-apps step (assuming logged in)
		await page.goto('/onboarding/starter-apps');

		// Should show multiple app suggestions
		const appCards = page.locator('[data-testid="app-suggestion"]');
		await expect(appCards).toHaveCount(3);

		// Select multiple apps
		await appCards.nth(0).click();
		await appCards.nth(1).click();

		// Should highlight selected apps
		await expect(appCards.nth(0)).toHaveClass(/selected/);
		await expect(appCards.nth(1)).toHaveClass(/selected/);

		// Continue
		await page.click('button:has-text("Continue")');

		// Should proceed to ready step
		await page.waitForURL('**/onboarding/ready');
	});

	test('profile creation during onboarding', async ({ page }) => {
		const user = createUniqueTestUser('profile');

		// Mock profile creation endpoint
		await page.route('**/api/profile', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					profile: {
						id: 'profile-123',
						name: user.name,
						email: user.email
					}
				})
			});
		});

		await page.goto('/onboarding/username');

		await page.fill('input[name="name"]', user.name);
		await page.fill('textarea[name="bio"]', 'Test bio for onboarding');

		await page.click('button:has-text("Continue")');

		// Wait for profile creation API call
		const response = await page.waitForResponse('**/api/profile');
		expect(response.status()).toBe(200);
	});

	test('workspace initialization during onboarding', async ({ page }) => {
		const user = createUniqueTestUser('workspace');

		// Mock workspace creation
		await page.route('**/api/workspaces', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					workspace: {
						id: 'workspace-123',
						name: 'My Workspace'
					}
				})
			});
		});

		// Complete onboarding flow
		await page.goto('/onboarding/ready');

		await page.click('button:has-text("Get Started")');

		// Should create workspace
		const response = await page.waitForResponse('**/api/workspaces');
		expect(response.status()).toBe(200);

		// Should redirect to dashboard with workspace
		await page.waitForURL('**/dashboard');
		await expect(page.locator('[data-testid="workspace-name"]')).toContainText('My Workspace');
	});
});
