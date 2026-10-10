# Secure Logging - Quick Reference Card

## Import
```go
import "github.com/rhl/businessos-backend/internal/logging"
```

---

## Basic Logging (with auto-sanitization)

```go
// Debug (shows only in development)
logging.Debug("Processing request for %s", logging.MaskUserID(userID))

// Info (normal operations)
logging.Info("User session created: %s", logging.MaskSessionID(sessionID))

// Warning
logging.Warn("Rate limit approaching for %s", logging.MaskIP(clientIP))

// Error
logging.Error("Failed to process request: %v", err)

// Security events
logging.Security("Unauthorized access attempt from %s", logging.MaskIP(ip))
```

---

## Structured Logging (recommended for production)

```go
// With fields
logging.InfoWithFields("User action completed", map[string]interface{}{
    "user_id":  userID,         // Auto-masked
    "action":   "create_post",
    "duration": duration.Milliseconds(),
})

// Error with context
logging.ErrorWithFields("Database query failed", map[string]interface{}{
    "error":    err.Error(),
    "query":    logging.SanitizeSQL(query),
    "user_id":  logging.MaskUserID(userID),
})

// Security event (structured)
logging.SecurityWithFields("Failed login attempt", map[string]interface{}{
    "user_id":  userID,
    "ip":       clientIP,
    "attempts": attemptCount,
})
```

---

## Security Event Logging (best for audit trail)

```go
logging.LogSecurityEvent(logging.SecurityEvent{
    EventType:   "authentication_failed",
    UserID:      userID,              // Auto-masked
    IP:          clientIP,            // Auto-masked
    Description: "Invalid password",
    Severity:    "medium",            // low, medium, high, critical
    Metadata: map[string]interface{}{
        "attempt_number": 3,
        "account_locked": false,
    },
})
```

---

## HTTP Request Logging

```go
logging.LogHTTPRequest(logging.HTTPRequestLog{
    Method:     c.Request.Method,
    Path:       c.Request.URL.Path,    // Auto-sanitized
    UserAgent:  c.Request.UserAgent(),
    IP:         c.ClientIP(),           // Auto-masked
    UserID:     user.ID,                // Auto-masked
    StatusCode: statusCode,
    Duration:   time.Since(startTime),
})
```

---

## Manual Masking Functions

```go
// Email: john.doe@example.com → j***@example.com
maskedEmail := logging.MaskEmail(email)

// Session ID: abc123-def456 → abc123-d********
maskedSession := logging.MaskSessionID(sessionID)

// User ID (same as session ID)
maskedUser := logging.MaskUserID(userID)

// IP: 192.168.1.100 → 192.168.xxx.xxx
maskedIP := logging.MaskIP(ipAddress)

// Token: Bearer abc123... → Bearer ***[TOKEN_REDACTED]
maskedToken := logging.MaskToken(token)
```

---

## Sanitization Functions

```go
// Detect and redact secrets (AWS keys, GitHub tokens, etc.)
sanitized, wasDetected := logging.DetectAndRedactSecrets(text)

// SQL query (removes parameters)
sanitizedSQL := logging.SanitizeSQL("SELECT * FROM users WHERE email = 'test@ex.com'")
// Result: "SELECT * FROM users WHERE email = '[REDACTED]'"

// URL (removes query params and sensitive paths)
sanitizedURL := logging.SanitizeURL("https://api.com/session/abc123?token=xyz")
// Result: "https://api.com/session/[REDACTED]?[PARAMS_REDACTED]"

// Cookies (masks all values)
sanitizedCookies := logging.SanitizeCookies("session=abc; token=xyz")
// Result: "session=[REDACTED]; token=[REDACTED]"

// Map fields (redacts password, token, api_key, etc.)
safeFields := logging.SafeLogFields(fields)
```

---

## Common Patterns

### ✅ DO: Log with Masking
```go
logging.Info("User %s logged in from %s",
    logging.MaskEmail(user.Email),
    logging.MaskIP(clientIP))
```

### ❌ DON'T: Log Raw Sensitive Data
```go
logging.Info("User %s logged in from %s", user.Email, clientIP)  // Exposes PII
```

---

