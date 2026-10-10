# Error Response Standardization Migration Guide

## Overview

This document describes the standardized error response format implemented in `internal/middleware/error_response.go` and how to migrate existing handlers.

## Standardized Error Format

All API errors now follow this format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "message": "Optional additional context",
  "details": {
    "field1": "error detail",
    "field2": "error detail"
  }
}
```

## Standard Error Codes

- `BAD_REQUEST` - Invalid request format or parameters
- `UNAUTHORIZED` - Authentication required or failed
- `FORBIDDEN` - Authenticated but not authorized
- `NOT_FOUND` - Resource does not exist
- `CONFLICT` - Resource conflict (duplicate, etc.)
- `VALIDATION_ERROR` - Input validation failed
- `INTERNAL_SERVER_ERROR` - Server-side error
- `SERVICE_UNAVAILABLE` - Service temporarily unavailable
- `DATABASE_ERROR` - Database operation failed
- `INVALID_INPUT` - Invalid input data

## Migration Examples

### Before (Old Style)

```go
// Various inconsistent formats
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
c.JSON(400, gin.H{"message": "bad request"})
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
```

### After (Standardized)

```go
import "github.com/rhl/businessos-backend/internal/middleware"

// Simple errors
middleware.RespondBadRequest(c, "Invalid input format")
middleware.RespondUnauthorized(c, "Session expired")
middleware.RespondNotFound(c, "User")
middleware.RespondInternalError(c, "Failed to process request")

// Validation errors with field details
middleware.RespondWithValidationError(c, map[string]string{
    "email": "Invalid email format",
    "age": "Must be 18 or older",
})

// Detailed errors with custom code
middleware.RespondWithDetailedError(c, http.StatusConflict,
    "Email already exists",
    "DUPLICATE_EMAIL",
    map[string]interface{}{
        "email": user.Email,
        "existing_user_id": existingUserID,
    })

// Database errors
middleware.RespondDatabaseError(c, "user creation")
```

## Success Responses

Standardized success responses are also available:

```go
// Simple success with data
middleware.RespondSuccess(c, userData)

// Created resource (201)
middleware.RespondCreated(c, newUser)

// Success with message
middleware.RespondWithMessage(c, "User updated successfully", userData)

// No content (204)
middleware.RespondNoContent(c)
```

## Migration Strategy

### Phase 1: Critical Endpoints (Week 1)
- Authentication endpoints (`/api/v1/auth/*`)
- User management (`/api/v1/users/*`)
- Workspace operations (`/api/v1/workspaces/*`)

### Phase 2: Feature Endpoints (Week 2)
- Chat endpoints (`/api/v1/chat/*`)
- Project management (`/api/v1/projects/*`)
- File operations (`/api/v1/files/*`)

### Phase 3: Integration Endpoints (Week 3)
- Third-party integrations (`/api/v1/integrations/*`)
- Webhooks (`/api/v1/webhooks/*`)
- Admin endpoints (`/api/v1/admin/*`)

## Handler Example

### Before

```go
func (h *Handlers) GetUser(c *gin.Context) {
    userID := c.Param("id")
    if userID == "" {
        c.JSON(400, gin.H{"error": "user id required"})
        return
    }

    user, err := h.queries.GetUser(c.Request.Context(), userID)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(404, gin.H{"error": "user not found"})
            return
        }
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, user)
}
```

### After

```go
func (h *Handlers) GetUser(c *gin.Context) {
    userID := c.Param("id")
    if userID == "" {
        middleware.RespondBadRequest(c, "User ID is required")
        return
    }

    user, err := h.queries.GetUser(c.Request.Context(), userID)
    if err != nil {
        if err == sql.ErrNoRows {
            middleware.RespondNotFound(c, "User")
            return
        }
        middleware.RespondDatabaseError(c, "user retrieval")
        return
    }

    middleware.RespondSuccess(c, user)
}
```

## Testing

Update tests to expect the new format:

```go
func TestGetUser_NotFound(t *testing.T) {
    // ... setup ...

    var response middleware.ErrorResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, "User not found", response.Error)
    assert.Equal(t, middleware.ErrCodeNotFound, response.Code)
}
```

## Benefits

1. **Consistency**: All errors follow the same format
2. **Client-Friendly**: Error codes enable proper error handling
3. **Debuggable**: Details field provides context
4. **Type-Safe**: Compile-time checking of response format
5. **Testable**: Easier to write tests with predictable format
6. **Documentation**: Auto-generated API docs are cleaner

## Backwards Compatibility

For a transition period, both old and new formats are accepted by clients. The old format will be deprecated after all endpoints are migrated.

## Rollout Checklist

- [ ] Import `internal/middleware` in handler files
- [ ] Replace `c.JSON(status, gin.H{...})` with `middleware.Respond*` functions
- [ ] Update handler tests to expect new format
- [ ] Update API documentation
- [ ] Update frontend error handling (if needed)
- [ ] Remove old error response patterns

---

**Status**: ✅ Implemented (2026-01-19)
**Tracking**: CUS-91
**Maintained by**: Backend Team
