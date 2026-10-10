/**
 * E2E Test Helpers
 *
 * Common functions used across E2E tests.
 */

import { type Page, expect } from "@playwright/test";
import type { TestUser } from "./testUsers";

/**
 * Login helper function
 */
export async function login(page: Page, user: TestUser) {
  // Navigate to login page
  await page.goto("/login");

  // Fill in credentials
  await page.fill('input[name="email"]', user.email);
  await page.fill('input[name="password"]', user.password);

  // Submit form
  await page.click('button[type="submit"]');

  // Wait for navigation to complete
  await page.waitForURL("**/dashboard", { timeout: 10000 });

  // Verify we're logged in
  await expect(page).toHaveURL(/.*dashboard/);
}

/**
 * Logout helper function
 */
export async function logout(page: Page) {
  // Click user menu
  await page.click('[data-testid="user-menu"]');

  // Click logout
  await page.click('[data-testid="logout-button"]');

  // Wait for redirect to login
  await page.waitForURL("**/login", { timeout: 5000 });
}

/**
 * Wait for an element to be visible
 */
export async function waitForElement(
  page: Page,
  selector: string,
  timeout = 5000,
) {
  await page.waitForSelector(selector, { state: "visible", timeout });
}

/**
 * Wait for text to appear
 */
export async function waitForText(page: Page, text: string, timeout = 5000) {
  await page.waitForSelector(`text=${text}`, { timeout });
}

/**
 * Navigate to a specific page
 */
export async function navigateTo(page: Page, path: string) {
  await page.goto(path);
  await page.waitForLoadState("networkidle");
}

/**
 * Fill form and submit
 */
export async function submitForm(
  page: Page,
  formData: Record<string, string>,
  submitSelector = 'button[type="submit"]',
) {
  // Fill all form fields
  for (const [name, value] of Object.entries(formData)) {
    await page.fill(`input[name="${name}"], textarea[name="${name}"]`, value);
  }

  // Submit form
  await page.click(submitSelector);
}

/**
 * Wait for API call to complete
 */
export async function waitForApiCall(
  page: Page,
  urlPattern: string | RegExp,
  timeout = 10000,
) {
  return await page.waitForResponse(
    (response) => {
      const url = response.url();
      if (typeof urlPattern === "string") {
        return url.includes(urlPattern);
      }
      return urlPattern.test(url);
    },
    { timeout },
  );
}

/**
 * Wait for SSE event
 */
export async function waitForSSEEvent(
  page: Page,
  eventType: string,
  timeout = 10000,
) {
  return await page.evaluate((type) => {
    return new Promise((resolve) => {
      const eventSource = new EventSource("/api/osa/stream");
      const timer = setTimeout(() => {
        eventSource.close();
        resolve(null);
      }, timeout);

      eventSource.addEventListener(type, (event) => {
        clearTimeout(timer);
        eventSource.close();
        resolve(JSON.parse(event.data));
      });
    });
  }, eventType);
}

/**
 * Take a screenshot with a custom name
 */
export async function takeScreenshot(page: Page, name: string) {
  await page.screenshot({
    path: `test-results/screenshots/${name}.png`,
    fullPage: true,
  });
}

/**
 * Check if element is visible
 */
export async function isVisible(
  page: Page,
  selector: string,
): Promise<boolean> {
  try {
    await page.waitForSelector(selector, { state: "visible", timeout: 1000 });
    return true;
  } catch {
    return false;
  }
}

/**
 * Click and wait for navigation
 */
export async function clickAndNavigate(
  page: Page,
  selector: string,
  expectedUrl?: string | RegExp,
) {
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle" }),
    page.click(selector),
  ]);

  if (expectedUrl) {
    await expect(page).toHaveURL(expectedUrl);
  }
}

/**
 * Wait for loading spinner to disappear
 */
export async function waitForLoadingToFinish(page: Page) {
  await page
    .waitForSelector('[data-testid="loading-spinner"]', {
      state: "hidden",
      timeout: 10000,
    })
    .catch(() => {
      // Ignore if spinner doesn't exist
    });
}

/**
 * Check for error messages
 */
export async function checkForErrors(page: Page): Promise<boolean> {
  const errorSelectors = [
    '[data-testid="error-message"]',
    ".error",
    '[role="alert"]',
  ];

  for (const selector of errorSelectors) {
    if (await isVisible(page, selector)) {
      return true;
    }
  }

  return false;
}

/**
 * Clear local storage and cookies
 */
export async function clearBrowserData(page: Page) {
  // Navigate to base URL first to avoid SecurityError on about:blank
  try {
    // Only clear if we're on a valid page (not about:blank)
    const url = page.url();
    if (url === "about:blank" || url === "") {
      // Navigate to root first
      await page.goto("/");
    }

    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
  } catch (error) {
    // If clearing fails (e.g., on about:blank), just navigate to root
    await page.goto("/");
  }
  await page.context().clearCookies();
}

/**
 * Setup test isolation (clear data before test)
 */
export async function setupTestIsolation(page: Page) {
  await clearBrowserData(page);
}
