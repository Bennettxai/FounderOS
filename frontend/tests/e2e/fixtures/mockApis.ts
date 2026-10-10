/**
 * Mock API Responses
 *
 * Mock responses for external services (OSA, Gmail, Groq) used in E2E tests.
 */

import type { Page } from "@playwright/test";

/**
 * Mock OSA API responses
 */
export async function mockOSAApi(page: Page) {
  // Mock app generation endpoint
  await page.route("**/api/osa/generate", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: true,
        app_id: "test-app-123",
        message: "App generation started",
      }),
    });
  });

  // Mock app status endpoint
  await page.route("**/api/osa/deployment/*/status", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "building",
        progress: 50,
        message: "Building application...",
      }),
    });
  });

  // Mock app list endpoint
  await page.route("**/api/osa/apps", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        apps: [
          {
            id: "app-1",
            name: "Test CRM",
            status: "completed",
            created_at: new Date().toISOString(),
          },
        ],
      }),
    });
  });
}

/**
 * Mock Gmail API responses
 */
export async function mockGmailApi(page: Page) {
  // Mock Gmail OAuth callback
  await page.route("**/api/auth/google/callback", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: true,
        message: "Gmail connected successfully",
      }),
    });
  });

  // Mock email analysis endpoint
  await page.route("**/api/onboarding/analyze-emails", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        analyzed: true,
        insights: {
          total_emails: 150,
          categories: {
            work: 80,
            personal: 50,
            marketing: 20,
          },
          top_contacts: [
            { email: "john@example.com", count: 25 },
            { email: "jane@example.com", count: 20 },
          ],
        },
      }),
    });
  });
}

/**
 * Mock Groq LLM API responses
 */
export async function mockGroqApi(page: Page) {
  // Mock chat streaming endpoint
  await page.route("**/api/chat/stream", async (route) => {
    // Simulate SSE stream
    const stream = `data: {"type":"start","message":"Hello!"}\n\ndata: {"type":"chunk","content":"I "}\n\ndata: {"type":"chunk","content":"can "}\n\ndata: {"type":"chunk","content":"help "}\n\ndata: {"type":"chunk","content":"you!"}\n\ndata: {"type":"end"}\n\n`;

    await route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      body: stream,
    });
  });
}

/**
 * Mock authentication endpoints
 */
export async function mockAuthApi(page: Page) {
  // Mock login endpoint
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: true,
        user: {
          id: "test-user-123",
          email: "test@businessos.com",
          name: "Test User",
        },
        token: "mock-jwt-token",
      }),
    });
  });

  // Mock signup endpoint
  await page.route("**/api/auth/signup", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        success: true,
        user: {
          id: "new-user-123",
          email: "newuser@businessos.com",
          name: "New User",
        },
        token: "mock-jwt-token",
      }),
    });
  });

  // Mock session check endpoint
  await page.route("**/api/auth/session", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        authenticated: true,
        user: {
          id: "test-user-123",
          email: "test@businessos.com",
          name: "Test User",
        },
      }),
    });
  });
}

/**
 * Enable all mocks
 */
export async function enableAllMocks(page: Page) {
  await mockOSAApi(page);
  await mockGmailApi(page);
  await mockGroqApi(page);
  await mockAuthApi(page);
}
