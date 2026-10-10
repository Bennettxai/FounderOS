# API Versioning Strategy

## Overview

BusinessOS API uses URL path versioning to provide stable, predictable API evolution while allowing breaking changes in future versions.

**Current Version:** v1
**Versioning Scheme:** `/api/{version}/{resource}`
**Example:** `POST /api/v1/chat/message`

---

## Supported Versions

| Version | Status | Sunset Date | Notes |
|---------|--------|-------------|-------|
| **v1** | **Current** | N/A | Stable production version |
| (none) | Deprecated | TBD | Non-versioned `/api/*` paths redirect to v1 |

---

## Migration Guide

### For API Consumers (Frontend, Mobile Apps, Integrations)

**Action Required:** Update all API calls to use versioned endpoints.

**Before:**
```typescript
POST /api/chat/message
GET /api/workspaces
```

**After:**
```typescript
POST /api/v1/chat/message
GET /api/v1/workspaces
```

### Backward Compatibility Period

All `/api/*` endpoints (non-versioned) currently redirect to `/api/v1/*` with deprecation headers:

```http
HTTP/1.1 200 OK
Deprecation: true
Link: </api/v1>; rel="successor-version"
Warning: 299 - "Direct /api access is deprecated. Use /api/v1 instead"
API-Version: v1
```

**Recommended Action:** Update clients immediately to avoid future breaking changes.

---

## Deprecation Policy

### When a Version is Deprecated

1. **Announcement:** Minimum 6 months notice via:
   - API changelog
   - Email to registered developers
   - In-app notifications

2. **Deprecation Headers:** Deprecated versions include:
   ```http
   Deprecation: true
   Sunset: 2026-12-31T23:59:59Z
   Link: </api/v2>; rel="successor-version"
   ```

3. **Sunset Period:** 12 months from deprecation announcement

4. **Removal:** After sunset date, version returns `410 Gone`

### Monitoring Deprecated API Usage

Check server logs for deprecated API access:
```bash
# Find non-versioned API calls (deprecated)
grep "non_versioned_api_access" /var/log/backend.log

# Find deprecated version usage
grep "deprecated_api_version_accessed" /var/log/backend.log
```

---

## Version Headers

All API responses include versioning headers:

```http
API-Version: v1
```

For deprecated versions:
```http
Deprecation: true
Sunset: 2026-06-01T00:00:00Z
Link: </api/v2>; rel="successor-version"
```

---

## Breaking vs Non-Breaking Changes

### Non-Breaking Changes (Same Version)

These changes do NOT require a new API version:
- Adding new endpoints
- Adding optional request parameters
- Adding new fields to responses
- Adding new enum values (with backward compatible defaults)
- Fixing bugs that match documented behavior
- Performance improvements

### Breaking Changes (New Version Required)

These changes REQUIRE a new API version:
- Removing or renaming endpoints
- Removing or renaming request/response fields
- Changing field types (e.g., string → number)
- Changing authentication mechanisms
- Changing error response formats
- Making optional parameters required
- Removing enum values

---

## Versioning Implementation

### Adding a New API Version

1. **Create version configuration** in `internal/middleware/versioning.go`:
```go
"v2": {
    Version:          "v2",
    IsDeprecated:     false,
    SunsetDate:       time.Time{},
    SuccessorVersion: "",
}
```

2. **Create route group** in `cmd/server/main.go`:
```go
apiv2 := router.Group("/api/v2")
apiv2.Use(middleware.DeprecationHeaders(v2Config))
h.RegisterRoutesV2(apiv2)
```

3. **Mark previous version as deprecated**:
```go
"v1": {
    Version:          "v1",
    IsDeprecated:     true,
    SunsetDate:       time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
    SuccessorVersion: "v2",
}
```

4. **Update documentation** with migration guide

---

## API Stability Guarantees

### Stable (Production)

**Version:** v1
**Guarantee:** No breaking changes within same major version
**Deprecation Notice:** Minimum 6 months
**Support Duration:** 12 months after deprecation

### Beta (Testing)

**Version:** vX-beta
**Guarantee:** May change without notice
**Deprecation Notice:** Not guaranteed
**Support Duration:** Until promoted to stable or removed

### Alpha (Experimental)

**Version:** vX-alpha
**Guarantee:** No stability guarantee
**Deprecation Notice:** None
**Support Duration:** May be removed at any time

---

## Client Best Practices

### 1. Always Specify Version

```typescript
// Good
const response = await fetch('/api/v1/chat/message', { ... })

// Bad - Will break when defaults change
const response = await fetch('/api/chat/message', { ... })
```

### 2. Handle Deprecation Headers

```typescript
const response = await fetch('/api/v1/chat/message')

if (response.headers.get('Deprecation') === 'true') {
  const sunset = response.headers.get('Sunset')
  const successor = response.headers.get('Link')

  console.warn(`API v1 deprecated. Sunset: ${sunset}. Migrate to: ${successor}`)
  // Log to analytics, show user warning, etc.
}
```

### 3. Version Your Client

```typescript
const API_VERSION = 'v1'
const baseURL = `/api/${API_VERSION}`

// Centralized version management
```

### 4. Test Against Multiple Versions

```typescript
describe('API compatibility', () => {
  ['v1', 'v2'].forEach(version => {
    it(`works with ${version}`, async () => {
      const response = await fetch(`/api/${version}/chat/message`)
      expect(response.ok).toBe(true)
    })
  })
})
```

---

## OpenAPI/Swagger Documentation

Each version has separate OpenAPI documentation:

- **v1:** `/api/v1/docs` (Swagger UI)
- **v1 Spec:** `/api/v1/openapi.json`

Future versions will follow same pattern:
- **v2:** `/api/v2/docs`
- **v2 Spec:** `/api/v2/openapi.json`

---

## Common Questions

### Q: Why URL versioning instead of header-based?

**A:** URL versioning is:
- Easier to test (just change URL)
- Visible in browser/logs
- No special headers needed
- Better cache-ability

### Q: Do I need to migrate immediately?

**A:** No, but recommended. Non-versioned `/api/*` paths work but:
- Include deprecation warnings
- May break without notice in future
- Performance overhead from redirect

### Q: What if I find a bug in v1?

**A:** Bug fixes are non-breaking changes and will be patched in v1.
Security fixes are applied to ALL supported versions.

### Q: Can I use multiple versions simultaneously?

**A:** Yes! This is recommended during migration:
```typescript
// Gradual migration
const chat = await fetch('/api/v2/chat/message')  // New feature
const users = await fetch('/api/v1/users')        // Legacy
```

### Q: How do I know which version to use?

**A:** Always use the latest stable version unless:
- Migrating from deprecated version (use intermediate version)
- Testing new features (use beta version)
- Maintaining legacy integration (use original version until migration)

---

## Version History

| Version | Release Date | Sunset Date | Status |
|---------|--------------|-------------|--------|
| v1 | 2026-01-19 | - | Current |

---

## Support

- **API Issues:** https://github.com/Miosa-osa/businessos/issues
- **Migration Help:** Create issue with `api-migration` label
- **Breaking Changes:** Announced in CHANGELOG.md

---

**Last Updated:** 2026-01-19
**Document Version:** 1.0
