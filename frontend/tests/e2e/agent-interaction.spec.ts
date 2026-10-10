/**
 * Agent Interaction E2E Tests
 *
 * Tests for creating custom agents, testing in sandbox, and agent delegation.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForElement, waitForApiCall } from './fixtures/helpers';
import { testAgents } from './fixtures/testData';

test.describe('Agent Interaction', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test('create custom agent', async ({ page }) => {
		// Navigate to agents page
		await page.goto('/agents');

		// Click create new agent
		await page.click('button:has-text("Create Agent")');

		// Should navigate to agent creation form
		await page.waitForURL('**/agents/new');

		// Fill agent details
		await page.fill('input[name="name"]', testAgents.custom.name);
		await page.fill('textarea[name="description"]', testAgents.custom.description);
		await page.fill('textarea[name="systemPrompt"]', testAgents.custom.systemPrompt);

		// Set temperature
		await page.fill('input[name="temperature"]', testAgents.custom.temperature.toString());

		// Submit form
		await page.click('button[type="submit"]');

		// Wait for agent creation API call
		const response = await waitForApiCall(page, '/api/agents');
		expect(response.status()).toBe(200);

		// Should redirect to agent details
		await page.waitForURL(/\/agents\/[a-z0-9-]+$/);

		// Should show success message
		await expect(page.locator('text=/agent.*created/i')).toBeVisible();

		// Should display agent name
		await expect(page.locator(`h1:has-text("${testAgents.custom.name}")`)).toBeVisible();
	});

	test('test agent in sandbox', async ({ page }) => {
		// Navigate to agent page (assuming agent exists)
		await page.goto('/agents/test-agent-id');

		// Click test in sandbox button
		await page.click('button:has-text("Test in Sandbox")');

		// Should open sandbox dialog
		await expect(page.locator('[data-testid="sandbox-dialog"]')).toBeVisible();

		// Type test message
		await page.fill('textarea[placeholder*="test message"]', 'Hello agent!');

		// Send message
		await page.click('[data-testid="sandbox-send"]');

		// Should show agent response
		await expect(page.locator('[data-testid="sandbox-response"]')).toBeVisible();
	});

	test('verify context injection in agent', async ({ page }) => {
		// Mock profile endpoint
		await page.route('**/api/profile', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					profile: {
						name: 'Test User',
						role: 'Developer',
						preferences: { theme: 'dark' }
					}
				})
			});
		});

		await page.goto('/agents/test-agent-id');

		// Click test button
		await page.click('button:has-text("Test in Sandbox")');

		// Should inject profile context
		await expect(page.locator('text=/context.*injected/i')).toBeVisible();

		// Send message that references context
		await page.fill('textarea[placeholder*="test message"]', 'What is my role?');
		await page.click('[data-testid="sandbox-send"]');

		// Response should reference injected context
		await expect(page.locator('text=/developer/i')).toBeVisible();
	});

	test('edit existing agent', async ({ page }) => {
		await page.goto('/agents/test-agent-id');

		// Click edit button
		await page.click('button:has-text("Edit")');

		// Should navigate to edit page
		await page.waitForURL('**/agents/test-agent-id/edit');

		// Update description
		await page.fill('textarea[name="description"]', 'Updated description');

		// Save changes
		await page.click('button:has-text("Save")');

		// Wait for update API call
		const response = await page.waitForResponse(response =>
			response.url().includes('/api/agents/test-agent-id') &&
			response.request().method() === 'PUT'
		);
		expect(response.status()).toBe(200);

		// Should show success message
		await expect(page.locator('text=/updated.*successfully/i')).toBeVisible();
	});

	test('delete agent', async ({ page }) => {
		// Mock delete endpoint
		await page.route('**/api/agents/*', async (route) => {
			if (route.request().method() === 'DELETE') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ success: true })
				});
			}
		});

		await page.goto('/agents/test-agent-id');

		// Click delete button
		await page.click('button:has-text("Delete")');

		// Should show confirmation dialog
		await expect(page.locator('[role="dialog"]')).toBeVisible();
		await expect(page.locator('text=/confirm.*delete/i')).toBeVisible();

		// Confirm deletion
		await page.click('button:has-text("Confirm")');

		// Wait for delete API call
		await page.waitForResponse(response =>
			response.url().includes('/api/agents/test-agent-id') &&
			response.request().method() === 'DELETE'
		);

		// Should redirect to agents list
		await page.waitForURL('**/agents');

		// Should show success message
		await expect(page.locator('text=/deleted.*successfully/i')).toBeVisible();
	});

	test('agent list displays all agents', async ({ page }) => {
		// Mock agents list endpoint
		await page.route('**/api/agents', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [
						{
							id: 'agent-1',
							name: 'Agent 1',
							description: 'First agent'
						},
						{
							id: 'agent-2',
							name: 'Agent 2',
							description: 'Second agent'
						}
					]
				})
			});
		});

		await page.goto('/agents');

		// Should show agent cards
		await expect(page.locator('[data-testid="agent-card"]')).toHaveCount(2);
		await expect(page.locator('text=Agent 1')).toBeVisible();
		await expect(page.locator('text=Agent 2')).toBeVisible();
	});

	test('agent presets available', async ({ page }) => {
		await page.goto('/agents/presets');

		// Should show preset agents
		const presetCount = await page.locator('[data-testid="preset-agent"]').count();
		expect(presetCount).toBeGreaterThan(0);

		// Click on a preset
		await page.click('[data-testid="preset-agent"]:first-child');

		// Should show preset details
		await expect(page.locator('[data-testid="preset-details"]')).toBeVisible();

		// Click use preset
		await page.click('button:has-text("Use This Preset")');

		// Should navigate to create page with preset filled
		await page.waitForURL('**/agents/new');

		// Form should be pre-filled
		const nameValue = await page.inputValue('input[name="name"]');
		expect(nameValue).not.toBe('');
	});

	test('skill execution in agent', async ({ page }) => {
		// Mock skill execution endpoint
		await page.route('**/api/agents/*/skills/execute', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					result: 'Skill executed successfully'
				})
			});
		});

		await page.goto('/agents/test-agent-id');

		// Open sandbox
		await page.click('button:has-text("Test in Sandbox")');

		// Send message that triggers skill
		await page.fill('textarea[placeholder*="test message"]', 'Search the web for cats');
		await page.click('[data-testid="sandbox-send"]');

		// Should execute skill
		await page.waitForResponse('**/api/agents/*/skills/execute');

		// Should show skill execution result
		await expect(page.locator('text=/skill.*executed/i')).toBeVisible();
	});

	test('agent delegation', async ({ page }) => {
		await page.goto('/chat');

		// Send message that should delegate to specialist agent
		await page.fill('textarea[name="message"]', 'Create a database schema for users');
		await page.click('button[aria-label="Send"]');

		// Should detect delegation need
		await expect(page.locator('text=/delegating.*to.*database.*agent/i')).toBeVisible();

		// Should show delegated agent response
		await expect(page.locator('[data-testid="delegated-response"]')).toBeVisible();
	});

	test('clone existing agent', async ({ page }) => {
		await page.goto('/agents/test-agent-id');

		// Click clone button
		await page.click('button:has-text("Clone")');

		// Should navigate to create page with cloned data
		await page.waitForURL('**/agents/new');

		// Name should be prefixed with "Copy of"
		const nameValue = await page.inputValue('input[name="name"]');
		expect(nameValue).toContain('Copy of');

		// Other fields should be populated
		const description = await page.inputValue('textarea[name="description"]');
		expect(description).not.toBe('');
	});

	test('agent conversation history', async ({ page }) => {
		await page.goto('/agents/test-agent-id');

		// Click view history
		await page.click('button:has-text("View History")');

		// Should show conversation history
		await expect(page.locator('[data-testid="conversation-history"]')).toBeVisible();

		// Should show past conversations
		const conversationCount = await page.locator('[data-testid="conversation-item"]').count();
		expect(conversationCount).toBeGreaterThan(0);
	});

	test('agent role configuration', async ({ page }) => {
		await page.goto('/agents/new');

		// Select role
		await page.click('[data-testid="role-selector"]');
		await page.click('text=Developer');

		// Should configure agent for developer role
		await expect(page.locator('input[name="name"]')).toHaveValue(/developer/i);

		// Should inject developer-specific context
		const systemPrompt = await page.inputValue('textarea[name="systemPrompt"]');
		expect(systemPrompt).toContain('developer');
	});
});
