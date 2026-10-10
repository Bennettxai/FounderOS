# Sentry Error Tracking Setup

## Overview

This guide covers integrating Sentry for error tracking and monitoring in both the BusinessOS backend (Go) and frontend (SvelteKit).

## Table of Contents

1. [Sentry Project Setup](#sentry-project-setup)
2. [Backend Integration (Go)](#backend-integration-go)
3. [Frontend Integration (SvelteKit)](#frontend-integration-sveltekit)
4. [Configuration](#configuration)
5. [Testing](#testing)
6. [Monitoring & Alerts](#monitoring--alerts)

---

## Sentry Project Setup

### 1. Create Sentry Account

1. Go to [sentry.io](https://sentry.io/)
2. Sign up or log in
3. Create a new organization (or use existing)

### 2. Create Projects

Create two separate projects:

**Backend Project:**
- Platform: **Go**
- Project name: `businessos-backend`
- Copy the DSN (Data Source Name)

**Frontend Project:**
- Platform: **SvelteKit** (or JavaScript)
- Project name: `businessos-frontend`
- Copy the DSN

---

## Backend Integration (Go)

### 1. Install Sentry SDK

```bash
cd desktop/backend-go
go get github.com/getsentry/sentry-go
```

### 2. Create Sentry Middleware

Create file: `desktop/backend-go/internal/middleware/sentry.go`

```go
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// SentryConfig holds Sentry configuration
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64 // 0.0 to 1.0
	TracesSampleRate float64 // 0.0 to 1.0
}

// InitSentry initializes Sentry error tracking
func InitSentry(config SentryConfig) error {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		SampleRate:       config.SampleRate,
		TracesSampleRate: config.TracesSampleRate,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter sensitive data
			if event.Request != nil {
				// Remove sensitive headers
				delete(event.Request.Headers, "Authorization")
				delete(event.Request.Headers, "Cookie")
			}
			return event
		},
	})

	if err != nil {
		return err
	}

	slog.Info("Sentry initialized",
		"environment", config.Environment,
		"release", config.Release,
	)

	return nil
}

// SentryMiddleware captures panics and errors in Gin
func SentryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		hub.Scope().SetContext("request", map[string]interface{}{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"query":  c.Request.URL.RawQuery,
		})

		// Store hub in context for later use
		c.Request = c.Request.WithContext(
			sentry.SetHubOnContext(c.Request.Context(), hub),
		)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(
					context.WithValue(c.Request.Context(), sentry.RequestContextKey, c.Request),
					err,
				)
				hub.Flush(2 * time.Second)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()

		// Capture errors from context
		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				hub.CaptureException(ginErr.Err)
			}
		}

		// Capture 5xx errors
		if c.Writer.Status() >= 500 {
			hub.Scope().SetLevel(sentry.LevelError)
			hub.Scope().SetTag("status_code", string(rune(c.Writer.Status())))
			hub.CaptureMessage("HTTP 5xx Error")
		}
	}
}

// CaptureError captures an error to Sentry with context
func CaptureError(ctx context.Context, err error, tags map[string]string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			for key, value := range tags {
				scope.SetTag(key, value)
			}
			hub.CaptureException(err)
		})
	} else {
		sentry.CaptureException(err)
	}
}

// CaptureMessage captures a message to Sentry
func CaptureMessage(ctx context.Context, message string, level sentry.Level) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.Scope().SetLevel(level)
		hub.CaptureMessage(message)
	} else {
		sentry.CaptureMessage(message)
	}
}
```

### 3. Update Main Server Initialization

In `desktop/backend-go/cmd/server/main.go`, add Sentry initialization:

```go
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"your-module/internal/middleware"
)

func main() {
	// Initialize Sentry
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := middleware.InitSentry(middleware.SentryConfig{
			DSN:              sentryDSN,
			Environment:      getEnv("ENVIRONMENT", "development"),
			Release:          getEnv("RELEASE", "dev"),
			SampleRate:       1.0, // 100% in production
			TracesSampleRate: 0.1, // 10% performance monitoring
		})
		if err != nil {
			slog.Warn("Failed to initialize Sentry", "error", err)
		}
		defer sentry.Flush(2 * time.Second)
	} else {
		slog.Info("Sentry DSN not configured, error tracking disabled")
	}

	// Setup Gin router
	router := gin.Default()

	// Add Sentry middleware (must be added early)
	if sentryDSN != "" {
		router.Use(middleware.SentryMiddleware())
	}

	// ... rest of your server setup
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

### 4. Add Sentry Helper Functions

For capturing slow queries and specific errors:

```go
// In your database layer or service layer
import (
	"context"
	"time"
	"your-module/internal/middleware"
)

// Example: Capture slow database queries
func (r *Repository) GetUser(ctx context.Context, userID string) (*User, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 1*time.Second {
			middleware.CaptureMessage(ctx,
				"Slow database query: GetUser",
				sentry.LevelWarning,
			)
		}
	}()

	// Your database query logic
	// ...
}

// Example: Capture specific errors with tags
func (s *Service) ProcessPayment(ctx context.Context, amount float64) error {
	err := s.paymentGateway.Charge(amount)
	if err != nil {
		middleware.CaptureError(ctx, err, map[string]string{
			"operation": "payment_charge",
			"amount":    fmt.Sprintf("%.2f", amount),
		})
		return err
	}
	return nil
}
```

---

## Frontend Integration (SvelteKit)

### 1. Install Sentry SDK

```bash
cd frontend
npm install @sentry/sveltekit
```

### 2. Initialize Sentry

Create file: `frontend/src/hooks.client.ts`

```typescript
import { handleErrorWithSentry, replayIntegration } from "@sentry/sveltekit";
import * as Sentry from '@sentry/sveltekit';

Sentry.init({
  dsn: import.meta.env.PUBLIC_SENTRY_DSN,
  environment: import.meta.env.PUBLIC_ENVIRONMENT || 'development',

  // Set sample rates
  tracesSampleRate: 0.1, // 10% of transactions
  replaysSessionSampleRate: 0.1, // 10% of sessions
  replaysOnErrorSampleRate: 1.0, // 100% of errors

  integrations: [
    replayIntegration({
      maskAllText: true,
      blockAllMedia: true,
    }),
  ],

  // Filter sensitive data
  beforeSend(event) {
    // Remove cookies and authorization headers
    if (event.request) {
      delete event.request.cookies;
      if (event.request.headers) {
        delete event.request.headers['Authorization'];
        delete event.request.headers['Cookie'];
      }
    }
    return event;
  },
});

// Custom error handler
export const handleError = handleErrorWithSentry();
```

Create file: `frontend/src/hooks.server.ts`

```typescript
import { handleErrorWithSentry, sentryHandle } from "@sentry/sveltekit";
import * as Sentry from '@sentry/sveltekit';
import type { Handle } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';

Sentry.init({
  dsn: import.meta.env.PUBLIC_SENTRY_DSN,
  environment: import.meta.env.PUBLIC_ENVIRONMENT || 'development',
  tracesSampleRate: 0.1,
});

export const handle: Handle = sequence(sentryHandle());
export const handleError = handleErrorWithSentry();
```

### 3. Add Sentry to SvelteKit Config

Update `frontend/svelte.config.js`:

```javascript
import { sveltekit } from '@sveltejs/kit/vite';
import { sentrySvelteKit } from "@sentry/sveltekit";
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sentrySvelteKit({
			sourceMapsUploadOptions: {
				org: process.env.SENTRY_ORG,
				project: process.env.SENTRY_PROJECT,
				authToken: process.env.SENTRY_AUTH_TOKEN,
			}
		}),
		sveltekit()
	]
});
```

### 4. Create Sentry Utility Functions

Create file: `frontend/src/lib/sentry.ts`

```typescript
import * as Sentry from '@sentry/sveltekit';

export function captureError(error: Error, context?: Record<string, any>) {
  Sentry.captureException(error, {
    contexts: {
      custom: context,
    },
  });
}

export function captureMessage(message: string, level: Sentry.SeverityLevel = 'info') {
  Sentry.captureMessage(message, level);
}

export function setUserContext(user: { id: string; email?: string; username?: string }) {
  Sentry.setUser({
    id: user.id,
    email: user.email,
    username: user.username,
  });
}

export function clearUserContext() {
  Sentry.setUser(null);
}

// Example usage in API error handling
export async function fetchWithErrorTracking(url: string, options?: RequestInit) {
  try {
    const response = await fetch(url, options);

    if (!response.ok) {
      const error = new Error(`HTTP ${response.status}: ${response.statusText}`);
      captureError(error, {
        url,
        status: response.status,
        statusText: response.statusText,
      });
      throw error;
    }

    return response;
  } catch (error) {
    if (error instanceof Error) {
      captureError(error, { url });
    }
    throw error;
  }
}
```

### 5. Usage in Components

```svelte
<script lang="ts">
  import { captureError, captureMessage } from '$lib/sentry';

  async function handleSubmit() {
    try {
      await submitForm();
      captureMessage('Form submitted successfully', 'info');
    } catch (error) {
      captureError(error as Error, {
        component: 'ContactForm',
        action: 'submit',
      });
      // Show error to user
    }
  }
</script>
```

---

## Configuration

### Environment Variables

**Backend (.env):**
```env
SENTRY_DSN=https://[key]@[org].ingest.sentry.io/[project]
ENVIRONMENT=production
RELEASE=v1.0.0
```

**Frontend (.env):**
```env
PUBLIC_SENTRY_DSN=https://[key]@[org].ingest.sentry.io/[project]
PUBLIC_ENVIRONMENT=production
SENTRY_ORG=your-org
SENTRY_PROJECT=businessos-frontend
SENTRY_AUTH_TOKEN=your-auth-token
```

### GitHub Secrets

Add these secrets to your GitHub repository:

- `SENTRY_DSN_BACKEND`
- `SENTRY_DSN_FRONTEND`
- `SENTRY_AUTH_TOKEN` (for source map uploads)

---

## Testing

### Backend Testing

Create a test endpoint to verify Sentry integration:

```go
router.GET("/test/sentry", func(c *gin.Context) {
	// Test error capture
	err := errors.New("test error from backend")
	middleware.CaptureError(c.Request.Context(), err, map[string]string{
		"test": "true",
	})

	c.JSON(200, gin.H{"message": "Sentry test error sent"})
})

router.GET("/test/sentry-panic", func(c *gin.Context) {
	// Test panic recovery
	panic("test panic from backend")
})
```

### Frontend Testing

Create a test page: `frontend/src/routes/test-sentry/+page.svelte`

```svelte
<script lang="ts">
  import { captureError, captureMessage } from '$lib/sentry';

  function testError() {
    try {
      throw new Error('Test error from frontend');
    } catch (error) {
      captureError(error as Error);
    }
  }

  function testMessage() {
    captureMessage('Test message from frontend', 'info');
  }
</script>

<button on:click={testError}>Test Error</button>
<button on:click={testMessage}>Test Message</button>
```

### Verification Checklist

- [ ] Visit Sentry dashboard after triggering test errors
- [ ] Verify errors appear in correct project (backend/frontend)
- [ ] Check that user context is attached to errors
- [ ] Verify stack traces are readable
- [ ] Confirm sensitive data is filtered (no auth tokens)
- [ ] Test that 5xx errors are captured
- [ ] Test that panics are recovered and reported

---

## Monitoring & Alerts

### Key Metrics to Monitor

1. **Error Rate:** Errors per minute
2. **Error Types:** Most common error messages
3. **Affected Users:** Number of users impacted
4. **Error Trends:** Compare with previous periods

### Recommended Alerts

**Backend Alerts:**
- Error rate > 10 errors/min
- HTTP 500 errors > 5/min
- Slow database queries > 1s
- Panic/crash detected

**Frontend Alerts:**
- Error rate > 50 errors/min
- API request failures > 20%
- Page load errors
- JavaScript errors in critical paths

### Setting Up Alerts

1. Go to Sentry project
2. Navigate to **Alerts** > **Create Alert Rule**
3. Configure alert conditions:
   - Condition: "An event is seen"
   - Filter: "error.type equals 'Error'"
   - Action: Send to Slack/Email

---

## Best Practices

### 1. Error Context

Always include context when capturing errors:

```go
middleware.CaptureError(ctx, err, map[string]string{
	"user_id":   userID,
	"operation": "create_workspace",
	"workspace": workspaceID,
})
```

### 2. Breadcrumbs

Add breadcrumbs for debugging:

```typescript
Sentry.addBreadcrumb({
  category: 'navigation',
  message: 'User navigated to dashboard',
  level: 'info',
});
```

### 3. Filter Sensitive Data

Never send:
- Passwords
- API keys
- Session tokens
- Personal information (emails, phone numbers)

### 4. Sample Rates

Production recommendations:
- Error sampling: **100%** (capture all errors)
- Performance tracing: **10-20%** (sample transactions)
- Session replays: **10%** normal, **100%** on error

### 5. Release Tracking

Tag releases for better error tracking:

```bash
# Backend
RELEASE=v1.2.0 go run cmd/server/main.go

# Frontend (in CI/CD)
SENTRY_RELEASE=$(git rev-parse HEAD) npm run build
```

---

## Troubleshooting

### Errors Not Appearing in Sentry

1. Check DSN is correct
2. Verify network connectivity to sentry.io
3. Check sample rates (should be 1.0 for testing)
4. Look for initialization errors in logs

### Source Maps Not Working (Frontend)

1. Verify `SENTRY_AUTH_TOKEN` is set
2. Check `sentry.properties` file exists
3. Run build with `--debug` flag
4. Verify source maps are uploaded after build

### Too Many Errors

1. Adjust sample rates
2. Add filters in `beforeSend`
3. Ignore known non-critical errors
4. Fix the bugs (best solution)

---

## Resources

- [Sentry Go Documentation](https://docs.sentry.io/platforms/go/)
- [Sentry SvelteKit Documentation](https://docs.sentry.io/platforms/javascript/guides/sveltekit/)
- [Sentry Best Practices](https://docs.sentry.io/product/best-practices/)

---

**Last Updated:** 2026-01-18
**Maintainer:** BusinessOS Team
