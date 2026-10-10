/**
 * Templates E2E Tests
 *
 * Tests for browsing template gallery and using templates.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForApiCall } from './fixtures/helpers';
import { testTemplates } from './fixtures/testData';

test.describe('Templates', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);

		// Mock templates API
		await page.route('**/api/templates', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					templates: testTemplates
				})
			});
		});

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test('browse template gallery', async ({ page }) => {
		await page.goto('/templates');

		// Should show template cards
		await expect(page.locator('[data-testid="template-card"]')).toHaveCount(testTemplates.length);

		// Should display template names
		for (const template of testTemplates) {
			await expect(page.locator(`text=${template.name}`)).toBeVisible();
		}
	});

	test('view template details', async ({ page }) => {
		await page.goto('/templates');

		// Click on first template
		await page.click('[data-testid="template-card"]:first-child');

		// Should navigate to template details
		await page.waitForURL(/\/templates\/[a-z0-9-]+$/);

		// Should show template information
		await expect(page.locator('[data-testid="template-name"]')).toBeVisible();
		await expect(page.locator('[data-testid="template-description"]')).toBeVisible();
		await expect(page.locator('[data-testid="template-category"]')).toBeVisible();

		// Should show "Use Template" button
		await expect(page.locator('button:has-text("Use Template")')).toBeVisible();
	});

	test('use template to generate app', async ({ page }) => {
		// Mock app generation endpoint
		await page.route('**/api/osa/generate/from-template', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					app_id: 'generated-app-123',
					message: 'App generation started from template'
				})
			});
		});

		await page.goto('/templates/crm-template-id');

		// Click use template
		await page.click('button:has-text("Use Template")');

		// Should show customization dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/customize/i')).toBeVisible();

		// Fill customization options
		await page.fill('input[name="appName"]', 'My Custom CRM');
		await page.fill('textarea[name="requirements"]', 'Add contact management');

		// Confirm
		await page.click('button:has-text("Generate App")');

		// Wait for generation API call
		const response = await waitForApiCall(page, '/api/osa/generate/from-template');
		expect(response.status()).toBe(200);

		// Should show success message
		await expect(page.locator('text=/generation.*started/i')).toBeVisible();

		// Should redirect to generated apps
		await page.waitForURL('**/generated-apps');
	});

	test('filter templates by category', async ({ page }) => {
		await page.goto('/templates');

		// Click category filter
		await page.click('[data-testid="category-filter"]');
		await page.click('text=Business');

		// Should show only business templates
		const visibleCards = page.locator('[data-testid="template-card"]');
		const cardCount = await visibleCards.count();
		expect(cardCount).toBeGreaterThan(0);

		// All visible templates should be business category
		const categoryBadges = page.locator('[data-testid="template-category"]:visible');
		const count = await categoryBadges.count();
		for (let i = 0; i < count; i++) {
			await expect(categoryBadges.nth(i)).toContainText('business');
		}
	});

	test('search templates', async ({ page }) => {
		await page.goto('/templates');

		// Type in search box
		await page.fill('input[placeholder*="Search templates"]', 'CRM');

		// Should filter templates
		await page.waitForTimeout(500);
		await expect(page.locator('text=CRM Template')).toBeVisible();

		// Other templates should not be visible
		const visibleCards = page.locator('[data-testid="template-card"]:visible');
		await expect(visibleCards).toHaveCount(1);
	});

	test('template recommendations based on profile', async ({ page }) => {
		// Mock profile endpoint
		await page.route('**/api/profile', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					profile: {
						industry: 'e-commerce',
						interests: ['sales', 'marketing']
					}
				})
			});
		});

		// Mock recommendations endpoint
		await page.route('**/api/templates/recommendations', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					templates: [testTemplates[2]] // E-commerce template
				})
			});
		});

		await page.goto('/templates');

		// Should show recommended section
		await expect(page.locator('[data-testid="recommended-templates"]')).toBeVisible();

		// Should show e-commerce template as recommended
		await expect(page.locator('[data-testid="recommended-templates"] text=E-commerce')).toBeVisible();
	});

	test('template preview', async ({ page }) => {
		await page.goto('/templates/crm-template-id');

		// Click preview button
		await page.click('button:has-text("Preview")');

		// Should show preview dialog/modal
		await expect(page.locator('[data-testid="template-preview"]')).toBeVisible();

		// Should show screenshots or demo
		await expect(page.locator('[data-testid="preview-image"]')).toBeVisible();

		// Should show features list
		await expect(page.locator('[data-testid="features-list"]')).toBeVisible();
	});

	test('save template to favorites', async ({ page }) => {
		// Mock favorites endpoint
		await page.route('**/api/templates/*/favorite', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ success: true })
			});
		});

		await page.goto('/templates/crm-template-id');

		// Click favorite button
		await page.click('button[aria-label="Add to favorites"]');

		// Should update UI
		await expect(page.locator('button[aria-label="Remove from favorites"]')).toBeVisible();

		// Should call favorites API
		await page.waitForResponse('**/api/templates/*/favorite');
	});

	test('view template source code', async ({ page }) => {
		// Mock source code endpoint
		await page.route('**/api/templates/*/source', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					files: [
						{
							path: 'src/index.ts',
							content: 'console.log("Hello World");'
						}
					]
				})
			});
		});

		await page.goto('/templates/crm-template-id');

		// Click view source
		await page.click('button:has-text("View Source")');

		// Should show source code viewer
		await expect(page.locator('[data-testid="source-viewer"]')).toBeVisible();

		// Should show code content
		await expect(page.locator('text=console.log')).toBeVisible();
	});

	test('template rating and reviews', async ({ page }) => {
		await page.goto('/templates/crm-template-id');

		// Should show rating
		await expect(page.locator('[data-testid="template-rating"]')).toBeVisible();

		// Should show reviews section
		await expect(page.locator('[data-testid="reviews-section"]')).toBeVisible();

		// Click add review
		await page.click('button:has-text("Write Review")');

		// Should show review form
		await expect(page.locator('[data-testid="review-form"]')).toBeVisible();

		// Submit review
		await page.fill('textarea[name="review"]', 'Great template!');
		await page.click('[data-testid="star-rating"] button:nth-child(5)');
		await page.click('button:has-text("Submit Review")');

		// Should show success message
		await expect(page.locator('text=/review.*submitted/i')).toBeVisible();
	});

	test('sort templates', async ({ page }) => {
		await page.goto('/templates');

		// Click sort dropdown
		await page.click('[data-testid="sort-dropdown"]');

		// Sort by popularity
		await page.click('text=Most Popular');

		// Templates should be reordered
		await page.waitForTimeout(500);

		// Verify sort was applied (check URL or state)
		await expect(page.url()).toContain('sort=popular');
	});
});
