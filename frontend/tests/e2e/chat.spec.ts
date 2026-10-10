/**
 * Chat E2E Tests
 *
 * Tests for chat functionality, streaming responses, and memory context.
 */

import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation, waitForElement } from './fixtures/helpers';
import { mockGroqApi } from './fixtures/mockApis';
import { testChatMessages } from './fixtures/testData';

test.describe('Chat', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);
		await mockGroqApi(page);

		const user = getTestUser('regularUser');
		await login(page, user);
	});

	test('send message and receive response', async ({ page }) => {
		await page.goto('/chat');

		// Type message
		await page.fill('textarea[name="message"]', testChatMessages.simple);

		// Send message
		await page.click('button[aria-label="Send"]');

		// Should show user message
		await expect(page.locator('[data-testid="user-message"]').last()).toContainText(testChatMessages.simple);

		// Should show assistant response
		await expect(page.locator('[data-testid="assistant-message"]').last()).toBeVisible();
	});

	test('streaming response displays progressively', async ({ page }) => {
		// Mock streaming endpoint
		await page.route('**/api/chat/stream', async (route) => {
			const stream = `data: {"type":"chunk","content":"Hello "}\n\ndata: {"type":"chunk","content":"there!"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Hello');
		await page.click('button[aria-label="Send"]');

		// Wait for streaming to start
		await page.waitForTimeout(500);

		// Should show streaming indicator
		await expect(page.locator('[data-testid="streaming-indicator"]')).toBeVisible();

		// Wait for stream to complete
		await page.waitForTimeout(2000);

		// Should show complete response
		await expect(page.locator('[data-testid="assistant-message"]').last()).toContainText('Hello there!');
	});

	test('conversation history persists', async ({ page }) => {
		await page.goto('/chat');

		// Send first message
		await page.fill('textarea[name="message"]', 'Message 1');
		await page.click('button[aria-label="Send"]');

		await page.waitForTimeout(1000);

		// Send second message
		await page.fill('textarea[name="message"]', 'Message 2');
		await page.click('button[aria-label="Send"]');

		// Should show both messages
		await expect(page.locator('[data-testid="user-message"]')).toHaveCount(2);

		// Reload page
		await page.reload();

		// Should still show conversation history
		await expect(page.locator('[data-testid="user-message"]')).toHaveCount(2);
	});

	test('memory context injection', async ({ page }) => {
		// Mock memory endpoint
		await page.route('**/api/memories/search', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					memories: [
						{
							id: 'mem-1',
							content: 'User prefers dark mode',
							relevance: 0.9
						}
					]
				})
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'What are my preferences?');
		await page.click('button[aria-label="Send"]');

		// Should call memory API
		await page.waitForResponse('**/api/memories/search');

		// Should inject memory context
		await expect(page.locator('text=/dark mode/i')).toBeVisible();
	});

	test('new conversation button', async ({ page }) => {
		await page.goto('/chat');

		// Send message
		await page.fill('textarea[name="message"]', 'Test message');
		await page.click('button[aria-label="Send"]');

		await page.waitForTimeout(1000);

		// Click new conversation
		await page.click('button:has-text("New Conversation")');

		// Should clear conversation
		await expect(page.locator('[data-testid="user-message"]')).toHaveCount(0);

		// Should focus input
		await expect(page.locator('textarea[name="message"]')).toBeFocused();
	});

	test('code blocks in responses', async ({ page }) => {
		// Mock response with code
		await page.route('**/api/chat/stream', async (route) => {
			const stream = `data: {"type":"chunk","content":"Here is code:\\n\\n\`\`\`javascript\\nconst x = 1;\\n\`\`\`"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Show me code');
		await page.click('button[aria-label="Send"]');

		await page.waitForTimeout(2000);

		// Should render code block
		await expect(page.locator('[data-testid="code-block"]')).toBeVisible();

		// Should have copy button
		await expect(page.locator('button:has-text("Copy")')).toBeVisible();
	});

	test('copy code block', async ({ page }) => {
		// Mock response with code
		await page.route('**/api/chat/stream', async (route) => {
			const stream = `data: {"type":"chunk","content":"\`\`\`javascript\\nconst x = 1;\\n\`\`\`"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Code please');
		await page.click('button[aria-label="Send"]');

		await page.waitForTimeout(2000);

		// Click copy button
		await page.click('[data-testid="code-block"] button:has-text("Copy")');

		// Should show copied feedback
		await expect(page.locator('text=Copied')).toBeVisible();
	});

	test('markdown rendering in responses', async ({ page }) => {
		// Mock response with markdown
		await page.route('**/api/chat/stream', async (route) => {
			const stream = `data: {"type":"chunk","content":"# Heading\\n\\n**Bold text**\\n\\n- List item"}\n\ndata: {"type":"end"}\n\n`;

			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: stream
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Format text');
		await page.click('button[aria-label="Send"]');

		await page.waitForTimeout(2000);

		// Should render heading
		await expect(page.locator('[data-testid="assistant-message"] h1')).toContainText('Heading');

		// Should render bold
		await expect(page.locator('[data-testid="assistant-message"] strong')).toContainText('Bold text');

		// Should render list
		await expect(page.locator('[data-testid="assistant-message"] li')).toContainText('List item');
	});

	test('error handling in chat', async ({ page }) => {
		// Mock error response
		await page.route('**/api/chat/stream', async (route) => {
			await route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({
					error: 'Chat service unavailable'
				})
			});
		});

		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Test error');
		await page.click('button[aria-label="Send"]');

		// Should show error message
		await expect(page.locator('text=/error|unavailable/i')).toBeVisible();

		// Should enable retry
		await expect(page.locator('button:has-text("Retry")')).toBeVisible();
	});

	test('typing indicator while sending', async ({ page }) => {
		await page.goto('/chat');

		await page.fill('textarea[name="message"]', 'Test');
		await page.click('button[aria-label="Send"]');

		// Should show typing indicator
		await expect(page.locator('[data-testid="typing-indicator"]')).toBeVisible();

		// Wait for response
		await page.waitForTimeout(2000);

		// Typing indicator should disappear
		await expect(page.locator('[data-testid="typing-indicator"]')).not.toBeVisible();
	});

	test('message input validation', async ({ page }) => {
		await page.goto('/chat');

		// Try to send empty message
		await page.click('button[aria-label="Send"]');

		// Should not send (button should be disabled or message should not appear)
		await expect(page.locator('[data-testid="user-message"]')).toHaveCount(0);

		// Send button should be disabled for empty input
		const sendButton = page.locator('button[aria-label="Send"]');
		await expect(sendButton).toBeDisabled();
	});
});
