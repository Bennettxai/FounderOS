# CRIT-2: HMAC Signature on Session Cookie Stripped and Never Verified

**Severity:** CRITICAL | **Exploitability:** 5/10 | **CVSS:** 8.1

## Location
- `desktop/backend-go/internal/middleware/auth.go:62-65`
- `desktop/backend-go/internal/middleware/redis_auth.go:272-277, 364-368, 432-436`

## Description
Better Auth signs cookies as `{token}.{HMAC-SHA256(secret, token)}`. The Go middleware strips the signature and uses only the raw token for DB lookup:

```go
sessionToken := sessionCookie
if idx := strings.Index(sessionCookie, "."); idx != -1 {
    sessionToken = sessionCookie[:idx]  // Signature discarded
}
```

This pattern appears in 4 middleware functions. The HMAC signature provides zero server-side protection.

## Impact
- Token forgery: anyone who knows/guesses a raw token bypasses signing
- Session oracle: `{guessed_token}.anything` triggers DB lookup
- Split-brain: Go accepts sessions that Better Auth has invalidated

## Fix
```go
func verifyAndExtractToken(sessionCookie, secret string) (string, bool) {
    idx := strings.LastIndex(sessionCookie, ".")
    if idx == -1 { return "", false }
    token, sig := sessionCookie[:idx], sessionCookie[idx+1:]
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(token))
    expected := hex.EncodeToString(mac.Sum(nil))
    if !hmac.Equal([]byte(sig), []byte(expected)) { return "", false }
    return token, true
}
```
Apply in all 4 middleware locations.