### ✅ DO: Use Structured Logging
```go
logging.SecurityWithFields("Login success", map[string]interface{}{
    "user_id": userID,     // Auto-masked
    "ip":      clientIP,   // Auto-masked
})
```

### ❌ DON'T: Mix Formats
```go
logging.Security("Login success user=%s ip=%s", userID, clientIP)  // Manual formatting
```

---

### ✅ DO: Log Errors with Context
```go
logging.ErrorWithFields("Database error", map[string]interface{}{
    "error":    err.Error(),
    "query":    logging.SanitizeSQL(query),
    "user_id":  logging.MaskUserID(userID),
})
```

### ❌ DON'T: Log Full Queries
```go
logging.Error("Query failed: %s", query)  // Might expose user data
```

---

## Configuration

### Production (main.go)
```go
logging.SetGlobalConfig(&logging.LogConfig{
    Format:              "json",           // For log aggregation
    MinLevel:            logging.LevelInfo, // Hide debug logs
    MaskSensitiveData:   true,             // Always on
    SessionIDMaskLength: 8,
    FilterTerminalIO:    true,
})
```

### Development
```go
logging.SetGlobalConfig(&logging.LogConfig{
    Format:              "text",            // Human-readable
    MinLevel:            logging.LevelDebug, // Show all
    MaskSensitiveData:   true,              // Keep masking!
    SessionIDMaskLength: 12,                // Show more
    FilterTerminalIO:    false,
})
```

---

## Field Names Auto-Redacted

The following field names are **automatically redacted** in `SafeLogFields()`:

- `password`, `passwd`, `pwd`
- `token`, `api_key`, `apikey`, `api-key`
- `secret`, `credential`, `auth`
- `session_id`, `sessionid`, `session-id`
- `bearer`, `authorization`
- `private_key`, `privatekey`, `private-key`
- `cookie`, `csrf`

---

## Secret Patterns Auto-Detected

These patterns are **automatically detected and redacted**:

1. AWS Access Keys (`AKIA*`, `ASIA*`)
2. GitHub Tokens (`ghp_*`, `gho_*`, `github_pat_*`)
3. JWT Tokens (`eyJ*`)
4. API Keys in assignments (`api_key = 'sk-...'`)
5. Base64 secrets (40+ chars)
6. Private Keys (`-----BEGIN PRIVATE KEY-----`)
7. OAuth tokens (`ya29.*`, `xox*`)
8. Generic long tokens (32+ chars)

---

## Migration from Standard `log`

### Before
```go
import "log"

log.Printf("User %s created session %s", userID, sessionID)
log.Printf("Error: %v", err)
```

### After
```go
import "github.com/rhl/businessos-backend/internal/logging"

logging.InfoWithFields("User created session", map[string]interface{}{
    "user_id":    userID,    // Auto-masked
    "session_id": sessionID, // Auto-masked
})
logging.Error("Error: %v", err)
```

---

## Testing

```bash
# Run logging tests
go test ./internal/logging/... -v

# Run benchmarks
go test ./internal/logging/... -bench=. -benchmem
```

---

## Quick Checklist

- [ ] Never log passwords, tokens, or API keys in full
- [ ] Always use `MaskEmail()` for email addresses
- [ ] Always use `MaskSessionID()` for session/user IDs
- [ ] Always use `MaskIP()` for IP addresses
- [ ] Use `SanitizeSQL()` before logging SQL queries
- [ ] Use structured logging (`*WithFields`) for complex events
- [ ] Use `LogSecurityEvent()` for authentication/authorization events
- [ ] Keep `MaskSensitiveData: true` even in development
- [ ] Configure JSON logging for production

---

## Help

- Full Guide: `/internal/logging/SANITIZED_LOGGING_GUIDE.md`
- Implementation: `/internal/logging/IMPLEMENTATION_SUMMARY.md`
- Examples: `/internal/terminal/websocket.go` (reference implementation)
- Tests: `/internal/logging/*_test.go`

---

## Emergency: Disable Masking (NOT recommended)

```go
// ONLY for local debugging, NEVER in production
logging.SetGlobalConfig(&logging.LogConfig{
    MaskSensitiveData: false,  // DANGER: Exposes sensitive data
})
```

**WARNING**: This should NEVER be used in production or committed to git.
