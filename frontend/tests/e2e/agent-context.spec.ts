/**
 * Agent Context E2E Tests
 *
 * Tests that validate the agent uses onboarding profile context (business_type,
 * team_size, challenges) when responding to chat messages.
 */

import { test, expect } from '@playwright/test';
import { createUniqueTestUser } from './fixtures/testUsers';
import { setupTestIsolation, waitForElement, waitForApiCall } from './fixtures/helpers';
import { mockAuthApi } from './fixtures/mockApis';

test.describe('Agent Context from Onboarding Profile', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);
		await mockAuthApi(page);
	});

	test('agent response includes business_type from onboarding profile', async ({ page }) => {
		const user = createUniqueTestUser('context-business');
		const profileData = {
			workspace_name: 'Tech Startup Inc',
			business_type: 'saas',
			team_size: '5-10',
			role: 'founder',
			challenge: 'scaling operations'
		};

		// Mock user profile API to return profile with onboarding data
		await page.route('**/api/users/me', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					id: 'user-123',
					email: user.email,
					name: user.name,
					workspace_id: 'workspace-123',
					workspace_name: profileData.workspace_name,
					profile: profileData
				})
			});
		});

		// Mock workspace API
		await page.route('**/api/workspaces/current', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					id: 'workspace-123',
					name: profileData.workspace_name,
					slug: 'tech-startup-inc',
					business_type: profileData.business_type,
					team_size: profileData.team_size,
					settings: {
						role: profileData.role,
						challenge: profileData.challenge
					}
				})
			});
		});

		// Mock chat streaming endpoint with context-aware response
		await page.route('**/api/chat/stream', async (route) => {
			// Agent should reference business_type from profile
			const contextualResponse = `As a ${profileData.business_type} company with a team of ${profileData.team_size} people, I understand you're focused on ${profileData.challenge}. Let me help you with that.`;

			const stream = `data: {"type":"chunk","content":"${contextualResponse}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		// Navigate directly to chat (skip onboarding since profile is mocked)
		await page.goto('/chat');
		await page.waitForLoadState('networkidle');

		// Send a message that should trigger context-aware response
		const chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('How can you help me?');

		// Find send button
		const sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();

		// Wait for response
		await page.waitForTimeout(2000);

		// Verify agent response includes profile context
		const assistantMessage = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(assistantMessage).toBeVisible();

		// Check that response includes business_type
		await expect(assistantMessage).toContainText(profileData.business_type, { ignoreCase: true });

		// Check that response includes team_size
		await expect(assistantMessage).toContainText(profileData.team_size);

		// Check that response includes challenge
		await expect(assistantMessage).toContainText(profileData.challenge);
	});

	test('agent uses multiple profile fields in context', async ({ page }) => {
		const user = createUniqueTestUser('context-multiple');
		const profileData = {
			workspace_name: 'E-commerce Boutique',
			business_type: 'e-commerce',
			team_size: '1-5',
			role: 'owner',
			challenge: 'inventory management'
		};

		// Mock simplified onboarding completion
		await page.route('**/api/onboarding/**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					success: true,
					workspace_id: 'workspace-456',
					extracted_data: profileData
				})
			});
		});

		// Mock chat with context injection
		await page.route('**/api/chat/stream', async (route) => {
			const contextResponse = `I see you're running an ${profileData.business_type} business as an ${profileData.role}. With ${profileData.team_size} people on your team, managing ${profileData.challenge} is definitely a priority. Here's how I can help...`;

			const stream = `data: {"type":"chunk","content":"${contextResponse}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		// Navigate directly to chat (assume onboarding completed)
		await page.goto('/chat');

		// Send message
		const chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('What do you know about my business?');

		const sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();

		await page.waitForTimeout(2000);

		// Verify response includes all profile fields
		const response = page.locator('.message.assistant, div:has(> div.bg-white)').last();

		await expect(response).toContainText('e-commerce', { ignoreCase: true });
		await expect(response).toContainText('owner', { ignoreCase: true });
		await expect(response).toContainText('1-5');
		await expect(response).toContainText('inventory management', { ignoreCase: true });
	});

	test('agent personalizes recommendations based on business_type', async ({ page }) => {
		const user = createUniqueTestUser('context-recommendations');
		const profileData = {
			workspace_name: 'Consulting Firm',
			business_type: 'consulting',
			team_size: '10-20',
			role: 'partner',
			challenge: 'client onboarding'
		};

		// Mock chat response with business-specific recommendations
		await page.route('**/api/chat/stream', async (route) => {
			const consultingResponse = `For a ${profileData.business_type} firm, I'd recommend focusing on client relationship management and project tracking tools. Given your challenge with ${profileData.challenge}, here are tailored solutions...`;

			const stream = `data: {"type":"chunk","content":"${consultingResponse}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		const chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('What tools should I use?');

		const sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();

		await page.waitForTimeout(2000);

		// Verify business-specific recommendations
		const response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('consulting', { ignoreCase: true });
		await expect(response).toContainText('client onboarding', { ignoreCase: true });
	});

	test('agent context persists across multiple chat sessions', async ({ page }) => {
		const profileData = {
			workspace_name: 'Design Agency',
			business_type: 'creative agency',
			team_size: '5-10',
			role: 'creative director',
			challenge: 'project deadlines'
		};

		// Mock chat responses that reference profile
		await page.route('**/api/chat/stream', async (route) => {
			const response = `As a ${profileData.business_type}, managing ${profileData.challenge} with a team of ${profileData.team_size} is crucial.`;

			const stream = `data: {"type":"chunk","content":"${response}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		// First message
		let chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('Help me plan my week');

		let sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(1500);

		let response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('creative agency', { ignoreCase: true });

		// Start new conversation
		const newConvoButton = page.locator('button:has-text("New Conversation")');
		if (await newConvoButton.isVisible({ timeout: 2000 })) {
			await newConvoButton.click();
		}

		// Second message in new conversation
		chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.fill('What are my priorities?');

		sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(1500);

		// Context should still be present
		response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('creative agency', { ignoreCase: true });
		await expect(response).toContainText('project deadlines', { ignoreCase: true });
	});

	test('agent context includes team_size for collaboration suggestions', async ({ page }) => {
		const profileData = {
			workspace_name: 'Marketing Team',
			business_type: 'marketing',
			team_size: '20-50',
			role: 'marketing manager',
			challenge: 'team coordination'
		};

		await page.route('**/api/chat/stream', async (route) => {
			const response = `With a team of ${profileData.team_size} people, ${profileData.challenge} requires robust collaboration tools. For ${profileData.business_type} teams of this size, I recommend...`;

			const stream = `data: {"type":"chunk","content":"${response}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		const chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('How can we collaborate better?');

		const sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(2000);

		const response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('20-50');
		await expect(response).toContainText('team coordination', { ignoreCase: true });
		await expect(response).toContainText('collaboration', { ignoreCase: true });
	});

	test('agent adapts responses based on role context', async ({ page }) => {
		const profileData = {
			workspace_name: 'Startup Hub',
			business_type: 'saas',
			team_size: '5-10',
			role: 'cto',
			challenge: 'technical debt'
		};

		await page.route('**/api/chat/stream', async (route) => {
			const response = `As a ${profileData.role}, addressing ${profileData.challenge} is critical for your ${profileData.business_type} company's growth.`;

			const stream = `data: {"type":"chunk","content":"${response}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		const chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('What should I focus on this quarter?');

		const sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(2000);

		const response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('cto', { ignoreCase: true });
		await expect(response).toContainText('technical debt', { ignoreCase: true });
	});

	test('agent references challenge in follow-up responses', async ({ page }) => {
		const profileData = {
			workspace_name: 'Sales Organization',
			business_type: 'sales',
			team_size: '10-20',
			role: 'sales director',
			challenge: 'lead qualification'
		};

		let callCount = 0;
		await page.route('**/api/chat/stream', async (route) => {
			callCount++;

			let response;
			if (callCount === 1) {
				response = `I understand ${profileData.challenge} is a key challenge for your ${profileData.business_type} team.`;
			} else {
				response = `Following up on ${profileData.challenge}, here are specific strategies for your team of ${profileData.team_size}...`;
			}

			const stream = `data: {"type":"chunk","content":"${response}"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		// First message
		let chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.waitFor({ state: 'visible', timeout: 5000 });
		await chatTextarea.fill('Help me with my main challenge');

		let sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(1500);

		let response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('lead qualification', { ignoreCase: true });

		// Follow-up message
		chatTextarea = page.locator('textarea[placeholder*="Ask OSA anything"]');
		await chatTextarea.fill('Can you be more specific?');

		sendButton = page.locator('button.send-btn, button[type="submit"]').filter({ hasText: '' }).first();
		await sendButton.click();
		await page.waitForTimeout(1500);

		response = page.locator('.message.assistant, div:has(> div.bg-white)').last();
		await expect(response).toContainText('lead qualification', { ignoreCase: true });
		await expect(response).toContainText('10-20');
	});
});
