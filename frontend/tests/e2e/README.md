# E2E Testing Guide for BusinessOS

This directory contains end-to-end tests for BusinessOS using Playwright.

## Overview

The E2E test suite covers all critical user flows:

- **Authentication** - Login, signup, logout, session management
- **Onboarding** - Complete onboarding flow from signup to dashboard
- **App Generation** - OSA app generation, build monitoring, deployment
- **Chat** - Chat functionality with streaming responses and memory context
- **Agent Interaction** - Custom agents, sandbox testing, skill execution
- **Templates** - Template gallery browsing and app generation from templates
- **App Management** - Listing, viewing, deploying, stopping, and deleting apps

## Directory Structure

```
tests/e2e/
├── fixtures/
│   ├── testUsers.ts      # Test user accounts
│   ├── testData.ts       # Seed data (workspaces, apps, templates)
│   ├── mockApis.ts       # Mock API responses (OSA, Gmail, Groq)
│   └── helpers.ts        # Common test helper functions
├── auth.spec.ts          # Authentication tests
├── onboarding.spec.ts    # Onboarding flow tests
├── app-generation.spec.ts # App generation tests
├── chat.spec.ts          # Chat functionality tests
├── agent-interaction.spec.ts # Agent tests
├── templates.spec.ts     # Template gallery tests
├── app-management.spec.ts # App management tests
└── README.md             # This file
```

## Running Tests

### Prerequisites

1. Install dependencies:
   ```bash
   cd frontend
   pnpm install
   ```

2. Install Playwright browsers:
   ```bash
   pnpm exec playwright install
   ```

3. Ensure backend is running:
   ```bash
   cd ../desktop/backend-go
   go run cmd/server/main.go
   ```

### Run All Tests

```bash
pnpm test:e2e
```

### Run Tests in UI Mode (Interactive)

```bash
pnpm test:e2e:ui
```

### Run Tests in Debug Mode

```bash
pnpm test:e2e:debug
```

### Run Tests in Headed Mode (See Browser)

```bash
pnpm test:e2e:headed
```

### Run Tests for Specific Browser

```bash
pnpm test:e2e:chromium
pnpm test:e2e:firefox
pnpm test:e2e:webkit
```

### Run Specific Test File

```bash
pnpm exec playwright test auth.spec.ts
```

### Run Specific Test

```bash
pnpm exec playwright test auth.spec.ts -g "should login with valid credentials"
```

### View Test Report

```bash
pnpm test:e2e:report
```

## Configuration

Test configuration is in `playwright.config.ts`:

- **Base URL**: `http://localhost:5173`
- **Timeout**: 30 seconds per test
- **Retries**: 2 retries on CI, 0 locally
- **Workers**: 4 parallel workers (2 on CI)
- **Screenshots**: Only on failure
- **Videos**: Only on failure
- **Trace**: On first retry

## Test Data Management

### Test Users

Test users are defined in `fixtures/testUsers.ts`:

- `admin` - Admin user account
- `regularUser` - Standard user account
- `newUser` - Unique user created per test
- `onboardingUser` - User for onboarding tests

### Test Data

Test data (apps, templates, workspaces) is defined in `fixtures/testData.ts`.

### Mock APIs

Mock API responses are configured in `fixtures/mockApis.ts`:

- **OSA API** - App generation, status, deployment
- **Gmail API** - OAuth, email analysis
- **Groq API** - Chat streaming
- **Auth API** - Login, signup, session

## Helper Functions

Common test helpers are in `fixtures/helpers.ts`:

- `login(page, user)` - Login as a user
- `logout(page)` - Logout current user
- `waitForElement(page, selector)` - Wait for element
- `waitForText(page, text)` - Wait for text to appear
- `navigateTo(page, path)` - Navigate to path
- `submitForm(page, formData)` - Fill and submit form
- `waitForApiCall(page, urlPattern)` - Wait for API call
- `setupTestIsolation(page)` - Clear browser data

## Writing New Tests

### Basic Test Structure

```typescript
import { test, expect } from '@playwright/test';
import { getTestUser } from './fixtures/testUsers';
import { login, setupTestIsolation } from './fixtures/helpers';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    await setupTestIsolation(page);
    const user = getTestUser('regularUser');
    await login(page, user);
  });

  test('should do something', async ({ page }) => {
    // Navigate
    await page.goto('/some-page');

    // Interact
    await page.click('button');

    // Assert
    await expect(page.locator('text=Success')).toBeVisible();
  });
});
```

### Best Practices

1. **Use data-testid attributes** for stable selectors
   ```svelte
   <button data-testid="submit-button">Submit</button>
   ```

2. **Always use test isolation** to prevent test pollution
   ```typescript
   test.beforeEach(async ({ page }) => {
     await setupTestIsolation(page);
   });
   ```

3. **Use page object pattern** for complex flows
   ```typescript
   class LoginPage {
     constructor(private page: Page) {}

     async login(email: string, password: string) {
       await this.page.fill('[name="email"]', email);
       await this.page.fill('[name="password"]', password);
       await this.page.click('[type="submit"]');
     }
   }
   ```

4. **Mock external services** to avoid flakiness
   ```typescript
   await page.route('**/api/external', async (route) => {
     await route.fulfill({ status: 200, body: '{"success": true}' });
   });
   ```

5. **Use waitFor methods** instead of arbitrary timeouts
   ```typescript
   // Good
   await page.waitForSelector('[data-testid="result"]');

   // Bad
   await page.waitForTimeout(5000);
   ```

## CI/CD Integration

Tests run automatically on GitHub Actions:

- **Trigger**: Push to `main` or `pedro-dev`, or pull request to `main`
- **Environment**: Ubuntu with PostgreSQL and Redis services
- **Artifacts**: Test reports and videos uploaded on failure
- **Timeout**: 60 minutes

See `.github/workflows/e2e-tests.yml` for full configuration.

## Debugging Failed Tests

### View Test Report

```bash
pnpm test:e2e:report
```

### Run in Debug Mode

```bash
pnpm test:e2e:debug
```

### View Screenshots

Failed test screenshots are saved to `test-results/*/screenshots/`.

### View Videos

Failed test videos are saved to `test-results/*/videos/`.

### View Traces

Traces are captured on first retry. View them at:
```
test-results/*/trace.zip
```

Load trace in Playwright Trace Viewer:
```bash
pnpm exec playwright show-trace test-results/path/to/trace.zip
```

## Common Issues

### Backend Not Running

Error: `ECONNREFUSED 127.0.0.1:8001`

**Solution**: Start the backend server:
```bash
cd desktop/backend-go
go run cmd/server/main.go
```

### Port Already in Use

Error: `Port 5173 is already in use`

**Solution**: Kill the process or change the port in `playwright.config.ts`.

### Flaky Tests

If tests are flaky:
1. Increase timeouts in `playwright.config.ts`
2. Add explicit waits for dynamic content
3. Mock external services
4. Use `page.waitForLoadState('networkidle')`

### Database State Issues

If tests fail due to database state:
1. Use test database isolation
2. Reset database before each test
3. Use transactions that rollback

## Resources

- [Playwright Documentation](https://playwright.dev)
- [Best Practices](https://playwright.dev/docs/best-practices)
- [API Reference](https://playwright.dev/docs/api/class-playwright)
- [Debugging Guide](https://playwright.dev/docs/debug)

## Contributing

When adding new features:

1. Write E2E tests first (TDD)
2. Use `data-testid` attributes for test selectors
3. Mock external services
4. Ensure tests pass in CI before merging
5. Update this README if adding new test patterns
