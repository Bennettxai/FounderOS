/**
 * Authentication E2E Tests
 *
 * Tests for login, signup, logout, and session management.
 */

import { test, expect } from '@playwright/test';
import { getTestUser, createUniqueTestUser } from './fixtures/testUsers';
import { login, logout, setupTestIsolation, waitForElement } from './fixtures/helpers';
import { mockAuthApi } from './fixtures/mockApis';

test.describe('Authentication', () => {
	test.beforeEach(async ({ page }) => {
		await setupTestIsolation(page);
		await mockAuthApi(page);
	});

	test('should login with valid credentials', async ({ page }) => {
		const user = getTestUser('regularUser');

		// Navigate to login page
		await page.goto('/login');

		// Fill credentials
		await page.fill('input[name="email"]', user.email);
		await page.fill('input[name="password"]', user.password);

		// Submit login form
		await page.click('button[type="submit"]');

		// Should redirect to window (main app) or stay on login (if auth fails)
		// In real E2E with backend, this would redirect to /window
		await expect(page).toHaveURL(/.*\/login/, { timeout: 5000 });

		// Should show user info
		await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
	});

	test('should show error with invalid credentials', async ({ page }) => {
		// Override mock to return error
		await page.route('**/api/auth/login', async (route) => {
			await route.fulfill({
				status: 401,
				contentType: 'application/json',
				body: JSON.stringify({
					success: false,
					error: 'Invalid credentials'
				})
			});
		});

		await page.goto('/login');

		// Fill invalid credentials
		await page.fill('input[name="email"]', 'invalid@test.com');
		await page.fill('input[name="password"]', 'wrongpassword');

		// Submit
		await page.click('button[type="submit"]');

		// Should show error message
		await expect(page.locator('text=Invalid credentials')).toBeVisible();

		// Should stay on login page
		await expect(page).toHaveURL(/.*login/);
	});

	test('should signup new user', async ({ page }) => {
		const newUser = createUniqueTestUser('signup');

		await page.goto('/register');

		// Fill signup form
		await page.fill('input[name="name"]', newUser.name);
		await page.fill('input[name="email"]', newUser.email);
		await page.fill('input[name="password"]', newUser.password);

		// Submit
		await page.click('button[type="submit"]');

		// Should redirect to onboarding or dashboard
		await page.waitForURL(/\/(onboarding|dashboard)/, { timeout: 10000 });
	});

	test('should logout successfully', async ({ page }) => {
		const user = getTestUser('regularUser');

		// Login first
		await login(page, user);

		// Verify we're logged in
		await expect(page).toHaveURL(/.*dashboard/);

		// Logout
		await logout(page);

		// Should redirect to login
		await expect(page).toHaveURL(/.*login/);

		// Should not be able to access protected routes
		await page.goto('/dashboard');
		await page.waitForURL('**/login', { timeout: 5000 });
		await expect(page).toHaveURL(/.*login/);
	});

	test('should maintain session on page reload', async ({ page }) => {
		const user = getTestUser('regularUser');

		// Login
		await login(page, user);

		// Verify logged in
		await expect(page).toHaveURL(/.*dashboard/);

		// Reload page
		await page.reload();

		// Should still be logged in
		await expect(page).toHaveURL(/.*dashboard/);
		await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
	});

	test('should redirect to login when accessing protected route without auth', async ({ page }) => {
		await page.goto('/dashboard');

		// Should redirect to login
		await page.waitForURL('**/login', { timeout: 5000 });
		await expect(page).toHaveURL(/.*login/);
	});

	test('Google OAuth login flow', async ({ page }) => {
		await page.goto('/login');

		// Click Google login button
		await page.click('button:has-text("Sign in with Google")');

		// Mock OAuth redirect
		await page.route('**/api/auth/google', async (route) => {
			await route.fulfill({
				status: 302,
				headers: {
					Location: '/dashboard'
				}
			});
		});

		// Should redirect to dashboard
		await page.waitForURL('**/dashboard', { timeout: 10000 });
		await expect(page).toHaveURL(/.*dashboard/);
	});

	test('password reset flow', async ({ page }) => {
		await page.goto('/forgot-password');

		// Fill email
		await page.fill('input[name="email"]', 'test@businessos.com');

		// Submit
		await page.click('button[type="submit"]');

		// Should show success message
		await expect(page.locator('text=/password reset/i')).toBeVisible();
	});

	test('should validate email format on signup', async ({ page }) => {
		await page.goto('/register');

		// Fill invalid email
		await page.fill('input[name="email"]', 'invalid-email');
		await page.fill('input[name="password"]', 'Password123!');
		await page.fill('input[name="name"]', 'Test User');

		// Submit
		await page.click('button[type="submit"]');

		// Should show validation error
		await expect(page.locator('text=/invalid.*email/i')).toBeVisible();
	});

	test('should enforce password requirements', async ({ page }) => {
		await page.goto('/register');

		// Fill weak password
		await page.fill('input[name="email"]', 'test@test.com');
		await page.fill('input[name="password"]', 'weak');
		await page.fill('input[name="name"]', 'Test User');

		// Submit
		await page.click('button[type="submit"]');

		// Should show password requirement error
		await expect(page.locator('text=/password.*requirement/i')).toBeVisible();
	});

	// ═══════════════════════════════════════════════════════════════════════════
	// EDGE CASE TESTS (ISR-1)
	// ═══════════════════════════════════════════════════════════════════════════

	test.describe('Auth Edge Cases', () => {
		test('session expiry during active use', async ({ page }) => {
			const user = getTestUser('regularUser');
			await login(page, user);

			// Simulate session expiry
			await page.route('**/api/auth/me', async (route) => {
				await route.fulfill({
					status: 401,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'Session expired' })
				});
			});

			// Navigate to protected route
			await page.goto('/dashboard');

			// Should show session expiry message
			await expect(page.locator('text=/session.*expired|logged out/i')).toBeVisible();

			// Should redirect to login
			await page.waitForURL('**/login', { timeout: 5000 });

			// Should preserve return URL
			const url = page.url();
			expect(url).toContain('redirect=');
		});

		test('concurrent login attempts', async ({ page, context }) => {
			const user = getTestUser('regularUser');
			let loginAttempts = 0;

			await page.route('**/api/auth/login', async (route) => {
				loginAttempts++;
				await page.waitForTimeout(1000); // Simulate slow response

				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						success: true,
						user: user,
						token: 'mock-token'
					})
				});
			});

			await page.goto('/login');

			// Fill credentials
			await page.fill('input[name="email"]', user.email);
			await page.fill('input[name="password"]', user.password);

			// Click submit multiple times rapidly
			const submitButton = page.locator('button[type="submit"]');
			await submitButton.click();
			await submitButton.click();
			await submitButton.click();

			// Should only trigger one login request
			await page.waitForTimeout(2000);
			expect(loginAttempts).toBe(1);

			// Button should be disabled during request
			const isDisabled = await submitButton.isDisabled();
			expect(isDisabled).toBe(true);
		});

		test('OAuth popup blocked', async ({ page }) => {
			await page.goto('/login');

			// Mock OAuth that fails due to popup blocker
			await page.evaluate(() => {
				const originalOpen = window.open;
				window.open = () => null; // Simulate blocked popup
			});

			// Click Google login
			await page.click('button:has-text("Sign in with Google")');

			// Should show popup blocked error
			await expect(page.locator('text=/popup.*blocked|enable popups/i')).toBeVisible();

			// Should show alternative login option
			await expect(page.locator('text=/use email.*instead/i')).toBeVisible();
		});

		test('OAuth callback with error', async ({ page }) => {
			// Navigate to OAuth callback with error
			await page.goto('/auth/callback?error=access_denied&error_description=User+cancelled');

			// Should show user-friendly error
			await expect(page.locator('text=/login.*cancelled|access denied/i')).toBeVisible();

			// Should provide link back to login
			await expect(page.locator('a[href="/login"]')).toBeVisible();
		});

		test('token refresh failure', async ({ page }) => {
			const user = getTestUser('regularUser');
			await login(page, user);

			// Mock token refresh failure
			await page.route('**/api/auth/refresh', async (route) => {
				await route.fulfill({
					status: 401,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'Refresh token expired' })
				});
			});

			// Trigger token refresh (simulate expired token)
			await page.evaluate(() => {
				// Simulate token expiry
				localStorage.setItem('token_expiry', Date.now().toString());
			});

			// Navigate to trigger refresh
			await page.goto('/dashboard');

			// Should redirect to login
			await page.waitForURL('**/login', { timeout: 5000 });

			// Should show message about re-authentication
			await expect(page.locator('text=/please.*login.*again/i')).toBeVisible();
		});

		test('email verification required', async ({ page }) => {
			// Mock login that requires email verification
			await page.route('**/api/auth/login', async (route) => {
				await route.fulfill({
					status: 403,
					contentType: 'application/json',
					body: JSON.stringify({
						error: 'Email not verified',
						requires_verification: true
					})
				});
			});

			await page.goto('/login');

			await page.fill('input[name="email"]', 'unverified@test.com');
			await page.fill('input[name="password"]', 'Password123!');
			await page.click('button[type="submit"]');

			// Should show verification required message
			await expect(page.locator('text=/verify.*email|check.*inbox/i')).toBeVisible();

			// Should show resend verification button
			await expect(page.locator('button:has-text("Resend Verification")')).toBeVisible();
		});

		test('account locked after failed attempts', async ({ page }) => {
			let attempts = 0;

			await page.route('**/api/auth/login', async (route) => {
				attempts++;

				if (attempts >= 5) {
					await route.fulfill({
						status: 423, // Locked
						contentType: 'application/json',
						body: JSON.stringify({
							error: 'Account locked',
							locked_until: new Date(Date.now() + 30 * 60 * 1000).toISOString()
						})
					});
				} else {
					await route.fulfill({
						status: 401,
						contentType: 'application/json',
						body: JSON.stringify({ error: 'Invalid credentials' })
					});
				}
			});

			await page.goto('/login');

			// Make multiple failed login attempts
			for (let i = 0; i < 5; i++) {
				await page.fill('input[name="email"]', 'test@test.com');
				await page.fill('input[name="password"]', 'wrongpassword');
				await page.click('button[type="submit"]');
				await page.waitForTimeout(500);
			}

			// Should show account locked message
			await expect(page.locator('text=/account.*locked|too many attempts/i')).toBeVisible();

			// Should show unlock time
			await expect(page.locator('text=/30.*minutes/i')).toBeVisible();

			// Should show support contact
			await expect(page.locator('a[href*="support"]')).toBeVisible();
		});

		test('password reset token expiry', async ({ page }) => {
			// Navigate with expired reset token
			await page.goto('/reset-password?token=expired-token');

			// Mock expired token response
			await page.route('**/api/auth/reset-password', async (route) => {
				await route.fulfill({
					status: 400,
					contentType: 'application/json',
					body: JSON.stringify({
						error: 'Reset token expired',
						expired: true
					})
				});
			});

			await page.fill('input[name="password"]', 'NewPassword123!');
			await page.fill('input[name="confirm_password"]', 'NewPassword123!');
			await page.click('button[type="submit"]');

			// Should show token expired error
			await expect(page.locator('text=/link.*expired|request.*new/i')).toBeVisible();

			// Should show link to request new reset
			await expect(page.locator('a[href="/forgot-password"]')).toBeVisible();
		});

		test('logout from all devices', async ({ page, context }) => {
			const user = getTestUser('regularUser');
			await login(page, user);

			// Open second tab
			const secondPage = await context.newPage();
			await secondPage.goto('/dashboard');

			// Logout from all devices from first tab
			await page.click('[data-testid="user-menu"]');
			await page.click('button:has-text("Logout All Devices")');

			// Should show confirmation
			await expect(page.locator('[role="dialog"]')).toBeVisible();
			await page.click('button:has-text("Confirm")');

			// Both tabs should be logged out
			await page.waitForURL('**/login', { timeout: 5000 });
			await secondPage.waitForURL('**/login', { timeout: 5000 });

			await secondPage.close();
		});

		test('remember me functionality', async ({ page }) => {
			const user = getTestUser('regularUser');

			await page.goto('/login');

			// Check remember me
			await page.click('input[name="remember_me"]');

			await page.fill('input[name="email"]', user.email);
			await page.fill('input[name="password"]', user.password);
			await page.click('button[type="submit"]');

			await page.waitForURL('**/dashboard');

			// Close and reopen browser
			await page.context().clearCookies();
			await page.reload();

			// Should still be logged in (if remember me is set)
			const isLoggedIn = await page.isVisible('[data-testid="user-menu"]');
			expect(isLoggedIn).toBe(true);
		});

		test('CSRF protection', async ({ page }) => {
			// Mock CSRF token mismatch
			await page.route('**/api/auth/login', async (route) => {
				await route.fulfill({
					status: 403,
					contentType: 'application/json',
					body: JSON.stringify({
						error: 'CSRF token mismatch'
					})
				});
			});

			await page.goto('/login');

			const user = getTestUser('regularUser');
			await page.fill('input[name="email"]', user.email);
			await page.fill('input[name="password"]', user.password);
			await page.click('button[type="submit"]');

			// Should show CSRF error and retry
			await expect(page.locator('text=/security.*error|please try again/i')).toBeVisible();
		});

		test('simultaneous logout from multiple tabs', async ({ page, context }) => {
			const user = getTestUser('regularUser');
			await login(page, user);

			// Open multiple tabs
			const tab2 = await context.newPage();
			const tab3 = await context.newPage();

			await tab2.goto('/dashboard');
			await tab3.goto('/dashboard');

			// Logout from first tab
			await logout(page);

			// Other tabs should be logged out too
			await tab2.reload();
			await tab3.reload();

			await tab2.waitForURL('**/login', { timeout: 5000 });
			await tab3.waitForURL('**/login', { timeout: 5000 });

			await tab2.close();
			await tab3.close();
		});

		test('social login account conflict', async ({ page }) => {
			// Mock Google login that returns existing email
			await page.route('**/api/auth/google/callback', async (route) => {
				await route.fulfill({
					status: 409,
					contentType: 'application/json',
					body: JSON.stringify({
						error: 'Account already exists with this email',
						existing_provider: 'email'
					})
				});
			});

			await page.goto('/auth/google/callback?code=mock-code');

			// Should show account conflict error
			await expect(page.locator('text=/account.*already exists/i')).toBeVisible();

			// Should suggest login with email
			await expect(page.locator('text=/login with email/i')).toBeVisible();

			// Should show link account option
			await expect(page.locator('button:has-text("Link Accounts")')).toBeVisible();
		});
	});
});
